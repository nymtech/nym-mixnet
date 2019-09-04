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

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/loopix-messaging/client"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/daemon"
	"github.com/nymtech/loopix-messaging/pki"
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
			if err := proto.Unmarshal(results, &providerInfo); err != nil {
				panic(err)
			}

			// pubC, privC, err := sphinx.GenerateKeyPair()
			// if err != nil {
			// 	panic(err)
			// }

			privC1 := []byte{207, 106, 72, 12, 133, 115, 162, 78, 69, 11, 244, 117, 100, 109, 32, 28, 181, 195, 113, 116, 241, 129, 181, 123, 90, 89, 244, 56}
			pubC1 := []byte{4, 253, 28, 89, 51, 55, 225, 42, 11, 122, 43, 244, 1, 56, 230, 252, 68, 87, 107, 105, 157, 171, 212, 101, 48, 184, 2, 31, 188, 229, 57, 71, 81, 157, 144, 161, 44, 65, 0, 43, 238, 199, 200, 189, 124, 92, 1, 175, 79, 172, 222, 252, 57, 97, 235, 82, 72}

			privC2 := []byte{251, 207, 106, 200, 172, 109, 158, 158, 180, 55, 158, 231, 96, 234, 134, 137, 242, 4, 181, 170, 11, 20, 251, 4, 158, 107, 242, 173}
			pubC2 := []byte{4, 135, 189, 82, 245, 150, 224, 233, 57, 59, 242, 8, 142, 7, 3, 147, 51, 103, 243, 23, 190, 69, 148, 150, 88, 234, 183, 187, 37, 227, 247, 57, 83, 85, 250, 21, 162, 163, 64, 168, 6, 27, 2, 236, 76, 225, 133, 152, 102, 28, 42, 254, 225, 21, 12, 221, 211}

			var pubC, privC []byte
			switch id {
			case "Client1":
				pubC = pubC1
				privC = privC1
			case "Client2":
				pubC = pubC2
				privC = privC2
			default:
				fmt.Fprintf(os.Stderr, "Unknown client instance: %v\n", id)
				os.Exit(-1)
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
