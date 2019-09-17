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
	Package helpers implements all useful functions which are used in the code of anonymous messaging system.
*/

package helpers

import (
	"encoding/base64"
	"errors"
	"net"

	"github.com/nymtech/directory-server/models"
	"github.com/nymtech/loopix-messaging/config"
)

// GetMixesPKI returns PKI data for mix nodes, grouped by layer
func GetMixesPKI(mixPresence map[string]models.MixNodePresence) (map[uint][]config.MixConfig, error) {
	mixes := make(map[uint][]config.MixConfig)
	for k, v := range mixPresence {
		b, err := base64.StdEncoding.DecodeString(v.PubKey)
		if err != nil {
			continue
		}
		host, port, err := net.SplitHostPort(v.Host) // TODO: do we want to split them?
		if err != nil {
			continue
		}
		newMixEntry := config.MixConfig{
			Id:     k,
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
	b, err := base64.StdEncoding.DecodeString(presence.PubKey)
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
	b, err := base64.StdEncoding.DecodeString(client.PubKey)
	if err != nil {
		return config.ClientConfig{}, errors.New("invalid client information")
	}
	host, port, err := net.SplitHostPort(client.Host) // TODO: do we want to split them?
	if err != nil {
		return config.ClientConfig{}, errors.New("invalid client information")
	}
	return config.ClientConfig{
		Id:     client.PubKey,
		Host:   host,
		Port:   port,
		PubKey: b,
	}, nil
}

// GetClientPKI returns a map of the current client PKI from the PKI database
func GetClientPKI(providerPresence map[string]models.MixProviderPresence) ([]config.ClientConfig, error) {
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
