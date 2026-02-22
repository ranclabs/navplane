package org

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Datastore handles persistence operations for organizations.
// It performs only database operations and returns raw errors.
// Business logic and error translation belong in the Manager.
type Datastore struct {
	db *sql.DB
}

// NewDatastore creates a new organization datastore.
func NewDatastore(db *sql.DB) *Datastore {
	return &Datastore{db: db}
}

// Create inserts a new organization into the database.
// Returns the created org or raw database error.
func (ds *Datastore) Create(ctx context.Context, name, apiKeyHash string) (*Org, error) {
	org := &Org{
		ID:         uuid.New(),
		Name:       name,
		APIKeyHash: apiKeyHash,
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	query := `
		INSERT INTO organizations (id, name, api_key_hash, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`

	err := ds.db.QueryRowContext(ctx, query,
		org.ID, org.Name, org.APIKeyHash, org.Enabled, org.CreatedAt, org.UpdatedAt,
	).Scan(&org.CreatedAt, &org.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return org, nil
}

// GetByID retrieves an organization by its ID.
// Returns sql.ErrNoRows if not found.
func (ds *Datastore) GetByID(ctx context.Context, id uuid.UUID) (*Org, error) {
	query := `
		SELECT id, name, api_key_hash, enabled, created_at, updated_at
		FROM organizations
		WHERE id = $1`

	org := &Org{}
	err := ds.db.QueryRowContext(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.APIKeyHash, &org.Enabled, &org.CreatedAt, &org.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return org, nil
}

// GetByAPIKeyHash retrieves an organization by its API key hash.
// Returns sql.ErrNoRows if not found.
func (ds *Datastore) GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*Org, error) {
	query := `
		SELECT id, name, api_key_hash, enabled, created_at, updated_at
		FROM organizations
		WHERE api_key_hash = $1`

	org := &Org{}
	err := ds.db.QueryRowContext(ctx, query, apiKeyHash).Scan(
		&org.ID, &org.Name, &org.APIKeyHash, &org.Enabled, &org.CreatedAt, &org.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return org, nil
}

// Update modifies an existing organization.
// Returns sql.ErrNoRows equivalent via RowsAffected check.
func (ds *Datastore) Update(ctx context.Context, org *Org) (int64, error) {
	query := `
		UPDATE organizations
		SET name = $2, api_key_hash = $3, enabled = $4, updated_at = NOW()
		WHERE id = $1`

	result, err := ds.db.ExecContext(ctx, query, org.ID, org.Name, org.APIKeyHash, org.Enabled)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Delete removes an organization from the database.
// Returns rows affected count for caller to interpret.
func (ds *Datastore) Delete(ctx context.Context, id uuid.UUID) (int64, error) {
	query := `DELETE FROM organizations WHERE id = $1`

	result, err := ds.db.ExecContext(ctx, query, id)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// SetEnabled updates the enabled status of an organization.
// Returns rows affected count for caller to interpret.
func (ds *Datastore) SetEnabled(ctx context.Context, id uuid.UUID, enabled bool) (int64, error) {
	query := `
		UPDATE organizations
		SET enabled = $2, updated_at = NOW()
		WHERE id = $1`

	result, err := ds.db.ExecContext(ctx, query, id, enabled)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// List retrieves all organizations with pagination.
func (ds *Datastore) List(ctx context.Context, limit, offset int) ([]*Org, error) {
	query := `
		SELECT id, name, api_key_hash, enabled, created_at, updated_at
		FROM organizations
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := ds.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var orgs []*Org
	for rows.Next() {
		org := &Org{}
		if err := rows.Scan(
			&org.ID, &org.Name, &org.APIKeyHash, &org.Enabled, &org.CreatedAt, &org.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orgs, nil
}
