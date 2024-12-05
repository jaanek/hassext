package floorheating

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jaanek/hassext/homeassistant"
	"github.com/jaanek/hassext/model"
	"github.com/jaanek/hassext/mq"
	"github.com/jaanek/hassext/sqldb"
	"github.com/jaanek/hassext/sqlite"
	"github.com/zerodha/logf"
)

type FloorHeatingValveEntityId string
type FloorHeatingValveName string

const (
	// floor heating valves
	FLOOR_HEATING_VALVE_ESIK          FloorHeatingValveEntityId = "tasmota"   // floor1_esik
	FLOOR_HEATING_VALVE_ELUTUBA_1     FloorHeatingValveEntityId = "tasmota_2" // floor1_elutuba1
	FLOOR_HEATING_VALVE_ELUTUBA_2     FloorHeatingValveEntityId = "tasmota_3" // floor1_elutuba2
	FLOOR_HEATING_VALVE_KITCHEN       FloorHeatingValveEntityId = "tasmota_4" // floor1_kook
	FLOOR_HEATING_VALVE_DUSSIRUUM     FloorHeatingValveEntityId = "tasmota_5" // floor1_dussiruum
	FLOOR_HEATING_VALVE_SAUNA_EESRUUM FloorHeatingValveEntityId = "tasmota_6" // floor1_sauna_eesruum
	FLOOR_HEATING_VALVE_SUUR_KORIDOR2 FloorHeatingValveEntityId = "tasmota_7" // floor1_suur_koridor2
	FLOOR_HEATING_VALVE_SUUR_KORIDOR1 FloorHeatingValveEntityId = "tasmota_8" // floor1_suur_koridor1
	// floor heating valve names
	FLOOR_HEATING_VALVE_NAME_ESIK          FloorHeatingValveName = "floor1_esik"
	FLOOR_HEATING_VALVE_NAME_ELUTUBA_1     FloorHeatingValveName = "floor1_elutuba1"
	FLOOR_HEATING_VALVE_NAME_ELUTUBA_2     FloorHeatingValveName = "floor1_elutuba2"
	FLOOR_HEATING_VALVE_NAME_KITCHEN       FloorHeatingValveName = "floor1_kook"
	FLOOR_HEATING_VALVE_NAME_DUSSIRUUM     FloorHeatingValveName = "floor1_dussiruum"
	FLOOR_HEATING_VALVE_NAME_SAUNA_EESRUUM FloorHeatingValveName = "floor1_sauna_eesruum"
	FLOOR_HEATING_VALVE_NAME_SUUR_KORIDOR2 FloorHeatingValveName = "floor1_suur_koridor2"
	FLOOR_HEATING_VALVE_NAME_SUUR_KORIDOR1 FloorHeatingValveName = "floor1_suur_koridor1"
)

type FloorHeatingManualOperationState string

const (
	FLOOR_HEATING_MANUAL_OPERATION_SWITCH FloorHeatingManualOperationState = "floorheating_valves_manual_operation"
	FloorHeatingManualOperationOff                                         = "off"
	FloorHeatingManualOperationOn                                          = "on"
)

var ValveEntityIds = []FloorHeatingValveEntityId{
	FLOOR_HEATING_VALVE_ELUTUBA_1,
	FLOOR_HEATING_VALVE_ELUTUBA_2,
	FLOOR_HEATING_VALVE_ESIK,
	FLOOR_HEATING_VALVE_KITCHEN,
	FLOOR_HEATING_VALVE_DUSSIRUUM,
	FLOOR_HEATING_VALVE_SAUNA_EESRUUM,
	FLOOR_HEATING_VALVE_SUUR_KORIDOR1,
	FLOOR_HEATING_VALVE_SUUR_KORIDOR2,
}

type FloorheatingValve interface {
	EntityId() FloorHeatingValveEntityId
	PowerId() string
	Name() FloorHeatingValveName
}
type floorheatingValve struct {
	stateId FloorHeatingValveEntityId
	powerId string
	name    FloorHeatingValveName
}

func (v floorheatingValve) EntityId() FloorHeatingValveEntityId {
	return v.stateId
}
func (v floorheatingValve) PowerId() string {
	return v.powerId
}
func (v floorheatingValve) Name() FloorHeatingValveName {
	return v.name
}

func FloorheatingValveByName(entityId FloorHeatingValveEntityId) (FloorheatingValve, error) {
	switch entityId {
	case FLOOR_HEATING_VALVE_ESIK:
		return floorheatingValve{entityId, "", FLOOR_HEATING_VALVE_NAME_ESIK}, nil
	case FLOOR_HEATING_VALVE_ELUTUBA_1:
		return floorheatingValve{entityId, "2", FLOOR_HEATING_VALVE_NAME_ELUTUBA_1}, nil
	case FLOOR_HEATING_VALVE_ELUTUBA_2:
		return floorheatingValve{entityId, "3", FLOOR_HEATING_VALVE_NAME_ELUTUBA_2}, nil
	case FLOOR_HEATING_VALVE_KITCHEN:
		return floorheatingValve{entityId, "4", FLOOR_HEATING_VALVE_NAME_KITCHEN}, nil
	case FLOOR_HEATING_VALVE_DUSSIRUUM:
		return floorheatingValve{entityId, "5", FLOOR_HEATING_VALVE_NAME_DUSSIRUUM}, nil
	case FLOOR_HEATING_VALVE_SAUNA_EESRUUM:
		return floorheatingValve{entityId, "6", FLOOR_HEATING_VALVE_NAME_SAUNA_EESRUUM}, nil
	case FLOOR_HEATING_VALVE_SUUR_KORIDOR2:
		return floorheatingValve{entityId, "7", FLOOR_HEATING_VALVE_NAME_SUUR_KORIDOR2}, nil
	case FLOOR_HEATING_VALVE_SUUR_KORIDOR1:
		return floorheatingValve{entityId, "8", FLOOR_HEATING_VALVE_NAME_SUUR_KORIDOR1}, nil
	}
	return nil, fmt.Errorf("No floorheating valve found with stateId: %s", entityId)
}

type FloorHeatingValveStatus int

func (v FloorHeatingValveStatus) String() string {
	if v == FloorHeatingValveStatusOff {
		return "OFF"
	} else if v == FloorHeatingValveStatusOn {
		return "ON"
	}
	return ""
}
func (v FloorHeatingValveStatus) State() bool {
	if v == FloorHeatingValveStatusOn {
		return true
	}
	return false
}

const (
	FloorHeatingValveStatusOff FloorHeatingValveStatus = 0
	FloorHeatingValveStatusOn  FloorHeatingValveStatus = 1
)

type FloorHeating interface {
	CheckFloorHeatingValves(valveStates map[FloorHeatingValveEntityId]*homeassistant.SwitchState) error
}

type floorHeating struct {
	prefix    string
	log       logf.Logger
	done      chan struct{}
	doneOnce  sync.Once
	sqliteDir string
	topic     string
	mq        mq.MqttClient
}

func New(log logf.Logger, sqliteDir string, mq mq.MqttClient) FloorHeating {
	return &floorHeating{
		prefix:    "[floor-heating] ",
		log:       log,
		done:      make(chan struct{}),
		sqliteDir: sqliteDir,
		topic:     "heatingvalves1",
		mq:        mq,
	}
}

// func (t *floorHeating) Start(ctx context.Context) (err error) {
// 	t.log.Info(t.prefix + "Starting floor heating")
// 	// start polling loop or react to triggering event
// 	var syncInterval = time.Minute * 1
// 	ticker := time.NewTicker(syncInterval)
// 	var triggerCheckQueue = make(chan struct{})
// 	var tickerReset = make(chan struct{})
// 	defer ticker.Stop()
// 	go func() {
// 		triggerCheckQueue <- struct{}{}
// 	}()
// 	for {
// 		select {
// 		case <-triggerCheckQueue:
// 			t.log.Info(t.prefix + "Checking if floor heating needs updating ...")
// 			err := t.checkFloorHeating()
// 			if err != nil {
// 				t.log.Error(t.prefix+"error while checking floor heating", "error", err)
// 			}
// 			go func() {
// 				tickerReset <- struct{}{}
// 			}()
// 		case <-tickerReset:
// 			ticker.Stop()
// 			ticker = time.NewTicker(syncInterval)
// 		case <-ticker.C:
// 			go func() {
// 				triggerCheckQueue <- struct{}{}
// 			}()
// 		case <-t.done:
// 			t.log.Info(t.prefix + "Exiting floor heating.")
// 			return nil
// 		}
// 	}
// }

// func (t *floorHeating) Stop(ctx context.Context) error {
// 	t.log.Info(t.prefix + "Stopping floor heating")
// 	t.doneOnce.Do(func() {
// 		close(t.done)
// 	})
// 	return nil
// }

func (t *floorHeating) CheckFloorHeatingValves(valveStates map[FloorHeatingValveEntityId]*homeassistant.SwitchState) error {
	t.log.Info(t.prefix + "Checking thermostats ...")
	// open local sqlite db
	db, err := t.openAppDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// read in all thermostat values in 1'st floor
	var thermostats = []model.Thermostat{}
	if err := db.Select(&thermostats, "select * from thermostat"); err != nil {
		return err
	}
	var dbUpdates = []DBUpdateValve{}
	var turnOn = FloorHeatingValveStatusOn
	var turnOff = FloorHeatingValveStatusOff
	for _, ts := range thermostats {
		t.log.Info(t.prefix+"Thermostat reading", "name", ts.Name, "current temperature", ts.CurrentTemperature, "target temperature", ts.TargetTemperature, "last update", ts.LastUpdate)
		var resolvedValves = t.resolveValves(ThermostatName(ts.Name))
		var turnAction *FloorHeatingValveStatus
		if ts.CurrentTemperature > ts.TargetTemperature+resolvedValves.upGap {
			// t.log.Info(t.prefix+"Turning OFF floor heating", "trigger thermostat name", ts.Name)
			turnAction = &turnOff
		} else if ts.CurrentTemperature < ts.TargetTemperature-resolvedValves.downGap {
			// t.log.Info(t.prefix+"Turning ON floor heating", "trigger thermostat name", ts.Name)
			turnAction = &turnOn
		}
		if turnAction != nil && len(resolvedValves.list) > 0 {
			t.turnFloorHeatingValves(valveStates, resolvedValves, *turnAction, &dbUpdates)
		}
	}

	// save valve updates into the database
	var now = time.Now()
	for _, item := range dbUpdates {
		var valve = item.valve
		var status = item.value
		// save it to db
		err := FloorHeatingValveUpsert(db, valve.Name(), status.State(), now)
		if err != nil {
			t.log.Error("Error while updating valve status in db!", "valve name", valve.Name(), "error", err)
		}
	}
	return nil
}

func (t *floorHeating) turnFloorHeatingValves(valveStates map[FloorHeatingValveEntityId]*homeassistant.SwitchState, resolvedValves ResolvedValves, value FloorHeatingValveStatus, dbUpdates *[]DBUpdateValve) {
	// check if we already have turned off, if so then do nothing
	var notInSync = []DBUpdateValve{}
	var inSyncValveStates = []string{}
	var inSyncValveNames = []string{}
	for _, valve := range resolvedValves.list {
		var haState = valveStates[valve.EntityId()]
		var newState = value
		if haState == nil {
			t.log.Error(t.prefix+"Valve state not found from parsed states!", "valve stateId", valve.EntityId())
			notInSync = append(notInSync, DBUpdateValve{valve, newState})
			continue
		}
		// there are certain valves that needs to be turned on always on some times, check them here
		switch valve.Name() {
		case FLOOR_HEATING_VALVE_NAME_ESIK:
			{
				// needs to be turned on on winter time becasue esik gets cold otherwise
				newState = FloorHeatingValveStatusOn
			}
		}
		// check if homeassistant valve state is not in sync with new calculation of valve state based on thermostat value
		var stateNeedsUpdate = strings.ToUpper(string(haState.State)) != newState.String()
		if stateNeedsUpdate {
			notInSync = append(notInSync, DBUpdateValve{valve, newState})
		} else {
			inSyncValveStates = append(inSyncValveStates, newState.String())
			inSyncValveNames = append(inSyncValveNames, string(valve.Name()))
		}
	}
	var inSyncValveStatesStr = strings.Join(inSyncValveStates, ", ")
	var inSyncValveNamesStr = strings.Join(inSyncValveNames, ", ")
	if len(notInSync) <= 0 {
		// already in sync, do nothing
		t.log.Info(t.prefix+"All valves are already in sync. Do nothing!", "section", resolvedValves.section, "states", inSyncValveStatesStr, "valves", inSyncValveNamesStr)
		return
	}

	// turn on/off not synced valves
	for _, item := range notInSync {
		var valve = item.valve
		var value = item.value
		t.log.Info(t.prefix+"Turning "+value.String()+" floor heating", "valve", valve.PowerId(), "section", resolvedValves.section)
		err := t.MqPublishData(context.Background(), valve, value)
		if err != nil {
			t.log.Error(t.prefix+"Valve cannot be updated with new state!", "valve stateId", valve.EntityId(), "error", err)
			continue
		}
	}
	*dbUpdates = append(*dbUpdates, notInSync...)
}

type DBUpdateValve struct {
	valve FloorheatingValve
	value FloorHeatingValveStatus
}

type ResolvedValves struct {
	list    []FloorheatingValve
	upGap   float32
	downGap float32
	section ThermostatTargetName
}

func (t *floorHeating) resolveValves(name ThermostatName) ResolvedValves {
	var valves = []FloorheatingValve{}
	var upGap float32 = 0.3
	var downGap float32 = -0.1
	var section ThermostatTargetName
	switch name {
	case THERMOSTAT_SAUNA_EESRUUM:
		valves = t.appendValve(valves, FLOOR_HEATING_VALVE_SAUNA_EESRUUM)
		section = THERMOSTAT_SAUNA_EESRUUM_TARGET
	case THERMOSTAT_DUSSIRUUM:
		valves = t.appendValve(valves, FLOOR_HEATING_VALVE_DUSSIRUUM)
		section = THERMOSTAT_DUSSIRUUM_TARGET
	// case THERMOSTAT_ELUTUBA_WALL:
	// 	fallthrough
	case THERMOSTAT_ELUTUBA_WALL: // THERMOSTAT_ELUTUBA_SOFA
		valves = t.appendValve(valves, FLOOR_HEATING_VALVE_ELUTUBA_1)
		valves = t.appendValve(valves, FLOOR_HEATING_VALVE_ELUTUBA_2)
		section = THERMOSTAT_ELUTUBA_WALL_TARGET // THERMOSTAT_ELUTUBA_SOFA_TARGET
	case THERMOSTAT_ESIK:
		valves = t.appendValve(valves, FLOOR_HEATING_VALVE_ESIK)
		valves = t.appendValve(valves, FLOOR_HEATING_VALVE_KITCHEN)
		section = THERMOSTAT_ESIK_TARGET
	case THERMOSTAT_SUUR_KORIDOR_WORKPLACE:
		valves = t.appendValve(valves, FLOOR_HEATING_VALVE_SUUR_KORIDOR1)
		valves = t.appendValve(valves, FLOOR_HEATING_VALVE_SUUR_KORIDOR2)
		section = THERMOSTAT_SUUR_KORIDOR_WORKPLACE_TARGET
	}
	return ResolvedValves{valves, upGap, downGap, section}
}

func (t *floorHeating) appendValve(valves []FloorheatingValve, name FloorHeatingValveEntityId) []FloorheatingValve {
	var valve, err = FloorheatingValveByName(name)
	if err != nil {
		t.log.Error(t.prefix+"Floor heating valve with name: %s not found", name)
		return valves
	}
	valves = append(valves, valve)
	return valves
}

func (t *floorHeating) openAppDatabase() (*sqldb.DB, error) {
	return sqlite.NewDB(t.log, t.sqliteDir, sqlite.DBDefault, false)
}

func ThermostatUpsert(db *sqldb.DB, thermostatName ThermostatName, targetTemperature float64, currentTemperature float64, now time.Time) (err error) {
	_, err = db.Exec("INSERT INTO thermostat (tname, target_temperature, current_temperature, last_update) VALUES (?, ?, ?, ?) ON CONFLICT(tname) DO UPDATE SET target_temperature=excluded.target_temperature, current_temperature=excluded.current_temperature, last_update=excluded.last_update", thermostatName, targetTemperature, currentTemperature, now)
	return
}

func ThermostatTemperatureUpsert(db *sqldb.DB, thermostatName ThermostatName, currentTemperature float64, now time.Time) (err error) {
	_, err = db.Exec("INSERT INTO thermostat (tname, current_temperature, last_update) VALUES (?, ?, ?) ON CONFLICT(tname) DO UPDATE SET current_temperature=excluded.current_temperature, last_update=excluded.last_update", thermostatName, currentTemperature, now)
	return
}

func FloorHeatingValveUpsert(db *sqldb.DB, valveName FloorHeatingValveName, stateUP bool, lastStateChange time.Time) (err error) {
	var stateB int = 0
	if stateUP {
		stateB = 1
	}
	_, err = db.Exec("INSERT INTO floor_heating_valve (vname, vstate, last_state_change) VALUES (?, ?, ?) ON CONFLICT(vname) DO UPDATE SET vstate=excluded.vstate, last_state_change=excluded.last_state_change", valveName, stateB, lastStateChange)
	return
}

// https://tasmota.github.io/docs/MQTT/#command-flow
func (t *floorHeating) MqPublishData(ctx context.Context, valve FloorheatingValve, value FloorHeatingValveStatus) error {
	var topic = "cmnd/" + t.topic + "/Power" + valve.PowerId()
	t.log.Info("Publish", "topic", topic, "data", value.String(), "value", value)
	err := t.mq.Publish(ctx, 10*time.Second, topic, value.String())
	if err != nil {
		return fmt.Errorf("floorheating valve publish error %w", err)
	}
	return nil
}
