package floorheating

type ThermostatName string
type ThermostatTargetName string

const (
	// thermostat names
	THERMOSTAT_ELUTUBA_WALL           ThermostatName = "floor1_elutuba"
	THERMOSTAT_ELUTUBA_SOFA           ThermostatName = "korrus1_elutuba_temperature_sensor"
	THERMOSTAT_ESIK                   ThermostatName = "floor1_esik"
	THERMOSTAT_DUSSIRUUM              ThermostatName = "floor1_dussiruum"
	THERMOSTAT_SAUNA_EESRUUM          ThermostatName = "floor1_sauna_eesruum"
	THERMOSTAT_SUUR_KORIDOR_WORKPLACE ThermostatName = "korrus1_suur_koridor_temperature_sensor1"
	// target temperatures for uponor wall thermostats
	THERMOSTAT_ELUTUBA_WALL_TARGET           ThermostatTargetName = "floor1_elutuba_target"
	THERMOSTAT_ELUTUBA_SOFA_TARGET           ThermostatTargetName = ThermostatTargetName(THERMOSTAT_ELUTUBA_SOFA)
	THERMOSTAT_ESIK_TARGET                   ThermostatTargetName = "floor1_esik_target"
	THERMOSTAT_DUSSIRUUM_TARGET              ThermostatTargetName = "floor1_dussiruum_target"
	THERMOSTAT_SAUNA_EESRUUM_TARGET          ThermostatTargetName = "floor1_sauna_eesruum_target"
	THERMOSTAT_SUUR_KORIDOR_WORKPLACE_TARGET ThermostatTargetName = ThermostatTargetName(THERMOSTAT_SUUR_KORIDOR_WORKPLACE)
)

var FloorHeatingSimpleThermostats = []ThermostatName{
	THERMOSTAT_ELUTUBA_SOFA,
	THERMOSTAT_SUUR_KORIDOR_WORKPLACE,
}
var FloorHeatingSimpleThermostatTargets = []ThermostatTargetName{
	THERMOSTAT_ELUTUBA_SOFA_TARGET,
	THERMOSTAT_SUUR_KORIDOR_WORKPLACE_TARGET,
}
