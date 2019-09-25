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
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/nym-mixnet/config"
	"github.com/nymtech/nym-mixnet/flags"
	"github.com/nymtech/nym-mixnet/helpers"
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

func (p *BenchProvider) startSendingPresence() {
	ticker := time.NewTicker(presenceInterval)
	for {
		select {
		case <-ticker.C:
			if err := helpers.RegisterMixProviderPresence(p.GetPublicKey(),
				p.convertRecordsToModelData(),
				net.JoinHostPort(p.host, p.port),
			); err != nil {
				p.log.Errorf("Failed to register presence: %v", err)
			}
		case <-p.haltedCh:
			return
		}
	}
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
	go p.startSendingPresence()

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
			msgContent := string(dePacket[38:])
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
			if e, ok := err.(net.Error); ok && !e.Temporary() {
				p.log.Panicf("Critical accept failure: %v", err)
				return
			}
			continue
		}

		p.log.Infof("Received new connection from %s", conn.RemoteAddr())
		go p.handleConnection(conn)
	}
}

// HandleConnection handles the received packets; it checks the flag of the
// packet and schedules a corresponding process function and returns an error.
func (p *BenchProvider) handleConnection(conn net.Conn) {
	defer func() {
		p.log.Debugf("Closing Connection to %v", conn.RemoteAddr())
		if err := conn.Close(); err != nil {
			p.log.Warnf("error when closing connection from %s: %v", conn.RemoteAddr(), err)
		}
	}()

	buff := make([]byte, 1024)
	reqLen, err := conn.Read(buff)
	if err != nil {
		p.log.Errorf("Error while reading from the connection: %v", err)
		return
	}

	var packet config.GeneralPacket
	if err = proto.Unmarshal(buff[:reqLen], &packet); err != nil {
		p.log.Errorf("Error while unmarshalling received packet: %v", err)
		return
	}

	switch flags.PacketTypeFlagFromBytes(packet.Flag) {
	case flags.CommFlag:
		if err := p.receivedPacket(packet.Data); err != nil {
			panic(err)
		}

	default:
		fmt.Fprintf(os.Stderr, "%v", string(packet.Data))
		panic(errors.New("unknown packet received - can't have those during benchmark"))
	}
}

func NewBenchProvider(provider *ProviderServer, numMessages int) (*BenchProvider, error) {
	bp := &BenchProvider{
		doneCh:           make(chan struct{}),
		ProviderServer:   provider,
		numMessages:      numMessages,
		receivedMessages: make([]timestampedMessage, 0, numMessages),
	}
	bp.ProviderServer.log.Out = ioutil.Discard
	return bp, nil
}
