package brain

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/emodul"
	"github.com/jaanek/hassext/floorheating"
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
	tempElutuba := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(floorheating.THERMOSTAT_ELUTUBA_WALL), string(floorheating.THERMOSTAT_ELUTUBA_WALL), "hassext/"+string(floorheating.THERMOSTAT_ELUTUBA_WALL)+"_temp")
	tempEsik := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(floorheating.THERMOSTAT_ESIK), string(floorheating.THERMOSTAT_ESIK), "hassext/"+string(floorheating.THERMOSTAT_ESIK)+"_temp")
	tempDussiruum := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(floorheating.THERMOSTAT_DUSSIRUUM), string(floorheating.THERMOSTAT_DUSSIRUUM), "hassext/"+string(floorheating.THERMOSTAT_DUSSIRUUM)+"_temp")
	tempSaunaEesruum := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(floorheating.THERMOSTAT_SAUNA_EESRUUM), string(floorheating.THERMOSTAT_SAUNA_EESRUUM), "hassext/"+string(floorheating.THERMOSTAT_SAUNA_EESRUUM)+"_temp")
	// target temperatures set
	tempTargetElutuba := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(floorheating.THERMOSTAT_ELUTUBA_WALL_TARGET), string(floorheating.THERMOSTAT_ELUTUBA_WALL_TARGET), "hassext/"+string(floorheating.THERMOSTAT_ELUTUBA_WALL_TARGET)+"_temp")
	tempTargetEsik := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(floorheating.THERMOSTAT_ESIK_TARGET), string(floorheating.THERMOSTAT_ESIK_TARGET), "hassext/"+string(floorheating.THERMOSTAT_ESIK_TARGET)+"_temp")
	tempTargetDussiruum := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(floorheating.THERMOSTAT_DUSSIRUUM_TARGET), string(floorheating.THERMOSTAT_DUSSIRUUM_TARGET), "hassext/"+string(floorheating.THERMOSTAT_DUSSIRUUM_TARGET)+"_temp")
	tempTargetSaunaEesruum := emodul.NewMqttTemperatureSensor(b.lo, b.mq, uponor.DeviceUponorWallThermostat, string(floorheating.THERMOSTAT_SAUNA_EESRUUM_TARGET), string(floorheating.THERMOSTAT_SAUNA_EESRUUM_TARGET), "hassext/"+string(floorheating.THERMOSTAT_SAUNA_EESRUUM_TARGET)+"_temp")
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

	// save/update all thermostat values
	b.parseSaveAllTemperatures(uponorData)

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
}

func (b *brain) parseSaveAllTemperatures(uponorData *uponor.UponorControllerData) {
	for _, v := range uponorData.Output.Vars {
		parseSaveTemperatures(b.lo, ESIK, v, &b.uponorEsik, floorheating.THERMOSTAT_ESIK)
		parseSaveTemperatures(b.lo, ELUTUBA, v, &b.uponorElutuba, floorheating.THERMOSTAT_ELUTUBA_WALL)
		parseSaveTemperatures(b.lo, SAUNA_EESRUUM, v, &b.uponorSaunaEesruum, floorheating.THERMOSTAT_SAUNA_EESRUUM)
		parseSaveTemperatures(b.lo, DUSSIRUUM, v, &b.uponorDussiruum, floorheating.THERMOSTAT_DUSSIRUUM)
	}
	// open local sqlite db
	db, err := b.openAppDatabase()
	if err != nil {
		b.lo.Warn("No app database can be opened!", "error", err)
		return
	}
	defer db.Close()
	// save parsed temperature values to db
	var now = time.Now()
	b.saveTemperatureDB(db, &b.uponorEsik, floorheating.THERMOSTAT_ESIK, now)
	b.saveTemperatureDB(db, &b.uponorElutuba, floorheating.THERMOSTAT_ELUTUBA_WALL, now)
	b.saveTemperatureDB(db, &b.uponorSaunaEesruum, floorheating.THERMOSTAT_SAUNA_EESRUUM, now)
	b.saveTemperatureDB(db, &b.uponorDussiruum, floorheating.THERMOSTAT_DUSSIRUUM, now)
}

func publishTemperature(lo logf.Logger, ts *ClimateState, sensor emodul.Sensor) error {
	lo.Info("Uponor thermostat", ts.Id, ts)
	var err = sensor.PublishData(context.Background(), float32(ts.CurrentTemperature))
	if err != nil {
		lo.Error("Error while publishing uponor temperature to mq!", "thermostat", ts, "value", ts.CurrentTemperature)
		return err
	}
	return nil
}

func publishTargetTemperature(lo logf.Logger, ts *ClimateState, sensor emodul.Sensor) error {
	lo.Info("Uponor thermostat", ts.Id, ts)
	var err = sensor.PublishData(context.Background(), float32(ts.SetTemperature))
	if err != nil {
		lo.Error("Error while publishing uponor target temperature to mq!", "thermostat", ts, "target value", ts.SetTemperature)
		return err
	}
	return nil
}

func parseSaveTemperatures(lo logf.Logger, room string, v uponor.UponorWaspVar, t *ClimateState, name floorheating.ThermostatName) {
	t.Id = string(name)
	if v.VarName == room+"_room_temperature" {
		fahrenheit, err := strconv.ParseFloat(v.VarValue, 64)
		if err != nil {
			lo.Error("Error while converting string value to float! VarName: %v", v.VarName)
		}
		fahrenheit = fahrenheit / 10 // because data contains the value like 743 not 74.3
		var temperature = uponor.FahrenheitToCelsius(fahrenheit)
		t.CurrentTemperature = math.Round(temperature*10) / 10 // homeassistant.Truncate(temperature, 1)
		lo.Info("Setting room temperature", room, temperature, "original", v.VarValue, "parsed fahrenheit", fahrenheit)
	} else if v.VarName == room+"_setpoint" {
		fahrenheit, err := strconv.ParseFloat(v.VarValue, 32)
		if err != nil {
			lo.Error("Error while converting string value to float! VarName: %v", v.VarName)
		}
		fahrenheit = fahrenheit / 10 // because data contains the value like 743 not 74.3
		var temperature = uponor.FahrenheitToCelsius(fahrenheit)
		lo.Info("Setting setpoint temperature", room, temperature, "original", v.VarValue, "parsed fahrenheit", fahrenheit)
		t.SetTemperature = math.Round(temperature*10) / 10 // homeassistant.Truncate(temperature, 1)
	}
}
