package homeassistant

import (
	"fmt"

	"github.com/jaanek/hassext/data"
)

type EntityState struct {
	Id    string `json:"entity_id"`
	State string `json:"state"`
}

func (m *homeassistant) ParseEntityState(entityId string) (*EntityState, error) {
	var entity EntityState = EntityState{
		Id: entityId,
	}
	var errors data.Errors

	// get sensor data
	entityPath := fmt.Sprintf("$[?(@.entity_id == \"%v\")]", entityId)
	json := m.stateData.GetObject(entityPath)
	if json == nil {
		return nil, fmt.Errorf("Entity state not found! entity_id: %v", entityId)
	}
	entityData := data.Data{Value: json}
	entity.State = entityData.GetString("$.state", &errors)
	if errors.HasAny() {
		return nil, errors.FirstError()
	}
	return &entity, nil
}
