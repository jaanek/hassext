package brain

import (
	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/floorheating"
	"github.com/jaanek/hassext/homeassistant"
)

func (b *brain) FloorHeating(state data.DataValue) {
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

// update floorheating valve states
func (b *brain) UpdateFloorheatingValves(valveStates map[floorheating.FloorHeatingValveStateId]*homeassistant.SwitchState) error {
	return b.flootHeating.CheckFloorHeatingValves(b.floorHeatingValves)
}
