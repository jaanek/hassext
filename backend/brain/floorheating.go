package brain

import (
	"time"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/floorheating"
	"github.com/jaanek/hassext/homeassistant"
)

func (b *brain) FloorHeating(state data.DataValue) {
	// update floor heating valve states
	for _, refId := range floorheating.ValveRefIds {
		valveState, err := homeassistant.ParseSwitchState(homeassistant.StateSwitchPrefix+string(refId), state)
		if err != nil {
			b.errors <- err
		} else {
			b.floorHeatingValves[refId] = valveState
			b.lo.Info("FloorHeating", "valve state", *valveState)
		}
	}
	// update simple thermostat states
	for _, refId := range floorheating.FloorHeatingSimpleThermostats {
		state, err := homeassistant.ParseThermostatState(homeassistant.StateThermostatPrefix+string(refId)+homeassistant.StateThermostatSuffix, state)
		if err != nil {
			b.errors <- err
		} else {
			b.floorHeatingSimpleThermostats[refId] = state
			b.lo.Info("FloorHeating", "simple thermostat state", *state)
		}
	}
	// update simple thermostat targets
	for _, refId := range floorheating.FloorHeatingSimpleThermostatTargets {
		state, err := homeassistant.ParseThermostatState(homeassistant.StateThermostatInputNumberPrefix+string(refId)+homeassistant.StateThermostatInputNumberSuffix, state)
		if err != nil {
			b.errors <- err
		} else {
			b.floorHeatingSimpleThermostatTargets[refId] = state
			b.lo.Info("FloorHeating", "simple thermostat target", *state)
		}
	}
	// save simple thermostat values to db
	db, err := b.openAppDatabase()
	if err != nil {
		b.lo.Warn("No app database can be opened!", "error", err)
		return
	}
	defer db.Close()
	var now = time.Now()
	for name, thermostat := range b.floorHeatingSimpleThermostats {
		var target = b.floorHeatingSimpleThermostatTargets[floorheating.ThermostatTargetName(name)]
		if target == nil {
			b.lo.Error("Error while getting thermostat target. Target name does not exist!", "thermostat target name", name)
			continue
		}
		// save it to db
		err := floorheating.ThermostatUpsert(db, name, target.CurrentTemperature, thermostat.CurrentTemperature, now)
		if err != nil {
			b.lo.Error("Error while updating thermostat temperature value in db!", "thermostat name", name, "error", err)
		}
	}
}

// update floorheating valve states
func (b *brain) UpdateFloorheatingValves(valveStates map[floorheating.FloorHeatingValveStateId]*homeassistant.SwitchState) error {
	return b.flootHeating.CheckFloorHeatingValves(b.floorHeatingValves)
}
