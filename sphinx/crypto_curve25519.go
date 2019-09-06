package sphinx

import (
	"crypto/rand"
	"crypto/subtle"
	"io"

	"golang.org/x/crypto/curve25519"
)

const (
	GroupElementSize = 32
	PrivateKeySize   = GroupElementSize
	PublicKeySize    = GroupElementSize
)

type Big struct {
	bytes [GroupElementSize]byte
}

type PrivateKey struct {
	bytes [PrivateKeySize]byte
}

type PublicKey struct {
	bytes [PublicKeySize]byte
}

func init() {
	// TODO: do we need to seed the crypto/rand?
}
func (pk *PrivateKey) Bytes() []byte {
	return pk.bytes[:]
}

func (pub *PublicKey) Bytes() []byte {
	return pub.bytes[:]
}

// GenerateKeyPair returns public and private keypair bytes for Curve25519 elliptic curve, or an error.
func GenerateKeyPair() (*PrivateKey, *PublicKey, error) {
	priv := new(PrivateKey)
	pub := new(PublicKey)
	if _, err := io.ReadFull(rand.Reader, priv.Bytes()); err != nil {
		return nil, nil, err
	}
	curve25519.ScalarBaseMult(&pub.bytes, &priv.bytes)
	return priv, pub, nil
}

func CompareKeys(p1, p2 *PublicKey) bool {
	return subtle.ConstantTimeCompare(p1.Bytes(), p2.Bytes()) == 1
}
