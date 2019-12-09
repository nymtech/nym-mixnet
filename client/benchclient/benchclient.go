// Copyright 2019 The Nym Mixnet Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
	Package benchclient defines functionalities for client designed to benchmark the capabilities of the system
*/

package benchclient

import (
	"fmt"
	"os"
	"time"

	"github.com/nymtech/nym-mixnet/client"
	"github.com/nymtech/nym-mixnet/config"
	"github.com/nymtech/nym-mixnet/flags"
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

	recipient          config.ClientConfig
	numberMessages     int
	interval           time.Duration
	sentMessages       []timestampedMessage
	pregen             bool
	pregeneratedPacket []byte
}

func (bc *BenchClient) sendMessages(n int, interval time.Duration) {
	fmt.Printf("Going to try sending %v messages every %v\n", n, interval)
	if bc.pregen {
		fmt.Println("Going to be sending the pre-generated packet")
		for i := 0; i < n; i++ {
			bc.OutQueue() <- bc.pregeneratedPacket
			bc.sentMessages[i] = timestampedMessage{
				content:   payloadPrefix,
				timestamp: time.Now(),
			}
			time.Sleep(interval)
		}
	} else {
		for i := 0; i < n; i++ {
			msg := fmt.Sprintf("%v%v", payloadPrefix, i)
			fmt.Println("Sending", msg)
			if err := bc.SendMessage([]byte(msg), bc.recipient); err != nil {
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
}

func (bc *BenchClient) createSummaryDoc() error {
	fmt.Println("Creating summary doc")
	f, err := os.Create(summaryFileName)
	if err != nil {
		return err
	}
	fmt.Fprintf(f, "Timestamp\tContent\n")
	earliestMessageTimestamp := bc.sentMessages[0].timestamp
	latestMessageTimestamp := bc.sentMessages[0].timestamp

	for _, msg := range bc.sentMessages {
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

func (bc *BenchClient) RunBench() error {
	defer bc.Shutdown()
	fmt.Println("starting bench client")

	if err := bc.NetClient.Start(); err != nil {
		return err
	}
	if bc.pregen {
		if err := bc.pregeneratePacket(payloadPrefix, bc.recipient); err != nil {
			return err
		}
	}

	bc.sendMessages(bc.numberMessages, bc.interval)

	if err := bc.createSummaryDoc(); err != nil {
		return err
	}
	return nil
}

func (bc *BenchClient) pregeneratePacket(message string, recipient config.ClientConfig) error {
	sphinxPacket, err := bc.EncodeMessage([]byte(message), recipient)
	if err != nil {
		return err
	}

	packetBytes, err := config.WrapWithFlag(flags.CommFlag, sphinxPacket)
	if err != nil {
		return err
	}

	bc.pregeneratedPacket = packetBytes
	return nil
}

func NewBenchClient(nc *client.NetClient, numMsgs int, interval time.Duration, pregen bool) (*BenchClient, error) {
	bc := &BenchClient{
		NetClient:    nc,
		sentMessages: make([]timestampedMessage, numMsgs),
		recipient: config.ClientConfig{
			Id:   "BenchmarkClientRecipient",
			Host: "localhost",
			Port: "9998",
			PubKey: []byte{21, 103, 130, 37, 105, 58, 162, 113, 91, 198, 76, 156, 194, 36, 45,
				219, 121, 158, 255, 247, 44, 159, 243, 155, 215, 90, 67, 103, 64, 242, 95, 45},
			Provider: &config.MixConfig{
				Id:   "BenchmarkProvider",
				Host: "localhost",
				Port: "11000",
				PubKey: []byte{17, 170, 15, 150, 155, 75, 240, 66, 54, 100, 131, 127, 193, 10,
					133, 32, 62, 155, 9, 46, 200, 55, 60, 125, 223, 76, 170, 167, 100, 34, 176, 117},
			},
		},
		numberMessages:     numMsgs,
		interval:           interval,
		pregen:             pregen,
		pregeneratedPacket: nil,
	}
	return bc, nil
}
