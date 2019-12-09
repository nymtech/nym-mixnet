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

package websocket

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/nymtech/nym-mixnet/client"
	"github.com/nymtech/nym-mixnet/client/rpc/requesthandler"
	"github.com/nymtech/nym-mixnet/client/rpc/types"
	"github.com/nymtech/nym-mixnet/logger"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 40 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 30 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 2048
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096, // TODO: is this enough?
	WriteBufferSize: 4096,
	// only allow requests from local
	CheckOrigin: func(r *http.Request) bool {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return false
		}
		return net.ParseIP(ip).IsLoopback()
	},
}

var jsonPbUnmarshaler = jsonpb.Unmarshaler{
	AllowUnknownFields: false,
	AnyResolver:        nil,
}

var jsonPbMarshaler = jsonpb.Marshaler{
	EnumsAsInts:  false,
	EmitDefaults: false,
	Indent:       "  ",
	OrigName:     false,
	AnyResolver:  nil,
}

type SocketServer struct {
	client   *client.NetClient
	haltedCh chan struct{}
	haltOnce sync.Once
	log      *logrus.Logger
	srv      *http.Server
	address  string
}

func (s *SocketServer) handleRequest(req *types.Request) *types.Response {
	switch r := req.Value.(type) {
	case *types.Request_Send:
		s.log.Info("Send request")
		return requesthandler.HandleSendMessage(r, s.client)
	case *types.Request_Fetch:
		s.log.Info("Fetch request")
		return requesthandler.HandleFetchMessages(r, s.client)
	case *types.Request_Clients:
		s.log.Info("Clients request")
		return requesthandler.HandleGetClients(r, s.client)
	case *types.Request_Details:
		s.log.Info("Details request")
		return requesthandler.HandleOwnDetails(r, s.client)
	//case *types.Request_Flush:
	//	return requesthandler.HandleFlush(r) // doesn't do anything
	default:
		s.log.Info("Unknown request")
		return requesthandler.HandleInvalidRequest()
	}
}

func (s *SocketServer) handleBinaryMixRequest(msg []byte) (int, []byte, error) {
	req := &types.Request{}
	if err := proto.Unmarshal(msg, req); err != nil {
		return websocket.BinaryMessage, nil, fmt.Errorf("failed to unmarshal send message: %v", err)
	}
	res := s.handleRequest(req)
	resB, err := proto.Marshal(res)
	if err != nil {
		return websocket.BinaryMessage, nil, fmt.Errorf("failed to marshal response: %v", err)
	}
	return websocket.BinaryMessage, resB, nil
}

func (s *SocketServer) handleTextMixRequest(msg []byte) (int, []byte, error) {
	s.log.Debugf("Received json request: %v", string(msg))
	req := &types.Request{}
	if err := jsonPbUnmarshaler.Unmarshal(bytes.NewBuffer(msg), req); err != nil {
		return websocket.TextMessage, nil, fmt.Errorf("failed to unmarshal send message: %v", err)
	}
	res := s.handleRequest(req)
	jsonRes := bytes.NewBufferString("") // TODO: I doubt that's the best way for doing this
	if err := jsonPbMarshaler.Marshal(jsonRes, res); err != nil {
		return websocket.TextMessage, nil, fmt.Errorf("failed to marshal response: %v", err)
	}
	return websocket.TextMessage, jsonRes.Bytes(), nil
}

func (s *SocketServer) handleMixRequest(reqTyp int, req []byte) (int, []byte, error) {
	switch reqTyp {
	case websocket.BinaryMessage:
		return s.handleBinaryMixRequest(req)
	case websocket.TextMessage:
		return s.handleTextMixRequest(req)
	default:
		// TODO: is it the proper way of using 'CloseMessage'?
		return websocket.CloseMessage, []byte{}, fmt.Errorf("invalid request type: %v", reqTyp)
	}

	// TODO: do rest of below need to be explicitly handled?
	/*
		// TextMessage denotes a text data message. The text message payload is
		// interpreted as UTF-8 encoded text data.
		TextMessage = 1

		// BinaryMessage denotes a binary data message.
		BinaryMessage = 2

		// CloseMessage denotes a close control message. The optional message
		// payload contains a numeric code and text. Use the FormatCloseMessage
		// function to format a close message payload.
		CloseMessage = 8

		// PingMessage denotes a ping control message. The optional message payload
		// is UTF-8 encoded text.
		PingMessage = 9

		// PongMessage denotes a pong control message. The optional message payload
		// is UTF-8 encoded text.
		PongMessage = 10
	*/

}

func (s *SocketServer) pingConnection(ws *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				s.log.Errorf("ping error: %v", err)
			}
		case <-done:
			return
		}
	}
}

func (s *SocketServer) serveMixClient(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Errorf("failed to upgrade: %v", err)
		return
	}

	pingCh := make(chan struct{})
	defer func() {
		close(pingCh)
		c.Close()
	}()

	c.SetReadLimit(maxMessageSize)
	if err := c.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		s.log.Errorf("failed to set write deadline: %v", err)
		return
	}
	if err := c.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		s.log.Errorf("failed to set read deadline: %v", err)
		return
	}

	go s.pingConnection(c, pingCh)
	c.SetPongHandler(func(string) error {
		if err := c.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
			return err
		}
		return c.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		reqTyp, req, err := c.ReadMessage()
		if err != nil {
			s.log.Errorf("failed to read send message: %v", err)
			break
		}

		resTyp, res, err := s.handleMixRequest(reqTyp, req)
		if err != nil {
			s.log.Errorf("failed to handle mix request: %v", err)
			break
		}
		if reqTyp == websocket.TextMessage {
			s.log.Debugf("sending json reply: %v", string(res))
		}
		if err := c.WriteMessage(resTyp, res); err != nil {
			s.log.Errorf("failed to send reply: %v", err)
			break
		}
	}
}

// TODO: serve html file that sends proper proto requests, though I guess that will be done by the electron app
func (s *SocketServer) serveHome(w http.ResponseWriter, r *http.Request) {
	s.log.Info(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	//http.ServeFile(w, r, "client/rpc/websocket/home.html")
}

func (s *SocketServer) Start() error {
	if err := s.client.Start(); err != nil {
		return err
	}

	http.HandleFunc("/", s.serveHome)
	http.HandleFunc("/mix", s.serveMixClient)

	go func() {
		if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
			s.log.Fatalf("Failed to listen on websocket: %v", err)
		}
	}()

	return nil
}

func (s *SocketServer) Shutdown() {
	s.haltOnce.Do(func() { s.halt() })
}

// calls any required cleanup code
func (s *SocketServer) halt() {
	s.log.Info("Starting graceful shutdown")

	if err := s.srv.Shutdown(context.TODO()); err != nil {
		s.log.Errorf("failed to cleanly shutdown http server: %v", err)
	}
	s.client.Shutdown()

	close(s.haltedCh)
}

func (s *SocketServer) Wait() {
	<-s.haltedCh
}

func NewSocketServer(address string, logger *logger.Logger, c *client.NetClient) types.SocketListener {
	s := &SocketServer{
		address: address,
		log:     logger.GetLogger("websocket-server"),
		client:  c,
		srv: &http.Server{
			Addr: address,
		},
	}

	return s
}
