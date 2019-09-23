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

package commands

import (
	"fmt"
	"os"
	"path/filepath"

	clientConfig "github.com/nymtech/loopix-messaging/client/config"
	"github.com/nymtech/loopix-messaging/constants"
	"github.com/nymtech/loopix-messaging/helpers"
	"github.com/nymtech/loopix-messaging/helpers/topology"
	"github.com/nymtech/loopix-messaging/sphinx"
)

func InitCmd(args []string, usage string) {
	opts := newOpts("run [OPTIONS]", usage)
	id := opts.Flags("--id").Label("ID").String("Id of the loopix-client we want to create config for", "")
	providerID := opts.Flags("--provider").Label("PROVIDER").String("Id of the provider we have preference "+
		"to connect to. If left empty, a random provider will be chosen", "")
	local := opts.Flags("--local").Label("LOCAL").Bool("Flag to indicate whether the client is expected " +
		"to run on the local mixnet deployment")

	params := opts.Parse(args)
	if len(params) != 0 {
		opts.PrintUsage()
		os.Exit(1)
	}

	priv, pub, err := sphinx.GenerateKeyPair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate sphinx keypair: %v", err)
		os.Exit(1)
	}

	var clientID string
	if len(*id) == 0 {
		randomID := helpers.RandomString(8)
		fmt.Fprintf(os.Stdout, "No clientID provided. Random string will be used instead: %v.\n", randomID)
		clientID = randomID
	} else {
		clientID = *id
	}

	defaultCfg, err := clientConfig.DefaultConfig(clientID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create config: %v", err)
		os.Exit(1)
	}

	defaultCfg.Client.ProviderID = *providerID
	if *local {
		fmt.Fprintf(os.Stdout, "Using the local directory server")
		defaultCfg.Client.DirectoryServerTopologyEndpoint = clientConfig.DefaultLocalDirectoryServerTopologyEndpoint
	}

	configPath, err := clientConfig.DefaultConfigPath(clientID)
	if err != nil {
		// This should have never been thrown but as with all the cases of errors that should have never been thrown
		// let's do check for it...
		fmt.Fprintf(os.Stderr, "congratulations! You've managed to reach an impossible execution path "+
			"while getting default config path for %v: %v", clientID, err)
		os.Exit(1)
	}

	configDir, _ := filepath.Split(configPath)
	if err := helpers.EnsureDir(configDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create client directory: %v", err)
		os.Exit(1)
	}

	if err := helpers.ToPEMFile(priv, defaultCfg.Client.PrivateKeyFile(), constants.PrivateKeyPEMType); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save private key: %v", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Saved generated private key to %v\n", defaultCfg.Client.PrivateKeyFile())

	if err := helpers.ToPEMFile(pub, defaultCfg.Client.PublicKeyFile(), constants.PublicKeyPEMType); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save public key: %v", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Saved generated public key to %v\n", defaultCfg.Client.PublicKeyFile())

	// if we haven't specified a provider, let's try to obtain one now
	initialTopology, err := topology.GetNetworkTopology()
	if err != nil || len(initialTopology.MixProviderNodes) == 0 {
		fmt.Fprintf(os.Stderr, "failed to obtain network topology: %v", err)
		os.Exit(1)
	}

	// iterating through map is not deterministic so in theory multiple clients should be getting
	// different providers
	for provID := range initialTopology.MixProviderNodes {
		// get the first entry
		defaultCfg.Client.ProviderID = provID
		break
	}

	// finally write our config to a file
	if err := clientConfig.WriteConfigFile(configPath, defaultCfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write config to a file: %v", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Saved generated config to %v\n", configPath)
}
