package main

import (
	"fmt"
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/nymtech/loopix-messaging/client"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/pki"
	"github.com/nymtech/loopix-messaging/sphinx"
	"github.com/tav/golly/optparse"
)

const (
	// PkiDir is the location of the database file, relative to the project root. TODO: move this to homedir.
	PkiDir            = "pki/database.db"
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

	params := opts.Parse(args)
	if len(params) != 0 {
		opts.PrintUsage()
		os.Exit(1)
	}

	db, err := pki.OpenDatabase(PkiDir, "sqlite3")
	if err != nil {
		panic(err)
	}
	row := db.QueryRow("SELECT Config FROM Pki WHERE Id = ? AND Typ = ?", providerID, "Provider")

	var results []byte
	err = row.Scan(&results)
	if err != nil {
		fmt.Println(err)
	}
	var providerInfo config.MixConfig
	if err := proto.Unmarshal(results, &providerInfo); err != nil {
		panic(err)
	}

	privC1 := sphinx.BytesToPrivateKey([]byte{66, 32, 162, 223, 15, 199, 170, 43, 68, 239, 37, 97, 73, 113, 106, 176, 56, 244, 146, 107, 187, 145, 29, 206, 200, 133, 167, 250, 19, 255, 242, 127})
	pubC1 := sphinx.BytesToPublicKey([]byte{202, 54, 182, 74, 58, 128, 66, 117, 198, 114, 255, 254, 100, 155, 20, 238, 234, 96, 62, 187, 68, 173, 114, 95, 131, 248, 227, 164, 221, 39, 43, 89})

	privC2 := sphinx.BytesToPrivateKey([]byte{51, 206, 63, 231, 196, 148, 31, 110, 183, 209, 1, 16, 184, 47, 238, 103, 127, 213, 81, 180, 56, 178, 84, 45, 30, 196, 22, 51, 3, 108, 175, 87})
	pubC2 := sphinx.BytesToPublicKey([]byte{21, 103, 130, 37, 105, 58, 162, 113, 91, 198, 76, 156, 194, 36, 45, 219, 121, 158, 255, 247, 44, 159, 243, 155, 215, 90, 67, 103, 64, 242, 95, 45})

	var privC *sphinx.PrivateKey
	var pubC *sphinx.PublicKey

	switch *id {
	case "Client1":
		pubC = pubC1
		privC = privC1
	case "Client2":
		pubC = pubC2
		privC = privC2
	default:
		fmt.Fprintf(os.Stderr, "Unknown client instance: %v\n", *id)
		os.Exit(-1)
	}

	client, err := client.NewClient(*id, *host, *port, privC, pubC, PkiDir, providerInfo)
	if err != nil {
		panic(err)
	}

	err = client.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to spawn client instance: %v\n", err)
		os.Exit(-1)
	}

	wait := make(chan struct{})
	<-wait
}

func newOpts(command string, usage string) *optparse.Parser {
	return optparse.New("Usage: loopix-client " + command + "\n\n  " + usage + "\n")
}
