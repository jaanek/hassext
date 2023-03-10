package brain

import (
	"time"

	"github.com/jaanek/hassext/homeassistant"
)

func (b *brain) SetHeatingStartTime() error {
	// set start time for soojuspumpt kyte
	// t1 := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, t.Nanosecond(), t.Location())
	n := time.Now()
	kyteStart := time.Date(n.Year(), n.Month(), n.Day(), 3, 0, 0, 0, n.Location())
	if b.kyteStart != kyteStart {
		err := b.ha.SetInputDateTime("input_datetime.soojuspump_kyte_start", kyteStart, homeassistant.INPUT_TIME)
		if err != nil {
			return err
		}
		b.kyteStart = kyteStart
	}
	return nil
}

// set the heating allowed
func (b *brain) SetHeatingAllowed() error {
	var heatingAllowed bool = true
	if b.heatingAllowed != heatingAllowed {
		err := b.ha.SetInputBoolean("input_boolean.katel_heating_allowed", homeassistant.BOOLEAN_TURN_ON)
		if err != nil {
			return err
		}
		b.heatingAllowed = heatingAllowed
	}
	return nil
}
