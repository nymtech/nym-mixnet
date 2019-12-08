package main

import (
	"encoding/base64"
    "fmt"
    "os"
    "time"
	"github.com/golang/protobuf/proto"
	"github.com/nymtech/nym-mixnet/client"
	"github.com/nymtech/nym-mixnet/config"
	clientConfig "github.com/nymtech/nym-mixnet/client/config"
    //"./message/message.pb.go"
	"github.com/nymtech/demo-mixnet-chat-client/message"
)

func createMessagePayload(
    client client.NetClient, msg string) ([]byte, error) {

	protoPayload := &message.ChatMessage{
		Content:                 []byte(msg),
		SenderPublicKey:         client.GetPublicKey().Bytes(),
		SenderProviderPublicKey: nil,
		MessageNonce:            110,
		SenderTimestamp:         time.Now().UnixNano(),
		Signature:               nil, // will be done later
	}

	return proto.Marshal(protoPayload)
}

func parseReceivedMessages(msgs [][]byte) []*message.ChatMessage {
	parsedMsgs := make([]*message.ChatMessage, 0, len(msgs))
	if msgs == nil {
		return parsedMsgs
	}
	for _, msg := range msgs {
		if msg != nil {
			parsedMsg := &message.ChatMessage{}
			if err := proto.Unmarshal(msg, parsedMsg); err == nil {
				parsedMsgs = append(parsedMsgs, parsedMsg)
			}
		}
	}

	// for now completely ignore ordering
	return parsedMsgs
}

func main() {
    fmt.Println("Simple chat demo")

    configPath := "/home/nar/.loopix/clients/amir/config/config.toml"

	cfg, err := clientConfig.LoadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not load the config file: %v\n", err)
		os.Exit(1)
	}

	mixClient, err := client.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create client: %v\n", err)
		os.Exit(1)
	}

    err = mixClient.UpdateNetworkView()
    if err != nil {
        panic("bad die die")
    }

    var recipient config.ClientConfig
    for _, remote := range mixClient.Network.Clients {
	    b64Key := base64.URLEncoding.EncodeToString(remote.PubKey)
        if b64Key == "F9xzbjnMQVN4ZidcqN2ip9kVnI9wbS39aVayZGiMihY=" {
            recipient = remote
        }
    }

    b64Key := base64.URLEncoding.EncodeToString(recipient.PubKey)
    fmt.Println(b64Key)

    mixClient.Start()

    //message := []byte(`{"command": "fetch_history", "id": null, "client": "4JEtSrKsonmBuDvxJ9nITSu7iC4f8reutXRAVugPgS4=", "data": ["mtovQPnUuCeAUJkhqZn5vJ99vxJYNGXoEn"]}`)
    message := []byte("ping")
	if err := mixClient.SendMessage(message, recipient);
        err != nil {

		fmt.Fprintf(os.Stderr, "Error send message: %v\n", err)
		os.Exit(1)
	}

	heartbeat := time.NewTicker(50 * time.Millisecond)
loop:
    for {
        select {
        case <-heartbeat.C:
            msgs := mixClient.GetReceivedMessages()
            if len(msgs) > 0 {
				for _, msg := range msgs {
                    fmt.Println(string(msg))
                }
                break loop
            }
        }
    }
}

