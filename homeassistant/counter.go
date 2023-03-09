package homeassistant

func (m *homeassistant) CounterConfigure(entityId string, min uint64, max uint64, step uint64, initial uint64, value uint64) error {
	var req = struct {
		EntityId string `json:"entity_id"`
		Minimum  uint64 `json:"minimum"`
		Maximum  uint64 `json:"maximum"`
		Step     uint64 `json:"step"`
		Initial  uint64 `json:"initial"`
		Value    uint64 `json:"value"`
	}{
		EntityId: entityId,
		Minimum:  min,
		Maximum:  max,
		Step:     step,
		Initial:  initial,
		Value:    value,
	}

	err := m.callService("counter", "configure", req)
	if err != nil {
		return err
	}
	return nil
}

type CounterAction string

const (
	CounterIncrement CounterAction = "increment"
	CounterDecrement CounterAction = "decrement"
	CounterReset     CounterAction = "reset"
)

func (m *homeassistant) Counter(entityId string, action CounterAction) error {
	var req = struct {
		EntityId string `json:"entity_id"`
	}{
		EntityId: entityId,
	}

	err := m.callService("counter", string(action), req)
	if err != nil {
		return err
	}
	return nil
}
