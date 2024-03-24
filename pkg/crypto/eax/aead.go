package eax

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

// NewAEAD returns an EAX-based cipher.AEAD using AES and a fixed nonce length.
// The AEAD uses a full 16-byte tag.
func NewAEAD(key []byte, nonceLen int) (cipher.AEAD, error) {
	if nonceLen <= 0 {
		return nil, errors.New("eax: nonce length must be > 0")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &aead{block: block, nonceLen: nonceLen}, nil
}

type aead struct {
	block    cipher.Block
	nonceLen int
}

func (a *aead) NonceSize() int { return a.nonceLen }

func (a *aead) Overhead() int { return 16 }

func (a *aead) Seal(dst, nonce, plaintext, additionalData []byte) []byte {
	if len(nonce) != a.nonceLen {
		panic("eax: incorrect nonce length")
	}
	ciphertext, tag, err := EncryptWithBlock(a.block, nonce, additionalData, plaintext)
	if err != nil {
		panic("eax: encrypt failed")
	}
	out := append(dst, ciphertext...)
	out = append(out, tag...)
	return out
}

func (a *aead) Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	if len(nonce) != a.nonceLen {
		return nil, errors.New("eax: incorrect nonce length")
	}
	if len(ciphertext) < a.Overhead() {
		return nil, errors.New("eax: message too short")
	}
	msg := ciphertext[:len(ciphertext)-a.Overhead()]
	tag := ciphertext[len(ciphertext)-a.Overhead():]
	plaintext, err := DecryptWithBlock(a.block, nonce, additionalData, msg, tag)
	if err != nil {
		return nil, err
	}
	return append(dst, plaintext...), nil
}
