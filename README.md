# NavPlane

NavPlane is a high-performance AI gateway and control plane for governed LLM traffic.

## Project Structure

```
navplane/
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
- `navplane-backend` - Scratch-based Go binary (~10MB)
- `navplane-dashboard` - Nginx serving static assets (~25MB)

### Without Docker

```bash
# Backend
cd backend && go build -o bin/navplane main.go

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
