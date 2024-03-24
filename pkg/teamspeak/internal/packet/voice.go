package packet

import (
	"Jabba_The_Bot/pkg/teamspeak/internal/protocol"
	"encoding/binary"
)

func NewVoicePacket(voiceID uint16, codec uint8, frame []byte) *protocol.C2SPacket {
	data := make([]byte, 3+len(frame))
	binary.BigEndian.PutUint16(data[0:2], voiceID)
	data[2] = codec
	copy(data[3:], frame)

	return &protocol.C2SPacket{
		Type: protocol.PTVoice,
		Data: data,
	}
}
