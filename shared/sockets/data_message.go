package sockets

import (
	"bytes"
	"errors"
	"fmt"
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
	data      map[uint8][]byte
	time      time.Time
}

const fragmentExpiryTime = time.Second * time.Duration(30)

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

func (s *Socket) dataMessageHandler(message []byte) bool {
	// Ignore all packets before version negotiation
	if s.remoteProtocolVersion == UndeterminedProtocolVersion || !s.isReady || s.isClosing {
		return true
	}

	if !s.fragmentationEnabled {
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
			data:      make(map[uint8][]byte),
		}
		s.defragBuffer[packetID] = fragInfo
	}

	fragInfo.time = time.Now()
	buf := make([]byte, len(message)-5)
	copy(buf, message[5:])
	fragInfo.data[fragIndex] = buf
	if isLastFragment {
		fragInfo.lastIndex = int(fragIndex)
	}

	if len(fragInfo.data) == fragInfo.lastIndex+1 {
		delete(s.defragBuffer, packetID)
		s.defragLock.Unlock()

		buf := &bytes.Buffer{}
		for i := uint8(0); i <= uint8(fragInfo.lastIndex); i++ {
			buf.Write(fragInfo.data[i])
		}
		return s.processPacket(buf.Bytes())
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
	// Ignore all packets before version negotiation
	if s.remoteProtocolVersion == UndeterminedProtocolVersion || !s.isReady || s.isClosing {
		return nil
	}

	if !s.fragmentationEnabled {
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
		if err == nil || !errors.Is(err, adapters.ErrDataPayloadTooLarge) {
			break
		}
		newMaxLen := s.adapter.MaxDataPayloadLen()
		if newMaxLen == maxLen {
			break
		}
		s.log.Printf("Resending packet with max data payload size decreased from %d to %d", maxLen, newMaxLen)
	}
	return err
}

func (s *Socket) writePacketFragmented(data []byte, maxLen uint16) error {
	dataLen := uint16(len(data))

	buf := &bytes.Buffer{}
	if dataLen+1 <= maxLen {
		buf.WriteByte(0b10000000)
		buf.Write(data)
		return s.adapter.WriteDataMessage(buf.Bytes())
	}

	packetID := atomic.AddUint32(&s.lastFragmentID, 1)

	maxLen -= 5 // 5 byte header (frag|LF ID ID ID ID)!
	fragmentCount := uint16(dataLen / maxLen)
	if dataLen%maxLen != 0 {
		fragmentCount++
	}

	packetID1 := uint8(packetID % 0xFF)
	packetID2 := uint8((packetID >> 8) % 0xFF)
	packetID3 := uint8((packetID >> 16) % 0xFF)
	packetID4 := uint8((packetID >> 24) % 0xFF)
	for frag := uint16(0); frag < fragmentCount; frag++ {
		buf.Reset()

		fragFlag := uint8(frag)
		if frag == fragmentCount-1 {
			fragFlag |= 0b10000000
		}
		buf.WriteByte(fragFlag)
		buf.WriteByte(packetID4)
		buf.WriteByte(packetID3)
		buf.WriteByte(packetID2)
		buf.WriteByte(packetID1)

		fragStart := frag * maxLen
		fragEnd := fragStart + maxLen
		if fragEnd > dataLen {
			fragEnd = dataLen
		}
		buf.Write(data[fragStart:fragEnd])
		err := s.adapter.WriteDataMessage(buf.Bytes())
		if err != nil {
			return err
		}
	}

	return nil
}
