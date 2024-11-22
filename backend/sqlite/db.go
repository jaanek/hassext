package sqlite

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jaanek/hassext/sqldb"

	// sqlite3 access
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/zerodha/logf"
)

const DBDefault = "default"

//go:embed migrations/**/*.sql
var migrationsFs embed.FS

// NewDB. dbDir like: /opt/data, dbName: default
func NewDB(log logf.Logger, dbDir, dbName string, initDb bool) (*sqldb.DB, error) {
	var dbPath = LibSqlDB(dbDir, "default")
	var parentDir = filepath.Dir(dbPath)
	var err = ensureDirExists(log, parentDir)
	if err != nil {
		return nil, fmt.Errorf("DB dir does not exit: %s, error: %w", dbDir, err)
	}
	if !initDb {
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("DB file does not exit: %s, error: %w", dbPath, err)
		}
	}
	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening db file: %s, error: %w", dbPath, err)
	}
	if _, err := db.Exec(`
-- Apply Litestream recommendations: https://litestream.io/tips/
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA journal_mode = WAL;
PRAGMA wal_autocheckpoint = 0;
-- To support foreign keys in sqlite3 we need to switch them on explicitly
PRAGMA foreign_keys = ON;
		`); err != nil {
		return nil, fmt.Errorf("failed to set sqlite3 database pragmas: %w", err)
	}
	// initiate database, create tables etc.
	if initDb {
		err = applyMigrations(log, db, migrationsFs, "migrations/"+dbName)
		if err != nil {
			return nil, err
		}
	}
	// log.Info("[sqlite] Opened database", "name", dbName)
	return &sqldb.DB{DB: db}, nil
}

func ensureDirExists(log logf.Logger, dir string) error {
	if dir == "" {
		return errors.New("databases dir cannot be empty")
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Info("[sqlite] Creating database data dir", "dir", dir)
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func LibSqlDB(dbDir, dbName string) string {
	return filepath.Join(dbDir, "dbs", dbName, "data")
}
