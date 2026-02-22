package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"navplane/internal/org"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func setupAdminTest(t *testing.T) (*AdminOrgsHandler, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	ds := org.NewDatastore(db)
	manager := org.NewManager(ds)
	handler := NewAdminOrgsHandler(manager)
	return handler, mock, func() { db.Close() }
}

func TestAdminOrgsHandler_List(t *testing.T) {
	handler, mock, cleanup := setupAdminTest(t)
	defer cleanup()

	id1, id2 := uuid.New(), uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"}).
		AddRow(id1, "Org 1", "hash1", true, now, now).
		AddRow(id2, "Org 2", "hash2", false, now, now)

	mock.ExpectQuery(`SELECT .+ FROM organizations`).
		WithArgs(20, 0).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/admin/orgs", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	orgs := response["organizations"].([]any)
	if len(orgs) != 2 {
		t.Errorf("expected 2 orgs, got %d", len(orgs))
	}
}

func TestAdminOrgsHandler_Get(t *testing.T) {
	handler, mock, cleanup := setupAdminTest(t)
	defer cleanup()

	id := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"}).
		AddRow(id, "Test Org", "hash123", true, now, now)

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/admin/orgs/"+id.String(), nil)
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response orgResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Name != "Test Org" {
		t.Errorf("expected name 'Test Org', got %q", response.Name)
	}
}

func TestAdminOrgsHandler_Get_NotFound(t *testing.T) {
	handler, mock, cleanup := setupAdminTest(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"}))

	req := httptest.NewRequest(http.MethodGet, "/admin/orgs/"+id.String(), nil)
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestAdminOrgsHandler_Create(t *testing.T) {
	handler, mock, cleanup := setupAdminTest(t)
	defer cleanup()

	now := time.Now()

	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs(sqlmock.AnyArg(), "New Org", sqlmock.AnyArg(), true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now))

	body := bytes.NewBufferString(`{"name": "New Org"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/orgs", body)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var response createOrgResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Name != "New Org" {
		t.Errorf("expected name 'New Org', got %q", response.Name)
	}
	if response.APIKey == "" {
		t.Error("expected API key to be returned")
	}
}

func TestAdminOrgsHandler_Create_InvalidName(t *testing.T) {
	handler, _, cleanup := setupAdminTest(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"name": ""}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/orgs", body)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestAdminOrgsHandler_SetEnabled_Disable(t *testing.T) {
	handler, mock, cleanup := setupAdminTest(t)
	defer cleanup()

	id := uuid.New()
	now := time.Now()

	mock.ExpectExec(`UPDATE organizations`).
		WithArgs(id, false).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"}).
			AddRow(id, "Test Org", "hash123", false, now, now))

	body := bytes.NewBufferString(`{"enabled": false}`)
	req := httptest.NewRequest(http.MethodPut, "/admin/orgs/"+id.String()+"/enabled", body)
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()

	handler.SetEnabled(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response orgResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Enabled {
		t.Error("expected org to be disabled")
	}
}

func TestAdminOrgsHandler_SetEnabled_Enable(t *testing.T) {
	handler, mock, cleanup := setupAdminTest(t)
	defer cleanup()

	id := uuid.New()
	now := time.Now()

	mock.ExpectExec(`UPDATE organizations`).
		WithArgs(id, true).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"}).
			AddRow(id, "Test Org", "hash123", true, now, now))

	body := bytes.NewBufferString(`{"enabled": true}`)
	req := httptest.NewRequest(http.MethodPut, "/admin/orgs/"+id.String()+"/enabled", body)
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()

	handler.SetEnabled(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response orgResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !response.Enabled {
		t.Error("expected org to be enabled")
	}
}

func TestAdminOrgsHandler_Delete(t *testing.T) {
	handler, mock, cleanup := setupAdminTest(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectExec(`DELETE FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/admin/orgs/"+id.String(), nil)
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()

	handler.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}
}

func TestAdminOrgsHandler_Delete_NotFound(t *testing.T) {
	handler, mock, cleanup := setupAdminTest(t)
	defer cleanup()

	id := uuid.New()

	mock.ExpectExec(`DELETE FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(http.MethodDelete, "/admin/orgs/"+id.String(), nil)
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()

	handler.Delete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestAdminOrgsHandler_RotateAPIKey(t *testing.T) {
	handler, mock, cleanup := setupAdminTest(t)
	defer cleanup()

	id := uuid.New()
	now := time.Now()

	// GetByID call
	mock.ExpectQuery(`SELECT .+ FROM organizations WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "api_key_hash", "enabled", "created_at", "updated_at"}).
			AddRow(id, "Test Org", "hash123", true, now, now))

	// Update call
	mock.ExpectExec(`UPDATE organizations`).
		WithArgs(id, "Test Org", sqlmock.AnyArg(), true).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, "/admin/orgs/"+id.String()+"/rotate-key", nil)
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()

	handler.RotateAPIKey(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["api_key"] == "" {
		t.Error("expected API key to be returned")
	}
}

func TestParseOrgID_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/orgs/invalid", nil)
	req.SetPathValue("id", "invalid")

	_, err := parseOrgID(req)
	if err == nil {
		t.Error("expected error for invalid UUID")
	}
}

func TestParseOrgID_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/orgs/", nil)
	req = req.WithContext(context.Background())

	_, err := parseOrgID(req)
	if err == nil {
		t.Error("expected error for missing ID")
	}
}
