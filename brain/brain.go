package brain

import (
	"context"
	"fmt"
	"time"

	"github.com/jaanek/hassext/homeassistant"
	"github.com/jaanek/hassext/nordpool"
	"github.com/zerodha/logf"
)

type Brain interface {
	Run(context.Context)
}

type brain struct {
	lo              logf.Logger
	ha              homeassistant.HomeAssistant
	errors          chan error
	nordpoolPrices  nordpool.NordpoolPrices
	dishwasherStart time.Time
	kyteStart       time.Time
	heatingAllowed  bool
}

func NewBrain(lo logf.Logger, ha homeassistant.HomeAssistant) Brain {
	return &brain{
		lo:     lo,
		ha:     ha,
		errors: make(chan error, 10),
	}
}

func (b *brain) Run(ctx context.Context) {
	// log errors if they happen
	go func() {
		for {
			select {
			case err := <-b.errors:
				b.lo.Error("brain", "error", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// start fetching data
	ticker := time.NewTicker(10 * time.Second)
	for {
		// fetch data
		err := b.ha.FetchData()
		if err != nil {
			werr := fmt.Errorf("fetch error %w", err)
			b.errors <- werr
			b.lo.Error("brain", "error", werr)
		} else {
			b.onDataUpdate()
		}

		// wait next tick
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (b *brain) onDataUpdate() {
	state := b.ha.GetStateData()

	// get nordpool prices
	prices, err := nordpool.ParseNordpoolPrices(state)
	if err != nil {
		b.errors <- err
	} else {
		b.nordpoolPrices = *prices
		b.nordpoolPrices.Updated = time.Now()
		b.lo.Info("Nordpool", "prices", b.nordpoolPrices)
	}

	// timer.test . States: idle (when pressed finish or cancel), active (when pressed start), paused (when pressed cancel)
	entity, err := ParseEntityState("timer.test", state)
	if err != nil {
		b.errors <- err
	} else {
		b.lo.Info("Timer test", "state", entity)
	}

	// emodul controller state
	entity, err = ParseEntityState("sensor.controller_state", state)
	if err != nil {
		b.errors <- err
	} else {
		b.lo.Info("Emodul", "controller state", entity)
	}
	// emodul operation mode
	entity, err = ParseEntityState("sensor.operation_modes", state)
	if err != nil {
		b.errors <- err
	} else {
		b.lo.Info("Emodul", "operation mode", entity)
	}
	// external temperature
	entity, err = ParseEntityState("sensor.external_temperature", state)
	if err != nil {
		b.errors <- err
	} else {
		b.lo.Info("Emodul", "external temperature", entity)
	}
	// komfovent operation mode. Modes: Normal, Intensive, Away, Boost
	entity, err = ParseEntityState("sensor.komfovent_operation_mode", state)
	if err != nil {
		b.errors <- err
	} else {
		b.lo.Info("Komfovent", "operation mode", entity)
	}

	// update states
	err = b.updateData()
	if err != nil {
		b.errors <- err
	}
}

func (b *brain) updateData() error {
	// todayPrices := append([]float64(nil), b.nordpoolPrices.Today...)
	tomorrowPrices := append([]float64(nil), b.nordpoolPrices.Tomorrow...)

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

	// set start time for diswasher for tomorrow night
	if len(tomorrowPrices) > 0 {
		// we concat today late night + tomorrow night till morning hours to get cheapest from those
		// hours := append([]float64(nil), todayPrices[23:]...) // from 23:00 -> 23:00
		hours := append([]float64(nil), tomorrowPrices[0:7]...) // from 00:00 -> 06:00

		// get the cheapestAll 4 sequential hours from provided list. 4 hours is enough for dishwasher to finish in eco mode
		cheapestAll := nordpool.FindCheapestElectricityHours(hours, 4)
		if len(cheapestAll) > 0 {
			cheapest := cheapestAll[0]
			dishwasherStart := time.Date(n.Year(), n.Month(), n.Day(), cheapest.StartIndex, 0, 0, 0, n.Location())
			if b.dishwasherStart != dishwasherStart {
				err := b.ha.SetInputDateTime("input_datetime.dishwasher_start", dishwasherStart, homeassistant.INPUT_TIME)
				if err != nil {
					return err
				}
				b.dishwasherStart = dishwasherStart
			}
		}
	}

	// set the heating allowed
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
