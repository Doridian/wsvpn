package sockets

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
)

// Packet wire format:
// First byte: 1 bit "last fragment", 7 bits uint for index within current packet (first fragment is 0, etc)
// a) If "last fragment" is set and index is 0, data that follows is payload
// b) In any other case, this is followed by a 32 bit identifier
// Example for case a): [10000000] PAYLOAD
// Example for case b): [00000000] [00000000 00000000 00000000 00000001] PAYLOAD_PART_0
//                      [10000001] [00000000 00000000 00000000 00000001] PAYLOAD_PART_1 (last fragment)

type fragmentsInfo struct {
	lastIndex int
	received  int
	totalLen  int
	data      [][]byte
	time      time.Time
}

const fragmentExpiryTime = time.Second * time.Duration(30)

const fragmentHeaderLen = 5

// Adapters copy outgoing messages before WriteDataMessage returns, so frame
// buffers can be reused instead of allocating one per packet
var frameBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0xFFFF+fragmentHeaderLen)
		return &buf
	},
}

func (s *Socket) processPacket(packet []byte) bool {
	if len(packet) < 1 {
		s.log.Printf("Packet too short for processing")
		return false
	}

	if s.packetHandler != nil {
		res, err := s.packetHandler.HandlePacket(s, packet)
		if err != nil {
			s.log.Printf("Error in packet handler: %v", err)
			return false
		}
		if res {
			return true
		}
	}

	if s.iface == nil {
		return true
	}
	_, err := s.iface.Interface.Write(packet)
	if err != nil {
		s.log.Printf("Error in interface write: %v", err)
		return false
	}
	return true
}

// Ignore all packets before version negotiation and after close
func (s *Socket) canPassData() bool {
	return s.remoteVersion() != UndeterminedProtocolVersion && s.isReady.Load() && !s.isClosing.Load()
}

func (s *Socket) dataMessageHandler(message []byte) bool {
	if !s.canPassData() {
		return true
	}

	if !s.fragmentationEnabled.Load() {
		return s.processPacket(message)
	}

	if len(message) < 2 { // At least a single byte actual body is needed
		s.log.Printf("Data message too short for processing")
		return false
	}

	fragHeader := message[0]
	if fragHeader == 0b10000000 { // Last fragment at index 0 => unfragmented packet
		return s.processPacket(message[1:])
	}

	if len(message) < 6 { // At least a single byte actual body is needed
		s.log.Printf("Data message too short for defragmenting")
		return false
	}

	fragIndex := fragHeader & 0b01111111
	isLastFragment := fragHeader&0b10000000 == 0b10000000
	packetID := (uint32(message[1]) << 24) | (uint32(message[2]) << 16) | (uint32(message[3]) << 8) | uint32(message[4])

	s.defragLock.Lock()
	fragInfo := s.defragBuffer[packetID]
	if fragInfo == nil {
		fragInfo = &fragmentsInfo{
			lastIndex: -1000, // Very small value as an indicator for "not set, yet"
			data:      make([][]byte, 0, 2),
		}
		s.defragBuffer[packetID] = fragInfo
	}

	fragInfo.time = time.Now()
	if int(fragIndex) >= len(fragInfo.data) {
		fragInfo.data = append(fragInfo.data, make([][]byte, int(fragIndex)+1-len(fragInfo.data))...)
	}
	if fragInfo.data[fragIndex] == nil { // Ignore duplicates
		buf := make([]byte, len(message)-fragmentHeaderLen)
		copy(buf, message[fragmentHeaderLen:])
		fragInfo.data[fragIndex] = buf
		fragInfo.received++
		fragInfo.totalLen += len(buf)
	}
	if isLastFragment {
		fragInfo.lastIndex = int(fragIndex)
	}

	if fragInfo.received == fragInfo.lastIndex+1 {
		delete(s.defragBuffer, packetID)
		s.defragLock.Unlock()

		packet := make([]byte, 0, fragInfo.totalLen)
		for _, frag := range fragInfo.data[:fragInfo.lastIndex+1] {
			packet = append(packet, frag...)
		}
		return s.processPacket(packet)
	}

	s.defragLock.Unlock()
	return true
}

func (s *Socket) cleanupFragmentsLoop() {
	for {
		select {
		case <-s.fragmentCleanupTicker.C:
			s.cleanupFragments()
		case <-s.closeChan:
			return
		}
	}
}

func (s *Socket) cleanupFragments() {
	s.defragLock.Lock()
	defer s.defragLock.Unlock()

	deleteIndices := make([]uint32, 0)

	for idx, fragInfo := range s.defragBuffer {
		if time.Since(fragInfo.time) > fragmentExpiryTime {
			deleteIndices = append(deleteIndices, idx)
		}
	}

	for _, idx := range deleteIndices {
		delete(s.defragBuffer, idx)
	}
}

func (s *Socket) WritePacket(data []byte) error {
	if !s.canPassData() {
		return nil
	}

	if !s.fragmentationEnabled.Load() {
		err := s.adapter.WriteDataMessage(data)
		if err != nil {
			s.CloseError(fmt.Errorf("error unfragmented data message: %v", err))
		}
		return err
	}

	dataLen := len(data)
	if dataLen <= 0 || dataLen > 0xFFFF {
		err := errors.New("packet size out of bounds (0 < size <= 65535)")
		s.CloseError(err)
		return err
	}

	var err error
	for {
		maxLen := s.adapter.MaxDataPayloadLen()
		err = s.writePacketFragmented(data, maxLen)
		if err == nil || !errors.Is(err, adapters.ErrDataPayloadLimitReduced) {
			break
		}
		newMaxLen := s.adapter.MaxDataPayloadLen()
		s.log.Printf("Resending packet with max data payload size decreased from %d to %d", maxLen, newMaxLen)
	}
	return err
}

func (s *Socket) writePacketFragmented(data []byte, maxLen uint16) error {
	dataLen := uint16(len(data))

	bufPtr := frameBufferPool.Get().(*[]byte)
	defer frameBufferPool.Put(bufPtr)
	buf := *bufPtr

	if int(dataLen)+1 <= int(maxLen) {
		buf[0] = 0b10000000
		n := copy(buf[1:], data)
		return s.adapter.WriteDataMessage(buf[:n+1])
	}

	packetID := atomic.AddUint32(&s.lastFragmentID, 1)

	maxLen -= fragmentHeaderLen // 5 byte header (frag|LF ID ID ID ID)!
	fragmentCount := uint16(dataLen / maxLen)
	if dataLen%maxLen != 0 {
		fragmentCount++
	}

	packetID1 := uint8(packetID % 0xFF)
	packetID2 := uint8((packetID >> 8) % 0xFF)
	packetID3 := uint8((packetID >> 16) % 0xFF)
	packetID4 := uint8((packetID >> 24) % 0xFF)
	for frag := uint16(0); frag < fragmentCount; frag++ {
		fragFlag := uint8(frag)
		if frag == fragmentCount-1 {
			fragFlag |= 0b10000000
		}
		buf[0] = fragFlag
		buf[1] = packetID4
		buf[2] = packetID3
		buf[3] = packetID2
		buf[4] = packetID1

		fragStart := frag * maxLen
		fragEnd := fragStart + maxLen
		if fragEnd > dataLen {
			fragEnd = dataLen
		}
		n := copy(buf[fragmentHeaderLen:], data[fragStart:fragEnd])
		err := s.adapter.WriteDataMessage(buf[:fragmentHeaderLen+n])
		if err != nil {
			return err
		}
	}

	return nil
}
