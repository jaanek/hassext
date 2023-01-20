package hass

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

type Errors []error

func (list Errors) Error() string {
	errs := make([]string, 0, len(list))
	for _, e := range list {
		errs = append(errs, e.Error())
	}
	return strings.Join(errs, ",")
}

func (m *emodul) ParseValve(id uint) (*BuiltInValve, error) {
	// check if we have a valve with specified id
	parentPath := fmt.Sprintf("$.tiles[?(@.id == %v)].params", id)
	obj := m.GetObject(parentPath)
	if obj == nil {
		return nil, fmt.Errorf(fmt.Sprintf("Valve with id %q not found", id))
	}
	errs := Errors{}
	valve := &BuiltInValve{
		description:       m.GetString(parentPath+".description", &errs),
		valveNumber:       m.GetInt64(parentPath+".valveNumber", &errs),
		workingStatus:     m.GetBool(parentPath+".workingStatus", &errs),
		openingPercentage: m.GetInt64(parentPath+".openingPercentage", &errs),
		currentTemp:       m.GetInt64(parentPath+".currentTemp", &errs),
		returnTemp:        m.GetInt64(parentPath+".returnTemp", &errs),
		setTemp:           m.GetInt64(parentPath+".setTemp", &errs),
	}
	if len(errs) > 0 {
		return valve, errs
	}
	return valve, nil
}

type Sensor interface {
	PublishConfig(context.Context) error
	PublishData(context.Context, float32) error
}

// MQTT Temperature Sensor
type mqttTempSensor struct {
	lo     logf.Logger
	mq     MqttClient
	uid    string
	config SensorMqttConfig
}

func NewMqttTemperatureSensor(log logf.Logger, mq MqttClient, device SensorMqttConfigDevice, uid string, name string, topic string) Sensor {
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
	s.lo.Info("Publish", "data", string(bytes), "value", value)
	err = s.mq.Publish(ctx, 10*time.Second, s.config.StateTopic, bytes)
	if err != nil {
		return fmt.Errorf("sensor publish error %w", err)
	}
	return nil
}
