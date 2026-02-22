# Lectr

Lectr is a high-performance AI gateway and control plane for governed LLM traffic.

## Project Structure

```
lectr/
├── backend/          # Go API server
├── dashboard/        # React + Vite SPA (Tailwind CSS)
├── docker-compose.yml
├── Makefile
└── README.md
```

## Prerequisites

**Local Development:**
- Go 1.22+
- Node.js 20+
- npm 10+

**Docker:**
- Docker 24+
- Docker Compose v2+

## Quick Start

### Using Docker (Recommended)

```bash
# Build and start all services
make docker

# Or step by step:
docker compose build
docker compose up -d
```

- **Dashboard**: http://localhost:3000
- **Backend API**: http://localhost:8080

```bash
# View logs
make docker-logs

# Stop services
make docker-down
```

### Local Development

**Terminal 1 - Backend:**
```bash
cd backend && go run main.go
```

**Terminal 2 - Dashboard:**
```bash
cd dashboard && npm install && npm run dev
```

- **Dashboard**: http://localhost:3000 (with API proxy)
- **Backend**: http://localhost:8080

## Configuration

The backend requires the following environment variables:

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `PROVIDER_BASE_URL` | Base URL for the upstream AI provider | `https://api.openai.com/v1` |
| `PROVIDER_API_KEY` | API key for the upstream provider | `sk-...` |

### Optional

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `ENV` | Environment (`development`, `staging`, `production`) | `development` |

**Note:** The provider configuration is temporary MVP setup. This will be replaced by BYOK vault and per-organization provider management in future releases.

**Example:**

```bash
export PROVIDER_BASE_URL="https://api.openai.com/v1"
export PROVIDER_API_KEY="sk-..."
cd backend && go run cmd/server/main.go
```

The service will fail fast on startup if required environment variables are missing.

### Testing in Production Environment

#### 1. Using Docker Compose (Recommended)

Create a `.env` file in the project root:

```bash
# .env
ENV=production
PROVIDER_BASE_URL=https://api.openai.com/v1
PROVIDER_API_KEY=sk-proj-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

Then start the services:

```bash
# Build and start with environment variables
docker compose up -d

# Verify backend started successfully
docker compose logs backend

# You should see: "Lectr server starting on :8080 (env: production)"
```

#### 2. Direct Binary Execution

```bash
# Build the binary
cd backend && go build -o lectr ./cmd/server

# Set environment variables and run
export ENV=production
export PROVIDER_BASE_URL="https://api.openai.com/v1"
export PROVIDER_API_KEY="sk-proj-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
./lectr
```

#### 3. Testing Fail-Fast Behavior

Verify the service fails immediately when configuration is missing:

```bash
# Test without any provider config
docker compose up backend
# Expected: "failed to load configuration: missing required environment variables: [PROVIDER_BASE_URL PROVIDER_API_KEY]"

# Test with only one variable
PROVIDER_BASE_URL="https://api.openai.com/v1" docker compose up backend
# Expected: "failed to load configuration: missing required environment variables: [PROVIDER_API_KEY]"
```

#### 4. Production Deployment Checklist

- [ ] Set `ENV=production`
- [ ] Set `PROVIDER_BASE_URL` to your AI provider endpoint
- [ ] Set `PROVIDER_API_KEY` securely (use secrets manager, not plain text)
- [ ] Verify service starts successfully
- [ ] Check logs show correct environment
- [ ] Never commit `.env` file with real credentials

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/v1/status` | GET | Service status |

## Building for Production

### With Docker

```bash
docker compose build
```

Images:
- `lectr-backend` - Scratch-based Go binary (~10MB)
- `lectr-dashboard` - Nginx serving static assets (~25MB)

### Without Docker

```bash
# Backend
cd backend && go build -o bin/lectr main.go

# Dashboard
cd dashboard && npm run build
```

## Make Targets

```bash
make help
```

| Target | Description |
|--------|-------------|
| `docker` | Build and start all containers |
| `docker-build` | Build Docker images |
| `docker-up` | Start containers |
| `docker-down` | Stop containers |
| `docker-logs` | Follow container logs |
| `dev-backend` | Run Go backend locally |
| `dev-dashboard` | Run Vite dev server |
| `build` | Build both services |
| `lint` | Run dashboard linter |

## License

MIT
