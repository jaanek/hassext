package heatpump

import "github.com/jaanek/hassext/homeassistant"

type EntityState string

const (
	ENTITY_HEATING      EntityState = "climate.altherma_leaving_water_offset" // "climate.altherma"
	ENTITY_WATER_HEATER EntityState = "water_heater.altherma"
)

type InputEntity string

const (
	ENTITY_TIME_HEATING_START              InputEntity = "input_datetime.soojuspump_kyte_start"
	ENTITY_TIME_HEATING_STOP               InputEntity = "input_datetime.soojuspump_kyte_stop"
	ENTITY_TIME_WATER_HEATER_START         InputEntity = "input_datetime.soojuspump_vesi_start"
	ENTITY_TIME_WATER_HEATER_STOP          InputEntity = "input_datetime.soojuspump_vesi_stop"
	ENTITY_BOOL_IGNORE_MAX_PRICE_PER_HOUR  InputEntity = "input_boolean.soojuspump_kyte_ignore_max_price_per_hour"
	ENTITY_BOOL_KEEP_WATERHEATER_ACTIVE    InputEntity = "input_boolean.soojuspump_keep_waterheater_active"
	ENTITY_BOOL_KEEP_WATERHEATER_BOOST     InputEntity = "input_boolean.soojuspump_keep_waterheater_boost"
	ENTITY_BOOL_TRIGGER_HEATING_ACTIVE     InputEntity = "input_boolean.soojuspump_trigger_kyte_active"
	ENTITY_BOOL_TRIGGER_WATERHEATER_ACTIVE InputEntity = "input_boolean.soojuspump_trigger_waterheater_active"
	ENTITY_BOOL_TRIGGER_WATERHEATER_BOOST  InputEntity = "input_boolean.soojuspump_trigger_waterheater_boost"
	ENTITY_NUMBER_TRIGGER_HEATER_SET_TEMP  InputEntity = "input_number.soojuspump_trigger_kyte_set_temp"
	ENTITY_NUMBER_HEATER_TEMP_SHIFT        InputEntity = "input_number.soojuspump_kyte_temp_shift"
)

type AutomationEntity string

const (
	ENTITY_AUTOMATION_WATER_HEATER_START AutomationEntity = "automation.daikin_kaivita_soojuspump_vesi"
	ENTITY_AUTOMATION_WATER_HEATER_STOP                   = "automation.seiska_soojuspump_vesi"
)

func SetWaterHeaterMode(ha homeassistant.WaterHeater, mode homeassistant.WaterHeaterOperationMode) error {
	return ha.WaterHeaterSetOperationMode(string(ENTITY_WATER_HEATER), mode)
}

func SetHeating(ha homeassistant.Climate, mode homeassistant.ClimateHvacMode) error {
	return ha.ClimateSetHvacMode(string(ENTITY_HEATING), mode)
}

func SetHeatingTemperature(ha homeassistant.Climate, temp float32, mode *homeassistant.ClimateHvacMode) error {
	return ha.ClimateSetTemperature(string(ENTITY_HEATING), temp, mode)
}

// example: Automation(ha, ENTITY_WATER_TANK_START, AUTOMATION_TURN_ON)
func Automation(ha homeassistant.Automation, entity AutomationEntity, action homeassistant.AutomationAction) error {
	return ha.Automation(string(entity), action)
}
