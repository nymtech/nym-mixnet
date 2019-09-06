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
	assert.True(t, CompareKeys(pub, &PublicKey{bytes: pubBytes}))
}
