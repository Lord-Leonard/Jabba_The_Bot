package handshake

import "math/big"

func SolveRSAPuzzleY(xBytes, nBytes []byte, level uint32) []byte {
	x := new(big.Int).SetBytes(xBytes)
	n := new(big.Int).SetBytes(nBytes)
	if n.Sign() == 0 {
		return make([]byte, 64)
	}

	// y = x^(2^level) mod n == (((x^2 mod n)^2 mod n) ... ) level times
	y := new(big.Int).Mod(x, n)
	for i := uint32(0); i < level; i++ {
		y.Mul(y, y)
		y.Mod(y, n)
	}
	out := y.Bytes()
	if len(out) >= 64 {
		return out[len(out)-64:]
	}
	padded := make([]byte, 64)
	copy(padded[64-len(out):], out)
	return padded
}
