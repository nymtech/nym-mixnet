// Copyright 2018 The Loopix-Messaging Authors
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

package client

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/jmoiron/sqlx"
	"github.com/nymtech/loopix-messaging/config"
	sphinx "github.com/nymtech/loopix-messaging/sphinx"
	"github.com/stretchr/testify/assert"
)

const (
	pkiDir = "testDatabase.db"
)

// I guess in the case of a test file, globals are fine
//nolint: gochecknoglobals
var (
	providerPubs config.MixConfig
	testMixSet   []config.MixConfig
)

func setupTestDatabase() (*sqlx.DB, error) {

	db, err := sqlx.Connect("sqlite3", pkiDir)
	if err != nil {
		return nil, err
	}

	query := `CREATE TABLE Pki (
		idx INTEGER PRIMARY KEY,
    	Id TEXT,
    	Typ TEXT,
    	Config BLOB);`

	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return db, err
}

func SetupTestMixesInDatabase(t *testing.T) error {
	if err := clean(); err != nil {
		t.Fatal(err)
	}

	db, err := setupTestDatabase()
	assert.Nil(t, err)

	insertQuery := `INSERT INTO Pki (Id, Typ, Config) VALUES (?, ?, ?)`

	for i := 0; i < 10; i++ {
		_, pub, err := sphinx.GenerateKeyPair()
		assert.Nil(t, err)

		m := config.MixConfig{Id: fmt.Sprintf("Mix%d", i),
			Host:   "localhost",
			Port:   strconv.Itoa(9980 + i),
			PubKey: pub.Bytes()}
		mBytes, err := proto.Marshal(&m)
		assert.Nil(t, err)

		_, err = db.Exec(insertQuery, m.Id, "Mix", mBytes)
		assert.Nil(t, err)

		testMixSet = append(testMixSet, m)
	}
	return nil
}

//nolint: unused
func SetupTestClientsInDatabase(t *testing.T) {
	if err := clean(); err != nil {
		t.Fatal(err)
	}

	db, err := setupTestDatabase()
	assert.Nil(t, err)

	insertQuery := `INSERT INTO Pki (Id, Typ, Config) VALUES (?, ?, ?)`

	for i := 0; i < 10; i++ {
		_, pub, err := sphinx.GenerateKeyPair()
		if err != nil {
			t.Fatal(err)
		}
		c := config.ClientConfig{Id: fmt.Sprintf("Client%d", i),
			Host:   "localhost",
			Port:   strconv.Itoa(9980 + i),
			PubKey: pub.Bytes()}
		cBytes, err := proto.Marshal(&c)
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(insertQuery, c.Id, "Client", cBytes)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func SetupTestClient(t *testing.T) *NetClient {
	_, pubP, err := sphinx.GenerateKeyPair()
	assert.Nil(t, err)
	providerPubs = config.MixConfig{Id: "Provider", Host: "localhost", Port: "9995", PubKey: pubP.Bytes()}

	privC, pubC, err := sphinx.GenerateKeyPair()
	assert.Nil(t, err)
	client, err := NewTestClient("Client", "localhost", "3332", privC, pubC, pkiDir, providerPubs)
	assert.Nil(t, err)

	return client
}

func clean() error {
	if _, err := os.Stat(pkiDir); err == nil {
		err := os.Remove(pkiDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestMain(m *testing.M) {

	defer func() {
		if err := clean(); err != nil {
			os.Exit(-1)
		}
	}()

	code := m.Run()
	if err := clean(); err != nil {
		os.Exit(-1)
	}

	os.Exit(code)

}

func TestClient_GetMessagesFromProvider(t *testing.T) {

}

// TODO: Fix this test
//func TestClient_RegisterToken_Pass(t *testing.T) {
//client := SetupTestClient(t)
//client.RegisterToken([]byte("TestToken"))
//r := <- client.registrationDone
//assert.True(t, r)
//assert.Equal(t, []byte("TestToken"), client.token, "Client should register only given token")
//}

//func TestClient_RegisterToken_Fail(t *testing.T) {
//	client := SetupTestClient(t)
//	client.RegisterToken([]byte("TestToken"))
//	assert.NotEqual(t, []byte("WrongToken"), client.token, "Client should register only the given token")
//}

// TODO: Fix this test
func TestClient_RegisterToProvider(t *testing.T) {

}

// TODO: Fix this test
//func TestClient_SendMessage(t *testing.T) {
//	pubP, _, err := sphinx.GenerateKeyPair()
//	if err != nil{
//		t.Fatal(err)
//	}
//	providerPubs = config.MixConfig{Id: "Provider", Host: "localhost", Port: "9995", PubKey: pubP}
//
//	pubR, _, err := sphinx.GenerateKeyPair()
//	if err != nil{
//		t.Fatal(err)
//	}
// recipient := config.ClientConfig{Id:"Recipient",
// 	Host:"localhost",
// 	Port:"9999",
// 	PubKey: pubR,
// 	Provider: &providerPubs,
// }
//	fmt.Println(recipient)
//	pubM1, _, err := sphinx.GenerateKeyPair()
//	if err != nil{
//		t.Fatal(err)
//	}
//	pubM2, _, err := sphinx.GenerateKeyPair()
//	if err != nil{
//		t.Fatal(err)
//	}
//	m1 := config.MixConfig{Id: "Mix1", Host: "localhost", Port: strconv.Itoa(9980), PubKey: pubM1}
//	m2 := config.MixConfig{Id: "Mix2", Host: "localhost", Port: strconv.Itoa(9981), PubKey: pubM2}
//
//	client := SetupTestClient(t)
//	client.ActiveMixes = []config.MixConfig{m1, m2}
//
//	addr, err := helpers.ResolveTCPAddress(client.Host, client.Port)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	client.Listener, err = net.ListenTCP("tcp", addr)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	err = client.SendMessage("TestMessage", recipient)
//	if err != nil{
//		t.Fatal(err)
//	}
//	err = client.Listener.Close()
//	if err != nil{
//		t.Fatal(err)
//	}
//}

// TODO: Fix this test
func TestClient_ProcessPacket(t *testing.T) {

}

func TestClient_ReadInMixnetPKI(t *testing.T) {
	if err := clean(); err != nil {
		t.Fatal(err)
	}
	if err := SetupTestMixesInDatabase(t); err != nil {
		t.Fatal(err)
	}

	client := SetupTestClient(t)
	err := client.ReadInNetworkFromPKI("testDatabase.db")
	assert.Nil(t, err)

	assert.Equal(t, len(testMixSet), len(client.Network.Mixes))
	for i := range testMixSet {
		assert.True(t, proto.Equal(&testMixSet[i], &client.Network.Mixes[i]))
	}
}
