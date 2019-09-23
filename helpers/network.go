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
func RegisterMixNodePresence(publicKey *sphinx.PublicKey, layer int, host ...string) error {
	b64Key := base64.StdEncoding.EncodeToString(publicKey.Bytes())
	values := map[string]interface{}{"pubKey": b64Key, "layer": layer}
	if len(host) == 1 {
		values["host"] = host[0]
	}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return err
	}

	endpoint := config.DirectoryServerMixPresenceURL
	if len(host) == 1 {
		ip, _, err := net.SplitHostPort(host[0])
		if err == nil && (ip == "localhost" || net.ParseIP(ip).IsLoopback()) {
			endpoint = config.LocalDirectoryServerMixPresenceURL
		} else if err.Error() == "missing port in address" &&
			(host[0] == "localhost" || net.ParseIP(host[0]).IsLoopback()) {
			endpoint = config.LocalDirectoryServerMixPresenceURL
		}
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_ = resp
	// TODO: properly parse it, etc.

	// body, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	return err
	// }
	// fmt.Println(string(body))
	// _ = resp
	return nil
}

// SendMixMetrics sends the mixnode related packet metrics to the directory server.
func SendMixMetrics(metric models.MixMetric, host ...string) error {
	values := map[string]interface{}{"sent": metric.Sent, "pubKey": metric.PubKey, "received": metric.Received}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return err
	}

	endpoint := config.DirectoryServerMetricsURL
	if len(host) == 1 {
		ip, _, err := net.SplitHostPort(host[0])
		if err == nil && (ip == "localhost" || net.ParseIP(ip).IsLoopback()) {
			endpoint = config.LocalDirectoryServerMetricsURL
		} else if err.Error() == "missing port in address" &&
			(host[0] == "localhost" || net.ParseIP(host[0]).IsLoopback()) {
			endpoint = config.LocalDirectoryServerMetricsURL
		}
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// body, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	return err
	// }
	// fmt.Println(string(body))
	// _ = resp
	// TODO: properly parse it, etc.

	return nil
}

// RegisterMixProviderPresence registers server presence at the directory server.
func RegisterMixProviderPresence(publicKey *sphinx.PublicKey, clients []models.RegisteredClient, host ...string) error {
	b64Key := base64.StdEncoding.EncodeToString(publicKey.Bytes())
	values := map[string]interface{}{"pubKey": b64Key, "registeredClients": clients}
	if len(host) == 1 {
		values["host"] = host[0]
	}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return err
	}

	endpoint := config.DirectoryServerMixProviderPresenceURL
	if len(host) == 1 {
		ip, _, err := net.SplitHostPort(host[0])
		if err == nil && (ip == "localhost" || net.ParseIP(ip).IsLoopback()) {
			endpoint = config.LocalDirectoryServerMixProviderPresenceURL
		} else if err.Error() == "missing port in address" &&
			(host[0] == "localhost" || net.ParseIP(host[0]).IsLoopback()) {
			endpoint = config.LocalDirectoryServerMixProviderPresenceURL
		}
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_ = resp
	// TODO: properly parse it, etc.

	return nil
}
