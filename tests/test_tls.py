from tests.bins import GoBin
from tests.conftest import INVALID_HOST, VALID_HOST
from tests.packet_utils import basic_traffic_test
from tests.tls_utils import TLSCertSet


def test_run_tls_wss_invalid_ca(svbin: GoBin, clbin: GoBin, tls_cert_server: TLSCertSet, tls_cert_server_noip: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server)
    clbin.enable_tls(tls_cert_server_noip) # This is for having an invalid CA
    clbin.connect_to(svbin, protocol="wss")

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok(should=False)


def test_run_tls_webtransport_invalid_ca(svbin: GoBin, clbin: GoBin, tls_cert_server: TLSCertSet, tls_cert_server_noip: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server)
    clbin.enable_tls(tls_cert_server_noip) # This is for having an invalid CA
    svbin.cfg["server"]["enable-http3"] = True
    clbin.connect_to(svbin, protocol="webtransport")

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok(should=False)


def test_run_tls_wss_invalid_name(svbin: GoBin, clbin: GoBin, tls_cert_server: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server)
    clbin.enable_tls(tls_cert_server)
    clbin.cfg["client"]["tls"]["server-name"] = INVALID_HOST
    clbin.connect_to(svbin, protocol="wss")

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok(should=False)


def test_run_tls_webtransport_invalid_name(svbin: GoBin, clbin: GoBin, tls_cert_server: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server)
    clbin.enable_tls(tls_cert_server)
    svbin.cfg["server"]["enable-http3"] = True
    clbin.cfg["client"]["tls"]["server-name"] = INVALID_HOST
    clbin.connect_to(svbin, protocol="webtransport")

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok(should=False)


def test_run_tls_wss_secondary_name(svbin: GoBin, clbin: GoBin, tls_cert_server: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server)
    clbin.enable_tls(tls_cert_server)
    clbin.cfg["client"]["tls"]["server-name"] = VALID_HOST
    clbin.connect_to(svbin, protocol="wss")

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok()

    basic_traffic_test(svbin=svbin, clbin=clbin, minimal=True)


def test_run_tls_webtransport_secondary_name(svbin: GoBin, clbin: GoBin, tls_cert_server: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server)
    clbin.enable_tls(tls_cert_server)
    svbin.cfg["server"]["enable-http3"] = True
    clbin.cfg["client"]["tls"]["server-name"] = VALID_HOST
    clbin.connect_to(svbin, protocol="webtransport")

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok()

    basic_traffic_test(svbin=svbin, clbin=clbin, minimal=True)


def test_run_tls_wss_invalid_default_name(svbin: GoBin, clbin: GoBin, tls_cert_server_noip: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server_noip)
    clbin.enable_tls(tls_cert_server_noip)
    clbin.connect_to(svbin, protocol="wss")

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok(should=False)


def test_run_tls_webtransport_invalid_default_name(svbin: GoBin, clbin: GoBin, tls_cert_server_noip: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server_noip)
    clbin.enable_tls(tls_cert_server_noip)
    svbin.cfg["server"]["enable-http3"] = True
    clbin.connect_to(svbin, protocol="webtransport")

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok(should=False)
