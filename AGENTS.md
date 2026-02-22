# NavPlane Development Guidelines

## Project Overview

NavPlane is a high-performance AI gateway and control plane for governed LLM traffic. It acts as a passthrough proxy for various AI providers, tracking usage and enabling policy-based routing.

## Architecture

```text
navplane/
├── backend/          # Go API server (net/http, no framework)
│   ├── cmd/server/   # Entry point
│   ├── internal/
│   │   ├── auth/       # Authentication helpers
│   │   ├── config/     # Environment-based configuration
│   │   ├── database/   # PostgreSQL connection and migrations
│   │   ├── handler/    # HTTP handlers
│   │   ├── middleware/ # HTTP middleware (auth, logging, etc.)
│   │   ├── openai/     # OpenAI-compatible types
│   │   └── org/        # Organization domain (manager/datastore pattern)
│   └── migrations/   # SQL migration files
├── dashboard/        # React + Vite SPA
└── docker-compose.yml
```

## Manager/Datastore Pattern

Domain resources follow a consistent manager/datastore pattern:

```text
internal/<resource>/
├── model.go      # Domain object (pure data struct)
├── datastore.go  # Persistence layer (DB CRUD operations)
├── manager.go    # Business logic, coordinates operations
└── *_test.go     # Tests (datastore_test.go, manager_test.go, model_test.go)
```

### Layer Responsibilities

| Layer | Responsibility |
|-------|----------------|
| **Model** | Pure data structures, no dependencies. Defines the domain object. May include helper functions (e.g., `GenerateAPIKey()`, `HashAPIKey()`). |
| **Datastore** | Database operations ONLY. Returns raw database errors. No business logic, no domain error translation. |
| **Manager** | Business logic, validation, coordination. Defines domain errors. Translates raw DB errors to domain errors. Handlers call managers. |

### Datastore Contract (CRITICAL)

The datastore layer has a strict contract:

1. **Return raw database errors** - Never wrap or translate errors. Return `sql.ErrNoRows` directly when a row is not found.
2. **Return `(rowsAffected int64, error)` for mutations** - For UPDATE/DELETE operations, return the number of rows affected. Let the manager interpret `rowsAffected == 0` as "not found".
3. **No domain errors** - Never define or return domain-specific errors like `ErrNotFound`. That's the manager's job.
4. **Stateless** - Receive `*sql.DB` or `DBTX` interface, return results. No caching or state.

```go
// CORRECT datastore method
func (ds *Datastore) Delete(ctx context.Context, id uuid.UUID) (int64, error) {
    result, err := ds.db.ExecContext(ctx, `DELETE FROM resources WHERE id = $1`, id)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}

// WRONG - datastore should NOT translate errors
func (ds *Datastore) Delete(ctx context.Context, id uuid.UUID) error {
    result, _ := ds.db.ExecContext(ctx, ...)
    if rowsAffected == 0 {
        return ErrNotFound  // WRONG! This is business logic
    }
    return nil
}
```

### Manager Contract (CRITICAL)

The manager layer has complementary responsibilities:

1. **Define domain errors** - `ErrNotFound`, `ErrInvalidName`, `ErrOrgDisabled`, etc.
2. **Validate inputs** - Check for empty names, invalid formats BEFORE calling datastore.
3. **Translate raw errors** - Convert `sql.ErrNoRows` to `ErrNotFound`.
4. **Interpret rowsAffected** - When datastore returns `rowsAffected == 0`, return `ErrNotFound`.
5. **Coordinate** - Can call multiple datastores if needed.

```go
// Domain errors defined in manager
var (
    ErrNotFound    = errors.New("org: not found")
    ErrInvalidName = errors.New("org: invalid name")
    ErrInvalidKey  = errors.New("org: invalid api key format")
    ErrOrgDisabled = errors.New("org: organization is disabled")
)

// Manager translates datastore results
func (m *Manager) Delete(ctx context.Context, id uuid.UUID) error {
    rowsAffected, err := m.ds.Delete(ctx, id)
    if err != nil {
        return err
    }
    if rowsAffected == 0 {
        return ErrNotFound  // Manager interprets this
    }
    return nil
}
```

### Example Flow

```text
Handler → Manager → Datastore → Database
                 ↘ Other services (if needed)
```

### Guidelines

- Handlers should only call managers, never datastores directly
- Managers can call multiple datastores if coordinating across resources
- Datastores are stateless - receive DB connection, return results
- Models have no methods beyond basic validation helpers
- Each resource lives in its own package under `internal/`

## Development Commands

### Backend
```bash
cd backend
go test ./...              # Run all tests
go test -race ./...        # Run tests with race detector
go build ./cmd/server      # Build binary
go run ./cmd/server        # Run locally
```

### Docker
```bash
docker compose up -d       # Start all services
docker compose logs -f     # Follow logs
docker compose down        # Stop services
```

## Code Style

### Go
- Use standard library where possible (no web frameworks)
- Error messages should be lowercase, no trailing punctuation
- Always handle errors explicitly
- Use table-driven tests
- Validate all config at startup (fail-fast)

### Commit Messages
Use semantic commits:
- `feat:` - New feature
- `fix:` - Bug fix
- `refactor:` - Code restructuring
- `test:` - Adding/updating tests
- `docs:` - Documentation
- `chore:` - Build, config, tooling

## Environment Variables

### Required

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `ENCRYPTION_KEY` | 32-byte base64-encoded key for encrypting provider API keys |
| `AUTH0_DOMAIN` | Auth0 tenant domain (e.g., `your-tenant.auth0.com`) |
| `AUTH0_AUDIENCE` | Auth0 API audience identifier |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | HTTP server port |
| `ENV` | development | Environment name |
| `DB_MAX_OPEN_CONNS` | 25 | Max open DB connections |
| `DB_MAX_IDLE_CONNS` | 5 | Max idle DB connections |
| `ENCRYPTION_KEY_NEW` | - | New encryption key for rotation (temporary) |

### Deprecated (Removed)

| Variable | Reason |
|----------|--------|
| `PROVIDER_BASE_URL` | Provider URLs are now hardcoded per provider |
| `PROVIDER_API_KEY` | API keys now come from org's BYOK storage |

## Database

### General
- PostgreSQL 16+
- Migrations run automatically on startup
- Migration files in `backend/migrations/`
- Naming: `NNNNNN_description.up.sql` and `NNNNNN_description.down.sql`

### PostgreSQL Patterns

#### Auto-updating `updated_at` Timestamps

PostgreSQL has NO `ON UPDATE CURRENT_TIMESTAMP` equivalent. `DEFAULT NOW()` only fires on INSERT.
You MUST use a trigger for `updated_at` to auto-update on modifications:

```sql
-- Create trigger function (once per database)
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to each table
CREATE TRIGGER trg_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
```

#### Migration File Paths

`golang-migrate` requires absolute paths for `file://` URLs. Always normalize paths:

```go
absPath, err := filepath.Abs(migrationsPath)
if err != nil {
    return err
}
sourceURL := "file://" + absPath
```

#### UUIDs as Primary Keys

Use `uuid_generate_v4()` for primary keys. Requires `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`.

### Docker Compose Security

- Do NOT expose PostgreSQL port to host (no `ports: "5432:5432"`)
- Backend connects via Docker internal network using service hostname

## Testing

### General Rules
- Unit tests required for all new code
- Integration tests for handlers
- Use `t.Setenv()` for environment-dependent tests
- Use table-driven tests for multiple scenarios
- Mock external dependencies

### Database Mocking with sqlmock

Use `github.com/DATA-DOG/go-sqlmock` for testing datastore and manager layers:

```go
func TestDatastore_GetByID(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create mock: %v", err)
    }
    defer db.Close()

    ds := NewDatastore(db)
    ctx := context.Background()
    id := uuid.New()

    // Expect the query and return mock rows
    rows := sqlmock.NewRows([]string{"id", "name", "enabled"}).
        AddRow(id, "Test Org", true)

    mock.ExpectQuery(`SELECT .+ FROM organizations WHERE id = \$1`).
        WithArgs(id).
        WillReturnRows(rows)

    org, err := ds.GetByID(ctx, id)
    // ... assertions ...

    // ALWAYS verify expectations were met
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("unfulfilled expectations: %v", err)
    }
}
```

### Testing Datastore vs Manager

| Test Target | What to Test |
|-------------|--------------|
| **Datastore** | SQL queries execute correctly, returns raw `sql.ErrNoRows`, returns correct `rowsAffected` |
| **Manager** | Validation logic, error translation (`sql.ErrNoRows` → `ErrNotFound`), business rules |

### Manager Tests Without DB

For pure validation tests, you can use a nil datastore:

```go
func TestManager_Create_InvalidName(t *testing.T) {
    m := &Manager{ds: nil}  // No DB needed for validation tests
    _, err := m.Create(context.Background(), "")
    if !errors.Is(err, ErrInvalidName) {
        t.Errorf("expected ErrInvalidName, got %v", err)
    }
}
```

## Authentication

### NavPlane API Keys

- Format: `np_<uuid>` (e.g., `np_550e8400-e29b-41d4-a716-446655440000`)
- Prefix `np_` allows easy identification and validation
- Keys are hashed with SHA-256 before storage (never store plaintext)
- Use Bearer token authentication: `Authorization: Bearer np_...`

### Key Generation and Hashing

```go
// Generate a new API key
func GenerateAPIKey() *APIKey {
    id := uuid.New().String()
    plaintext := APIKeyPrefix + id  // "np_" + uuid
    return &APIKey{
        Plaintext: plaintext,
        Hash:      HashAPIKey(plaintext),
    }
}

// Hash for storage (SHA-256, hex encoded)
func HashAPIKey(key string) string {
    hash := sha256.Sum256([]byte(key))
    return hex.EncodeToString(hash[:])
}
```

### Authentication Flow (API Proxy)

For `/v1/chat/completions` and other proxy endpoints:

1. Extract Bearer token from `Authorization` header
2. Validate key format (must start with `np_`)
3. Hash the key and lookup org by hash
4. Check org is enabled (kill switch)
5. Inject org into request context

### User Authentication (Dashboard)

For dashboard/admin endpoints, we use Auth0:

1. User logs in via Auth0 (handles passwords, MFA, social login)
2. Client receives JWT from Auth0
3. Client sends JWT to NavPlane API: `Authorization: Bearer <jwt>`
4. NavPlane verifies JWT signature using Auth0 JWKS
5. Extract `sub` (auth0_user_id), `email`, `name` from claims
6. Upsert user in `user_identities` table
7. Check org membership for authorization

### User Types

| Type | Description | Access |
|------|-------------|--------|
| **Regular User** | Normal org member | Only their orgs |
| **Admin User** | Internal NavPlane admin (`is_admin=true`) | All orgs + admin endpoints |

Admin users can also be members of regular orgs.

### Middleware Pattern

```go
func (m *Auth) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractBearerToken(r)
        org, err := m.orgManager.Authenticate(r.Context(), token)
        if err != nil {
            // Return appropriate 401/403
            return
        }
        ctx := context.WithValue(r.Context(), OrgContextKey, org)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Error Handling

### HTTP Error Responses

Use consistent error response format:

```go
type APIError struct {
    Error struct {
        Message string `json:"message"`
        Type    string `json:"type"`
        Code    string `json:"code,omitempty"`
    } `json:"error"`
}
```

### Defensive Error Handling

- NEVER ignore error return values (linter enforced)
- Log errors even if you can't recover
- Use helper functions for common patterns:

```go
// For defer Close() calls
func closeBody(body io.Closer) {
    if err := body.Close(); err != nil {
        log.Printf("error closing body: %v", err)
    }
}

// Usage
defer closeBody(resp.Body)
```

## BYOK (Bring Your Own Key)

Organizations store their own provider API keys (OpenAI, Anthropic, etc.) in NavPlane.

### Supported Providers

| Provider | Base URL | Models |
|----------|----------|--------|
| **OpenAI** | `https://api.openai.com/v1` | `gpt-4o`, `gpt-4o-mini`, `gpt-4-turbo`, `o1`, `o1-mini` |
| **Anthropic** | `https://api.anthropic.com/v1` | `claude-3-5-sonnet-*`, `claude-3-5-haiku-*`, `claude-3-opus-*` |

Provider base URLs are hardcoded - no configuration needed.

### Envelope Encryption

Provider API keys are encrypted using envelope encryption for secure storage and key rotation:

```
┌─────────────────────────────────────────────────────────┐
│  KEK (Key Encryption Key) - from ENCRYPTION_KEY env     │
└─────────────────────────────────────────────────────────┘
                        │ encrypts
                        ▼
┌─────────────────────────────────────────────────────────┐
│  DEK (Data Encryption Key) - random per provider key    │
│  Stored in: provider_keys.encrypted_dek                 │
└─────────────────────────────────────────────────────────┘
                        │ encrypts
                        ▼
┌─────────────────────────────────────────────────────────┐
│  Provider API Key (e.g., sk-...)                        │
│  Stored in: provider_keys.encrypted_key                 │
└─────────────────────────────────────────────────────────┘
```

### Key Rotation

To rotate the master encryption key (`ENCRYPTION_KEY`):

1. Set `ENCRYPTION_KEY_NEW` with the new key
2. Run: `navplane migrate-keys` (re-encrypts all DEKs)
3. Update `ENCRYPTION_KEY` to the new value
4. Remove `ENCRYPTION_KEY_NEW`

### Provider Key Validation

When adding a provider key, we validate it by making a test API call:
- OpenAI: `GET /v1/models`
- Anthropic: `GET /v1/models` (or minimal chat request)

Invalid keys are rejected before storage.

### Provider Interface

```go
type Provider interface {
    Name() string                    // "openai", "anthropic"
    DisplayName() string             // "OpenAI", "Anthropic"
    BaseURL() string                 // Hardcoded API URL
    Models() []Model                 // Supported models
    ValidateKey(ctx context.Context, key string) error
}
```

## Admin API

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/admin/orgs` | List all organizations |
| `POST` | `/admin/orgs` | Create organization (returns API key) |
| `GET` | `/admin/orgs/{id}` | Get organization by ID |
| `PUT` | `/admin/orgs/{id}` | Update organization name |
| `DELETE` | `/admin/orgs/{id}` | Delete organization |
| `PUT` | `/admin/orgs/{id}/enabled` | Enable/disable org (kill switch) |
| `POST` | `/admin/orgs/{id}/rotate-key` | Rotate API key |

### Kill Switch

The kill switch allows instant disabling of an organization:

```bash
# Disable an org
curl -X PUT http://localhost:8080/admin/orgs/{id}/enabled \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'

# Re-enable an org
curl -X PUT http://localhost:8080/admin/orgs/{id}/enabled \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'
```

When disabled:
- All requests with the org's API key return 403 immediately
- No upstream provider calls are made
- Re-enabling restores access instantly

### Response Format

Organization responses never expose the API key hash:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My Organization",
  "enabled": true,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

Create and rotate-key responses include the plaintext API key (only time it's available):

```json
{
  "id": "...",
  "name": "...",
  "api_key": "np_550e8400-e29b-41d4-a716-446655440000"
}
```

### Handler Pattern

Admin handlers follow this pattern:

```go
type AdminOrgsHandler struct {
    manager *org.Manager
}

func (h *AdminOrgsHandler) Get(w http.ResponseWriter, r *http.Request) {
    id, err := parseOrgID(r)  // Use r.PathValue("id")
    if err != nil {
        writeAdminError(w, http.StatusBadRequest, "invalid organization ID")
        return
    }

    o, err := h.manager.GetByID(r.Context(), id)
    if err != nil {
        if errors.Is(err, org.ErrNotFound) {
            writeAdminError(w, http.StatusNotFound, "organization not found")
            return
        }
        writeAdminError(w, http.StatusInternalServerError, "failed to get organization")
        return
    }

    writeJSON(w, http.StatusOK, toOrgResponse(o))
}
```

## Key Design Decisions

1. **Passthrough Proxy**: Requests forwarded as-is to preserve provider compatibility
2. **Streaming Support**: SSE passthrough with `http.Flusher`
3. **Fail-Fast Config**: Missing required config fails at startup, not runtime
4. **Auto-Migrations**: Database schema applied on server start
5. **OpenAI-Compatible**: API mimics OpenAI format for drop-in replacement
6. **Manager/Datastore Separation**: Clean separation of persistence and business logic
7. **Hashed API Keys**: Never store plaintext keys, always SHA-256 hash
8. **Instant Kill Switch**: Org disable takes effect immediately, no caching
