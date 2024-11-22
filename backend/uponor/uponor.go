package uponor

import (
	"context"
	"sync"

	"github.com/jaanek/hassext/sqldb"
	"github.com/jaanek/hassext/sqlite"
	"github.com/zerodha/logf"
)

// type EntityState string

// const (
// 	ENTITY_THERMOSTAT_ELUTUBA       EntityState = "climate.elutuba"
// 	ENTITY_THERMOSTAT_ESIK          EntityState = "climate.kook_esik"
// 	ENTITY_THERMOSTAT_DUSSIRUUM     EntityState = "climate.dussiruum"
// 	ENTITY_THERMOSTAT_SAUNA_EESRUUM EntityState = "climate.sauna_eesruum"
// )

type Uponor interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type uponor struct {
	prefix    string
	log       logf.Logger
	done      chan struct{}
	doneOnce  sync.Once
	sqliteDir string
}

func New(log logf.Logger, sqliteDir string) Uponor {
	return &uponor{
		prefix:    "[uponor] ",
		log:       log,
		done:      make(chan struct{}),
		sqliteDir: sqliteDir,
	}
}

func (t *uponor) Start(ctx context.Context) (err error) {
	t.log.Info(t.prefix + "Starting Uponor")
	return nil
}

func (t *uponor) Stop(ctx context.Context) error {
	t.log.Info(t.prefix + "Stopping Uponor")
	t.doneOnce.Do(func() {
		close(t.done)
	})
	return nil
}

func (t *uponor) openAppDatabase() (*sqldb.DB, error) {
	return sqlite.NewDB(t.log, t.sqliteDir, sqlite.DBDefault, false)
}
