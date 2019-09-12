// Copyright 2018-2019 The Loopix-Messaging Authors
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

/*
	Package helpers implements all useful functions which are used in the code of anonymous messaging system.
*/

package helpers

import (
	"github.com/golang/protobuf/proto"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/pki"
)

///////////////////////////////
// The below will be removed once we get rid of .db file with pki
///////////////////////////////

// AddToDatabase adds a record to the PKI database into a given table.
func AddToDatabase(pkiPath string, tableName, id, typ string, config []byte) error {
	db, err := pki.OpenDatabase(pkiPath, "sqlite3")
	if err != nil {
		return err
	}
	defer db.Close()

	err = pki.InsertIntoTable(db, tableName, id, typ, config)
	if err != nil {
		return err
	}
	return nil
}

// GetMixesPKI returns PKI data for mix nodes.
func GetMixesPKI(pkiDir string) ([]config.MixConfig, error) {
	var mixes []config.MixConfig

	db, err := pki.OpenDatabase(pkiDir, "sqlite3")
	if err != nil {
		return nil, err
	}

	recordsMixes, err := pki.QueryDatabase(db, "Pki", "Mix")
	if err != nil {
		return nil, err
	}

	for recordsMixes.Next() {
		result := make(map[string]interface{})
		err := recordsMixes.MapScan(result)
		if err != nil {
			return nil, err
		}

		var mixConfig config.MixConfig
		err = proto.Unmarshal(result["Config"].([]byte), &mixConfig)
		if err != nil {
			return nil, err
		}
		mixes = append(mixes, mixConfig)
	}

	return mixes, nil
}

// GetClientPKI returns a map of the current client PKI from the PKI database
func GetClientPKI(pkiDir string) ([]config.ClientConfig, error) {
	var clients []config.ClientConfig

	db, err := pki.OpenDatabase(pkiDir, "sqlite3")
	if err != nil {
		return nil, err
	}

	recordsClients, err := pki.QueryDatabase(db, "Pki", "Client")
	if err != nil {
		return nil, err
	}
	for recordsClients.Next() {
		result := make(map[string]interface{})
		err := recordsClients.MapScan(result)

		if err != nil {
			return nil, err
		}

		var clientConfig config.ClientConfig
		err = proto.Unmarshal(result["Config"].([]byte), &clientConfig)
		if err != nil {
			return nil, err
		}

		clients = append(clients, clientConfig)
	}
	return clients, nil
}
