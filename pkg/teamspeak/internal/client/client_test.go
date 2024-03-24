package client

import (
	"Jabba_The_Bot/pkg/teamspeak/internal/protocol"
	"testing"
	"time"
)

func TestHandleCommandPacket_ReassemblesFragmentedCommand(t *testing.T) {
	c := &Client{}

	first := &protocol.S2CPacket{
		PId:   10,
		Type:  protocol.PTCommand,
		Flags: protocol.FlagFR,
		Data:  []byte("notifytextmessage"),
	}
	msg, err := c.handleCommandPacket(first)
	if err != nil {
		t.Fatalf("first fragment failed: %v", err)
	}
	if msg != nil {
		t.Fatalf("expected nil message while assembling, got %+v", msg)
	}

	last := &protocol.S2CPacket{
		PId:   11,
		Type:  protocol.PTCommand,
		Flags: protocol.FlagFR,
		Data:  []byte(" msg=hi"),
	}
	msg, err = c.handleCommandPacket(last)
	if err != nil {
		t.Fatalf("last fragment failed: %v", err)
	}
	if msg == nil || msg.Name != "notifytextmessage" {
		t.Fatalf("unexpected final message: %+v", msg)
	}
}

func TestHandleCommandPacket_RecoversAfterFragmentLoss(t *testing.T) {
	c := &Client{}

	first := &protocol.S2CPacket{
		PId:   100,
		Type:  protocol.PTCommand,
		Flags: protocol.FlagFR,
		Data:  []byte("brokenfragment"),
	}
	msg, err := c.handleCommandPacket(first)
	if err != nil {
		t.Fatalf("first fragment failed: %v", err)
	}
	if msg != nil {
		t.Fatalf("expected nil message while assembling, got %+v", msg)
	}

	standalone := &protocol.S2CPacket{
		PId:  250,
		Type: protocol.PTCommand,
		Data: []byte("notifytextmessage msg=hello"),
	}
	msg, err = c.handleCommandPacket(standalone)
	if err != nil {
		t.Fatalf("standalone parse failed: %v", err)
	}
	if msg != nil {
		t.Fatalf("expected ambiguous packet to be dropped after fragment loss, got %+v", msg)
	}
	if c.cmdFragActive {
		t.Fatalf("fragment state should have been reset")
	}

	nextStandalone := &protocol.S2CPacket{
		PId:  251,
		Type: protocol.PTCommand,
		Data: []byte("notifytextmessage msg=hello_again"),
	}
	msg, err = c.handleCommandPacket(nextStandalone)
	if err != nil {
		t.Fatalf("next standalone parse failed: %v", err)
	}
	if msg == nil || msg.Name != "notifytextmessage" {
		t.Fatalf("expected recovery on next command packet, got %+v", msg)
	}
}

func TestHandleCommandPacket_ResetsStalledFragmentStream(t *testing.T) {
	c := &Client{
		cmdFragActive:     true,
		cmdFragCompressed: false,
		cmdFragBuf:        []byte("stale"),
		cmdFragNextPID:    42,
		cmdFragStarted:    time.Now().Add(-(commandFragmentStallTimeout + 100*time.Millisecond)),
	}

	standalone := &protocol.S2CPacket{
		PId:  500,
		Type: protocol.PTCommand,
		Data: []byte("notifytextmessage msg=after_timeout"),
	}
	msg, err := c.handleCommandPacket(standalone)
	if err != nil {
		t.Fatalf("standalone parse failed: %v", err)
	}
	if msg == nil || msg.Name != "notifytextmessage" {
		t.Fatalf("expected recovery after timeout, got %+v", msg)
	}
	if c.cmdFragActive {
		t.Fatalf("fragment state should be reset after stall timeout")
	}
}

func TestHandleCommandPacket_RejectsBinaryName(t *testing.T) {
	c := &Client{}

	msg, err := c.handleCommandPacket(&protocol.S2CPacket{
		PId:  1,
		Type: protocol.PTCommand,
		Data: []byte{0x80, 0x01, 0x02, 0x03, 0x20, 0x41},
	})
	if err == nil {
		t.Fatalf("expected error for binary command payload, got nil")
	}
	if msg != nil {
		t.Fatalf("expected nil message for binary command payload, got %+v", msg)
	}
}
