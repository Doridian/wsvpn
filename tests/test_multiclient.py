from email.generator import Generator
from tests.bins import GoBin, new_clbin
from tests.packet_utils import basic_traffic_test
import pytest

@pytest.fixture(scope="function")
def clbin2() -> Generator:
    gobin = new_clbin()
    yield gobin
    gobin.stop()


def runtest(svbin: GoBin, clbins: list, one_iface: bool, mode: str) -> None:
    svbin.cfg["interface"]["one-interface-per-connection"] = one_iface
    svbin.cfg["tunnel"]["mode"] = mode

    if mode == "TAP" and not svbin.is_tap_supported():
        pytest.skip("TAP not supported on this platform")

    if one_iface and not svbin.is_one_interface_per_connection_supported():
        pytest.skip("One-Interface-Per-Connection not supported on this platform")


    try:
        for clbin in clbins:
            clbin.connect_to(svbin)

        svbin.start()
        svbin.assert_ready_ok()

        for clbin in clbins:
            clbin.start()
            clbin.assert_ready_ok()

        for clbin in clbins:
            basic_traffic_test(svbin=svbin, clbin=clbin)

    finally:
        for clbin in clbins:
            clbin.stop()


def test_run_e2e_oneiface_tap(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    runtest(svbin, [clbin, clbin2], False, "TAP")


def test_run_e2e_oneiface_tun(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    runtest(svbin, [clbin, clbin2], False, "TUN")


def test_run_e2e_manyiface_tap(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    runtest(svbin, [clbin, clbin2], True, "TAP")

def test_run_e2e_manyiface_tun(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    runtest(svbin, [clbin, clbin2], True, "TUN")

def test_run_e2e_manyiface_customname(svbin: GoBin, clbin: GoBin, clbin2: GoBin) -> None:
    svbin.cfg["interface"]["name"] = "wsserver"
    runtest(svbin, [clbin, clbin2], True, "TUN")
