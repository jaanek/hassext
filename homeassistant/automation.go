package homeassistant

type Automation interface {
	Automation(string, AutomationAction) error
}

type AutomationAction string

const (
	AUTOMATION_TRIGGER  AutomationAction = "trigger"
	AUTOMATION_TOGGLE   AutomationAction = "toggle"
	AUTOMATION_TURN_ON  AutomationAction = "turn_on"
	AUTOMATION_TURN_OFF AutomationAction = "turn_off"
	AUTOMATION_RELOAD   AutomationAction = "reload"
)

func (m *homeassistant) Automation(entityId string, action AutomationAction) error {
	// automation.kaivita_soojuspump
	var req = struct {
		EntityId string `json:"entity_id"`
	}{
		EntityId: entityId,
	}

	err := m.callService("automation", string(action), req)
	if err != nil {
		return err
	}
	return nil
}
