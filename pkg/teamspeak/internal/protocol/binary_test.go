package protocol

import (
	"bytes"
	"testing"
)

func TestByteWriterAndReader_RoundTrip(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 0, 16)
	buf = append(buf, make([]byte, 9)...)

	w := ByteWriter{B: buf}
	w.U32be(0xAABBCCDD)
	w.U8(0x11)
	w.Bytes([]byte{0x22, 0x33, 0x44, 0x55})

	r := ByteReader{B: buf}
	got32, ok := r.U32be()
	if !ok {
		t.Fatal("u32be read failed")
	}
	if got32 != 0xAABBCCDD {
		t.Fatalf("u32be = %#x, want %#x", got32, 0xAABBCCDD)
	}

	got8, ok := r.U8()
	if !ok {
		t.Fatal("u8 read failed")
	}
	if got8 != 0x11 {
		t.Fatalf("u8 = %#x, want %#x", got8, 0x11)
	}

	var tail [4]byte
	if !r.Bytes(tail[:]) {
		t.Fatal("bytes read failed")
	}
	if !bytes.Equal(tail[:], []byte{0x22, 0x33, 0x44, 0x55}) {
		t.Fatalf("bytes = %#v, want %#v", tail[:], []byte{0x22, 0x33, 0x44, 0x55})
	}
}

func TestByteReader_Bounds(t *testing.T) {
	t.Parallel()

	r := ByteReader{B: []byte{0x01}}
	if _, ok := r.U32be(); ok {
		t.Fatal("u32be should fail on short buffer")
	}
	if _, ok := r.U8(); !ok {
		t.Fatal("u8 should succeed")
	}
	if _, ok := r.U8(); ok {
		t.Fatal("u8 should fail when out of data")
	}
}
