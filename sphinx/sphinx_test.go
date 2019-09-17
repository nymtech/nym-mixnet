// Copyright 2018 The Loopix-Messaging Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sphinx

import (
	"crypto/aes"
	"fmt"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/curve25519"
)

func TestMain(m *testing.M) {

	os.Exit(m.Run())
}

func TestExpoSingleValue(t *testing.T) {
	randomPoint1, err := RandomElement()
	assert.Nil(t, err)

	randomPoint2, err := RandomElement()
	assert.Nil(t, err)

	exp := []*FieldElement{randomPoint2}
	res := expo(randomPoint1, exp)
	expectedRes := new(FieldElement)
	curve25519.ScalarMult(&expectedRes.bytes, &randomPoint2.bytes, &randomPoint1.bytes)

	assert.True(t, CompareElements(res, expectedRes))
}

func TestExpoMultipleValue(t *testing.T) {
	// TODO: figure out a way to test it without having the FeMul function
}

func TestExpoBaseSingleValue(t *testing.T) {
	randomPoint, err := RandomElement()
	assert.Nil(t, err)

	exp := []*FieldElement{randomPoint}

	result := expoGroupBase(exp)
	expectedRes := new(FieldElement)
	curve25519.ScalarBaseMult(expectedRes.el(), randomPoint.el())

	assert.Equal(t, result, expectedRes)
}

func TestExpoBaseMultipleValue(t *testing.T) {
	// TODO: figure out a way to test it without having the FeMul function
}

func TestHash(t *testing.T) {
	randomPoint, err := RandomElement()
	assert.Nil(t, err)

	hVal, err := hash(randomPoint.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, 32, len(hVal))
}

func TestGetAESKey(t *testing.T) {
	randomPoint, err := RandomElement()
	assert.Nil(t, err)

	aesKey, err := KDF(randomPoint.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, aes.BlockSize, len(aesKey))

}

func TestComputeBlindingFactor(t *testing.T) {
	// TODO: FIXME: currently I cannot test whether what we produce is actually the expected value
	// as I don't have any reference output for any input on this curve

	// basePoint is the x coordinate of the generator of the curve.
	// So it's as good point as any for the computation
	basePoint := [32]byte{9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	key, err := hash(basePoint[:])
	assert.Nil(t, err)
	b, err := computeBlindingFactor(key)
	assert.Nil(t, err)

	// I'M NOT SURE OF THAT EXPECTED VALUE
	expected := [32]byte{0xd, 0xe6, 0xd2, 0x55, 0xc7, 0xde, 0x9a, 0x67, 0x16, 0x92, 0x2f, 0x5d, 0xe9, 0xee,
		0x69, 0x44, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}

	assert.Equal(t, expected, b.bytes)
}

func TestGetSharedSecrets(t *testing.T) {
	_, pub1, err := GenerateKeyPair()
	assert.Nil(t, err)

	_, pub2, err := GenerateKeyPair()
	assert.Nil(t, err)

	_, pub3, err := GenerateKeyPair()
	assert.Nil(t, err)

	pubs := []*PublicKey{pub1, pub2, pub3}

	m1 := config.MixConfig{Id: "", Host: "", Port: "", PubKey: pub1.Bytes()}
	m2 := config.MixConfig{Id: "", Host: "", Port: "", PubKey: pub2.Bytes()}
	m3 := config.MixConfig{Id: "", Host: "", Port: "", PubKey: pub3.Bytes()}

	nodes := []config.MixConfig{m1, m2, m3}

	x, err := RandomElement()
	assert.Nil(t, err)

	result, err := getSharedSecrets(nodes, x)
	assert.Nil(t, err)

	var expected []HeaderInitials
	blindFactors := []*FieldElement{x}

	v := x
	alpha0 := new(FieldElement)
	curve25519.ScalarBaseMult(alpha0.el(), v.el()) // alpha0 = g^x
	s0 := expo(pubs[0].ToFieldElement(), blindFactors)
	aesS0, err := KDF(s0.Bytes())
	assert.Nil(t, err)
	b0, err := computeBlindingFactor(aesS0)
	assert.Nil(t, err)

	expected = append(expected, HeaderInitials{Alpha: alpha0.Bytes(),
		Secret:     s0.Bytes(),
		Blinder:    b0.Bytes(),
		SecretHash: aesS0,
	})
	blindFactors = append(blindFactors, b0)

	alpha1 := new(FieldElement)
	curve25519.ScalarMult(alpha1.el(), b0.el(), alpha0.el()) // alpha1 = g^(x * b0)
	s1 := expo(pubs[1].ToFieldElement(), blindFactors)
	aesS1, err := KDF(s1.Bytes())
	assert.Nil(t, err)
	b1, err := computeBlindingFactor(aesS1)
	assert.Nil(t, err)

	expected = append(expected, HeaderInitials{Alpha: alpha1.Bytes(),
		Secret:     s1.Bytes(),
		Blinder:    b1.Bytes(),
		SecretHash: aesS1,
	})
	blindFactors = append(blindFactors, b1)

	alpha2 := new(FieldElement)
	curve25519.ScalarMult(alpha2.el(), b1.el(), alpha1.el()) // alpha2 = g^(x * b0 * b1)
	s2 := expo(pubs[2].ToFieldElement(), blindFactors)
	aesS2, err := KDF(s2.Bytes())
	assert.Nil(t, err)
	b2, err := computeBlindingFactor(aesS2)
	assert.Nil(t, err)

	expected = append(expected, HeaderInitials{Alpha: alpha2.Bytes(),
		Secret:     s2.Bytes(),
		Blinder:    b2.Bytes(),
		SecretHash: aesS2,
	})
	// this assignment was ineffectual
	// blindFactors = append(blindFactors, *b2)

	assert.Equal(t, expected, result)
}

func TestComputeFillers(t *testing.T) {
	basePoint := [32]byte{9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	h1 := HeaderInitials{Alpha: []byte{}, Secret: basePoint[:], Blinder: []byte{}, SecretHash: []byte("1111111111111111")}
	h2 := HeaderInitials{Alpha: []byte{}, Secret: basePoint[:], Blinder: []byte{}, SecretHash: []byte("1111111111111111")}
	h3 := HeaderInitials{Alpha: []byte{}, Secret: basePoint[:], Blinder: []byte{}, SecretHash: []byte("1111111111111111")}
	tuples := []HeaderInitials{h1, h2, h3}

	_, pub1, err := GenerateKeyPair()
	assert.Nil(t, err)

	_, pub2, err := GenerateKeyPair()
	assert.Nil(t, err)

	_, pub3, err := GenerateKeyPair()
	assert.Nil(t, err)

	m1 := config.MixConfig{Id: "", Host: "", Port: "", PubKey: pub1.Bytes()}
	m2 := config.MixConfig{Id: "", Host: "", Port: "", PubKey: pub2.Bytes()}
	m3 := config.MixConfig{Id: "", Host: "", Port: "", PubKey: pub3.Bytes()}

	fillers, err := computeFillers([]config.MixConfig{m1, m2, m3}, tuples)
	assert.Nil(t, err)

	fmt.Println("FILLER: ", fillers)

}

func TestXorBytesPass(t *testing.T) {
	result := XorBytes([]byte("00101"), []byte("10110"))
	assert.Equal(t, []byte{1, 0, 0, 1, 1}, result)
}

func TestXorBytesFail(t *testing.T) {
	result := XorBytes([]byte("00101"), []byte("10110"))
	assert.NotEqual(t, []byte("00000"), result)
}

func TestEncapsulateHeader(t *testing.T) {
	_, pub1, err := GenerateKeyPair()
	assert.Nil(t, err)

	_, pub2, err := GenerateKeyPair()
	assert.Nil(t, err)

	_, pub3, err := GenerateKeyPair()
	assert.Nil(t, err)

	_, pubD, err := GenerateKeyPair()
	assert.Nil(t, err)

	m1 := config.NewMixConfig("Node1", "localhost", "3331", pub1.Bytes(), 1)
	m2 := config.NewMixConfig("Node2", "localhost", "3332", pub2.Bytes(), 2)
	m3 := config.NewMixConfig("Node3", "localhost", "3333", pub3.Bytes(), 3)

	nodes := []config.MixConfig{m1, m2, m3}

	c1 := Commands{Delay: 0.34, Flag: []byte("0")}
	c2 := Commands{Delay: 0.25, Flag: []byte("1")}
	c3 := Commands{Delay: 1.10, Flag: []byte("1")}
	commands := []Commands{c1, c2, c3}

	x, err := RandomElement()
	assert.Nil(t, err)
	sharedSecrets, err := getSharedSecrets(nodes, x)
	assert.Nil(t, err)

	actualHeader, err := encapsulateHeader(sharedSecrets, nodes, commands,
		config.ClientConfig{Id: "DestinationId", Host: "DestinationAddress", Port: "9998", PubKey: pubD.Bytes()})

	assert.Nil(t, err)

	routing1 := RoutingInfo{NextHop: &Hop{Id: "DestinationId",
		Address: "DestinationAddress:9998",
		PubKey:  []byte{},
	}, RoutingCommands: &c3,
		NextHopMetaData: []byte{},
		Mac:             []byte{},
	}

	routing1Bytes, err := proto.Marshal(&routing1)
	assert.Nil(t, err)

	kdfRes, err := KDF(sharedSecrets[2].SecretHash)
	assert.Nil(t, err)
	encRouting1, err := AesCtr(kdfRes, routing1Bytes)
	assert.Nil(t, err)

	mac1, err := computeMac(kdfRes, encRouting1)
	assert.Nil(t, err)

	routing2 := RoutingInfo{NextHop: &Hop{Id: "Node3",
		Address: "localhost:3333",
		PubKey:  pub3.Bytes(),
	}, RoutingCommands: &c2,
		NextHopMetaData: encRouting1,
		Mac:             mac1,
	}

	routing2Bytes, err := proto.Marshal(&routing2)
	assert.Nil(t, err)

	kdfRes, err = KDF(sharedSecrets[1].SecretHash)
	assert.Nil(t, err)

	encRouting2, err := AesCtr(kdfRes, routing2Bytes)
	assert.Nil(t, err)

	mac2, err := computeMac(kdfRes, encRouting2)
	assert.Nil(t, err)

	expectedRouting := RoutingInfo{NextHop: &Hop{Id: "Node2",
		Address: "localhost:3332",
		PubKey:  pub2.Bytes(),
	}, RoutingCommands: &c1,
		NextHopMetaData: encRouting2,
		Mac:             mac2,
	}

	expectedRoutingBytes, err := proto.Marshal(&expectedRouting)
	assert.Nil(t, err)

	kdfRes, err = KDF(sharedSecrets[0].SecretHash)
	assert.Nil(t, err)

	encExpectedRouting, err := AesCtr(kdfRes, expectedRoutingBytes)
	assert.Nil(t, err)

	mac3, err := computeMac(kdfRes, encExpectedRouting)
	assert.Nil(t, err)

	expectedHeader := Header{Alpha: sharedSecrets[0].Alpha,
		Beta: encExpectedRouting,
		Mac:  mac3,
	}

	assert.Equal(t, expectedHeader, actualHeader)
}

func TestProcessSphinxHeader(t *testing.T) {
	priv1, pub1, err := GenerateKeyPair()
	assert.Nil(t, err)

	_, pub2, err := GenerateKeyPair()
	assert.Nil(t, err)

	_, pub3, err := GenerateKeyPair()
	assert.Nil(t, err)

	c1 := Commands{Delay: 0.34}
	c2 := Commands{Delay: 0.25}
	c3 := Commands{Delay: 1.10}

	m1 := config.NewMixConfig("Node1", "localhost", "3331", pub1.Bytes(), 1)
	m2 := config.NewMixConfig("Node2", "localhost", "3332", pub2.Bytes(), 2)
	m3 := config.NewMixConfig("Node3", "localhost", "3333", pub3.Bytes(), 3)

	nodes := []config.MixConfig{m1, m2, m3}

	x, err := RandomElement()
	assert.Nil(t, err)
	sharedSecrets, err := getSharedSecrets(nodes, x)
	assert.Nil(t, err)

	// Intermediate steps, which are needed to check whether the processing of the header was correct
	routing1 := RoutingInfo{NextHop: &Hop{Id: "DestinationId",
		Address: "DestinationAddress", PubKey: []byte{},
	}, RoutingCommands: &c3,
		NextHopMetaData: []byte{},
		Mac:             []byte{},
	}

	routing1Bytes, err := proto.Marshal(&routing1)
	assert.Nil(t, err)

	kdfRes, err := KDF(sharedSecrets[2].SecretHash)
	assert.Nil(t, err)

	encRouting1, err := AesCtr(kdfRes, routing1Bytes)
	assert.Nil(t, err)

	mac1, err := computeMac(kdfRes, encRouting1)
	assert.Nil(t, err)

	routing2 := RoutingInfo{NextHop: &Hop{Id: "Node3",
		Address: "localhost:3333",
		PubKey:  pub3.Bytes(),
	}, RoutingCommands: &c2,
		NextHopMetaData: encRouting1,
		Mac:             mac1,
	}

	routing2Bytes, err := proto.Marshal(&routing2)
	assert.Nil(t, err)

	kdfRes, err = KDF(sharedSecrets[1].SecretHash)
	assert.Nil(t, err)

	encRouting2, err := AesCtr(kdfRes, routing2Bytes)
	assert.Nil(t, err)

	mac2, err := computeMac(kdfRes, encRouting2)
	assert.Nil(t, err)

	routing3 := RoutingInfo{NextHop: &Hop{Id: "Node2",
		Address: "localhost:3332",
		PubKey:  pub2.Bytes(),
	}, RoutingCommands: &c1,
		NextHopMetaData: encRouting2,
		Mac:             mac2,
	}

	routing3Bytes, err := proto.Marshal(&routing3)
	assert.Nil(t, err)

	kdfRes, err = KDF(sharedSecrets[0].SecretHash)
	assert.Nil(t, err)

	encExpectedRouting, err := AesCtr(kdfRes, routing3Bytes)
	assert.Nil(t, err)

	mac3, err := computeMac(kdfRes, encExpectedRouting)
	assert.Nil(t, err)

	header := Header{Alpha: sharedSecrets[0].Alpha,
		Beta: encExpectedRouting,
		Mac:  mac3,
	}

	nextHop, newCommands, newHeader, err := ProcessSphinxHeader(header, priv1)

	assert.Nil(t, err)

	assert.True(t, proto.Equal(&nextHop, &Hop{Id: "Node2", Address: "localhost:3332", PubKey: pub2.Bytes()}))
	assert.True(t, proto.Equal(&newCommands, &c1))
	assert.True(t, proto.Equal(&newHeader, &Header{Alpha: sharedSecrets[1].Alpha, Beta: encRouting2, Mac: mac2}))

}

func TestProcessSphinxPayload(t *testing.T) {

	message := "Plaintext message"

	priv1, pub1, err := GenerateKeyPair()
	assert.Nil(t, err)

	priv2, pub2, err := GenerateKeyPair()
	assert.Nil(t, err)

	priv3, pub3, err := GenerateKeyPair()
	assert.Nil(t, err)

	m1 := config.NewMixConfig("Node1", "localhost", "3331", pub1.Bytes(), 1)
	m2 := config.NewMixConfig("Node2", "localhost", "3332", pub2.Bytes(), 2)
	m3 := config.NewMixConfig("Node3", "localhost", "3333", pub3.Bytes(), 3)

	nodes := []config.MixConfig{m1, m2, m3}

	x, err := RandomElement()
	assert.Nil(t, err)
	headerInitials, err := getSharedSecrets(nodes, x)
	assert.Nil(t, err)

	encMsg, err := encapsulateContent(headerInitials, message)
	assert.Nil(t, err)

	decMsg := encMsg
	privs := []*PrivateKey{priv1, priv2, priv3}
	for i, v := range privs {
		decMsg, err = ProcessSphinxPayload(headerInitials[i].Alpha, decMsg, v)
		if err != nil {
			t.Error(err)
		}
	}
	assert.Equal(t, []byte(message), decMsg)
}
