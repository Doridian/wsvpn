#!/usr/bin/env python3

from __future__ import annotations
from copy import deepcopy
from os.path import join, dirname
from asyncio.subprocess import DEVNULL, PIPE
from os import remove
from signal import SIGTERM
from subprocess import Popen, check_output
from tempfile import mktemp
from threading import Thread, Condition
from time import sleep
from typing import Any, Optional
from yaml import dump as yaml_dump, safe_load as yaml_load
from ipaddress import IPv4Address

from build import get_local_arch, get_local_platform
from tests.tls_utils import TLSCertSet

LOCAL_ARCH = get_local_arch()
LOCAL_PLATFORM = get_local_platform()

BIN_DIR = join(dirname(__file__), "../dist/")

SCRIPT_HDL = join(dirname(__file__), "script_hdl.py")

_default_configs: map = {}
def _get_default_config(binf: str) -> Any:
    if binf in _default_configs:
        return _default_configs[binf]
    cfg_str = check_output([binf, "--print-default-config"])
    cfg = yaml_load(cfg_str)
    
    cfg["scripts"]["down"] = SCRIPT_HDL
    cfg["scripts"]["up"] = SCRIPT_HDL

    _default_configs[binf] = cfg
    return cfg


def split_ip(ipsub: str) -> str:
    return ipsub.split("/")[0]


LAST_PORT = 4000

class GoBin(Thread):
    def __init__(self, proj: str) -> None:
        super().__init__(daemon=True)

        self.proj = proj
        self.bin = join(BIN_DIR, f"{proj}-{LOCAL_PLATFORM}-{LOCAL_ARCH}")
        self.cfg = deepcopy(_get_default_config(self.bin))

        self.is_server = proj == "server"
        self.is_client = proj == "client"

        if self.is_server:
            global LAST_PORT
            port = LAST_PORT
            LAST_PORT += 1

            tmp_ip = split_ip(self.cfg["tunnel"]["subnet"])
            self.ip = (IPv4Address(tmp_ip) + 1).exploded
            self.cfg["server"]["listen"] = f"127.0.0.1:{port}"
        else:
            self.ip = None

        self.proc_wait_cond = Condition()
        self.is_ready_or_done = False
        self.proc = None
        self.ready_ok = None
        
        self.iface_names = {}
        self.auth_names = {}
        self.startup_timeout = None

        self.http_auth_enabled = False
        self.mtls_auth_enabled = False


    def is_tap_supported(self) -> bool:
        return get_local_platform() != "darwin"


    def is_one_interface_per_connection_supported(self) -> bool:
        return get_local_platform() != "windows"


    def connect_to(self, server: GoBin, user: str = "", password: str = "", protocol: str = "AUTO") -> None:
        if not self.is_client or not server.is_server:
            raise ValueError("Can only connect client to server")

        listen = server.cfg["server"]["listen"]
        lspl = listen.split(":")
        port = lspl[-1]

        is_tls = server.cfg["server"]["tls"]["key"]

        if protocol == "AUTO":
            if server.cfg["server"]["enable-http3"]:
                protocol = "webtransport"
            elif is_tls:
                protocol = "wss"
            else:
                protocol = "ws"

        auth_str = ""
        if user or password:
            auth_str = f"{user}:{password}@"
            self.http_auth_enabled = True

        self.cfg["client"]["server"] = f"{protocol}://{auth_str}127.0.0.1:{port}"


    def enable_tls(self, tls_cert_set: Optional[TLSCertSet]) -> None:
        if self.is_client:
            self.cfg["client"]["tls"]["ca"] = tls_cert_set.ca if tls_cert_set else None
            return

        self.cfg["server"]["tls"]["certificate"] = tls_cert_set.cert if tls_cert_set else None
        self.cfg["server"]["tls"]["key"] = tls_cert_set.key if tls_cert_set else None


    def enable_mtls(self, tls_cert_set: Optional[TLSCertSet]) -> None:
        self.mtls_auth_enabled = tls_cert_set is not None

        if self.is_server:
            self.cfg["server"]["tls"]["client-ca"] = tls_cert_set.ca if tls_cert_set else None
            return

        self.cfg["client"]["tls"]["certificate"] = tls_cert_set.cert if tls_cert_set else None
        self.cfg["client"]["tls"]["key"] = tls_cert_set.key if tls_cert_set else None


    def wait_ready_or_done(self) -> None:
        self.proc_wait_cond.acquire()
        self.proc_wait_cond.wait_for(predicate=lambda : self.is_ready_or_done)
        self.proc_wait_cond.release()

    
    def start(self) -> None:
        def startup_wait():
            sleep(5)
            self._notify_ready(False)
        self.startup_timeout = Thread(daemon=True, target=startup_wait)
        super().start()
        self.startup_timeout.start()


    def stop(self) -> None:
        if self.proc is not None and self.proc.returncode is None:
            self.proc.send_signal(SIGTERM)

        if self.is_alive():
            self.join(timeout=1)
            if self.proc is not None:
                self.proc.kill()
            self.join()


    def handle_line(self, line: str) -> None:
        print(line)
    
        if self.is_server and "VPN server online at" in line:
            self._notify_ready(True)

        if "SCRIPT_HDL" in line:
            lspl = line.split(" ")[2:]

            if lspl[0] == "up":
                ip = split_ip(lspl[1])
                if self.is_client:
                    self.iface_names["server"] = lspl[2]
                    self.ip = ip
                    self._notify_ready(True)

                if self.is_server:
                    self.iface_names[ip] = lspl[2]
                    self.auth_names[ip] = lspl[3] if (len(lspl) >= 4) else ""

            elif lspl[0] == "down":
                if self.is_client:
                    self.iface_names.pop("server")
                    self.ip = None

                if self.is_server:
                    ip = split_ip(lspl[1])
                    self.iface_names.pop(ip)
                    self.auth_names.pop(ip)

            else:
                raise Exception(f"script called with invalid args: {lspl}")


    def get_ip(self) -> str:
        return self.ip


    def get_auth_for(self, clbin: GoBin = None) -> str:
        if not self.is_server:
            raise Exception("Only servers can use get_auth_for")

        client_ip = clbin.get_ip()
        return self.auth_names[client_ip]


    def get_interface_for(self, clbin: GoBin = None) -> str:
        if self.is_client:
            # clbin does not matter here, we only have one iface
            return self.iface_names["server"]

        client_ip = clbin.get_ip()
        return self.iface_names[client_ip]


    def _notify_ready(self, ok: bool) -> None:
        if self.is_ready_or_done:
            return

        self.is_ready_or_done = True
        self.ready_ok = ok
        self.proc_wait_cond.acquire()
        self.proc_wait_cond.notify_all()
        self.proc_wait_cond.release()


    def assert_ready_ok(self, should: bool = True) -> None:
        self.wait_ready_or_done()
        assert self.ready_ok == should


    def run(self) -> None:
        cfgfile = mktemp()
        with open(cfgfile, "w") as f:
            yaml_dump(self.cfg, f)

        try:
            self.proc = Popen([self.bin, "-config", cfgfile], stdin=DEVNULL, stderr=PIPE, text=True)

            while self.proc.returncode is None:
                res = self.proc.stderr.readline()
                if not res:
                    break
                self.handle_line(res.strip())
        finally:
            self._notify_ready(False)
            remove(cfgfile)

def new_clbin():
    return GoBin("client")

def new_svbin():
    return GoBin("server")

