package esphome

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

// floor heating esphome device
var FloorHeatingCoverDevice = DeviceEndpoint{
	Address:       "192.168.2.152:6053",
	EncryptionKey: "4gCwwRpi1COquHR58yv64o2/rkQUp66Y98zU69DteGU=",
}

// floor heating "küte peale vesi klapi ava" entity id
const floorHeatingCoverKey = uint32(3992273367)

func SetCoverPosition(endpoint DeviceEndpoint, position float32) error {
	var client, err = NewNativeAPIClient(endpoint, func(msg proto.Message) {
		fmt.Printf("received a message, type: %T, value: [%v]\n", msg, msg)
	})
	if err != nil {
		return err
	}
	defer client.Close()

	// authenticate itselt to esp device to make certain requests
	err = client.Login("")
	if err != nil {
		return err
	}
	// Example: Send a command to a cover entity
	// var position float32 = 0.3
	err = SendCoverCommand(client, floorHeatingCoverKey, &position, nil, nil)
	if err != nil {
		return err
	}
	return nil
}
