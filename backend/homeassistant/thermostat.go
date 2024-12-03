package homeassistant

import (
	"fmt"
	"math"
	"strconv"

	"github.com/jaanek/hassext/data"
)

type ThermostatState struct {
	Id                 string  `json:"entity_id"`
	State              string  `json:"state"`
	CurrentTemperature float64 `json:"current_temperature"`
	FriendlyName       string  `json:"friendly_name"`
}

const StateThermostatPrefix = "sensor."
const StateThermostatInputNumberPrefix = "input_number."
const StateThermostatSuffix = "_temperature"
const StateThermostatInputNumberSuffix = "_target"

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
	temperature, err := strconv.ParseFloat(entity.State, 64)
	if err != nil {
		return nil, fmt.Errorf("Error while converting string value to float! State: %v", entity.State)
	}
	entity.CurrentTemperature = math.Round(temperature*10) / 10 // Truncate(temperature, 1)
	entity.FriendlyName = entityData.GetString("$.attributes.friendly_name", &errors)
	if errors.HasAny() {
		return nil, fmt.Errorf("%s parsing error: %w", entityId, errors.FirstError())
	}
	return &entity, nil
}

func Truncate(f float64, decimals int) float64 {
	shift := math.Pow(10, float64(decimals))
	return math.Trunc(f*shift) / shift
}
