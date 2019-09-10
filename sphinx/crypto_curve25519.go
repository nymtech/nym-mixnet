package sphinx

import (
	"crypto/rand"
	"crypto/subtle"
	"io"

	"golang.org/x/crypto/curve25519"
)

const (
	FieldElementSize = 32
	PrivateKeySize   = FieldElementSize
	PublicKeySize    = FieldElementSize
)

// TODO: better name
type CryptoElement interface {
	Bytes() []byte
}

type FieldElement struct {
	bytes [FieldElementSize]byte
}

// TODO: redefine private and public keys to be interfaces instead?
type PrivateKey struct {
	bytes [PrivateKeySize]byte
}

type PublicKey struct {
	bytes [PublicKeySize]byte
}

func BytesToFieldElement(b []byte) *FieldElement {
	if len(b) > FieldElementSize {
		panic("The byte slice is larger than the field element")
	}
	fe := new(FieldElement)
	copy(fe.bytes[:], b)
	return fe
}

func (fe *FieldElement) Bytes() []byte {
	return fe.bytes[:]
}

func (fe *FieldElement) el() *[FieldElementSize]byte {
	return &fe.bytes
}

func BytesToPrivateKey(b []byte) *PrivateKey {
	if len(b) > PrivateKeySize {
		panic("The byte slice is larger than the field element")
	}
	pk := new(PrivateKey)
	copy(pk.bytes[:], b)
	return pk
}

func (pk *PrivateKey) Bytes() []byte {
	return pk.bytes[:]
}

func (pk *PrivateKey) ToFieldElement() *FieldElement {
	return BytesToFieldElement(pk.Bytes())
}

func BytesToPublicKey(b []byte) *PublicKey {
	if len(b) > PublicKeySize {
		panic("The byte slice is larger than the field element")
	}
	pub := new(PublicKey)
	copy(pub.bytes[:], b)
	return pub
}

func (pub *PublicKey) Bytes() []byte {
	return pub.bytes[:]
}

func (pub *PublicKey) ToFieldElement() *FieldElement {
	return BytesToFieldElement(pub.Bytes())
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

func CompareElements(e1, e2 CryptoElement) bool {
	return subtle.ConstantTimeCompare(e1.Bytes(), e2.Bytes()) == 1
}

func RandomElement() (*FieldElement, error) {
	b := [32]byte{}
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return nil, err
	}
	return &FieldElement{
		bytes: b,
	}, nil
}

func expo(base *FieldElement, exp []*FieldElement) *FieldElement {
	x := exp[0]
	res := new(FieldElement)
	curve25519.ScalarMult(res.el(), x.el(), base.el())

	for _, val := range exp[1:] {
		curve25519.ScalarMult(res.el(), val.el(), res.el())
	}

	return res
}

func expoGroupBase(exp []*FieldElement) *FieldElement {
	x := exp[0]
	res := new(FieldElement)
	curve25519.ScalarBaseMult(res.el(), x.el())

	for _, val := range exp[1:] {
		curve25519.ScalarMult(res.el(), val.el(), res.el())
	}

	return res
}

//////////////////////////////////////////////
// REFERENCE
//////////////////////////////////////////////
// If we really had to use field arithmetic, we could copy code from
// golang.org/x/crypto/ed25519/internal/edwards25519
// We can't directly import it because go forbids imports from internal packages...
// If we decided for this, the following function would be needed to convert from twisted edwards to montgomery points:
// // Source: https://github.com/agl/ed25519/blob/master/extra25519/extra25519.go
// // BSD license
// func edwardsToMontgomeryX(outX, y *edwards25519.FieldElement) {
// 	// We only need the x-coordinate of the curve25519 point, which I'll
// 	// call u. The isomorphism is u=(y+1)/(1-y), since y=Y/Z, this gives
// 	// u=(Y+Z)/(Z-Y). We know that Z=1, thus u=(Y+1)/(1-Y).
// 	var oneMinusY edwards25519.FieldElement
// 	edwards25519.FeOne(&oneMinusY)
// 	edwards25519.FeSub(&oneMinusY, &oneMinusY, y)
// 	edwards25519.FeInvert(&oneMinusY, &oneMinusY)

// 	edwards25519.FeOne(outX)
// 	edwards25519.FeAdd(outX, outX, y)

// 	edwards25519.FeMul(outX, outX, &oneMinusY)
// }
