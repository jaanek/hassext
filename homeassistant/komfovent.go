package homeassistant

type KomfoventOperationMode string

const (
	OperationModeAway      KomfoventOperationMode = "1"
	OperationModeNormal    KomfoventOperationMode = "2"
	OperationModeIntensive KomfoventOperationMode = "3"
	OperationModeBoost     KomfoventOperationMode = "4"
)

const operationModeAddress uint = 4
const operationModeUnit byte = 1

func KomfoventSetOperationMode(modbus Modbus, mode KomfoventOperationMode) error {
	return modbus.ModbusWriteRegister("Komfovent", operationModeAddress, operationModeUnit, string(mode))
}
