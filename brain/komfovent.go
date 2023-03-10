package brain

import "github.com/jaanek/hassext/data"

func (b *brain) Komfovent(state data.DataValue) {
	// komfovent operation mode. Modes: Normal, Intensive, Away, Boost
	entity, err := ParseEntityState("sensor.komfovent_operation_mode", state)
	if err != nil {
		b.errors <- err
	} else {
		b.lo.Info("Komfovent", "operation mode", entity)
	}
}
