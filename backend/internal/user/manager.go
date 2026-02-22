package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"navplane/internal/jwtauth"

	"github.com/google/uuid"
)

// Domain errors
var (
	ErrNotFound         = errors.New("user not found")
	ErrInvalidEmail     = errors.New("invalid email")
	ErrNotMember        = errors.New("user is not a member of this organization")
	ErrAlreadyMember    = errors.New("user is already a member of this organization")
	ErrInvalidAuth0ID   = errors.New("invalid Auth0 user ID")
	ErrCannotRemoveOwner = errors.New("cannot remove the owner from the organization")
)

// Manager handles business logic for users.
type Manager struct {
	ds *Datastore
}

// NewManager creates a new user manager.
func NewManager(ds *Datastore) *Manager {
	return &Manager{ds: ds}
}

// UpsertFromClaims creates or updates a user identity from JWT claims.
// This is called on login to sync Auth0 user data.
func (m *Manager) UpsertFromClaims(ctx context.Context, claims *jwtauth.Claims) (*Identity, error) {
	auth0ID := claims.Auth0UserID()
	if auth0ID == "" {
		return nil, ErrInvalidAuth0ID
	}

	email := strings.TrimSpace(claims.Email)
	if email == "" {
		return nil, ErrInvalidEmail
	}

	identity := &Identity{
		Auth0UserID: auth0ID,
		Email:       email,
		Name:        claims.Name,
		IsAdmin:     false, // Admin status is set separately
	}

	if err := m.ds.UpsertIdentity(ctx, identity); err != nil {
		return nil, fmt.Errorf("failed to upsert identity: %w", err)
	}

	return identity, nil
}

// GetByID retrieves a user by ID.
func (m *Manager) GetByID(ctx context.Context, id uuid.UUID) (*Identity, error) {
	identity, err := m.ds.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return identity, nil
}

// GetByAuth0ID retrieves a user by Auth0 user ID.
func (m *Manager) GetByAuth0ID(ctx context.Context, auth0ID string) (*Identity, error) {
	identity, err := m.ds.GetByAuth0ID(ctx, auth0ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return identity, nil
}

// SetAdmin sets the admin status for a user.
func (m *Manager) SetAdmin(ctx context.Context, id uuid.UUID, isAdmin bool) error {
	rowsAffected, err := m.ds.SetAdmin(ctx, id, isAdmin)
	if err != nil {
		return fmt.Errorf("failed to set admin status: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// AddToOrg adds a user to an organization.
func (m *Manager) AddToOrg(ctx context.Context, orgID, userID uuid.UUID, role string) (*OrgMember, error) {
	if role == "" {
		role = RoleMember
	}

	// Check if already a member
	isMember, err := m.ds.IsMember(ctx, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}
	if isMember {
		return nil, ErrAlreadyMember
	}

	member := &OrgMember{
		OrgID:  orgID,
		UserID: userID,
		Role:   role,
	}

	if err := m.ds.AddOrgMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	return member, nil
}

// RemoveFromOrg removes a user from an organization.
func (m *Manager) RemoveFromOrg(ctx context.Context, orgID, userID uuid.UUID) error {
	// Check current membership to prevent removing owner
	membership, err := m.ds.GetOrgMembership(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotMember
		}
		return fmt.Errorf("failed to get membership: %w", err)
	}

	if membership.Role == RoleOwner {
		return ErrCannotRemoveOwner
	}

	rowsAffected, err := m.ds.RemoveOrgMember(ctx, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotMember
	}

	return nil
}

// IsMember checks if a user is a member of an organization.
func (m *Manager) IsMember(ctx context.Context, orgID, userID uuid.UUID) (bool, error) {
	return m.ds.IsMember(ctx, orgID, userID)
}

// ListUserOrgs retrieves all organizations a user is a member of.
func (m *Manager) ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*OrgMember, error) {
	return m.ds.ListUserOrgs(ctx, userID)
}

// ListOrgMembers retrieves all members of an organization.
func (m *Manager) ListOrgMembers(ctx context.Context, orgID uuid.UUID) ([]*OrgMember, error) {
	return m.ds.ListOrgMembers(ctx, orgID)
}

// CanAccessOrg checks if a user can access an organization.
// Admins can access any org, regular users must be members.
func (m *Manager) CanAccessOrg(ctx context.Context, userID, orgID uuid.UUID) (bool, error) {
	user, err := m.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}

	// Admins can access any org
	if user.IsAdmin {
		return true, nil
	}

	// Regular users must be members
	return m.IsMember(ctx, orgID, userID)
}
