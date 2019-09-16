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
/*
	Package node implements the core functions for a mix node, which allow to process the received cryptographic packets.
*/
package node

import (
	"time"

	"github.com/nymtech/loopix-messaging/flags"
	"github.com/nymtech/loopix-messaging/sphinx"
)

type Mix struct {
	pubKey *sphinx.PublicKey
	prvKey *sphinx.PrivateKey
}

type PacketProcessingResult struct {
	packetData []byte
	nextHop    sphinx.Hop
	flag       flags.SphinxFlag
	err        error
}

func (p *PacketProcessingResult) PacketData() []byte {
	return p.packetData
}

func (p *PacketProcessingResult) NextHop() sphinx.Hop {
	return p.nextHop
}

func (p *PacketProcessingResult) Flag() flags.SphinxFlag {
	return p.flag
}

func (p *PacketProcessingResult) Err() error {
	return p.err
}

// ProcessPacket performs the processing operation on the received packet, including cryptographic operations and
// extraction of the meta information.
func (m *Mix) ProcessPacket(packet []byte) *PacketProcessingResult {
	res := new(PacketProcessingResult)

	nextHop, commands, newPacket, err := sphinx.ProcessSphinxPacket(packet, m.prvKey)
	res.err = err

	// rather than sleeping in new gouroutine and waiting for channel data that is sent from it
	// just sleep in the main goroutine and avoid extra communication overhead
	time.Sleep(time.Second * time.Duration(commands.Delay))

	res.packetData = newPacket
	res.nextHop = nextHop
	res.flag = flags.SphinxFlagFromBytes(commands.Flag)

	return res
}

// GetPublicKey returns the public key of the mixnode.
func (m *Mix) GetPublicKey() *sphinx.PublicKey {
	return m.pubKey
}

// NewMix creates a new instance of Mix struct with given public and private key
func NewMix(prvKey *sphinx.PrivateKey, pubKey *sphinx.PublicKey) *Mix {
	return &Mix{prvKey: prvKey,
		pubKey: pubKey,
	}
}
