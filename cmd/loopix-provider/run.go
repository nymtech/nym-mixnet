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

package main

import (
	"fmt"
	"os"

	"github.com/nymtech/nym-mixnet/constants"
	"github.com/nymtech/nym-mixnet/helpers"
	"github.com/nymtech/nym-mixnet/server/provider"
	"github.com/nymtech/nym-mixnet/sphinx"
	"github.com/tav/golly/optparse"
)

const (
	defaultHost           = ""
	defaultID             = "Provider"
	defaultPort           = "1789"
	defaultPrivateKeyFile = "privateKey.key"
	defaultPublicKeyFile  = "publicKey.key"
)

func loadKeys() (*sphinx.PrivateKey, *sphinx.PublicKey, error) {
	prvKey := new(sphinx.PrivateKey)
	pubKey := new(sphinx.PublicKey)

	if _, err := os.Stat(defaultPrivateKeyFile); os.IsNotExist(err) {
		return nil, nil, err
	}

	if _, err := os.Stat(defaultPublicKeyFile); os.IsNotExist(err) {
		return nil, nil, err
	}

	if err := helpers.FromPEMFile(prvKey, defaultPrivateKeyFile, constants.PrivateKeyPEMType); err != nil {
		return nil, nil, fmt.Errorf("Failed to load the private key: %v", err)
	}

	if err := helpers.FromPEMFile(pubKey, defaultPublicKeyFile, constants.PublicKeyPEMType); err != nil {
		return nil, nil, fmt.Errorf("Failed to load the public key: %v", err)
	}

	fmt.Fprintf(os.Stdout, "Loaded existing keys\n")
	return prvKey, pubKey, nil
}

func saveKeys(privP *sphinx.PrivateKey, pubP *sphinx.PublicKey) {
	if err := helpers.ToPEMFile(privP, defaultPrivateKeyFile, constants.PrivateKeyPEMType); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save private key: %v", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Saved generated private key to %v\n", defaultPrivateKeyFile)

	if err := helpers.ToPEMFile(pubP, defaultPublicKeyFile, constants.PublicKeyPEMType); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save public key: %v", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Saved generated public key to %v\n", defaultPublicKeyFile)
}

func cmdRun(args []string, usage string) {
	opts := newOpts("run [OPTIONS]", usage)
	id := opts.Flags("--id").Label("ID").String("Id of the loopix-provider we want to run", defaultID)
	host := opts.Flags("--host").Label("HOST").String("The host on which the loopix-provider is running", defaultHost)
	port := opts.Flags("--port").Label("PORT").String("Port on which loopix-provider listens", defaultPort)

	params := opts.Parse(args)
	if len(params) != 0 {
		opts.PrintUsage()
		os.Exit(1)
	}

	ip, err := helpers.GetLocalIP()
	if err != nil {
		panic(err)
	}

	if host == nil || len(*host) < 7 {
		host = &ip
	}

	privP, pubP, err := loadKeys()
	if err != nil {
		privP, pubP, err = sphinx.GenerateKeyPair()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to generate new keypair: %v", err)
			os.Exit(1)
		}

		saveKeys(privP, pubP)
	}

	providerServer, err := provider.NewProviderServer(*id, *host, *port, privP, pubP)
	if err != nil {
		panic(err)
	}

	err = providerServer.Start()
	if err != nil {
		panic(err)
	}

	wait := make(chan struct{})
	<-wait
}

func newOpts(command string, usage string) *optparse.Parser {
	return optparse.New("Usage: loopix-provider " + command + "\n\n  " + usage + "\n")
}
