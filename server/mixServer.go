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

// Package server implements the mix server.
package server

import (
	"bytes"
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/loopix-messaging/config"
	"github.com/nymtech/loopix-messaging/helpers"
	"github.com/nymtech/loopix-messaging/logging"
	"github.com/nymtech/loopix-messaging/networker"
	"github.com/nymtech/loopix-messaging/node"
	"github.com/nymtech/loopix-messaging/sphinx"
	"github.com/sirupsen/logrus"
)

const (
	metricsInterval = time.Second
)

// TODO: another case of the global logger
var logLocal = logging.PackageLogger()

// TODO: actually remove it in production code. This is only used to have easier access to different debug levels
//nolint: gochecknoinits
func init() {
	// For easier access for modifying logging level,
	logLocal.Logger.SetLevel(logrus.InfoLevel)
}

// MixServerIt is the interface of a mix server.
type MixServerIt interface {
	networker.NetworkServer
	networker.NetworkClient
	GetConfig() config.MixConfig
	Start() error
}

// MixServer is the data of a mix server
type MixServer struct {
	id       string
	host     string
	port     string
	layer    int
	listener *net.TCPListener
	*node.Mix

	config  config.MixConfig
	metrics *metrics
}

type metrics struct {
	sync.Mutex
	sentMessages map[string]uint
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
			logLocal.Errorf("Failed to send metrics: %v", err)
		}
	}(metricsCopy)
}

func newMetrics() *metrics {
	return &metrics{
		sentMessages: make(map[string]uint),
	}
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
	logLocal.Infof("%s: Received new sphinx packet", m.id)

	c := make(chan []byte)
	cAdr := make(chan sphinx.Hop)
	cFlag := make(chan []byte)
	errCh := make(chan error)

	go m.ProcessPacket(packet, c, cAdr, cFlag, errCh)
	dePacket := <-c
	nextHop := <-cAdr
	flag := <-cFlag
	err := <-errCh

	if err != nil {
		return err
	}

	if bytes.Equal(flag, sphinx.RelayFlag) {
		if err := m.forwardPacket(dePacket, nextHop.Address); err != nil {
			return err
		}
		// add it only if we didn't return an error
		m.metrics.addMessage(nextHop.Address)
	} else {
		logLocal.Info("Packet has non-forward flag. Packet dropped")
	}
	return nil
}

func (m *MixServer) forwardPacket(sphinxPacket []byte, address string) error {
	packetBytes, err := config.WrapWithFlag(config.CommFlag, sphinxPacket)
	if err != nil {
		return err
	}
	err = m.send(packetBytes, address)
	if err != nil {
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

	_, err = conn.Write(packet)
	if err != nil {
		return err
	}
	return nil
}

func (m *MixServer) run() {

	defer m.listener.Close()
	finish := make(chan struct{})

	go m.startSendingMetrics(finish)

	go func() {
		logLocal.Infof("Listening on %s", m.host+":"+m.port)
		m.listenForIncomingConnections()
	}()

	<-finish
}

func (m *MixServer) startSendingMetrics(finishCh chan struct{}) {
	ticker := time.NewTicker(metricsInterval)
	for {
		select {
		case <-ticker.C:
			m.metrics.sendToDirectoryServer()
			m.metrics.reset()
		case <-finishCh:
			return
		}
	}
}

func (m *MixServer) listenForIncomingConnections() {
	for {
		conn, err := m.listener.Accept()

		if err != nil {
			logLocal.WithError(err).Error(err)
		} else {
			logLocal.Infof("Received connection from %s", conn.RemoteAddr())
			errs := make(chan error, 1)
			go m.handleConnection(conn, errs)
			err = <-errs
			if err != nil {
				logLocal.WithError(err).Error(err)
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
	err = proto.Unmarshal(buff[:reqLen], &packet)
	if err != nil {
		errs <- err
	}

	switch {
	case bytes.Equal(packet.Flag, config.CommFlag):
		err = m.receivedPacket(packet.Data)
		if err != nil {
			errs <- err
		}
	default:
		logLocal.Infof("Packet flag %s not recognised. Packet dropped", packet.Flag)
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
	mix := node.NewMix(prvKey, pubKey)
	mixServer := MixServer{id: id,
		host:    host,
		port:    port,
		Mix:     mix,
		layer:   layer,
		metrics: newMetrics(),
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
