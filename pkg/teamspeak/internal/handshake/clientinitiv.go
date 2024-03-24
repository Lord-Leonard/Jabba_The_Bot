package handshake

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/asn1"
	"encoding/base64"
	"fmt"
)

func GenerateAlphaB64() (string, error) {
	alpha := make([]byte, 10)
	if _, err := rand.Read(alpha); err != nil {
		return "", fmt.Errorf("clientinitiv: read alpha: %w", err)
	}
	return base64.StdEncoding.EncodeToString(alpha), nil
}

func OmegaB64FromPublicKey(pub *ecdsa.PublicKey) (string, error) {
	omegaDER, err := asn1.Marshal(omegaASN{
		Bits:    asn1.BitString{Bytes: []byte{0x00}, BitLength: 1},
		KeySize: 32,
		X:       pub.X,
		Y:       pub.Y,
	})
	if err != nil {
		return "", fmt.Errorf("clientinitiv: marshal omega: %w", err)
	}
	return base64.StdEncoding.EncodeToString(omegaDER), nil
}

func OmegaB64FromPublicPEM(pubPEM []byte) (string, error) {
	pub, err := PublicKeyFromPEM(pubPEM)
	if err != nil {
		return "", err
	}
	return OmegaB64FromPublicKey(pub)
}

func OmegaB64FromPrivatePEM(privPEM []byte) (string, error) {
	priv, err := PrivateKeyFromPEM(privPEM)
	if err != nil {
		return "", err
	}
	return OmegaB64FromPublicKey(&priv.PublicKey)
}

func OmegaB64FromPublicPEMFile(path string) (string, error) {
	pub, err := PublicKeyFromPEMFile(path)
	if err != nil {
		return "", err
	}
	return OmegaB64FromPublicKey(pub)
}

func OmegaB64FromPrivatePEMFile(path string) (string, error) {
	priv, err := PrivateKeyFromPEMFile(path)
	if err != nil {
		return "", err
	}
	return OmegaB64FromPublicKey(&priv.PublicKey)
}
