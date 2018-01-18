package node

import (
	"os"
	"testing"

	"anonymous-messaging/publics"
	sphinx "anonymous-messaging/new_packet_format"
	"crypto/elliptic"
	"github.com/stretchr/testify/assert"
	"reflect"
)

var mixWorker Mix
var testPacket sphinx.SphinxPacket
var nodes []publics.MixPubs
var curve elliptic.Curve

func TestMain(m *testing.M) {
	curve := elliptic.P224()

	pub1, priv1 := publics.GenerateKeyPair()
	pub2, _ := publics.GenerateKeyPair()
	pub3, _ := publics.GenerateKeyPair()

	m1 := publics.MixPubs{Id: "Mix1", Host: "localhost", Port: "3330", PubKey: pub1}
	m2 := publics.MixPubs{Id: "Mix2", Host: "localhost", Port: "3331", PubKey: pub2}
	m3 := publics.MixPubs{Id: "Mix2", Host: "localhost", Port: "3332", PubKey: pub3}

	mixWorker = *NewMix("MixWorker", pub1, priv1)

	nodes = []publics.MixPubs{m1, m2, m3}
	c1 := sphinx.Commands{Delay: 1.4, Flag: ""}
	c2 := sphinx.Commands{Delay: 2.5, Flag: ""}
	c3 := sphinx.Commands{Delay: 2.3, Flag: ""}

	pubD, _ := publics.GenerateKeyPair()
	dest := publics.MixPubs{Id : "Destination", Host: "localhost", Port: "3334", PubKey: pubD}

	testPacket = sphinx.PackForwardMessage(curve, nodes, []publics.PublicKey{pub1, pub2, pub3}, []sphinx.Commands{c1, c2, c3}, dest, "Test Message")
	os.Exit(m.Run())
}

func TestMixProcessPacket(t *testing.T) {
	ch := make(chan []byte, 1)
	chHop := make(chan sphinx.Hop, 1)

	mixWorker.ProcessPacket(testPacket.Bytes(), ch, chHop)
	dePacket := <-ch
	nextHop := <- chHop

	assert.Equal(t, sphinx.Hop{Id: "Mix2", Address: "localhost:3331", PubKey: nodes[1].PubKey.Bytes()}, nextHop, "Next hope does not match")
	assert.Equal(t, reflect.TypeOf([]byte{}), reflect.TypeOf(dePacket))
}
