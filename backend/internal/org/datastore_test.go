package org

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestDatastore_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs(sqlmock.AnyArg(), "Test Org", "hash123", true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now))

	org, err := ds.Create(ctx, "Test Org", "hash123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if org.Name != "Test Org" {
		t.Errorf("expected name 'Test Org', got %q", org.Name)
	}
	if org.APIKeyHash != "hash123" {
		t.Errorf("expected hash 'hash123', got %q", org.APIKeyHash)
	}
	if !org.Enabled {
		t.Error("expected org to be enabled")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDatastore_Create_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	ctx := context.Background()

	mock.ExpectQuery(`INSERT INTO organizations`).
		WillReturnError(sql.ErrConnDone)

	_, err = ds.Create(ctx, "Test Org", "hash123")
	if err == nil {
		t.Error("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDatastore_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	ctx := context.Background()
	id := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"}).
		AddRow(id, "Test Org", "hash123", true, now, now)

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(rows)

	org, err := ds.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if org.ID != id {
		t.Errorf("expected ID %v, got %v", id, org.ID)
	}
	if org.Name != "Test Org" {
		t.Errorf("expected name 'Test Org', got %q", org.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDatastore_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	_, err = ds.GetByID(ctx, id)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDatastore_GetByAPIKeyHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	ctx := context.Background()
	id := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"}).
		AddRow(id, "Test Org", "hash123", true, now, now)

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE api_key_hash = \$1`).
		WithArgs("hash123").
		WillReturnRows(rows)

	org, err := ds.GetByAPIKeyHash(ctx, "hash123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if org.APIKeyHash != "hash123" {
		t.Errorf("expected hash 'hash123', got %q", org.APIKeyHash)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDatastore_GetByAPIKeyHash_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE api_key_hash = \$1`).
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	_, err = ds.GetByAPIKeyHash(ctx, "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDatastore_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	ctx := context.Background()
	id := uuid.New()

	org := &Org{
		ID:         id,
		Name:       "Updated Org",
		APIKeyHash: "hash456",
		Enabled:    false,
	}

	mock.ExpectExec(`UPDATE organizations`).
		WithArgs(id, "Updated Org", "hash456", false).
		WillReturnResult(sqlmock.NewResult(0, 1))

	rowsAffected, err := ds.Update(ctx, org)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", rowsAffected)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDatastore_Update_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	ctx := context.Background()
	id := uuid.New()

	org := &Org{
		ID:         id,
		Name:       "Updated Org",
		APIKeyHash: "hash456",
		Enabled:    false,
	}

	mock.ExpectExec(`UPDATE organizations`).
		WithArgs(id, "Updated Org", "hash456", false).
		WillReturnResult(sqlmock.NewResult(0, 0))

	rowsAffected, err := ds.Update(ctx, org)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rowsAffected != 0 {
		t.Errorf("expected 0 rows affected, got %d", rowsAffected)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDatastore_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(`DELETE FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	rowsAffected, err := ds.Delete(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", rowsAffected)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDatastore_SetEnabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(`UPDATE organizations`).
		WithArgs(id, false).
		WillReturnResult(sqlmock.NewResult(0, 1))

	rowsAffected, err := ds.SetEnabled(ctx, id, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", rowsAffected)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDatastore_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	ctx := context.Background()
	id1, id2 := uuid.New(), uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"}).
		AddRow(id1, "Org 1", "hash1", true, now, now).
		AddRow(id2, "Org 2", "hash2", false, now, now)

	mock.ExpectQuery(`SELECT .+ FROM organizations ORDER BY created_at DESC LIMIT \$1 OFFSET \$2`).
		WithArgs(10, 0).
		WillReturnRows(rows)

	orgs, err := ds.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(orgs) != 2 {
		t.Errorf("expected 2 orgs, got %d", len(orgs))
	}

	if orgs[0].Name != "Org 1" {
		t.Errorf("expected first org name 'Org 1', got %q", orgs[0].Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDatastore_List_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"})

	mock.ExpectQuery(`SELECT .+ FROM organizations ORDER BY created_at DESC LIMIT \$1 OFFSET \$2`).
		WithArgs(10, 0).
		WillReturnRows(rows)

	orgs, err := ds.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(orgs) != 0 {
		t.Errorf("expected 0 orgs, got %d", len(orgs))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
