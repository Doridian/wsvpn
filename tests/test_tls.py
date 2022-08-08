import pytest

from typing import Generator
from os import remove
from shutil import rmtree
from tempfile import mktemp
from tests.bins import GoBin
from tests.conftest import INVALID_HOST, VALID_HOST
from tests.packet_utils import basic_traffic_test
from tests.tls_utils import TLSCertSet, tls_cert_set


@pytest.fixture(scope="module")
def tls_cert_server_noip() -> Generator:
    conftmp = mktemp()
    with open(conftmp, "w") as f:
        f.write("[req]\n")
        f.write("default_bits = 2048\n")
        f.write("prompt = no\n")
        f.write("req_extensions = req_ext\n")
        f.write("x509_extensions = v3_req\n")
        f.write("distinguished_name = req_distinguished_name\n")
        f.write("[req_distinguished_name]\n")
        f.write("commonName = localhost\n")
        f.write("[req_ext]\n")
        f.write("subjectAltName = @alt_names\n")
        f.write("[v3_req]\n")
        f.write("subjectAltName = @alt_names\n")
        f.write("[alt_names]\n")
        f.write("DNS.1 = localhost\n")
        f.write(f"DNS.2 = {VALID_HOST}\n")

    res = tls_cert_set("localhost", conf=conftmp)
    remove(conftmp)

    yield res
    rmtree(res.dir)


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
