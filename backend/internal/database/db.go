package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	DefaultMaxOpenConns    = 25
	DefaultMaxIdleConns    = 5
	DefaultConnMaxLifetime = 5 * time.Minute
	DefaultConnMaxIdleTime = 1 * time.Minute
	DefaultPingTimeout     = 5 * time.Second
)

type DB struct {
	*sql.DB
}

// Open creates a new database connection pool and verifies connectivity.
func Open(databaseURL string) (*DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(DefaultMaxOpenConns)
	db.SetMaxIdleConns(DefaultMaxIdleConns)
	db.SetConnMaxLifetime(DefaultConnMaxLifetime)
	db.SetConnMaxIdleTime(DefaultConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultPingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("failed to close database after ping failure: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

// Health checks database connectivity. Returns nil if healthy.
func (db *DB) Health(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultPingTimeout)
		defer cancel()
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}
