# NavPlane

NavPlane is a high-performance AI gateway and control plane for governed LLM traffic.

## Project Structure

This is a monorepo containing:

```
navplane/
├── backend/     # Go API server
├── dashboard/   # React + Vite SPA
└── README.md
```

## Prerequisites

- Go 1.22+
- Node.js 20+
- npm 10+

## Getting Started

### Backend

```bash
cd backend
go run main.go
```

The server starts on `http://localhost:8080`.

### Dashboard

```bash
cd dashboard
npm install
npm run dev
```

The dev server starts on `http://localhost:3000` with API proxy to the backend.

## Development

Run both services concurrently for full-stack development:

**Terminal 1 - Backend:**
```bash
cd backend && go run main.go
```

**Terminal 2 - Dashboard:**
```bash
cd dashboard && npm run dev
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/v1/status` | GET | Service status |

## Building for Production

### Backend

```bash
cd backend
go build -o bin/navplane main.go
```

### Dashboard

```bash
cd dashboard
npm run build
```

The built assets will be in `dashboard/dist/`.

## License

MIT
