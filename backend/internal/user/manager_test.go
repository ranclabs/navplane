package user

import (
	"context"
	"testing"
	"time"

	"navplane/internal/jwtauth"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestManager_UpsertFromClaims(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	mgr := NewManager(ds)
	ctx := context.Background()

	claims := &jwtauth.Claims{
		Email: "test@example.com",
		Name:  "Test User",
	}
	claims.Subject = "auth0|123456"

	now := time.Now()
	id := uuid.New()

	mock.ExpectQuery(`INSERT INTO user_identities`).
		WithArgs(sqlmock.AnyArg(), "auth0|123456", "test@example.com", "Test User", false, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(id, now, now))

	identity, err := mgr.UpsertFromClaims(ctx, claims)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if identity.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", identity.Email)
	}
	if identity.Auth0UserID != "auth0|123456" {
		t.Errorf("expected auth0 ID 'auth0|123456', got %q", identity.Auth0UserID)
	}
}

func TestManager_UpsertFromClaims_EmptyEmail(t *testing.T) {
	ds := NewDatastore(nil) // nil db is fine, we won't hit it
	mgr := NewManager(ds)

	claims := &jwtauth.Claims{
		Email: "",
	}
	claims.Subject = "auth0|123456"

	_, err := mgr.UpsertFromClaims(context.Background(), claims)
	if err != ErrInvalidEmail {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestManager_UpsertFromClaims_EmptyAuth0ID(t *testing.T) {
	ds := NewDatastore(nil)
	mgr := NewManager(ds)

	claims := &jwtauth.Claims{
		Email: "test@example.com",
	}
	// Subject is empty

	_, err := mgr.UpsertFromClaims(context.Background(), claims)
	if err != ErrInvalidAuth0ID {
		t.Errorf("expected ErrInvalidAuth0ID, got %v", err)
	}
}

func TestManager_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	mgr := NewManager(ds)

	id := uuid.New()
	mock.ExpectQuery(`SELECT .+ FROM user_identities WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{}))

	_, err = mgr.GetByID(context.Background(), id)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestManager_AddToOrg(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	mgr := NewManager(ds)
	ctx := context.Background()

	orgID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	// Check membership - not a member
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(orgID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Add member
	mock.ExpectQuery(`INSERT INTO org_members`).
		WithArgs(sqlmock.AnyArg(), orgID, userID, RoleMember, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(now))

	member, err := mgr.AddToOrg(ctx, orgID, userID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if member.Role != RoleMember {
		t.Errorf("expected role '%s', got %q", RoleMember, member.Role)
	}
}

func TestManager_AddToOrg_AlreadyMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	mgr := NewManager(ds)

	orgID := uuid.New()
	userID := uuid.New()

	// Already a member
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(orgID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	_, err = mgr.AddToOrg(context.Background(), orgID, userID, "")
	if err != ErrAlreadyMember {
		t.Errorf("expected ErrAlreadyMember, got %v", err)
	}
}

func TestManager_RemoveFromOrg_Owner(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	mgr := NewManager(ds)

	orgID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	// Get membership - is owner
	mock.ExpectQuery(`SELECT .+ FROM org_members WHERE org_id = \$1 AND user_id = \$2`).
		WithArgs(orgID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "org_id", "user_id", "role", "created_at"}).
			AddRow(uuid.New(), orgID, userID, RoleOwner, now))

	err = mgr.RemoveFromOrg(context.Background(), orgID, userID)
	if err != ErrCannotRemoveOwner {
		t.Errorf("expected ErrCannotRemoveOwner, got %v", err)
	}
}

func TestManager_CanAccessOrg_Admin(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	mgr := NewManager(ds)

	userID := uuid.New()
	orgID := uuid.New()
	now := time.Now()

	// Get user - is admin
	mock.ExpectQuery(`SELECT .+ FROM user_identities WHERE id = \$1`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "auth0_user_id", "email", "name", "is_admin", "created_at", "updated_at"}).
			AddRow(userID, "auth0|admin", "admin@example.com", "Admin", true, now, now))

	canAccess, err := mgr.CanAccessOrg(context.Background(), userID, orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !canAccess {
		t.Error("expected admin to have access")
	}
}

func TestManager_CanAccessOrg_Member(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	mgr := NewManager(ds)

	userID := uuid.New()
	orgID := uuid.New()
	now := time.Now()

	// Get user - not admin
	mock.ExpectQuery(`SELECT .+ FROM user_identities WHERE id = \$1`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "auth0_user_id", "email", "name", "is_admin", "created_at", "updated_at"}).
			AddRow(userID, "auth0|user", "user@example.com", "User", false, now, now))

	// Check membership - is member
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(orgID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	canAccess, err := mgr.CanAccessOrg(context.Background(), userID, orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !canAccess {
		t.Error("expected member to have access")
	}
}

func TestManager_CanAccessOrg_NotMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ds := NewDatastore(db)
	mgr := NewManager(ds)

	userID := uuid.New()
	orgID := uuid.New()
	now := time.Now()

	// Get user - not admin
	mock.ExpectQuery(`SELECT .+ FROM user_identities WHERE id = \$1`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "auth0_user_id", "email", "name", "is_admin", "created_at", "updated_at"}).
			AddRow(userID, "auth0|user", "user@example.com", "User", false, now, now))

	// Check membership - not member
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(orgID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	canAccess, err := mgr.CanAccessOrg(context.Background(), userID, orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if canAccess {
		t.Error("expected non-member to not have access")
	}
}
