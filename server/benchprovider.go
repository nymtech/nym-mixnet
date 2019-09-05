// benchprovider.go
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

package server

import (
	"errors"
	"os"
	"time"

	"bytes"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/sphinx"
)

const (
	summaryFileName = "benchProviderSummary"
)

type timestampedMessage struct {
	content   string
	timestamp time.Time
}

type BenchProvider struct {
	*ProviderServer
	doneCh                chan struct{}
	numMessages           int
	receivedMessages      []timestampedMessage
	receivedMessagesCount int
}

func DisableLogging() {
	logLocal.Warn("Disabling logging")
	logLocal.Logger.Out = ioutil.Discard
}

// Start creates loggers for capturing info and error logs
// and starts the listening server. Returns an error
// if any operation was unsuccessful.
func (p *BenchProvider) RunBench() error {
	fmt.Println("Expecting to receive", p.numMessages, "messages")
	p.run()

	return nil
}

// Function opens the listener to start listening on provider's host and port
func (p *BenchProvider) run() {

	defer p.listener.Close()

	go func() {
		logLocal.Infof("Listening on %s", p.host+":"+p.port)
		p.listenForIncomingConnections()
	}()

	<-p.doneCh

	if err := p.createSummaryDoc(); err != nil {
		panic(err)
	}
}

func (p *BenchProvider) createSummaryDoc() error {
	fmt.Println("Creating summary doc")
	f, err := os.Create(summaryFileName)
	if err != nil {
		return err
	}
	fmt.Fprintf(f, "Timestamp\tContent\n")
	for _, msg := range p.receivedMessages {
		fmt.Fprintf(f, "%v\t%v\n", msg.timestamp, msg.content)
	}
	return nil
}

// Function processes the received sphinx packet, performs the
// unwrapping operation and checks whether the packet should be
// forwarded or stored. If the processing was unsuccessful and error is returned.
func (p *BenchProvider) receivedPacket(packet []byte) error {
	logLocal.Info("Received new sphinx packet")

	c := make(chan []byte)
	cAdr := make(chan sphinx.Hop)
	cFlag := make(chan []byte)
	errCh := make(chan error)

	go p.ProcessPacket(packet, c, cAdr, cFlag, errCh)
	dePacket := <-c
	nextHop := <-cAdr
	flag := <-cFlag
	err := <-errCh

	if err != nil {
		return err
	}

	if bytes.Equal(flag, sphinx.LastHopFlag) {
		if nextHop.Id == "BenchmarkClientRecipient" {
			msgContent := string(dePacket[63:])
			p.receivedMessages = append(p.receivedMessages, timestampedMessage{timestamp: time.Now(), content: msgContent})
			p.receivedMessagesCount++
			if p.receivedMessagesCount == p.numMessages {
				fmt.Println("Received all expected messages")
				close(p.doneCh)
			}
		} else {
			panic(errors.New("Unknown recipient"))
		}
	} else {
		fmt.Fprintf(os.Stderr, "%v - %v", nextHop.Address, nextHop.Id)
		panic(errors.New("Unknown type packet received - benchmarking results will be unreliable"))
	}

	return nil
}

func (p *BenchProvider) listenForIncomingConnections() {
	for {
		conn, err := p.listener.Accept()

		if err != nil {
			logLocal.WithError(err).Error(err)
		} else {
			logLocal.Infof("Received new connection from %s", conn.RemoteAddr())
			errs := make(chan error, 1)
			go p.handleConnection(conn, errs)
			err = <-errs
			if err != nil {
				logLocal.WithError(err).Error(err)
			}
		}
	}
}

// HandleConnection handles the received packets; it checks the flag of the
// packet and schedules a corresponding process function and returns an error.
func (p *BenchProvider) handleConnection(conn net.Conn, errs chan<- error) {

	buff := make([]byte, 1024)
	reqLen, err := conn.Read(buff)
	defer conn.Close()

	if err != nil {
		errs <- err
	}

	var packet config.GeneralPacket
	err = proto.Unmarshal(buff[:reqLen], &packet)
	if err != nil {
		errs <- err
	}

	if bytes.Equal(packet.Flag, config.CommFlag) {
		if err := p.receivedPacket(packet.Data); err != nil {
			panic(err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "%v", string(packet.Data))
		panic(errors.New("Unknown packet received - can't have those during benchmark"))
	}
	errs <- nil
}

func NewBenchProvider(provider *ProviderServer, numMessages int) (*BenchProvider, error) {
	return &BenchProvider{
		doneCh:           make(chan struct{}),
		ProviderServer:   provider,
		numMessages:      numMessages,
		receivedMessages: make([]timestampedMessage, 0, numMessages),
	}, nil
}
