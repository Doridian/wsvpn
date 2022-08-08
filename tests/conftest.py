import pytest
from typing import Generator
from tempfile import mktemp
from shutil import rmtree
from os import remove

from tests.bins import new_clbin, new_svbin
from tests.tls_utils import tls_cert_set


TEST_USER = "testuser"
TEST_PASSWORD = "pAsSwOrD1234"

INVALID_TEXT = "invalid"

INVALID_HOST = "invalid.local"
VALID_HOST = "valid.local"


@pytest.fixture(scope="session")
def authenticator_config() -> Generator:
    aconf = mktemp()
    with open(aconf, "w") as f:
        f.write(f"{TEST_USER}:{TEST_PASSWORD}\n")
    
    yield aconf
    remove(aconf)


@pytest.fixture(scope="session")
def tls_cert_server() -> Generator:
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
        f.write("IP.1 = 127.0.0.1\n")

    res = tls_cert_set("localhost", conf=conftmp)
    remove(conftmp)

    yield res
    rmtree(res.dir)


@pytest.fixture(scope="session")
def tls_cert_client() -> Generator:
    res = tls_cert_set(TEST_USER, conf="")
    yield res
    rmtree(res.dir)


@pytest.fixture(scope="function")
def svbin() -> Generator:
    gobin = new_svbin()
    yield gobin
    gobin.stop()


@pytest.fixture(scope="function")
def clbin() -> Generator:
    gobin = new_clbin()
    yield gobin
    gobin.stop()
