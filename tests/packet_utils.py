from dataclasses import dataclass
from threading import Thread

from build import get_local_platform
from tests.bins import GoBin

import scapy.layers.all as scapy_layers
import scapy.packet as scapy_packet
import scapy.sendrecv as scapy_sendrecv
from scapy.interfaces import ifaces as scapy_ifaces


def is_ignored_payload(self):
    return isinstance(self, scapy_packet.NoPayload) or isinstance(self, scapy_packet.Padding)

# This is essentially the __eq__ function from Scapy, except it ignores values that are None in either item


def packet_equal(self, other):
    if is_ignored_payload(self):
        return is_ignored_payload(other)

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


def get_ip_version(ip: str) -> int:
    if ":" in ip:
        return 6
    return 4


@dataclass
class PktTuple:
    iface: str
    ip: str
    mac: str
    ip_version: int


class PacketTestRun:
    def __init__(self, pkts: list, src: PktTuple, dst: PktTuple) -> None:
        self.src = src
        self.dst = dst

        if src.ip_version != dst.ip_version:
            raise ValueError(
                f"ip_version mismatch src={src.ip_version} dst={dst.ip_version}")

        self.ip_version = src.ip_version
        self.pkts = []

        for pkt_in in pkts:
            pkt = pkt_in.copy()

            if self.ip_version == 4:
                ip_layer = pkt.getlayer(scapy_layers.IP)
            elif self.ip_version == 6:
                ip_layer = pkt.getlayer(scapy_layers.IPv6)
            else:
                raise ValueError(f"invalid ip_version: {self.ip_version}")
            ip_layer.src = self.src.ip
            ip_layer.dst = self.dst.ip

            eth_layer = pkt.getlayer(scapy_layers.Ether)
            if eth_layer:
                eth_layer.src = self.src.mac
                eth_layer.dst = self.dst.mac

            self.pkts.append(pkt)

        self._expected_packets = None

    def _send_packets(self):
        scapy_sendrecv.sendp(self.pkts, iface=self.src.iface,
                             realtime=False, count=1, return_packets=False)

    def _handle_packet(self, pkt):
        # Scapy likes decoding IPv6 payloads on TUN as IPv4...
        if self.ip_version == 6 and isinstance(pkt, scapy_layers.IP) and pkt.version == 6:
            pkt = scapy_layers.IPv6(bytes(pkt))

        for i, expected_pkt in enumerate(self._expected_packets):
            if packet_equal(pkt, expected_pkt):
                self._expected_packets.pop(i)
                break

        return len(self._expected_packets) == 0

    def run(self):
        self._expected_packets = self.pkts[:]

        t = Thread(target=self._send_packets)
        scapy_sendrecv.sniff(iface=self.dst.iface, started_callback=t.start,
                             stop_filter=self._handle_packet, count=0, store=False, timeout=2)
        t.join()

        assert len(self._expected_packets) == 0


class PacketTest:
    def __init__(self, svbin: GoBin, clbin: GoBin, ip_version: int = 4) -> None:
        self.svbin = svbin
        self.clbin = clbin
        self.ethernet = svbin.cfg["tunnel"]["mode"] == "TAP"
        self.pkts = []
        self.ip_version = ip_version
        self.need_dummy_layer = (not self.ethernet) and (
            get_local_platform() == "darwin")

    def pkt_add(self, pkt):
        if self.ethernet:
            pkt = scapy_layers.Ether()/pkt
        self.pkts.append(pkt)

    def simple_pkt(self, pktlen: int):
        payload = scapy_layers.UDP(sport=124, dport=125)
        if pktlen > 0:
            payload = payload / scapy_packet.Raw(bytes(b"A"*pktlen))

        if self.ip_version == 4:
            pkt = scapy_layers.IP(version=4) / payload
        elif self.ip_version == 6:
            pkt = scapy_layers.IPv6(version=6) / payload
        else:
            raise ValueError(f"Invalid ip_version {self.ip_version}")

        if self.need_dummy_layer:
            pkt = scapy_layers.Loopback(
                type=0x1e if self.ip_version == 6 else 0x2) / pkt

        self.pkt_add(pkt)

    def add_defaults(self, minimal: bool):
        self.simple_pkt(1)
        if minimal:
            return
        self.simple_pkt(10)
        self.simple_pkt(1000)
        self.simple_pkt(1300)

    def run(self):
        self.svbin.assert_ready_ok()
        self.clbin.assert_ready_ok()

        scapy_ifaces.reload()

        server_iface = self.svbin.get_interface_for(self.clbin)
        client_iface = self.clbin.get_interface_for()
        server_ip = self.svbin.get_ip()
        client_ip = self.clbin.get_ip()
        server_mac = None
        client_mac = None
        if self.ethernet:
            server_mac = self.svbin.get_mac_for(self.clbin)
            client_mac = self.clbin.get_mac_for()

        server_tuple = PktTuple(iface=server_iface, ip=server_ip,
                                mac=server_mac, ip_version=get_ip_version(server_ip))
        client_tuple = PktTuple(iface=client_iface, ip=client_ip,
                                mac=client_mac, ip_version=get_ip_version(client_ip))

        print("CLIENT SENDING, SERVER RECEIVING")
        test = PacketTestRun(self.pkts, src=client_tuple, dst=server_tuple)
        test.run()

        print("SERVER SENDING, CLIENT RECEIVING")
        test = PacketTestRun(self.pkts, src=server_tuple, dst=client_tuple)
        test.run()


def basic_traffic_test(svbin: GoBin, clbin: GoBin, minimal: bool = False, ip_version: int = 4) -> None:
    t = PacketTest(svbin=svbin, clbin=clbin, ip_version=ip_version)
    t.add_defaults(minimal=minimal)
    t.run()
