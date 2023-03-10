package heatpump

import "github.com/jaanek/hassext/homeassistant"

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
