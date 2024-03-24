package packet

import "Jabba_The_Bot/pkg/teamspeak/internal/protocol"

func NewCommandPacket(message any) (*protocol.C2SPacket, error) {
	data, err := protocol.EncodeMessage(message)
	if err != nil {
		return nil, err
	}

	return &protocol.C2SPacket{
		Type:  protocol.PTCommand,
		Flags: protocol.FlagNP,
		Data:  data,
	}, nil
}
