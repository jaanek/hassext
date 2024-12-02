package floorheating

type ThermostatName string
type ThermostatTargetName string

const (
	// target temperatures for uponor wall thermostats
	THERMOSTAT_ELUTUBA_TARGET       ThermostatTargetName = "floor1_elutuba_target"
	THERMOSTAT_ESIK_TARGET          ThermostatTargetName = "floor1_esik_target"
	THERMOSTAT_DUSSIRUUM_TARGET     ThermostatTargetName = "floor1_dussiruum_target"
	THERMOSTAT_SAUNA_EESRUUM_TARGET ThermostatTargetName = "floor1_sauna_eesruum_target"
	// thermostat names
	THERMOSTAT_ELUTUBA_WALL  ThermostatName = "floor1_elutuba"
	THERMOSTAT_ELUTUBA_SOFA  ThermostatName = "floor1_elutuba_sofa"
	THERMOSTAT_ESIK          ThermostatName = "floor1_esik"
	THERMOSTAT_DUSSIRUUM     ThermostatName = "floor1_dussiruum"
	THERMOSTAT_SAUNA_EESRUUM ThermostatName = "floor1_sauna_eesruum"
)
