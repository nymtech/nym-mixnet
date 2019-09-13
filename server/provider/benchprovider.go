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

package provider

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/flags"
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
	// logLocal.Warn("Disabling logging")
	// logLocal.Logger.Out = ioutil.Discard
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
		p.log.Infof("Listening on %s", p.host+":"+p.port)
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
	earliestMessageTimestamp := p.receivedMessages[0].timestamp
	latestMessageTimestamp := p.receivedMessages[0].timestamp

	for _, msg := range p.receivedMessages {
		if msg.timestamp.Before(earliestMessageTimestamp) {
			earliestMessageTimestamp = msg.timestamp
		}
		if msg.timestamp.After(latestMessageTimestamp) {
			latestMessageTimestamp = msg.timestamp
		}

		fmt.Fprintf(f, "%v\t%v\n", msg.timestamp, msg.content)
	}

	fmt.Printf("Earliest timestamp: %v\nLatest timestamp: %v\ntimedelta: %v\n",
		earliestMessageTimestamp,
		latestMessageTimestamp,
		latestMessageTimestamp.Sub(earliestMessageTimestamp),
	)

	return nil
}

// Function processes the received sphinx packet, performs the
// unwrapping operation and checks whether the packet should be
// forwarded or stored. If the processing was unsuccessful and error is returned.
func (p *BenchProvider) receivedPacket(packet []byte) error {
	p.log.Info("Received new sphinx packet")

	res := p.ProcessPacket(packet)
	dePacket := res.PacketData()
	nextHop := res.NextHop()
	flag := res.Flag()
	if err := res.Err(); err != nil {
		return err
	}

	if flag == flags.LastHopFlag {
		if nextHop.Id == "BenchmarkClientRecipient" {
			msgContent := string(dePacket[37:])
			p.receivedMessages = append(p.receivedMessages, timestampedMessage{timestamp: time.Now(), content: msgContent})
			p.receivedMessagesCount++
			if p.receivedMessagesCount == p.numMessages {
				fmt.Println("Received all expected messages")
				close(p.doneCh)
			}
		} else {
			panic(errors.New("unknown recipient"))
		}
	} else {
		fmt.Fprintf(os.Stderr, "%v - %v", nextHop.Address, nextHop.Id)
		panic(errors.New("unknown type packet received - benchmarking results will be unreliable"))
	}

	return nil
}

func (p *BenchProvider) listenForIncomingConnections() {
	for {
		conn, err := p.listener.Accept()

		if err != nil {
			p.log.Errorf("Error when listening for incoming connection: %v", err)
		} else {
			p.log.Infof("Received new connection from %s", conn.RemoteAddr())
			errs := make(chan error, 1)
			go p.handleConnection(conn, errs)
			err = <-errs
			if err != nil {
				p.log.Errorf("Error when listening for incoming connection: %v", err)
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

	if flags.PacketTypeFlagFromBytes(packet.Flag) == flags.CommFlag {
		if err := p.receivedPacket(packet.Data); err != nil {
			panic(err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "%v", string(packet.Data))
		panic(errors.New("unknown packet received - can't have those during benchmark"))
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
