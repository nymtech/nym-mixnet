// Copyright 2018 The Nym Mixnet Authors
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

/*	Package sphinx implements the library of a cryptographic packet format,
	which can be used to secure the content as well as the metadata of the transported
    messages.
*/

package sphinx

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/nym-mixnet/config"
	"github.com/nymtech/nym-mixnet/flags"
	"golang.org/x/crypto/curve25519"
)

const (
	// K TODO: document padding-related Sphinx parameter
	K            = 16
	headerLength = 192
)

// PackForwardMessage encapsulates the given message into the cryptographic Sphinx packet format.
// As arguments the function takes the path, consisting of the sequence of nodes the packet should traverse
// and the destination of the message, a set of delays and the information about the curve used to perform cryptographic
// operations.
// In order to encapsulate the message PackForwardMessage computes two parts of the packet - the header and
// the encrypted payload. If creating of any of the packet block failed, an error is returned. Otherwise,
// a Sphinx packet format is returned.
func PackForwardMessage(path config.E2EPath, delays []float64, message []byte) (SphinxPacket, error) {
	nodes := []config.MixConfig{path.IngressProvider}
	nodes = append(nodes, path.Mixes...)
	nodes = append(nodes, path.EgressProvider)
	dest := path.Recipient

	headerInitials, header, err := createHeader(nodes, delays, dest)
	if err != nil {
		errMsg := fmt.Errorf("error in PackForwardMessage - createHeader failed: %v", err)
		return SphinxPacket{}, errMsg
	}

	payload, err := encapsulateContent(headerInitials, message)
	if err != nil {
		errMsg := fmt.Errorf("error in PackForwardMessage - encapsulateContent failed: %v", err)
		return SphinxPacket{}, errMsg
	}
	return SphinxPacket{Hdr: &header, Pld: payload}, nil
}

// createHeader builds the Sphinx packet header, consisting of three parts: the public element,
// the encapsulated routing information and the message authentication code.
// createHeader layer encapsulates the routing information for each given node. The routing information
// contains information where the packet should be forwarded next, how long it should be delayed by the node,
// and if relevant additional auxiliary information. The message authentication code allows to detect tagging attacks.
// createHeader computes the secret shared key between sender and the nodes and destination,
// which are used as keys for encryption.
// createHeader returns the header and a list of the initial elements, used for creating the header.
// If any operation was unsuccessful createHeader returns an error.
func createHeader(nodes []config.MixConfig,
	delays []float64,
	dest config.ClientConfig,
) ([]HeaderInitials, Header, error) {
	x, err := RandomElement()
	if err != nil {
		errMsg := fmt.Errorf("error in createHeader - Random failed: %v", err)
		return nil, Header{}, errMsg
	}

	headerInitials, err := getSharedSecrets(nodes, x)
	if err != nil {
		errMsg := fmt.Errorf("error in createHeader - getSharedSecrets failed: %v", err)
		return nil, Header{}, errMsg
	}

	if len(headerInitials) != len(nodes) {
		errMsg := fmt.Errorf("error in createHeader - wrong number of shared secrets failed: %v", err)
		return nil, Header{}, errMsg
	}

	commands := make([]Commands, len(nodes))
	for i := range nodes {
		var c Commands
		if i == len(nodes)-1 {
			c = Commands{Delay: delays[i], Flag: flags.LastHopFlag.Bytes()}
		} else {
			c = Commands{Delay: delays[i], Flag: flags.RelayFlag.Bytes()}
		}
		commands[i] = c
	}

	header, err := encapsulateHeader(headerInitials, nodes, commands, dest)
	if err != nil {
		errMsg := fmt.Errorf("error in createHeader - encapsulateHeader failed: %v", err)
		return nil, Header{}, errMsg
	}
	return headerInitials, header, nil

}

// encapsulateHeader layer encrypts the meta-data of the packet, containing information about the
// sequence of nodes the packet should traverse before reaching the destination, and message authentication codes,
// given the pre-computed shared keys which are used for encryption.
// encapsulateHeader returns the Header, or an error if any internal cryptographic of parsing operation failed.
func encapsulateHeader(headerInitials []HeaderInitials,
	nodes []config.MixConfig,
	commands []Commands,
	destination config.ClientConfig,
) (Header, error) {
	finalHop := RoutingInfo{NextHop: &Hop{Id: destination.Id,
		Address: destination.Host + ":" + destination.Port,
		PubKey:  []byte{},
	}, RoutingCommands: &commands[len(commands)-1],
		NextHopMetaData: []byte{},
		Mac:             []byte{},
	}

	finalHopBytes, err := proto.Marshal(&finalHop)
	if err != nil {
		return Header{}, err
	}

	kdfRes, err := KDF(headerInitials[len(headerInitials)-1].SecretHash)
	if err != nil {
		return Header{}, err
	}

	encFinalHop, err := AesCtr(kdfRes, finalHopBytes)
	if err != nil {
		errMsg := fmt.Errorf("error in encapsulateHeader - AES_CTR encryption failed: %v", err)
		return Header{}, errMsg
	}

	mac, err := computeMac(kdfRes, encFinalHop)
	if err != nil {
		return Header{}, err
	}

	routingCommands := [][]byte{encFinalHop}

	var encRouting []byte
	for i := len(nodes) - 2; i >= 0; i-- {
		nextNode := nodes[i+1]
		routing := RoutingInfo{NextHop: &Hop{Id: nextNode.Id,
			Address: nextNode.Host + ":" + nextNode.Port,
			PubKey:  nodes[i+1].PubKey,
		}, RoutingCommands: &commands[i],
			NextHopMetaData: routingCommands[len(routingCommands)-1],
			Mac:             mac,
		}

		encKey, err := KDF(headerInitials[i].SecretHash)
		if err != nil {
			return Header{}, err
		}

		routingBytes, err := proto.Marshal(&routing)
		if err != nil {
			return Header{}, err
		}

		encRouting, err = AesCtr(encKey, routingBytes)
		if err != nil {
			return Header{}, err
		}

		routingCommands = append(routingCommands, encRouting)
		kdfResL, err := KDF(headerInitials[i].SecretHash)
		if err != nil {
			return Header{}, nil
		}
		mac, err = computeMac(kdfResL, encRouting)
		if err != nil {
			return Header{}, err
		}

	}
	return Header{Alpha: headerInitials[0].Alpha, Beta: encRouting, Mac: mac}, nil

}

// encapsulateContent layer encrypts the given messages using a set of shared keys
// and the AES_CTR encryption.
// encapsulateContent returns the encrypted payload in byte representation. If the AES_CTR
// encryption failed encapsulateContent returns an error.
func encapsulateContent(headerInitials []HeaderInitials, message []byte) ([]byte, error) {

	enc := message

	for i := len(headerInitials) - 1; i >= 0; i-- {
		sharedKey, err := KDF(headerInitials[i].SecretHash)
		if err != nil {
			return nil, err
		}
		enc, err = AesCtr(sharedKey, enc)
		if err != nil {
			errMsg := fmt.Errorf("error in encapsulateContent - AES_CTR encryption failed: %v", err)
			return nil, errMsg
		}

	}
	return enc, nil
}

// getSharedSecrets computes a sequence of HeaderInitial values, containing the initial elements,
// shared secrets and blinding factors for each node on the path. As input getSharedSecrets takes the initial
// secret value, the list of nodes, and the curve in which the cryptographic operations are performed.
// getSharedSecrets returns the list of computed HeaderInitials or an error.
func getSharedSecrets(nodes []config.MixConfig, initialVal *FieldElement) ([]HeaderInitials, error) {

	blindFactors := []*FieldElement{initialVal}
	tuples := make([]HeaderInitials, len(nodes))
	for i, n := range nodes {

		// initial implementation:
		// for x1, x2, ... xn in blindFactors:
		// compute tmp := x1 * x2 * ... xn
		// return g^tmp

		// replacing to:
		// for x1, x2, ... xn in blindFactors:
		// compute tmp1 := g^x1
		// tmp2 := tmp1^x2
		// ...
		// return tmp{n-1}^xn
		alpha := expoGroupBase(blindFactors)

		if len(n.PubKey) != PublicKeySize {
			errMsg := fmt.Errorf("invalid public key provided for node %v", i)
			return nil, errMsg
		}

		// initial implementation:
		// for x1, x2, ... xn in blindFactors:
		// compute tmp := x1 * x2 * ... xn
		// return base^tmp

		// replacing to:
		// for x1, x2, ... xn in blindFactors:
		// compute tmp1 := base^x1
		// tmp2 := tmp1^x2
		// ...
		// return tmpn-1^xn
		s := expo(BytesToPublicKey(n.PubKey).ToFieldElement(), blindFactors)

		// TODO: move to the other crypto file?
		aesS, err := KDF(s.Bytes())
		if err != nil {
			return nil, err
		}

		blinder, err := computeBlindingFactor(aesS)
		if err != nil {
			errMsg := fmt.Errorf("error in getSharedSecrets - computeBlindingFactor failed: %v", err)
			return nil, errMsg
		}

		blindFactors = append(blindFactors, blinder)
		tuples[i] = HeaderInitials{Alpha: alpha.Bytes(), Secret: s.Bytes(), Blinder: blinder.Bytes(), SecretHash: aesS}
	}
	return tuples, nil

}

// TODO: computeFillers needs to be fixed
func computeFillers(nodes []config.MixConfig, tuples []HeaderInitials) (string, error) {

	filler := ""
	minLen := headerLength - 32
	for i := 1; i < len(nodes); i++ {
		base := filler + strings.Repeat("\x00", K)
		kx, err := computeSharedSecretHash(tuples[i-1].SecretHash, []byte("hrhohrhohrhohrho"))
		if err != nil {
			return "", err
		}
		mx := strings.Repeat("\x00", minLen) + base

		xorVal, err := AesCtr(kx, []byte(mx))
		if err != nil {
			errMsg := fmt.Errorf("error in computeFillers - AES_CTR failed: %v", err)
			return "", errMsg
		}

		filler = BytesToString(xorVal)
		filler = filler[minLen:]

		minLen -= K
	}

	return filler, nil

}

// computeBlindingFactor computes the blinding factor extracted from the
// shared secrets. Blinding factors allow both the sender and intermediate nodes
// recompute the shared keys used at each hop of the message processing.
// computeBlindingFactor returns a value of a blinding factor or an error.
func computeBlindingFactor(key []byte) (*FieldElement, error) {
	iv := []byte("initialvector000")
	blinderBytes, err := computeSharedSecretHash(key, iv)

	if err != nil {
		errMsg := fmt.Errorf("error in computeBlindingFactor - computeSharedSecretHash failed: %v", err)
		return nil, errMsg
	}

	return BytesToFieldElement(blinderBytes), nil
}

// computeSharedSecretHash computes the hash value of the shared secret key
// using AES_CTR.
func computeSharedSecretHash(key []byte, iv []byte) ([]byte, error) {
	aesCipher, err := aes.NewCipher(key)

	if err != nil {
		errMsg := fmt.Errorf("error in computeSharedSecretHash - creating new AES cipher failed: %v", err)
		return nil, errMsg
	}

	stream := cipher.NewCTR(aesCipher, iv)
	plaintext := []byte("0000000000000000")

	ciphertext := make([]byte, len(plaintext))
	stream.XORKeyStream(ciphertext, plaintext)

	return ciphertext, nil
}

// ProcessSphinxPacket processes the sphinx packet using the given private key.
// ProcessSphinxPacket unwraps one layer of both the header and the payload encryption.
// ProcessSphinxPacket returns a new packet and the routing information which should
// be used by the processing node. If any cryptographic or parsing operation failed ProcessSphinxPacket
// returns an error.
func ProcessSphinxPacket(packetBytes []byte, privKey *PrivateKey) (Hop, Commands, []byte, error) {

	var packet SphinxPacket
	err := proto.Unmarshal(packetBytes, &packet)

	if err != nil {
		errMsg := fmt.Errorf("error in ProcessSphinxPacket - unmarshal of packet failed: %v", err)
		return Hop{}, Commands{}, nil, errMsg
	}

	hop, commands, newHeader, err := ProcessSphinxHeader(*packet.Hdr, privKey)
	if err != nil {
		errMsg := fmt.Errorf("error in ProcessSphinxPacket - ProcessSphinxHeader failed: %v", err)
		return Hop{}, Commands{}, nil, errMsg
	}

	newPayload, err := ProcessSphinxPayload(packet.Hdr.Alpha, packet.Pld, privKey)
	if err != nil {
		errMsg := fmt.Errorf("error in ProcessSphinxPacket - ProcessSphinxPayload failed: %v", err)
		return Hop{}, Commands{}, nil, errMsg
	}

	newPacket := SphinxPacket{Hdr: &newHeader, Pld: newPayload}
	newPacketBytes, err := proto.Marshal(&newPacket)
	if err != nil {
		errMsg := fmt.Errorf("error in ProcessSphinxPacket - marshal of packet failed: %v", err)
		return Hop{}, Commands{}, nil, errMsg
	}

	return hop, commands, newPacketBytes, nil
}

// ProcessSphinxHeader unwraps one layer of encryption from the header of a sphinx packet.
// ProcessSphinxHeader recomputes the shared key and checks whether the message authentication code is valid.
// If not, the packet is dropped and error is returned. If MAC checking was passed successfully ProcessSphinxHeader
// performs the AES_CTR decryption, recomputes the blinding factor and updates the init public element from the header.
// Next, ProcessSphinxHeader extracts the routing information from the decrypted packet and returns it,
// together with the updated init public element.
// If any crypto or parsing operation failed ProcessSphinxHeader returns an error.
func ProcessSphinxHeader(packet Header, privKey *PrivateKey) (Hop, Commands, Header, error) {
	alpha := BytesToFieldElement(packet.Alpha)
	beta := packet.Beta
	mac := packet.Mac

	sharedSecret := new(FieldElement)
	curve25519.ScalarMult(sharedSecret.el(), privKey.ToFieldElement().el(), alpha.el())

	aesS, err := KDF(sharedSecret.Bytes())
	if err != nil {
		return Hop{}, Commands{}, Header{}, err
	}
	encKey, err := KDF(aesS)
	if err != nil {
		return Hop{}, Commands{}, Header{}, err
	}

	recomputedMac, err := computeMac(encKey, beta)
	if err != nil {
		return Hop{}, Commands{}, Header{}, err
	}

	if !bytes.Equal(recomputedMac, mac) {
		return Hop{}, Commands{}, Header{}, errors.New("packet processing error: MACs are not matching")
	}

	blinder, err := computeBlindingFactor(aesS)
	if err != nil {
		errMsg := fmt.Errorf("error in ProcessSphinxHeader - computeBlindingFactor failed: %v", err)
		return Hop{}, Commands{}, Header{}, errMsg
	}

	newAlpha := new(FieldElement)
	curve25519.ScalarMult(newAlpha.el(), blinder.el(), alpha.el())

	decBeta, err := AesCtr(encKey, beta)
	if err != nil {
		errMsg := fmt.Errorf("error in ProcessSphinxHeader - AES_CTR failed: %v", err)
		return Hop{}, Commands{}, Header{}, errMsg
	}

	var routingInfo RoutingInfo
	err = proto.Unmarshal(decBeta, &routingInfo)
	if err != nil {
		errMsg := fmt.Errorf("error in ProcessSphinxHeader - unmarshal of beta failed: %v", err)
		return Hop{}, Commands{}, Header{}, errMsg
	}
	nextHop, commands, nextBeta, nextMac := readBeta(routingInfo)

	return nextHop, commands, Header{Alpha: newAlpha.Bytes(), Beta: nextBeta, Mac: nextMac}, nil
}

// readBeta extracts all the fields from the RoutingInfo structure
func readBeta(beta RoutingInfo) (Hop, Commands, []byte, []byte) {
	nextHop := *beta.NextHop
	commands := *beta.RoutingCommands
	nextBeta := beta.NextHopMetaData
	nextMac := beta.Mac

	return nextHop, commands, nextBeta, nextMac
}

// ProcessSphinxPayload unwraps a single layer of the encryption from the sphinx packet payload.
// ProcessSphinxPayload first recomputes the shared secret which is used to perform the AES_CTR decryption.
// ProcessSphinxPayload returns the new packet payload or an error if the decryption failed.
func ProcessSphinxPayload(alpha []byte, payload []byte, privKey *PrivateKey) ([]byte, error) {
	sharedSecret := new(FieldElement)
	curve25519.ScalarMult(sharedSecret.el(), privKey.ToFieldElement().el(), BytesToFieldElement(alpha).el())

	aesS, err := KDF(sharedSecret.Bytes())
	if err != nil {
		return nil, err
	}

	decKey, err := KDF(aesS)
	if err != nil {
		return nil, err
	}

	decPayload, err := AesCtr(decKey, payload)
	if err != nil {
		errMsg := fmt.Errorf("error in ProcessSphinxPayload - AES_CTR decryption failed: %v", err)
		return nil, errMsg
	}

	return decPayload, nil
}
