package brain

import (
	"strconv"
	"strings"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/emodul"
)

func (b *brain) EmodulIsDamped() bool {
	// state: PID Damped
	if strings.ToLower(b.emodulControllerState.State) == "pid damped" {
		return true
	}
	return false
}

func (b *brain) Emodule(state data.DataValue) {
	// check if heating is in winter mode
	entity, err := ParseEntityState(string(emodul.ENTITY_BOOL_HEATING_ALLOWED), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heapPumpHeatingAllowedWinterMode = *entity
		b.lo.Info("HeatPump", "heating heatpump allowed / winter mode", *entity)
	}
	// emodul controller state
	entity, err = ParseEntityState(string(emodul.ENTITY_CONTROLLER_STATE), state)
	if err != nil {
		b.errors <- err
	} else {
		b.emodulControllerState = *entity
		b.lo.Info("Emodul", "controller state", entity)
	}
	// emodul operation mode
	entity, err = ParseEntityState(string(emodul.ENTITY_OPERATION_MODE), state)
	if err != nil {
		b.errors <- err
	} else {
		b.emodulOperationMode = *entity
		b.lo.Info("Emodul", "operation mode", entity)
	}
	// external temperature
	entity, err = ParseEntityState(string(emodul.ENTITY_EXTERNAL_TEMPERATURE), state)
	if err != nil {
		b.errors <- err
	} else {
		outsideTemp, err := strconv.ParseFloat((*entity).State, 32) // example: "state": "5.4"
		if err != nil {
			b.errors <- err
		} else {
			b.emodulOutsideTemp = outsideTemp
		}
		b.lo.Info("Emodul", "outside temp", outsideTemp, "external temperature", entity)
	}
	// dhw (boileri) temperature
	entity, err = ParseEntityState(string(emodul.ENTITY_BOILER_TEMPERATURE), state)
	if err != nil {
		b.errors <- err
	} else {
		boilerTemp, err := strconv.ParseFloat((*entity).State, 32) // example: "state": "50.4"
		if err != nil {
			b.errors <- err
		} else {
			b.emodulBoilerTemp = boilerTemp
		}
		b.lo.Info("Emodul", "boiler temp", boilerTemp, "boiler (dhw) temperature", entity)
	}
}

// set the heating allowed
// func (b *brain) SetEmodulHeatingAllowed() error {
// 	var heatingAllowed bool = true
// 	if b.emodulHeatingAllowed != heatingAllowed {
// 		err := b.ha.SetInputBoolean(string(emodul.ENTITY_BOOL_HEATING_ALLOWED), homeassistant.BOOLEAN_TURN_ON)
// 		if err != nil {
// 			return err
// 		}
// 		b.emodulHeatingAllowed = heatingAllowed
// 	}
// 	return nil
// }
