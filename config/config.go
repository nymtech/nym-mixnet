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
	"github.com/nymtech/loopix-messaging/flags"
)

const (
	DirectoryServerBaseURL                = "http://localhost:8080/"
	DirectoryServerHealthcheckURL         = "http://localhost:8080/api/healthcheck"
	DirectoryServerMetricsURL             = "http://localhost:8080/api/metrics/mixes"
	DirectoryServerPkiURL                 = "http://localhost:8080/api/nodes"
	DirectoryServerMixPresenceURL         = "http://localhost:8080/api/presence/mixnodes"
	DirectoryServerMixProviderPresenceURL = "http://localhost:8080/api/presence/mixproviders"
	DirectoryServerTopology               = "http://localhost:8080/api/presence/topology"

	// TODO: somehow split mixConfig to distinguish providers and mixnodes?
	// But then we would have to deal with nasty interfaces and protobuf issues...
	ProviderLayer = 1000000
)

// NewMixConfig constructor
func NewMixConfig(mixID, host, port string, pubKey []byte, layer uint) MixConfig {
	return MixConfig{Id: mixID, Host: host, Port: port, PubKey: pubKey, Layer: uint64(layer)}
}

// NewClientConfig constructor
func NewClientConfig(clientID, host, port string, pubKey []byte, providerInfo MixConfig) ClientConfig {
	client := ClientConfig{Id: clientID, Host: host, Port: port, PubKey: pubKey, Provider: &providerInfo}
	return client
}

// WrapWithFlag packs the given byte information together with a specified flag into the
// packet.
func WrapWithFlag(flag flags.PacketTypeFlag, data []byte) ([]byte, error) {
	m := GeneralPacket{Flag: flag.Bytes(), Data: data}
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
