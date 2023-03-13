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
	// water tank automation start
	entity, err := ParseEntityState(string(heatpump.ENTITY_WATER_HEATER_START), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpWaterHeaterAutomationStart = *entity
		b.lo.Info("HeatPump", "automation water tank start", entity)
	}
	// water heater automation stop
	entity, err = ParseEntityState(string(heatpump.ENTITY_WATER_HEATER_STOP), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpWaterHeaterAutomationStop = *entity
		b.lo.Info("HeatPump", "automation water tank stop", entity)
	}
	// heating
	thermostat, err := ParseThermostatState(string(heatpump.ENTITY_HEATING), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpHeating = *thermostat
		b.lo.Info("HeatPump", "heating", *thermostat)
	}
	// water tank/heater
	thermostat, err = ParseThermostatState(string(heatpump.ENTITY_WATER_HEATER), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpWaterHeater = *thermostat
		b.lo.Info("HeatPump", "water heater", *thermostat)
	}
	// water tank start time
	entity, err = ParseEntityState(string(heatpump.ENTITY_TIME_WATER_HEATER_START), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpWaterHeaterStartState = *entity
		b.lo.Info("HeatPump", "water tank start state", entity)
	}
	// heating ignore max price per hour
	entity, err = ParseEntityState(string(heatpump.ENTITY_BOOL_IGNORE_MAX_PRICE_PER_HOUR), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heapPumpHeatingIgnoreMaxPricePerHour = *entity
		b.lo.Info("HeatPump", "heating ignore max price per hour", entity)
	}
}

func (b *brain) SetHeatPumpAutomationsOnOff() error {
	var setOff = !b.EmodulIsDamped()
	if setOff {
		// turn off automation because katel is doing periodic heating
		if b.heatPumpWaterHeaterAutomationStart.State != "off" {
			b.lo.Info("HeatPump", "katel running", setOff, "turn OFF water tank start automation")
			err := heatpump.Automation(b.ha, heatpump.ENTITY_WATER_HEATER_START, homeassistant.AUTOMATION_TURN_OFF)
			if err != nil {
				b.errors <- err
			}
		}
		if b.heatPumpWaterHeaterAutomationStop.State != "off" {
			b.lo.Info("HeatPump", "katel running", setOff, "turn OFF water tank stop automation")
			err := heatpump.Automation(b.ha, heatpump.ENTITY_WATER_HEATER_STOP, homeassistant.AUTOMATION_TURN_OFF)
			if err != nil {
				b.errors <- err
			}
		}
	} else {
		// turn off automation because katel is doing periodic heating
		if b.heatPumpWaterHeaterAutomationStart.State != "on" {
			b.lo.Info("HeatPump", "katel running", setOff, "turn ON water tank start automation")
			err := heatpump.Automation(b.ha, heatpump.ENTITY_WATER_HEATER_START, homeassistant.AUTOMATION_TURN_ON)
			if err != nil {
				b.errors <- err
			}
		}
		if b.heatPumpWaterHeaterAutomationStop.State != "on" {
			b.lo.Info("HeatPump", "katel running", setOff, "turn ON water tank stop automation")
			err := heatpump.Automation(b.ha, heatpump.ENTITY_WATER_HEATER_STOP, homeassistant.AUTOMATION_TURN_ON)
			if err != nil {
				b.errors <- err
			}
		}
	}
	return nil
}

const (
	WaterTankHeatingLength = 3 // heating length
)

const BOILER_HEATING_TRIGGER_TEMP float64 = 50 // 42
const (
	TODAY_PM_9        = 21
	TODAY_PM_11       = 23
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

	// get the cheapestAll 3 sequential hours from provided list. 3 hours should be enough for water tank heat up
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
		if b.heatPumpWaterHeaterStart != start {
			err := b.ha.SetInputDateTime(string(heatpump.ENTITY_TIME_WATER_HEATER_START), start, homeassistant.INPUT_TIME)
			if err != nil {
				return err
			}
			b.heatPumpWaterHeaterStart = start
		}

		// set end time
		endHourNr := hourNr + WaterTankHeatingLength
		if endHourNr <= 24 {
			endHourNr -= 24
		}
		end := time.Date(now.Year(), now.Month(), now.Day(), endHourNr, 0, 0, 0, now.Location())
		if b.heatPumpWaterHeaterStop != start {
			err := b.ha.SetInputDateTime(string(heatpump.ENTITY_TIME_WATER_HEATER_STOP), end, homeassistant.INPUT_TIME)
			if err != nil {
				return err
			}
			b.heatPumpWaterHeaterStop = start
		}
	}
	return nil
}

const MAX_PRICE_PER_HOUR float64 = 200

func (b *brain) SetHeatPumpHeating(todayPrices []float64, maxPricePerHour float64) error {
	now := time.Now()
	nowHour := now.Hour()
	var atNight = nowHour >= TODAY_PM_11 && nowHour < TOMORROW_AM_7

	// if katel is running and it is in winter mode then switch the heatpump off
	var katelRunning = !b.EmodulIsDamped()
	var katelInWinterMode = false
	if katelRunning && katelInWinterMode && b.heatPumpHeating.State != "off" {
		// TODO: switch the heatpump off
	}

	// if at night and heatpump's water heating is running
	var waterHeaterRunning = b.heatPumpWaterHeater.State != "off"
	if atNight && waterHeaterRunning {
		// then do not heat because water is not heating otherwise
		// if b.heatPumpHeating.State != "off" {
		// 	b.lo.Info("HeatPump setting heating off", "water tank running")
		// 	err := heatpump.SetHeating(b.ha, homeassistant.CLIMATE_OFF)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	return nil
		// }
	}

	// else turn heatpump on or off based of current hour price

	// remove the last 3 maximum hours per day and max price from there
	ascending := nordpool.OrderHoursAscending(todayPrices)
	lastFour := ascending[len(ascending)-4:]
	maxPrice := lastFour[0]

	currentPrice := todayPrices[nowHour]
	maxPriceAbove := currentPrice > MAX_PRICE_PER_HOUR
	if b.heapPumpHeatingIgnoreMaxPricePerHour.State == "on" {
		maxPriceAbove = false
	}

	// if current price is bigger than we allow then stop the heater
	var heatingOff = (currentPrice > maxPrice) || maxPriceAbove
	var heatingNote = "on"
	if heatingOff {
		heatingNote = "off"
	}
	b.lo.Info(fmt.Sprintf("HeatPump setting heating %v", heatingNote), "nowHour", nowHour, "currentPrice", currentPrice, "max price from today", maxPrice, "hard max price", MAX_PRICE_PER_HOUR, "ignore max price per hour", b.heapPumpHeatingIgnoreMaxPricePerHour.State)
	if heatingOff {
		if b.heatPumpHeating.State != "off" {
			b.lo.Info("HeatPump", "setting heating off")
			err := heatpump.SetHeating(b.ha, homeassistant.CLIMATE_OFF)
			if err != nil {
				return err
			}
		}
	} else {
		if b.heatPumpHeating.State != "heat" {
			b.lo.Info("HeatPump", "setting heating on")
			err := heatpump.SetHeating(b.ha, homeassistant.CLIMATE_HEAT)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *brain) SetHeatPumpTemperature(todayPrices []float64) error {
	now := time.Now()
	nowHour := now.Hour()
	currentPrice := todayPrices[nowHour]

	// check outside temperature
	outsideTemp := b.emodulOutsideTemp
	var newTemp float32 = 0
	if outsideTemp < -15 {
		newTemp = 10
	} else if outsideTemp < -10 {
		newTemp = 8
	} else if outsideTemp < -5 {
		newTemp = 6
	} else if outsideTemp < 0 {
		newTemp = 4
	}

	// do not heat that much at night
	if nowHour >= 0 {
		if nowHour <= 4 {
			newTemp -= 4
		} else if nowHour <= 5 {
			newTemp -= 2
		}
	}

	// if water tank is running then take the heating to the lows to enable water tank heating
	// if b.heatPumpWaterTank.State == "on" {
	// 	newTemp = 0
	// }

	// just validate that we are not out of bounds
	if newTemp < -10 {
		newTemp = -10
	} else if newTemp > 10 {
		newTemp = 10
	}
	// TODO. Take into account if it's sunny day then lower the newTemp
	// TODO. Take into account "elutuba" thermostat temp

	// if current price is bigger than we allow then stop the heater
	b.lo.Info("HeatPump temperature", "water tank running", b.heatPumpWaterHeater.State, "outside temperature", outsideTemp, "new temperature", newTemp, "nowHour", nowHour, "currentPrice", currentPrice)
	if b.heatPumpHeatingTemp != newTemp {
		err := heatpump.SetHeatingTemperature(b.ha, newTemp, nil)
		if err != nil {
			return err
		}
		b.heatPumpHeatingTemp = newTemp
	}
	return nil
}
