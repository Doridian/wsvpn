#!/usr/bin/env python3

from __future__ import annotations
from copy import deepcopy
from os.path import join, dirname, realpath
from os import remove, environ
from signal import SIGTERM
from socket import socket, AF_INET, SOCK_STREAM, SOCK_DGRAM
from subprocess import Popen, check_output, DEVNULL, PIPE, TimeoutExpired
from tempfile import NamedTemporaryFile
from threading import Thread, Condition
from traceback import format_exc
from typing import Any, Callable, Optional
from yaml import dump as yaml_dump, safe_load as yaml_load
from ipaddress import ip_address
from getmac import get_mac_address
from sys import executable
from itertools import count

from build import get_local_arch, get_local_platform
from tests.tls_utils import TLSCertSet

LOCAL_ARCH = get_local_arch()
LOCAL_PLATFORM = get_local_platform()

BIN_DIR = join(dirname(__file__), "..", "dist")

SCRIPT_HDL = join(dirname(__file__), "script_hdl.py")

# How long to wait for a process to come up (or for an event it reports),
# slow CI runners can override this
STARTUP_TIMEOUT = float(environ.get("WSVPN_TEST_STARTUP_TIMEOUT", "30"))
STOP_TIMEOUT = 5

_default_configs: dict = {}

# Every server gets its own subnet, independent of its (OS assigned) port
_subnet_ids = count(1)


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


def _find_free_port() -> int:
    # The server binds TCP and (with HTTP/3) UDP on the same port, so the
    # port needs to be free for both
    while True:
        with socket(AF_INET, SOCK_STREAM) as tcp:
            tcp.bind(("127.0.0.1", 0))
            port = tcp.getsockname()[1]
            try:
                with socket(AF_INET, SOCK_DGRAM) as udp:
                    udp.bind(("127.0.0.1", port))
            except OSError:
                continue
            return port


def split_ip(ipsub: str) -> str:
    return ipsub.split("/")[0]


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
            self.port = _find_free_port()
            self.subnet_id = next(_subnet_ids)
            self.cfg["server"]["listen"] = f"127.0.0.1:{self.port}"
            self.cfg["tunnel"]["subnet"] = None
            self.cfg["server"]["headers"] = {
                "X-Test-Header": ["test-header-value"]
            }
        else:
            self.port = None
            self.cfg["client"]["headers"] = {
                "X-Test-Header": ["test-header-value"]
            }
            self.cfg["firewall-mark"] = 1337

        self.ip = None

        # Guards every piece of state below that the reader thread updates
        self.cond = Condition()
        self.proc = None
        self.ready_ok = None
        self.exited = False
        self.reader_errors = []

        self.iface_names = {}
        self.auth_names = {}
        self.iface_macs = {}

        self.http_auth_enabled = False
        self.mtls_auth_enabled = False

        self.ip_version = 4

    def __repr__(self) -> str:
        return f"<{self.proj} port={self.port} ip={self.ip}>"

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

        self.cfg["client"]["server"] = f"{protocol}://{auth_str}127.0.0.1:{server.port}"

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

    def _wait_for(self, predicate: Callable[[], bool], what: str, timeout: float = STARTUP_TIMEOUT) -> None:
        with self.cond:
            if not self.cond.wait_for(predicate, timeout=timeout):
                raise TimeoutError(f"{self!r}: timed out after {timeout}s waiting for {what}")

    def wait_ready_or_done(self) -> None:
        # A process that neither became ready nor exited counts as not ready
        with self.cond:
            if not self.cond.wait_for(lambda: self.ready_ok is not None, timeout=STARTUP_TIMEOUT):
                self.ready_ok = False

    def start(self) -> None:
        if self.is_server and not self.cfg["tunnel"]["subnet"]:
            subnet_index = self.subnet_id
            if self.cfg["tunnel"]["mode"] == "TAP":
                subnet_index |= 0b10000000_00000000

            if self.ip_version == 4:
                self.cfg["tunnel"]["subnet"] = "10.%d.%d.0/24" % (
                    ((subnet_index >> 8) & 0xFF), (subnet_index & 0xFF))
            elif self.ip_version == 6:
                self.cfg["tunnel"]["subnet"] = "fd42:1337:%x::/64" % subnet_index

        if self.is_server:
            tmp_ip = split_ip(self.cfg["tunnel"]["subnet"])
            self.ip = (ip_address(tmp_ip) + 1).exploded

        super().start()

    def stop(self) -> None:
        if self.proc is not None and self.proc.poll() is None:
            self.proc.send_signal(SIGTERM)
            try:
                self.proc.wait(timeout=STOP_TIMEOUT)
            except TimeoutExpired:
                print(f"{self!r}: did not exit after SIGTERM, killing", flush=True)
                self.proc.kill()
                self.proc.wait()

        if self.is_alive():
            self.join()

    def handle_line(self, line: str) -> None:
        print(f"[{self.proj}:{self.port or ''}] {line}", flush=True)

        if self.is_server and "VPN server online at" in line:
            self._notify_ready(True)

        if "SCRIPT_HDL" not in line:
            return

        lspl = line.split(" ")[2:]

        with self.cond:
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
                    self.iface_names.pop("server", None)
                    self.ip = None

                if self.is_server:
                    ip = split_ip(lspl[1])
                    self.iface_names.pop(ip, None)
                    self.auth_names.pop(ip, None)

            else:
                raise ValueError(f"script called with invalid args: {lspl}")

            self.cond.notify_all()

    def get_ip(self) -> str:
        return self.ip

    def _wait_client_registered(self, clbin: GoBin) -> str:
        if not self.is_server:
            raise ValueError("Only servers track clients")

        client_ip = clbin.get_ip()
        if client_ip is None:
            raise ValueError(f"{clbin!r} has no IP, it is not connected")

        # The server reports a client's interface on its own stderr, which
        # we parse on a different thread than the client's: it might lag
        self._wait_for(lambda: client_ip in self.iface_names,
                       f"server side interface of client {client_ip}")
        return client_ip

    def get_auth_for(self, clbin: GoBin = None) -> str:
        client_ip = self._wait_client_registered(clbin)
        return self.auth_names[client_ip]

    def get_interface_for(self, clbin: GoBin = None) -> str:
        if self.is_client:
            # clbin does not matter here, we only have one iface
            return self.iface_names["server"]

        client_ip = self._wait_client_registered(clbin)
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
        with self.cond:
            if self.ready_ok is None:
                self.ready_ok = ok
            self.cond.notify_all()

    def assert_ready_ok(self, should: bool = True) -> None:
        self.wait_ready_or_done()
        assert self.ready_ok == should, f"{self!r}: expected ready={should}, got {self.ready_ok}"
        assert not self.reader_errors, f"{self!r}: errors parsing output: {self.reader_errors}"

    def run(self) -> None:
        cfgfile = None
        with NamedTemporaryFile(mode="w", delete=False) as f:
            yaml_dump(self.cfg, f)
            cfgfile = f.name

        try:
            self.proc = Popen(args=[self.bin, "-config", cfgfile],
                              stdin=DEVNULL, stderr=PIPE, text=True, executable=self.bin)

            # Keep draining stderr no matter what: if we stopped, the Go
            # process would eventually block on writing its log
            for line in self.proc.stderr:
                try:
                    self.handle_line(line.strip())
                except Exception:
                    self.reader_errors.append(format_exc())
                    print(format_exc(), flush=True)
        finally:
            with self.cond:
                self.exited = True
                if self.ready_ok is None:
                    self.ready_ok = False
                self.cond.notify_all()
            remove(cfgfile)


def new_clbin():
    return GoBin("client")


def new_svbin():
    return GoBin("server")
