package sockets_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/Doridian/wsvpn/shared/sockets/sockettest"
)

const receiveTimeout = 5 * time.Second

func testPacket(size int, seed byte) []byte {
	pkt := make([]byte, size)
	for i := range pkt {
		pkt[i] = seed + byte(i)
	}
	return pkt
}

func sendAndExpect(t *testing.T, from *sockettest.Peer, to *sockettest.Peer, pkt []byte) {
	t.Helper()
	from.Tun.Inject(pkt)
	got, err := to.Receive(receiveTimeout)
	if err != nil {
		t.Fatalf("packet of %d bytes: %v", len(pkt), err)
	}
	if !bytes.Equal(got, pkt) {
		t.Fatalf("packet of %d bytes corrupted in transit (got %d bytes)", len(pkt), len(got))
	}
}

func TestDataPath(t *testing.T) {
	sizes := []int{1, 64, 1200, 1500, 4000, 4096, 4097, 9000, 16000}

	for _, transport := range sockettest.Transports {
		for _, fragmentation := range []bool{false, true} {
			name := fmt.Sprintf("%s/fragmentation=%v", transport, fragmentation)
			t.Run(name, func(t *testing.T) {
				server, client := sockettest.NewSocketPair(t, sockettest.PairOptions{
					Transport:     transport,
					MTU:           16000,
					Fragmentation: fragmentation,
				})

				for _, size := range sizes {
					// WebTransport datagrams cannot carry more than ~1200 bytes,
					// fragmentation is what makes larger packets possible there.
					if transport == sockettest.TransportWebTransport && !fragmentation && size > 1184 {
						continue
					}
					t.Run(fmt.Sprintf("size=%d", size), func(t *testing.T) {
						sendAndExpect(t, client, server, testPacket(size, 1))
						sendAndExpect(t, server, client, testPacket(size, 2))
					})
				}
			})
		}
	}
}

// Many packets back to back, in both directions at once. Ordering is not
// guaranteed for WebTransport datagrams, and loss is possible in principle,
// so we only check the multiset of what arrived.
func TestDataPathBurst(t *testing.T) {
	const count = 500

	for _, transport := range sockettest.Transports {
		t.Run(transport, func(t *testing.T) {
			server, client := sockettest.NewSocketPair(t, sockettest.PairOptions{
				Transport:     transport,
				Fragmentation: true,
			})

			go func() {
				for i := 0; i < count; i++ {
					client.Tun.Inject(testPacket(1400, byte(i)))
					server.Tun.Inject(testPacket(1400, byte(i)))
				}
			}()

			// WebTransport datagrams have no backpressure: an unpaced burst
			// overflows quic-go's datagram queue, which drops silently
			minReceived := count
			timeout := receiveTimeout
			if transport == sockettest.TransportWebTransport {
				minReceived = count / 2
				timeout = time.Second
			}

			for _, dst := range []*sockettest.Peer{server, client} {
				for i := 0; i < count; i++ {
					pkt, err := dst.Receive(timeout)
					if err != nil {
						if i >= minReceived {
							t.Logf("datagram loss: received %d/%d", i, count)
							break
						}
						t.Fatalf("after %d/%d packets: %v", i, count, err)
					}
					if !bytes.Equal(pkt, testPacket(1400, pkt[0])) {
						t.Fatalf("corrupted packet")
					}
				}
			}
		})
	}
}

// Throughput of the full data path (fake device -> socket -> adapter ->
// loopback -> adapter -> socket -> fake device), one direction.
// Profile with e.g.:
//
//	go test ./shared/sockets -run '^$' -bench DataPath -cpuprofile cpu.out -memprofile mem.out
func BenchmarkDataPath(b *testing.B) {
	for _, transport := range sockettest.Transports {
		for _, size := range []int{64, 1400, 9000} {
			b.Run(fmt.Sprintf("%s/size=%d", transport, size), func(b *testing.B) {
				server, client := sockettest.NewSocketPair(b, sockettest.PairOptions{
					Transport:     transport,
					MTU:           9000,
					Fragmentation: true,
				})
				pkt := testPacket(size, 0)

				b.SetBytes(int64(size))
				b.ReportAllocs()
				b.ResetTimer()

				go func() {
					for i := 0; i < b.N; i++ {
						client.Tun.Inject(pkt)
					}
				}()

				received := 0
				for received < b.N {
					// Datagrams may be dropped under load, count instead of hanging
					if _, err := server.Receive(time.Second); err != nil {
						break
					}
					received++
				}
				b.StopTimer()
				b.ReportMetric(float64(b.N-received)*100/float64(b.N), "%lost")
			})
		}
	}
}
