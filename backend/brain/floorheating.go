package brain

import (
	"time"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/floorheating"
	"github.com/jaanek/hassext/homeassistant"
)

func (b *brain) FloorHeating(state data.DataValue) {
	floorHeatingValvesManualOp, err := homeassistant.ParseSwitchState(homeassistant.StateInputBooleanPrefix+string(floorheating.FLOOR_HEATING_VALVES_MANUAL_OPERATION_SWITCH), state)
	if err != nil {
		b.errors <- err
	} else {
		b.floorHeatingValvesManualOperation = *floorHeatingValvesManualOp
		b.lo.Info("FloorHeating", "valves manual operation state", *floorHeatingValvesManualOp)
	}
	floorHeatingKlappManualOp, err := homeassistant.ParseSwitchState(homeassistant.StateInputBooleanPrefix+string(floorheating.FLOOR_HEATING_KLAPP_MANUAL_OPERATION_SWITCH), state)
	if err != nil {
		b.errors <- err
	} else {
		b.floorHeatingKlappManualOperation = *floorHeatingKlappManualOp
		b.lo.Info("FloorHeating", "klapp manual operation state", *floorHeatingKlappManualOp)
	}
	floorHeatingTargetTempState, err := homeassistant.ParseThermostatState(homeassistant.StateThermostatInputNumberPrefix+string(floorheating.FLOOR_HEATING_TARGET_TEMPERATURE), state)
	if err != nil {
		b.errors <- err
	} else {
		b.floorHeatingKlappStates[floorheating.FLOOR_HEATING_TARGET_TEMPERATURE] = floorHeatingTargetTempState
		b.lo.Info("FloorHeating", "target temperature state", *floorHeatingTargetTempState)
	}
	floorHeatingPealeTempState, err := homeassistant.ParseThermostatState(homeassistant.StateThermostatPrefix+string(floorheating.FLOOR_HEATING_PEALE_TEMPERATURE), state)
	if err != nil {
		b.errors <- err
	} else {
		b.floorHeatingKlappStates[floorheating.FLOOR_HEATING_PEALE_TEMPERATURE] = floorHeatingPealeTempState
		b.lo.Info("FloorHeating", "peale temperature state", *floorHeatingPealeTempState)
	}
	floorHeatingKlapiAvaState, err := homeassistant.ParseThermostatState(homeassistant.StateThermostatPrefix+string(floorheating.FLOOR_HEATING_KLAPI_AVA), state)
	if err != nil {
		b.errors <- err
	} else {
		b.floorHeatingKlappStates[floorheating.FLOOR_HEATING_KLAPI_AVA] = floorHeatingKlapiAvaState
		b.lo.Info("FloorHeating", "kontuur klapi ava state", *floorHeatingKlapiAvaState)
	}
	floorHeatingBufferTopTemp, err := homeassistant.ParseThermostatState(homeassistant.StateThermostatPrefix+string(floorheating.FLOOR_HEATING_BUFFER_TOP_TEMPERATURE), state)
	if err != nil {
		b.errors <- err
	} else {
		b.floorHeatingKlappStates[floorheating.FLOOR_HEATING_BUFFER_TOP_TEMPERATURE] = floorHeatingBufferTopTemp
		b.lo.Info("FloorHeating", "buffer top temperature", *floorHeatingBufferTopTemp)
	}
	floorHeatingBufferBottomTemp, err := homeassistant.ParseThermostatState(homeassistant.StateThermostatPrefix+string(floorheating.FLOOR_HEATING_BUFFER_BOTTOM_TEMPERATURE), state)
	if err != nil {
		b.errors <- err
	} else {
		b.floorHeatingKlappStates[floorheating.FLOOR_HEATING_BUFFER_BOTTOM_TEMPERATURE] = floorHeatingBufferBottomTemp
		b.lo.Info("FloorHeating", "buffer bottom temperature", *floorHeatingBufferBottomTemp)
	}
	// update floor heating valve states
	for _, refId := range floorheating.ValveEntityIds {
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
func (b *brain) UpdateFloorheatingValves(valveStates map[floorheating.FloorHeatingValveEntityId]*homeassistant.SwitchState) error {
	if b.floorHeatingValvesManualOperation.State == floorheating.SwitchOn {
		b.lo.Info("Floor heating valves. Manual operation is activated! Will not auto turn on/off floor heating valves!")
		return nil
	}
	return b.flootHeating.CheckFloorHeatingValves(b.floorHeatingValves)
}

// Check the floor heating target temp if it's in range to peale temp
func (b *brain) UpdateFloorheatingKontuurTemp(klappStates map[floorheating.FloorHeatingKlapp]*homeassistant.ThermostatState) error {
	if b.floorHeatingKlappManualOperation.State == floorheating.SwitchOn {
		b.lo.Info("Floor heating kontuur open klapp/controller. Manual operation is activated! Will not automatically update klapp ava %!")
		return nil
	}
	return b.flootHeating.CheckFloorHeatingKontuurTemp(b.floorHeatingKlappStates)
}
