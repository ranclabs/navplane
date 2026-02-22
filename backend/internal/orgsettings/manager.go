package orgsettings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"navplane/internal/provider"

	"github.com/google/uuid"
)

// Domain errors
var (
	ErrNotFound        = errors.New("provider settings not found")
	ErrInvalidProvider = errors.New("invalid provider")
	ErrProviderDisabled = errors.New("provider is disabled for this organization")
	ErrModelBlocked    = errors.New("model is blocked for this organization")
)

// Manager handles business logic for org provider settings.
type Manager struct {
	ds       *Datastore
	registry *provider.Registry
}

// NewManager creates a new org settings manager.
func NewManager(ds *Datastore, registry *provider.Registry) *Manager {
	return &Manager{
		ds:       ds,
		registry: registry,
	}
}

// GetOrDefault retrieves provider settings for an org, or returns defaults if none exist.
func (m *Manager) GetOrDefault(ctx context.Context, orgID uuid.UUID, providerName string) (*ProviderSettings, error) {
	providerName = strings.ToLower(providerName)

	settings, err := m.ds.GetByOrgAndProvider(ctx, orgID, providerName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return default settings (enabled, no restrictions)
			return &ProviderSettings{
				OrgID:    orgID,
				Provider: providerName,
				Enabled:  true,
			}, nil
		}
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	return settings, nil
}

// ListByOrg retrieves all provider settings for an organization.
func (m *Manager) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*ProviderSettings, error) {
	settings, err := m.ds.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list settings: %w", err)
	}
	return settings, nil
}

// UpdateInput holds input for updating provider settings.
type UpdateInput struct {
	OrgID         uuid.UUID
	Provider      string
	Enabled       *bool
	AllowedModels []string
	BlockedModels []string
}

// Update updates provider settings for an organization.
func (m *Manager) Update(ctx context.Context, input UpdateInput) (*ProviderSettings, error) {
	input.Provider = strings.ToLower(input.Provider)

	// Validate provider
	if _, err := m.registry.Get(input.Provider); err != nil {
		return nil, ErrInvalidProvider
	}

	// Get current settings or create new
	settings, err := m.ds.GetByOrgAndProvider(ctx, input.OrgID, input.Provider)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			settings = &ProviderSettings{
				OrgID:    input.OrgID,
				Provider: input.Provider,
				Enabled:  true,
			}
		} else {
			return nil, fmt.Errorf("failed to get settings: %w", err)
		}
	}

	// Apply updates
	if input.Enabled != nil {
		settings.Enabled = *input.Enabled
	}
	if input.AllowedModels != nil {
		settings.AllowedModels = input.AllowedModels
	}
	if input.BlockedModels != nil {
		settings.BlockedModels = input.BlockedModels
	}

	if err := m.ds.Upsert(ctx, settings); err != nil {
		return nil, fmt.Errorf("failed to update settings: %w", err)
	}

	return settings, nil
}

// EnableProvider enables a provider for an organization.
func (m *Manager) EnableProvider(ctx context.Context, orgID uuid.UUID, providerName string) error {
	enabled := true
	_, err := m.Update(ctx, UpdateInput{
		OrgID:    orgID,
		Provider: providerName,
		Enabled:  &enabled,
	})
	return err
}

// DisableProvider disables a provider for an organization.
func (m *Manager) DisableProvider(ctx context.Context, orgID uuid.UUID, providerName string) error {
	enabled := false
	_, err := m.Update(ctx, UpdateInput{
		OrgID:    orgID,
		Provider: providerName,
		Enabled:  &enabled,
	})
	return err
}

// CheckAccess verifies that an org can use a specific provider and model.
func (m *Manager) CheckAccess(ctx context.Context, orgID uuid.UUID, providerName, model string) error {
	settings, err := m.GetOrDefault(ctx, orgID, providerName)
	if err != nil {
		return err
	}

	if !settings.Enabled {
		return ErrProviderDisabled
	}

	if !settings.IsModelAllowed(model) {
		return ErrModelBlocked
	}

	return nil
}
