// benchclient.go
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

package benchclient

import (
	"fmt"
	"os"
	"time"

	"github.com/nymtech/loopix-messaging/client"
	"github.com/nymtech/loopix-messaging/config"
)

const (
	payloadPrefix   = "testMessage"
	summaryFileName = "benchClientSummary"
)

type timestampedMessage struct {
	content   string
	timestamp time.Time
}

type BenchClient struct {
	*client.NetClient

	recipient      config.ClientConfig
	numberMessages int
	interval       time.Duration
	sentMessages   []timestampedMessage
}

func (bc *BenchClient) sendMessages(n int, interval time.Duration) {
	fmt.Printf("Going to try sending %v messages every %v\n", n, interval)
	for i := 0; i < n; i++ {
		msg := fmt.Sprintf("%v%v", payloadPrefix, i)
		if err := bc.SendMessage(msg, bc.recipient); err != nil {
			// if there was error while sending message, we need to panic as otherwise the result might be biased
			panic(err)
		}
		bc.sentMessages[i] = timestampedMessage{
			content:   msg,
			timestamp: time.Now(),
		}

		time.Sleep(interval)
	}
}

func (bc *BenchClient) createSummaryDoc() error {
	fmt.Println("Creating summary doc")
	f, err := os.Create(summaryFileName)
	if err != nil {
		return err
	}
	fmt.Fprintf(f, "Timestamp\tContent\n")
	for _, msg := range bc.sentMessages {
		fmt.Fprintf(f, "%v\t%v\n", msg.timestamp, msg.content)
	}
	return nil
}

func (bc *BenchClient) RunBench() error {
	defer bc.Shutdown()
	fmt.Println("starting bench client")

	// ignore all loopix requirements about cover trafic, etc. and just blast the system with messages
	client.ToggleControlMessageFetching(false)
	client.ToggleDropCoverTraffic(false)
	client.ToggleLoopCoverTraffic(false)
	client.ToggleRateCompliantCoverMessages(false)
	client.UpdateDesiredRateParameter(10000000.0)
	// to reduce effect of writing to stdout
	client.DisableLogging()
	// start underlying client
	if err := bc.NetClient.Start(); err != nil {
		return err
	}
	bc.sendMessages(bc.numberMessages, bc.interval)

	if err := bc.createSummaryDoc(); err != nil {
		return err
	}
	return nil
}

func NewBenchClient(nc *client.NetClient, numberMessages int, interval time.Duration) (*BenchClient, error) {

	return &BenchClient{
		NetClient:    nc,
		sentMessages: make([]timestampedMessage, numberMessages),
		recipient: config.ClientConfig{
			Id:       "Client2",
			Host:     "localhost",
			Port:     "9998",
			PubKey:   []byte{4, 135, 189, 82, 245, 150, 224, 233, 57, 59, 242, 8, 142, 7, 3, 147, 51, 103, 243, 23, 190, 69, 148, 150, 88, 234, 183, 187, 37, 227, 247, 57, 83, 85, 250, 21, 162, 163, 64, 168, 6, 27, 2, 236, 76, 225, 133, 152, 102, 28, 42, 254, 225, 21, 12, 221, 211},
			Provider: &nc.Provider,
		},
		numberMessages: numberMessages,
		interval:       interval,
	}, nil
}
