package komfovent

import (
	"fmt"
	"time"

	"github.com/jaanek/hassext/homeassistant"
	"github.com/simonvetter/modbus"
)

type OperationMode string

const (
	OperationModeAway      OperationMode = "1"
	OperationModeNormal    OperationMode = "2"
	OperationModeIntensive OperationMode = "3"
	OperationModeBoost     OperationMode = "4"
)

const operationModeAddress uint = 4
const operationModeUnit byte = 1

func SetOperationMode(modbus homeassistant.Modbus, mode OperationMode) error {
	return modbus.ModbusWriteRegister("Komfovent", operationModeAddress, operationModeUnit, string(mode))
}

func Test() error {
	// for a TCP endpoint. Use udp:// for modbus TCP over UDP
	// (see examples/tls_client.go for TLS usage and options)
	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:     "tcp://192.168.1.90:502",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return nil
	}

	// now that the client is created and configured, attempt to connect
	err = client.Open()
	if err != nil {
		return nil
		// error out if we failed to connect/open the device
		// note: multiple Open() attempts can be made on the same client until
		// the connection succeeds (i.e. err == nil), calling the constructor again
		// is unnecessary.
		// likewise, a client can be opened and closed as many times as needed.
	}
	defer client.Close()

	// read a single 16-bit holding register at address 100
	fmt.Println("Connected reading register")
	var reg16 uint16
	reg16, err = client.ReadRegister(0, modbus.HOLDING_REGISTER)
	if err != nil {
		return err
	}
	// use value
	fmt.Printf("value: %v", reg16)        // as unsigned integer
	fmt.Printf("value: %v", int16(reg16)) // as signed integer
	return nil
}
