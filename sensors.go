package hass

import (
	"fmt"
	"strings"
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
	errs := make([]string, 0)
	for _, e := range list {
		errs = append(errs, e.Error())
	}
	return strings.Join(errs, ",")
}

func (m *emodul) ParseValve(id uint) (*BuiltInValve, error) {
	// check if we have a valve with id
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
