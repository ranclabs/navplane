# NavPlane Development Guidelines

## Project Overview

NavPlane is a high-performance AI gateway and control plane for governed LLM traffic. It acts as a passthrough proxy for various AI providers, tracking usage and enabling policy-based routing.

## Architecture

```text
navplane/
├── backend/          # Go API server (net/http, no framework)
│   ├── cmd/server/   # Entry point
│   ├── internal/
│   │   ├── auth/     # Authentication helpers
│   │   ├── config/   # Environment-based configuration
│   │   ├── database/ # PostgreSQL connection and migrations
│   │   ├── handler/  # HTTP handlers
│   │   └── openai/   # OpenAI-compatible types
│   └── migrations/   # SQL migration files
├── dashboard/        # React + Vite SPA
└── docker-compose.yml
```

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
| `PROVIDER_BASE_URL` | Upstream AI provider URL |
| `PROVIDER_API_KEY` | Provider API key |
| `DATABASE_URL` | PostgreSQL connection string |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | HTTP server port |
| `ENV` | development | Environment name |
| `DB_MAX_OPEN_CONNS` | 25 | Max open DB connections |
| `DB_MAX_IDLE_CONNS` | 5 | Max idle DB connections |

## Database

- PostgreSQL 16+
- Migrations run automatically on startup
- Migration files in `backend/migrations/`
- Naming: `NNNNNN_description.up.sql` and `NNNNNN_description.down.sql`

## Testing

- Unit tests required for all new code
- Integration tests for handlers
- Use `t.Setenv()` for environment-dependent tests
- Mock external dependencies

## Key Design Decisions

1. **Passthrough Proxy**: Requests forwarded as-is to preserve provider compatibility
2. **Streaming Support**: SSE passthrough with `http.Flusher`
3. **Fail-Fast Config**: Missing required config fails at startup, not runtime
4. **Auto-Migrations**: Database schema applied on server start
5. **OpenAI-Compatible**: API mimics OpenAI format for drop-in replacement
