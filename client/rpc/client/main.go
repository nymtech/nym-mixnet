// THIS IS ENTIRELY FOR TESTING PURPOSES TO DEMONSTRATE THAT YOU CAN COMMUNICATE WITH THE CLIENT ON TCP SOCKET

package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/nymtech/nym-mixnet/client/rpc/types"
	"net/url"
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

	// TCP_SOCKET:
	//conn, err := net.Dial("tcp", "127.0.0.1:9000")
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = utils.WriteProtoMessage(msg, conn)
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = utils.WriteProtoMessage(flushMsg, conn)
	//if err != nil {
	//	panic(err)
	//}
	//
	//time.Sleep(time.Second)
	//
	//res := &types.Response{}
	//err = utils.ReadProtoMessage(res, conn)
	//if err != nil {
	//	panic(err)
	//}
	//
	//fmt.Printf("%v", res)

	_ = flushMsg
	// WEB_SOCKET:
	u := url.URL{
		Scheme: "ws",
		Host:   "127.0.0.1:9000",
		Path:   "/mix",
	}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		panic(err)
	}

	defer c.Close()

	msgB, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	err = c.WriteMessage(websocket.BinaryMessage, msgB)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second)

	resT, resB, err := c.ReadMessage()
	if err != nil {
		panic(err)
	}

	res := &types.Response{}
	err = proto.Unmarshal(resB, res)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Res type: %v (expect: %v)\n %v", resT, websocket.BinaryMessage, res)

}
