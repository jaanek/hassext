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
	lo                                   logf.Logger
	ha                                   homeassistant.HomeAssistant
	errors                               chan error
	nordpoolPrices                       nordpool.NordpoolPrices
	dishwasherStart                      time.Time
	heapPumpKeepWaterHeatingActive       EntityState // example: "on" / "off"
	heapPumpKeepWaterHeaterBoost         EntityState // example: "on" / "off"
	heapPumpTriggerHeatingActive         EntityState // example: "on" / "off"
	heapPumpTriggerWaterHeaterActive     EntityState // example: "on" / "off"
	heapPumpTriggerWaterHeaterBoost      EntityState // example: "on" / "off"
	heapPumpTriggerHeatingSetTemp        EntityState
	heapPumpHeatingIgnoreMaxPricePerHour EntityState // example: "on" / "off"
	heatPumpHeating                      ThermostatState
	heatPumpWaterHeater                  ThermostatState
	heatPumpWaterHeaterStartState        EntityState // example: "03:00:00"
	heatPumpWaterHeaterStopState         EntityState // example: "06:00:00"
	heatPumpWaterHeaterStart             time.Time
	heatPumpWaterHeaterStop              time.Time
	heatPumpHeatingTempShift             int32
	emodulHeatingAllowed                 bool
	emodulControllerState                EntityState
	emodulOperationMode                  EntityState
	emodulOutsideTemp                    float64
	emodulBoilerTemp                     float64
	uponorElutuba                        ThermostatState // example: "off" / "heat"
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

	// timer.test . States: idle (when pressed finish or cancel), active (when pressed start), paused (when pressed cancel)
	entity, err := ParseEntityState("timer.test", state)
	if err != nil {
		b.errors <- err
	} else {
		b.lo.Info("Timer test", "state", entity)
	}

	b.Uponor(state)
	b.Nordpool(state)
	b.Emodule(state)
	b.HeatPump(state)
	b.Komfovent(state)

	// update states
	err = b.updateData()
	if err != nil {
		b.errors <- err
	}
}

func (b *brain) updateData() error {
	todayPrices := append([]float64(nil), b.nordpoolPrices.Today...)
	tomorrowPrices := append([]float64(nil), b.nordpoolPrices.Tomorrow...)
	if len(todayPrices) == 0 {
		b.lo.Warn("HeatPump", "No nordpool today prices. Todays prices is 0")
		return nil
	}

	// heating
	err := b.SetHeatPumpWaterTankStartStopTime(TODAY_PM_9, TOMORROW_AM_7, todayPrices, tomorrowPrices)
	if err != nil {
		b.errors <- err
	}
	err = b.SetHeatPumpHeating(todayPrices, MAX_PRICE_PER_HOUR)
	if err != nil {
		b.errors <- err
	}
	err = b.SetHeatPumpTemperature(todayPrices)
	if err != nil {
		b.errors <- err
	}
	// err = b.SetHeatPumpAutomationsOnOff()
	// if err != nil {
	// 	b.errors <- err
	// }

	// dishwasher
	err = b.SetDishwasherStartTime(tomorrowPrices)
	if err != nil {
		b.errors <- err
	}
	return nil
}
