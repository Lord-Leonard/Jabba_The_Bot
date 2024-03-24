package session

import (
	"Jabba_The_Bot/pkg/teamspeak/internal/handshake"
	"Jabba_The_Bot/pkg/teamspeak/internal/protocol"
	"sync"
)

type Session struct {
	mu sync.Mutex

	SharedIV []byte
	ClientID uint32

	HandshakeState *handshake.State

	nextPacketIDByType map[protocol.PacketType]uint16
	sendGenIDByType    map[protocol.PacketType]uint32
}

func NewSession(sharedIV []byte, clientID uint32) *Session {
	return &Session{
		SharedIV:           sharedIV,
		ClientID:           clientID,
		HandshakeState:     &handshake.State{},
		nextPacketIDByType: make(map[protocol.PacketType]uint16),
		sendGenIDByType:    make(map[protocol.PacketType]uint32),
	}
}

func (s *Session) NextPIDAndGenID(pt protocol.PacketType) (uint16, uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pid := s.nextPacketIDByType[pt]
	if pid == 0 {
		pid = 1
	}
	genID := s.sendGenIDByType[pt]

	if pid == 0xFFFF {
		// TS3 expects the generation to increment on the overflowing packet itself.
		genID++
		s.sendGenIDByType[pt] = genID
		s.nextPacketIDByType[pt] = 1
		return pid, genID
	}

	s.nextPacketIDByType[pt] = pid + 1
	return pid, genID
}
