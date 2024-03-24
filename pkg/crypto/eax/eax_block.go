package eax

import (
	"crypto/cipher"
	"errors"
)

// EncryptWithBlock encrypts plaintext using AES-EAX on the provided block and returns (ciphertext, tag).
// This is useful when callers already manage the cipher.Block instance.
func EncryptWithBlock(block cipher.Block, nonce, header, plaintext []byte) ([]byte, []byte, error) {
	nonceTag := omac(block, 0x00, nonce)
	headerTag := omac(block, 0x01, header)

	ciphertext := make([]byte, len(plaintext))
	ctr := cipher.NewCTR(block, nonceTag)
	ctr.XORKeyStream(ciphertext, plaintext)

	msgTag := omac(block, 0x02, ciphertext)
	tag := make([]byte, 16)
	for i := 0; i < 16; i++ {
		tag[i] = nonceTag[i] ^ headerTag[i] ^ msgTag[i]
	}
	return ciphertext, tag, nil
}

// DecryptWithBlock decrypts ciphertext using AES-EAX on the provided block and verifies the tag (MAC).
func DecryptWithBlock(block cipher.Block, nonce, header, ciphertext, tag []byte) ([]byte, error) {
	if len(tag) != 16 {
		return nil, errors.New("EAX tag must be 16 bytes")
	}
	return decryptWithBlock(block, nonce, header, ciphertext, tag)
}

// DecryptTruncatedWithBlock decrypts ciphertext and verifies a truncated EAX tag.
// The tag must be between 8 and 16 bytes; verification compares only the provided prefix.
func DecryptTruncatedWithBlock(block cipher.Block, nonce, header, ciphertext, tag []byte) ([]byte, error) {
	if len(tag) < 8 || len(tag) > 16 {
		return nil, errors.New("EAX tag length must be between 8 and 16 bytes")
	}
	return decryptWithBlock(block, nonce, header, ciphertext, tag)
}

func decryptWithBlock(block cipher.Block, nonce, header, ciphertext, tag []byte) ([]byte, error) {
	nonceTag := omac(block, 0x00, nonce)
	headerTag := omac(block, 0x01, header)
	msgTag := omac(block, 0x02, ciphertext)
	expected := make([]byte, 16)
	for i := 0; i < 16; i++ {
		expected[i] = nonceTag[i] ^ headerTag[i] ^ msgTag[i]
	}
	for i := 0; i < len(tag); i++ {
		if expected[i] != tag[i] {
			return nil, errors.New("EAX tag mismatch")
		}
	}

	plaintext := make([]byte, len(ciphertext))
	ctr := cipher.NewCTR(block, nonceTag)
	ctr.XORKeyStream(plaintext, ciphertext)
	return plaintext, nil
}
