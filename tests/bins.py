#!/usr/bin/env python3

from copy import deepcopy
import pytest

from os.path import join, dirname
from asyncio.subprocess import DEVNULL, PIPE
from os import getenv, remove
from signal import SIGTERM
from subprocess import Popen
from tempfile import mktemp
from threading import Thread, Condition
from typing import Any, Optional
from yaml import dump as yaml_dump

from build import get_local_arch, get_local_platform

LOCAL_ARCH = get_local_arch()
LOCAL_PLATFORM = get_local_platform()

BASIC_CONFIG_SERVER = {
    "server": {
        "listen": "127.0.0.1:9000",
    },
    "tunnel": {
        "mtu": 1420,
    },
    "interface": {
        "name": "utun10",
        "one-interface-per-connection": True,
    },
}
BASIC_CONFIG_CLIENT = {
    "client": {
        "server": "ws://127.0.0.1:9000",
    },
    "interface": {
        "name": "utun20",
    },
}

BIN_DIR = join(dirname(__file__), "../dist/")

class GoBin(Thread):
    def __init__(self, proj: str) -> None:
        super().__init__(daemon=True)

        self.proj = proj
        self.bin = join(BIN_DIR, f"{proj}-{LOCAL_PLATFORM}-{LOCAL_ARCH}")
        self.cfg = self._get_basic_config()

        self.proc_wait_cond = Condition()
        self.is_ready_or_done = False
        self.proc = None
        self.ready_ok = None
    
    def _get_basic_config(self) -> Any:
        if self.proj == "client":
            return deepcopy(BASIC_CONFIG_CLIENT)
        elif self.proj == "server":
            return deepcopy(BASIC_CONFIG_SERVER)
        else:
            raise ValueError("Invalid proj")

    def wait_ready_or_done(self, timeout: Optional[int] = 5) -> None:
        while not self.is_ready_or_done:
            self.proc_wait_cond.acquire()
            self.proc_wait_cond.wait(timeout=timeout)
            self.proc_wait_cond.release()

    def stop(self) -> None:
        if self.proc is not None and self.proc.returncode is None:
            self.proc.send_signal(SIGTERM)

        if self.is_alive():
            self.join(timeout=5)
            if self.proc is not None:
                self.proc.kill()
            self.join()

    def handle_line(self, line: str) -> None:
        if "VPN server online at" in line or "Configured interface, VPN online" in line:
            self._notify_ready(True)
        print(line)

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

            while True:
                res = self.proc.stderr.readline()
                print(res)
                if not res:
                    break
                self.handle_line(res.strip())
        finally:
            self._notify_ready(False)
            remove(cfgfile)
