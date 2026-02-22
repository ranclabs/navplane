# NavPlane Code Rules

AI assistant guidelines for this codebase.

## Architecture

- **Go backend** using `net/http` (no frameworks)
- **PostgreSQL** with auto-migrations on startup
- **Passthrough proxy** - forward requests as-is, minimal parsing

## Manager/Datastore Pattern

Every domain resource follows this structure:

```
internal/<resource>/
├── model.go      # Pure data struct
├── datastore.go  # DB operations only
├── manager.go    # Business logic
└── *_test.go
```

### Datastore Rules (CRITICAL)

1. Return raw `sql.ErrNoRows` - never translate to domain errors
2. Return `(rowsAffected int64, error)` for mutations
3. No business logic - just execute SQL and return results

```go
// CORRECT
func (ds *Datastore) Delete(ctx context.Context, id uuid.UUID) (int64, error) {
    result, err := ds.db.ExecContext(ctx, `DELETE FROM x WHERE id = $1`, id)
    if err != nil { return 0, err }
    return result.RowsAffected()
}

// WRONG - don't translate errors in datastore
func (ds *Datastore) Delete(ctx context.Context, id uuid.UUID) error {
    if rowsAffected == 0 { return ErrNotFound }  // NO!
}
```

### Manager Rules

1. Define domain errors (`ErrNotFound`, `ErrInvalidName`, etc.)
2. Validate inputs before calling datastore
3. Translate `sql.ErrNoRows` → `ErrNotFound`
4. Interpret `rowsAffected == 0` as not found

```go
func (m *Manager) Delete(ctx context.Context, id uuid.UUID) error {
    rowsAffected, err := m.ds.Delete(ctx, id)
    if err != nil { return err }
    if rowsAffected == 0 { return ErrNotFound }
    return nil
}
```

## Code Style

- Error messages: lowercase, no trailing punctuation
- Always handle errors explicitly (linter enforced)
- Use table-driven tests
- Use `t.Setenv()` for env-dependent tests
- Use `sqlmock` for DB tests

## PostgreSQL

- Use triggers for `updated_at` (not `DEFAULT NOW()`)
- Use `filepath.Abs()` for migration paths
- Don't expose DB port in docker-compose

```sql
CREATE TRIGGER trg_updated_at BEFORE UPDATE ON table_name
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
```

## Authentication

### API Keys (Proxy)
- Format: `np_<uuid>`
- SHA-256 hashed before storage
- Lookup by hash, check `enabled` flag

### JWT (Dashboard)
- Auth0 handles user management
- NavPlane verifies JWT via JWKS
- Upsert user in `user_identities` on login

## Provider Keys (BYOK)

- Envelope encryption: KEK encrypts DEK, DEK encrypts key
- Validate key with provider API before storing
- Provider URLs hardcoded (OpenAI, Anthropic)

## Error Responses

```json
{"error": {"message": "...", "type": "..."}}
```

Types: `authentication_error`, `invalid_request_error`, `server_error`

## Key Files

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | Entry point, wires dependencies |
| `handler/routes.go` | All route registration |
| `handler/chat_completions.go` | Main proxy handler |
| `middleware/auth.go` | API key auth middleware |
| `org/model.go` | API key generation/hashing |
