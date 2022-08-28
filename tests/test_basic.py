import pytest
from build import get_local_platform

from tests.bins import GoBin
from tests.tls_utils import TLSCertSet
from tests.packet_utils import basic_traffic_test


def test_run_e2e_wss(svbin: GoBin, clbin: GoBin, tls_cert_server: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server)
    clbin.enable_tls(tls_cert_server)
    clbin.connect_to(svbin, protocol="wss")

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok()

    basic_traffic_test(svbin=svbin, clbin=clbin)


def test_run_e2e_webtransport(svbin: GoBin, clbin: GoBin, tls_cert_server: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server)
    clbin.enable_tls(tls_cert_server)
    svbin.cfg["server"]["enable-http3"] = True
    clbin.connect_to(svbin, protocol="webtransport")

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok()

    basic_traffic_test(svbin=svbin, clbin=clbin)


def test_run_server(svbin: GoBin) -> None:
    svbin.start()
    svbin.assert_ready_ok()


def test_run_e2e_tun(svbin: GoBin, clbin: GoBin) -> None:
    svbin.cfg["tunnel"]["mode"] = "TUN"
    clbin.connect_to(svbin)

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok()

    basic_traffic_test(svbin=svbin, clbin=clbin)


def test_run_e2e_tap(svbin: GoBin, clbin: GoBin) -> None:
    if not svbin.is_tap_supported():
        pytest.skip("TAP not supported on this platform")

    svbin.cfg["tunnel"]["mode"] = "TAP"

    if get_local_platform() == "windows":
        svbin.cfg["interface"]["name"] = "TAP0"
        clbin.cfg["interface"]["name"] = "TAP1"

    clbin.connect_to(svbin)

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok()

    basic_traffic_test(svbin=svbin, clbin=clbin)
