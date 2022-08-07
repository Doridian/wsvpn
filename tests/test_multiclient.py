from tests.bins import GoBin, new_clbin
from tests.packet_utils import basic_traffic_test


def configure(svbin: GoBin) -> None:
    svbin.cfg["interface"]["one-interface-per-connection"] = False

def runtest(svbin: GoBin, clbins: list, one_iface: bool, mode: str) -> None:
    svbin.cfg["interface"]["one-interface-per-connection"] = one_iface
    svbin.cfg["tunnel"]["mode"] = mode
    
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


def test_run_e2e_oneiface_tap(svbin: GoBin, clbin: GoBin) -> None:
    clbin2 = new_clbin()
    runtest(svbin, [clbin, clbin2], False, "TAP")


def test_run_e2e_oneiface_tun(svbin: GoBin, clbin: GoBin) -> None:
    clbin2 = new_clbin()
    runtest(svbin, [clbin, clbin2], False, "TUN")


def test_run_e2e_manyiface_tap(svbin: GoBin, clbin: GoBin) -> None:
    clbin2 = new_clbin()
    runtest(svbin, [clbin, clbin2], True, "TAP")


def test_run_e2e_manyiface_tun(svbin: GoBin, clbin: GoBin) -> None:
    clbin2 = new_clbin()
    runtest(svbin, [clbin, clbin2], True, "TUN")
