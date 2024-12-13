package esphome

import (
	"fmt"
	"log"
	"time"

	"github.com/mycontroller-org/esphome_api/pkg/api"
	espclient "github.com/mycontroller-org/esphome_api/pkg/client"
	"google.golang.org/protobuf/proto"
)

type DeviceEndpoint struct {
	Address       string
	EncryptionKey string
}

type NativeClient struct {
	*espclient.Client
}

func NewNativeAPIClient(ep DeviceEndpoint, handlerFunc func(msg proto.Message)) (*NativeClient, error) {
	if handlerFunc == nil {
		handlerFunc = handlerFuncImpl
	}
	// Create a new ESPHome API client
	client, err := espclient.GetClient("hassext", ep.Address, ep.EncryptionKey, 10*time.Second, handlerFunc)
	if err != nil {
		log.Fatalln(err)
	}
	return &NativeClient{client}, nil
}

func handlerFuncImpl(msg proto.Message) {
	fmt.Printf("received a message, type: %T, value: [%v]\n", msg, msg)
}

func SendCoverCommand(client *NativeClient, key uint32, position, tilt *float32, stop *bool) error {
	var cmd = api.CoverCommandRequest{
		Key: key,
	}
	if position != nil {
		cmd.HasPosition = true
		cmd.Position = *position
	} else if tilt != nil {
		cmd.HasTilt = true
		cmd.Tilt = *tilt
	} else if stop != nil {
		cmd.Stop = *stop
	}
	if err := client.Send(&cmd); err != nil {
		return err
	}
	return nil
}
