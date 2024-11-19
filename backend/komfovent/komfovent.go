package komfovent

import (
	"github.com/jaanek/hassext/homeassistant"
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
