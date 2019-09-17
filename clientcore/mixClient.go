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
	Package clientcore implements all the necessary functions for the mix client, i.e., the core of the client
	which allows to process the received cryptographic packets.
*/

package clientcore

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/helpers"
	"github.com/nymtech/loopix-messaging/sphinx"
	"github.com/sirupsen/logrus"
)

const (
	maximumTopologyAge = 1 * time.Minute
)

var (
	// ErrInvalidMixes defines an error when either the mix map is nil or contains insufficient number of entries
	ErrInvalidMixes = errors.New("insufficient number of mixes provided")
)

// NetworkPKI holds PKI data about tne current network topology.
// This allows public-key encryption to happen.
type NetworkPKI struct {
	lastUpdated time.Time
	Mixes       map[uint][]config.MixConfig
	Clients     []config.ClientConfig
}

func (n *NetworkPKI) UpdateNetwork(newMixes map[uint][]config.MixConfig, newClients []config.ClientConfig) {
	n.Mixes = newMixes
	n.Clients = newClients
	n.lastUpdated = time.Now()
}

func (n *NetworkPKI) ShouldUpdate() bool {
	return n.lastUpdated.Add(maximumTopologyAge).Before(time.Now())
}

// MixClient does sphinx packet encoding and decoding.
type MixClient interface {
	EncodeIntoSphinxPacket(message string, recipient config.ClientConfig) ([]byte, error)
	DecodeSphinxPacket(packet sphinx.SphinxPacket) (sphinx.SphinxPacket, error)
	GetPublicKey() *sphinx.PublicKey
}

// CryptoClient contains a public/private keypair and an elliptic curve for a given provider and network.
type CryptoClient struct {
	pubKey   *sphinx.PublicKey
	prvKey   *sphinx.PrivateKey
	Provider config.MixConfig
	Network  NetworkPKI
	log      *logrus.Logger
}

const (
	desiredRateParameter = 5
	pathLength           = 3
)

// CreateSphinxPacket responsible for sending a real message. Takes as input the message string
// and the public information about the destination.
// The function generates a random path and a set of random values from exponential distribution.
// Given those values it triggers the encode function, which packs the message into the
// sphinx cryptographic packet format. Next, the encoded packet is combined with a
// flag signalling that this is a usual network packet, and passed to be send.
// The function returns an error if any issues occurred.
func (c *CryptoClient) createSphinxPacket(message string, recipient config.ClientConfig) ([]byte, error) {

	path, err := c.buildPath(recipient)
	if err != nil {
		c.log.Errorf("Error in CreateSphinxPacket - generating random path failed: %v", err)
		return nil, err
	}

	delays, err := c.generateDelaySequence(desiredRateParameter, path.Len())
	if err != nil {
		c.log.Errorf("Error in CreateSphinxPacket - generating sequence of delays failed: %v", err)
		return nil, err
	}

	sphinxPacket, err := sphinx.PackForwardMessage(path, delays, message)
	if err != nil {
		c.log.Errorf("Error in CreateSphinxPacket - the pack procedure failed: %v", err)
		return nil, err
	}

	return proto.Marshal(&sphinxPacket)
}

// buildPath builds a path containing the sender's provider,
// a sequence (of length pre-defined in a config file) of randomly
// selected mixes and the recipient's provider
func (c *CryptoClient) buildPath(recipient config.ClientConfig) (config.E2EPath, error) {
	mixSeq, err := c.getRandomMixSequence(c.Network.Mixes, pathLength)
	if err != nil {
		c.log.Errorf("error in buildPath - generating random mix path failed: %v", err)
		return config.E2EPath{}, err
	}

	if recipient.Provider == nil || len(recipient.Provider.PubKey) == 0 {
		err := fmt.Errorf("error in buildPath - could not create path to the recipient," +
			" the EgressProvider has invalid configuration")
		c.log.Error(err.Error())
		return config.E2EPath{}, err
	}
	path := config.E2EPath{IngressProvider: c.Provider,
		Mixes:          mixSeq,
		EgressProvider: *recipient.Provider,
		Recipient:      recipient,
	}
	return path, nil
}

// getRandomMixSequence generates a random sequence of given length from all possible mixes.
// If the list of all active mixes is empty or the given length is larger than the set of active mixes,
// an error is returned.
func (c *CryptoClient) getRandomMixSequence(mixes map[uint][]config.MixConfig, length int) ([]config.MixConfig, error) {
	if mixes == nil || len(mixes) < length {
		return nil, ErrInvalidMixes
	}

	mixSequence := make([]config.MixConfig, length)
	for i := 1; i <= length; i++ {
		if layerMixes, ok := mixes[uint(i)]; ok {
			mixSequence[i-1] = helpers.RandomMix(layerMixes)
		} else {
			return nil, fmt.Errorf("No valid mixes for layer: %v", i)
		}
	}

	return mixSequence, nil
}

// generateDelaySequence generates a given length sequence of float64 values. Values are generated
// following the exponential distribution. generateDelaySequence returnes a sequence or an error
// if any of the values could not be generate.
func (c *CryptoClient) generateDelaySequence(desiredRateParameter float64, length int) ([]float64, error) {
	var delays []float64
	for i := 0; i < length; i++ {
		d, err := helpers.RandomExponential(desiredRateParameter)
		if err != nil {
			c.log.Errorf("Error in generateDelaySequence - generating random exponential sample failed: %v", err)
			return nil, err
		}
		delays = append(delays, d)
	}
	return delays, nil
}

// EncodeMessage encodes given message into the Sphinx packet format. EncodeMessage takes as inputs
// the message and the recipient's public configuration.
// EncodeMessage returns the byte representation of the packet or an error if the packet could not be created.
func (c *CryptoClient) EncodeMessage(message string, recipient config.ClientConfig) ([]byte, error) {

	packet, err := c.createSphinxPacket(message, recipient)
	if err != nil {
		c.log.Errorf("Error in EncodeMessage - the pack procedure failed: %v", err)
		return nil, err
	}
	return packet, err
}

// DecodeMessage decodes the received sphinx packet.
// TODO: this function is finished yet.
func (c *CryptoClient) DecodeMessage(packet sphinx.SphinxPacket) (sphinx.SphinxPacket, error) {
	return packet, nil
}

// GetPublicKey returns the public key for this CryptoClient
func (c *CryptoClient) GetPublicKey() *sphinx.PublicKey {
	return c.pubKey
}

// NewCryptoClient constructor function
// TODO: Same issue as with the 'NewClient' function
func NewCryptoClient(privKey *sphinx.PrivateKey,
	pubKey *sphinx.PublicKey,
	provider config.MixConfig,
	network NetworkPKI,
	log *logrus.Logger,
) *CryptoClient {
	return &CryptoClient{prvKey: privKey,
		pubKey:   pubKey,
		Provider: provider,
		Network:  network,
		log:      log,
	}
}
