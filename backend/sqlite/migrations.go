package sqlite

// Copied & Modified from: https://github.com/mtlynch/logpaste/blob/master/store/sqlite/migrations.go

import (
	"context"
	"embed"
	"fmt"
	"path"
	"sort"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

type dbMigration struct {
	version int
	query   string
}

func applyMigrations(log logf.Logger, db *sqlx.DB, migrationsFs embed.FS, migrationsDir string) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("failed to get user_version: %w", err)
	}

	migrations, err := loadMigrations(log, migrationsFs, migrationsDir)
	if err != nil {
		return fmt.Errorf("error loading database migrations: %w", err)
	}

	log.Info("[sqlite] Loaded migrations", "current db version", version, "migrations count", len(migrations))

	for _, migration := range migrations {
		if migration.version <= version {
			continue
		}
		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("failed to create migration transaction %d: %w", migration.version, err)
		}

		_, err = tx.Exec(migration.query)
		if err != nil {
			return fmt.Errorf("failed to perform DB migration %d: %w", migration.version, err)
		}

		_, err = tx.Exec(fmt.Sprintf(`pragma user_version=%d`, migration.version))
		if err != nil {
			return fmt.Errorf("failed to update DB version to %d: %w", migration.version, err)
		}

		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.version, err)
		}
		log.Info("[sqlite] Migration commited", "version", migration.version, "migrations count", len(migrations))
	}
	return nil
}

func loadMigrations(log logf.Logger, migrationsFs embed.FS, migrationsDir string) ([]dbMigration, error) {
	migrations := []dbMigration{}

	entries, err := migrationsFs.ReadDir(migrationsDir)
	if err != nil {
		return []dbMigration{}, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		version, err := migrationVersionFromFilename(entry.Name())
		if err != nil {
			return []dbMigration{}, err
		}

		log.Info("[sqlite] Reading migrations file", "file", path.Join(migrationsDir, entry.Name()))
		query, err := migrationsFs.ReadFile(path.Join(migrationsDir, entry.Name()))
		if err != nil {
			return []dbMigration{}, err
		}

		migrations = append(migrations, dbMigration{version, string(query)})
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return migrations, nil
}

func migrationVersionFromFilename(filename string) (int, error) {
	version, err := strconv.ParseInt(filename[:3], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid migration number in filename: %v", filename)
	}

	return int(version), nil
}
