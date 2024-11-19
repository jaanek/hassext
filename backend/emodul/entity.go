package emodul

type InputEntity string

const (
	ENTITY_BOOL_HEATING_ALLOWED InputEntity = "input_boolean.katel_heating_allowed"
	ENTITY_CONTROLLER_STATE     InputEntity = "sensor.controller_state"
	ENTITY_OPERATION_MODE       InputEntity = "sensor.operation_modes"
	ENTITY_EXTERNAL_TEMPERATURE InputEntity = "sensor.external_temperature"
	ENTITY_BOILER_TEMPERATURE   InputEntity = "sensor.dhw_temperature"
)
