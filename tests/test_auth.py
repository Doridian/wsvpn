from os import remove
from tempfile import mktemp
from typing import Optional
from tests.bins import GoBin, new_clbin
from tests.conftest import INVALID_TEXT, TEST_PASSWORD, TEST_USER
from tests.tls_utils import TLSCertSet
from tests.packet_utils import basic_traffic_test
import pytest


def run_client_auth(svbin: GoBin, protocol: str, tls_cert_server: Optional[TLSCertSet], mtls: Optional[TLSCertSet], user: str, password: str, should_be_ok: bool) -> None:
    clbin = new_clbin()

    try:
        clbin.connect_to(server=svbin, protocol=protocol, user=user, password=password)

        if tls_cert_server:
            clbin.cfg["client"]["tls"]["ca"] = tls_cert_server.ca

        if mtls:
            clbin.cfg["client"]["tls"]["key"] = mtls.key
            clbin.cfg["client"]["tls"]["certificate"] = mtls.cert
        else:
            clbin.cfg["client"]["tls"]["key"] = ""
            clbin.cfg["client"]["tls"]["certificate"] = ""

        clbin.start()
        clbin.assert_ready_ok(should=should_be_ok)

        if should_be_ok:
            basic_traffic_test(svbin=svbin, clbin=clbin, minimal=True)

    finally:
        clbin.stop()


def run_auth_server(svbin: GoBin, protocol: str, tls_cert_server: Optional[TLSCertSet], mtls_on: bool, mtls_server: Optional[TLSCertSet], mtls_clients: list[TLSCertSet] = None, authenticator: str = "allow-all", authenticator_config: str = "") -> None:
    if mtls_server and mtls_on:
        svbin.cfg["server"]["tls"]["client-ca"] = mtls_server.ca
    else:
        svbin.cfg["server"]["tls"]["client-ca"] = ""

    svbin.cfg["server"]["authenticator"]["type"] = authenticator
    svbin.cfg["server"]["authenticator"]["config"] = authenticator_config

    svbin.start()
    svbin.assert_ready_ok()

    htpasswd_on = authenticator == "htpasswd"

    if not mtls_clients:
        mtls_clients = [mtls_server]

    for mtls_client in mtls_clients:
        mtls_valid = (not mtls_server) or (mtls_server == mtls_client)
        if mtls_valid and mtls_client and htpasswd_on:
            mtls_valid = mtls_client.cn == TEST_USER

        # If we can't do mTLS (WS plaintext), these tests are pointless!
        if mtls_server:
            # Valid mTLS with valid user
            run_client_auth(svbin=svbin, tls_cert_server=tls_cert_server, protocol=protocol, mtls=mtls_client, user=TEST_USER, password=TEST_PASSWORD, should_be_ok=mtls_valid)
            # Valid mTLS with no user
            run_client_auth(svbin=svbin, tls_cert_server=tls_cert_server, protocol=protocol, mtls=mtls_client, user="", password="", should_be_ok=(not htpasswd_on and mtls_valid))

        # No mTLS with valid user
        run_client_auth(svbin=svbin, tls_cert_server=tls_cert_server, protocol=protocol, mtls=None, user=TEST_USER, password=TEST_PASSWORD, should_be_ok=(not mtls_on))
        # No mTLS with no user
        run_client_auth(svbin=svbin, tls_cert_server=tls_cert_server, protocol=protocol, mtls=None, user="", password="", should_be_ok=(not mtls_on and not htpasswd_on))
        # Valid mTLS with invalid user
        run_client_auth(svbin=svbin, tls_cert_server=tls_cert_server, protocol=protocol, mtls=mtls_client, user=INVALID_TEXT, password=TEST_PASSWORD, should_be_ok=(not htpasswd_on and mtls_valid))
        # Valid mTLS with invalid password
        run_client_auth(svbin=svbin, tls_cert_server=tls_cert_server, protocol=protocol, mtls=mtls_client, user=TEST_USER, password=INVALID_TEXT, should_be_ok=(not htpasswd_on and mtls_valid))


# WebSocket
def test_run_e2e_ws_htpasswd(svbin: GoBin, authenticator_config: str) -> None:
    run_auth_server(svbin=svbin, tls_cert_server=None, mtls_server=None, mtls_on=False, protocol="ws", authenticator="htpasswd", authenticator_config=authenticator_config)


# WebSocket Secure
def test_run_e2e_wss_dual(svbin: GoBin, tls_cert_server: TLSCertSet, tls_cert_client: TLSCertSet, tls_cert_client2: TLSCertSet, authenticator_config: str) -> None:
    svbin.enable_tls(tls_cert_server)

    run_auth_server(svbin=svbin, protocol="wss", tls_cert_server=tls_cert_server, mtls_server=tls_cert_client, mtls_clients=[tls_cert_client,tls_cert_client2], mtls_on=True, authenticator="htpasswd", authenticator_config=authenticator_config)


def test_run_e2e_wss_dual_mismatch(svbin: GoBin, tls_cert_server: TLSCertSet, tls_cert_client: TLSCertSet, tls_cert_client_invalid_user: TLSCertSet, authenticator_config: str) -> None:
    svbin.enable_tls(tls_cert_server)

    run_auth_server(svbin=svbin, protocol="wss", tls_cert_server=tls_cert_server, mtls_server=tls_cert_client, mtls_clients=[tls_cert_client_invalid_user], mtls_on=True, authenticator="htpasswd", authenticator_config=authenticator_config)


def test_run_e2e_wss_mtls(svbin: GoBin, tls_cert_server: TLSCertSet, tls_cert_client: TLSCertSet, tls_cert_client2: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server)

    run_auth_server(svbin=svbin, protocol="wss", tls_cert_server=tls_cert_server, mtls_server=tls_cert_client, mtls_clients=[tls_cert_client,tls_cert_client2], mtls_on=True)


def test_run_e2e_wss_htpasswd(svbin: GoBin, tls_cert_server: TLSCertSet, tls_cert_client: TLSCertSet, authenticator_config: str) -> None:
    svbin.enable_tls(tls_cert_server)

    run_auth_server(svbin=svbin, protocol="wss", tls_cert_server=tls_cert_server, mtls_server=tls_cert_client, mtls_on=False, authenticator="htpasswd", authenticator_config=authenticator_config)


# WebTransport
def test_run_e2e_webtransport_dual(svbin: GoBin, tls_cert_server: TLSCertSet, tls_cert_client: TLSCertSet, tls_cert_client2: TLSCertSet, authenticator_config: str) -> None:
    svbin.enable_tls(tls_cert_server)
    svbin.cfg["server"]["enable-http3"] = True

    run_auth_server(svbin=svbin, protocol="webtransport", tls_cert_server=tls_cert_server, mtls_server=tls_cert_client, mtls_clients=[tls_cert_client,tls_cert_client2], mtls_on=True, authenticator="htpasswd", authenticator_config=authenticator_config)


def test_run_e2e_webtransport_dual_mismatch(svbin: GoBin, tls_cert_server: TLSCertSet, tls_cert_client: TLSCertSet, tls_cert_client_invalid_user: TLSCertSet, authenticator_config: str) -> None:
    svbin.enable_tls(tls_cert_server)
    svbin.cfg["server"]["enable-http3"] = True

    run_auth_server(svbin=svbin, protocol="webtransport", tls_cert_server=tls_cert_server, mtls_server=tls_cert_client, mtls_clients=[tls_cert_client_invalid_user], mtls_on=True, authenticator="htpasswd", authenticator_config=authenticator_config)


def test_run_e2e_webtransport_mtls(svbin: GoBin, tls_cert_server: TLSCertSet, tls_cert_client: TLSCertSet, tls_cert_client2: TLSCertSet) -> None:
    svbin.enable_tls(tls_cert_server)
    svbin.cfg["server"]["enable-http3"] = True

    run_auth_server(svbin=svbin, protocol="webtransport", tls_cert_server=tls_cert_server, mtls_server=tls_cert_client, mtls_clients=[tls_cert_client,tls_cert_client2], mtls_on=True)


def test_run_e2e_webtransport_htpasswd(svbin: GoBin, tls_cert_server: TLSCertSet, tls_cert_client: TLSCertSet, authenticator_config: str) -> None:
    svbin.enable_tls(tls_cert_server)
    svbin.cfg["server"]["enable-http3"] = True

    run_auth_server(svbin=svbin, protocol="webtransport", tls_cert_server=tls_cert_server, mtls_server=tls_cert_client, mtls_on=False, authenticator="htpasswd", authenticator_config=authenticator_config)
