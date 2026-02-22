package providerkey

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
	ErrNotFound        = errors.New("provider key not found")
	ErrInvalidProvider = errors.New("invalid provider")
	ErrInvalidAlias    = errors.New("invalid key alias")
	ErrInvalidKey      = errors.New("invalid API key")
	ErrKeyExists       = errors.New("provider key already exists for this org and provider")
)

// Manager handles business logic for provider keys.
type Manager struct {
	ds        *Datastore
	encryptor *Encryptor
	registry  *provider.Registry
}

// NewManager creates a new provider key manager.
func NewManager(ds *Datastore, encryptor *Encryptor, registry *provider.Registry) *Manager {
	return &Manager{
		ds:        ds,
		encryptor: encryptor,
		registry:  registry,
	}
}

// CreateInput holds the input for creating a provider key.
type CreateInput struct {
	OrgID       uuid.UUID
	Provider    string
	KeyAlias    string
	APIKey      string
	ValidateKey bool // If true, validate the key against the provider API
}

// Create creates a new provider key with encryption.
func (m *Manager) Create(ctx context.Context, input CreateInput) (*ProviderKey, error) {
	// Validate provider
	input.Provider = strings.TrimSpace(strings.ToLower(input.Provider))
	p, err := m.registry.Get(input.Provider)
	if err != nil {
		return nil, ErrInvalidProvider
	}

	// Validate alias
	input.KeyAlias = strings.TrimSpace(input.KeyAlias)
	if input.KeyAlias == "" {
		return nil, ErrInvalidAlias
	}

	// Validate API key
	input.APIKey = strings.TrimSpace(input.APIKey)
	if input.APIKey == "" {
		return nil, ErrInvalidKey
	}

	// Optionally validate the key against the provider
	if input.ValidateKey {
		if err := p.ValidateKey(ctx, input.APIKey); err != nil {
			if errors.Is(err, provider.ErrInvalidAPIKey) {
				return nil, ErrInvalidKey
			}
			return nil, fmt.Errorf("failed to validate key: %w", err)
		}
	}

	// Encrypt the API key
	encrypted, err := m.encryptor.Encrypt([]byte(input.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt key: %w", err)
	}

	pk := &ProviderKey{
		OrgID:        input.OrgID,
		Provider:     input.Provider,
		KeyAlias:     input.KeyAlias,
		EncryptedKey: encrypted.EncryptedKey,
		KeyNonce:     encrypted.KeyNonce,
		EncryptedDEK: encrypted.EncryptedDEK,
		DEKNonce:     encrypted.DEKNonce,
		IsActive:     true,
	}

	if err := m.ds.Create(ctx, pk); err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrKeyExists
		}
		return nil, fmt.Errorf("failed to create provider key: %w", err)
	}

	return pk, nil
}

// GetByID retrieves a provider key by ID (encrypted, not decrypted).
func (m *Manager) GetByID(ctx context.Context, id uuid.UUID) (*ProviderKey, error) {
	pk, err := m.ds.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get provider key: %w", err)
	}
	return pk, nil
}

// GetDecryptedKey retrieves and decrypts a provider API key for use in requests.
func (m *Manager) GetDecryptedKey(ctx context.Context, orgID uuid.UUID, providerName string) (string, error) {
	pk, err := m.ds.GetActiveByOrgAndProvider(ctx, orgID, providerName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get provider key: %w", err)
	}

	plaintext, err := m.encryptor.Decrypt(&EncryptedData{
		EncryptedKey: pk.EncryptedKey,
		KeyNonce:     pk.KeyNonce,
		EncryptedDEK: pk.EncryptedDEK,
		DEKNonce:     pk.DEKNonce,
	})
	if err != nil {
		return "", fmt.Errorf("failed to decrypt key: %w", err)
	}

	return string(plaintext), nil
}

// ListByOrg retrieves all provider keys for an organization (without decryption).
func (m *Manager) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*ProviderKey, error) {
	keys, err := m.ds.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list provider keys: %w", err)
	}
	return keys, nil
}

// Activate activates a provider key.
func (m *Manager) Activate(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := m.ds.SetActive(ctx, id, true)
	if err != nil {
		return fmt.Errorf("failed to activate provider key: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Deactivate deactivates a provider key.
func (m *Manager) Deactivate(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := m.ds.SetActive(ctx, id, false)
	if err != nil {
		return fmt.Errorf("failed to deactivate provider key: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a provider key.
func (m *Manager) Delete(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := m.ds.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider key: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
