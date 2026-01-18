.PHONY: dev dev-backend dev-dashboard build build-backend build-dashboard clean install \
        docker docker-build docker-up docker-down docker-logs

# Development
dev-backend:
	cd backend && go run main.go

dev-dashboard:
	cd dashboard && npm run dev

# Install dependencies
install:
	cd dashboard && npm install

# Build
build: build-backend build-dashboard

build-backend:
	cd backend && go build -o bin/navplane main.go

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

# Clean
clean:
	rm -rf backend/bin
	rm -rf dashboard/dist
	rm -rf dashboard/node_modules

# Lint
lint:
	cd dashboard && npm run lint

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
	@echo "Other:"
	@echo "  clean          - Remove build artifacts"
	@echo "  lint           - Run dashboard linter"
