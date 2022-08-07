from fcntl import ioctl
from platform import system
from socket import AF_INET, SOCK_DGRAM, socket
from struct import pack
from tests.conftest import GoBin

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


# https://stackoverflow.com/a/4789267
def get_mac(ifname):
    s = socket(AF_INET, SOCK_DGRAM)
    info = ioctl(s.fileno(), 0x8927,  pack('256s', bytes(ifname, 'utf-8')[:15]))
    return ':'.join('%02x' % b for b in info[18:24])


class PacketTest:
    def __init__(self, svbin: GoBin, clbin: GoBin) -> None:
        self.svbin = svbin
        self.clbin = clbin
        self.ethernet = svbin.cfg["tunnel"]["mode"] == "TAP"
        self.pkts = []
        self.need_dummy_layer = (not self.ethernet) and (system() == "Darwin")


    def pkt_add(self, pkt):
        if self.ethernet:
            pkt = scapy_layers.Ether()/pkt
        self.pkts.append((pkt, pkt))


    def simple_pkt(self, pktlen: int):
        payload = scapy_layers.ICMP(type=0, code=0, id=0x0, seq=0x0)
        if pktlen > 0:
            payload = payload / scapy_packet.Raw(bytes(b"A"*pktlen))
        
        pkt = scapy_layers.IP(version=4) / payload

        if self.need_dummy_layer:
            pkt = scapy_layers.Loopback(type=0x2) / pkt

        self.pkt_add(pkt)


    def add_defaults(self):
        self.simple_pkt(0)
        self.simple_pkt(10)
        self.simple_pkt(1000)
        self.simple_pkt(1300)


    def run(self):
        self.svbin.assert_ready_ok()
        self.clbin.assert_ready_ok()

        for pkt, raw_pkt in self.pkts:
            src_iface = None
            dst_iface = None
            src_ip = None
            dst_ip = None

            def sendpkt():
                scapy_sendrecv.sendp(raw_pkt, iface=src_iface, count=1, return_packets=True)

            def dosniff() -> scapy_plist.PacketList:
                ip_layer = pkt.getlayer(scapy_layers.IP)
                ip_layer.src = src_ip
                ip_layer.dst = dst_ip

                eth_layer = pkt.getlayer(scapy_layers.Ether)
                if eth_layer:
                    eth_layer.src = get_mac(src_iface)
                    eth_layer.dst = get_mac(dst_iface)

                res: scapy_plist.PacketList = scapy_sendrecv.sniff(iface=dst_iface, started_callback=sendpkt, filter="ip" if system() == "Linux" else None, count=1, store=1, timeout=2)
                assert len(res.res) > 0

                actual_pkt = res.res[0]

                try:
                    assert packet_equal(pkt, actual_pkt)
                except Exception:
                    print("Expected packet:")
                    pkt.show()

                    print("Actual packet:")
                    actual_pkt.show()

                    raise

            server_iface = self.svbin.get_interface_for(self.clbin)
            client_iface = self.clbin.get_interface_for()
            server_ip = self.svbin.get_ip()
            client_ip = self.clbin.get_ip()

            src_iface = client_iface
            dst_iface = server_iface
            src_ip = client_ip
            dst_ip = server_ip
            dosniff()

            src_iface = server_iface
            dst_iface = client_iface
            src_ip = server_ip
            dst_ip = client_ip
            dosniff()


def basic_traffic_test(svbin: GoBin, clbin: GoBin) -> None:
    t = PacketTest(svbin=svbin, clbin=clbin)
    t.add_defaults()
    t.run()
