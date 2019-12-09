// Copyright 2019 The Nym Mixnet Authors
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
	"encoding/base64"
	"fmt"
	"os"

	"github.com/nymtech/nym-mixnet/server/provider"
	"github.com/nymtech/nym-mixnet/sphinx"
	"github.com/tav/golly/optparse"
)

const (
	defaultBenchmarkProviderHost = "localhost"
	defaultBenchmarkProviderPort = "11000"
	defaultBenchmarkProviderID   = "BenchmarkProvider"
)

//nolint: lll
func cmdRun(args []string, usage string) {
	opts := newOpts("run [OPTIONS]", usage)
	port := opts.Flags("--port").Label("PORT").String("Port on which nym-mixnet-provider listens", defaultBenchmarkProviderPort)
	numMessages := opts.Flags("--num").Label("NUMMESSAGES").Int("Number of benchmark messages to send", 0)

	params := opts.Parse(args)
	if len(params) != 0 {
		opts.PrintUsage()
		os.Exit(1)
	}

	if err := os.RemoveAll("inboxes/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="); err != nil {
		fmt.Fprintf(os.Stderr, "failed to empty bench inbox: %v", err)
		os.Exit(1)
	}

	// have constant keys to simplify the procedure so that pki/database would not need to be reset every run
	privP := sphinx.BytesToPrivateKey([]byte{191, 43, 90, 175, 50, 224, 156, 22, 204, 173, 87, 255, 64, 152, 17,
		30, 48, 162, 36, 95, 57, 34, 187, 183, 203, 215, 25, 172, 55, 199, 211, 59})
	pubP := sphinx.BytesToPublicKey([]byte{17, 170, 15, 150, 155, 75, 240, 66, 54, 100, 131, 127, 193, 10,
		133, 32, 62, 155, 9, 46, 200, 55, 60, 125, 223, 76, 170, 167, 100, 34, 176, 117})

	baseProviderServer, err := provider.NewProviderServer(defaultBenchmarkProviderID,
		defaultBenchmarkProviderHost,
		*port,
		privP,
		pubP,
	)
	if err != nil {
		panic(err)
	}

	b64Key := base64.URLEncoding.EncodeToString(pubP.Bytes())
	fmt.Println(b64Key)

	benchmarkProviderServer, err := provider.NewBenchProvider(baseProviderServer, *numMessages)
	if err != nil {
		panic(err)
	}

	err = benchmarkProviderServer.RunBench()
	if err != nil {
		panic(err)
	}
}

func newOpts(command string, usage string) *optparse.Parser {
	return optparse.New("Usage: nym-mixnet-provider " + command + "\n\n  " + usage + "\n")
}
