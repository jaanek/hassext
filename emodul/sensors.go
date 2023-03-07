package emodul

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/mq"
	"github.com/zerodha/logf"
)

type SensorMqttConfig struct {
	Device            SensorMqttConfigDevice `json:"device"`
	DeviceClass       string                 `json:"device_class"`
	UniqueId          string                 `json:"unique_id"`
	UnitOfMeasurement string                 `json:"unit_of_measurement"`
	ValueTemplate     string                 `json:"value_template"`
	EnabledByDefault  bool                   `json:"enabled_by_default"`
	Name              string                 `json:"name"`
	StateClass        string                 `json:"state_class"`
	StateTopic        string                 `json:"state_topic"`
}

type SensorMqttConfigDevice struct {
	Identifiers  []string `json:"identifiers"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
	Name         string   `json:"name"`
}

type BuiltInValve struct {
	description       *string
	valveNumber       *int64
	workingStatus     *bool
	openingPercentage *int64
	currentTemp       *int64
	returnTemp        *int64
	setTemp           *int64
}

type TemperatureSensor struct {
	currentTemp *int64
	targetTemp  *int64
	min         *int64
	max         *int64
}

func (m *emodul) ParseValve(id uint) (*BuiltInValve, error) {
	// check if we have a valve with specified id
	parentPath := fmt.Sprintf("$.tiles[?(@.id == %v)].params", id)
	obj := m.moduleData.GetObject(parentPath)
	if obj == nil {
		return nil, fmt.Errorf(fmt.Sprintf("Valve with id %q not found", id))
	}
	errs := data.Errors{}
	valve := &BuiltInValve{
		description:       m.moduleData.GetString(parentPath+".description", &errs),
		valveNumber:       m.moduleData.GetInt64(parentPath+".valveNumber", &errs),
		workingStatus:     m.moduleData.GetBool(parentPath+".workingStatus", &errs),
		openingPercentage: m.moduleData.GetInt64(parentPath+".openingPercentage", &errs),
		currentTemp:       m.moduleData.GetInt64(parentPath+".currentTemp", &errs),
		returnTemp:        m.moduleData.GetInt64(parentPath+".returnTemp", &errs),
		setTemp:           m.moduleData.GetInt64(parentPath+".setTemp", &errs),
	}
	if len(errs) > 0 {
		return valve, errs
	}
	return valve, nil
}

func (m *emodul) ParseTempSensor(id uint) (*TemperatureSensor, error) {
	var sensor TemperatureSensor
	var errors data.Errors

	// get sensor data
	parentPath := fmt.Sprintf("$.tiles[?(@.id == %v)]", id)
	tileData := m.moduleData.GetObject(parentPath)
	if tileData == nil {
		return nil, fmt.Errorf("Tile not found! id: %v", id)
	}
	tile := data.Data{Value: tileData}
	sensor.currentTemp = tile.GetInt64("$.params.value", &errors)
	if errors.HasAny() {
		return nil, errors.FirstError()
	}
	menuId := tile.GetInt64("$.menuId", &errors)
	if errors.HasAny() {
		return nil, errors.FirstError()
	}

	// get menu data - sensor settings
	parentPath = fmt.Sprintf("$.elements[?(@.id == %v)]", *menuId)
	menuData := m.menuData.GetObject(parentPath)
	if menuData == nil {
		return nil, fmt.Errorf("Menu not found! id: %v", id)
	}
	menu := data.Data{Value: menuData}
	// parse target, min, max
	sensor.targetTemp = menu.GetInt64("$.params.value", &errors)
	sensor.min = menu.GetInt64("$.params.min", &errors)
	sensor.max = menu.GetInt64("$.params.max", &errors)
	if errors.HasAny() {
		return nil, errors.FirstError()
	}
	return &sensor, nil
}

type Sensor interface {
	PublishConfig(context.Context) error
	PublishData(context.Context, float32) error
}

// MQTT Temperature Sensor
type mqttTempSensor struct {
	lo     logf.Logger
	mq     mq.MqttClient
	uid    string
	config SensorMqttConfig
}

func NewMqttTemperatureSensor(log logf.Logger, mq mq.MqttClient, device SensorMqttConfigDevice, uid string, name string, topic string) Sensor {
	return &mqttTempSensor{
		lo:  log,
		uid: uid,
		mq:  mq,
		config: SensorMqttConfig{
			Device:            device,
			DeviceClass:       "temperature",
			UniqueId:          "hassext_" + uid,
			UnitOfMeasurement: "°C",
			ValueTemplate:     "{{ value_json.temperature }}",
			EnabledByDefault:  true,
			Name:              name,
			StateClass:        "measurement",
			StateTopic:        topic,
		},
	}
}

func (s *mqttTempSensor) PublishConfig(ctx context.Context) error {
	bytes, err := json.Marshal(s.config)
	if err != nil {
		return fmt.Errorf("mqtt config serialize error %w", err)
	}
	s.lo.Info("Publish", "config", string(bytes))
	err = s.mq.Publish(ctx, 10*time.Second, "homeassistant/sensor/"+s.uid+"/"+s.config.DeviceClass+"/config", bytes)
	if err != nil {
		return fmt.Errorf("Valve config mqtt publish error %w", err)
	}
	return nil
}

func (s *mqttTempSensor) PublishData(ctx context.Context, value float32) error {
	data := struct {
		Temperature float32 `json:"temperature"`
	}{
		Temperature: value,
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("Sensor data serialize error %w", err)
	}
	s.lo.Info("Publish", "topic", s.config.StateTopic, "data", string(bytes), "value", value)
	err = s.mq.Publish(ctx, 10*time.Second, s.config.StateTopic, bytes)
	if err != nil {
		return fmt.Errorf("sensor publish error %w", err)
	}
	return nil
}
