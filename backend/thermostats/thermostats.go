package thermostats

import (
	"context"
	"sync"
	"time"

	"github.com/jaanek/hassext/sqldb"
	"github.com/jaanek/hassext/sqlite"
	"github.com/zerodha/logf"
)

type ThermostatName string
type FloorHeatingValveName string

const (
	// target temperatures for uponor wall thermostats
	THERMOSTAT_ELUTUBA_TARGET       ThermostatName = "floor1_elutuba_target"
	THERMOSTAT_ESIK_TARGET          ThermostatName = "floor1_esik_target"
	THERMOSTAT_DUSSIRUUM_TARGET     ThermostatName = "floor1_dussiruum_target"
	THERMOSTAT_SAUNA_EESRUUM_TARGET ThermostatName = "floor1_sauna_eesruum_target"
	// thermostat names
	THERMOSTAT_ELUTUBA_WALL  ThermostatName = "floor1_elutuba"
	THERMOSTAT_ELUTUBA_SOFA  ThermostatName = "floor1_elutuba_sofa"
	THERMOSTAT_ESIK          ThermostatName = "floor1_esik"
	THERMOSTAT_DUSSIRUUM     ThermostatName = "floor1_dussiruum"
	THERMOSTAT_SAUNA_EESRUUM ThermostatName = "floor1_sauna_eesruum"
	// floor heating valves
	FLOOR_HEATING_VALVE_ELUTUBA_1     FloorHeatingValveName = "floor1_elutuba1"
	FLOOR_HEATING_VALVE_ELUTUBA_2     FloorHeatingValveName = "floor1_elutuba2"
	FLOOR_HEATING_VALVE_ESIK          FloorHeatingValveName = "floor1_esik"
	FLOOR_HEATING_VALVE_KITCHEN       FloorHeatingValveName = "floor1_kook"
	FLOOR_HEATING_VALVE_DUSSIRUUM     FloorHeatingValveName = "floor1_dussiruum"
	FLOOR_HEATING_VALVE_SAUNA_EESRUUM FloorHeatingValveName = "floor1_sauna_eesruum"
	FLOOR_HEATING_VALVE_SUUR_KORIDOR1 FloorHeatingValveName = "floor1_suur_koridor1"
	FLOOR_HEATING_VALVE_SUUR_KORIDOR2 FloorHeatingValveName = "floor1_suur_koridor2"
)

type Thermostats interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type thermostats struct {
	prefix    string
	log       logf.Logger
	done      chan struct{}
	doneOnce  sync.Once
	sqliteDir string
}

func New(log logf.Logger, sqliteDir string) Thermostats {
	return &thermostats{
		prefix:    "[thermostats] ",
		log:       log,
		done:      make(chan struct{}),
		sqliteDir: sqliteDir,
	}
}

func (t *thermostats) Start(ctx context.Context) (err error) {
	t.log.Info(t.prefix + "Starting Thermostats")
	return nil
}

func (t *thermostats) Stop(ctx context.Context) error {
	t.log.Info(t.prefix + "Stopping Thermostats")
	t.doneOnce.Do(func() {
		close(t.done)
	})
	return nil
}

func (t *thermostats) openAppDatabase() (*sqldb.DB, error) {
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
