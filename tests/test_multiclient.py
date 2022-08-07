from tests.bins import GoBin, new_clbin
from tests.packet_utils import basic_traffic_test


def configure(svbin: GoBin) -> None:
    svbin.cfg["interface"]["one-interface-per-connection"] = False


def test_run_e2e_tun(svbin: GoBin, clbin: GoBin) -> None:
    clbin2 = new_clbin()

    configure(svbin=svbin)
    svbin.cfg["tunnel"]["mode"] = "TUN"
    clbin.connect_to(svbin)
    clbin2.connect_to(svbin)

    svbin.start()
    svbin.assert_ready_ok()

    try:
        clbin.start()
        clbin.assert_ready_ok()
        clbin2.start()
        clbin2.assert_ready_ok()

        basic_traffic_test(svbin=svbin, clbin=clbin)
        basic_traffic_test(svbin=svbin, clbin=clbin2)
    finally:
        clbin2.stop()


def test_run_e2e_tap(svbin: GoBin, clbin: GoBin) -> None:
    clbin2 = new_clbin()

    configure(svbin=svbin)
    svbin.cfg["tunnel"]["mode"] = "TAP"
    clbin.connect_to(svbin)
    clbin2.connect_to(svbin)

    svbin.start()
    svbin.assert_ready_ok()

    try:
        clbin.start()
        clbin.assert_ready_ok()
        clbin2.start()
        clbin2.assert_ready_ok()

        basic_traffic_test(svbin=svbin, clbin=clbin)
        basic_traffic_test(svbin=svbin, clbin=clbin2)
    finally:
        clbin2.stop()
