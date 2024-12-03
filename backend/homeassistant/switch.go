package homeassistant

import (
	"fmt"

	"github.com/jaanek/hassext/data"
)

type Switch interface {
	Switch(string, SwitchAction) error
}

type SwitchAction string

const (
	SWITCH_ON     SwitchAction = "turn_on"
	SWITCH_OFF    SwitchAction = "turn_off"
	SWITCH_TOGGLE SwitchAction = "toggle"
)

func (m *homeassistant) Switch(entityId string, action SwitchAction) error {
	var req = struct {
		EntityId string `json:"entity_id"`
	}{
		EntityId: entityId,
	}

	err := m.callService("switch", string(action), req)
	if err != nil {
		return err
	}
	return nil
}

type SwitchStateValue string

const (
	SwitchStateOn  SwitchStateValue = "on"
	SwitchStateOff SwitchStateValue = "off"
)

type SwitchState struct {
	Id           string           `json:"entity_id"`
	State        SwitchStateValue `json:"state"`
	FriendlyName string           `json:"friendly_name"`
}

const StateSwitchPrefix = "switch."
const StateInputBooleanPrefix = "input_boolean."

func ParseSwitchState(entityId string, state data.DataValue) (*SwitchState, error) {
	var entity SwitchState = SwitchState{
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
	entity.State = SwitchStateValue(entityData.GetString("$.state", &errors))
	entity.FriendlyName = entityData.GetString("$.attributes.friendly_name", &errors)
	if errors.HasAny() {
		return nil, fmt.Errorf("%s parsing error: %w", entityId, errors.FirstError())
	}
	return &entity, nil
}
