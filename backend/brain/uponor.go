package brain

import (
	"context"
	"math"
	"strconv"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/emodul"
	"github.com/jaanek/hassext/uponor"
	"github.com/zerodha/logf"
)

const (
	ESIK          = "C1_T1"
	ELUTUBA       = "C1_T2"
	SAUNA_EESRUUM = "C1_T3"
	DUSSIRUUM     = "C1_T4"
)

func (b *brain) Uponor(state data.DataValue) {
	// current temperatures reading
	tempElutuba := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(uponor.THERMOSTAT_ELUTUBA), string(uponor.THERMOSTAT_ELUTUBA), "hassext/"+string(uponor.THERMOSTAT_ELUTUBA)+"_temp")
	tempEsik := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(uponor.THERMOSTAT_ESIK), string(uponor.THERMOSTAT_ESIK), "hassext/"+string(uponor.THERMOSTAT_ESIK)+"_temp")
	tempDussiruum := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(uponor.THERMOSTAT_DUSSIRUUM), string(uponor.THERMOSTAT_DUSSIRUUM), "hassext/"+string(uponor.THERMOSTAT_DUSSIRUUM)+"_temp")
	tempSaunaEesruum := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(uponor.THERMOSTAT_SAUNA_EESRUUM), string(uponor.THERMOSTAT_SAUNA_EESRUUM), "hassext/"+string(uponor.THERMOSTAT_SAUNA_EESRUUM)+"_temp")
	// target temperatures set
	tempTargetElutuba := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(uponor.THERMOSTAT_ELUTUBA_TARGET), string(uponor.THERMOSTAT_ELUTUBA_TARGET), "hassext/"+string(uponor.THERMOSTAT_ELUTUBA_TARGET)+"_temp")
	tempTargetEsik := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(uponor.THERMOSTAT_ESIK_TARGET), string(uponor.THERMOSTAT_ESIK_TARGET), "hassext/"+string(uponor.THERMOSTAT_ESIK_TARGET)+"_temp")
	tempTargetDussiruum := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(uponor.THERMOSTAT_DUSSIRUUM_TARGET), string(uponor.THERMOSTAT_DUSSIRUUM_TARGET), "hassext/"+string(uponor.THERMOSTAT_DUSSIRUUM_TARGET)+"_temp")
	tempTargetSaunaEesruum := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(uponor.THERMOSTAT_SAUNA_EESRUUM_TARGET), string(uponor.THERMOSTAT_SAUNA_EESRUUM_TARGET), "hassext/"+string(uponor.THERMOSTAT_SAUNA_EESRUUM_TARGET)+"_temp")
	sensors := make([]emodul.Sensor, 0)
	sensors = append(
		sensors,
		tempElutuba,
		tempEsik,
		tempDussiruum,
		tempSaunaEesruum,
		tempTargetElutuba,
		tempTargetEsik,
		tempTargetDussiruum,
		tempTargetSaunaEesruum,
	)
	// send configs
	for _, sensor := range sensors {
		err := sensor.PublishConfig(context.Background())
		if err != nil {
			b.lo.Error("Uponor config mqtt publish", "error", err)
		}
	}

	// send thermostat data
	uponorData := b.uponorClient.GetData()
	if len(uponorData.Output.Vars) <= 0 {
		b.lo.Warn("No uponor controller data available to fetch temperatures from!")
		return
	}
	for _, v := range uponorData.Output.Vars {
		parseSaveTemperatures(b.lo, ESIK, v, &b.uponorEsik)
		parseSaveTemperatures(b.lo, ELUTUBA, v, &b.uponorElutuba)
		parseSaveTemperatures(b.lo, SAUNA_EESRUUM, v, &b.uponorSaunaEesruum)
		parseSaveTemperatures(b.lo, DUSSIRUUM, v, &b.uponorDussiruum)
	}
	// publish current temperatures
	publishTemperature(b.lo, &b.uponorEsik, tempEsik)
	publishTemperature(b.lo, &b.uponorElutuba, tempElutuba)
	publishTemperature(b.lo, &b.uponorSaunaEesruum, tempSaunaEesruum)
	publishTemperature(b.lo, &b.uponorDussiruum, tempDussiruum)
	// publish target temperatures
	publishTargetTemperature(b.lo, &b.uponorEsik, tempTargetEsik)
	publishTargetTemperature(b.lo, &b.uponorElutuba, tempTargetElutuba)
	publishTargetTemperature(b.lo, &b.uponorSaunaEesruum, tempTargetSaunaEesruum)
	publishTargetTemperature(b.lo, &b.uponorDussiruum, tempTargetDussiruum)

	// // elutuba
	// thermostat, err := ParseThermostatState(string(uponor.ENTITY_THERMOSTAT_ELUTUBA), state)
	// if err != nil {
	// 	b.errors <- err
	// } else {
	// 	var ts = *thermostat
	// 	b.uponorElutuba = ts
	// 	b.lo.Info("Uponor thermostat", "elutuba", ts)
	// 	err = tempElutuba.PublishData(context.Background(), float32(ts.CurrentTemperature))
	// 	if err != nil {
	// 		b.lo.Error("Error while publishing uponor temperature to mq!", "thermostat", uponor.THERMOSTAT_ELUTUBA, "value", ts.CurrentTemperature)
	// 	}
	// }
	// // esik
	// thermostat, err = ParseThermostatState(string(uponor.ENTITY_THERMOSTAT_ESIK), state)
	// if err != nil {
	// 	b.errors <- err
	// } else {
	// 	var ts = *thermostat
	// 	b.uponorEsik = ts
	// 	b.lo.Info("Uponor thermostat", "esik", ts)
	// 	err = tempEsik.PublishData(context.Background(), float32(ts.CurrentTemperature))
	// 	if err != nil {
	// 		b.lo.Error("Error while publishing uponor temperature to mq!", "thermostat", uponor.THERMOSTAT_ESIK, "value", ts.CurrentTemperature)
	// 	}
	// }
	// // dussiruum
	// thermostat, err = ParseThermostatState(string(uponor.ENTITY_THERMOSTAT_DUSSIRUUM), state)
	// if err != nil {
	// 	b.errors <- err
	// } else {
	// 	var ts = *thermostat
	// 	b.uponorDussiruum = ts
	// 	b.lo.Info("Uponor thermostat", "dussiruum", ts)
	// 	err = tempDussiruum.PublishData(context.Background(), float32(ts.CurrentTemperature))
	// 	if err != nil {
	// 		b.lo.Error("Error while publishing uponor temperature to mq!", "thermostat", uponor.THERMOSTAT_DUSSIRUUM, "value", ts.CurrentTemperature)
	// 	}
	// }
	// // sauna eesruum
	// thermostat, err = ParseThermostatState(string(uponor.ENTITY_THERMOSTAT_SAUNA_EESRUUM), state)
	// if err != nil {
	// 	b.errors <- err
	// } else {
	// 	var ts = *thermostat
	// 	b.uponorSaunaEesruum = ts
	// 	b.lo.Info("Uponor thermostat", "sauna eesruum", ts)
	// 	err = tempSaunaEesruum.PublishData(context.Background(), float32(ts.CurrentTemperature))
	// 	if err != nil {
	// 		b.lo.Error("Error while publishing uponor temperature to mq!", "thermostat", uponor.THERMOSTAT_SAUNA_EESRUUM, "value", ts.CurrentTemperature)
	// 	}
	// }
}

func publishTemperature(lo logf.Logger, ts *ThermostatState, sensor emodul.Sensor) error {
	lo.Info("Uponor thermostat", ts.Id, ts)
	var err = sensor.PublishData(context.Background(), float32(ts.CurrentTemperature))
	if err != nil {
		lo.Error("Error while publishing uponor temperature to mq!", "thermostat", ts, "value", ts.CurrentTemperature)
		return err
	}
	return nil
}

func publishTargetTemperature(lo logf.Logger, ts *ThermostatState, sensor emodul.Sensor) error {
	lo.Info("Uponor thermostat", ts.Id, ts)
	var err = sensor.PublishData(context.Background(), float32(ts.SetTemperature))
	if err != nil {
		lo.Error("Error while publishing uponor target temperature to mq!", "thermostat", ts, "target value", ts.SetTemperature)
		return err
	}
	return nil
}

func parseSaveTemperatures(lo logf.Logger, room string, v uponor.UponorWaspVar, t *ThermostatState) {
	if v.VarName == room+"_room_temperature" {
		fahrenheit, err := strconv.ParseFloat(v.VarValue, 32)
		if err != nil {
			lo.Error("Error while converting string value to float! VarName: %v", v.VarName)
		}
		fahrenheit = fahrenheit / 10 // because data contains the value like 743 not 74.3
		var temperature = uponor.FahrenheitToCelsius(fahrenheit)
		t.CurrentTemperature = truncate(temperature, 1)
		lo.Info("Setting room temperature", room, temperature, "original", v.VarValue, "parsed fahrenheit", fahrenheit)
	} else if v.VarName == room+"_setpoint" {
		fahrenheit, err := strconv.ParseFloat(v.VarValue, 32)
		if err != nil {
			lo.Error("Error while converting string value to float! VarName: %v", v.VarName)
		}
		fahrenheit = fahrenheit / 10 // because data contains the value like 743 not 74.3
		var temperature = uponor.FahrenheitToCelsius(fahrenheit)
		lo.Info("Setting setpoint temperature", room, temperature, "original", v.VarValue, "parsed fahrenheit", fahrenheit)
		t.SetTemperature = truncate(temperature, 1)
	}
}

func truncate(f float64, decimals int) float64 {
	shift := math.Pow(10, float64(decimals))
	return math.Trunc(f*shift) / shift
}
