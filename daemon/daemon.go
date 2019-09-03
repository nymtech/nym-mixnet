// daemon.go - daemon base for services.
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

// Package daemon defines common structure for all daemonizable services.
// TODO: it was literally copied from the coconut source. We need to somehow combine them to not end up with two different versions
package daemon

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
)

type Service interface {
	Shutdown()
	Wait()
}

type StartUpFunc func() Service
type FlagDefineFunc func()

func Start(flagsFn FlagDefineFunc, startFn StartUpFunc) {
	const PtrSize = 32 << uintptr(^uintptr(0)>>63)
	if PtrSize != 64 || strconv.IntSize != 64 {
		fmt.Fprintf(os.Stderr,
			"The binary seems to not have been compiled in 64bit mode. Runtime pointer size: %v, Int size: %v\n",
			PtrSize,
			strconv.IntSize,
		)
		os.Exit(-1)
	}

	flagsFn()
	flag.Parse()

	syscall.Umask(0077)

	// Ensure that a sane number of OS threads is allowed.
	if os.Getenv("GOMAXPROCS") == "" {
		// But only if the user isn't trying to override it.
		nProcs := runtime.GOMAXPROCS(0)
		nCPU := runtime.NumCPU()
		if nProcs < nCPU {
			runtime.GOMAXPROCS(nCPU)
		}
	}

	// Setup the signal handling.
	haltCh := make(chan os.Signal, 1)
	signal.Notify(haltCh, os.Interrupt, syscall.SIGTERM)
	// for now ignore SIGHUP signal, TODO: handle it similarly to katzenpost

	service := startFn()
	defer service.Shutdown()

	// Halt the service gracefully on SIGINT/SIGTERM.
	go func() {
		<-haltCh
		service.Shutdown()
	}()

	// Wait for the service to explode or be terminated.
	service.Wait()
}
