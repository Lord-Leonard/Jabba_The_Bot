package cmac

import "crypto/cipher"

// Sum computes CMAC (AES-CMAC) for the provided message.
// It returns a 16-byte tag.
func Sum(block cipher.Block, msg []byte) []byte {
	const blockSize = 16
	k1, k2 := subkeys(block)

	var last []byte
	if len(msg) == 0 {
		last = make([]byte, blockSize)
		last[0] = 0x80
		xorInPlace(last, k2)
	} else if len(msg)%blockSize == 0 {
		last = append([]byte{}, msg[len(msg)-blockSize:]...)
		xorInPlace(last, k1)
	} else {
		n := len(msg) % blockSize
		last = make([]byte, blockSize)
		copy(last, msg[len(msg)-n:])
		last[n] = 0x80
		xorInPlace(last, k2)
	}

	nblocks := 1
	if len(msg) > 0 {
		nblocks = (len(msg) + blockSize - 1) / blockSize
	}
	var x [blockSize]byte
	var y [blockSize]byte
	for i := 0; i < nblocks-1; i++ {
		blockStart := i * blockSize
		copy(y[:], x[:])
		xorInPlaceBytes(y[:], msg[blockStart:blockStart+blockSize])
		block.Encrypt(x[:], y[:])
	}

	copy(y[:], x[:])
	xorInPlaceBytes(y[:], last)
	block.Encrypt(x[:], y[:])
	return x[:]
}

func subkeys(block cipher.Block) ([]byte, []byte) {
	const rb = 0x87
	zero := make([]byte, 16)
	l := make([]byte, 16)
	block.Encrypt(l, zero)
	k1 := leftShiftOne(l)
	if l[0]&0x80 != 0 {
		k1[15] ^= rb
	}
	k2 := leftShiftOne(k1)
	if k1[0]&0x80 != 0 {
		k2[15] ^= rb
	}
	return k1, k2
}

func leftShiftOne(in []byte) []byte {
	out := make([]byte, len(in))
	var carry byte
	for i := len(in) - 1; i >= 0; i-- {
		out[i] = in[i]<<1 | carry
		carry = (in[i] & 0x80) >> 7
	}
	return out
}

func xorInPlace(dst []byte, src []byte) {
	for i := range dst {
		dst[i] ^= src[i]
	}
}

func xorInPlaceBytes(dst []byte, src []byte) {
	for i := range src {
		dst[i] ^= src[i]
	}
}
