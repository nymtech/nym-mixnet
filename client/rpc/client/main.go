// THIS IS ENTIRELY FOR TESTING PURPOSES TO DEMONSTRATE THAT YOU CAN COMMUNICATE WITH THE CLIENT ON TCP SOCKET

package main

import (
	"fmt"
	"github.com/nymtech/nym-mixnet/client/rpc/types"
	"github.com/nymtech/nym-mixnet/client/rpc/utils"
	"net"
	"time"
)

// u7UtjC3...

func main() {
	//rec_key := "1I1XFLNq9fIP7gDcmJZNH6GtCk5r9-wb3Ay_fZa9fnI="
	//k := base64.URLEncoding.DecodeString(rec_key)
	//
	//
	//recipient := config.ClientConfig{
	//	Id:                   rec_key,
	//	Host:                 "",
	//	Port:                 "",
	//	PubKey:               k,
	//	Provider:             config.MixConfig{
	//		Id:                   "XiVE6xA10xFkAwfIQuBDc_JRXWerL0Pcqi7DipEUeTE=",
	//		Host:                 "3.8.176.11",
	//		Port:                 "1789",
	//		PubKey:               base64.URLEncoding.DecodeString("XiVE6xA10xFkAwfIQuBDc_JRXWerL0Pcqi7DipEUeTE="),
	//	}
	//}
	//
	//content :=

	msg := &types.Request{
		Value: &types.Request_Fetch{
			Fetch: &types.RequestFetchMessages{

			},
		},
	}

	flushMsg := &types.Request{
		Value: &types.Request_Flush{
			Flush: &types.RequestFlush{},
		},
	}

	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		panic(err)
	}

	err = utils.WriteProtoMessage(msg, conn)
	if err != nil {
		panic(err)
	}

	err = utils.WriteProtoMessage(flushMsg, conn)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second)

	res := &types.Response{}
	err = utils.ReadProtoMessage(res, conn)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%v", res)
}
