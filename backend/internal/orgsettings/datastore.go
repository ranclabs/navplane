package orgsettings

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// DBTX is the interface for database operations.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Datastore handles database operations for org provider settings.
type Datastore struct {
	db DBTX
}

// NewDatastore creates a new org settings datastore.
func NewDatastore(db DBTX) *Datastore {
	return &Datastore{db: db}
}

// Upsert creates or updates provider settings for an organization.
func (ds *Datastore) Upsert(ctx context.Context, settings *ProviderSettings) error {
	now := time.Now()

	query := `
		INSERT INTO org_provider_settings (id, org_id, provider, enabled, allowed_models, blocked_models, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (org_id, provider)
		DO UPDATE SET enabled = $4, allowed_models = $5, blocked_models = $6, updated_at = $8
		RETURNING id, created_at, updated_at`

	if settings.ID == uuid.Nil {
		settings.ID = uuid.New()
	}

	return ds.db.QueryRowContext(ctx, query,
		settings.ID, settings.OrgID, settings.Provider, settings.Enabled,
		pq.Array(settings.AllowedModels), pq.Array(settings.BlockedModels),
		now, now,
	).Scan(&settings.ID, &settings.CreatedAt, &settings.UpdatedAt)
}

// GetByOrgAndProvider retrieves provider settings for a specific org and provider.
func (ds *Datastore) GetByOrgAndProvider(ctx context.Context, orgID uuid.UUID, provider string) (*ProviderSettings, error) {
	query := `
		SELECT id, org_id, provider, enabled, allowed_models, blocked_models, created_at, updated_at
		FROM org_provider_settings
		WHERE org_id = $1 AND provider = $2`

	settings := &ProviderSettings{}
	err := ds.db.QueryRowContext(ctx, query, orgID, provider).Scan(
		&settings.ID, &settings.OrgID, &settings.Provider, &settings.Enabled,
		pq.Array(&settings.AllowedModels), pq.Array(&settings.BlockedModels),
		&settings.CreatedAt, &settings.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

// ListByOrg retrieves all provider settings for an organization.
func (ds *Datastore) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*ProviderSettings, error) {
	query := `
		SELECT id, org_id, provider, enabled, allowed_models, blocked_models, created_at, updated_at
		FROM org_provider_settings
		WHERE org_id = $1
		ORDER BY provider`

	rows, err := ds.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []*ProviderSettings
	for rows.Next() {
		s := &ProviderSettings{}
		if err := rows.Scan(
			&s.ID, &s.OrgID, &s.Provider, &s.Enabled,
			pq.Array(&s.AllowedModels), pq.Array(&s.BlockedModels),
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}

	return settings, rows.Err()
}

// SetEnabled sets the enabled flag for a provider.
func (ds *Datastore) SetEnabled(ctx context.Context, orgID uuid.UUID, provider string, enabled bool) (int64, error) {
	query := `UPDATE org_provider_settings SET enabled = $3 WHERE org_id = $1 AND provider = $2`
	result, err := ds.db.ExecContext(ctx, query, orgID, provider, enabled)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Delete removes provider settings.
func (ds *Datastore) Delete(ctx context.Context, orgID uuid.UUID, provider string) (int64, error) {
	query := `DELETE FROM org_provider_settings WHERE org_id = $1 AND provider = $2`
	result, err := ds.db.ExecContext(ctx, query, orgID, provider)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
