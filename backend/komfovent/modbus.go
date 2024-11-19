package komfovent

import (
	"fmt"
	"time"

	"github.com/simonvetter/modbus"
)

type ModbusClient interface {
	ReadRegister(addr uint16, regType modbus.RegType, value any) error
	WriteRegister(addr uint16, value any) error
	Open() error
	Close() error
}

type client struct {
	mc *modbus.ModbusClient
}

// "tcp://192.168.1.90:502", 5 * time.Second
func NewModbusClient(url string, timeout time.Duration) (ModbusClient, error) {
	// for a TCP endpoint. Use udp:// for modbus TCP over UDP
	// (see examples/tls_client.go for TLS usage and options)
	c, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:     url,
		Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}
	return &client{
		mc: c,
	}, nil
}

func (c *client) Close() error {
	return c.mc.Close()
}

func (c *client) Open() error {
	// now that the client is created and configured, attempt to connect
	err := c.mc.Open()
	if err != nil {
		return err
		// error out if we failed to connect/open the device
		// note: multiple Open() attempts can be made on the same client until
		// the connection succeeds (i.e. err == nil), calling the constructor again
		// is unnecessary.
		// likewise, a client can be opened and closed as many times as needed.
	}
	return nil
}

const (
	ADDRESS_ONOFF      uint16 = 0
	ADDRESS_ECO        uint16 = 2
	ADDRESS_AUTO       uint16 = 3
	ADDRESS_MODE       uint16 = 4
	MODE_AWAY_ON       uint16 = 1
	MODE_AWAY_OFF      uint16 = 2 // back to normal
	MODE_NORMAL_ON     uint16 = 2
	MODE_NORMAL_OFF    uint16 = 1 // back to away
	MODE_INTENSIVE_ON  uint16 = 3
	MODE_INTENSIVE_OFF uint16 = 2 // back to normal
	MODE_BOOST_ON      uint16 = 4
	MODE_BOOST_OFF     uint16 = 2 // back to normal
)

func (c *client) ReadRegister(addr uint16, regType modbus.RegType, value any) error {
	switch v := value.(type) {
	case *uint16:
		var reg16 uint16
		reg16, err := c.mc.ReadRegister(addr, regType)
		if err != nil {
			return err
		}
		*v = reg16
	default:
		return fmt.Errorf("Unsupported read type: %T", value)
	}
	return nil
}

func (c *client) WriteRegister(addr uint16, value any) error {
	switch v := value.(type) {
	case uint16:
		err := c.mc.WriteRegister(addr, v)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unsupported write type: %T", value)
	}
	return nil
}
