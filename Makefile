.PHONY: dev dev-backend dev-dashboard build build-backend build-dashboard clean install \
        docker docker-build docker-up docker-down docker-logs \
        db-up db-down migrate migrate-up migrate-down migrate-create migrate-status

# Database URL for local development (port 5434 to avoid conflicts with local postgres)
DATABASE_URL ?= postgres://navplane:navplane@localhost:5434/navplane?sslmode=disable

# Development
dev-backend:
	cd backend && go run ./cmd/server

dev-dashboard:
	cd dashboard && npm run dev

# Install dependencies
install:
	cd dashboard && npm install

# Build
build: build-backend build-dashboard

build-backend:
	cd backend && go build -o bin/navplane ./cmd/server

build-dashboard:
	cd dashboard && npm run build

# Docker
docker: docker-build docker-up

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-clean:
	docker compose down -v --rmi local

# Database
db-up:
	docker compose up -d postgres
	@echo "Waiting for postgres to be ready..."
	@sleep 2
	@echo "Postgres is running on localhost:5434"

db-down:
	docker compose stop postgres

# Migrations (requires golang-migrate CLI)
# Install: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
MIGRATE := $(shell which migrate 2>/dev/null || echo $(HOME)/go/bin/migrate)

migrate: migrate-up

migrate-up:
	$(MIGRATE) -path backend/migrations -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path backend/migrations -database "$(DATABASE_URL)" down 1

migrate-down-all:
	$(MIGRATE) -path backend/migrations -database "$(DATABASE_URL)" down -all

migrate-status:
	$(MIGRATE) -path backend/migrations -database "$(DATABASE_URL)" version

migrate-create:
	@read -p "Migration name: " name; \
	$(MIGRATE) create -ext sql -dir backend/migrations -seq $$name

# Clean
clean:
	rm -rf backend/bin
	rm -rf dashboard/dist
	rm -rf dashboard/node_modules

# Lint
lint:
	cd dashboard && npm run lint

# Test
test:
	cd backend && go test ./...

test-verbose:
	cd backend && go test -v ./...

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Development:"
	@echo "  dev-backend    - Run Go backend in development mode"
	@echo "  dev-dashboard  - Run Vite dev server"
	@echo "  install        - Install dashboard dependencies"
	@echo ""
	@echo "Build:"
	@echo "  build          - Build both backend and dashboard"
	@echo "  build-backend  - Build Go binary"
	@echo "  build-dashboard - Build dashboard for production"
	@echo ""
	@echo "Docker:"
	@echo "  docker         - Build and start all containers"
	@echo "  docker-build   - Build Docker images"
	@echo "  docker-up      - Start containers in background"
	@echo "  docker-down    - Stop containers"
	@echo "  docker-logs    - Follow container logs"
	@echo "  docker-clean   - Remove containers, volumes, and images"
	@echo ""
	@echo "Database:"
	@echo "  db-up          - Start postgres container"
	@echo "  db-down        - Stop postgres container"
	@echo "  migrate        - Run all pending migrations (alias for migrate-up)"
	@echo "  migrate-up     - Run all pending migrations"
	@echo "  migrate-down   - Rollback last migration"
	@echo "  migrate-down-all - Rollback all migrations"
	@echo "  migrate-status - Show current migration version"
	@echo "  migrate-create - Create a new migration file"
	@echo ""
	@echo "Testing:"
	@echo "  test           - Run all backend tests"
	@echo "  test-verbose   - Run all backend tests with verbose output"
	@echo ""
	@echo "Other:"
	@echo "  clean          - Remove build artifacts"
	@echo "  lint           - Run dashboard linter"
