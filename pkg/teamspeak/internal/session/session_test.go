package session

import (
	"Jabba_The_Bot/pkg/teamspeak/internal/protocol"
	"testing"
)

func TestNextPIDAndGenID_OverflowIncrementsGenerationOnOverflowPacket(t *testing.T) {
	s := NewSession(nil, 0)

	var pid uint16
	var gen uint32
	for i := 0; i < 0xFFFF; i++ {
		pid, gen = s.NextPIDAndGenID(protocol.PTVoice)
	}

	if pid != 0xFFFF {
		t.Fatalf("overflow packet pid = %d, want %d", pid, 0xFFFF)
	}
	if gen != 1 {
		t.Fatalf("overflow packet gen = %d, want %d", gen, 1)
	}

	pid, gen = s.NextPIDAndGenID(protocol.PTVoice)
	if pid != 1 {
		t.Fatalf("post-overflow pid = %d, want %d", pid, 1)
	}
	if gen != 1 {
		t.Fatalf("post-overflow gen = %d, want %d", gen, 1)
	}
}

func TestNextPIDAndGenID_IsPerPacketType(t *testing.T) {
	s := NewSession(nil, 0)

	voicePID, voiceGen := s.NextPIDAndGenID(protocol.PTVoice)
	cmdPID, cmdGen := s.NextPIDAndGenID(protocol.PTCommand)

	if voicePID != 1 || cmdPID != 1 {
		t.Fatalf("first pid mismatch: voice=%d command=%d", voicePID, cmdPID)
	}
	if voiceGen != 0 || cmdGen != 0 {
		t.Fatalf("first generation mismatch: voice=%d command=%d", voiceGen, cmdGen)
	}
}
