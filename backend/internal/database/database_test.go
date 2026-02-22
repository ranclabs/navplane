package database

import (
	"testing"

	"lectr/internal/config"
)

func TestNew_InvalidURL(t *testing.T) {
	cfg := config.DatabaseConfig{
		URL:             "invalid-url",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("New should not fail for invalid URL (sql.Open doesn't validate): %v", err)
	}
	defer db.Close()

	// Ping should fail for invalid URL
	err = db.Ping()
	if err == nil {
		t.Error("Ping should fail for invalid connection string")
	}
}

func TestNew_ConfiguresPoolSettings(t *testing.T) {
	cfg := config.DatabaseConfig{
		URL:             "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		MaxOpenConns:    10,
		MaxIdleConns:    3,
		ConnMaxLifetime: 120,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer db.Close()

	stats := db.Stats()
	if stats.MaxOpenConnections != 10 {
		t.Errorf("expected MaxOpenConnections to be 10, got %d", stats.MaxOpenConnections)
	}
}

func TestConnect_FailsForUnreachableDB(t *testing.T) {
	cfg := config.DatabaseConfig{
		URL:             "postgres://user:pass@localhost:59999/nonexistent?sslmode=disable&connect_timeout=1",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 60,
	}

	_, err := Connect(cfg)
	if err == nil {
		t.Error("Connect should fail for unreachable database")
	}
}
