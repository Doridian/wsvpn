package sockets

import (
	"bytes"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

// Packet wire format:
// First byte: 1 bit "last fragment", 7 bits uint for index within current packet (first fragment is 0, etc)
// a) If "last fragment" is set and index is 0, data that follows is payload
// b) In any other case, this is followed by a 16 bit identifier
// Example for case a): [10000000] PAYLOAD
// Example for case b): [00000000] [00000000 00000001] PAYLOAD_PART_0
//                      [10000001] [00000000 00000001] PAYLOAD_PART_1 (last fragment)

type fragmentsInfo struct {
	lastIndex int
	data      map[uint8][]byte
	time      time.Time
}

func (s *Socket) processPacket(packet []byte) bool {
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
	s.iface.Write(packet)
	return true
}

func (s *Socket) dataMessageHandler(message []byte) bool {
	fragHeader := message[0]
	if fragHeader == 0b10000000 {
		return s.processPacket(message[1:])
	}

	fragIndex := fragHeader & 0b01111111
	isLastFragment := fragHeader&0b10000000 == 0b10000000
	packetId := (uint16(message[1]) << 8) | uint16(message[2])

	s.defragLock.Lock()
	fragInfo := s.defragBuffer[packetId]
	if fragInfo == nil {
		fragInfo = &fragmentsInfo{
			lastIndex: -1000, // Very small value as an indicator for "not set, yet"
			data:      make(map[uint8][]byte),
			time:      time.Now(),
		}
		s.defragBuffer[packetId] = fragInfo
	}

	fragInfo.data[fragIndex] = message[3:]
	if isLastFragment {
		fragInfo.lastIndex = int(fragIndex)
	}

	if len(fragInfo.data) == fragInfo.lastIndex+1 {
		delete(s.defragBuffer, packetId)
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

func (s *Socket) sendDataWithError(data []byte) error {
	err := s.adapter.WriteDataMessage(data)
	if err != nil {
		s.CloseError(fmt.Errorf("error sending data message: %v", err))
	}
	return err
}

func (s *Socket) WritePacket(data []byte) error {
	realDataLen := len(data)
	if realDataLen <= 0 || realDataLen > 0xFFFF {
		err := errors.New("packet size out of bounds")
		s.CloseError(err)
		return err
	}

	maxLen := s.adapter.MaxDataPayloadLen()
	dataLen := uint16(realDataLen)

	buf := &bytes.Buffer{}
	if dataLen+1 <= maxLen {
		buf.WriteByte(0b10000000)
		buf.Write(data)
		return s.sendDataWithError(buf.Bytes())
	}

	packetId := uint16(atomic.AddUint32(&s.lastFragmentId, 1) % 0xFFFF)

	maxLen -= 3 // 3 byte header!
	fragmentCount := uint16(dataLen / maxLen)
	if dataLen%maxLen != 0 {
		fragmentCount++
	}

	packetIdLow := byte(packetId % 0xFF)
	packetIdHigh := byte((packetId >> 8) % 0xFF)
	for frag := uint16(0); frag < fragmentCount; frag++ {
		buf.Reset()
		buf.WriteByte(0b10000000 | uint8(frag))
		buf.WriteByte(packetIdHigh)
		buf.WriteByte(packetIdLow)

		fragStart := frag * maxLen
		fragEnd := fragStart + maxLen
		if fragEnd > dataLen {
			fragEnd = dataLen
		}
		buf.Write(data[fragStart:fragEnd])
		err := s.sendDataWithError(buf.Bytes())
		if err != nil {
			return err
		}
	}

	return nil
}
