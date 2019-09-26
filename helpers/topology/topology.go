// Copyright 2019 The Loopix-Messaging Authors
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
	Package topology implements all useful topology-related functions
	which are used in the code of anonymous messaging system.
*/

package topology

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/nymtech/nym-directory/models"
	"github.com/nymtech/nym-mixnet/config"
)

// MixPresence defines map containing presence information of all mix nodes in given topology.
type MixPresence []models.MixNodePresence

// MixPresence defines map containing presence information of all providers in given topology.
type ProviderPresence []models.MixProviderPresence

// LayeredMixes defines map of list of mix nodes corresponding to particular layer in given topology.
type LayeredMixes map[uint][]config.MixConfig

const (
	DefaultClientHost = "0.0.0.0"
	DefaultClientPort = "42"
)

func GetNetworkTopology(endpoint string) (*models.Topology, error) {
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	model := &models.Topology{}
	if err := json.Unmarshal(body, model); err != nil {
		return nil, err
	}

	return model, nil
}

// GetMixesPKI returns PKI data for mix nodes, grouped by layer
func GetMixesPKI(mixPresence MixPresence) (LayeredMixes, error) {
	mixes := make(LayeredMixes)
	for k, v := range mixPresence {
		b, err := base64.URLEncoding.DecodeString(v.PubKey)
		if err != nil {
			continue
		}
		host, port, err := net.SplitHostPort(v.Host) // TODO: do we want to split them?
		if err != nil {
			continue
		}
		newMixEntry := config.MixConfig{
			Id:     mixPresence[k].PubKey,
			Host:   host,
			Port:   port,
			PubKey: b,
			Layer:  uint64(v.Layer),
		}
		if layerMixes, ok := mixes[v.Layer]; ok {
			extendedLayer := append(layerMixes, newMixEntry)
			mixes[v.Layer] = extendedLayer
		} else {
			mixes[v.Layer] = []config.MixConfig{newMixEntry}
		}
	}
	return mixes, nil
}

func ProviderPresenceToConfig(presence models.MixProviderPresence) (config.MixConfig, error) {
	b, err := base64.URLEncoding.DecodeString(presence.PubKey)
	if err != nil {
		return config.MixConfig{}, errors.New("invalid provider presence")
	}
	host, port, err := net.SplitHostPort(presence.Host) // TODO: do we want to split them?
	if err != nil {
		return config.MixConfig{}, err
	}

	return config.NewMixConfig(presence.Host, host, port, b, config.ProviderLayer), nil
}

func RegisteredClientToConfig(client models.RegisteredClient) (config.ClientConfig, error) {
	b, err := base64.URLEncoding.DecodeString(client.PubKey)
	if err != nil {
		return config.ClientConfig{}, errors.New("invalid client information")
	}

	return config.ClientConfig{
		Id:     client.PubKey,
		Host:   DefaultClientHost,
		Port:   DefaultClientPort,
		PubKey: b,
	}, nil
}

// GetClientPKI returns a map of the current client PKI from the PKI database
func GetClientPKI(providerPresence ProviderPresence) ([]config.ClientConfig, error) {
	var clientsNum int = 0
	for _, v := range providerPresence {
		clientsNum += len(v.RegisteredClients)
	}

	clients := make([]config.ClientConfig, 0, clientsNum)
	for _, provider := range providerPresence {
		providerCfg, err := ProviderPresenceToConfig(provider)
		if err != nil {
			continue
		}
		for _, client := range provider.RegisteredClients {
			clientCfg, err := RegisteredClientToConfig(client)
			if err != nil {
				continue
			}
			clientCfg.Provider = &providerCfg
			clients = append(clients, clientCfg)
		}
	}
	return clients, nil
}
