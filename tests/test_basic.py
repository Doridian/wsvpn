from tests.bins import GoBin
from tests.tls_utils import TLSCertSet
from tests.packet_utils import PacketTest

def basic_traffic_test(svbin: GoBin, clbin: GoBin, ethernet: bool) -> None:
    t = PacketTest(svbin=svbin, clbin=clbin, ethernet=ethernet)
    t.add_defaults()
    t.run()


def test_run_e2e_wss(svbin: GoBin, clbin: GoBin, tls_cert_server: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server)
    clbin.enable_tls(tls_cert_server)
    clbin.connect_to(svbin, protocol="wss")

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok()

    basic_traffic_test(svbin=svbin, clbin=clbin, ethernet=False)


def test_run_e2e_webtransport(svbin: GoBin, clbin: GoBin, tls_cert_server: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server)
    clbin.enable_tls(tls_cert_server)
    svbin.cfg["server"]["enable-http3"] = True
    clbin.connect_to(svbin, protocol="webtransport")

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok()

    basic_traffic_test(svbin=svbin, clbin=clbin, ethernet=False)


def test_run_server(svbin: GoBin) -> None:
    svbin.start()
    svbin.assert_ready_ok()


def test_run_e2e_base(svbin: GoBin, clbin: GoBin) -> None:
    clbin.connect_to(svbin)

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok()

    basic_traffic_test(svbin=svbin, clbin=clbin, ethernet=False)
