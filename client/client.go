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
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"os"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/golang/protobuf/proto"
	"github.com/nymtech/directory-server/models"
	clientConfig "github.com/nymtech/loopix-messaging/client/config"
	"github.com/nymtech/loopix-messaging/clientcore"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/constants"
	"github.com/nymtech/loopix-messaging/flags"
	"github.com/nymtech/loopix-messaging/helpers"
	"github.com/nymtech/loopix-messaging/helpers/topology"
	"github.com/nymtech/loopix-messaging/logger"
	"github.com/nymtech/loopix-messaging/networker"
	"github.com/nymtech/loopix-messaging/sphinx"
	"github.com/sirupsen/logrus"
)

const (
	loopLoad = "LoopCoverMessage"
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
	// TODO: somehow rename or completely remove config.ClientConfig because it's waaaay too confusing right now
	cfg      *clientConfig.Config
	config   config.ClientConfig
	token    []byte // TODO: combine with the 'Provider' field considering it's provider specific
	outQueue chan []byte
	haltedCh chan struct{}
	haltOnce sync.Once
	log      *logrus.Logger
}

// OutQueue returns a reference to the client's outQueue. It's a queue
// which holds outgoing packets while their order is randomised.
func (c *NetClient) OutQueue() chan<- []byte {
	return c.outQueue
}

// Start reads the network and users information from the topology
// and starts the listening server. Returns an error
// signalling whenever any operation was unsuccessful.
func (c *NetClient) Start() error {

	c.outQueue = make(chan []byte)

	initialTopology, err := topology.GetNetworkTopology(c.cfg.Client.DirectoryServerTopologyEndpoint)
	if err != nil {
		return err
	}
	if err := c.ReadInNetworkFromTopology(initialTopology); err != nil {
		return err
	}

	if _, ok := initialTopology.MixProviderNodes[c.cfg.Client.ProviderID]; !ok {
		return fmt.Errorf("specified provider does not seem to be online: %v", c.cfg.Client.ProviderID)
	}
	provider, err := topology.ProviderPresenceToConfig(initialTopology.MixProviderNodes[c.cfg.Client.ProviderID])
	// provider, err := providerFromTopology(initialTopology)
	if err != nil {
		return err
	}
	c.Provider = provider

	for {
		if err := c.sendRegisterMessageToProvider(); err != nil {
			c.log.Errorf("Error during registration to provider: %v", err)
			time.Sleep(5 * time.Second)
		} else {
			c.log.Debug("Registration done!")
			break
		}
	}

	// before we start traffic, we must wait until registration of some client reaches directory server
	for {
		initialTopology, err := topology.GetNetworkTopology(c.cfg.Client.DirectoryServerTopologyEndpoint)
		if err != nil {
			return err
		}
		if err := c.ReadInNetworkFromTopology(initialTopology); err != nil {
			return err
		}
		if len(c.Network.Clients) > 0 {
			break
		}
		c.log.Debug("No registered clients available. Waiting for a second before retrying.")
		time.Sleep(time.Second)
	}
	c.log.Info("Obtained valid network topology")

	c.startTraffic()

	// if public is zeroed, it means either something went terribly wrong
	// or we are running a benchmark. In either case, we don't want to be accepting inputs
	if !helpers.IsZeroElement(c.GetPublicKey()) {
		go c.startInputRoutine()
	}
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

func toChoosable(client config.ClientConfig) string {
	b64Key := base64.URLEncoding.EncodeToString(client.PubKey)
	b64ProviderKey := base64.URLEncoding.EncodeToString(client.Provider.PubKey)
	// while normally it's unsafe to directly index string, it's safe here
	// as id is guaranteed to only hold ascii characters due to being b64 encoding of the key
	return fmt.Sprintf("ID: %s\t@[Provider]\t%s", b64Key, b64ProviderKey)
}

func makeChoosables(clients []config.ClientConfig) (map[string]config.ClientConfig, []string) {
	choosableClients := make(map[string]config.ClientConfig)
	options := make([]string, len(clients))
	for i, client := range clients {
		choosableClient := toChoosable(client)
		options[i] = choosableClient
		choosableClients[choosableClient] = client // basically a mapping from the string back to original struct
	}
	return choosableClients, options
}

func shouldStopInput(msg string) bool {
	quitMessages := []string{
		"quit",
		"/q",
		":q",
		":q!",
		"exit",
	}

	for _, qm := range quitMessages {
		if qm == msg {
			return true
		}
	}

	return false
}

func (c *NetClient) startInputRoutine() {

	choosableClients, choosableOptions := makeChoosables(c.Network.Clients)

	var chosenClientOption string
	prompt := &survey.Select{
		Message: "Choose another client to communicate with:",
		Options: choosableOptions,
	}
	if err := survey.AskOne(prompt, &chosenClientOption, nil); err == terminal.InterruptErr {
		// we got an interrupt so we're killing whole client
		c.log.Warningf("Received an interrupt - stopping entire client")
		c.Shutdown()
		return
	}

	chosenClient := choosableClients[chosenClientOption]

	for {
		select {
		case <-c.haltedCh:
			return
		default:
		}
		messageToSend := ""
		b64Key := base64.URLEncoding.EncodeToString(chosenClient.GetPubKey())
		prompt := &survey.Input{
			Message: fmt.Sprintf("Type in a message to send to %s...", b64Key),
		}
		if err := survey.AskOne(prompt, &messageToSend); err == terminal.InterruptErr {
			// we got an interrupt so we're killing whole client
			c.log.Warningf("Received an interrupt - stopping entire client")
			c.Shutdown()
			return
		}
		if shouldStopInput(messageToSend) {
			c.log.Warningf("Received a stop signal. Stopping the input routine")
			return
		}

		c.log.Infof("Sending: %v to %v", messageToSend, chosenClient.GetId())
		if err := c.SendMessage(messageToSend, chosenClient); err != nil {
			c.log.Errorf("Failed to send %v to %x...: %v", messageToSend, chosenClient.GetPubKey()[:8], err)
		}
	}
}

func (c *NetClient) checkTopology() error {
	if c.Network.ShouldUpdate() {
		newTopology, err := topology.GetNetworkTopology(c.cfg.Client.DirectoryServerTopologyEndpoint)
		if err != nil {
			c.log.Errorf("error while reading network topology: %v", err)
			return err
		}
		if err := c.ReadInNetworkFromTopology(newTopology); err != nil {
			c.log.Errorf("error while trying to update topology: %v", err)
			return err
		}
	}
	return nil
}

// SendMessage responsible for sending a real message. Takes as input the message string
// and the public information about the destination.
func (c *NetClient) SendMessage(message string, recipient config.ClientConfig) error {
	// before we send a message, ensure our topology is up to date
	if err := c.checkTopology(); err != nil {
		c.log.Errorf("error in updating topology: %v", err)
		return err
	}
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
// Otherwise it returns the response sent by server
func (c *NetClient) send(packet []byte, host string, port string) (config.ProviderResponse, error) {

	conn, err := net.Dial("tcp", host+":"+port)

	if err != nil {
		c.log.Errorf("Error in send - dial returned an error: %v", err)
		return config.ProviderResponse{}, err
	}
	defer conn.Close()

	if _, err := conn.Write(packet); err != nil {
		c.log.Errorf("Failed to write to connection: %v", err)
		return config.ProviderResponse{}, err
	}

	buff, err := ioutil.ReadAll(conn)
	if err != nil {
		c.log.Errorf("Failed to read response: %v", err)
		return config.ProviderResponse{}, err
	}

	var resPacket config.ProviderResponse
	if err = proto.Unmarshal(buff, &resPacket); err != nil {
		c.log.Errorf("Error while unmarshalling received packet: %v", err)
		return config.ProviderResponse{}, err
	}

	return resPacket, nil
}

// RegisterToken stores the authentication token received from the provider
func (c *NetClient) registerToken(token []byte) {
	c.token = token
	c.log.Debugf("Registered token %s", c.token)
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

func (c *NetClient) startTraffic() {
	go func() {
		err := c.controlOutQueue()
		if err != nil {
			c.log.Fatalf("Error in the controller of the outgoing packets queue. Possible security threat.: %v", err)
		}
	}()

	if c.cfg.Debug.LoopCoverTrafficRate > 0.0 {
		c.turnOnLoopCoverTraffic()
	}

	if c.cfg.Debug.FetchMessageRate > 0.0 {
		go func() {
			c.controlMessagingFetching()
		}()
	}
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

	response, err := c.send(pktBytes, c.Provider.Host, c.Provider.Port)
	if err != nil {
		c.log.Errorf("Error in register provider - send registration packet returned an error: %v", err)
		return err
	}

	packets, err := config.UnmarshalProviderResponse(response)
	if err != nil || len(packets) != 1 {
		c.log.Errorf("error in register provider - failed to unmarshal response: %v", err)
	}

	c.registerToken(packets[0].Data)

	return nil
}

// GetMessagesFromProvider allows to fetch messages from the inbox stored by the
// provider. The client sends a pull packet to the provider, along with
// the authentication token. An error is returned if occurred.
func (c *NetClient) getMessagesFromProvider() error {
	pullRqs := config.PullRequest{ClientPublicKey: c.GetPublicKey().Bytes(), Token: c.token}
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

	response, err := c.send(pktBytes, c.Provider.Host, c.Provider.Port)
	if err != nil {
		return err
	}

	packets, err := config.UnmarshalProviderResponse(response)
	if err != nil {
		c.log.Errorf("error in register provider - failed to unmarshal response: %v", err)
	}
	for _, packet := range packets {
		packetData, err := c.processPacket(packet.Data)
		if err != nil {
			c.log.Errorf("Error in processing received packet: %v", err)
		}
		packetDataStr := string(packetData)
		switch packetDataStr {
		case loopLoad:
			c.log.Debugf("Received loop cover message %v", packetDataStr)
		default:
			fmt.Fprintf(os.Stdout, "\nReceived: %s\n?> ", packetDataStr) // print to stdout regardless of logging location
			c.log.Infof("Received new message: %v", packetDataStr)
		}
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
			response, err := c.send(realPacket, c.Provider.Host, c.Provider.Port)
			if err != nil {
				c.log.Errorf("Could not send real packet: %v", err)
			}
			c.log.Debugf("Real packet was sent")
			c.log.Debugf("Received response: %v", response)
		default:
			if !c.cfg.Debug.RateCompliantCoverMessagesDisabled {
				dummyPacket, err := c.createLoopCoverMessage()
				if err != nil {
					return err
				}
				response, err := c.send(dummyPacket, c.Provider.Host, c.Provider.Port)
				if err != nil {
					c.log.Errorf("Could not send dummy packet: %v", err)
				}
				c.log.Debugf("Dummy packet was sent")
				c.log.Debugf("Received response: %v", response)
			}
		}
		err := delayBeforeContinue(c.cfg.Debug.MessageSendingRate)
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
		err := delayBeforeContinue(c.cfg.Debug.FetchMessageRate)
		if err != nil {
			c.log.Errorf("Error in ControlMessagingFetching - generating random exp. value failed: %v", err)
		}
	}
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
		response, err := c.send(loopPacket, c.Provider.Host, c.Provider.Port)
		if err != nil {
			c.log.Errorf("Could not send loop cover traffic message: %v", err)
			return err
		}
		c.log.Debugf("Loop message sent")
		c.log.Debugf("Received response: %v", response)

		if err := delayBeforeContinue(c.cfg.Debug.LoopCoverTrafficRate); err != nil {
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

// ReadInNetworkFromTopology reads in the public information about active mixes
// from the topology and stores them locally. In case
// the connection or fetching data from the PKI went wrong,
// an error is returned.
func (c *NetClient) ReadInNetworkFromTopology(topologyData *models.Topology) error {
	c.log.Debugf("Reading network information from the PKI")

	mixes, err := topology.GetMixesPKI(topologyData.MixNodes)
	if err != nil {
		c.log.Errorf("error while reading mixes from PKI: %v", err)
		return err
	}
	clients, err := topology.GetClientPKI(topologyData.MixProviderNodes)
	if err != nil {
		c.log.Errorf("error while reading clients from PKI: %v", err)
		return err
	}

	c.Network.UpdateNetwork(mixes, clients)

	return nil
}

// TODO: make it variable, perhaps choose provider with least number of clients? or by preference?
// But for now just get the first provider on the list
func providerFromTopology(initialTopology *models.Topology) (config.MixConfig, error) {
	if initialTopology == nil || initialTopology.MixProviderNodes == nil || len(initialTopology.MixProviderNodes) == 0 {
		return config.MixConfig{}, errors.New("invalid topology")
	}

	for _, v := range initialTopology.MixProviderNodes {
		// get the first entry
		return topology.ProviderPresenceToConfig(v)
	}
	return config.MixConfig{}, errors.New("unknown state")
}

// NewClient constructor function to create an new client object.
// Returns a new client object or an error, if occurred.
func NewClient(cfg *clientConfig.Config) (*NetClient, error) {

	baseLogger, err := logger.New(cfg.Logging.File, cfg.Logging.Level, cfg.Logging.Disable)
	if err != nil {
		return nil, err
	}

	prvKey := new(sphinx.PrivateKey)
	pubKey := new(sphinx.PublicKey)
	if err := helpers.FromPEMFile(prvKey, cfg.Client.PrivateKeyFile(), constants.PrivateKeyPEMType); err != nil {
		return nil, fmt.Errorf("Failed to load the private key: %v", err)
	}

	if err := helpers.FromPEMFile(pubKey, cfg.Client.PublicKeyFile(), constants.PublicKeyPEMType); err != nil {
		return nil, fmt.Errorf("Failed to load the public key: %v", err)
	}

	core := clientcore.NewCryptoClient(prvKey,
		pubKey,
		config.MixConfig{},
		clientcore.NetworkPKI{},
		baseLogger.GetLogger("cryptoClient "+cfg.Client.ID),
	)

	log := baseLogger.GetLogger(cfg.Client.ID)

	c := NetClient{CryptoClient: core,
		cfg:      cfg,
		haltedCh: make(chan struct{}),
		log:      log,
	}

	c.log.Infof("Logging level set to %v", c.cfg.Logging.Level)

	b64Key := base64.URLEncoding.EncodeToString(c.GetPublicKey().Bytes())
	c.log.Infof("Our full ID/Public Key is: %v", b64Key)

	c.config = config.ClientConfig{Id: b64Key,
		Host:     "", // TODO: remove
		Port:     "", // TODO: remove
		PubKey:   c.GetPublicKey().Bytes(),
		Provider: &c.Provider,
	}

	return &c, nil
}

// NewTestClient constructs a client object, which can be used for testing. The object contains the crypto core
// and the top-level of client, but does not involve networking and starting a listener.
// TODO: similar issue as with 'NewClient' - need to create some config struct with the parameters
func NewTestClient(cfg *clientConfig.Config, prvKey *sphinx.PrivateKey, pubKey *sphinx.PublicKey) (*NetClient, error) {
	baseDisabledLogger, err := logger.New(cfg.Logging.File, cfg.Logging.Level, cfg.Logging.Disable)
	if err != nil {
		return nil, err
	}

	// this logger can be shared as it will be disabled anyway
	disabledLog := baseDisabledLogger.GetLogger("test")

	core := clientcore.NewCryptoClient(prvKey,
		pubKey,
		config.MixConfig{},
		clientcore.NetworkPKI{},
		disabledLog,
	)

	c := NetClient{CryptoClient: core,
		cfg:      cfg,
		haltedCh: make(chan struct{}),
		log:      disabledLog,
	}

	b64Key := base64.URLEncoding.EncodeToString(c.GetPublicKey().Bytes())

	c.config = config.ClientConfig{Id: b64Key,
		Host:     "", // TODO: remove
		Port:     "", // TODO: remove
		PubKey:   c.GetPublicKey().Bytes(),
		Provider: &c.Provider,
	}

	return &c, nil
}
