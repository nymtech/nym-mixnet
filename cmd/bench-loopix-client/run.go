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

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/loopix-messaging/client"
	"github.com/nymtech/loopix-messaging/client/benchclient"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/pki"
	"github.com/nymtech/loopix-messaging/sphinx"
	"github.com/tav/golly/optparse"
)

const (
	// PkiDir is the location of the database file, relative to the project root. TODO: move this to homedir.
	PkiDir                     = "pki/database.db"
	defaultBenchmarkClientHost = "localhost"
	defaultBenchmarkClientID   = "BenchmarkClient"
	defaultBenchmarkClientPort = "10000"
	// this will be our ingress provider so it needs to be a 'fully functiona' one
	defaultBenchmarkProviderID = "Provider"
)

// I think here we need to sacrifice the linter error of too long lines for the formatting as it would hideous
// if we split the 'preGenerate' definition line
//nolint: lll
func cmdRun(args []string, usage string) {
	opts := newOpts("run [OPTIONS]", usage)
	port := opts.Flags("--port").Label("PORT").String("Port on which loopix-client listens", defaultBenchmarkClientPort)
	numMessages := opts.Flags("--num").Label("NUMMESSAGES").Int("Number of benchmark messages to send", 0)
	interval := opts.Flags("--interval").Label("INTERVAL").Duration("Minimum interval between messages to be sent", 0)
	preGenerate := opts.Flags("--pregenerate").Label("PREGENERATE").Bool("Whether to pregenerate single packet to send it over and over again")

	params := opts.Parse(args)
	if len(params) != 0 {
		opts.PrintUsage()
		os.Exit(1)
	}

	db, err := pki.OpenDatabase(PkiDir, "sqlite3")
	if err != nil {
		panic(err)
	}
	row := db.QueryRow("SELECT Config FROM Pki WHERE Id = ? AND Typ = ?", defaultBenchmarkProviderID, "Provider")

	var results []byte
	if err := row.Scan(&results); err != nil {
		panic(err)
	}
	var providerInfo config.MixConfig
	if err := proto.Unmarshal(results, &providerInfo); err != nil {
		panic(err)
	}

	privC := sphinx.BytesToPrivateKey([]byte{66, 32, 162, 223, 15, 199, 170, 43, 68, 239, 37, 97, 73, 113, 106,
		176, 56, 244, 146, 107, 187, 145, 29, 206, 200, 133, 167, 250, 19, 255, 242, 127})
	pubC := sphinx.BytesToPublicKey([]byte{202, 54, 182, 74, 58, 128, 66, 117, 198, 114, 255, 254, 100, 155, 20,
		238, 234, 96, 62, 187, 68, 173, 114, 95, 131, 248, 227, 164, 221, 39, 43, 89})

	client, err := client.NewClient(defaultBenchmarkClientID,
		defaultBenchmarkClientHost,
		*port,
		privC,
		pubC,
		PkiDir,
		providerInfo,
		config.ClientConfig{},
	)
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
	return optparse.New("Usage: loopix-client " + command + "\n\n  " + usage + "\n")
}
