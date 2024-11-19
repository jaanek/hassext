package homeassistant

type Notify interface {
	Notify(string, string, string) error
}

func (m *homeassistant) Notify(entityId string, title string, msg string) error {
	// persistent_notification
	// mobile_app_ac2003
	var req = struct {
		Title   string `json:"title"`
		Message string `json:"message"`
	}{
		Title:   title,
		Message: msg,
	}

	err := m.callService("notify", entityId, req)
	if err != nil {
		return err
	}
	return nil
}
