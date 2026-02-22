package user

import (
	"time"

	"github.com/google/uuid"
)

// Identity represents a user identity from Auth0.
// We only store minimal information needed for authorization.
type Identity struct {
	ID          uuid.UUID `json:"id"`
	Auth0UserID string    `json:"auth0_user_id"`
	Email       string    `json:"email"`
	Name        string    `json:"name,omitempty"`
	IsAdmin     bool      `json:"is_admin"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OrgMember represents a user's membership in an organization.
type OrgMember struct {
	ID        uuid.UUID `json:"id"`
	OrgID     uuid.UUID `json:"org_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// Common roles
const (
	RoleOwner  = "owner"
	RoleMember = "member"
)
