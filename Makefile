# =============================================================================
# ServicePro — the ONE Makefile (CLAUDE.md: no Makefiles in subdirectories)
#
# `make` or `make help` lists every documented target (lines with `## `).
# CI (.github/workflows/ci.yml) runs the same commands these targets wrap.
# =============================================================================

SHELL := /bin/bash
.DEFAULT_GOAL := help

# -----------------------------------------------------------------------------
# Tool versions — single source of truth for local + CI
# -----------------------------------------------------------------------------
# Go and Node come from files CI also reads (setup-go go-version-file,
# setup-node node-version-file), so these are informational.
GO_VERSION           := $(shell awk '$$1 == "go" { print $$2; exit }' backend/go.mod)
NODE_VERSION         := $(shell cat frontend/.nvmrc 2>/dev/null)
# Keep in sync with golangci/golangci-lint-action `version:` in ci.yml.
GOLANGCI_LINT_VERSION ?= v2.11.4
# swag must match the github.com/swaggo/swag version in backend/go.mod, or the
# generated spec drifts from what CI regenerates.
SWAG_VERSION         := $(shell awk '$$1 == "github.com/swaggo/swag" { print $$2 }' backend/go.mod)
GOIMPORTS_VERSION    := $(shell awk '$$1 == "golang.org/x/tools" { print $$2 }' backend/go.mod)
PRETTIER             := frontend/node_modules/.bin/prettier

# Local docker-compose database (see docker-compose.yml)
LOCAL_DATABASE_URL   ?= postgresql://postgres:password@localhost:5432/servicepro?sslmode=disable
PG_CONTAINER         := servicepro-postgres
SEED_FILE            := backend/seeds/dev_seed.sql

.PHONY: help versions \
	setup install-deps install-golangci-lint install-hooks \
	dev dev-db dev-backend dev-frontend up down \
	migrate migrate-version migrate-down migrate-create seed db-wait db-fresh db-fresh-seed db-reset hash-password invite-code \
	lint lint-check lint-fix format format-check \
	backend-lint backend-lint-fix backend-format backend-test backend-build backend-vuln backend-docker \
	frontend-lint frontend-lint-fix frontend-format frontend-typecheck frontend-test frontend-build \
	frontend-bundle-check frontend-dead-code frontend-dead-code-update frontend-audit frontend-worker-dev \
	dead-code dead-code-update \
	test test-unit test-integration test-integration-api test-integration-db test-e2e test-e2e-open test-all \
	coverage coverage-view \
	ci-local ci-backend ci-frontend \
	swagger generate-api docs openapi-check \
	docker-dev docker-dev-d docker-down docker-clean docker-logs docker-logs-backend docker-logs-frontend \
	docker-logs-db docker-ps docker-shell-backend docker-shell-frontend docker-shell-db \
	docker-rebuild-backend docker-rebuild-frontend logs ps \
	bump release-check clean

help: ## Show this help
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-26s\033[0m %s\n", $$1, $$2}'

versions: ## Print the pinned tool versions
	@echo "go:            $(GO_VERSION) (backend/go.mod)"
	@echo "node:          $(NODE_VERSION) (frontend/.nvmrc)"
	@echo "golangci-lint: $(GOLANGCI_LINT_VERSION)"
	@echo "swag:          $(SWAG_VERSION) (backend/go.mod)"
	@echo "VERSION:       $$(tr -d '[:space:]' < VERSION)"

# =============================================================================
# Setup
# =============================================================================

setup: install-deps install-hooks dev-db db-wait migrate seed ## Complete local setup (deps, hooks, db, migrations, seed)
	@echo ""
	@echo "Setup complete. Login: dev@servicepro.local / password123"
	@echo "Next: make dev-backend  (terminal 1)  and  make dev-frontend  (terminal 2)"

install-deps: ## Install frontend (npm ci) and backend (go mod download) dependencies
	@cd frontend && npm ci
	@cd backend && go mod download

install-golangci-lint: ## Install golangci-lint at the pinned version
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/$(GOLANGCI_LINT_VERSION)/install.sh \
		| sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

install-hooks: ## Symlink the versioned git hooks (scripts/git-hooks) into .git/hooks
	@hooks_dir="$$(git rev-parse --git-path hooks)"; mkdir -p "$$hooks_dir"; \
	for h in pre-commit commit-msg; do \
		chmod +x "scripts/git-hooks/$$h"; \
		ln -sf "$(CURDIR)/scripts/git-hooks/$$h" "$$hooks_dir/$$h"; \
		echo "installed $$hooks_dir/$$h -> scripts/git-hooks/$$h"; \
	done

# =============================================================================
# Local development
# =============================================================================

dev: dev-db ## Start db services and print how to run backend + frontend
	@echo "Run in separate terminals:"
	@echo "  make dev-backend   # http://localhost:8080"
	@echo "  make dev-frontend  # http://localhost:3000"

dev-db: ## Start PostgreSQL, Redis, MinIO and Mailpit containers
	@docker compose up -d postgres redis minio minio-init mailpit
	@docker compose ps

dev-backend: ## Run the API on the host (go run ./cmd)
	@cd backend && go run ./cmd

dev-frontend: ## Run the Vite dev server on the host
	@cd frontend && npm run dev

up: docker-dev ## Alias for docker-dev
down: docker-down ## Alias for docker-down

# =============================================================================
# Database
# =============================================================================

migrate: ## Apply pending migrations to the local db (go run ./cmd/migrate up)
	@cd backend && DATABASE_URL="$(LOCAL_DATABASE_URL)" MIGRATION_DATABASE_URL="$(LOCAL_DATABASE_URL)" go run ./cmd/migrate up

migrate-version: ## Print the applied schema version + dirty flag (local db)
	@cd backend && DATABASE_URL="$(LOCAL_DATABASE_URL)" MIGRATION_DATABASE_URL="$(LOCAL_DATABASE_URL)" go run ./cmd/migrate version

migrate-down: ## Roll back migrations on the LOCAL db only
	@cd backend && DATABASE_URL="$(LOCAL_DATABASE_URL)" MIGRATION_DATABASE_URL="$(LOCAL_DATABASE_URL)" go run ./cmd/migrate down

migrate-create: ## Create the next migration pair (make migrate-create NAME=add_foo)
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=add_foo"; exit 1; fi
	@last=$$(ls backend/migrations/[0-9]*_*.up.sql 2>/dev/null | sed -E 's#.*/0*([0-9]+)_.*#\1#' | sort -n | tail -1); \
	next=$$(printf "%06d" $$(( $${last:-0} + 1 ))); \
	touch "backend/migrations/$${next}_$(NAME).up.sql" "backend/migrations/$${next}_$(NAME).down.sql"; \
	echo "Created backend/migrations/$${next}_$(NAME).up.sql"; \
	echo "Created backend/migrations/$${next}_$(NAME).down.sql"

seed: ## Load dev seed data (backend/seeds/dev_seed.sql) into the local db
	@test -f "$(SEED_FILE)" || { echo "error: $(SEED_FILE) not found"; exit 1; }
	@docker exec -i $(PG_CONTAINER) psql -v ON_ERROR_STOP=1 -U postgres -d servicepro < "$(SEED_FILE)"
	@echo "Seed loaded. Login: dev@servicepro.local / password123"

db-wait: ## Wait until the local PostgreSQL container accepts connections
	@until docker exec $(PG_CONTAINER) pg_isready -U postgres -d servicepro >/dev/null 2>&1; do \
		echo "  waiting for PostgreSQL..."; sleep 2; \
	done

db-fresh: ## Recreate the local db volumes and migrate (no confirmation)
	@docker compose down -v 2>/dev/null || true
	@docker compose up -d postgres redis minio minio-init mailpit
	@$(MAKE) --no-print-directory db-wait migrate

db-fresh-seed: db-fresh seed ## Recreate, migrate and seed the local db

db-reset: ## Recreate the local db after confirmation (deletes all data)
	@read -p "Delete ALL local data? [y/N] " -n 1 -r; echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then $(MAKE) --no-print-directory db-fresh; else echo "Cancelled."; fi

hash-password: ## Print a bcrypt hash (make hash-password PASSWORD=secret)
	@cd backend && go run ../scripts/hash-password.go $(PASSWORD)

invite-code: ## Create a registration invite code (make invite-code [MAX_USES=1] [EXPIRES=168h] [NOTE=...]; DATABASE_URL overrides the local db)
	@cd backend && DATABASE_URL="$${DATABASE_URL:-$(LOCAL_DATABASE_URL)}" go run ./cmd/invite create \
		-max-uses "$(or $(MAX_USES),1)" -expires "$(or $(EXPIRES),168h)" -note "$(NOTE)"

# =============================================================================
# Lint + format
# =============================================================================

lint: frontend-lint backend-lint ## Run all linters

lint-check: frontend-lint backend-lint frontend-typecheck format-check ## All lint checks, no modifications

lint-fix: frontend-lint-fix backend-lint-fix format ## Lint with auto-fix, then format

format: ## Format all files with Prettier
	@$(PRETTIER) --write .

format-check: ## Check Prettier formatting without modifying
	@$(PRETTIER) --check .

backend-lint: ## golangci-lint the backend
	@cd backend && golangci-lint run --config .golangci.yml ./...

backend-lint-fix: ## gofmt + goimports the backend
	@cd backend && gofmt -w . && go run golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION) -w -local github.com/javaknight1/servicepro .

backend-format: ## gofmt the backend
	@cd backend && gofmt -w .

frontend-lint: ## ESLint the frontend (zero warnings)
	@cd frontend && npm run lint

frontend-lint-fix: ## ESLint the frontend with --fix
	@cd frontend && npx eslint . --fix --max-warnings 0

frontend-format: ## Prettier the frontend
	@$(PRETTIER) --write "frontend/**/*.{js,jsx,ts,tsx,json,css,scss,md}"

frontend-typecheck: ## tsc --noEmit
	@cd frontend && npm run type-check

# =============================================================================
# Tests + builds
# =============================================================================

test: backend-test frontend-test ## Run backend + frontend unit tests

test-unit: test ## Alias for test

backend-test: ## Backend unit tests (race, integration skipped)
	@cd backend && go test -race -timeout 5m ./... -skip Integration

backend-build: ## Compile every backend package
	@cd backend && go build ./...

backend-vuln: ## govulncheck the backend
	@cd backend && go run golang.org/x/vuln/cmd/govulncheck@latest ./...

backend-docker: ## Build backend/Dockerfile locally (tag servicepro-api:local)
	docker build -f backend/Dockerfile \
		--build-arg GIT_SHA="$$(git rev-parse HEAD)" \
		--build-arg APP_VERSION="$$(tr -d '[:space:]' < VERSION)-dev" \
		-t servicepro-api:local backend

frontend-test: ## Frontend unit tests (vitest run)
	@cd frontend && npx vitest run --passWithNoTests

frontend-build: ## Production build of the frontend
	@cd frontend && npm run build

frontend-bundle-check: ## Check bundle size budget (after frontend-build)
	@cd frontend && npm run analyze:bundle -- --ci

frontend-dead-code: ## Check for new unused exports
	@cd frontend && npm run dead-code

frontend-dead-code-update: ## Update the dead-code baseline
	@cd frontend && npm run dead-code:update

frontend-audit: ## npm audit, high+ on production deps (matches CI)
	@cd frontend && npm audit --audit-level=high --omit=dev

frontend-worker-dev: ## Run the Cloudflare Worker (static assets + /api proxy) locally after a build
	@cd frontend && npx wrangler dev

dead-code: frontend-dead-code ## Alias for frontend-dead-code
dead-code-update: frontend-dead-code-update ## Alias for frontend-dead-code-update

# Integration tests (backend/tests/integration, build tag `integration`).
# They expect Postgres + Redis + Mailpit (the docker-compose stack) and, for
# the api/ suite, a running API on API_BASE_URL. NOTE: the suite does not
# compile today (SetupTest lives in a _test.go file other packages import) —
# see the TODO on the db-integration job in ci.yml.
INTEGRATION_ENV := TEST_DATABASE_URL="$(LOCAL_DATABASE_URL)" REDIS_URL=redis://localhost:6379 \
	MAILHOG_API=http://localhost:8025 API_BASE_URL=http://localhost:8080

test-integration: ## Backend integration tests against the compose stack
	@cd backend && $(INTEGRATION_ENV) go test -tags=integration -count=1 -timeout=10m ./tests/integration/...

test-integration-api: ## API integration tests only
	@cd backend && $(INTEGRATION_ENV) go test -tags=integration -count=1 -timeout=5m ./tests/integration/api/...

test-integration-db: ## Database integration tests only
	@cd backend && $(INTEGRATION_ENV) go test -tags=integration -count=1 -timeout=5m ./tests/integration/database/...

test-e2e: ## Cypress E2E tests (headless; needs the app running)
	@cd frontend && npx cypress run

test-e2e-open: ## Open the Cypress runner
	@cd frontend && npx cypress open

test-all: test test-integration test-e2e ## Unit + integration + E2E

coverage: ## Coverage reports (backend/coverage.html, frontend/coverage/)
	@cd backend && go test -coverprofile=coverage.out -covermode=atomic ./... -skip Integration && go tool cover -html=coverage.out -o coverage.html
	@cd frontend && npm run test:coverage

coverage-view: ## Open the coverage reports
	@open backend/coverage.html 2>/dev/null || xdg-open backend/coverage.html 2>/dev/null || true
	@open frontend/coverage/index.html 2>/dev/null || xdg-open frontend/coverage/index.html 2>/dev/null || true

# =============================================================================
# CI parity
# =============================================================================

ci-local: ci-backend ci-frontend openapi-check ## Run the core CI gates locally
	@echo "All local CI checks passed."

ci-backend: backend-lint backend-test backend-build ## Backend CI gates (lint, test, build)

ci-frontend: ## Frontend CI gates (typecheck, lint, test, dead-code, build, bundle)
	@cd frontend && npm run type-check && npm run lint && npx vitest run --passWithNoTests
	@cd frontend && npm run dead-code && npm run build && npm run analyze:bundle -- --ci

# =============================================================================
# API documentation (OpenAPI)
# =============================================================================
# backend/docs/{swagger.json,swagger.yaml,docs.go} and
# frontend/src/types/api.generated.ts are COMMITTED; ci.yml's openapi-drift job
# reruns `make docs` and fails on any diff.

swagger: ## Generate backend/docs from swag annotations (pinned swag)
	@cd backend && go tool swag init \
		-g cmd/main.go -o docs --parseDependency --parseInternal --quiet
	@echo "Swagger docs generated in backend/docs/"

generate-api: ## Generate frontend/src/types/api.generated.ts from the spec
	@cd frontend && npm run --silent generate:api
	@echo "TypeScript API types generated in frontend/src/types/api.generated.ts"

docs: swagger generate-api ## Full API doc pipeline (swagger + generate-api)

openapi-check: docs ## Regenerate and fail if the committed API artifacts drift
	@git diff --exit-code -- backend/docs frontend/src/types/api.generated.ts \
		|| { echo "OpenAPI artifacts are stale: commit the regenerated files."; exit 1; }

# =============================================================================
# Docker (local stack)
# =============================================================================

docker-dev: ## Start the full compose stack in the foreground
	@APP_VERSION="$$(tr -d '[:space:]' < VERSION)-dev" docker compose up

docker-dev-d: ## Start the full compose stack detached
	@APP_VERSION="$$(tr -d '[:space:]' < VERSION)-dev" docker compose up -d
	@docker compose ps

docker-down: ## Stop all containers
	@docker compose down

docker-clean: ## Remove containers, volumes and local images
	@docker compose down -v --rmi local
	@docker system prune -f

docker-logs: ## Follow logs from all containers
	@docker compose logs -f

docker-logs-backend: ## Follow backend logs
	@docker compose logs -f backend

docker-logs-frontend: ## Follow frontend logs
	@docker compose logs -f frontend

docker-logs-db: ## Follow PostgreSQL logs
	@docker compose logs -f postgres

docker-ps: ## Show running containers
	@docker compose ps

docker-shell-backend: ## Shell into the backend container
	@docker exec -it servicepro-backend sh

docker-shell-frontend: ## Shell into the frontend container
	@docker exec -it servicepro-frontend sh

docker-shell-db: ## psql into the local database
	@docker exec -it $(PG_CONTAINER) psql -U postgres -d servicepro

docker-rebuild-backend: ## Recreate the backend container
	@docker compose up -d --force-recreate backend

docker-rebuild-frontend: ## Recreate the frontend container
	@docker compose up -d --force-recreate frontend

logs: docker-logs ## Alias for docker-logs
ps: docker-ps ## Alias for docker-ps

# =============================================================================
# Release — production ships from a v* tag only (RELEASING.md)
# =============================================================================
# `make bump` edits VERSION and stops; committing and tagging stay explicit,
# because pushing a tag is what starts the release pipeline.

bump: ## Bump VERSION for the next release (ARGS="--dry-run" / ARGS="--set X.Y.Z")
	@./scripts/bump-version.sh $(ARGS)

release-check: ## Verify the working tree is ready to tag a release
	@test -z "$$(git status --porcelain)" || { echo "error: working tree is dirty"; exit 1; }
	@echo "VERSION:      $$(tr -d '[:space:]' < VERSION)"
	@echo "branch:       $$(git rev-parse --abbrev-ref HEAD)"
	@echo "last release: $$(git tag --list 'v*' --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -1 || echo '<none>')"
	@echo "-> git tag v$$(tr -d '[:space:]' < VERSION) && git push origin v$$(tr -d '[:space:]' < VERSION)"

# =============================================================================
# Clean
# =============================================================================

clean: ## Remove build artifacts and coverage output
	@rm -rf backend/bin backend/*.out backend/coverage.html backend/main backend/server
	@rm -rf frontend/dist frontend/coverage frontend/bundle-report.html
	@rm -rf frontend/cypress/screenshots frontend/cypress/videos frontend/cypress/reports
