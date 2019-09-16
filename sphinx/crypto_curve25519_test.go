package sphinx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/curve25519"
)

func TestGenerateKey(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	assert.Nil(t, err)
	assert.NotNil(t, priv)
	assert.NotNil(t, pub)
	assert.Len(t, priv.Bytes(), PrivateKeySize)
	assert.Len(t, pub.Bytes(), PublicKeySize)
	assert.NotZero(t, priv.Bytes())
	assert.NotZero(t, pub.Bytes())

	var pkBytes [PrivateKeySize]byte
	var pubBytes [PublicKeySize]byte
	copy(pkBytes[:], priv.Bytes())

	curve25519.ScalarBaseMult(&pubBytes, &pkBytes)
	assert.True(t, CompareElements(pub, &PublicKey{bytes: pubBytes}))
}

// Just a sanity check for my personal use
func TestCommutativity(t *testing.T) {
	// (g^x1)^x2 == (g^x2)^x1
	x1 := [32]byte{42}
	x2 := [32]byte{90, 0, 1}

	res1 := [32]byte{}
	res2 := [32]byte{}

	// g^x1
	curve25519.ScalarBaseMult(&res1, &x1)
	// (g^x1)^x2
	curve25519.ScalarMult(&res1, &x2, &res1)

	// g^x2
	curve25519.ScalarBaseMult(&res2, &x2)
	// (g^x2)^x1
	curve25519.ScalarMult(&res2, &x1, &res2)

	assert.Equal(t, res1, res2)
}
