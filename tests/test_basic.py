from tests.bins import GoBin
from tests.conftest import TLSCertSet
import scapy.layers.all as scapy_layers
import scapy.plist as scapy_plist
import scapy.packet as scapy_packet
import scapy.sendrecv as scapy_sendrecv

# This is essentially the __eq__ function from Scapy, except it ignores values that are None in either item
def packet_equal(self, other):
    if isinstance(self, scapy_packet.NoPayload):
        return self == other

    if not isinstance(other, self.__class__):
        return False

    for f in self.fields_desc:
        if f not in other.fields_desc:
            return False
        
        self_val = self.getfieldval(f.name)
        other_val = other.getfieldval(f.name)

        if self_val is not None and other_val is not None and self_val != other_val:
            return False

    return packet_equal(self.payload, other.payload)

def basic_traffic_test(svbin: GoBin, clbin: GoBin, ethernet: bool) -> None:
    pkts = []

    def pkt_add(pkt):
        if ethernet:
            pkt = scapy_layers.Ether()/pkt
        pkts.append(pkt)

    def simple_pkt(pktlen: int):
        payload = scapy_layers.ICMP(type=0, code=0, id=0x0, seq=0x0, chksum=0xf7ff)
        if pktlen > 0:
            payload = payload / scapy_packet.Raw(bytes(b"A"*pktlen))
        pkt_add(scapy_layers.IP(src="192.168.3.2",id=len(pkts)+1,version=4,ihl=5,dst="192.168.3.1",ttl=1) / payload)

    simple_pkt(0)
    simple_pkt(10)
    simple_pkt(1000)
    simple_pkt(2000)
    simple_pkt(3000)

    for pkt in pkts:
        send_iface = None
        recv_iface = None

        def sendpkt():
            scapy_sendrecv.sendp(x=[pkt], iface=send_iface, count=1, return_packets=True)

        def dosniff() -> scapy_plist.PacketList:
            res: scapy_plist.PacketList = scapy_sendrecv.sniff(iface=recv_iface, started_callback=sendpkt, filter="ip" if ethernet else "", count=1, store=1, timeout=2)
            assert len(res.res) > 0

            actual_pkt = res.res[0]

            assert packet_equal(pkt, actual_pkt)

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
