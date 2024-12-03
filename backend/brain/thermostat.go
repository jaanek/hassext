package brain

import (
	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/floorheating"
	"github.com/jaanek/hassext/homeassistant"
)

func (b *brain) Thermostats(state data.DataValue) {
	for _, refId := range floorheating.ValveRefIds {
		valveState, err := homeassistant.ParseSwitchState(homeassistant.StateSwitchPrefix+string(refId), state)
		if err != nil {
			b.errors <- err
		} else {
			b.floorHeatingValves[refId] = valveState
			b.lo.Info("FloorHeating", "valve state", *valveState)
		}
	}
}
