// Copyright 2019 The Loopix-Messaging Authors
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
Package server is used to start local socket listener.

It contains three server implementations:
* gRPC server (TODO)
* TCP socket server
* websocket server (TODO)
*/

package server

import (
	"fmt"
	"github.com/nymtech/nym-mixnet/client"
	"github.com/nymtech/nym-mixnet/client/rpc/tcpsocket"
	"github.com/nymtech/nym-mixnet/client/rpc/types"
	"github.com/nymtech/nym-mixnet/logger"
)


func NewSocketListener(address, typ string, logger *logger.Logger, c *client.NetClient) (types.SocketListener, error) {
	var s types.SocketListener
	var err error
	switch typ {
	case "tcp":
		s = tcpsocket.NewSocketServer(address, logger, c)
	case "grpc":
		panic("NOT IMPLEMENTED")
	case "websocket":
		panic("NOT IMPLEMENTED")
	default:
		err = fmt.Errorf("unknown server type: %s", typ)
	}
	return s, err
}
