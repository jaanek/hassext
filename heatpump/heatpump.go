package heatpump

import "github.com/jaanek/hassext/homeassistant"

type EntityState string

const (
	ENTITY_HEATING          EntityState = "climate.altherma"
	ENTITY_WATER_TANK       EntityState = "switch.altherma_tank"
	ENTITY_WATER_TANK_BOOST EntityState = "switch.altherma_boost"
)

type InputEntity string

const (
	ENTITY_TIME_HEATING_START InputEntity = "input_datetime.soojuspump_kyte_start"
	ENTITY_TIME_HEATING_STOP  InputEntity = "input_datetime.soojuspump_kyte_stop"
	ENTITY_TIME_WATER_START   InputEntity = "input_datetime.soojuspump_vesi_start"
	ENTITY_TIME_WATER_STOP    InputEntity = "input_datetime.soojuspump_vesi_stop"
)

type AutomationEntity string

const (
	ENTITY_HEATING_START    AutomationEntity = "automation.kaivita_soojuspump"
	ENTITY_HEATING_STOP                      = "automation.seiska_soojuspump"
	ENTITY_WATER_TANK_START                  = "automation.daikin_kaivita_soojuspump_vesi"
	ENTITY_WATER_TANK_STOP                   = "automation.seiska_soojuspump_vesi"
)

type WaterTankMode string

const (
	WaterTank  WaterTankMode = "switch.altherma_tank"
	WaterBoost WaterTankMode = "switch.altherma_boost"
)

func SetWaterTank(ha homeassistant.Switch, entity WaterTankMode, action homeassistant.SwitchAction) error {
	return ha.Switch(string(entity), action)
}

const HeatingEntity = "climate.altherma"

func SetHeating(ha homeassistant.Climate, mode homeassistant.ClimateHvacMode) error {
	return ha.ClimateSetHvacMode(HeatingEntity, mode)
}

func SetHeatingTemperature(ha homeassistant.Climate, temp float32, mode *homeassistant.ClimateHvacMode) error {
	return ha.ClimateSetTemperature(HeatingEntity, temp, mode)
}

// example: Automation(ha, ENTITY_WATER_TANK_START, AUTOMATION_TURN_ON)
func Automation(ha homeassistant.Automation, entity AutomationEntity, action homeassistant.AutomationAction) error {
	return ha.Automation(string(entity), action)
}
