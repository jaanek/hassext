package homeassistant

import "fmt"

type WaterHeater interface {
	WaterHeater(string, WaterHeaterAction) error
	WaterHeaterSetOperationMode(string, WaterHeaterOperationMode) error
	WaterHeaterSetTemperature(string, float32, *WaterHeaterOperationMode) error
}

type WaterHeaterAction string

const (
	WATER_HEATER_TURN_ON  WaterHeaterAction = "turn_on"
	WATER_HEATER_TURN_OFF WaterHeaterAction = "turn_off"
)

func (m *homeassistant) WaterHeater(entityId string, action WaterHeaterAction) error {
	// water_heater.altherma
	var req = struct {
		EntityId string `json:"entity_id"`
	}{
		EntityId: entityId,
	}

	err := m.callService("water_heater", string(action), req)
	if err != nil {
		return err
	}
	return nil
}

type WaterHeaterOperationMode string

const (
	WATER_HEATER_OFF       WaterHeaterOperationMode = "off"
	WATER_HEATER_HEAT_PUMP WaterHeaterOperationMode = "heat_pump"
	WATER_HEATER_BOOST     WaterHeaterOperationMode = "performance" // boost
)

func (m *homeassistant) WaterHeaterSetOperationMode(entityId string, mode WaterHeaterOperationMode) error {
	// water_heater.altherma
	var req = struct {
		EntityId      string `json:"entity_id"`
		OperationMode string `json:"operation_mode"`
	}{
		EntityId:      entityId,
		OperationMode: string(mode),
	}

	err := m.callService("water_heater", "set_operation_mode", req)
	if err != nil {
		return err
	}
	return nil
}

func (m *homeassistant) WaterHeaterSetTemperature(entityId string, temp float32, mode *WaterHeaterOperationMode) error {
	// heating pumps: climate.altherma
	var req = struct {
		EntityId      string  `json:"entity_id"`
		Temperature   float32 `json:"temperature"`
		OperationMode string  `json:"operation_mode,omitempty"`
	}{
		EntityId:    entityId,
		Temperature: temp,
	}
	if mode != nil {
		req.OperationMode = string(*mode)
	}

	// validate params
	if temp > 100 {
		return fmt.Errorf("Invalid temp value! Max 100 allowed. Provided value: %v", temp)
	}

	err := m.callService("water_heater", "set_temperature", req)
	if err != nil {
		return err
	}
	return nil
}
