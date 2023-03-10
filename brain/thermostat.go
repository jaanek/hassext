package brain

import (
	"fmt"

	"github.com/jaanek/hassext/data"
)

type ThermostatState struct {
	Id                 string  `json:"entity_id"`
	State              string  `json:"state"`
	CurrentTemperature float64 `json:"current_temperature"`
}

func ParseThermostatState(entityId string, state data.DataValue) (*ThermostatState, error) {
	var entity ThermostatState = ThermostatState{
		Id: entityId,
	}
	var errors data.Errors

	// get sensor data
	entityPath := fmt.Sprintf("$[?(@.entity_id == \"%v\")]", entityId)
	parent := state.GetObject(entityPath)
	if parent == nil {
		return nil, fmt.Errorf("Entity state not found! entity_id: %v", entityId)
	}
	entityData := data.NewDataValue(parent)
	entity.State = entityData.GetString("$.state", &errors)
	entity.CurrentTemperature = entityData.GetFloat64("$.attributes.current_temperature", &errors)
	if errors.HasAny() {
		return nil, errors.FirstError()
	}
	return &entity, nil
}
