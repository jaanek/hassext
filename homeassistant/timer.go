package homeassistant

import (
	"fmt"
	"regexp"
)

func (m *homeassistant) TimerStart(entityId string, duration string) error {
	// https://regex101.com/r/9Qa071/6
	// validate duration
	var value string
	var reNum = regexp.MustCompile("^[0-9]{1,2}$")
	var reTime = regexp.MustCompile("^(?:[01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$") // (?:...) means is not a capturing group
	if reNum.MatchString(duration) {
		value = duration
	} else if reTime.MatchString(duration) {
		value = duration
	} else {
		return fmt.Errorf("Invalid duration pattern! Allowed: '00:01:00 or 60'. Param: %v", duration)
	}

	var req = struct {
		EntityId string `json:"entity_id"`
		Duration string `json:"duration"`
	}{
		EntityId: entityId,
		Duration: value,
	}

	err := m.callService("timer", "start", req)
	if err != nil {
		return err
	}
	return nil
}

type TimerAction string

const (
	TimerStart  TimerAction = "start"
	TimerPause  TimerAction = "pause"
	TimerCancel TimerAction = "cancel"
	TimerFinish TimerAction = "finish"
)

func (m *homeassistant) Timer(entityId string, action TimerAction) error {
	var req = struct {
		EntityId string `json:"entity_id"`
	}{
		EntityId: entityId,
	}

	err := m.callService("timer", string(action), req)
	if err != nil {
		return err
	}
	return nil
}
