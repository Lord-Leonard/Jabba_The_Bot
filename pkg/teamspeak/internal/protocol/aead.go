package protocol

import (
	"Jabba_The_Bot/pkg/crypto/eax"
	"crypto/sha1"
	"crypto/sha256"
	"sync"
)

var InitMac = [8]byte{0x54, 0x53, 0x33, 0x49, 0x4E, 0x49, 0x54, 0x31}

var (
	bootstrapKey   = []byte{0x63, 0x3A, 0x5C, 0x77, 0x69, 0x6E, 0x64, 0x6F, 0x77, 0x73, 0x5C, 0x73, 0x79, 0x73, 0x74, 0x65}
	bootstrapNonce = []byte{0x6D, 0x5C, 0x66, 0x69, 0x72, 0x65, 0x77, 0x61, 0x6C, 0x6C, 0x33, 0x32, 0x2E, 0x63, 0x70, 0x6C}
)

func BootstrapKey() []byte   { return append([]byte(nil), bootstrapKey...) }
func BootstrapNonce() []byte { return append([]byte(nil), bootstrapNonce...) }

type AEAD struct {
	mu sync.RWMutex

	sharedIV []byte
	genID    uint32
}

func NewAEAD() *AEAD {
	return &AEAD{}
}

func (a *AEAD) SetSharedIV(iv []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sharedIV = append([]byte(nil), iv...)
}

func (a *AEAD) Seal(p *C2SPacket) error {
	a.mu.RLock()
	genID := a.genID
	sharedIV := append([]byte(nil), a.sharedIV...)
	a.mu.RUnlock()

	key, nonce := DeriveKeyNonce(p.Type, p.PId, genID, false, sharedIV)
	return a.SealWithKeyNonce(p, key[:], nonce[:])
}

func (a *AEAD) SealWithKeyNonce(p *C2SPacket, key, nonce []byte) error {
	meta := p.EncodeMeta()

	ct, tag, err := eax.Encrypt(key, nonce, meta[:], p.Data)
	if err != nil {
		return err
	}

	var mac [8]byte
	copy(mac[:], tag[:8])
	p.MAC = mac
	p.Data = ct

	return nil
}

func (a *AEAD) SealBootstrap(p *C2SPacket) error {
	return a.SealWithKeyNonce(p, BootstrapKey(), BootstrapNonce())
}

func OpenBootstrap(p *S2CPacket) error {
	meta := p.encodeMeta()
	data, err := eax.DecryptTruncated(BootstrapKey(), BootstrapNonce(), meta, p.Data, p.MAC[:])
	if err != nil {
		return err
	}
	p.Data = data
	return nil
}

func EncryptCommandPayload(p *C2SPacket, key, nonce []byte, plaintext []byte) ([]byte, []byte, error) {
	meta := p.EncodeMeta()
	return eax.Encrypt(key, nonce, meta, plaintext)
}

// DeriveKeyNonce derives the per-packet AES key/nonce from SharedIV (SIV),
// packet type, direction, generation id, and packet id.
func DeriveKeyNonce(pt PacketType, pid uint16, genID uint32, fromServer bool, sharedIV []byte) (key [16]byte, nonce [16]byte) {
	if len(sharedIV) == 0 {
		return key, nonce
	}
	tmpLen := 6 + len(sharedIV)
	tmp := make([]byte, tmpLen)
	if fromServer {
		tmp[0] = 0x30
	} else {
		tmp[0] = 0x31
	}
	tmp[1] = byte(pt)
	tmp[2] = byte(genID >> 24)
	tmp[3] = byte(genID >> 16)
	tmp[4] = byte(genID >> 8)
	tmp[5] = byte(genID)
	copy(tmp[6:], sharedIV)
	keynonce := sha256.Sum256(tmp)
	copy(key[:], keynonce[0:16])
	copy(nonce[:], keynonce[16:32])
	key[0] ^= byte(pid >> 8)
	key[1] ^= byte(pid)
	return key, nonce
}

// SharedMacFromIV returns sha1(sharedIV)[:8].
func SharedMacFromIV(sharedIV []byte) [8]byte {
	sum := sha1.Sum(sharedIV)
	return [8]byte(sum[:8])
}

func (a *AEAD) Open(p *S2CPacket) error {
	a.mu.RLock()
	genID := a.genID
	sharedIV := append([]byte(nil), a.sharedIV...)
	a.mu.RUnlock()

	key, nonce := DeriveKeyNonce(p.Type, p.PId, genID, true, sharedIV)
	meta := p.EncodeMeta()

	pt, err := eax.DecryptTruncated(key[:], nonce[:], meta[:], p.Data, p.MAC[:])
	if err != nil {
		return err
	}
	p.Data = pt
	return nil
}
