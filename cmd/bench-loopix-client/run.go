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
	"github.com/tav/golly/optparse"
)

const (
	// this will be our ingress provider so it needs to be a 'fully functional' one
	defaultBenchmarkProviderID = "Provider"
	defaultBenchmarkClientID   = "BenchmarkClient"
)

// I think here we need to sacrifice the linter error of too long lines for the formatting as it would hideous
// if we split the 'preGenerate' definition line
//nolint: lll
func cmdRun(args []string, usage string) {
	panic("The benchmark client is not yet adjusted to work with the remote directory server")
	// opts := newOpts("run [OPTIONS]", usage)
	// // port := opts.Flags("--port").Label("PORT").String("Port on which loopix-client listens", defaultBenchmarkClientPort)
	// numMessages := opts.Flags("--num").Label("NUMMESSAGES").Int("Number of benchmark messages to send", 0)
	// interval := opts.Flags("--interval").Label("INTERVAL").Duration("Minimum interval between messages to be sent", 0)
	// preGenerate := opts.Flags("--pregenerate").Label("PREGENERATE").Bool("Whether to pregenerate single packet to send it over and over again")

	// params := opts.Parse(args)
	// if len(params) != 0 {
	// 	opts.PrintUsage()
	// 	os.Exit(1)
	// }

	// privC := sphinx.BytesToPrivateKey([]byte{66, 32, 162, 223, 15, 199, 170, 43, 68, 239, 37, 97, 73, 113, 106,
	// 	176, 56, 244, 146, 107, 187, 145, 29, 206, 200, 133, 167, 250, 19, 255, 242, 127})
	// pubC := sphinx.BytesToPublicKey([]byte{202, 54, 182, 74, 58, 128, 66, 117, 198, 114, 255, 254, 100, 155, 20,
	// 	238, 234, 96, 62, 187, 68, 173, 114, 95, 131, 248, 227, 164, 221, 39, 43, 89})

	// cfg, err := clientConfig.DefaultConfig(defaultBenchmarkClientID)
	// if err != nil {
	// 	panic(err)
	// }

	// cfg.Debug.LoopCoverTrafficRate = 0.0
	// cfg.Debug.DropCoverTrafficRate = 0.0
	// cfg.Debug.FetchMessageRate = 0.0
	// cfg.Debug.MessageSendingRate = 10000000.0
	// cfg.Debug.RateCompliantCoverMessagesDisabled = true

	// client, err := client.NewTestClient(cfg, privC, pubC)
	// if err != nil {
	// 	panic(err)
	// }

	// benchClient, err := benchclient.NewBenchClient(client, *numMessages, *interval, *preGenerate)
	// if err != nil {
	// 	panic(err)
	// }

	// if err := benchClient.RunBench(); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Failed to spawn client instance: %v\n", err)
	// 	os.Exit(-1)
	// }
}

func newOpts(command string, usage string) *optparse.Parser {
	return optparse.New("Usage: loopix-client " + command + "\n\n  " + usage + "\n")
}
