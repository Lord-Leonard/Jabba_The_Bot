package handshake

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"

	"filippo.io/edwards25519"
)

// SharedIVFromInitivexpand2 derives SharedIV/SharedMac and prepares clientek data.
// The same ephemeral scalar is used for both shared secret derivation and clientek,
// which is required for server verification.
func SharedIVFromInitivexpand2(alphaB64, betaB64, lB64 string, identityPriv *ecdsa.PrivateKey) (sharedIV []byte, sharedMac []byte, clientEK []byte, clientEKProof []byte, err error) {
	alpha, err := base64.StdEncoding.DecodeString(alphaB64)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("alpha decode: %w", err)
	}
	beta, err := base64.StdEncoding.DecodeString(betaB64)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("beta decode: %w", err)
	}
	lBytes, err := base64.StdEncoding.DecodeString(lB64)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("l decode: %w", err)
	}

	nextKeyBytes, err := computeLicenseChain(lBytes)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var ephScalar edwards25519.Scalar
	rnd := make([]byte, 32)
	if _, err := rand.Read(rnd); err != nil {
		return nil, nil, nil, nil, err
	}
	if _, err := ephScalar.SetBytesWithClamping(rnd); err != nil {
		return nil, nil, nil, nil, err
	}

	clientPub := new(edwards25519.Point).ScalarMult(&ephScalar, edwards25519.NewGeneratorPoint())
	nextKey := new(edwards25519.Point)
	if _, err := nextKey.SetBytes(nextKeyBytes); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("license chain key decode: %w", err)
	}
	negKey := new(edwards25519.Point).Negate(nextKey)
	sharedPoint := new(edwards25519.Point).ScalarMult(&ephScalar, negKey)
	sharedTmp := sharedPoint.Bytes()
	sharedTmp[31] ^= 0x80

	sharedIV = make([]byte, 64)
	sum := sha512.Sum512(sharedTmp)
	copy(sharedIV, sum[:])
	for i := 0; i < 10 && i < len(alpha); i++ {
		sharedIV[i] ^= alpha[i]
	}
	for i := 0; i < 54 && i < len(beta); i++ {
		sharedIV[10+i] ^= beta[i]
	}
	ivSum := sha1.Sum(sharedIV)
	sharedMac = ivSum[:8]

	clientEK = clientPub.Bytes()
	proofMsg := append([]byte{}, clientEK...)
	proofMsg = append(proofMsg, beta...)
	hash := sha256.Sum256(proofMsg)
	proof, err := ecdsa.SignASN1(rand.Reader, identityPriv, hash[:])
	if err != nil {
		return nil, nil, nil, nil, err
	}
	clientEKProof = proof
	return sharedIV, sharedMac, clientEK, clientEKProof, nil
}

func VerifyInitivexpand2Proof(omegaB64, lB64, proofB64 string) error {
	pub, err := ParseOmegaPublicKey(omegaB64)
	if err != nil {
		return err
	}
	lBytes, err := base64.StdEncoding.DecodeString(lB64)
	if err != nil {
		return fmt.Errorf("l decode: %w", err)
	}
	proof, err := base64.StdEncoding.DecodeString(proofB64)
	if err != nil {
		return fmt.Errorf("proof decode: %w", err)
	}
	hash := sha256.Sum256(lBytes)
	if !ecdsa.VerifyASN1(pub, hash[:], proof) {
		return errors.New("initivexpand2 proof verification failed")
	}
	return nil
}
