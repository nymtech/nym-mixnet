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

package provider

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/nym-mixnet/config"
	"github.com/nymtech/nym-mixnet/helpers"
	"github.com/nymtech/nym-mixnet/server/mixnode"
	"github.com/nymtech/nym-mixnet/sphinx"
	"github.com/stretchr/testify/assert"
)

//nolint: gochecknoglobals
var (
	mixServer      *mixnode.MixServer
	providerServer *ProviderServer
)

func TestMain(m *testing.M) {
	var err error
	mixServer, err = mixnode.CreateTestMixnode()
	if err != nil {
		fmt.Println(err)
		panic(m)
	}

	providerServer, err = CreateTestProvider()
	if err != nil {
		fmt.Println(err)
		panic(m)
	}

	code := m.Run()
	clean()
	os.Exit(code)
}

func clean() {
	os.RemoveAll("./inboxes")
}

func createFakeClientListener(host, port string) (*net.TCPListener, error) {
	addr, err := helpers.ResolveTCPAddress(host, port)
	if err != nil {
		return nil, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	return listener, nil
}

func TestProviderServer_AuthenticateUser_Pass(t *testing.T) {
	key := []byte{1, 2, 3, 4, 5}
	testToken := []byte("AuthenticationToken")
	record := ClientRecord{id: "Alice", host: "localhost", port: "1111", pubKey: key, token: testToken}
	b64Key := base64.URLEncoding.EncodeToString(key)
	providerServer.assignedClients[b64Key] = record
	assert.True(t,
		providerServer.authenticateUser(key, []byte("AuthenticationToken")),
		" Authentication should be successful",
	)
}

func TestProviderServer_AuthenticateUser_Fail(t *testing.T) {
	key := []byte{1, 2, 3, 4, 5}
	record := ClientRecord{id: "Alice", host: "localhost", port: "1111", pubKey: key, token: []byte("AuthenticationToken")}
	b64Key := base64.URLEncoding.EncodeToString(key)
	providerServer.assignedClients[b64Key] = record
	assert.False(t,
		providerServer.authenticateUser(key, []byte("WrongAuthToken")),
		" Authentication should not be successful",
	)
}

func createInbox(id string, t *testing.T) {
	path := filepath.Join("./inboxes", id)
	exists, err := helpers.DirExists(path)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		os.RemoveAll(path)
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
	} else if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func createTestMessage(id string, t *testing.T) {
	file, err := os.Create(filepath.Join("./inboxes", id, "TestMessage.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	_, err = file.Write([]byte("This is a test message"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestProviderServer_StoreMessage(t *testing.T) {

	inboxID := "ClientInbox"
	fileID := "12345"
	inboxDir := "./inboxes/" + inboxID
	filePath := inboxDir + "/" + fileID + ".txt"

	err := os.MkdirAll(inboxDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	message := []byte("Hello world message")
	if err := providerServer.storeMessage(message, inboxID, fileID); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filePath); err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err, "The file with the message should be created")

	dat, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, message, dat, "Messages should be the same")

}

func createTestPacket(t *testing.T) *sphinx.SphinxPacket {
	path := config.E2EPath{IngressProvider: providerServer.config,
		Mixes:          []config.MixConfig{mixServer.GetConfig()},
		EgressProvider: providerServer.config,
	}
	sphinxPacket, err := sphinx.PackForwardMessage(path, []float64{0.1, 0.2, 0.3}, []byte("Hello world"))
	if err != nil {
		t.Fatal(err)
		return nil
	}
	return &sphinxPacket
}

func TestProviderServer_ReceivedPacket(t *testing.T) {
	sphinxPacket := createTestPacket(t)
	bSphinxPacket, err := proto.Marshal(sphinxPacket)
	if err != nil {
		t.Fatal(err)
	}
	err = providerServer.receivedPacket(bSphinxPacket)
	if err != nil {
		t.Fatal(err)
	}
}
