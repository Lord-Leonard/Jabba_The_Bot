package packet

import (
	"Jabba_The_Bot/pkg/teamspeak/internal/protocol"
	"time"
)

type Step byte

const (
	Step0 Step = iota
	Step1
	Step2
	Step3
	Step4
)

const (
	packet0PayloadSize = 4 + 1 + 4 + 4 + 8
	packet1PayloadSize = 1 + 16 + 4
	packet2PayloadSize = 4 + 1 + 16 + 4
	packet3PayloadSize = 1 + 64 + 64 + 4 + 100
	packet4PayloadSize = 4 + 1 + 64 + 64 + 4 + 100 + 64
)

type Packet0Payload struct {
	Version uint32
	Step    Step
	UnixTS  uint32
	A0      [4]byte
}

func NewPacket0Payload(now time.Time, A0 [4]byte) Packet0Payload {
	return Packet0Payload{
		Version: clientVersionField(now),
		Step:    Step0,
		UnixTS:  uint32(now.Unix()),
		A0:      A0,
	}
}

func (p Packet0Payload) Encode() []byte {
	out := make([]byte, packet0PayloadSize)
	w := protocol.ByteWriter{B: out}
	w.U32be(p.Version)
	w.U8(byte(p.Step))
	w.U32be(p.UnixTS)
	w.Bytes(p.A0[:])
	return out
}

func NewPacket0(payload Packet0Payload) *protocol.C2SPacket {
	return &protocol.C2SPacket{
		PId:   101,
		CId:   0,
		Type:  protocol.PTInit1,
		Flags: protocol.FlagUE | protocol.FlagNP,
		Data:  payload.Encode(),
	}
}

type Packet1Payload struct {
	Step Step
	A1   [16]byte
	A0r  [4]byte
}

func DecodePacket1Payload(data []byte) (bool, *Packet1Payload) {
	if len(data) < packet1PayloadSize {
		return false, nil
	}
	p := &Packet1Payload{}
	r := protocol.ByteReader{B: data}

	step, ok := r.U8()
	if !ok || step != byte(Step1) {
		return false, nil
	}
	p.Step = Step(step)

	if !r.Bytes(p.A1[:]) {
		return false, nil
	}
	if !r.Bytes(p.A0r[:]) {
		return false, nil
	}

	return true, p
}

type Packet2Payload struct {
	Version uint32
	Step    Step
	A1      [16]byte
	A0r     [4]byte
}

func NewPacket2Payload(now time.Time, A1 [16]byte, A0r [4]byte) Packet2Payload {
	return Packet2Payload{
		Version: clientVersionField(now),
		Step:    Step2,
		A1:      A1,
		A0r:     A0r,
	}
}

func (p Packet2Payload) Encode() []byte {
	out := make([]byte, packet2PayloadSize)
	w := protocol.ByteWriter{B: out}
	w.U32be(p.Version)
	w.U8(byte(p.Step))
	w.Bytes(p.A1[:])
	w.Bytes(p.A0r[:])
	return out
}

func NewPacket2(payload Packet2Payload) *protocol.C2SPacket {
	return &protocol.C2SPacket{
		PId:   101,
		CId:   0,
		Type:  protocol.PTInit1,
		Flags: protocol.FlagUE | protocol.FlagNP,
		Data:  payload.Encode(),
	}
}

type Packet3Payload struct {
	Step  byte
	X     [64]byte
	N     [64]byte
	Level uint32
	A2    [100]byte
}

func DecodePacket3Payload(data []byte) (bool, *Packet3Payload) {
	if len(data) < packet3PayloadSize {
		return false, nil
	}
	p := &Packet3Payload{}
	r := protocol.ByteReader{B: data}

	var ok bool
	if p.Step, ok = r.U8(); !ok || Step(p.Step) != Step3 {
		return false, nil
	}
	if !r.Bytes(p.X[:]) {
		return false, nil
	}
	if !r.Bytes(p.N[:]) {
		return false, nil
	}
	if p.Level, ok = r.U32be(); !ok {
		return false, nil
	}
	if !r.Bytes(p.A2[:]) {
		return false, nil
	}

	return true, p
}

type Packet4Payload struct {
	Version      uint32
	Step         Step
	X            [64]byte
	N            [64]byte
	Level        uint32
	A2           [100]byte
	Y            [64]byte
	ClientInitIV []byte
}

func NewPacket4Payload(now time.Time, X, N, Y [64]byte, A2 [100]byte, Level uint32, clientInitIV []byte) Packet4Payload {
	return Packet4Payload{
		Version:      clientVersionField(now),
		Step:         Step4,
		X:            X,
		N:            N,
		Level:        Level,
		A2:           A2,
		Y:            Y,
		ClientInitIV: clientInitIV,
	}
}

func (p Packet4Payload) Encode() []byte {
	out := make([]byte, packet4PayloadSize+len(p.ClientInitIV))
	w := protocol.ByteWriter{B: out}
	w.U32be(p.Version)
	w.U8(byte(p.Step))
	w.Bytes(p.X[:])
	w.Bytes(p.N[:])
	w.U32be(p.Level)
	w.Bytes(p.A2[:])
	w.Bytes(p.Y[:])
	w.Bytes(p.ClientInitIV)
	return out
}

func NewPacket4(payload Packet4Payload) *protocol.C2SPacket {
	return &protocol.C2SPacket{
		PId:   101,
		CId:   0,
		Type:  protocol.PTInit1,
		Flags: protocol.FlagUE | protocol.FlagNP,
		Data:  payload.Encode(),
	}
}

// --- utils ---

func clientVersionField(now time.Time) uint32 {
	const base = 1356998400
	u := uint32(now.Unix())
	if u < base {
		return 0
	}
	return u - base
}
