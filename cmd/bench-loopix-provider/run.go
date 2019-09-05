package main

import (
	"os"

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
	privP := []byte{36, 15, 19, 37, 25, 137, 18, 6, 135, 122, 193, 134, 112, 92, 20, 237, 54, 204, 150, 242, 13, 113, 178, 175, 209, 164, 83, 201}
	pubP := []byte{4, 212, 28, 250, 98, 86, 155, 24, 162, 117, 236, 179, 218, 173, 182, 40, 1, 18, 244, 31, 0, 246, 217, 108, 240, 152, 78, 215, 51, 70, 232, 202, 47, 45, 222, 165, 241, 132, 198, 137, 95, 126, 108, 47, 153, 49, 156, 105, 202, 153, 8, 249, 231, 84, 76, 241, 178}

	baseProviderServer, err := server.NewProviderServer(defaultBenchmarkProviderID, defaultBenchmarkProviderHost, *port, pubP, privP, PkiDir)
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
