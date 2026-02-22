package org

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Domain errors returned by the Manager.
var (
	ErrNotFound    = errors.New("organization not found")
	ErrInvalidName = errors.New("organization name is required")
	ErrInvalidKey  = errors.New("invalid API key format")
	ErrOrgDisabled = errors.New("organization is disabled")
)

// Manager handles business logic for organizations.
// It coordinates operations and translates datastore errors to domain errors.
type Manager struct {
	ds *Datastore
}

// NewManager creates a new organization manager.
func NewManager(ds *Datastore) *Manager {
	return &Manager{ds: ds}
}

// CreateOrgResult contains the result of creating an organization.
// The API key plaintext is only available at creation time.
type CreateOrgResult struct {
	Org    *Org
	APIKey APIKey
}

// Create creates a new organization with a generated API key.
// Returns the org and the plaintext API key (only available once).
func (m *Manager) Create(ctx context.Context, name string) (*CreateOrgResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidName
	}

	apiKey := GenerateAPIKey()

	org, err := m.ds.Create(ctx, name, apiKey.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return &CreateOrgResult{
		Org:    org,
		APIKey: apiKey,
	}, nil
}

// GetByID retrieves an organization by ID.
func (m *Manager) GetByID(ctx context.Context, id uuid.UUID) (*Org, error) {
	org, err := m.ds.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return org, nil
}

// Authenticate validates an API key and returns the associated organization.
// Returns ErrNotFound if key doesn't exist, ErrOrgDisabled if org is disabled.
func (m *Manager) Authenticate(ctx context.Context, apiKey string) (*Org, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" || !strings.HasPrefix(apiKey, "np_") {
		return nil, ErrInvalidKey
	}

	hash := HashAPIKey(apiKey)
	org, err := m.ds.GetByAPIKeyHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	if !org.Enabled {
		return nil, ErrOrgDisabled
	}

	return org, nil
}

// Enable enables an organization.
func (m *Manager) Enable(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := m.ds.SetEnabled(ctx, id, true)
	if err != nil {
		return fmt.Errorf("failed to enable organization: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Disable disables an organization (kill switch).
func (m *Manager) Disable(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := m.ds.SetEnabled(ctx, id, false)
	if err != nil {
		return fmt.Errorf("failed to disable organization: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Update updates an organization's name.
func (m *Manager) Update(ctx context.Context, id uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}

	org, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}

	org.Name = name
	rowsAffected, err := m.ds.Update(ctx, org)
	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes an organization and all associated data.
func (m *Manager) Delete(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := m.ds.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// List retrieves organizations with pagination.
func (m *Manager) List(ctx context.Context, limit, offset int) ([]*Org, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	orgs, err := m.ds.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	return orgs, nil
}

// RotateAPIKey generates a new API key for an organization.
// Returns the new plaintext key (only available once).
func (m *Manager) RotateAPIKey(ctx context.Context, id uuid.UUID) (*APIKey, error) {
	org, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	newKey := GenerateAPIKey()
	org.APIKeyHash = newKey.Hash

	rowsAffected, err := m.ds.Update(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("failed to rotate API key: %w", err)
	}
	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	return &newKey, nil
}
