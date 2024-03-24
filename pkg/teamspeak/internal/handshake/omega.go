package handshake

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
)

type omegaASN struct {
	Bits    asn1.BitString
	KeySize int
	X       *big.Int
	Y       *big.Int
}

func ParseOmegaPublicKey(omegaB64 string) (*ecdsa.PublicKey, error) {
	der, err := base64.StdEncoding.DecodeString(omegaB64)
	if err != nil {
		return nil, fmt.Errorf("omega decode: %w", err)
	}
	var o omegaASN
	if _, err := asn1.Unmarshal(der, &o); err != nil {
		return nil, fmt.Errorf("omega asn1: %w", err)
	}
	if o.X == nil || o.Y == nil {
		return nil, errors.New("omega missing coordinates")
	}
	curve := elliptic.P256()
	return &ecdsa.PublicKey{Curve: curve, X: o.X, Y: o.Y}, nil
}
