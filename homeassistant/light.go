package homeassistant

type Light interface {
	Light(string, LightAction) error
}

type LightAction string

const (
	LIGHT_ON     LightAction = "turn_on"
	LIGHT_OFF    LightAction = "turn_off"
	LIGHT_TOGGLE LightAction = "toggle"
)

func (m *homeassistant) Light(entityId string, action LightAction) error {
	var req = struct {
		EntityId string `json:"entity_id"`
	}{
		EntityId: entityId,
	}

	err := m.callService("light", string(action), req)
	if err != nil {
		return err
	}
	return nil
}
