// Copyright 2018 The Loopix-Messaging Authors
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
	Package client implements the class of a network client which can interact with a mix network.
*/

package client

import (
	"bytes"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/loopix-messaging/clientcore"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/helpers"
	"github.com/nymtech/loopix-messaging/logging"
	"github.com/nymtech/loopix-messaging/networker"
)

var (
	logLocal                = logging.PackageLogger()
	loopCoverTrafficEnabled = true
	dropCoverTrafficEnabled = true
)

const (
	// the parameter of the exponential distribution which defines the rate of sending by client
	// the desiredRateParameter is the reciprocal of the expected value of the exponential distribution
	desiredRateParameter = 0.2
	loopRate             = 0.1
	dropRate             = 0.1
	// the rate at which clients are querying the provider for received packets. fetchRate value is the
	// parameter of an exponential distribution, and is the reciprocal of the expected value of the exp. distribution
	fetchRate = 0.1

	dummyPacketPayload = "foo"
)

// Client is the client networking interface
type Client interface {
	networker.NetworkClient
	networker.NetworkServer

	Start() error
	SendMessage(message string, recipient config.ClientConfig) error
	ReadInNetworkFromPKI(pkiName string) error
}

// TCPClient is a queuing TCP network client for the mixnet.
type TCPClient struct {
	id   string
	host string
	port string

	listener *net.TCPListener
	pkiDir   string

	config config.ClientConfig
	token  []byte

	outQueue         chan []byte
	registrationDone chan bool

	*clientcore.CryptoClient
	haltedCh chan struct{}
	haltOnce sync.Once
}

// it reads the network and users information from the PKI database
// and starts the listening server. Function returns an error
// signaling whenever any operation was unsuccessful.
func (c *TCPClient) Start() error {

	if err := c.resolveAddressAndStartListening(); err != nil {
		return err
	}

	c.outQueue = make(chan []byte)
	c.registrationDone = make(chan bool)

	err := c.ReadInNetworkFromPKI(c.pkiDir)
	if err != nil {
		logLocal.WithError(err).Error("Error during reading in network PKI")
		return err
	}

	go func() {
		for {
			select {
			case <-c.registrationDone:
				return
			default:
				err = c.sendRegisterMessageToProvider()
				if err != nil {
					logLocal.WithError(err).Error("Error during registration to provider", err)
				}
				time.Sleep(5 * time.Second)
			}
		}
	}()

	go c.startListenerInNewRoutine()

	// only start the sending routine for client1
	if c.config.Id == "Client1" {
		go c.startSenderInNewRoutine()
	}

	return nil
}

// Wait waits till the client is terminated for any reason.
func (c *TCPClient) Wait() {
	<-c.haltedCh
}

// TODO: create daemon to call this upon sigterm or something
// Shutdown cleanly shuts down a given client instance.
func (c *TCPClient) Shutdown() {
	c.haltOnce.Do(func() { c.halt() })
}

// calls any required cleanup code
func (c *TCPClient) halt() {
	logLocal.Info("Starting graceful shutdown")
	// close any listeners, free resources, etc

	close(c.haltedCh)
}

func (c *TCPClient) startSenderInNewRoutine() {
	// for now just send once for test sake
	time.Sleep(5 * time.Second)
	logLocal.Warn("send routine start")
	i := 0
	for {
		msg := fmt.Sprintf("%v%v", dummyPacketPayload, i)
		// recipient := c.config // just send to ourself, change it to other client once better PKI is figured out
		// randomRecipient, err := c.getRandomRecipient(c.Network.Clients)

		recipient := config.ClientConfig{
			Id:       "Client2",
			Host:     "localhost",
			Port:     "9998",
			PubKey:   []byte{4, 135, 189, 82, 245, 150, 224, 233, 57, 59, 242, 8, 142, 7, 3, 147, 51, 103, 243, 23, 190, 69, 148, 150, 88, 234, 183, 187, 37, 227, 247, 57, 83, 85, 250, 21, 162, 163, 64, 168, 6, 27, 2, 236, 76, 225, 133, 152, 102, 28, 42, 254, 225, 21, 12, 221, 211},
			Provider: c.config.Provider,
		}

		logLocal.Infof("sending %v to %v", msg, recipient.Id)

		// if err != nil {
		// 	logLocal.Warn(err)
		// 	break
		// }

		//nolint: errcheck
		c.SendMessage(msg, recipient)
		i++
		time.Sleep(5 * time.Second)

		select {
		case <-c.haltedCh:
			logLocal.Warn("send routine end")
			return
		default:
		}
	}

}

func (c *TCPClient) resolveAddressAndStartListening() error {
	addr, err := helpers.ResolveTCPAddress(c.host, c.port)
	if err != nil {
		return err
	}

	c.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	return nil
}

// SendMessage responsible for sending a real message. Takes as input the message string
// and the public information about the destination.
func (c *TCPClient) SendMessage(message string, recipient config.ClientConfig) error {
	packet, err := c.encodeMessage(message, recipient)
	if err != nil {
		logLocal.WithError(err).Error("Error in sending message - encode message returned error")
		return err
	}
	c.outQueue <- packet
	return nil
}

// encodeMessage encapsulates the given message into a sphinx packet destinated for recipient
// and wraps with the flag pointing that it is the communication packet
func (c *TCPClient) encodeMessage(message string, recipient config.ClientConfig) ([]byte, error) {
	sphinxPacket, err := c.EncodeMessage(message, recipient)
	if err != nil {
		logLocal.WithError(err).Error("Error in sending message - create sphinx packet returned an error")
		return nil, err
	}

	packetBytes, err := config.WrapWithFlag(config.CommFlag, sphinxPacket)
	if err != nil {
		logLocal.WithError(err).Error("Error in sending message - wrap with flag returned an error")
		return nil, err
	}
	return packetBytes, nil
}

// Send opens a connection with selected network address
// and send the passed packet. If connection failed or
// the packet could not be send, an error is returned
func (c *TCPClient) send(packet []byte, host string, port string) error {

	conn, err := net.Dial("tcp", host+":"+port)

	if err != nil {
		logLocal.WithError(err).Error("Error in send - dial returned an error")
		return err
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	return err
}

// run opens the listener to start listening on clients host and port
func (c *TCPClient) startListenerInNewRoutine() {
	defer c.listener.Close()

	go func() {
		logLocal.Infof("Listening on address %s", c.host+":"+c.port)
		c.listenForIncomingConnections()
	}()

	c.Wait()
}

// ListenForIncomingConnections responsible for running the listening process of the server;
// The clients listener accepts incoming connections and
// passes the incoming packets to the packet handler.
// If the connection could not be accepted an error
// is logged into the log files, but the function is not stopped
func (c *TCPClient) listenForIncomingConnections() {
	for {
		conn, err := c.listener.Accept()

		if err != nil {
			logLocal.WithError(err).Error(err)
		} else {
			go c.handleConnection(conn)
		}
	}
}

// HandleConnection handles the received packets; it checks the flag of the
// packet and schedules a corresponding process function;
// The potential errors are logged into the log files.
func (c *TCPClient) handleConnection(conn net.Conn) {

	buff := make([]byte, 1024)
	defer conn.Close()

	reqLen, err := conn.Read(buff)
	if err != nil {
		logLocal.WithError(err).Error("Error while reading incoming connection")
		panic(err)
	}
	var packet config.GeneralPacket
	err = proto.Unmarshal(buff[:reqLen], &packet)
	if err != nil {
		logLocal.WithError(err).Error("Error in unmarshal incoming packet")
	}

	switch {
	case bytes.Equal(packet.Flag, config.TokenFlag):
		c.registerToken(packet.Data)
		go func() {
			err := c.controlOutQueue()
			if err != nil {
				logLocal.WithError(err).Panic("Error in the controller of the outgoing packets queue. Possible security threat.")
			}
		}()

		if loopCoverTrafficEnabled {
			c.turnOnLoopCoverTraffic()
		}

		if dropCoverTrafficEnabled {
			c.turnOnDropCoverTraffic()
		}

		go func() {
			c.controlMessagingFetching()
		}()

	case bytes.Equal(packet.Flag, config.CommFlag):
		_, err := c.processPacket(packet.Data)
		if err != nil {
			logLocal.WithError(err).Error("Error in processing received packet")
		}
		if strings.Contains(string(packet.Data), dummyPacketPayload) {
			logLocal.Infof("Received new message: %v", string(packet.Data))
		}
		logLocal.Infof("Received new message: %v", string(packet.Data))
	default:
		logLocal.Info("Packet flag not recognised. Packet dropped.")
	}
}

// RegisterToken stores the authentication token received from the provider
func (c *TCPClient) registerToken(token []byte) {
	c.token = token
	logLocal.Infof(" Registered token %s", c.token)
	c.registrationDone <- true
}

// ProcessPacket processes the received sphinx packet and returns the
// encapsulated message or error in case the processing
// was unsuccessful.
func (c *TCPClient) processPacket(packet []byte) ([]byte, error) {
	// logLocal.Info(" Processing packet")
	return packet, nil
}

// SendRegisterMessageToProvider allows the client to register with the selected provider.
// The client sends a special assignment packet, with its public information, to the provider
// or returns an error.
func (c *TCPClient) sendRegisterMessageToProvider() error {

	logLocal.Info("Sending request to provider to register")

	confBytes, err := proto.Marshal(&c.config)
	if err != nil {
		logLocal.WithError(err).Error("Error in register provider - marshal of provider config returned an error")
		return err
	}

	pktBytes, err := config.WrapWithFlag(config.AssigneFlag, confBytes)
	if err != nil {
		logLocal.WithError(err).Error("Error in register provider - wrap with flag returned an error")
		return err
	}

	err = c.send(pktBytes, c.Provider.Host, c.Provider.Port)
	if err != nil {
		logLocal.WithError(err).Error("Error in register provider - send registration packet returned an error")
		return err
	}
	return nil
}

// GetMessagesFromProvider allows to fetch messages from the inbox stored by the
// provider. The client sends a pull packet to the provider, along with
// the authentication token. An error is returned if occurred.
func (c *TCPClient) getMessagesFromProvider() error {
	pullRqs := config.PullRequest{ClientId: c.id, Token: c.token}
	pullRqsBytes, err := proto.Marshal(&pullRqs)
	if err != nil {
		logLocal.WithError(err).Error("Error in register provider - marshal of pull request returned an error")
		return err
	}

	pktBytes, err := config.WrapWithFlag(config.PullFlag, pullRqsBytes)
	if err != nil {
		logLocal.WithError(err).Error("Error in register provider - marshal of provider config returned an error")
		return err
	}

	err = c.send(pktBytes, c.Provider.Host, c.Provider.Port)
	if err != nil {
		return err
	}

	return nil
}

// controlOutQueue controls the outgoing queue of the client.
// If a message awaits in the queue, it is sent. Otherwise a
// drop cover message is sent instead.
func (c *TCPClient) controlOutQueue() error {
	logLocal.Info("Queue controller started")
	for {
		select {
		case realPacket := <-c.outQueue:
			if err := c.send(realPacket, c.Provider.Host, c.Provider.Port); err != nil {
				logLocal.WithError(err).Errorf("Could not send real packet: %v", err)
			}
			logLocal.Info("Real packet was sent")
		default:
			dummyPacket, err := c.createDropCoverMessage()
			if err != nil {
				return err
			}
			if err := c.send(dummyPacket, c.Provider.Host, c.Provider.Port); err != nil {
				logLocal.WithError(err).Errorf("Could not send dummy packet: %v", err)
			}
			// logLocal.Info("OutQueue empty. Dummy packet sent.")
		}
		err := delayBeforeContinue(desiredRateParameter)
		if err != nil {
			return err
		}
	}
}

// controlMessagingFetching periodically at random sends a query to the provider
// to fetch received messages
func (c *TCPClient) controlMessagingFetching() {
	for {
		if err := c.getMessagesFromProvider(); err != nil {
			logLocal.WithError(err).Errorf("Could not get message from provider: %v", err)
			continue
		}
		logLocal.Info("Sent request to provider to fetch messages")
		err := delayBeforeContinue(fetchRate)
		if err != nil {
			logLocal.Error("Error in ControlMessagingFetching - generating random exp. value failed")
		}
	}
}

// CreateCoverMessage packs a dummy message into a Sphinx packet.
// The dummy message is a loop message.
func (c *TCPClient) createDropCoverMessage() ([]byte, error) {
	dummyLoad := "DummyPayloadMessage"
	randomRecipient, err := c.getRandomRecipient(c.Network.Clients)
	if err != nil {
		return nil, err
	}
	sphinxPacket, err := c.EncodeMessage(dummyLoad, randomRecipient)
	if err != nil {
		return nil, err
	}

	packetBytes, err := config.WrapWithFlag(config.CommFlag, sphinxPacket)
	if err != nil {
		return nil, err
	}
	return packetBytes, nil
}

// getRandomRecipient picks a random client from the list of all available clients (stored by the client).
// getRandomRecipient returns the selected client public configuration and an error
func (c *TCPClient) getRandomRecipient(slice []config.ClientConfig) (config.ClientConfig, error) {
	randIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(slice))))
	if err != nil {
		return config.ClientConfig{}, err
	}
	return slice[randIdx.Int64()], nil
}

// createLoopCoverMessage packs a dummy loop message into
// a sphinx packet. The loop message is destinated back to the sender
// createLoopCoverMessage returns a byte representation of the encapsulated packet and an error
func (c *TCPClient) createLoopCoverMessage() ([]byte, error) {
	loopLoad := "LoopCoverMessage"
	sphinxPacket, err := c.EncodeMessage(loopLoad, c.config)
	if err != nil {
		return nil, err
	}
	packetBytes, err := config.WrapWithFlag(config.CommFlag, sphinxPacket)
	if err != nil {
		return nil, err
	}
	return packetBytes, nil
}

// runLoopCoverTrafficStream manages the stream of loop cover traffic.
// In each stream iteration it sends a freshly created loop packet and
// waits a random time before scheduling the next loop packet.
func (c *TCPClient) runLoopCoverTrafficStream() error {
	logLocal.Info("Stream of loop cover traffic started")
	for {
		loopPacket, err := c.createLoopCoverMessage()
		if err != nil {
			return err
		}
		if err := c.send(loopPacket, c.Provider.Host, c.Provider.Port); err != nil {
			logLocal.WithError(err).Errorf("Could not send loop cover traffic message")
			return err
		}
		logLocal.Info("Loop message sent")
		err = delayBeforeContinue(loopRate)
		if err != nil {
			return err
		}
	}
}

// runDropCoverTrafficStream manages the stream of drop cover traffic.
// In each stream iteration it creates a fresh drop cover message destinated
// to a randomly selected user in the network. The drop packet is sent
// and the next stream call is scheduled after random time.
func (c *TCPClient) runDropCoverTrafficStream() error {
	logLocal.Info("Stream of drop cover traffic started")
	for {
		dropPacket, err := c.createDropCoverMessage()
		if err != nil {
			return err
		}
		if err := c.send(dropPacket, c.Provider.Host, c.Provider.Port); err != nil {
			logLocal.WithError(err).Errorf("Could not send loop drop cover traffic message")
			return err
		}
		logLocal.Info("Drop packet sent")
		err = delayBeforeContinue(dropRate)
		if err != nil {
			return err
		}
	}
}

func delayBeforeContinue(rateParam float64) error {
	delaySec, err := helpers.RandomExponential(rateParam)
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(int64(delaySec*math.Pow10(9))) * time.Nanosecond)
	return nil
}

// turnOnLoopCoverTraffic starts the stream of loop cover traffic
func (c *TCPClient) turnOnLoopCoverTraffic() {
	go func() {
		err := c.runLoopCoverTrafficStream()
		if err != nil {
			logLocal.WithError(err).Panic("Error in the controller of the loop cover traffic. Possible security threat.")
		}
	}()
}

// turnOnDropCoverTraffic starts the stream of drop cover traffic
func (c *TCPClient) turnOnDropCoverTraffic() {
	go func() {
		err := c.runDropCoverTrafficStream()
		if err != nil {
			logLocal.WithError(err).Panic("Error in the controller of the drop cover traffic. Possible security threat.")
		}
	}()
}

// ReadInNetworkFromPKI reads in the public information about active mixes
// from the PKI database and stores them locally. In case
// the connection or fetching data from the PKI went wrong,
// an error is returned.
func (c *TCPClient) ReadInNetworkFromPKI(pkiName string) error {
	logLocal.Infof("Reading network information from the PKI: %s", pkiName)

	mixes, err := helpers.GetMixesPKI(pkiName)
	if err != nil {
		logLocal.WithError(err).Error("Error while reading mixes from PKI")
		return err
	}
	c.Network.Mixes = mixes

	clients, err := helpers.GetClientPKI(pkiName)
	if err != nil {
		logLocal.WithError(err).Error("Error while reading clients from PKI")
		return err
	}
	c.Network.Clients = clients

	logLocal.Info("Network information uploaded")
	return nil
}

// The constructor function to create an new client object.
// Function returns a new client object or an error, if occurred.
func NewClient(id, host, port string, pubKey []byte, prvKey []byte, pkiDir string, provider config.MixConfig) (*TCPClient, error) {
	core := clientcore.NewCryptoClient(pubKey, prvKey, elliptic.P224(), provider, clientcore.NetworkPKI{})
	c := TCPClient{id: id,
		host:         host,
		port:         port,
		CryptoClient: core,
		pkiDir:       pkiDir,
		haltedCh:     make(chan struct{}),
	}
	c.config = config.ClientConfig{Id: c.id, Host: c.host, Port: c.port, PubKey: c.GetPublicKey(), Provider: &c.Provider}

	configBytes, err := proto.Marshal(&c.config)

	if err != nil {
		return nil, err
	}
	err = helpers.AddToDatabase(pkiDir, "Pki", c.id, "Client", configBytes)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// NewTestClient constructs a client object, which can be used for testing. The object contains the crypto core
// and the top-level of client, but does not involve networking and starting a listener.
func NewTestClient(id, host, port string, pubKey []byte, prvKey []byte, pkiDir string, provider config.MixConfig) (*TCPClient, error) {
	core := clientcore.NewCryptoClient(pubKey, prvKey, elliptic.P224(), provider, clientcore.NetworkPKI{})
	c := TCPClient{id: id, host: host, port: port, CryptoClient: core, pkiDir: pkiDir}
	c.config = config.ClientConfig{Id: c.id, Host: c.host, Port: c.port, PubKey: c.GetPublicKey(), Provider: &c.Provider}

	return &c, nil
}
