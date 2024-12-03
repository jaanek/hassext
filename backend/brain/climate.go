package brain

import (
	"fmt"
	"time"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/floorheating"
	"github.com/jaanek/hassext/sqldb"
)

type ClimateState struct {
	Id                 string  `json:"entity_id"`
	State              string  `json:"state"`
	CurrentTemperature float64 `json:"current_temperature"`
	SetTemperature     float64 `json:"temperature"` // temp set on termostat
}

func ParseClimateState(entityId string, state data.DataValue) (*ClimateState, error) {
	var entity ClimateState = ClimateState{
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
	entity.SetTemperature = entityData.GetFloat64("$.attributes.temperature", &errors)
	if errors.HasAny() {
		return nil, fmt.Errorf("%s parsing error: %w", entityId, errors.FirstError())
	}
	return &entity, nil
}

func NewClimateState(entityId string, currentTemp float64, setTemp float64) ClimateState {
	var entity = ClimateState{
		Id: entityId,
	}
	entity.CurrentTemperature = currentTemp
	entity.SetTemperature = setTemp
	return entity
}

func (b *brain) saveTemperatureDB(db *sqldb.DB, t *ClimateState, name floorheating.ThermostatName, now time.Time) {
	// save it to db
	err := floorheating.ThermostatUpsert(db, name, t.SetTemperature, t.CurrentTemperature, now)
	if err != nil {
		b.lo.Error("Error while updating thermostat temperature value in db!", "error", err)
	}
}
