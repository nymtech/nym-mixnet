package main

import (
	"os"

	"github.com/tav/golly/optparse"
)

const (
	// PkiDir is the location of the database file, relative to the project root. TODO: move this to homedir.
	PkiDir            = "pki/database.db"
	defaultHost       = "localhost"
	defaultID         = "Client1"
	defaultPort       = "6666"
	defaultProviderID = "666"
)

func cmdRun(args []string, usage string) {
	opts := newOpts("run [OPTIONS]", usage)
	id := opts.Flags("--id").Label("ID").String("Id of the loopix-client we want to run", defaultID)
	host := opts.Flags("--host").Label("HOST").String("The host on which the loopix-client is running", defaultHost)
	port := opts.Flags("--port").Label("PORT").String("Port on which loopix-client listens", defaultPort)
	providerID := opts.Flags("--provider").Label("PROVIDER").String("Id of the provider to connect to", defaultProviderID)

	println(id, host, port, providerID)

	params := opts.Parse(args)
	if len(params) != 0 {
		opts.PrintUsage()
		os.Exit(1)
	}

	wait := make(chan struct{})
	<-wait
}

func newOpts(command string, usage string) *optparse.Parser {
	return optparse.New("Usage: loopix-mixnode " + command + "\n\n  " + usage + "\n")
}
