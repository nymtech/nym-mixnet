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

// Package mixnode implements the mix server.
package mixnode

import (
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/flags"
	"github.com/nymtech/loopix-messaging/helpers"
	"github.com/nymtech/loopix-messaging/logger"
	"github.com/nymtech/loopix-messaging/networker"
	"github.com/nymtech/loopix-messaging/node"
	"github.com/nymtech/loopix-messaging/sphinx"
	"github.com/sirupsen/logrus"
)

const (
	metricsInterval  = time.Second
	presenceInterval = 2 * time.Second

	// Below should be moved to a config file once we have it
	// logFileLocation can either point to some valid file to which all log data should be written
	// or if left an empty string, stdout will be used instead
	defaultLogFileLocation = ""
	// considering we are under heavy development and nowhere near production level, log EVERYTHING
	defaultLogLevel = "trace"
)

// MixServerIt is the interface of a mix server.
type MixServerIt interface {
	networker.NetworkServer
	networker.NetworkClient
	GetConfig() config.MixConfig
	Start() error
}

// MixServer is the data of a mix server
type MixServer struct {
	*node.Mix
	id       string
	host     string
	port     string
	layer    int
	listener *net.TCPListener
	config   config.MixConfig
	metrics  *metrics
	haltedCh chan struct{}
	haltOnce sync.Once
	log      *logrus.Logger
}

type metrics struct {
	sync.Mutex
	sentMessages map[string]uint
	log          *logrus.Logger
}

func (m *metrics) reset() {
	m.Lock()
	defer m.Unlock()
	m.sentMessages = make(map[string]uint)
}

func (m *metrics) addMessage(hopAddress string) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.sentMessages[hopAddress]; ok {
		m.sentMessages[hopAddress]++
	} else {
		m.sentMessages[hopAddress] = 1
	}
}

func (m *metrics) sendToDirectoryServer() {
	m.Lock()
	defer m.Unlock()
	// send the data in a new goroutine so we wouldn't block if there were issues in sending the data
	metricsCopy := make(map[string]uint)
	for k, v := range m.sentMessages {
		metricsCopy[k] = v
	}
	go func(metricsCopy map[string]uint) {
		if err := helpers.SendMixMetrics(metricsCopy); err != nil {
			m.log.Errorf("Failed to send metrics: %v", err)
		}
	}(metricsCopy)
}

func newMetrics(log *logrus.Logger) *metrics {
	return &metrics{
		log:          log,
		sentMessages: make(map[string]uint),
	}
}

// Wait waits till the mixserver is terminated for any reason.
func (m *MixServer) Wait() {
	<-m.haltedCh
}

// Shutdown cleanly shuts down a given mixserver instance.
func (m *MixServer) Shutdown() {
	m.haltOnce.Do(func() { m.halt() })
}

// calls any required cleanup code
func (m *MixServer) halt() {
	m.log.Info("Starting graceful shutdown")
	// close any listeners, free resources, etc
	// possibly send "remove presence" message

	close(m.haltedCh)
}

// Start runs a mix server
func (m *MixServer) Start() error {
	defer m.run()
	return nil
}

// GetConfig returns the config of the given mix server
func (m *MixServer) GetConfig() config.MixConfig {
	return m.config
}

func (m *MixServer) receivedPacket(packet []byte) error {
	m.log.Infof("%s: Received new sphinx packet", m.id)

	res := m.ProcessPacket(packet)
	dePacket := res.PacketData()
	nextHop := res.NextHop()
	flag := res.Flag()
	if err := res.Err(); err != nil {
		return err
	}

	if flag == flags.RelayFlag {
		if err := m.forwardPacket(dePacket, nextHop.Address); err != nil {
			return err
		}
		// add it only if we didn't return an error
		m.metrics.addMessage(nextHop.Address)
	} else {
		m.log.Info("Packet has non-forward flag. Packet dropped")
	}
	return nil
}

func (m *MixServer) forwardPacket(sphinxPacket []byte, address string) error {
	packetBytes, err := config.WrapWithFlag(flags.CommFlag, sphinxPacket)
	if err != nil {
		return err
	}
	if err := m.send(packetBytes, address); err != nil {
		return err
	}

	return nil
}

func (m *MixServer) send(packet []byte, address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.Write(packet); err != nil {
		return err
	}
	return nil
}

func (m *MixServer) run() {
	defer m.listener.Close()

	// go m.startSendingMetrics()
	go m.startSendingPresence()

	go func() {
		m.log.Infof("Listening on %s", m.host+":"+m.port)
		m.listenForIncomingConnections()
	}()

	m.Wait()
}

func (m *MixServer) startSendingMetrics() {
	ticker := time.NewTicker(metricsInterval)
	for {
		select {
		case <-ticker.C:
			m.metrics.sendToDirectoryServer()
			m.metrics.reset()
		case <-m.haltedCh:
			return
		}
	}
}

func (m *MixServer) startSendingPresence() {
	ticker := time.NewTicker(presenceInterval)
	for {
		select {
		case <-ticker.C:
			if err := helpers.RegisterPresence(m.id, m.GetPublicKey(), m.layer); err != nil {
				m.log.Errorf("Failed to register presence: %v", err)
			}
		case <-m.haltedCh:
			return
		}
	}
}

func (m *MixServer) listenForIncomingConnections() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			m.log.Errorf("Error when listening for incoming connection: %v", err)
		} else {
			m.log.Infof("Received connection from %s", conn.RemoteAddr())
			errs := make(chan error, 1)
			go m.handleConnection(conn, errs)
			err = <-errs
			if err != nil {
				m.log.Errorf("Error when listening for incoming connection: %v", err)
			}
		}
	}
}

func (m *MixServer) handleConnection(conn net.Conn, errs chan<- error) {
	defer conn.Close()

	buff := make([]byte, 1024)
	reqLen, err := conn.Read(buff)
	if err != nil {
		errs <- err
	}

	var packet config.GeneralPacket
	if err := proto.Unmarshal(buff[:reqLen], &packet); err != nil {
		errs <- err
	}

	switch flags.PacketTypeFlagFromBytes(packet.Flag) {
	case flags.CommFlag:
		if err := m.receivedPacket(packet.Data); err != nil {
			errs <- err
		}
	default:
		m.log.Infof("Packet flag %s not recognised. Packet dropped", packet.Flag)
		errs <- nil
	}
	errs <- nil
}

// NewMixServer constructor
// TODO: Identical case to 'NewClient'
// TODO: remove pkiPath once it becomes completely replaced with the directory server
func NewMixServer(id string,
	host string,
	port string,
	prvKey *sphinx.PrivateKey,
	pubKey *sphinx.PublicKey,
	pkiPath string,
	layer int,
) (*MixServer, error) {

	baseLogger, err := logger.New(defaultLogFileLocation, defaultLogLevel, false)
	if err != nil {
		return nil, err
	}

	log := baseLogger.GetLogger(id)

	mix := node.NewMix(prvKey, pubKey)
	mixServer := MixServer{id: id,
		host:     host,
		port:     port,
		Mix:      mix,
		layer:    layer,
		metrics:  newMetrics(baseLogger.GetLogger("metrics " + id)),
		haltedCh: make(chan struct{}),
		log:      log,
	}
	mixServer.config = config.MixConfig{Id: mixServer.id,
		Host:   mixServer.host,
		Port:   mixServer.port,
		PubKey: mixServer.GetPublicKey().Bytes(),
	}

	configBytes, err := proto.Marshal(&mixServer.config)
	if err != nil {
		return nil, err
	}

	if err := helpers.AddToDatabase(pkiPath, "Pki", mixServer.id, "Mix", configBytes); err != nil {
		return nil, err
	}

	if err := helpers.RegisterPresence(mixServer.id, mixServer.GetPublicKey(), layer); err != nil {
		return nil, err
	}

	addr, err := helpers.ResolveTCPAddress(mixServer.host, mixServer.port)

	if err != nil {
		return nil, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	mixServer.listener = listener

	return &mixServer, nil
}

func CreateTestMixnode() (*MixServer, error) {
	priv, pub, err := sphinx.GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	baseDisabledLogger, err := logger.New(defaultLogFileLocation, defaultLogLevel, true)
	if err != nil {
		return nil, err
	}
	// this logger can be shared as it will be disabled anyway
	disabledLog := baseDisabledLogger.GetLogger("test")

	node := node.NewMix(priv, pub)
	mix := MixServer{host: "localhost", port: "9995", Mix: node, log: disabledLog}
	mix.config = config.MixConfig{Id: mix.id,
		Host:   mix.host,
		Port:   mix.port,
		PubKey: mix.GetPublicKey().Bytes(),
	}
	addr, err := helpers.ResolveTCPAddress(mix.host, mix.port)
	if err != nil {
		return nil, err
	}

	mix.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &mix, nil
}
