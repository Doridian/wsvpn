from curses import nonl
from tests.bins import GoBin
from tests.conftest import TLSCertSet
import scapy.sendrecv as scapy_sendrecv
import scapy.layers.l2 as scapy_l2
import scapy.layers.inet as scapy_inet
import scapy.plist as scapy_plist

def basic_traffic_test(svbin: GoBin, clbin: GoBin, ethernet: bool) -> None:
    pkts = []

    def pkt_add(pkt):
        raw_pkt = pkt
        if ethernet:
            raw_pkt = scapy_l2.Ether()/raw_pkt
        pkts.append((raw_pkt,pkt))

    pkt_add(scapy_inet.IP(src="192.168.3.2",version=4,ihl=5,len=20,chksum=0x3296,dst="192.168.3.1",ttl=1))

    for raw_pkt, pkt in pkts:
        send_iface = None
        recv_iface = None

        def sendpkt():
            scapy_sendrecv.sendp(x=[raw_pkt], iface=send_iface, count=1, return_packets=True)

        def dosniff() -> scapy_plist.PacketList:
            res: scapy_plist.PacketList = scapy_sendrecv.sniff(iface=recv_iface, started_callback=sendpkt, filter="ip" if ethernet else "", count=1, store=1, timeout=2)
            assert len(res.res) > 0
            actual_pkt = res.res[0].getlayer(type(pkt))
            assert actual_pkt == pkt

        send_iface = "wsvpns0"
        recv_iface = "wsvpnc"
        dosniff()

        send_iface = "wsvpnc"
        recv_iface = "wsvpns0"
        dosniff()
    

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

    basic_traffic_test(svbin=svbin, clbin=clbin, ethernet=False)

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

    basic_traffic_test(svbin=svbin, clbin=clbin, ethernet=False)

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

    basic_traffic_test(svbin=svbin, clbin=clbin, ethernet=False)

    clbin.stop()
    svbin.stop()
