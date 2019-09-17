// Copyright 2018-2019 The Loopix-Messaging Authors
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
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net"
	"os"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/directory-server/models"
	"github.com/nymtech/loopix-messaging/clientcore"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/flags"
	"github.com/nymtech/loopix-messaging/helpers"
	"github.com/nymtech/loopix-messaging/logger"
	"github.com/nymtech/loopix-messaging/networker"
	"github.com/nymtech/loopix-messaging/sphinx"
	"github.com/sirupsen/logrus"
)

// TODO: we need to deal with all those globals
//nolint: gochecknoglobals
var (
	loopCoverTrafficEnabled           = true
	dropCoverTrafficEnabled           = true
	controlMessageFetchingEnabled     = true
	rateCompliantCoverMessagesEnabled = true
	// FIXME: temporarily moved to variable to make it possible to override it by bench client
	// the parameter of the exponential distribution which defines the rate of sending by client
	// the desiredRateParameter is the reciprocal of the expected value of the exponential distribution
	desiredRateParameter = 5.0
)

const (
	loopRate = 0.1
	dropRate = 0.1
	// the rate at which clients are querying the provider for received packets. fetchRate value is the
	// parameter of an exponential distribution, and is the reciprocal of the expected value of the exp. distribution
	fetchRate = 5

	// Below should be moved to a config file once we have it
	// logFileLocation can either point to some valid file to which all log data should be written
	// or if left an empty string, stdout will be used instead
	defaultLogFileLocation = ""
	// considering we are under heavy development and nowhere near production level, log EVERYTHING
	defaultLogLevel = "trace"

	dummyLoad = "DummyPayloadMessage"
	loopLoad  = "LoopCoverMessage"
)

// Client is the client networking interface
type Client interface {
	networker.NetworkClient
	networker.NetworkServer

	Start() error
	SendMessage(message string, recipient config.ClientConfig) error
	ReadInNetworkFromTopology(pkiName string) error
}

// NetClient is a queuing TCP network client for the mixnet.
type NetClient struct {
	*clientcore.CryptoClient
	id               string
	host             string
	port             string
	listener         *net.TCPListener
	config           config.ClientConfig
	token            []byte
	outQueue         chan []byte
	registrationDone chan bool
	haltedCh         chan struct{}
	haltOnce         sync.Once
	log              *logrus.Logger
	demoRecipient    config.ClientConfig
}

// OutQueue returns a reference to the client's outQueue. It's a queue
// which holds outgoing packets while their order is randomised.
func (c *NetClient) OutQueue() chan<- []byte {
	return c.outQueue
}

// ToggleRateCompliantCoverTraffic enables or disables rate compliant cover
// traffic.
func ToggleRateCompliantCoverTraffic(b bool) {
	if !b {
		fmt.Println("Rate compliant cover messages are disabled")
	} else {
		fmt.Println("Rate compliant cover messages are enabled")
	}
	rateCompliantCoverMessagesEnabled = b
}

// ToggleLoopCoverTraffic enables or disables loop cover traffic.
func ToggleLoopCoverTraffic(b bool) {
	if !b {
		fmt.Println("Loop cover traffic is disabled")
	} else {
		fmt.Println("Loop cover traffic is enabled")
	}
	loopCoverTrafficEnabled = b
}

// ToggleDropCoverTraffic enables or disables cover traffic.
func ToggleDropCoverTraffic(b bool) {
	if !b {
		fmt.Println("Drop cover traffic is disabled")
	} else {
		fmt.Println("Drop cover traffic is enabled")
	}
	dropCoverTrafficEnabled = b
}

// ToggleControlMessageFetching enables or disables control message fetching.
func ToggleControlMessageFetching(b bool) {
	if !b {
		fmt.Println("Control message fetching is disabled")
	} else {
		fmt.Println("Control message fetching is enabled")
	}
	controlMessageFetchingEnabled = b
}

// UpdateDesiredRateParameter sets the desired rate parameter.
func UpdateDesiredRateParameter(r float64) {
	fmt.Printf("Updating desired rate parameter to %v\n", r)
	desiredRateParameter = r
}

func (c *NetClient) DisableLogging() {
	c.log.Warn("Disabling logging")
	c.log.Out = ioutil.Discard
}

func (c *NetClient) ChangeLoggingLevel(levelStr string) {
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		c.log.Errorf("Failed to parse passed logging level '%v': %v", levelStr, err)
	}
	c.log.Infof("Changing logging level to %v", level.String())
	c.log.SetLevel(level)
}

func (c *NetClient) startInputRoutine() {
	reader := bufio.NewReader(os.Stdin)

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			c.log.Errorf("Failed to read user input: %v", err)
		}
		input = input[:len(input)-1]
		c.log.Infof("Sending: %v to %v", input, c.demoRecipient.GetId())
		if err := c.SendMessage(input, c.demoRecipient); err != nil {
			c.log.Errorf("Failed to send %v to %v: %v", input, c.demoRecipient.GetId(), err)
		}
	}
}

// Start reads the network and users information from the topology
// and starts the listening server. Returns an error
// signalling whenever any operation was unsuccessful.
func (c *NetClient) Start() error {
	if err := c.resolveAddressAndStartListening(); err != nil {
		return err
	}

	c.outQueue = make(chan []byte)
	c.registrationDone = make(chan bool)

	initialTopology, err := helpers.GetNetworkTopology()
	if err := c.ReadInNetworkFromTopology(initialTopology); err != nil {
		return err
	}

	provider, err := providerFromTopology(initialTopology)
	if err != nil {
		return err
	}
	c.Provider = provider

	if err != nil {
		c.log.Errorf("Error during reading in network PKI: %v", err)
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
					c.log.Errorf("Error during registration to provider: %v", err)
				}
				time.Sleep(5 * time.Second)
			}
		}
	}()

	go c.startListenerInNewRoutine()
	go c.startInputRoutine()
	return nil
}

// Wait waits till the client is terminated for any reason.
func (c *NetClient) Wait() {
	<-c.haltedCh
}

// Shutdown cleanly shuts down a given client instance.
// TODO: create daemon to call this upon sigterm or something
func (c *NetClient) Shutdown() {
	c.haltOnce.Do(func() { c.halt() })
}

// calls any required cleanup code
func (c *NetClient) halt() {
	c.log.Infof("Starting graceful shutdown")
	// close any listeners, free resources, etc

	close(c.haltedCh)
}

func (c *NetClient) resolveAddressAndStartListening() error {
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
func (c *NetClient) SendMessage(message string, recipient config.ClientConfig) error {
	packet, err := c.encodeMessage(message, recipient)
	if err != nil {
		c.log.Errorf("Error in sending message - encode message returned error: %v", err)
		return err
	}
	c.outQueue <- packet
	return nil
}

// encodeMessage encapsulates the given message into a sphinx packet destinated for recipient
// and wraps with the flag pointing that it is the communication packet
func (c *NetClient) encodeMessage(message string, recipient config.ClientConfig) ([]byte, error) {
	sphinxPacket, err := c.EncodeMessage(message, recipient)
	if err != nil {
		c.log.Errorf("Error in sending message - create sphinx packet returned an error: %v", err)
		return nil, err
	}

	packetBytes, err := config.WrapWithFlag(flags.CommFlag, sphinxPacket)
	if err != nil {
		c.log.Errorf("Error in sending message - wrap with flag returned an error: %v", err)
		return nil, err
	}
	return packetBytes, nil
}

// Send opens a connection with selected network address
// and send the passed packet. If connection failed or
// the packet could not be send, an error is returned
func (c *NetClient) send(packet []byte, host string, port string) error {

	conn, err := net.Dial("tcp", host+":"+port)

	if err != nil {
		c.log.Errorf("Error in send - dial returned an error: %v", err)
		return err
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	return err
}

// run opens the listener to start listening on clients host and port
func (c *NetClient) startListenerInNewRoutine() {
	defer c.listener.Close()

	go func() {
		c.log.Infof("Listening on address %s", c.host+":"+c.port)
		c.listenForIncomingConnections()
	}()

	c.Wait()
}

// ListenForIncomingConnections responsible for running the listening process of the server;
// The clients listener accepts incoming connections and
// passes the incoming packets to the packet handler.
// If the connection could not be accepted an error
// is logged into the log files, but the function is not stopped
func (c *NetClient) listenForIncomingConnections() {
	for {
		conn, err := c.listener.Accept()

		if err != nil {
			c.log.Errorf("Could not accept connection: %v", err)
		} else {
			go c.handleConnection(conn)
		}
	}
}

// HandleConnection handles the received packets; it checks the flag of the
// packet and schedules a corresponding process function;
// The potential errors are logged into the log files.
func (c *NetClient) handleConnection(conn net.Conn) {

	buff := make([]byte, 1024)
	defer conn.Close()

	reqLen, err := conn.Read(buff)
	if err != nil {
		c.log.Errorf("Error while reading incoming connection: %v", err)
		panic(err)
	}
	var packet config.GeneralPacket
	err = proto.Unmarshal(buff[:reqLen], &packet)
	if err != nil {
		c.log.Errorf("Error in unmarshal incoming packet: %v", err)
	}

	switch flags.PacketTypeFlagFromBytes(packet.Flag) {
	case flags.TokenFlag:
		c.registerToken(packet.Data)
		go func() {
			err := c.controlOutQueue()
			if err != nil {
				c.log.Panicf("Error in the controller of the outgoing packets queue. Possible security threat.: %v", err)
			}
		}()

		if loopCoverTrafficEnabled {
			c.turnOnLoopCoverTraffic()
		}

		if dropCoverTrafficEnabled {
			c.turnOnDropCoverTraffic()
		}

		if controlMessageFetchingEnabled {
			go func() {
				c.controlMessagingFetching()
			}()
		}

	case flags.CommFlag:
		packetData, err := c.processPacket(packet.Data)
		if err != nil {
			c.log.Errorf("Error in processing received packet: %v", err)
		}
		packetDataStr := string(packetData)
		switch packetDataStr {
		case loopLoad:
			c.log.Debugf("Received loop cover message %v", packetDataStr)
		case dummyLoad:
			c.log.Debugf("Received drop cover message %v", packetDataStr)
		default:
			c.log.Infof("Received new message: %v", packetDataStr)
		}
	default:
		c.log.Warnf("Packet flag not recognised. Packet dropped.")
	}
}

// RegisterToken stores the authentication token received from the provider
func (c *NetClient) registerToken(token []byte) {
	c.token = token
	c.log.Debugf(" Registered token %s", c.token)
	c.registrationDone <- true
}

// ProcessPacket processes the received sphinx packet and returns the
// encapsulated message or error in case the processing
// was unsuccessful.
func (c *NetClient) processPacket(packet []byte) ([]byte, error) {

	// c.log.Debugf(" Processing packet")
	// c.log.Tracef("Removing first 37 bytes of the message")
	if len(packet) > 38 {
		return packet[38:], nil
	}
	return packet, nil
}

// SendRegisterMessageToProvider allows the client to register with the selected provider.
// The client sends a special assignment packet, with its public information, to the provider
// or returns an error.
func (c *NetClient) sendRegisterMessageToProvider() error {

	c.log.Debugf("Sending request to provider to register")

	confBytes, err := proto.Marshal(&c.config)
	if err != nil {
		c.log.Errorf("Error in register provider - marshal of provider config returned an error: %v", err)
		return err
	}

	pktBytes, err := config.WrapWithFlag(flags.AssignFlag, confBytes)
	if err != nil {
		c.log.Errorf("Error in register provider - wrap with flag returned an error: %v", err)
		return err
	}

	err = c.send(pktBytes, c.Provider.Host, c.Provider.Port)
	if err != nil {
		c.log.Errorf("Error in register provider - send registration packet returned an error: %v", err)
		return err
	}
	return nil
}

// GetMessagesFromProvider allows to fetch messages from the inbox stored by the
// provider. The client sends a pull packet to the provider, along with
// the authentication token. An error is returned if occurred.
func (c *NetClient) getMessagesFromProvider() error {
	pullRqs := config.PullRequest{ClientId: c.id, Token: c.token}
	pullRqsBytes, err := proto.Marshal(&pullRqs)
	if err != nil {
		c.log.Errorf("Error in register provider - marshal of pull request returned an error: %v", err)
		return err
	}

	pktBytes, err := config.WrapWithFlag(flags.PullFlag, pullRqsBytes)
	if err != nil {
		c.log.Errorf("Error in register provider - marshal of provider config returned an error: %v", err)
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
func (c *NetClient) controlOutQueue() error {
	c.log.Debugf("Queue controller started")
	for {
		select {
		case realPacket := <-c.outQueue:
			if err := c.send(realPacket, c.Provider.Host, c.Provider.Port); err != nil {
				c.log.Errorf("Could not send real packet: %v", err)
			}
			c.log.Debugf("Real packet was sent")
		default:
			if rateCompliantCoverMessagesEnabled {
				dummyPacket, err := c.createDropCoverMessage()
				if err != nil {
					return err
				}
				if err := c.send(dummyPacket, c.Provider.Host, c.Provider.Port); err != nil {
					c.log.Errorf("Could not send dummy packet: %v", err)
				}
				// c.log.Infof("OutQueue empty. Dummy packet sent.")
			}
		}
		err := delayBeforeContinue(desiredRateParameter)
		if err != nil {
			return err
		}
	}
}

// controlMessagingFetching periodically at random sends a query to the provider
// to fetch received messages
func (c *NetClient) controlMessagingFetching() {
	for {
		if err := c.getMessagesFromProvider(); err != nil {
			c.log.Errorf("Could not get message from provider: %v", err)
			continue
		}
		// c.log.Infof("Sent request to provider to fetch messages")
		err := delayBeforeContinue(fetchRate)
		if err != nil {
			c.log.Errorf("Error in ControlMessagingFetching - generating random exp. value failed: %v", err)
		}
	}
}

// CreateCoverMessage packs a dummy message into a Sphinx packet.
// The dummy message is a loop message.
func (c *NetClient) createDropCoverMessage() ([]byte, error) {
	randomRecipient, err := c.getRandomRecipient(c.Network.Clients)
	if err != nil {
		return nil, err
	}
	sphinxPacket, err := c.EncodeMessage(dummyLoad, randomRecipient)
	if err != nil {
		return nil, err
	}

	packetBytes, err := config.WrapWithFlag(flags.CommFlag, sphinxPacket)
	if err != nil {
		return nil, err
	}
	return packetBytes, nil
}

// getRandomRecipient picks a random client from the list of all available clients (stored by the client).
// getRandomRecipient returns the selected client public configuration and an error
func (c *NetClient) getRandomRecipient(slice []config.ClientConfig) (config.ClientConfig, error) {
	randIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(slice))))
	if err != nil {
		return config.ClientConfig{}, err
	}
	return slice[randIdx.Int64()], nil
}

// createLoopCoverMessage packs a dummy loop message into
// a sphinx packet. The loop message is destinated back to the sender
// createLoopCoverMessage returns a byte representation of the encapsulated packet and an error
func (c *NetClient) createLoopCoverMessage() ([]byte, error) {
	sphinxPacket, err := c.EncodeMessage(loopLoad, c.config)
	if err != nil {
		return nil, err
	}
	packetBytes, err := config.WrapWithFlag(flags.CommFlag, sphinxPacket)
	if err != nil {
		return nil, err
	}
	return packetBytes, nil
}

// runLoopCoverTrafficStream manages the stream of loop cover traffic.
// In each stream iteration it sends a freshly created loop packet and
// waits a random time before scheduling the next loop packet.
func (c *NetClient) runLoopCoverTrafficStream() error {
	c.log.Debugf("Stream of loop cover traffic started")
	for {
		loopPacket, err := c.createLoopCoverMessage()
		if err != nil {
			return err
		}
		if err := c.send(loopPacket, c.Provider.Host, c.Provider.Port); err != nil {
			c.log.Errorf("Could not send loop cover traffic message: %v", err)
			return err
		}
		c.log.Debugf("Loop message sent")
		if err := delayBeforeContinue(loopRate); err != nil {
			return err
		}
	}
}

// runDropCoverTrafficStream manages the stream of drop cover traffic.
// In each stream iteration it creates a fresh drop cover message destinated
// to a randomly selected user in the network. The drop packet is sent
// and the next stream call is scheduled after random time.
func (c *NetClient) runDropCoverTrafficStream() error {
	c.log.Debugf("Stream of drop cover traffic started")
	for {
		dropPacket, err := c.createDropCoverMessage()
		if err != nil {
			return err
		}
		if err := c.send(dropPacket, c.Provider.Host, c.Provider.Port); err != nil {
			c.log.Errorf("Could not send loop drop cover traffic message: %v", err)
			return err
		}
		c.log.Debugf("Drop packet sent")
		if err := delayBeforeContinue(dropRate); err != nil {
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
func (c *NetClient) turnOnLoopCoverTraffic() {
	go func() {
		err := c.runLoopCoverTrafficStream()
		if err != nil {
			c.log.Errorf("Error in the controller of the loop cover traffic. Possible security threat.: %v", err)
		}
	}()
}

// turnOnDropCoverTraffic starts the stream of drop cover traffic
func (c *NetClient) turnOnDropCoverTraffic() {
	go func() {
		err := c.runDropCoverTrafficStream()
		if err != nil {
			c.log.Errorf("Error in the controller of the drop cover traffic. Possible security threat.: %v", err)
		}
	}()
}

// ReadInNetworkFromTopology reads in the public information about active mixes
// from the topology and stores them locally. In case
// the connection or fetching data from the PKI went wrong,
// an error is returned.
func (c *NetClient) ReadInNetworkFromTopology(topology *models.Topology) error {
	c.log.Debugf("Reading network information from the PKI")

	mixes, err := helpers.GetMixesPKI(topology.MixNodes)
	if err != nil {
		c.log.Errorf("Error while reading mixes from PKI: %v", err)
		return err
	}
	c.Network.Mixes = mixes

	clients, err := helpers.GetClientPKI(topology.MixProviderNodes)
	if err != nil {
		c.log.Errorf("Error while reading clients from PKI: %v", err)
		return err
	}
	c.Network.Clients = clients

	return nil
}

// TODO: make it variable, perhaps choose provider with least number of clients? or by preference?
// But for now just get the first provider on the list
func providerFromTopology(initialTopology *models.Topology) (config.MixConfig, error) {
	if initialTopology == nil || initialTopology.MixProviderNodes == nil || len(initialTopology.MixProviderNodes) == 0 {
		return config.MixConfig{}, errors.New("Invalid topology")
	}

	for _, v := range initialTopology.MixProviderNodes {
		// get the first entry
		return helpers.ProviderPresenceToConfig(v)
	}
	return config.MixConfig{}, errors.New("Unknown state")
}

// NewClient constructor function to create an new client object.
// Returns a new client object or an error, if occurred.
// TODO: temporarily just split the function signature in multiple lines to make the lines shorter,
// however, we should perhaps pass some struct instead like 'clientConfig'
// that would encapsulate all the parameters?
func NewClient(id string,
	host string,
	port string,
	prvKey *sphinx.PrivateKey,
	pubKey *sphinx.PublicKey,
	demoRecipient config.ClientConfig,
) (*NetClient, error) {

	baseLogger, err := logger.New(defaultLogFileLocation, defaultLogLevel, false)
	if err != nil {
		return nil, err
	}

	core := clientcore.NewCryptoClient(prvKey,
		pubKey,
		config.MixConfig{},
		clientcore.NetworkPKI{},
		baseLogger.GetLogger("cryptoClient "+id),
	)

	log := baseLogger.GetLogger(id)

	c := NetClient{id: id,
		host:          host,
		port:          port,
		CryptoClient:  core,
		haltedCh:      make(chan struct{}),
		log:           log,
		demoRecipient: demoRecipient,
	}
	c.config = config.ClientConfig{Id: c.id,
		Host:     c.host,
		Port:     c.port,
		PubKey:   c.GetPublicKey().Bytes(),
		Provider: &c.Provider,
	}

	return &c, nil
}

// NewTestClient constructs a client object, which can be used for testing. The object contains the crypto core
// and the top-level of client, but does not involve networking and starting a listener.
// TODO: similar issue as with 'NewClient' - need to create some config struct with the parameters
func NewTestClient(id string,
	host string,
	port string,
	prvKey *sphinx.PrivateKey,
	pubKey *sphinx.PublicKey,
	provider config.MixConfig,
) (*NetClient, error) {
	baseDisabledLogger, err := logger.New(defaultLogFileLocation, defaultLogLevel, true)
	if err != nil {
		return nil, err
	}
	// this logger can be shared as it will be disabled anyway
	disabledLog := baseDisabledLogger.GetLogger("test")

	core := clientcore.NewCryptoClient(prvKey, pubKey, provider, clientcore.NetworkPKI{}, disabledLog)
	c := NetClient{id: id,
		host:         host,
		port:         port,
		CryptoClient: core,
		log:          disabledLog,
	}
	c.config = config.ClientConfig{Id: c.id,
		Host:     c.host,
		Port:     c.port,
		PubKey:   c.GetPublicKey().Bytes(),
		Provider: &c.Provider,
	}

	return &c, nil
}
