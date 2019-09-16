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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/nymtech/directory-server/models"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/sphinx"
)

var (
	ErrInvalidLocalIP = errors.New("couldn't find a valid IP for your machine, check your internet connection")
)

// ResolveTCPAddress returns an address of TCP end point given a host and port.
func ResolveTCPAddress(host, port string) (*net.TCPAddr, error) {
	addr, err := net.ResolveTCPAddr("tcp", host+":"+port)
	if err != nil {
		return nil, err
	}
	return addr, nil
}

// GetLocalIP attempts to figure out a valid IP address for this machine.
func GetLocalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}

	return "", ErrInvalidLocalIP
}

// RegisterMixNodePresence registers server presence at the directory server.
func RegisterMixNodePresence(host string, publicKey *sphinx.PublicKey, layer int) error {
	b64Key := base64.StdEncoding.EncodeToString(publicKey.Bytes())
	values := map[string]interface{}{"host": host, "pubKey": b64Key, "layer": layer}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return err
	}

	resp, err := http.Post(config.DirectoryServerMixPresenceURL, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_ = resp
	// TODO: properly parse it, etc.

	return nil
}

// SendMixMetrics sends the mixnode related packet metrics to the directory server.
func SendMixMetrics(metrics map[string]uint) error {
	jsonValue, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	resp, err := http.Post(config.DirectoryServerMetricsURL, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_ = resp
	// TODO: properly parse it, etc.

	return nil
}

// RegisterMixProviderPresence registers server presence at the directory server.
func RegisterMixProviderPresence(host string, publicKey *sphinx.PublicKey, clients []models.RegisteredClient) error {
	b64Key := base64.StdEncoding.EncodeToString(publicKey.Bytes())
	values := map[string]interface{}{"host": host, "pubKey": b64Key, "registeredClients": clients}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return err
	}

	resp, err := http.Post(config.DirectoryServerMixProviderPresenceURL, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_ = resp
	// TODO: properly parse it, etc.

	return nil
}

func GetNetworkTopology() (*models.Topology, error) {
	resp, err := http.Get(config.DirectoryServerTopology)
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
