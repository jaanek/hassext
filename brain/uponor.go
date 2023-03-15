package brain

import (
	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/uponor"
)

func (b *brain) Uponor(state data.DataValue) {
	thermostat, err := ParseThermostatState(string(uponor.ENTITY_THERMOSTAT_ELUTUBA), state)
	if err != nil {
		b.errors <- err
	} else {
		b.uponorElutuba = *thermostat
		b.lo.Info("Uponor thermostat", "elutuba", *thermostat)
	}
	// water tank/heater
}
