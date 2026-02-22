package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"navplane/internal/config"
)

// DB wraps sql.DB to provide application-specific database operations.
type DB struct {
	*sql.DB
}

// New creates a new database connection pool with the provided configuration.
func New(cfg config.DatabaseConfig) (*DB, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	return &DB{DB: db}, nil
}

// Connect establishes the database connection and verifies connectivity.
func Connect(cfg config.DatabaseConfig) (*DB, error) {
	db, err := New(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Close closes the database connection pool.
func (db *DB) Close() error {
	return db.DB.Close()
}

// Health checks if the database is reachable.
func (db *DB) Health(ctx context.Context) error {
	return db.PingContext(ctx)
}
