package packet

import "Jabba_The_Bot/pkg/teamspeak/internal/protocol"

type AckPayload struct {
	PacketId uint16
}

func NewAckPayload(packetId uint16) AckPayload {
	return AckPayload{PacketId: packetId}
}

func (p *AckPayload) Encode() []byte {
	return []byte{byte(p.PacketId >> 8), byte(p.PacketId)}
}

func NewAck(payload AckPayload) *protocol.C2SPacket {
	return &protocol.C2SPacket{
		Type:  protocol.PTAck,
		Flags: protocol.FlagNP,
		Data:  payload.Encode(),
	}
}

func NewAckLow(payload AckPayload) *protocol.C2SPacket {
	return &protocol.C2SPacket{
		Type:  protocol.PTAckLow,
		Flags: protocol.FlagNP,
		Data:  payload.Encode(),
	}
}

func NewAckHeartbeat(payload AckPayload) *protocol.C2SPacket {
	return &protocol.C2SPacket{
		Type:  protocol.PTAckHeartbeat,
		Flags: protocol.FlagUE,
		Data:  payload.Encode(),
	}
}
