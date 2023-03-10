package brain

import (
	"fmt"
	"time"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/heatpump"
	"github.com/jaanek/hassext/homeassistant"
	"github.com/jaanek/hassext/nordpool"
)

func (b *brain) HeatPump(state data.DataValue) {
	// heating
	entity, err := ParseEntityState(string(heatpump.ENTITY_HEATING), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpHeating = *entity
		b.lo.Info("HeatPump", "heating", entity)
	}
	// water tank
	entity, err = ParseEntityState(string(heatpump.ENTITY_WATER_TANK), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpWaterTank = *entity
		b.lo.Info("HeatPump", "water tank", entity)
	}
	// water tank boost
	entity, err = ParseEntityState(string(heatpump.ENTITY_WATER_TANK_BOOST), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpWaterTankBoost = *entity
		b.lo.Info("HeatPump", "water tank boost", entity)
	}
	// water tank start time
	entity, err = ParseEntityState(string(heatpump.ENTITY_TIME_WATER_START), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpWaterTankStartState = *entity
		b.lo.Info("HeatPump", "water tank start state", entity)
	}
	// water tank automation start
	entity, err = ParseEntityState(string(heatpump.ENTITY_WATER_TANK_START), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpWaterTankAutomationStart = *entity
		b.lo.Info("HeatPump", "automation water tank start", entity)
	}
	// water tank automation stop
	entity, err = ParseEntityState(string(heatpump.ENTITY_WATER_TANK_STOP), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpWaterTankAutomationStop = *entity
		b.lo.Info("HeatPump", "automation water tank stop", entity)
	}
}

func (b *brain) SetHeatPumpAutomationsOnOff() error {
	var setOff = !b.EmodulIsDamped()
	if setOff {
		// turn off automation because katel is doing periodic heating
		if b.heatPumpWaterTankAutomationStart.State != "off" {
			b.lo.Info("HeatPump", "katel running", setOff, "turn OFF water tank start automation")
			err := heatpump.Automation(b.ha, heatpump.ENTITY_WATER_TANK_START, homeassistant.AUTOMATION_TURN_OFF)
			if err != nil {
				b.errors <- err
			}
		}
		if b.heatPumpWaterTankAutomationStop.State != "off" {
			b.lo.Info("HeatPump", "katel running", setOff, "turn OFF water tank stop automation")
			err := heatpump.Automation(b.ha, heatpump.ENTITY_WATER_TANK_STOP, homeassistant.AUTOMATION_TURN_OFF)
			if err != nil {
				b.errors <- err
			}
		}
	} else {
		// turn off automation because katel is doing periodic heating
		if b.heatPumpWaterTankAutomationStart.State != "on" {
			b.lo.Info("HeatPump", "katel running", setOff, "turn ON water tank start automation")
			err := heatpump.Automation(b.ha, heatpump.ENTITY_WATER_TANK_START, homeassistant.AUTOMATION_TURN_ON)
			if err != nil {
				b.errors <- err
			}
		}
		if b.heatPumpWaterTankAutomationStop.State != "on" {
			b.lo.Info("HeatPump", "katel running", setOff, "turn ON water tank stop automation")
			err := heatpump.Automation(b.ha, heatpump.ENTITY_WATER_TANK_STOP, homeassistant.AUTOMATION_TURN_ON)
			if err != nil {
				b.errors <- err
			}
		}
	}
	return nil
}

const (
	WaterTankHeatingLength = 4 // heating length
	// WaterTankNextHoursLength = 12 // length to choose the cheapest price
)

const BOILER_HEATING_TRIGGER_TEMP float64 = 50 // 42
const (
	TODAY_PM_9        = 21
	TOMORROW_MIDNIGHT = 24
	TOMORROW_AM_7     = 31
	TOMORROW_AM_9     = 33
)

// set start and stop time for the heat pump water tank
func (b *brain) SetHeatPumpWaterTankStartStopTime(hourStart, hourEnd int, todayPrices, tomorrowPrices []float64) error {
	// concatenate all hours and keep track of midnight
	hours := append([]float64(nil), todayPrices...) // from 00:00 -> 23:00
	if len(tomorrowPrices) > 0 {
		hours = append(hours, tomorrowPrices...) // from 00:00 -> 23:00
	}

	// validate
	if hourStart >= hourEnd {
		return fmt.Errorf("hour start cannot be bigger or equal to hourEnd. hourStart: %v, hourEnd: %v", hourStart, hourEnd)
	}
	if hourEnd > len(hours) {
		return fmt.Errorf("hour end cannot be bigger than len of hours. hourEnd: %v, len of hours: %v", hourEnd, len(hours))
	}

	// boiler temp must be available
	// if b.emodulBoilerTemp.State == "" {
	// 	return fmt.Errorf("HeatPump boiler temp state is empty! %v", b.emodulBoilerTemp.State)
	// }

	// if we are heating then do nothing
	// if b.heatPumpWaterTank.State == "on" {
	// 	b.lo.Info("HeatPump", "water tank heating", b.heatPumpWaterTank.State, "skip calculating next start stop time!")
	// 	return nil
	// }

	// check boiler temp
	// var boilerTempBelowTrigger = false
	// boilerTemp, err := strconv.ParseFloat(b.emodulBoilerTemp.State, 32)
	// if err != nil {
	// 	return err
	// }
	// if boilerTemp <= BOILER_HEATING_TRIGGER_TEMP {
	// 	boilerTempBelowTrigger = true
	// }
	// if !boilerTempBelowTrigger {
	// 	b.lo.Info("HeatPump water tank boiler temp above heating trigger", "boiler temp", boilerTemp, "heating trigger", BOILER_HEATING_TRIGGER_TEMP, "skip calculating next start stop time!")
	// 	return nil
	// }

	// t1 := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, t.Nanosecond(), t.Location())
	now := time.Now()
	// nowHour := now.Hour()

	// select next hour as starting point
	fromIdx := hourStart
	toIdx := hourEnd
	if toIdx > len(hours) {
		toIdx = len(hours)
	}
	b.lo.Info("HeatPump checking cheapest prices for next hours period", "from", fromIdx, "to", toIdx, "len hours", len(hours))

	// get the cheapestAll 4 sequential hours from provided list. 4 hours is enough for dishwasher to finish in eco mode
	cheapestAll := nordpool.FindCheapestElectricityHours(hours, WaterTankHeatingLength, fromIdx, toIdx)
	if len(cheapestAll) > 0 {
		cheapest := cheapestAll[0]
		b.lo.Info("HeatPump cheapest prices", "from", fromIdx, "to", toIdx, "len hours", len(hours), "prices", cheapestAll)

		// check if we are planning heating for tomorrow
		hourNr := cheapest.StartIndex
		if cheapest.StartIndex >= 24 { // index at tomorrow midnight
			hourNr = cheapest.StartIndex - 24
		}

		// set start time
		start := time.Date(now.Year(), now.Month(), now.Day(), hourNr, 0, 0, 0, now.Location())
		if b.heatPumpWaterTankStart != start {
			err := b.ha.SetInputDateTime(string(heatpump.ENTITY_TIME_WATER_START), start, homeassistant.INPUT_TIME)
			if err != nil {
				return err
			}
			b.heatPumpWaterTankStart = start
		}

		// set end time
		endHourNr := hourNr + WaterTankHeatingLength
		if endHourNr <= 24 {
			endHourNr -= 24
		}
		end := time.Date(now.Year(), now.Month(), now.Day(), endHourNr, 0, 0, 0, now.Location())
		if b.heatPumpWaterTankStop != start {
			err := b.ha.SetInputDateTime(string(heatpump.ENTITY_TIME_WATER_STOP), end, homeassistant.INPUT_TIME)
			if err != nil {
				return err
			}
			b.heatPumpWaterTankStop = start
		}
	}
	return nil
}

// func (b *brain) SetHeatPumpHeatingStartTime() error {
// 	// set start time for soojuspumpt kyte
// 	// t1 := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, t.Nanosecond(), t.Location())
// 	n := time.Now()
// 	heatingStart := time.Date(n.Year(), n.Month(), n.Day(), 3, 0, 0, 0, n.Location())
// 	if b.heapPumpHeatingStart != heatingStart {
// 		err := b.ha.SetInputDateTime(string(heatpump.ENTITY_TIME_HEATING_START), heatingStart, homeassistant.INPUT_TIME)
// 		if err != nil {
// 			return err
// 		}
// 		b.heapPumpHeatingStart = heatingStart
// 	}
// 	return nil
// }
