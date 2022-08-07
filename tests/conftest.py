from dataclasses import dataclass
from subprocess import check_call
from typing import Generator, Iterable
import pytest

from tempfile import mkdtemp, mktemp
from shutil import rmtree
from os.path import join
from os import remove

from tests.bins import GoBin

@dataclass
class TLSCertSet:
    ca: str
    cert: str
    key: str
    dir: str

def tls_cert_set(cn: str, conf: str) -> TLSCertSet:
    args = ["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-keyout", "key.pem", "-out", "cert.pem", "-sha256", "-days", "365", "-subj", f"/CN={cn}/"]
    if conf:
        args.append("-config")
        args.append(conf)

    tmpdir = mkdtemp()
    check_call(args, cwd=tmpdir)

    return TLSCertSet(ca=join(tmpdir, "cert.pem"), cert=join(tmpdir, "cert.pem"), key=join(tmpdir, "key.pem"), dir=tmpdir)

@pytest.fixture
def tls_cert_server() -> Generator:
    conftmp = mktemp()
    with open(conftmp, "w") as f:
        f.write("[req]\n")
        f.write("req_extensions = req_ext\n")
        f.write("x509_extensions = v3_req\n")
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

@pytest.fixture
def tls_cert_client() -> Generator:
    res = tls_cert_set("testclient", conf="")
    yield res
    rmtree(res.dir)

@pytest.fixture
def svbin() -> Generator:
    gobin = GoBin("server")
    yield gobin
    gobin.stop()

@pytest.fixture
def clbin() -> Generator:
    gobin = GoBin("client")
    yield gobin
    gobin.stop()
