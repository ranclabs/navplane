package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MigrateUp runs all pending migrations.
func (db *DB) MigrateUp(migrationsPath string) error {
	m, err := db.newMigrate(migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrate(m)

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// MigrateDown rolls back the last migration.
func (db *DB) MigrateDown(migrationsPath string) error {
	m, err := db.newMigrate(migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrate(m)

	if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	return nil
}

// MigrateVersion returns the current migration version.
func (db *DB) MigrateVersion(migrationsPath string) (uint, bool, error) {
	m, err := db.newMigrate(migrationsPath)
	if err != nil {
		return 0, false, err
	}
	defer closeMigrate(m)

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}

// MigrateReset rolls back all migrations (use with caution).
func (db *DB) MigrateReset(migrationsPath string) error {
	m, err := db.newMigrate(migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrate(m)

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to reset migrations: %w", err)
	}

	return nil
}

func closeMigrate(m *migrate.Migrate) {
	_, _ = m.Close()
}

func (db *DB) newMigrate(migrationsPath string) (*migrate.Migrate, error) {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	sourceURL := "file://" + migrationsPath
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}
