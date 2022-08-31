#!/usr/bin/env python3

from __future__ import annotations
from copy import deepcopy
from os.path import join, dirname, realpath
from os import remove
from signal import SIGTERM
from subprocess import Popen, check_output, DEVNULL, PIPE
from tempfile import NamedTemporaryFile
from threading import Thread, Condition
from time import sleep
from typing import Any, Optional
from yaml import dump as yaml_dump, safe_load as yaml_load
from ipaddress import ip_address
from getmac import get_mac_address
from sys import executable

from build import get_local_arch, get_local_platform
from tests.tls_utils import TLSCertSet

LOCAL_ARCH = get_local_arch()
LOCAL_PLATFORM = get_local_platform()

BIN_DIR = join(dirname(__file__), "..", "dist")

SCRIPT_HDL = join(dirname(__file__), "script_hdl.py")

SUBNET_ID = 0

_default_configs: map = {}


def _get_default_config(binf: str) -> Any:
    if binf in _default_configs:
        return _default_configs[binf]
    cfg_str = check_output(
        args=[binf, "--print-default-config"], executable=binf)
    cfg = yaml_load(cfg_str)

    cfg["scripts"]["down"] = [executable, SCRIPT_HDL]
    cfg["scripts"]["up"] = [executable, SCRIPT_HDL]

    _default_configs[binf] = cfg
    return cfg


def split_ip(ipsub: str) -> str:
    return ipsub.split("/")[0]


LAST_PORT = 4000


class GoBin(Thread):
    def __init__(self, proj: str) -> None:
        super().__init__(daemon=True)

        self.proj = proj
        self.bin = realpath(
            join(BIN_DIR, f"{proj}-{LOCAL_PLATFORM}-{LOCAL_ARCH}{self.executable_suffix()}"))
        self.cfg = deepcopy(_get_default_config(self.bin))

        self.is_server = proj == "server"
        self.is_client = proj == "client"

        if self.is_server:
            global LAST_PORT
            self.port = LAST_PORT
            LAST_PORT += 1
            self.cfg["server"]["listen"] = f"127.0.0.1:{self.port}"
            self.cfg["tunnel"]["subnet"] = None
        else:
            self.port = None
            self.ip = None

        self.proc_wait_cond = Condition()
        self.is_ready_or_done = False
        self.proc = None
        self.ready_ok = None

        self.iface_names = {}
        self.auth_names = {}
        self.iface_macs = {}
        self.startup_timeout = None

        self.http_auth_enabled = False
        self.mtls_auth_enabled = False

        self.ip_version = 4

    def is_tap_supported(self) -> bool:
        return True

    def is_one_interface_per_connection_supported(self, mode: str) -> bool:
        return get_local_platform() != "windows"

    def executable_suffix(self) -> str:
        if get_local_platform() == "windows":
            return ".exe"
        return ""

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
        self.proc_wait_cond.wait_for(predicate=lambda: self.is_ready_or_done)
        self.proc_wait_cond.release()

    def start(self) -> None:
        def startup_wait():
            sleep(10)
            self._notify_ready(False)
        self.startup_timeout = Thread(daemon=True, target=startup_wait)
        super().start()
        self.startup_timeout.start()

    def stop(self) -> None:
        if self.proc is not None and self.proc.returncode is None:
            self.proc.send_signal(SIGTERM)

        if self.is_alive():
            self.join(timeout=5)
            if self.proc is not None:
                self.proc.kill()
            self.join()

    def handle_line(self, line: str) -> None:
        print(line, flush=True)

        if self.is_server and "VPN server online at" in line:
            self._notify_ready(True)

        if "SCRIPT_HDL" in line:
            lspl = line.split(" ")[2:]

            if lspl[0] == "up":
                ip = split_ip(lspl[1])
                if self.is_client:
                    self.iface_names["server"] = lspl[2]
                    self.ip = ip
                    print(f"Setting client IP to {ip}", flush=True)
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

    def get_mac_for(self, clbin: GoBin = None) -> str:
        iface = self.get_interface_for(clbin=clbin)
        if not iface:
            return None
        if iface not in self.iface_macs:
            self.iface_macs[iface] = get_mac_address(
                interface=iface, network_request=False)
        return self.iface_macs[iface]

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
        if self.is_server:
            if not self.cfg["tunnel"]["subnet"]:
                subnet_index = self.port
                if self.cfg["tunnel"]["mode"] == "TAP":
                    subnet_index |= 0b10000000_00000000

                if self.ip_version == 4:
                    self.cfg["tunnel"]["subnet"] = "10.%d.%d.0/24" % (
                        (subnet_index & 0xFF), ((subnet_index >> 8)) & 0xFF)
                elif self.ip_version == 6:
                    self.cfg["tunnel"]["subnet"] = "fd42:1337:%x::/64" % subnet_index

            tmp_ip = split_ip(self.cfg["tunnel"]["subnet"])
            self.ip = (ip_address(tmp_ip) + 1).exploded

        cfgfile = None
        with NamedTemporaryFile(mode="w", delete=False) as f:
            yaml_dump(self.cfg, f)
            cfgfile = f.name

        try:
            self.proc = Popen(args=[self.bin, "-config", cfgfile],
                              stdin=DEVNULL, stderr=PIPE, text=True, executable=self.bin)

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
