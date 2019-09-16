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
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/pki"
	"github.com/nymtech/loopix-messaging/sphinx"
	"github.com/tav/golly/optparse"
)

const (
	// PkiDb is the location of the database file, relative to the project root. TODO: move this to homedir.
	PkiDb             = "pki/database.db"
	defaultHost       = "localhost"
	defaultID         = "Client1"
	defaultPort       = "9999"
	defaultProviderID = "Provider"
)

func cmdRun(args []string, usage string) {
	opts := newOpts("run [OPTIONS]", usage)
	id := opts.Flags("--id").Label("ID").String("Id of the loopix-client we want to run", defaultID)
	host := opts.Flags("--host").Label("HOST").String("The host on which the loopix-client is running", defaultHost)
	port := opts.Flags("--port").Label("PORT").String("Port on which loopix-client listens", defaultPort)
	providerID := opts.Flags("--provider").Label("PROVIDER").String("Id of the provider to connect to", defaultProviderID)
	demo := opts.Flags("--demo").Label("DEMO").Bool("Should the client be run in demo mode")

	params := opts.Parse(args)
	if len(params) != 0 {
		opts.PrintUsage()
		os.Exit(1)
	}

	db, err := pki.OpenDatabase(PkiDb, "sqlite3")
	if err != nil {
		panic(err)
	}
	row := db.QueryRow("SELECT Config FROM Pki WHERE Id = ? AND Typ = ?", providerID, "Provider")

	var results []byte
	if err := row.Scan(&results); err != nil {
		panic(err)
	}
	var providerInfo config.MixConfig
	if err := proto.Unmarshal(results, &providerInfo); err != nil {
		panic(err)
	}

	privC1 := sphinx.BytesToPrivateKey([]byte{66, 32, 162, 223, 15, 199, 170, 43, 68, 239, 37, 97, 73, 113, 106,
		176, 56, 244, 146, 107, 187, 145, 29, 206, 200, 133, 167, 250, 19, 255, 242, 127})
	pubC1 := sphinx.BytesToPublicKey([]byte{202, 54, 182, 74, 58, 128, 66, 117, 198, 114, 255, 254, 100, 155, 20,
		238, 234, 96, 62, 187, 68, 173, 114, 95, 131, 248, 227, 164, 221, 39, 43, 89})

	privC2 := sphinx.BytesToPrivateKey([]byte{51, 206, 63, 231, 196, 148, 31, 110, 183, 209, 1, 16, 184, 47, 238,
		103, 127, 213, 81, 180, 56, 178, 84, 45, 30, 196, 22, 51, 3, 108, 175, 87})
	pubC2 := sphinx.BytesToPublicKey([]byte{21, 103, 130, 37, 105, 58, 162, 113, 91, 198, 76, 156, 194, 36, 45,
		219, 121, 158, 255, 247, 44, 159, 243, 155, 215, 90, 67, 103, 64, 242, 95, 45})

	var privC *sphinx.PrivateKey
	var pubC *sphinx.PublicKey
	var demoRecipient config.ClientConfig

	switch *id {
	case "Client1":
		pubC = pubC1
		privC = privC1
		if *demo == true {
			demoRecipient = config.ClientConfig{
				Id:       "Client2",
				Host:     "localhost",
				Port:     "9998",
				PubKey:   pubC2.Bytes(),
				Provider: &providerInfo,
			}
		}
	case "Client2":
		pubC = pubC2
		privC = privC2
		if *demo {
			demoRecipient = config.ClientConfig{
				Id:       "Client1",
				Host:     "localhost",
				Port:     "9999",
				PubKey:   pubC1.Bytes(),
				Provider: &providerInfo,
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown client instance: %v\n", *id)
		os.Exit(-1)
	}

	client, err := client.NewClient(*id, *host, *port, privC, pubC, PkiDb, providerInfo, demoRecipient)
	if err != nil {
		panic(err)
	}

	if err := client.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to spawn client instance: %v\n", err)
		os.Exit(-1)
	}

	if *id == "Client1" {
		client.ChangeLoggingLevel("Info")
	}

	wait := make(chan struct{})
	<-wait
}

func newOpts(command string, usage string) *optparse.Parser {
	return optparse.New("Usage: loopix-client " + command + "\n\n  " + usage + "\n")
}
