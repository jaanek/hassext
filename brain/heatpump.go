package brain

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/heatpump"
	"github.com/jaanek/hassext/homeassistant"
	"github.com/jaanek/hassext/nordpool"
)

func (b *brain) HeatPump(state data.DataValue) {
	// water tank automation start
	// entity, err := ParseEntityState(string(heatpump.ENTITY_AUTOMATION_WATER_HEATER_START), state)
	// if err != nil {
	// 	b.errors <- err
	// } else {
	// 	b.heatPumpWaterHeaterAutomationStart = *entity
	// 	b.lo.Info("HeatPump", "automation water tank start", *entity)
	// }
	// // water heater automation stop
	// entity, err = ParseEntityState(string(heatpump.ENTITY_AUTOMATION_WATER_HEATER_STOP), state)
	// if err != nil {
	// 	b.errors <- err
	// } else {
	// 	b.heatPumpWaterHeaterAutomationStop = *entity
	// 	b.lo.Info("HeatPump", "automation water tank stop", *entity)
	// }
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
	entity, err := ParseEntityState(string(heatpump.ENTITY_TIME_WATER_HEATER_START), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpWaterHeaterStartState = *entity
		b.lo.Info("HeatPump", "water tank start state", *entity)
	}
	// water tank stop time
	entity, err = ParseEntityState(string(heatpump.ENTITY_TIME_WATER_HEATER_STOP), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heatPumpWaterHeaterStopState = *entity
		b.lo.Info("HeatPump", "water tank stop state", *entity)
	}
	// heating ignore max price per hour
	entity, err = ParseEntityState(string(heatpump.ENTITY_BOOL_IGNORE_MAX_PRICE_PER_HOUR), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heapPumpHeatingIgnoreMaxPricePerHour = *entity
		b.lo.Info("HeatPump", "heating ignore max price per hour", *entity)
	}
	// trigger heating active
	entity, err = ParseEntityState(string(heatpump.ENTITY_BOOL_TRIGGER_HEATING_ACTIVE), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heapPumpTriggerHeatingActive = *entity
		b.lo.Info("HeatPump", "trigger heating active", *entity)
	}
	// trigger water heater active
	entity, err = ParseEntityState(string(heatpump.ENTITY_BOOL_TRIGGER_WATERHEATER_ACTIVE), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heapPumpTriggerWaterHeaterActive = *entity
		b.lo.Info("HeatPump", "trigger waterheater active", *entity)
	}
	// trigger water heater boost
	entity, err = ParseEntityState(string(heatpump.ENTITY_BOOL_TRIGGER_WATERHEATER_BOOST), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heapPumpTriggerWaterHeaterBoost = *entity
		b.lo.Info("HeatPump", "trigger waterheater boost", *entity)
	}
	// trigger heating set temp
	entity, err = ParseEntityState(string(heatpump.ENTITY_NUMBER_TRIGGER_HEATER_SET_TEMP), state)
	if err != nil {
		b.errors <- err
	} else {
		b.heapPumpTriggerHeatingSetTemp = *entity
		b.lo.Info("HeatPump", "trigger heating set temp", *entity)
	}
	// heating temp shift
	entity, err = ParseEntityState(string(heatpump.ENTITY_NUMBER_HEATER_TEMP_SHIFT), state)
	if err != nil {
		b.errors <- err
	} else {
		temp, err := strconv.ParseFloat((*entity).State, 32) // example: "state": "2"
		if err != nil {
			b.errors <- err
		} else {
			b.heatPumpHeatingTempShift = int32(temp)
		}
		b.lo.Info("HeatPump", "trigger heating set temp", *entity)
	}
}

// func (b *brain) SetHeatPumpAutomationsOnOff() error {
// 	var setOff = !b.EmodulIsDamped()
// 	if setOff {
// 		// turn off automation because katel is doing periodic heating
// 		if b.heatPumpWaterHeaterAutomationStart.State != "off" {
// 			b.lo.Info("HeatPump", "katel running", setOff, "turn OFF water tank start automation")
// 			err := heatpump.Automation(b.ha, heatpump.ENTITY_AUTOMATION_WATER_HEATER_START, homeassistant.AUTOMATION_TURN_OFF)
// 			if err != nil {
// 				b.errors <- err
// 			}
// 		}
// 		if b.heatPumpWaterHeaterAutomationStop.State != "off" {
// 			b.lo.Info("HeatPump", "katel running", setOff, "turn OFF water tank stop automation")
// 			err := heatpump.Automation(b.ha, heatpump.ENTITY_AUTOMATION_WATER_HEATER_STOP, homeassistant.AUTOMATION_TURN_OFF)
// 			if err != nil {
// 				b.errors <- err
// 			}
// 		}
// 	} else {
// 		// turn off automation because katel is doing periodic heating
// 		if b.heatPumpWaterHeaterAutomationStart.State != "on" {
// 			b.lo.Info("HeatPump", "katel running", setOff, "turn ON water tank start automation")
// 			err := heatpump.Automation(b.ha, heatpump.ENTITY_AUTOMATION_WATER_HEATER_START, homeassistant.AUTOMATION_TURN_ON)
// 			if err != nil {
// 				b.errors <- err
// 			}
// 		}
// 		if b.heatPumpWaterHeaterAutomationStop.State != "on" {
// 			b.lo.Info("HeatPump", "katel running", setOff, "turn ON water tank stop automation")
// 			err := heatpump.Automation(b.ha, heatpump.ENTITY_AUTOMATION_WATER_HEATER_STOP, homeassistant.AUTOMATION_TURN_ON)
// 			if err != nil {
// 				b.errors <- err
// 			}
// 		}
// 	}
// 	return nil
// }

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
const (
	VERY_HIGH_PRICE_LEVEL   = 200
	PRETTY_HIGH_PRICE_LEVEL = 160
	HIGH_PRICE_LEVEL        = 120
	MIDDLE_PRICE_LEVEL      = 80
	MIDDLE_PRICE_LEVEL2     = 60
	LOW_PRICE_LEVEL         = 40
	VERY_LOW_PRICE_LEVEL    = 20
)
const (
	TEMP_LEVEL_24_LOW float64 = 24.2
	TEMP_LEVEL_22_LOW float64 = 22.2
	TEMP_LEVEL_20_LOW         = 20.2
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
	// var atNight = nowHour >= TODAY_PM_11 && nowHour < TOMORROW_AM_7

	// if katel is running and it is in winter mode then switch the heatpump off
	var katelRunning = !b.EmodulIsDamped()
	var katelInWinterMode = false
	if katelRunning && katelInWinterMode && b.heatPumpHeating.State != "off" {
		// TODO: switch the heatpump off
	}

	// remove the last 3 maximum hours per day and max price from there
	ascending := nordpool.OrderHoursAscending(todayPrices)
	lastFour := ascending[len(ascending)-4:]
	maxPrice := lastFour[0]

	currentPrice := todayPrices[nowHour]
	maxPriceAbove := currentPrice > MAX_PRICE_PER_HOUR

	// check if we need to trigger waterheater start
	var waterHeaterStartTime, waterHeaterStopTime *time.Time
	if b.heatPumpWaterHeaterStartState.State != "" {
		t, err := time.Parse("15:04:05", b.heatPumpWaterHeaterStartState.State)
		if err != nil {
			return fmt.Errorf("HeatPump error while parsing start time: %w", err)
		} else {
			waterHeaterStartTime = &t
		}
	}
	if b.heatPumpWaterHeaterStopState.State != "" {
		t, err := time.Parse("15:04:05", b.heatPumpWaterHeaterStopState.State)
		if err != nil {
			return fmt.Errorf("HeatPump error while parsing stop time: %w", err)
		} else {
			waterHeaterStopTime = &t
		}
	}
	if waterHeaterStartTime != nil && waterHeaterStopTime != nil {
		now := time.Now()
		var start, stop = *waterHeaterStartTime, *waterHeaterStopTime
		var betweenTheStartStop = start.Hour() <= now.Hour() && stop.Hour() >= now.Hour()
		if betweenTheStartStop {
			b.lo.Info("HeatPump trigger water heater ON, we are in the water heater schedule between start/stop", "start", start.Format("15:04:05"), "stop", stop.Format("15:04:05"), "current hour", now.Hour())
			err := b.ha.SetInputBoolean(string(heatpump.ENTITY_BOOL_TRIGGER_WATERHEATER_ACTIVE), homeassistant.BOOLEAN_TURN_ON)
			if err != nil {
				return err
			}
		} else {
			b.lo.Info("HeatPump trigger water heater OFF, we are NOT in the water heater schedule between start/stop", "start", start.Format("15:04:05"), "stop", stop.Format("15:04:05"), "current hour", now.Hour())
			err := b.ha.SetInputBoolean(string(heatpump.ENTITY_BOOL_TRIGGER_WATERHEATER_ACTIVE), homeassistant.BOOLEAN_TURN_OFF)
			if err != nil {
				return err
			}
		}
	}

	// if heatpump's water heating is running and the price is low
	var waterHeaterRunning = b.heatPumpWaterHeater.State != "off"
	if waterHeaterRunning {
		var isInBoostMode = b.heatPumpWaterHeater.State == "performance"
		if currentPrice < LOW_PRICE_LEVEL {
			if !isInBoostMode {
				b.lo.Info("HeatPump water heater running, boosting it because it's low price", "current price", currentPrice, "hour", nowHour)
				err := b.ha.SetInputBoolean(string(heatpump.ENTITY_BOOL_TRIGGER_WATERHEATER_BOOST), homeassistant.BOOLEAN_TURN_ON)
				if err != nil {
					return err
				}
				// err := heatpump.SetWaterHeaterMode(b.ha, homeassistant.WATER_HEATER_BOOST)
				// if err != nil {
				// 	return err
				// }
			}
		} else if isInBoostMode {
			b.lo.Info("HeatPump water heater running and in boost mode but it's not low price, setting back to normal", "current price", currentPrice, "hour", nowHour)
			err := b.ha.SetInputBoolean(string(heatpump.ENTITY_BOOL_TRIGGER_WATERHEATER_BOOST), homeassistant.BOOLEAN_TURN_OFF)
			if err != nil {
				return err
			}
			// err := heatpump.SetWaterHeaterMode(b.ha, homeassistant.WATER_HEATER_HEAT_PUMP)
			// if err != nil {
			// 	return err
			// }
		}

		// then do not heat because water is not heating otherwise
		// if b.heatPumpHeating.State != "off" {
		// 	b.lo.Info("HeatPump setting heating off", "water tank running")
		// 	err := heatpump.SetHeating(b.ha, homeassistant.CLIMATE_OFF)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	return nil
		// }
	} else if b.heapPumpTriggerWaterHeaterBoost.State == "on" {
		// turn off also the boost because water heater is not running
		err := b.ha.SetInputBoolean(string(heatpump.ENTITY_BOOL_TRIGGER_WATERHEATER_BOOST), homeassistant.BOOLEAN_TURN_OFF)
		if err != nil {
			return err
		}
	}

	// else turn heatpump on or off based of current hour price

	// if current price is bigger than we allow then stop the heater
	var ignoreMaxPriceCuts = b.heapPumpHeatingIgnoreMaxPricePerHour.State == "on"
	var heatingOff = ((currentPrice > maxPrice) || maxPriceAbove) && !ignoreMaxPriceCuts
	var heatingNote = "on"
	if heatingOff {
		heatingNote = "off"
	}
	b.lo.Info(fmt.Sprintf("HeatPump setting heating %v", heatingNote), "nowHour", nowHour, "currentPrice", currentPrice, "max price from today", maxPrice, "hard max price", MAX_PRICE_PER_HOUR, "ignore max price per hour", b.heapPumpHeatingIgnoreMaxPricePerHour.State)
	if heatingOff {
		if b.heatPumpHeating.State != "off" {
			b.lo.Info("HeatPump", "setting heating off")
			err := b.ha.SetInputBoolean(string(heatpump.ENTITY_BOOL_TRIGGER_HEATING_ACTIVE), homeassistant.BOOLEAN_TURN_OFF)
			if err != nil {
				return err
			}
			// err := heatpump.SetHeating(b.ha, homeassistant.CLIMATE_OFF)
			// if err != nil {
			// 	return err
			// }
		}
	} else {
		if b.heatPumpHeating.State != "heat" {
			b.lo.Info("HeatPump", "setting heating on")
			err := b.ha.SetInputBoolean(string(heatpump.ENTITY_BOOL_TRIGGER_HEATING_ACTIVE), homeassistant.BOOLEAN_TURN_ON)
			if err != nil {
				return err
			}
			// err := heatpump.SetHeating(b.ha, homeassistant.CLIMATE_HEAT)
			// if err != nil {
			// 	return err
			// }
		}
	}
	return nil
}

func (b *brain) SetHeatPumpTemperature(todayPrices []float64) error {
	now := time.Now()
	nowHour := now.Hour()
	nowMonth := now.Month()
	currentPrice := todayPrices[nowHour]
	var newTemp float32 = 0
	outsideTemp := b.emodulOutsideTemp

	// check outside temperature
	// if outsideTemp < -15 {
	// 	newTemp = 8
	// } else if outsideTemp < -10 {
	// 	newTemp = 6
	// } else if outsideTemp < -5 {
	// 	newTemp = 4
	// } else if outsideTemp < 0 {
	// 	newTemp = 2
	// }
	// b.lo.Info("HeatPump set temperature [1]", "newTemp", newTemp, "nowHour", nowHour, "outside temp", outsideTemp)

	isMiddlePrice := currentPrice <= MIDDLE_PRICE_LEVEL
	isMiddlePrice2 := currentPrice <= MIDDLE_PRICE_LEVEL2
	isLowPrice := currentPrice <= LOW_PRICE_LEVEL
	isVeryLowPrice := currentPrice <= VERY_LOW_PRICE_LEVEL
	isBelow24 := b.uponorElutuba.CurrentTemperature < TEMP_LEVEL_24_LOW
	isBelow22 := b.uponorElutuba.CurrentTemperature < TEMP_LEVEL_22_LOW
	isBelow20 := b.uponorElutuba.CurrentTemperature < TEMP_LEVEL_20_LOW
	isAfternoon := nowHour >= 13 && nowHour < 18
	isWinter := nowMonth <= 4 || nowMonth >= 10

	// check price and thermostat
	switch {
	case isVeryLowPrice && isBelow24:
		// boost the heater despite the hours
		newTemp = 10
		b.lo.Info("HeatPump set temperature [1]", "newTemp", newTemp, "nowHour", nowHour, "price below", VERY_LOW_PRICE_LEVEL, "current price", currentPrice, "current temperature", b.uponorElutuba.CurrentTemperature)
	case isBelow20:
		// we need to boost because it's cold inside
		switch {
		case isLowPrice:
			newTemp = 8
		default:
			newTemp = 6
		}
		b.lo.Info("HeatPump set temperature [2]", "newTemp", newTemp, "nowHour", nowHour, "temp below", TEMP_LEVEL_20_LOW, "current temperature", b.uponorElutuba.CurrentTemperature)
	case isBelow22:
		switch {
		case isLowPrice:
			newTemp = 6
		default:
			newTemp = 4
		}
		b.lo.Info("HeatPump set temperature [3]", "newTemp", newTemp, "nowHour", nowHour, "temp below", TEMP_LEVEL_22_LOW, "current temperature", b.uponorElutuba.CurrentTemperature)
	case isBelow24:
		switch {
		// we want to heat up before the evening
		case isWinter && isAfternoon && isMiddlePrice:
			newTemp = 2
		case isWinter && isAfternoon && isMiddlePrice2:
			newTemp = 4
		}
		b.lo.Info("HeatPump set temperature [4]", "newTemp", newTemp, "isWinter", isWinter, "isAfternoon", isAfternoon, "current temperature", b.uponorElutuba.CurrentTemperature, "currentPrice", currentPrice)
	}

	// just validate that we are not out of bounds
	if newTemp < -10 {
		newTemp = -10
	} else if newTemp > 10 {
		newTemp = 10
	}

	// check if there is temp shift set
	if b.heatPumpHeatingTempShift != 0 {
		newTemp += float32(b.heatPumpHeatingTempShift)
		b.lo.Info("HeatPump set temperature [4]", "newTemp", newTemp, "set temp shift", b.heatPumpHeatingTempShift)
	}

	// if current price is bigger than we allow then stop the heater
	b.lo.Info("HeatPump set temperature", "newTemp", newTemp, "heater SetTemp", b.heatPumpHeating.SetTemperature, "nowHour", nowHour, "water tank running", b.heatPumpWaterHeater.State, "outside temperature", outsideTemp, "currentPrice", currentPrice)
	if uint64(b.heatPumpHeating.SetTemperature) != uint64(newTemp) {
		err := b.ha.SetInputNumberValue(string(heatpump.ENTITY_NUMBER_TRIGGER_HEATER_SET_TEMP), uint64(newTemp))
		if err != nil {
			return err
		}
		// err := heatpump.SetHeatingTemperature(b.ha, newTemp, nil)
		// if err != nil {
		// 	return err
		// }
	}
	return nil
}
