package homeassistant

import "fmt"

type Climate interface {
	Climate(string, ClimateAction) error
	ClimateSetHvacMode(string, ClimateHvacMode) error
	ClimateSetTemperature(string, float32, *ClimateHvacMode) error
}

type ClimateAction string

const (
	CLIMATE_TURN_ON  ClimateAction = "turn_on"
	CLIMATE_TURN_OFF ClimateAction = "turn_off"
)

func (m *homeassistant) Climate(entityId string, action ClimateAction) error {
	// termostats: climate.elutuba
	// heating pumps: climate.altherma
	var req = struct {
		EntityId string `json:"entity_id"`
	}{
		EntityId: entityId,
	}

	err := m.callService("climate", string(action), req)
	if err != nil {
		return err
	}
	return nil
}

type ClimateHvacMode string

const (
	CLIMATE_OFF  ClimateHvacMode = "off"
	CLIMATE_HEAT ClimateHvacMode = "heat"
)

func (m *homeassistant) ClimateSetHvacMode(entityId string, mode ClimateHvacMode) error {
	// heating pumps: climate.altherma
	var req = struct {
		EntityId string `json:"entity_id"`
		HvacMode string `json:"hvac_mode"`
	}{
		EntityId: entityId,
		HvacMode: string(mode),
	}

	err := m.callService("climate", "set_hvac_mode", req)
	if err != nil {
		return err
	}
	return nil
}

func (m *homeassistant) ClimateSetTemperature(entityId string, temp float32, mode *ClimateHvacMode) error {
	// heating pumps: climate.altherma
	var req = struct {
		EntityId    string  `json:"entity_id"`
		Temperature float32 `json:"temperature"`
		HvacMode    string  `json:"hvac_mode,omitempty"`
	}{
		EntityId:    entityId,
		Temperature: temp,
	}
	if mode != nil {
		req.HvacMode = string(*mode)
	}

	// validate params
	if temp > 100 {
		return fmt.Errorf("Invalid temp value! Max 100 allowed. Provided value: %v", temp)
	}

	err := m.callService("climate", "set_temperature", req)
	if err != nil {
		return err
	}
	return nil
}
