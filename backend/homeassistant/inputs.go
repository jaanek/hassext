package homeassistant

import (
	"fmt"
	"time"
)

type Inputs interface {
	SetInputDateTime(string, time.Time, DateTimeOption) error
	SetInputBoolean(string, BooleanAction) error
	SetInputButton(string, ButtonAction) error
	SetInputTextValue(string, string) error
	SetInputNumberValue(string, int64) error
	SetInputNumber(string, NumberAction) error
}

type DateTimeOption string

const (
	INPUT_DATE     DateTimeOption = "date"
	INPUT_TIME     DateTimeOption = "time"
	INPUT_DATETIME DateTimeOption = "datetime"
)

func (m *homeassistant) SetInputDateTime(entityId string, time time.Time, format DateTimeOption) error {
	var req = struct {
		EntityId string  `json:"entity_id"`
		Date     *string `json:"date,omitempty"`     // 2019-04-20
		Time     *string `json:"time,omitempty"`     // 05:04:20
		DateTime *string `json:"datetime,omitempty"` // 2019-04-20 05:04:20
		// Timestamp *uint64 `json:"timestamp"` // 9223372036854776000
	}{
		EntityId: entityId,
	}

	switch format {
	case INPUT_DATE:
		t := time.Format("2006-01-02")
		req.Date = &t
	case INPUT_TIME:
		t := time.Format("15:04:05")
		req.Time = &t
	case INPUT_DATETIME:
		t := time.Format("2006-01-02 15:04:05")
		req.DateTime = &t
	default:
		return fmt.Errorf("Unknown datetime format option: %v", format)
	}

	err := m.callService("input_datetime", "set_datetime", req)
	if err != nil {
		return err
	}
	return nil
}

type BooleanAction string

const (
	BOOLEAN_TOGGLE   BooleanAction = "toggle"
	BOOLEAN_TURN_ON  BooleanAction = "turn_on"
	BOOLEAN_TURN_OFF BooleanAction = "turn_off"
)

func (m *homeassistant) SetInputBoolean(entityId string, action BooleanAction) error {
	var req = struct {
		EntityId string `json:"entity_id"`
	}{
		EntityId: entityId,
	}

	err := m.callService("input_boolean", string(action), req)
	if err != nil {
		return err
	}
	return nil
}

type ButtonAction string

const (
	ButtonPress ButtonAction = "press"
)

func (m *homeassistant) SetInputButton(entityId string, action ButtonAction) error {
	var req = struct {
		EntityId string `json:"entity_id"`
	}{
		EntityId: entityId,
	}

	err := m.callService("input_button", string(action), req)
	if err != nil {
		return err
	}
	return nil
}

func (m *homeassistant) SetInputTextValue(entityId string, value string) error {
	var req = struct {
		EntityId string `json:"entity_id"`
		Value    string `json:"value"`
	}{
		EntityId: entityId,
		Value:    value,
	}

	err := m.callService("input_text", "set_value", req)
	if err != nil {
		return err
	}
	return nil
}

func (m *homeassistant) SetInputNumberValue(entityId string, value int64) error {
	var req = struct {
		EntityId string `json:"entity_id"`
		Value    int64  `json:"value"`
	}{
		EntityId: entityId,
		Value:    value,
	}

	err := m.callService("input_number", "set_value", req)
	if err != nil {
		return err
	}
	return nil
}

type NumberAction string

const (
	NumberIncrement NumberAction = "increment"
	NumberDecrement NumberAction = "decrement"
)

func (m *homeassistant) SetInputNumber(entityId string, action NumberAction) error {
	var req = struct {
		EntityId string `json:"entity_id"`
	}{
		EntityId: entityId,
	}

	err := m.callService("input_number", string(action), req)
	if err != nil {
		return err
	}
	return nil
}
