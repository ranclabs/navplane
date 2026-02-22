package org

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestManager_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs(sqlmock.AnyArg(), "Test Org", sqlmock.AnyArg(), true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now))

	result, err := m.Create(ctx, "Test Org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Org.Name != "Test Org" {
		t.Errorf("expected name 'Test Org', got %q", result.Org.Name)
	}
	if result.APIKey.Plaintext == "" {
		t.Error("expected API key to be generated")
	}
	if result.APIKey.Hash == "" {
		t.Error("expected API key hash to be generated")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestManager_Create_InvalidName(t *testing.T) {
	m := &Manager{ds: nil}

	tests := []struct {
		name    string
		orgName string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tabs and spaces", " \t "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := m.Create(context.Background(), tt.orgName)
			if !errors.Is(err, ErrInvalidName) {
				t.Errorf("expected ErrInvalidName, got %v", err)
			}
		})
	}
}

func TestManager_Create_TrimsWhitespace(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs(sqlmock.AnyArg(), "Trimmed Name", sqlmock.AnyArg(), true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now))

	result, err := m.Create(ctx, "  Trimmed Name  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Org.Name != "Trimmed Name" {
		t.Errorf("expected name 'Trimmed Name', got %q", result.Org.Name)
	}
}

func TestManager_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()
	id := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"}).
		AddRow(id, "Test Org", "hash123", true, now, now)

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(rows)

	org, err := m.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if org.Name != "Test Org" {
		t.Errorf("expected name 'Test Org', got %q", org.Name)
	}
}

func TestManager_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	_, err = m.GetByID(ctx, id)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestManager_Authenticate_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()
	id := uuid.New()
	now := time.Now()

	apiKey := "np_test-key-12345"
	hash := HashAPIKey(apiKey)

	rows := sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"}).
		AddRow(id, "Test Org", hash, true, now, now)

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE api_key_hash = \$1`).
		WithArgs(hash).
		WillReturnRows(rows)

	org, err := m.Authenticate(ctx, apiKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if org.Name != "Test Org" {
		t.Errorf("expected name 'Test Org', got %q", org.Name)
	}
}

func TestManager_Authenticate_InvalidKey(t *testing.T) {
	m := &Manager{ds: nil}

	tests := []struct {
		name   string
		apiKey string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"wrong prefix", "sk_abc123"},
		{"no prefix", "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := m.Authenticate(context.Background(), tt.apiKey)
			if !errors.Is(err, ErrInvalidKey) {
				t.Errorf("expected ErrInvalidKey, got %v", err)
			}
		})
	}
}

func TestManager_Authenticate_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE api_key_hash = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	_, err = m.Authenticate(ctx, "np_nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestManager_Authenticate_OrgDisabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()
	id := uuid.New()
	now := time.Now()

	apiKey := "np_test-key-12345"
	hash := HashAPIKey(apiKey)

	rows := sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"}).
		AddRow(id, "Test Org", hash, false, now, now) // enabled = false

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE api_key_hash = \$1`).
		WithArgs(hash).
		WillReturnRows(rows)

	_, err = m.Authenticate(ctx, apiKey)
	if !errors.Is(err, ErrOrgDisabled) {
		t.Errorf("expected ErrOrgDisabled, got %v", err)
	}
}

func TestManager_Enable_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(`UPDATE organizations`).
		WithArgs(id, true).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = m.Enable(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManager_Enable_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(`UPDATE organizations`).
		WithArgs(id, true).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = m.Enable(ctx, id)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestManager_Disable_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(`UPDATE organizations`).
		WithArgs(id, false).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = m.Disable(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManager_Delete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(`DELETE FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = m.Delete(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManager_Delete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(`DELETE FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = m.Delete(ctx, id)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestManager_List_NormalizesPagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	m := NewManager(ds)
	ctx := context.Background()

	tests := []struct {
		name          string
		inputLimit    int
		inputOffset   int
		expectedLimit int
		expectedOff   int
	}{
		{"negative limit uses default", -5, 0, 20, 0},
		{"zero limit uses default", 0, 0, 20, 0},
		{"excessive limit capped", 500, 0, 100, 0},
		{"negative offset zeroed", 10, -5, 10, 0},
		{"valid params unchanged", 50, 10, 50, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"})

			mock.ExpectQuery(`SELECT .+ FROM organizations ORDER BY created_at DESC LIMIT \$1 OFFSET \$2`).
				WithArgs(tt.expectedLimit, tt.expectedOff).
				WillReturnRows(rows)

			_, err := m.List(ctx, tt.inputLimit, tt.inputOffset)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestManager_Update_InvalidName(t *testing.T) {
	m := &Manager{ds: nil}

	tests := []struct {
		name    string
		orgName string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.Update(context.Background(), uuid.New(), tt.orgName)
			if !errors.Is(err, ErrInvalidName) {
				t.Errorf("expected ErrInvalidName, got %v", err)
			}
		})
	}
}
