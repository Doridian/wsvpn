from dataclasses import dataclass
from build import get_local_platform
from tests.bins import GoBin

import scapy.layers.all as scapy_layers
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

@dataclass
class PktTuple:
    iface: str
    ip: str
    mac: str


class PacketTestRun:
    def __init__(self, pkts: list, src: PktTuple, dst: PktTuple) -> None:
        self.src = src
        self.dst = dst

        self.pkts = []

        for pkt_in in pkts:
            pkt = pkt_in.copy()

            ip_layer = pkt.getlayer(scapy_layers.IP)
            ip_layer.src = self.src.ip
            ip_layer.dst = self.dst.ip

            eth_layer = pkt.getlayer(scapy_layers.Ether)
            if eth_layer:
                eth_layer.src = self.src.mac
                eth_layer.dst = self.dst.mac

            self.pkts.append(pkt)

        self._expected_packets = None


    def _send_packets(self):
        scapy_sendrecv.sendp(self.pkts, iface=self.src.iface, count=1, return_packets=False)


    def _handle_packet(self, pkt):
        for i, expected_pkt in enumerate(self._expected_packets):
            if packet_equal(pkt, expected_pkt):
                self._expected_packets.pop(i)
                break

        return len(self._expected_packets) == 0


    def run(self):
        self._expected_packets = self.pkts[:]
        scapy_sendrecv.sniff(iface=self.dst.iface, started_callback=self._send_packets, stop_filter=self._handle_packet, count=0, store=False, timeout=2)
        assert len(self._expected_packets) == 0


class PacketTest:
    def __init__(self, svbin: GoBin, clbin: GoBin) -> None:
        self.svbin = svbin
        self.clbin = clbin
        self.ethernet = svbin.cfg["tunnel"]["mode"] == "TAP"
        self.pkts = []
        self.need_dummy_layer = (not self.ethernet) and (get_local_platform() == "darwin")


    def pkt_add(self, pkt):
        if self.ethernet:
            pkt = scapy_layers.Ether()/pkt
        self.pkts.append(pkt)


    def simple_pkt(self, pktlen: int):
        payload = scapy_layers.ICMP(type=0, code=0, id=0x0, seq=0x0)
        if pktlen > 0:
            payload = payload / scapy_packet.Raw(bytes(b"A"*pktlen))
        
        pkt = scapy_layers.IP(version=4) / payload

        if self.need_dummy_layer:
            pkt = scapy_layers.Loopback(type=0x2) / pkt

        self.pkt_add(pkt)


    def add_defaults(self, minimal: bool):
        self.simple_pkt(10)
        if minimal:
            return
        self.simple_pkt(0)
        self.simple_pkt(1000)
        self.simple_pkt(1300)


    def run(self):
        self.svbin.assert_ready_ok()
        self.clbin.assert_ready_ok()

        server_iface = self.svbin.get_interface_for(self.clbin)
        client_iface = self.clbin.get_interface_for()
        server_ip = self.svbin.get_ip()
        client_ip = self.clbin.get_ip()
        server_mac = self.svbin.get_mac_for(self.clbin)
        client_mac = self.clbin.get_mac_for()

        server_tuple = PktTuple(iface=server_iface, ip=server_ip, mac=server_mac)
        client_tuple = PktTuple(iface=client_iface, ip=client_ip, mac=client_mac)

        print("CLIENT SENDING, SERVER RECEIVING")
        test = PacketTestRun(self.pkts, src=client_tuple, dst=server_tuple)
        test.run()

        print("SERVER SENDING, CLIENT RECEIVING")
        test = PacketTestRun(self.pkts, src=server_tuple, dst=client_tuple)
        test.run()


def basic_traffic_test(svbin: GoBin, clbin: GoBin, minimal: bool = False) -> None:
    t = PacketTest(svbin=svbin, clbin=clbin)
    t.add_defaults(minimal=minimal)
    t.run()
