package main

import (
	"fmt"
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/nymtech/loopix-messaging/client"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/pki"
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

	privC1 := []byte{207, 106, 72, 12, 133, 115, 162, 78, 69, 11, 244, 117, 100, 109, 32, 28, 181, 195, 113, 116, 241, 129, 181, 123, 90, 89, 244, 56}
	pubC1 := []byte{4, 253, 28, 89, 51, 55, 225, 42, 11, 122, 43, 244, 1, 56, 230, 252, 68, 87, 107, 105, 157, 171, 212, 101, 48, 184, 2, 31, 188, 229, 57, 71, 81, 157, 144, 161, 44, 65, 0, 43, 238, 199, 200, 189, 124, 92, 1, 175, 79, 172, 222, 252, 57, 97, 235, 82, 72}

	privC2 := []byte{251, 207, 106, 200, 172, 109, 158, 158, 180, 55, 158, 231, 96, 234, 134, 137, 242, 4, 181, 170, 11, 20, 251, 4, 158, 107, 242, 173}
	pubC2 := []byte{4, 135, 189, 82, 245, 150, 224, 233, 57, 59, 242, 8, 142, 7, 3, 147, 51, 103, 243, 23, 190, 69, 148, 150, 88, 234, 183, 187, 37, 227, 247, 57, 83, 85, 250, 21, 162, 163, 64, 168, 6, 27, 2, 236, 76, 225, 133, 152, 102, 28, 42, 254, 225, 21, 12, 221, 211}

	var pubC, privC []byte
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

	client, err := client.NewClient(*id, *host, *port, pubC, privC, PkiDir, providerInfo)
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
