package user

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// DBTX is the interface for database operations.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Datastore handles database operations for users.
type Datastore struct {
	db DBTX
}

// NewDatastore creates a new user datastore.
func NewDatastore(db DBTX) *Datastore {
	return &Datastore{db: db}
}

// UpsertIdentity creates or updates a user identity.
// This is used when a user logs in via Auth0.
func (ds *Datastore) UpsertIdentity(ctx context.Context, identity *Identity) error {
	now := time.Now()

	query := `
		INSERT INTO user_identities (id, auth0_user_id, email, name, is_admin, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (auth0_user_id)
		DO UPDATE SET email = $3, name = $4, updated_at = $7
		RETURNING id, created_at, updated_at`

	if identity.ID == uuid.Nil {
		identity.ID = uuid.New()
	}

	return ds.db.QueryRowContext(ctx, query,
		identity.ID, identity.Auth0UserID, identity.Email, identity.Name,
		identity.IsAdmin, now, now,
	).Scan(&identity.ID, &identity.CreatedAt, &identity.UpdatedAt)
}

// GetByID retrieves a user identity by ID.
func (ds *Datastore) GetByID(ctx context.Context, id uuid.UUID) (*Identity, error) {
	query := `
		SELECT id, auth0_user_id, email, name, is_admin, created_at, updated_at
		FROM user_identities WHERE id = $1`

	identity := &Identity{}
	err := ds.db.QueryRowContext(ctx, query, id).Scan(
		&identity.ID, &identity.Auth0UserID, &identity.Email, &identity.Name,
		&identity.IsAdmin, &identity.CreatedAt, &identity.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

// GetByAuth0ID retrieves a user identity by Auth0 user ID.
func (ds *Datastore) GetByAuth0ID(ctx context.Context, auth0UserID string) (*Identity, error) {
	query := `
		SELECT id, auth0_user_id, email, name, is_admin, created_at, updated_at
		FROM user_identities WHERE auth0_user_id = $1`

	identity := &Identity{}
	err := ds.db.QueryRowContext(ctx, query, auth0UserID).Scan(
		&identity.ID, &identity.Auth0UserID, &identity.Email, &identity.Name,
		&identity.IsAdmin, &identity.CreatedAt, &identity.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

// SetAdmin sets the admin status for a user.
func (ds *Datastore) SetAdmin(ctx context.Context, id uuid.UUID, isAdmin bool) (int64, error) {
	query := `UPDATE user_identities SET is_admin = $2 WHERE id = $1`
	result, err := ds.db.ExecContext(ctx, query, id, isAdmin)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Delete removes a user identity.
func (ds *Datastore) Delete(ctx context.Context, id uuid.UUID) (int64, error) {
	query := `DELETE FROM user_identities WHERE id = $1`
	result, err := ds.db.ExecContext(ctx, query, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// --- Org Membership ---

// AddOrgMember adds a user to an organization.
func (ds *Datastore) AddOrgMember(ctx context.Context, member *OrgMember) error {
	member.ID = uuid.New()
	now := time.Now()

	query := `
		INSERT INTO org_members (id, org_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at`

	return ds.db.QueryRowContext(ctx, query,
		member.ID, member.OrgID, member.UserID, member.Role, now,
	).Scan(&member.CreatedAt)
}

// RemoveOrgMember removes a user from an organization.
func (ds *Datastore) RemoveOrgMember(ctx context.Context, orgID, userID uuid.UUID) (int64, error) {
	query := `DELETE FROM org_members WHERE org_id = $1 AND user_id = $2`
	result, err := ds.db.ExecContext(ctx, query, orgID, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetOrgMembership retrieves a user's membership in an organization.
func (ds *Datastore) GetOrgMembership(ctx context.Context, orgID, userID uuid.UUID) (*OrgMember, error) {
	query := `
		SELECT id, org_id, user_id, role, created_at
		FROM org_members WHERE org_id = $1 AND user_id = $2`

	member := &OrgMember{}
	err := ds.db.QueryRowContext(ctx, query, orgID, userID).Scan(
		&member.ID, &member.OrgID, &member.UserID, &member.Role, &member.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return member, nil
}

// ListUserOrgs retrieves all organizations a user is a member of.
func (ds *Datastore) ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*OrgMember, error) {
	query := `
		SELECT id, org_id, user_id, role, created_at
		FROM org_members WHERE user_id = $1`

	rows, err := ds.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*OrgMember
	for rows.Next() {
		member := &OrgMember{}
		if err := rows.Scan(&member.ID, &member.OrgID, &member.UserID, &member.Role, &member.CreatedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

// ListOrgMembers retrieves all members of an organization.
func (ds *Datastore) ListOrgMembers(ctx context.Context, orgID uuid.UUID) ([]*OrgMember, error) {
	query := `
		SELECT id, org_id, user_id, role, created_at
		FROM org_members WHERE org_id = $1`

	rows, err := ds.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*OrgMember
	for rows.Next() {
		member := &OrgMember{}
		if err := rows.Scan(&member.ID, &member.OrgID, &member.UserID, &member.Role, &member.CreatedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

// IsMember checks if a user is a member of an organization.
func (ds *Datastore) IsMember(ctx context.Context, orgID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM org_members WHERE org_id = $1 AND user_id = $2)`
	var exists bool
	err := ds.db.QueryRowContext(ctx, query, orgID, userID).Scan(&exists)
	return exists, err
}
