# NavPlane

AI gateway and control plane for governed LLM traffic. Passthrough proxy for OpenAI, Anthropic, etc. with usage tracking and policy-based routing.

## Quick Start

### Prerequisites

- Docker 24+ & Docker Compose v2+
- Go 1.24+ (for local dev)
- Auth0 account (for user auth)

### Environment Setup

```bash
cp .env.example .env
# Edit .env with your values
```

**Required variables:**

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `ENCRYPTION_KEY` | 32-byte base64 key (`openssl rand -base64 32`) |
| `AUTH0_DOMAIN` | Your Auth0 tenant (e.g., `acme.auth0.com`) |
| `AUTH0_AUDIENCE` | Auth0 API identifier |

### Run with Docker

```bash
docker compose up -d
# Backend: http://localhost:8080
# Dashboard: http://localhost:3000
```

### Run Locally

```bash
# Terminal 1 - Start Postgres
docker compose up -d postgres

# Terminal 2 - Backend
cd backend && go run ./cmd/server

# Terminal 3 - Dashboard (optional)
cd dashboard && npm install && npm run dev
```

## Development

```bash
cd backend
go test ./...           # Run tests
go test -race ./...     # With race detector
go build ./cmd/server   # Build binary
```

### Project Structure

```
backend/
├── cmd/server/       # Entry point
├── internal/
│   ├── config/       # Environment config
│   ├── database/     # PostgreSQL + migrations
│   ├── handler/      # HTTP handlers
│   ├── jwtauth/      # Auth0 JWT verification
│   ├── middleware/   # HTTP middleware
│   ├── org/          # Organizations
│   ├── orgsettings/  # Per-org provider settings
│   ├── provider/     # Provider interface (OpenAI, Anthropic)
│   ├── providerkey/  # BYOK encrypted storage
│   └── user/         # User identities
└── migrations/       # SQL migrations
```

## API Usage

### Proxy Endpoint

Replace your OpenAI base URL with NavPlane:

```python
# Before
client = OpenAI(api_key="sk-...")

# After
client = OpenAI(
    api_key="np_your-navplane-key",
    base_url="http://localhost:8080/v1"
)
```

### Admin Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /admin/orgs` | List organizations |
| `POST /admin/orgs` | Create org (returns API key) |
| `PUT /admin/orgs/{id}/enabled` | Kill switch |
| `POST /admin/orgs/{id}/rotate-key` | Rotate API key |
| `GET /admin/orgs/{id}/provider-keys` | List provider keys |
| `POST /admin/orgs/{id}/provider-keys` | Add provider key |

## Commit Convention

Use semantic commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`

## License

MIT
