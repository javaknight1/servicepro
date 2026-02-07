.PHONY: help lint lint-fix lint-check test format format-check dev up down migrate seed
.PHONY: test-unit test-integration test-e2e test-all
.PHONY: coverage ci ci-local ci-lint ci-backend ci-frontend
.PHONY: docker-dev docker-down docker-clean docker-logs docker-ps
.PHONY: db-fresh db-fresh-seed db-reset hash-password
.PHONY: swagger generate-api docs
.PHONY: dead-code dead-code-update frontend-bundle-check

# =============================================================================
# Centralized Tool Versions (used by CI and pre-commit)
# =============================================================================
GOLANGCI_LINT_VERSION ?= v2.1.6
PRETTIER_VERSION ?= 3.1.0
NODE_VERSION ?= 20
GO_VERSION ?= 1.21

help:
	@echo "Available targets:"
	@echo ""
	@echo "Development (without Docker):"
	@echo "  dev          - Instructions for local dev setup"
	@echo "  dev-db       - Start only database services (PostgreSQL, Redis)"
	@echo "  dev-backend  - Start backend server (requires dev-db)"
	@echo "  dev-frontend - Start frontend dev server"
	@echo "  migrate      - Run database migrations"
	@echo "  seed         - Load dev test data (user, orgs, customers, jobs, quotes)"
	@echo "  db-reset     - Reset database (WARNING: deletes all data)"
	@echo "  db-fresh-seed - Reset database AND load test data (recommended for dev)"
	@echo ""
	@echo "Setup:"
	@echo "  setup        - Complete setup (install deps, start db)"
	@echo "  install-deps - Install all dependencies"
	@echo ""
	@echo "Quality:"
	@echo "  lint            - Run all linters"
	@echo "  lint-check      - Run all linters in check-only mode (for CI)"
	@echo "  lint-fix        - Run linters with auto-fix"
	@echo "  format          - Format all code"
	@echo "  format-check    - Check formatting without modifying (for CI)"
	@echo "  dead-code       - Check for unused exports (dead code)"
	@echo "  dead-code-update - Update dead code baseline"
	@echo "  test            - Run all tests"
	@echo ""
	@echo "Testing:"
	@echo "  test-unit        - Run unit tests only"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e         - Run E2E tests (Cypress)"
	@echo "  test-all         - Run all test suites"
	@echo "  coverage         - Generate coverage reports"
	@echo ""
	@echo "CI/CD:"
	@echo "  ci-local     - Run exactly what CI runs (lint + backend + frontend)"
	@echo "  ci-lint      - Run lint checks only (matches CI)"
	@echo "  ci-backend   - Run backend tests + build (matches CI)"
	@echo "  ci-frontend  - Run frontend tests + build (matches CI)"
	@echo "  ci           - Run full CI pipeline (legacy)"
	@echo "  ci-quick     - Run quick CI (lint + unit tests)"
	@echo ""
	@echo "Docker:"
	@echo "  docker-dev   - Start dev environment (hot reload)"
	@echo "  docker-down  - Stop all containers"
	@echo "  docker-clean - Remove all Docker data and images"
	@echo "  docker-logs  - Follow logs from all containers"
	@echo "  docker-ps    - Show running containers"
	@echo ""
	@echo "API Documentation:"
	@echo "  swagger      - Generate Swagger docs from Go annotations"
	@echo "  generate-api - Generate TypeScript types from Swagger"
	@echo "  docs         - Run full API documentation pipeline"
	@echo ""
	@echo "Specific:"
	@echo "  frontend-*   - Frontend-specific targets"
	@echo "  backend-*    - Backend-specific targets"

# =============================================================================
# Linting (centralized rules - used by pre-commit and CI)
# =============================================================================

# Run all linters (allows auto-fix where applicable)
lint: frontend-lint backend-lint

# Run all linters with auto-fix
lint-fix: frontend-lint-fix backend-lint-fix format

# Run all linters in check-only mode (for CI - no modifications)
lint-check: frontend-lint-check backend-lint-check format-check
	@echo "✓ All lint checks passed"

# =============================================================================
# Formatting (centralized rules - used by pre-commit and CI)
# =============================================================================

# Format all files
format:
	@echo "Formatting all files with Prettier..."
	@npx prettier --write .

# Check formatting without modifying (for CI)
format-check:
	@echo "Checking formatting with Prettier..."
	@npx prettier --check .

# Combined testing
test: frontend-test backend-test

# Frontend targets
frontend-lint:
	@echo "Running ESLint on frontend..."
	@if [ -n "$$(find frontend/src -name '*.js' -o -name '*.jsx' -o -name '*.ts' -o -name '*.tsx' 2>/dev/null)" ]; then \
		cd frontend && ESLINT_USE_FLAT_CONFIG=true npx eslint . --max-warnings 0; \
	else \
		echo "No JavaScript/TypeScript files found in frontend/src"; \
	fi

frontend-lint-check: frontend-lint

frontend-lint-fix:
	@echo "Running ESLint with auto-fix on frontend..."
	@if [ -n "$$(find frontend/src -name '*.js' -o -name '*.jsx' -o -name '*.ts' -o -name '*.tsx' 2>/dev/null)" ]; then \
		cd frontend && ESLINT_USE_FLAT_CONFIG=true npx eslint . --fix --max-warnings 0; \
	else \
		echo "No JavaScript/TypeScript files found in frontend/src"; \
	fi

frontend-format:
	@echo "Formatting frontend code..."
	@npx prettier --write "frontend/**/*.{js,jsx,ts,tsx,json,css,scss,md}"

frontend-test:
	@echo "Running frontend tests..."
	@cd frontend && npm test -- --run --passWithNoTests

frontend-build:
	@echo "Building frontend..."
	@cd frontend && npm run build

frontend-bundle-check:
	@echo "Checking bundle size..."
	@cd frontend && npm run analyze:bundle -- --ci

frontend-dead-code:
	@echo "Checking for dead code in frontend..."
	@cd frontend && npm run dead-code

frontend-dead-code-update:
	@echo "Updating dead code baseline..."
	@cd frontend && npm run dead-code:update

# Dead code detection (alias)
dead-code: frontend-dead-code

dead-code-update: frontend-dead-code-update

# Backend targets
backend-lint:
	@echo "Running golangci-lint $(GOLANGCI_LINT_VERSION) on backend..."
	@if [ -n "$$(find backend -name '*.go' 2>/dev/null)" ]; then \
		cd backend && golangci-lint run --config .golangci.yml; \
	else \
		echo "No Go files found in backend"; \
	fi

backend-lint-check: backend-lint

backend-lint-fix:
	@echo "Running go fmt and go imports on backend..."
	@cd backend && gofmt -w . && goimports -w -local github.com/javaknight1/servicepro .

backend-format:
	@echo "Formatting backend code..."
	@cd backend && gofmt -w .

backend-test:
	@echo "Running backend tests..."
	@cd backend && go test -v -race -timeout 5m ./... -skip "Integration"

backend-build:
	@echo "Building backend..."
	@cd backend && go build -o /dev/null ./cmd

# Install dependencies
install-deps: install-golangci-lint
	@echo "Installing frontend dependencies..."
	@cd frontend && npm install
	@echo "Installing backend dependencies..."
	@cd backend && go mod download

# Install golangci-lint at centralized version
install-golangci-lint:
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

# Development targets
dev: dev-db
	@echo "Starting backend and frontend..."
	@echo "Backend will run on http://localhost:8080"
	@echo "Frontend will run on http://localhost:3000"
	@echo ""
	@echo "Run in separate terminals:"
	@echo "  Terminal 1: make dev-backend"
	@echo "  Terminal 2: make dev-frontend"

dev-db:
	@echo "Starting PostgreSQL, Redis, and MinIO..."
	@docker compose up -d postgres redis minio
	@echo "Waiting for services to be ready..."
	@sleep 3
	@docker compose ps

dev-backend:
	@echo "Starting backend server..."
	@cd backend && go run cmd/main.go

dev-frontend:
	@echo "Starting frontend dev server..."
	@cd frontend && npm run dev

# Legacy alias for docker-dev
up: docker-dev

down: docker-down

migrate:
	@echo "Running database migrations..."
	@cat backend/migrations/001_schema.sql | docker exec -i servicepro-postgres psql -U postgres -d servicepro
	@echo "Migrations complete!"

seed:
	@echo "Running development seed data..."
	@cat backend/migrations/002_seed_dev.sql | docker exec -i servicepro-postgres psql -U postgres -d servicepro
	@echo ""
	@echo "Seed data loaded!"
	@echo "  Login: dev@servicepro.local"
	@echo "  Password: password123"

db-reset:
	@echo "WARNING: This will delete all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		make db-fresh; \
	else \
		echo "Cancelled."; \
	fi

# Wait for postgres to be ready (used by other targets)
db-wait:
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker exec servicepro-postgres pg_isready -U postgres -d servicepro > /dev/null 2>&1; do \
		echo "  PostgreSQL not ready, waiting..."; \
		sleep 2; \
	done
	@echo "PostgreSQL is ready!"

# Quick database fresh start (no confirmation - use for dev)
db-fresh:
	@echo "Resetting database..."
	@docker compose down -v 2>/dev/null || true
	@docker compose up -d postgres redis minio mailpit
	@make db-wait
	@make migrate
	@echo ""
	@echo "Database reset complete!"
	@echo "Run 'make seed' to add dev test data"

# Fresh database with seed data (one command for dev)
db-fresh-seed: db-fresh seed
	@echo ""
	@echo "============================================"
	@echo "Database ready with test data!"
	@echo ""
	@echo "Test credentials:"
	@echo "  Email:    dev@servicepro.local"
	@echo "  Password: password123"
	@echo ""
	@echo "Next: Run 'docker compose up' to start all services"
	@echo "============================================"

# Generate bcrypt hash for a password (useful for updating seed data)
# Usage: make hash-password or make hash-password PASSWORD=mysecretpass
hash-password:
	@cd backend && go run ../scripts/hash-password.go $(PASSWORD)

setup: install-deps dev-db migrate seed
	@echo ""
	@echo "✅ Setup complete!"
	@echo ""
	@echo "Test credentials:"
	@echo "  Email:    dev@servicepro.local"
	@echo "  Password: password123"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Start backend:  make dev-backend"
	@echo "  2. Start frontend: make dev-frontend"
	@echo "  3. Open http://localhost:3000"

# =============================================================================
# Docker Environments
# =============================================================================

# Development environment (hot reload, volume mounts)
docker-dev:
	@echo "Starting development environment..."
	@docker compose up
	@echo ""
	@echo "Services:"
	@echo "  Frontend: http://localhost:3000"
	@echo "  Backend:  http://localhost:8080"

# Development environment (detached)
docker-dev-d:
	@echo "Starting development environment (detached)..."
	@docker compose up -d
	@echo ""
	@echo "Services started!"
	@echo "  Frontend: http://localhost:3000"
	@echo "  Backend:  http://localhost:8080"
	@echo ""
	@echo "View logs: make docker-logs"
	@docker compose ps

# Stop all Docker environments
docker-down:
	@echo "Stopping Docker containers..."
	@docker compose down 2>/dev/null || true
	@echo "All containers stopped."

# Clean all Docker data
docker-clean:
	@echo "Cleaning all Docker data..."
	@docker compose down -v --rmi local 2>/dev/null || true
	@docker system prune -f
	@echo "Docker cleanup complete."

# Docker logs
docker-logs:
	@docker compose logs -f

docker-logs-backend:
	@docker compose logs -f backend

docker-logs-frontend:
	@docker compose logs -f frontend

docker-logs-db:
	@docker compose logs -f postgres

# Docker status
docker-ps:
	@docker compose ps

# Shell into containers
docker-shell-backend:
	@docker exec -it servicepro-backend sh

docker-shell-frontend:
	@docker exec -it servicepro-frontend sh

docker-shell-db:
	@docker exec -it servicepro-postgres psql -U postgres -d servicepro

# Rebuild specific services
docker-rebuild-backend:
	@docker compose up -d --build backend

docker-rebuild-frontend:
	@docker compose up -d --build frontend

# Legacy aliases
logs: docker-logs
logs-backend: docker-logs-backend
logs-frontend: docker-logs-frontend
logs-db: docker-logs-db
ps: docker-ps

# =============================================================================
# Test Automation
# =============================================================================

# Unit tests only
test-unit:
	@echo "Running unit tests..."
	@./scripts/run-tests.sh -t unit

# Integration tests
test-integration:
	@echo "Running integration tests..."
	@./scripts/run-tests.sh -t integration

# E2E tests
test-e2e:
	@echo "Running E2E tests..."
	@./scripts/run-tests.sh -t e2e

# E2E tests in headed mode (for debugging)
test-e2e-headed:
	@echo "Opening Cypress..."
	@cd frontend && npx cypress open

# All tests
test-all:
	@echo "Running all tests..."
	@./scripts/run-tests.sh --ci -p

# Generate coverage reports
coverage:
	@echo "Generating coverage reports..."
	@./scripts/run-tests.sh -t unit -c
	@echo "Coverage reports generated"

coverage-view:
	@echo "Opening coverage reports..."
	@open backend/coverage.html || xdg-open backend/coverage.html 2>/dev/null || true
	@open frontend/coverage/index.html || xdg-open frontend/coverage/index.html 2>/dev/null || true

# =============================================================================
# CI/CD
# =============================================================================

# Run exactly what CI runs (matches checks.yml)
ci-local: ci-lint ci-backend ci-frontend
	@echo ""
	@echo "✅ All CI checks passed locally!"

# Lint check (matches CI lint job)
ci-lint:
	@echo "═══════════════════════════════════════════════════════════════"
	@echo "Running lint checks (matches CI)..."
	@echo "═══════════════════════════════════════════════════════════════"
	@pre-commit run --all-files --show-diff-on-failure

# Backend checks (matches CI backend-tests job)
ci-backend:
	@echo ""
	@echo "═══════════════════════════════════════════════════════════════"
	@echo "Running backend tests (matches CI)..."
	@echo "═══════════════════════════════════════════════════════════════"
	@cd backend && go test -v -race -timeout 5m ./... -skip "Integration"
	@echo "Verifying backend build..."
	@cd backend && go build -o /dev/null ./cmd
	@echo "✓ Backend checks passed"

# Frontend checks (matches CI frontend-tests job)
ci-frontend:
	@echo ""
	@echo "═══════════════════════════════════════════════════════════════"
	@echo "Running frontend tests (matches CI)..."
	@echo "═══════════════════════════════════════════════════════════════"
	@cd frontend && npm test -- --run --passWithNoTests
	@echo "Verifying frontend build..."
	@cd frontend && npm run build
	@echo "Checking bundle size..."
	@cd frontend && npm run analyze:bundle -- --ci
	@echo "✓ Frontend checks passed"

# Full CI pipeline (legacy - uses scripts)
ci:
	@echo "Running CI pipeline..."
	@./scripts/run-tests.sh --ci -p

# Quick CI (for pre-commit)
ci-quick:
	@echo "Running quick CI..."
	@./scripts/run-tests.sh -t unit

# =============================================================================
# Clean
# =============================================================================

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf backend/bin
	@rm -rf backend/*.out
	@rm -rf frontend/dist
	@rm -rf frontend/coverage
	@rm -rf frontend/cypress/screenshots
	@rm -rf frontend/cypress/videos
	@rm -rf frontend/cypress/reports
	@rm -rf coverage
	@echo "Clean complete"

clean-docker: docker-clean

# =============================================================================
# API Documentation
# =============================================================================

# Generate Swagger documentation from Go annotations
swagger:
	@echo "Generating Swagger documentation..."
	@cd backend && swag init -g cmd/main.go -o docs --parseDependency --parseInternal
	@echo "✓ Swagger docs generated in backend/docs/"

# Generate TypeScript API types from Swagger (requires swagger docs to exist)
generate-api:
	@echo "Generating TypeScript API types..."
	@cd frontend && npm run generate:api
	@echo "✓ TypeScript API types generated in frontend/src/types/api.generated.ts"

# Generate all API documentation and types
docs: swagger generate-api
	@echo ""
	@echo "✓ API documentation pipeline complete!"
	@echo "  - Swagger UI: http://localhost:8080/api/docs (when backend running)"
	@echo "  - TypeScript types: frontend/src/types/api.generated.ts"
