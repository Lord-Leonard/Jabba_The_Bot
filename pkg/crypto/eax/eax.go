package eax

import (
	"Jabba_The_Bot/pkg/crypto/cmac"
	"crypto/aes"
	"crypto/cipher"
)

// Encrypt encrypts plaintext using AES-EAX and returns (ciphertext, tag).
func Encrypt(key, nonce, header, plaintext []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	return EncryptWithBlock(block, nonce, header, plaintext)
}

// Decrypt decrypts ciphertext and verifies the tag (MAC).
func Decrypt(key, nonce, header, ciphertext, tag []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return DecryptWithBlock(block, nonce, header, ciphertext, tag)
}

// DecryptTruncated decrypts ciphertext and verifies a truncated EAX tag.
// The tag must be between 8 and 16 bytes; verification compares only the provided prefix.
func DecryptTruncated(key, nonce, header, ciphertext, tag []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return DecryptTruncatedWithBlock(block, nonce, header, ciphertext, tag)
}

// omac computes OMAC (CMAC with a domain-separating prefix) used by EAX.
func omac(block cipher.Block, tag byte, data []byte) []byte {
	prefix := make([]byte, 16)
	prefix[15] = tag
	msg := append(prefix, data...)
	return cmac.Sum(block, msg)
}
