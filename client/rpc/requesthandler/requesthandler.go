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

package requesthandler

import (
	"github.com/nymtech/nym-mixnet/client"
	"github.com/nymtech/nym-mixnet/client/rpc/types"
)

func returnSendError() *types.Response {
	return &types.Response{
		Value: &types.Response_Send{
			Send: &types.ResponseSendMessage{},
		},
	}
}

func HandleSendMessage(req *types.Request_Send, c *client.NetClient) *types.Response {
	sreq := req.Send
	if req == nil || sreq == nil || sreq.Message == nil || sreq.Recipient == nil {
		return returnSendError()
	}
	if err := c.SendMessage(sreq.Message, *sreq.Recipient); err != nil {
		return &types.Response{
			Value: &types.Response_Exception{
				Exception: &types.ResponseException{
					Error: err.Error(),
				},
			},
		}
	}
	return returnSendError()
}

func HandleFetchMessages(req *types.Request_Fetch, c *client.NetClient) *types.Response {
	msgs := c.GetReceivedMessages()
	return &types.Response{
		Value: &types.Response_Fetch{
			Fetch: &types.ResponseFetchMessages{
				Messages: msgs,
			},
		},
	}
}

func HandleGetClients(req *types.Request_Clients, c *client.NetClient) *types.Response {
	clients := c.GetAllPossibleRecipients()

	return &types.Response{
		Value: &types.Response_Clients{
			Clients: &types.ResponseGetClients{
				Clients: clients,
			},
		},
	}
}

func HandleFlush(req *types.Request_Flush) *types.Response {
	return &types.Response{
		Value: &types.Response_Flush{
			Flush: &types.ResponseFlush{},
		},
	}
}

func HandleInvalidRequest() *types.Response {
	return &types.Response{
		Value: &types.Response_Exception{
			Exception: &types.ResponseException{
				Error: "Invalid server request",
			},
		},
	}
}
