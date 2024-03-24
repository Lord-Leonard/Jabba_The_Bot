package handshake

import (
	"crypto/sha1"
	"fmt"
	"strconv"
)

// HashcashLevel computes the hashcash level for a public key string and offset.
// The publicKey string is the base64 omega string as used in clientinitiv.
func HashcashLevel(publicKey string, offset uint64) int {
	data := []byte(publicKey + strconv.FormatUint(offset, 10))
	sum := sha1.Sum(data)
	level := 0
	for i := 0; i < len(sum); i++ {
		b := sum[i]
		for bit := 0; bit < 8; bit++ {
			if (b>>bit)&1 != 0 {
				return level
			}
			level++
		}
	}
	return level
}

// FindKeyOffset searches for the smallest offset that reaches targetLevel.
func FindKeyOffset(publicKey string, targetLevel int) (uint64, int, error) {
	if targetLevel < 0 {
		return 0, 0, fmt.Errorf("invalid target level %d", targetLevel)
	}
	for off := uint64(0); ; off++ {
		level := HashcashLevel(publicKey, off)
		if level >= targetLevel {
			return off, level, nil
		}
	}
}
