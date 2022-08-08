from subprocess import check_call
from tempfile import mkdtemp
from os.path import join
from dataclasses import dataclass


@dataclass
class TLSCertSet:
    cn: str
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

    return TLSCertSet(cn=cn, ca=join(tmpdir, "cert.pem"), cert=join(tmpdir, "cert.pem"), key=join(tmpdir, "key.pem"), dir=tmpdir)
