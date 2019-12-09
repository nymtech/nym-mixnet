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

	"github.com/nymtech/nym-mixnet/client"
	"github.com/nymtech/nym-mixnet/client/benchclient"
	clientConfig "github.com/nymtech/nym-mixnet/client/config"
	"github.com/nymtech/nym-mixnet/helpers/topology"
	"github.com/nymtech/nym-mixnet/sphinx"
	"github.com/tav/golly/optparse"
)

const (
	defaultBenchmarkClientID = "BenchmarkClient"
	benchmarkProviderID      = "EaoPlptL8EI2ZIN_wQqFID6bCS7INzx930yqp2QisHU="
)

// I think here we need to sacrifice the linter error of too long lines for the formatting as it would hideous
// if we split the 'preGenerate' definition line
//nolint: lll
func cmdRun(args []string, usage string) {
	opts := newOpts("run [OPTIONS]", usage)
	numMessages := opts.Flags("--num").Label("NUMMESSAGES").Int("Number of benchmark messages to send", 0)
	interval := opts.Flags("--interval").Label("INTERVAL").Duration("Minimum interval between messages to be sent", 0)
	preGenerate := opts.Flags("--pregenerate").Label("PREGENERATE").Bool("Whether to pregenerate single packet to send it over and over again")

	params := opts.Parse(args)
	if len(params) != 0 {
		opts.PrintUsage()
		os.Exit(1)
	}

	// actually make the benchmark client's keys something obviously invalid
	privC := sphinx.BytesToPrivateKey([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0})
	pubC := sphinx.BytesToPublicKey([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0})

	cfg, err := clientConfig.DefaultConfig(defaultBenchmarkClientID)
	if err != nil {
		panic(err)
	}

	cfg.Logging.Disable = true
	cfg.Debug.LoopCoverTrafficRate = 0.0
	cfg.Debug.FetchMessageRate = 0.0
	cfg.Debug.MessageSendingRate = 10000000.0
	cfg.Debug.RateCompliantCoverMessagesDisabled = true
	cfg.Client.DirectoryServerTopologyEndpoint = clientConfig.DefaultLocalDirectoryServerTopologyEndpoint

	// get an Ingress provider that IS NOT the benchmark provider
	initialTopology, err := topology.GetNetworkTopology(cfg.Client.DirectoryServerTopologyEndpoint)
	if err != nil || len(initialTopology.MixProviderNodes) == 0 {
		fmt.Fprintf(os.Stderr, "failed to obtain network topology: %v", err)
		os.Exit(1)
	}

	for _, node := range initialTopology.MixProviderNodes {
		if node.PubKey != benchmarkProviderID {
			cfg.Client.ProviderID = node.PubKey
			break
		}
	}

	client, err := client.NewTestClient(cfg, privC, pubC)
	if err != nil {
		panic(err)
	}

	benchClient, err := benchclient.NewBenchClient(client, *numMessages, *interval, *preGenerate)
	if err != nil {
		panic(err)
	}

	if err := benchClient.RunBench(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to spawn client instance: %v\n", err)
		os.Exit(-1)
	}
}

func newOpts(command string, usage string) *optparse.Parser {
	return optparse.New("Usage: nym-mixnet-client " + command + "\n\n  " + usage + "\n")
}
