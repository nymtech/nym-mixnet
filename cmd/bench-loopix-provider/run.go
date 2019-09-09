package main

import (
	"os"

	"github.com/nymtech/loopix-messaging/sphinx"

	"github.com/nymtech/loopix-messaging/server"
	"github.com/tav/golly/optparse"
)

const (
	// PkiDir is the location of the database file, relative to the project root. TODO: move this to homedir.
	PkiDir                       = "pki/database.db"
	defaultBenchmarkProviderHost = "localhost"
	defaultBenchmarkProviderPort = "11000"
	defaultBenchmarkProviderID   = "BenchmarkProvider"
)

func cmdRun(args []string, usage string) {
	opts := newOpts("run [OPTIONS]", usage)
	port := opts.Flags("--port").Label("PORT").String("Port on which loopix-provider listens", defaultBenchmarkProviderPort)
	numMessages := opts.Flags("--num").Label("NUMMESSAGES").Int("Number of benchmark messages to send", 0)

	params := opts.Parse(args)
	if len(params) != 0 {
		opts.PrintUsage()
		os.Exit(1)
	}

	// have constant keys to simplify the procedure so that pki/database would not need to be reset every run
	privP := sphinx.BytesToPrivateKey([]byte{191, 43, 90, 175, 50, 224, 156, 22, 204, 173, 87, 255, 64, 152, 17, 30, 48, 162, 36, 95, 57, 34, 187, 183, 203, 215, 25, 172, 55, 199, 211, 59})
	pubP := sphinx.BytesToPublicKey([]byte{17, 170, 15, 150, 155, 75, 240, 66, 54, 100, 131, 127, 193, 10, 133, 32, 62, 155, 9, 46, 200, 55, 60, 125, 223, 76, 170, 167, 100, 34, 176, 117})

	baseProviderServer, err := server.NewProviderServer(defaultBenchmarkProviderID, defaultBenchmarkProviderHost, *port, privP, pubP, PkiDir)
	if err != nil {
		panic(err)
	}

	benchmarkProviderServer, err := server.NewBenchProvider(baseProviderServer, *numMessages)
	if err != nil {
		panic(err)
	}

	server.DisableLogging()

	err = benchmarkProviderServer.RunBench()
	if err != nil {
		panic(err)
	}
}

func newOpts(command string, usage string) *optparse.Parser {
	return optparse.New("Usage: loopix-provider " + command + "\n\n  " + usage + "\n")
}
