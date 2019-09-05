package main

import (
	"fmt"
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/nymtech/loopix-messaging/client"
	"github.com/nymtech/loopix-messaging/client/benchclient"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/pki"
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

func cmdRun(args []string, usage string) {
	opts := newOpts("run [OPTIONS]", usage)
	port := opts.Flags("--port").Label("PORT").String("Port on which loopix-client listens", defaultBenchmarkClientPort)
	numMessages := opts.Flags("--num").Label("NUMMESSAGES").Int("Number of benchmark messages to send", 0)
	interval := opts.Flags("--interval").Label("INTERVAL").Duration("Minimum interval between messages to be sent", 0)

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
	err = row.Scan(&results)
	if err != nil {
		fmt.Println(err)
	}
	var providerInfo config.MixConfig
	if err := proto.Unmarshal(results, &providerInfo); err != nil {
		panic(err)
	}

	privC := []byte{207, 106, 72, 12, 133, 115, 162, 78, 69, 11, 244, 117, 100, 109, 32, 28, 181, 195, 113, 116, 241, 129, 181, 123, 90, 89, 244, 56}
	pubC := []byte{4, 253, 28, 89, 51, 55, 225, 42, 11, 122, 43, 244, 1, 56, 230, 252, 68, 87, 107, 105, 157, 171, 212, 101, 48, 184, 2, 31, 188, 229, 57, 71, 81, 157, 144, 161, 44, 65, 0, 43, 238, 199, 200, 189, 124, 92, 1, 175, 79, 172, 222, 252, 57, 97, 235, 82, 72}

	client, err := client.NewClient(defaultBenchmarkClientID, defaultBenchmarkClientHost, *port, pubC, privC, PkiDir, providerInfo)
	if err != nil {
		panic(err)
	}

	benchClient, err := benchclient.NewBenchClient(client, *numMessages, *interval)
	if err != nil {
		panic(err)
	}

	err = benchClient.RunBench()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to spawn client instance: %v\n", err)
		os.Exit(-1)
	}
}

func newOpts(command string, usage string) *optparse.Parser {
	return optparse.New("Usage: loopix-client " + command + "\n\n  " + usage + "\n")
}
