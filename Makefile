.PHONY: dev dev-backend dev-dashboard build build-backend build-dashboard clean install

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
	@echo "  dev-backend    - Run Go backend in development mode"
	@echo "  dev-dashboard  - Run Vite dev server"
	@echo "  install        - Install dashboard dependencies"
	@echo "  build          - Build both backend and dashboard"
	@echo "  build-backend  - Build Go binary"
	@echo "  build-dashboard - Build dashboard for production"
	@echo "  clean          - Remove build artifacts"
	@echo "  lint           - Run dashboard linter"
