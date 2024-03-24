package handshake

import (
	"crypto/sha512"
	"errors"
	"fmt"

	"filippo.io/edwards25519"
)

var curve25519Root = []byte{
	0xcd, 0x0d, 0xe2, 0xae, 0xd4, 0x63, 0x45, 0x50, 0x9a, 0x7e, 0x3c,
	0xfd, 0x8f, 0x68, 0xb3, 0xdc, 0x75, 0x55, 0xb2, 0x9d, 0xcc, 0xec,
	0x73, 0xcd, 0x18, 0x75, 0x0f, 0x99, 0x38, 0x12, 0x40, 0x8a,
}

func computeLicenseChain(lBytes []byte) ([]byte, error) {
	blocks, err := parseLicenseBlocks(lBytes)
	if err != nil {
		return nil, err
	}
	parent := append([]byte(nil), curve25519Root...)
	for _, block := range blocks {
		if len(block) < 1+32 {
			return nil, errors.New("license block too short")
		}
		pubBytes := block[1 : 1+32]
		next, err := deriveLicenseKey(pubBytes, parent, block[1:])
		if err != nil {
			return nil, err
		}
		parent = next
	}
	return parent, nil
}

func parseLicenseBlocks(l []byte) ([][]byte, error) {
	if len(l) < 1 {
		return nil, errors.New("license too short")
	}
	off := 1 // skip version
	var blocks [][]byte
	for off < len(l) {
		start := off
		if off+1+32+1+4+4 > len(l) {
			return nil, errors.New("license block header truncated")
		}
		off++     // key type
		off += 32 // public key
		bt := l[off]
		off++    // block type
		off += 4 // not valid before
		off += 4 // not valid after

		switch bt {
		case 0x00:
			if off+4 > len(l) {
				return nil, errors.New("license block 0x00 truncated")
			}
			off += 4
			var err error
			off, err = readCString(l, off)
			if err != nil {
				return nil, err
			}
		case 0x01, 0x03:
			var err error
			off, err = readCString(l, off)
			if err != nil {
				return nil, err
			}
		case 0x02:
			if off+1+4 > len(l) {
				return nil, errors.New("license block 0x02 truncated")
			}
			off += 1 + 4
			var err error
			off, err = readCString(l, off)
			if err != nil {
				return nil, err
			}
		case 0x08:
			if off+2 > len(l) {
				return nil, errors.New("license block 0x08 truncated")
			}
			off++ // license type
			propCount := int(l[off])
			off++
			for i := 0; i < propCount; i++ {
				if off+1 > len(l) {
					return nil, errors.New("license block property truncated")
				}
				plen := int(l[off])
				off++
				if off+plen > len(l) {
					return nil, errors.New("license block property data truncated")
				}
				off += plen
			}
		case 0x20:
			// Ephemeral, no content
		default:
			return nil, fmt.Errorf("unknown license block type 0x%02x", bt)
		}

		blocks = append(blocks, l[start:off])
	}
	return blocks, nil
}

func readCString(b []byte, off int) (int, error) {
	for i := off; i < len(b); i++ {
		if b[i] == 0x00 {
			return i + 1, nil
		}
	}
	return 0, errors.New("unterminated string in license block")
}

func deriveLicenseKey(pubBytes, parentBytes, hashable []byte) ([]byte, error) {
	pubPoint := new(edwards25519.Point)
	if _, err := pubPoint.SetBytes(pubBytes); err != nil {
		return nil, fmt.Errorf("license block public key decode: %w", err)
	}
	pubPoint.Negate(pubPoint)

	parentPoint := new(edwards25519.Point)
	if _, err := parentPoint.SetBytes(parentBytes); err != nil {
		return nil, fmt.Errorf("license block parent key decode: %w", err)
	}
	parentPoint.Negate(parentPoint)

	hash := sha512.Sum512(hashable)
	var s edwards25519.Scalar
	if _, err := s.SetBytesWithClamping(hash[:32]); err != nil {
		return nil, err
	}
	tmp := new(edwards25519.Point).ScalarMult(&s, pubPoint)
	sum := new(edwards25519.Point).Add(tmp, parentPoint)
	out := sum.Bytes()
	out[31] ^= 0x80
	return out, nil
}
