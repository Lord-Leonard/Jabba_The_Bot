package protocol

import (
	"encoding/binary"
	"fmt"
	"strings"
)

type PacketType uint8

const (
	PTVoice        PacketType = 0x00
	PTVoiceWhisper PacketType = 0x01
	PTCommand      PacketType = 0x02
	PTCommandLow   PacketType = 0x03
	PTHeartbeat    PacketType = 0x04
	PTAckHeartbeat PacketType = 0x05
	PTAck          PacketType = 0x06
	PTAckLow       PacketType = 0x07
	PTInit1        PacketType = 0x08
)

type Flags uint8

const (
	FlagUE Flags = 0x80 // Unencrypted (set => NOT encrypted)
	FlagCP Flags = 0x40 // Compressed
	FlagNP Flags = 0x20 // Newprotocol
	FlagFR Flags = 0x10 // Fragmented
)

type C2SPacket struct {
	MAC   [8]byte
	PId   uint16
	CId   uint16
	Type  PacketType
	Flags Flags
	Data  []byte
}

type S2CPacket struct {
	MAC   [8]byte
	PId   uint16
	Type  PacketType
	Flags Flags
	Data  []byte
}

func (p *C2SPacket) Encode() []byte {
	out := make([]byte, 13+len(p.Data))

	copy(out[0:8], p.MAC[:])
	binary.BigEndian.PutUint16(out[8:10], p.PId)
	binary.BigEndian.PutUint16(out[10:12], p.CId)
	out[12] = EncodePT(p.Flags, p.Type)
	copy(out[13:], p.Data)

	return out
}

func EncodePT(pFlags Flags, pType PacketType) uint8 {
	return (uint8(pFlags) & 0xF0) | (uint8(pType) & 0x0F)
}

func DecodePT(b uint8) (PacketType, Flags) {
	return PacketType(b & 0x0F), Flags(b & 0xF0)
}

func (p *C2SPacket) EncodeMeta() []byte {
	buf := make([]byte, 5)
	pt := EncodePT(p.Flags, p.Type)
	binary.BigEndian.PutUint16(buf[0:2], p.PId)
	binary.BigEndian.PutUint16(buf[2:4], p.CId)
	buf[4] = pt
	return buf[:5]
}

func (p *S2CPacket) encodeMeta() []byte {
	buf := make([]byte, 3)
	pt := EncodePT(p.Flags, p.Type)
	binary.BigEndian.PutUint16(buf[0:2], p.PId)
	buf[2] = pt
	return buf[:3]
}

func (p *S2CPacket) EncodeMeta() []byte {
	return p.encodeMeta()
}

func DecodeS2CPacket(dat []byte) (*S2CPacket, error) {
	if len(dat) < 11 {
		return nil, fmt.Errorf("packet too short: %d", len(dat))
	}
	var pkt S2CPacket
	copy(pkt.MAC[:], dat[0:8])
	pkt.PId = binary.BigEndian.Uint16(dat[8:10])
	pkt.Type, pkt.Flags = DecodePT(dat[10])
	pkt.Data = append([]byte(nil), dat[11:]...)
	return &pkt, nil
}

func (t *PacketType) String() string {
	switch *t {
	case PTVoice:
		return "voice"
	case PTVoiceWhisper:
		return "voice_whisper"
	case PTCommand:
		return "command"
	case PTCommandLow:
		return "command_low"
	case PTHeartbeat:
		return "ping"
	case PTAckHeartbeat:
		return "pong"
	case PTAck:
		return "ack"
	case PTAckLow:
		return "ack_low"
	case PTInit1:
		return "init1"
	default:
		return fmt.Sprintf("unknown(0x%02x)", uint8(*t))
	}
}

func FlagsString(f Flags) string {
	if f == 0 {
		return "none"
	}
	var parts []string
	if f&FlagUE != 0 {
		parts = append(parts, "UE")
	}
	if f&FlagCP != 0 {
		parts = append(parts, "CP")
	}
	if f&FlagNP != 0 {
		parts = append(parts, "NP")
	}
	if f&FlagFR != 0 {
		parts = append(parts, "FR")
	}
	return strings.Join(parts, "|")
}
