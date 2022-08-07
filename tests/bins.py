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

    cfg["interface"]["one-interface-per-connection"] = True

    _default_configs[binf] = cfg
    return cfg


def split_ip(ipsub: str) -> str:
    return ipsub.split("/")[0]


class GoBin(Thread):
    def __init__(self, proj: str) -> None:
        super().__init__(daemon=True)

        self.proj = proj
        self.bin = join(BIN_DIR, f"{proj}-{LOCAL_PLATFORM}-{LOCAL_ARCH}")
        self.cfg = deepcopy(_get_default_config(self.bin))

        self.is_server = proj == "server"
        self.is_client = proj == "client"

        if self.is_server:
            tmp_ip = split_ip(self.cfg["tunnel"]["subnet"])
            self.ip = (IPv4Address(tmp_ip) + 1).exploded
        else:
            self.ip = None

        self.proc_wait_cond = Condition()
        self.is_ready_or_done = False
        self.proc = None
        self.ready_ok = None
        
        self.iface_names = {}
        self.startup_timeout = None


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

        self.cfg["client"]["server"] = f"{protocol}://{auth_str}127.0.0.1:{port}"


    def enable_tls(self, tls_cert_set: TLSCertSet) -> None:
        if self.is_client:
            self.cfg["client"]["tls"]["ca"] = tls_cert_set.ca
            return

        self.cfg["server"]["tls"]["certificate"] = tls_cert_set.cert
        self.cfg["server"]["tls"]["key"] = tls_cert_set.key


    def wait_ready_or_done(self) -> None:
        while not self.is_ready_or_done:
            self.proc_wait_cond.acquire()
            self.proc_wait_cond.wait()
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
        if self.is_server and "VPN server online at" in line:
            self._notify_ready(True)

        if "SCRIPT_HDL" in line:
            lspl = line.split(" ")[2:]

            if lspl[0] == "up":
                if self.is_client:
                    self.iface_names["server"] = lspl[2]
                    self.ip = split_ip(lspl[1])
                    self._notify_ready(True)

                if self.is_server:
                    self.iface_names[split_ip(lspl[1])] = lspl[2]

            elif lspl[0] == "down":
                if self.is_server:
                    self.iface_names.pop(split_ip(lspl[1]))

            return

        print(line)


    def get_ip(self) -> str:
        return self.ip

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


    def assert_ready_ok(self) -> None:
        self.wait_ready_or_done()
        assert self.ready_ok

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
