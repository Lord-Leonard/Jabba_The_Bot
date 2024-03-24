package protocol

import "encoding/binary"

type ByteWriter struct {
	B   []byte
	off int
}

func (w *ByteWriter) U8(v byte) {
	w.B[w.off] = v
	w.off++
}

func (w *ByteWriter) U32be(v uint32) {
	binary.BigEndian.PutUint32(w.B[w.off:w.off+4], v)
	w.off += 4
}

func (w *ByteWriter) Bytes(src []byte) {
	copy(w.B[w.off:w.off+len(src)], src)
	w.off += len(src)
}

type ByteReader struct {
	B   []byte
	off int
}

func (r *ByteReader) U8() (byte, bool) {
	if r.off+1 > len(r.B) {
		return 0, false
	}
	v := r.B[r.off]
	r.off++
	return v, true
}

func (r *ByteReader) U32be() (uint32, bool) {
	if r.off+4 > len(r.B) {
		return 0, false
	}
	v := binary.BigEndian.Uint32(r.B[r.off : r.off+4])
	r.off += 4
	return v, true
}

func (r *ByteReader) Bytes(dst []byte) bool {
	if r.off+len(dst) > len(r.B) {
		return false
	}
	copy(dst, r.B[r.off:r.off+len(dst)])
	r.off += len(dst)
	return true
}
