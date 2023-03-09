package homeassistant

type Modbus interface {
	ModbusWriteRegister(string, uint, byte, string) error
	ModbusWriteCoil(string, uint, byte, string) error
}

// https://www.home-assistant.io/integrations/modbus#modbus-services
func (m *homeassistant) ModbusWriteRegister(hub string, address uint, unit byte, value string) error {
	var req = struct {
		Hub     string `json:"hub"`
		Address uint   `json:"address"`
		Slave   byte   `json:"slave,omitempty"`
		Value   string `json:"value"`
	}{
		Hub:     hub,
		Address: address,
		Slave:   unit,
		Value:   value,
	}

	err := m.callService("modbus", "write_register", req)
	if err != nil {
		return err
	}
	return nil
}

func (m *homeassistant) ModbusWriteCoil(hub string, address uint, unit byte, state string) error {
	var req = struct {
		Hub     string `json:"hub"`
		Address uint   `json:"address"`
		Slave   byte   `json:"slave,omitempty"`
		State   string `json:"state"`
	}{
		Hub:     hub,
		Address: address,
		Slave:   unit,
		State:   state,
	}

	err := m.callService("modbus", "write_coil", req)
	if err != nil {
		return err
	}
	return nil
}
