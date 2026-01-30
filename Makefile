.PHONY: help lint lint-fix lint-check test format format-check dev up down migrate
.PHONY: test-unit test-integration test-e2e test-all
.PHONY: coverage ci ci-local ci-lint ci-backend ci-frontend
.PHONY: docker-dev docker-down docker-clean docker-logs docker-ps

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
	@echo "  db-reset     - Reset database (WARNING: deletes all data)"
	@echo ""
	@echo "Setup:"
	@echo "  setup        - Complete setup (install deps, start db)"
	@echo "  install-deps - Install all dependencies"
	@echo ""
	@echo "Quality:"
	@echo "  lint         - Run all linters"
	@echo "  lint-check   - Run all linters in check-only mode (for CI)"
	@echo "  lint-fix     - Run linters with auto-fix"
	@echo "  format       - Format all code"
	@echo "  format-check - Check formatting without modifying (for CI)"
	@echo "  test         - Run all tests"
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
	@echo "Deployment:"
	@echo "  deploy-backend   - Deploy backend to Fly.io"
	@echo "  deploy-frontend  - Deploy frontend to Cloudflare Pages"
	@echo "  deploy           - Deploy both backend and frontend"
	@echo "  logs-prod        - View production backend logs"
	@echo "  status-prod      - Check production backend status"
	@echo "  ssh-prod         - SSH into production backend"
	@echo "  secrets-list     - View production secrets (names only)"
	@echo "  migrate-prod     - Run migrations on production"
	@echo "  health-prod      - Health check production"
	@echo "  scale-backend    - View backend scale settings"
	@echo "  restart-prod     - Restart production backend"
	@echo ""
	@echo "Docker:"
	@echo "  docker-dev   - Start dev environment (hot reload)"
	@echo "  docker-down  - Stop all containers"
	@echo "  docker-clean - Remove all Docker data and images"
	@echo "  docker-logs  - Follow logs from all containers"
	@echo "  docker-ps    - Show running containers"
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
	@cat backend/migrations/*.sql | docker exec -i servicepro-postgres psql -U postgres -d servicepro
	@echo "Migrations complete!"

db-reset:
	@echo "WARNING: This will delete all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		make db-fresh; \
	else \
		echo "Cancelled."; \
	fi

# Quick database fresh start (no confirmation - use for dev)
db-fresh:
	@echo "Resetting database..."
	@docker compose down -v 2>/dev/null || true
	@docker compose up -d postgres redis minio
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@make migrate
	@echo "Database reset complete!"

setup: install-deps dev-db migrate
	@echo ""
	@echo "✅ Setup complete!"
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
# Deployment
# =============================================================================

# Deploy backend to Fly.io
deploy-backend:
	@echo "Deploying backend to Fly.io..."
	@cd backend && fly deploy --app servicepro-api
	@echo "✓ Backend deployed!"

# Deploy frontend to Cloudflare Pages
deploy-frontend:
	@echo "Building and deploying frontend to Cloudflare Pages..."
	@cd frontend && npm ci && npm run build
	@cd frontend && wrangler pages deploy dist --project-name=servicepro
	@echo "✓ Frontend deployed!"

# Deploy both
deploy: deploy-backend deploy-frontend
	@echo "✓ Full deployment complete!"

# View backend logs
logs-prod:
	@fly logs --app servicepro-api

# Check backend status
status-prod:
	@fly status --app servicepro-api

# SSH into production backend
ssh-prod:
	@fly ssh console --app servicepro-api

# View production secrets (names only)
secrets-list:
	@fly secrets list --app servicepro-api

# Run migrations on production
migrate-prod:
	@echo "Running migrations on production..."
	@psql $(DATABASE_URL) < backend/migrations/001_schema.sql
	@echo "✓ Migrations complete!"

# Health check production
health-prod:
	@curl -s https://servicepro-api.fly.dev/health | jq .

# Scale backend
scale-backend:
	@echo "Current scale:"
	@fly scale show --app servicepro-api

# Restart production backend
restart-prod:
	@echo "Restarting production backend..."
	@fly apps restart servicepro-api
	@echo "✓ Restarted!"

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
