from tests.bins import GoBin
from tests.conftest import TLSCertSet

def basic_traffic_test(svbin: GoBin, clbin: GoBin) -> None:
    # TODO: Actually send back and forth some packets here...
    pass

def test_run_e2e_wss(svbin: GoBin, clbin: GoBin, tls_cert_server: TLSCertSet) -> None:
    svbin.cfg["server"]["tls"] = {
        "key": tls_cert_server.key,
        "certificate": tls_cert_server.cert,
    }

    clbin.cfg["client"]["tls"] = {
        "ca": tls_cert_server.ca,
    }
    clbin.cfg["client"]["server"] = "wss://127.0.0.1:9000"

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok()

    basic_traffic_test(svbin=svbin, clbin=clbin)

    clbin.stop()
    svbin.stop()

def test_run_e2e_webtransport(svbin: GoBin, clbin: GoBin, tls_cert_server: TLSCertSet) -> None:
    svbin.cfg["server"]["tls"] = {
        "key": tls_cert_server.key,
        "certificate": tls_cert_server.cert,
    }
    svbin.cfg["server"]["enable-http3"] = True

    clbin.cfg["client"]["tls"] = {
        "ca": tls_cert_server.ca,
    }
    clbin.cfg["client"]["server"] = "webtransport://127.0.0.1:9000"

    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok()

    basic_traffic_test(svbin=svbin, clbin=clbin)

    clbin.stop()
    svbin.stop()    

def test_run_server(svbin: GoBin) -> None:
    svbin.start()
    svbin.assert_ready_ok()
    svbin.stop()

def test_run_e2e_base(svbin: GoBin, clbin: GoBin) -> None:
    svbin.start()
    svbin.assert_ready_ok()

    clbin.start()
    clbin.assert_ready_ok()

    basic_traffic_test(svbin=svbin, clbin=clbin)

    clbin.stop()
    svbin.stop()
