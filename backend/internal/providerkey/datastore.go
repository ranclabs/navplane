package providerkey

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// DBTX is the interface for database operations (supports both *sql.DB and *sql.Tx).
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Datastore handles database operations for provider keys.
type Datastore struct {
	db DBTX
}

// NewDatastore creates a new provider key datastore.
func NewDatastore(db DBTX) *Datastore {
	return &Datastore{db: db}
}

// Create inserts a new provider key.
func (ds *Datastore) Create(ctx context.Context, pk *ProviderKey) error {
	pk.ID = uuid.New()
	now := time.Now()

	query := `
		INSERT INTO provider_keys (
			id, org_id, provider, key_alias, encrypted_key, key_nonce,
			encrypted_dek, dek_nonce, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	return ds.db.QueryRowContext(ctx, query,
		pk.ID, pk.OrgID, pk.Provider, pk.KeyAlias,
		pk.EncryptedKey, pk.KeyNonce, pk.EncryptedDEK, pk.DEKNonce,
		pk.IsActive, now, now,
	).Scan(&pk.CreatedAt, &pk.UpdatedAt)
}

// GetByID retrieves a provider key by ID.
func (ds *Datastore) GetByID(ctx context.Context, id uuid.UUID) (*ProviderKey, error) {
	query := `
		SELECT id, org_id, provider, key_alias, encrypted_key, key_nonce,
		       encrypted_dek, dek_nonce, is_active, created_at, updated_at
		FROM provider_keys WHERE id = $1`

	pk := &ProviderKey{}
	err := ds.db.QueryRowContext(ctx, query, id).Scan(
		&pk.ID, &pk.OrgID, &pk.Provider, &pk.KeyAlias,
		&pk.EncryptedKey, &pk.KeyNonce, &pk.EncryptedDEK, &pk.DEKNonce,
		&pk.IsActive, &pk.CreatedAt, &pk.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return pk, nil
}

// GetActiveByOrgAndProvider retrieves the active provider key for an org and provider.
func (ds *Datastore) GetActiveByOrgAndProvider(ctx context.Context, orgID uuid.UUID, provider string) (*ProviderKey, error) {
	query := `
		SELECT id, org_id, provider, key_alias, encrypted_key, key_nonce,
		       encrypted_dek, dek_nonce, is_active, created_at, updated_at
		FROM provider_keys
		WHERE org_id = $1 AND provider = $2 AND is_active = true
		LIMIT 1`

	pk := &ProviderKey{}
	err := ds.db.QueryRowContext(ctx, query, orgID, provider).Scan(
		&pk.ID, &pk.OrgID, &pk.Provider, &pk.KeyAlias,
		&pk.EncryptedKey, &pk.KeyNonce, &pk.EncryptedDEK, &pk.DEKNonce,
		&pk.IsActive, &pk.CreatedAt, &pk.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return pk, nil
}

// ListByOrg retrieves all provider keys for an organization.
func (ds *Datastore) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*ProviderKey, error) {
	query := `
		SELECT id, org_id, provider, key_alias, encrypted_key, key_nonce,
		       encrypted_dek, dek_nonce, is_active, created_at, updated_at
		FROM provider_keys
		WHERE org_id = $1
		ORDER BY created_at DESC`

	rows, err := ds.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*ProviderKey
	for rows.Next() {
		pk := &ProviderKey{}
		err := rows.Scan(
			&pk.ID, &pk.OrgID, &pk.Provider, &pk.KeyAlias,
			&pk.EncryptedKey, &pk.KeyNonce, &pk.EncryptedDEK, &pk.DEKNonce,
			&pk.IsActive, &pk.CreatedAt, &pk.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		keys = append(keys, pk)
	}

	return keys, rows.Err()
}

// SetActive sets the is_active flag for a provider key.
func (ds *Datastore) SetActive(ctx context.Context, id uuid.UUID, isActive bool) (int64, error) {
	query := `UPDATE provider_keys SET is_active = $2 WHERE id = $1`
	result, err := ds.db.ExecContext(ctx, query, id, isActive)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Delete removes a provider key.
func (ds *Datastore) Delete(ctx context.Context, id uuid.UUID) (int64, error) {
	query := `DELETE FROM provider_keys WHERE id = $1`
	result, err := ds.db.ExecContext(ctx, query, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// UpdateEncryption updates the encrypted DEK (for key rotation).
func (ds *Datastore) UpdateEncryption(ctx context.Context, id uuid.UUID, encryptedDEK, dekNonce []byte) (int64, error) {
	query := `UPDATE provider_keys SET encrypted_dek = $2, dek_nonce = $3 WHERE id = $1`
	result, err := ds.db.ExecContext(ctx, query, id, encryptedDEK, dekNonce)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
