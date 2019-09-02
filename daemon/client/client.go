// client.go - mixnet client daemon.
// Copyright (C) 2019  Jedrzej Stuczynski.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/nymtech/loopix-messaging/client"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/pki"
	"github.com/nymtech/loopix-messaging/sphinx"
	"github.com/nymtech/nym/daemon"
)

const (
	PKI_DIR = "pki/database.db"
)

func main() {
	daemon.Start(func() {
		flag.String("id", "", "Id of the entity we want to run")
		flag.String("host", "", "The host on which the entity is running")
		flag.String("port", "", "The port on which the entity is running")
		flag.String("provider", "", "The port on which the entity is running")
	},
		func() daemon.Service {
			id := flag.Lookup("id").Value.(flag.Getter).Get().(string)
			host := flag.Lookup("host").Value.(flag.Getter).Get().(string)
			port := flag.Lookup("port").Value.(flag.Getter).Get().(string)
			providerId := flag.Lookup("provider").Value.(flag.Getter).Get().(string)

			db, err := pki.OpenDatabase(PKI_DIR, "sqlite3")

			if err != nil {
				panic(err)
			}

			row := db.QueryRow("SELECT Config FROM Pki WHERE Id = ? AND Typ = ?", providerId, "Provider")

			var results []byte
			err = row.Scan(&results)
			if err != nil {
				fmt.Println(err)
			}
			var providerInfo config.MixConfig
			err = proto.Unmarshal(results, &providerInfo)

			pubC, privC, err := sphinx.GenerateKeyPair()
			if err != nil {
				panic(err)
			}

			client, err := client.NewClient(id, host, port, pubC, privC, PKI_DIR, providerInfo)
			if err != nil {
				panic(err)
			}

			err = client.Start()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to spawn client instance: %v\n", err)
				os.Exit(-1)
			}

			return client
		})
}
