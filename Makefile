.PHONY: help setup-hooks lint lint-fix lint-check test format format-check dev up down migrate
.PHONY: test-unit test-integration test-e2e test-performance test-all
.PHONY: coverage ci ci-local ci-lint ci-backend ci-frontend notify
.PHONY: k6-smoke k6-load k6-stress k6-ci
.PHONY: artillery-stress artillery-peak artillery-soak artillery-spike
.PHONY: bench bench-api bench-db bench-e2e bench-critical bench-report bench-schedule
.PHONY: docker-dev docker-prod docker-test docker-down docker-clean docker-logs docker-ps

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
	@echo "  setup        - Complete setup (install deps, setup hooks, start db)"
	@echo "  setup-hooks  - Install pre-commit hooks"
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
	@echo "  test-performance - Run performance tests (k6)"
	@echo "  test-all         - Run all test suites"
	@echo "  coverage         - Generate coverage reports"
	@echo ""
	@echo "k6 Load Testing:"
	@echo "  k6-smoke         - Run quick smoke test (5 VUs)"
	@echo "  k6-load          - Run normal load test (100 VUs)"
	@echo "  k6-stress        - Run stress test (400 VUs)"
	@echo "  k6-ci            - Run CI pipeline test with JSON output"
	@echo ""
	@echo "Artillery Stress Testing:"
	@echo "  artillery-stress - Run default stress test"
	@echo "  artillery-peak   - Run peak load test (max capacity)"
	@echo "  artillery-soak   - Run soak test (2+ hours)"
	@echo "  artillery-spike  - Run spike tests (sudden load changes)"
	@echo ""
	@echo "Benchmark Suite:"
	@echo "  bench            - Run full benchmark suite"
	@echo "  bench-api        - Run API benchmarks only"
	@echo "  bench-db         - Run database benchmarks only"
	@echo "  bench-e2e        - Run end-to-end benchmarks only"
	@echo "  bench-critical   - Run critical path benchmarks"
	@echo "  bench-report     - Generate benchmark report"
	@echo "  bench-schedule   - Start benchmark scheduler"
	@echo ""
	@echo "CI/CD:"
	@echo "  ci-local     - Run exactly what CI runs (lint + backend + frontend)"
	@echo "  ci-lint      - Run lint checks only (matches CI)"
	@echo "  ci-backend   - Run backend tests + build (matches CI)"
	@echo "  ci-frontend  - Run frontend tests + build (matches CI)"
	@echo "  ci           - Run full CI pipeline (legacy)"
	@echo "  ci-quick     - Run quick CI (lint + unit tests)"
	@echo "  notify       - Send test notifications"
	@echo ""
	@echo "Docker:"
	@echo "  docker-dev   - Start dev environment (hot reload)"
	@echo "  docker-prod  - Build & run production images locally"
	@echo "  docker-test  - Run integration tests in Docker"
	@echo "  docker-down  - Stop all Docker environments"
	@echo "  docker-clean - Remove all Docker data and images"
	@echo "  docker-logs  - Follow logs from all containers"
	@echo "  docker-ps    - Show running containers"
	@echo ""
	@echo "Specific:"
	@echo "  frontend-*   - Frontend-specific targets"
	@echo "  backend-*    - Backend-specific targets"

# Pre-commit setup
setup-hooks:
	@./scripts/setup-pre-commit.sh

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
		cd frontend && ESLINT_USE_FLAT_CONFIG=true npx eslint .; \
	else \
		echo "No JavaScript/TypeScript files found in frontend/src"; \
	fi

frontend-lint-check: frontend-lint

frontend-lint-fix:
	@echo "Running ESLint with auto-fix on frontend..."
	@if [ -n "$$(find frontend/src -name '*.js' -o -name '*.jsx' -o -name '*.ts' -o -name '*.tsx' 2>/dev/null)" ]; then \
		cd frontend && ESLINT_USE_FLAT_CONFIG=true npx eslint . --fix; \
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
	@echo "Installing pre-commit..."
	@./scripts/setup-pre-commit.sh

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
	@echo "Starting PostgreSQL and Redis..."
	@docker compose up -d postgres redis
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
	@docker compose up -d postgres redis
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@make migrate
	@echo "Database reset complete!"

setup: install-deps setup-hooks dev-db migrate
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

# Production test environment (built images, no hot reload)
docker-prod:
	@echo "Building and starting production environment..."
	@docker compose -f docker-compose.prod.yml up --build
	@echo ""
	@echo "Production build running at: http://localhost:3000"

# Production environment (detached)
docker-prod-d:
	@echo "Building and starting production environment (detached)..."
	@docker compose -f docker-compose.prod.yml up --build -d
	@echo ""
	@echo "Production build running at: http://localhost:3000"
	@docker compose -f docker-compose.prod.yml ps

# Integration test environment
docker-test:
	@echo "Running integration tests..."
	@docker compose -f docker-compose.test.yml run --rm test-runner

# Start test API for E2E tests
docker-test-api:
	@echo "Starting test API server..."
	@docker compose -f docker-compose.test.yml up -d test-api
	@echo "Test API running at: http://localhost:8081"

# Stop all Docker environments
docker-down:
	@echo "Stopping all Docker environments..."
	@docker compose down 2>/dev/null || true
	@docker compose -f docker-compose.prod.yml down 2>/dev/null || true
	@docker compose -f docker-compose.test.yml down 2>/dev/null || true
	@echo "All containers stopped."

# Clean all Docker data
docker-clean:
	@echo "Cleaning all Docker data..."
	@docker compose down -v --rmi local 2>/dev/null || true
	@docker compose -f docker-compose.prod.yml down -v --rmi local 2>/dev/null || true
	@docker compose -f docker-compose.test.yml down -v --rmi local 2>/dev/null || true
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
	@echo "Development containers:"
	@docker compose ps 2>/dev/null || echo "  (none running)"
	@echo ""
	@echo "Production containers:"
	@docker compose -f docker-compose.prod.yml ps 2>/dev/null || echo "  (none running)"
	@echo ""
	@echo "Test containers:"
	@docker compose -f docker-compose.test.yml ps 2>/dev/null || echo "  (none running)"

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

# Performance tests
test-performance:
	@echo "Running performance tests..."
	@./scripts/run-tests.sh -t performance

# k6 load tests (direct)
k6-smoke:
	@echo "Running k6 smoke tests..."
	@k6 run -e K6_PROFILE=smoke tests/performance/k6/main.js

k6-load:
	@echo "Running k6 load tests..."
	@k6 run -e K6_PROFILE=load tests/performance/k6/main.js

k6-stress:
	@echo "Running k6 stress tests..."
	@k6 run -e K6_PROFILE=stress tests/performance/k6/main.js

k6-ci:
	@echo "Running k6 CI tests..."
	@k6 run -e K6_PROFILE=ci --out json=k6-results.json tests/performance/k6/main.js

# Artillery stress tests
artillery-stress:
	@echo "Running Artillery stress tests..."
	@artillery run tests/performance/artillery/stress-config.yml

artillery-peak:
	@echo "Running Artillery peak load tests..."
	@artillery run tests/performance/artillery/scenarios/peak-load.yml

artillery-soak:
	@echo "Running Artillery soak tests (2+ hours)..."
	@artillery run tests/performance/artillery/scenarios/sustained-load.yml

artillery-spike:
	@echo "Running Artillery spike tests..."
	@artillery run tests/performance/artillery/scenarios/spike-tests.yml

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
	@echo "✓ Frontend checks passed"

# Full CI pipeline (legacy - uses scripts)
ci:
	@echo "Running CI pipeline..."
	@./scripts/run-tests.sh --ci -p

# Quick CI (for pre-commit)
ci-quick:
	@echo "Running quick CI..."
	@./scripts/run-tests.sh -t unit

# Send notifications
notify:
	@./scripts/notify.sh success "All tests passed"

notify-failure:
	@./scripts/notify.sh failure "Tests failed"

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
# Benchmark Suite
# =============================================================================

# Full benchmark suite
bench:
	@echo "Running full benchmark suite..."
	@cd benchmarks && node index.js run --env=$(BENCH_ENV)

# API benchmarks
bench-api:
	@echo "Running API benchmarks..."
	@cd benchmarks && node runners/api-benchmarks.js --env=$(BENCH_ENV)

# Database benchmarks
bench-db:
	@echo "Running database benchmarks..."
	@cd benchmarks && node runners/db-benchmarks.js --env=$(BENCH_ENV)

# End-to-end benchmarks
bench-e2e:
	@echo "Running end-to-end benchmarks..."
	@cd benchmarks && node runners/end-to-end.js --env=$(BENCH_ENV)

# Critical path benchmarks
bench-critical:
	@echo "Running critical path benchmarks..."
	@cd benchmarks && node index.js run --critical --env=$(BENCH_ENV)

# Generate benchmark report
bench-report:
	@echo "Generating benchmark report..."
	@cd benchmarks && node index.js report

# Start benchmark scheduler
bench-schedule:
	@echo "Starting benchmark scheduler..."
	@cd benchmarks && node index.js schedule

# Benchmark environment (default: local)
BENCH_ENV ?= local
