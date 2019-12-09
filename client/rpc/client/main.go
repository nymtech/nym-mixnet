package main

import (
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"io"
	"net"
	"time"

	"github.com/nymtech/nym-mixnet/client/rpc/types"
	"github.com/nymtech/nym-mixnet/client/rpc/utils"
	"github.com/nymtech/nym-mixnet/config"
)

func main() {
	fmt.Println("Send and retrieve through mixnet demo")

	conn, err := net.Dial("tcp", "127.0.0.1:9001")
	if err != nil {
		panic(err)
	}

	myDetails := getOwnDetails(conn)

	fmt.Printf("myDetails: %+v\n\n", myDetails)

	sendMessage("foomp", myDetails, conn)

	fmt.Printf("We sent: %+v\n\n", "foomp")

	time.Sleep(time.Second * 1) // give it some time to send to the mixnet

	messages := fetchMessages(conn)

	fmt.Printf("We got back these bytes: %+v\n\n", messages)
	fmt.Printf("We got back this string: %+v\n\n", string(messages[0]))

}

func getOwnDetails(conn net.Conn) *config.ClientConfig {
	me := &types.Request{
		Value: &types.Request_Details{Details: &types.RequestOwnDetails{}},
	}

	flushRequest := &types.Request{
		Value: &types.Request_Flush{
			Flush: &types.RequestFlush{},
		},
	}

	err := utils.WriteProtoMessage(me, conn)
	if err != nil {
		panic(err)
	}

	err = utils.WriteProtoMessage(flushRequest, conn)
	if err != nil {
		panic(err)
	}

	res := &types.Response{}
	err = utils.ReadProtoMessage(res, conn)
	if err != nil {
		panic(err)
	}

	return res.Value.(*types.Response_Details).Details.Details

}

func sendMessage(msg string, recipient *config.ClientConfig, conn net.Conn) {

	msgBytes := []byte(msg)

	flushRequest := &types.Request{
		Value: &types.Request_Flush{
			Flush: &types.RequestFlush{},
		},
	}

	sendRequest := &types.Request{
		Value: &types.Request_Send{Send: &types.RequestSendMessage{
			Message:   msgBytes,
			Recipient: recipient,
		}},
	}

	err := utils.WriteProtoMessage(sendRequest, conn)
	if err != nil {
		panic(err)
	}

	err = utils.WriteProtoMessage(flushRequest, conn)
	if err != nil {
		panic(err)
	}

	res := &types.Response{}
	err = utils.ReadProtoMessage(res, conn)
	if err != nil {
		panic(err)
	}
}

func fetchMessages(conn net.Conn) [][]byte {
	flushRequest := &types.Request{
		Value: &types.Request_Flush{
			Flush: &types.RequestFlush{},
		},
	}

	fetchRequest := &types.Request{
		Value: &types.Request_Fetch{
			Fetch: &types.RequestFetchMessages{},
		},
	}

	err := writeProtoMessage(fetchRequest, conn)
	if err != nil {
		panic(err)
	}

	err = utils.WriteProtoMessage(flushRequest, conn)
	if err != nil {
		panic(err)
	}

	res2 := &types.Response{}
	err = utils.ReadProtoMessage(res2, conn)
	if err != nil {
		panic(err)
	}
	return res2.Value.(*types.Response_Fetch).Fetch.Messages

}

func writeProtoMessage(msg proto.Message, w io.Writer) error {
	b, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	return encodeByteSlice(w, b)
}

func encodeByteSlice(w io.Writer, bz []byte) (err error) {
	err = encodeBigEndianLen(w, uint64(len(bz)))
	if err != nil {
		return
	}
	_, err = w.Write(bz)
	return
}

func encodeBigEndianLen(w io.Writer, i uint64) (err error) {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, i)
	_, err = w.Write(buf)
	return
}