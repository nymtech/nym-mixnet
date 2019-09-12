// Copyright 2018-2019 The Loopix-Messaging Authors
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
	Package config implements struct for easy processing and storing of all public information
	of the network participants.
*/

package config

import (
	"github.com/golang/protobuf/proto"
)

//nolint: gochecknoglobals
var (
	// TODO: perhaps move it elsewhere?
	// TODO: proper comments on them
	// AssigneFlag ...
	AssigneFlag = []byte("\xa2")
	// CommFlag ...
	CommFlag = []byte("\xc6")
	// TokenFlag ...
	TokenFlag = []byte("\xa9")
	// PullFlag ...
	PullFlag = []byte("\xff")
)

// NewMixConfig constructor
func NewMixConfig(mixID, host, port string, pubKey []byte) MixConfig {
	return MixConfig{Id: mixID, Host: host, Port: port, PubKey: pubKey}
}

// NewClientConfig constructor
func NewClientConfig(clientID, host, port string, pubKey []byte, providerInfo MixConfig) ClientConfig {
	client := ClientConfig{Id: clientID, Host: host, Port: port, PubKey: pubKey, Provider: &providerInfo}
	return client
}

// WrapWithFlag packs the given byte information together with a specified flag into the
// packet.
func WrapWithFlag(flag []byte, data []byte) ([]byte, error) {
	m := GeneralPacket{Flag: flag, Data: data}
	mBytes, err := proto.Marshal(&m)
	if err != nil {
		return nil, err
	}
	return mBytes, nil
}

// E2EPath holds end to end path data for an entire route, prior to Sphinx header encryption
type E2EPath struct {
	IngressProvider MixConfig
	Mixes           []MixConfig
	EgressProvider  MixConfig
	Recipient       ClientConfig
}

// Len adds 3 to the mix path. TODO: why? Check this with Ania.
func (p *E2EPath) Len() int {
	return 3 + len(p.Mixes)
}
