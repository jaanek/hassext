package homeassistant

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
