package brain

import (
	"fmt"

	"github.com/jaanek/hassext/data"
)

type EntityState struct {
	Id    string `json:"entity_id"`
	State string `json:"state"`
}

func ParseEntityState(entityId string, state data.DataValue) (*EntityState, error) {
	var entity EntityState = EntityState{
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
	if errors.HasAny() {
		return nil, errors.FirstError()
	}
	return &entity, nil
}
