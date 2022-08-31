from email.generator import Generator
from build import get_local_platform
from tests.bins import GoBin, new_clbin
from tests.packet_utils import basic_traffic_test
import pytest


@pytest.fixture(scope="function")
def clbin2() -> Generator:
    gobin = new_clbin()
    yield gobin
    gobin.stop()


def runtest(svbin: GoBin, clbins: list, one_iface: bool, mode: str, ip_version: int) -> None:
    svbin.cfg["interface"]["one-interface-per-connection"] = one_iface
    svbin.cfg["tunnel"]["mode"] = mode
    svbin.ip_version = ip_version

    if mode == "TAP" and not svbin.is_tap_supported():
        pytest.skip("TAP not supported on this platform")

    if one_iface and not svbin.is_one_interface_per_connection_supported(mode):
        pytest.skip(
            "One-Interface-Per-Connection not supported on this platform")

    if get_local_platform() == "windows":
        svbin.cfg["interface"]["name"] = "%s0" % mode
        for i, clbin in enumerate(clbins):
            clbin.cfg["interface"]["name"] = "%s%d" % (mode, (i + 1))

    try:
        for clbin in clbins:
            clbin.connect_to(svbin)

        svbin.start()
        svbin.assert_ready_ok()

        for clbin in clbins:
            clbin.start()
            clbin.assert_ready_ok()

        for clbin in clbins:
            basic_traffic_test(svbin=svbin, clbin=clbin, ip_version=ip_version)

    finally:
        for clbin in clbins:
            clbin.stop()


def test_run_e2e_oneiface_tap(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    runtest(svbin, [clbin, clbin2], False, "TAP", 4)


def test_run_e2e_oneiface_tun(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    runtest(svbin, [clbin, clbin2], False, "TUN", 4)


def test_run_e2e_manyiface_tap(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    runtest(svbin, [clbin, clbin2], True, "TAP", 4)


def test_run_e2e_manyiface_tun(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    runtest(svbin, [clbin, clbin2], True, "TUN", 4)


def test_run_e2e_manyiface_customname(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    svbin.cfg["interface"]["name"] = "utun6"
    runtest(svbin, [clbin, clbin2], True, "TUN", 4)


def test_run_e2e_oneiface_ipv6_tap(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    runtest(svbin, [clbin, clbin2], False, "TAP", 6)


def test_run_e2e_oneiface_ipv6_tun(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    runtest(svbin, [clbin, clbin2], False, "TUN", 4)


def test_run_e2e_manyiface_ipv6_tap(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    runtest(svbin, [clbin, clbin2], True, "TAP", 6)


def test_run_e2e_manyiface_ipv6_tun(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    runtest(svbin, [clbin, clbin2], True, "TUN", 6)
