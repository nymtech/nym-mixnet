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
	"os"

	"github.com/nymtech/loopix-messaging/helpers"
	"github.com/nymtech/loopix-messaging/server/mixnode"
	"github.com/nymtech/loopix-messaging/sphinx"
	"github.com/tav/golly/optparse"
)

const (
	// PkiDb is the location of the database file, relative to the project root. TODO: move this to homedir.
	PkiDb        = "pki/database.db"
	defaultHost  = "localhost"
	defaultID    = "Mix1"
	defaultPort  = "6666"
	defaultLayer = -1
)

func cmdRun(args []string, usage string) {
	opts := newOpts("run [OPTIONS]", usage)
	id := opts.Flags("--id").Label("ID").String("Id of the loopix-client we want to run", defaultID)
	host := opts.Flags("--host").Label("HOST").String("The host on which the loopix-client is running", defaultHost)
	port := opts.Flags("--port").Label("PORT").String("Port on which loopix-client listens", defaultPort)
	layer := opts.Flags("--layer").Label("Layer").Int("Mixnet layer of this particular node", defaultLayer)

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

	pubM, privM, err := sphinx.GenerateKeyPair()
	if err != nil {
		panic(err)
	}

	mixServer, err := mixnode.NewMixServer(*id, *host, *port, pubM, privM, *layer)
	if err != nil {
		panic(err)
	}

	if err := mixServer.Start(); err != nil {
		panic(err)
	}

	mixServer.Wait()
}

func newOpts(command string, usage string) *optparse.Parser {
	return optparse.New("Usage: loopix-mixnode " + command + "\n\n  " + usage + "\n")
}
