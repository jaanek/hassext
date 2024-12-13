package brain

import (
	"context"
	"fmt"
	"time"

	"github.com/jaanek/hassext/floorheating"
	"github.com/jaanek/hassext/homeassistant"
	"github.com/jaanek/hassext/mq"
	"github.com/jaanek/hassext/nordpool"
	"github.com/jaanek/hassext/sqldb"
	"github.com/jaanek/hassext/sqlite"
	"github.com/jaanek/hassext/uponor"
	"github.com/zerodha/logf"
)

type Brain interface {
	Run(context.Context)
}

type brain struct {
	lo                                   logf.Logger
	ha                                   homeassistant.HomeAssistant
	mq                                   mq.MqttClient
	uponorClient                         uponor.UponorClient
	sqliteDir                            string
	errors                               chan error
	flootHeating                         floorheating.FloorHeating
	nordpoolPrices                       nordpool.NordpoolPrices
	dishwasherStart                      time.Time
	heapPumpHeatingAllowedWinterMode     EntityState
	heapPumpKeepWaterHeatingActive       EntityState // example: "on" / "off"
	heapPumpKeepWaterHeaterBoost         EntityState // example: "on" / "off"
	heapPumpTriggerHeatingActive         EntityState // example: "on" / "off"
	heapPumpTriggerWaterHeaterActive     EntityState // example: "on" / "off"
	heapPumpTriggerWaterHeaterBoost      EntityState // example: "on" / "off"
	heapPumpTriggerHeatingSetTemp        EntityState
	heapPumpHeatingIgnoreMaxPricePerHour EntityState // example: "on" / "off"
	heatPumpHeating                      ClimateState
	heatPumpWaterHeater                  ClimateState
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
	uponorElutuba                        ClimateState
	uponorEsik                           ClimateState
	uponorDussiruum                      ClimateState
	uponorSaunaEesruum                   ClimateState
	floorHeatingValvesManualOperation    homeassistant.SwitchState
	floorHeatingKlappManualOperation     homeassistant.SwitchState
	floorHeatingKlappStates              map[floorheating.FloorHeatingKlapp]*homeassistant.ThermostatState
	floorHeatingValves                   map[floorheating.FloorHeatingValveEntityId]*homeassistant.SwitchState
	floorHeatingSimpleThermostats        map[floorheating.ThermostatName]*homeassistant.ThermostatState
	floorHeatingSimpleThermostatTargets  map[floorheating.ThermostatTargetName]*homeassistant.ThermostatState
}

func NewBrain(lo logf.Logger, ha homeassistant.HomeAssistant, mq mq.MqttClient, uponorClient uponor.UponorClient, dataDir string, floorHeating floorheating.FloorHeating) Brain {
	return &brain{
		lo:                                  lo,
		ha:                                  ha,
		mq:                                  mq,
		uponorClient:                        uponorClient,
		sqliteDir:                           dataDir,
		flootHeating:                        floorHeating,
		errors:                              make(chan error, 10),
		floorHeatingKlappStates:             map[floorheating.FloorHeatingKlapp]*homeassistant.ThermostatState{},
		floorHeatingValves:                  map[floorheating.FloorHeatingValveEntityId]*homeassistant.SwitchState{},
		floorHeatingSimpleThermostats:       map[floorheating.ThermostatName]*homeassistant.ThermostatState{},
		floorHeatingSimpleThermostatTargets: map[floorheating.ThermostatTargetName]*homeassistant.ThermostatState{},
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
		var err = b.fetchData()
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

func (b *brain) fetchData() error {
	err := b.ha.FetchData()
	if err != nil {
		return err
	}
	errU := b.uponorClient.FetchData()
	if errU != nil {
		b.lo.Error("Error while fetching data from uponor controller!", "error", errU)
	}
	return err
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
	b.FloorHeating(state)

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

	// floorheating
	err = b.UpdateFloorheatingValves(b.floorHeatingValves)
	if err != nil {
		b.errors <- err
	}
	err = b.UpdateFloorheatingKontuurTemp(b.floorHeatingKlappStates)
	if err != nil {
		b.errors <- err
	}
	return nil
}

func (b *brain) openAppDatabase() (*sqldb.DB, error) {
	return sqlite.NewDB(b.lo, b.sqliteDir, sqlite.DBDefault, false)
}
