import pytest
from typing import Generator
from tempfile import mktemp
from shutil import rmtree
from os import remove

from tests.bins import GoBin
from tests.tls_utils import tls_cert_set

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
        f.write("IP.1 = 127.0.0.1\n")

    res = tls_cert_set("localhost", conf=conftmp)

    yield res
    remove(conftmp)
    rmtree(res.dir)

@pytest.fixture(scope="session")
def tls_cert_client() -> Generator:
    res = tls_cert_set("testclient", conf="")
    yield res
    rmtree(res.dir)

@pytest.fixture(scope="function")
def svbin() -> Generator:
    gobin = GoBin("server")
    yield gobin
    gobin.stop()

@pytest.fixture(scope="function")
def clbin() -> Generator:
    gobin = GoBin("client")
    yield gobin
    gobin.stop()
