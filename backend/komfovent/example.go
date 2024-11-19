package komfovent

import (
	"fmt"
	"time"

	"github.com/simonvetter/modbus"
)

func test() error {
	// modbus
	c, err := NewModbusClient("tcp://192.168.1.90:502", 5*time.Second)
	if err != nil {
		return err
	}
	err = c.Open()
	if err != nil {
		return nil
	}
	defer c.Close()

	// read komfovent mode value
	var val uint16
	err = c.ReadRegister(ADDRESS_MODE, modbus.HOLDING_REGISTER, &val)
	if err != nil {
		return err
	}
	fmt.Printf("komfovent mode register value: %v\n", val)

	// write mode value
	var mode uint16 = MODE_INTENSIVE_ON
	err = c.WriteRegister(ADDRESS_MODE, mode)
	if err != nil {
		return err
	}
	return nil
}
