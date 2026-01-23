.PHONY: help setup-hooks lint lint-fix test format dev up down migrate
.PHONY: test-unit test-integration test-e2e test-performance test-all
.PHONY: coverage ci notify
.PHONY: k6-smoke k6-load k6-stress k6-ci
.PHONY: artillery-stress artillery-peak artillery-soak artillery-spike
.PHONY: bench bench-api bench-db bench-e2e bench-critical bench-report bench-schedule
.PHONY: docker-dev docker-prod docker-test docker-down docker-clean docker-logs docker-ps

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
	@echo "  lint-fix     - Run linters with auto-fix"
	@echo "  format       - Format all code"
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
	@echo "  ci           - Run full CI pipeline locally"
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
	@echo "Terraform (TF_ENV=development|staging|production):"
	@echo "  tf-init          - Initialize Terraform"
	@echo "  tf-plan          - Create execution plan"
	@echo "  tf-apply         - Apply the saved plan"
	@echo "  tf-destroy       - Destroy infrastructure"
	@echo "  tf-fmt           - Format Terraform files"
	@echo "  tf-lint          - Run TFLint"
	@echo "  tf-security-scan - Run security scans"
	@echo "  tf-test          - Run full validation suite"
	@echo "  tf-output        - Show all outputs"
	@echo "  tf-help          - Show all terraform targets"
	@echo ""
	@echo "Specific:"
	@echo "  frontend-*   - Frontend-specific targets"
	@echo "  backend-*    - Backend-specific targets"

# Pre-commit setup
setup-hooks:
	@./scripts/setup-pre-commit.sh

# Combined linting
lint: frontend-lint backend-lint

lint-fix: frontend-lint-fix backend-lint-fix format

# Combined formatting
format:
	@echo "Formatting all files with Prettier..."
	@npx prettier --write .

# Combined testing
test: frontend-test backend-test

# Frontend targets
frontend-lint:
	@echo "Running ESLint on frontend..."
	@if [ -n "$$(find frontend/src -name '*.js' -o -name '*.jsx' -o -name '*.ts' -o -name '*.tsx' 2>/dev/null)" ]; then \
		cd frontend && npx eslint .; \
	else \
		echo "No JavaScript/TypeScript files found in frontend/src"; \
	fi

frontend-lint-fix:
	@echo "Running ESLint with auto-fix on frontend..."
	@if [ -n "$$(find frontend/src -name '*.js' -o -name '*.jsx' -o -name '*.ts' -o -name '*.tsx' 2>/dev/null)" ]; then \
		cd frontend && npx eslint . --fix; \
	else \
		echo "No JavaScript/TypeScript files found in frontend/src"; \
	fi

frontend-format:
	@echo "Formatting frontend code..."
	@npx prettier --write "frontend/**/*.{js,jsx,ts,tsx,json,css,scss,md}"

frontend-test:
	@echo "Running frontend tests..."
	@cd frontend && npm test -- --watchAll=false

frontend-build:
	@echo "Building frontend..."
	@cd frontend && npm run build

# Backend targets
backend-lint:
	@echo "Running golangci-lint on backend..."
	@if [ -n "$$(find backend -name '*.go' 2>/dev/null)" ]; then \
		cd backend && golangci-lint run --config .golangci.yml; \
	else \
		echo "No Go files found in backend"; \
	fi

backend-lint-fix:
	@echo "Running go fmt and go imports on backend..."
	@cd backend && gofmt -w . && goimports -w -local github.com/javaknight1/servicepro .

backend-format:
	@echo "Formatting backend code..."
	@cd backend && gofmt -w .

backend-test:
	@echo "Running backend tests..."
	@cd backend && go test ./... -v

backend-build:
	@echo "Building backend..."
	@cd backend && go build -o bin/servicepro ./cmd/server

# Install dependencies
install-deps:
	@echo "Installing frontend dependencies..."
	@cd frontend && npm install
	@echo "Installing backend dependencies..."
	@cd backend && go mod download
	@echo "Installing pre-commit..."
	@./scripts/setup-pre-commit.sh

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
	@cd backend && go run cmd/api/cmd/main.go

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

# Full CI pipeline
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

# =============================================================================
# Terraform Infrastructure
# =============================================================================

# Terraform variables
TF_ENV ?= development
TF_DIR = infrastructure
TF_VAR_FILE = environments/$(TF_ENV).tfvars
TF_PLAN_FILE = .terraform/plans/$(TF_ENV).tfplan

# Colors for terraform output
TF_BLUE := \033[0;34m
TF_GREEN := \033[0;32m
TF_YELLOW := \033[1;33m
TF_RED := \033[0;31m
TF_NC := \033[0m

.PHONY: tf-init tf-validate tf-plan tf-apply tf-destroy tf-fmt tf-lint
.PHONY: tf-security-scan tf-state-list tf-state-show tf-refresh tf-output
.PHONY: tf-workspace-list tf-workspace-new tf-workspace-select
.PHONY: tf-cost-estimate tf-graph tf-docs tf-clean
.PHONY: tf-ci-init tf-ci-validate tf-ci-plan tf-ci-apply
.PHONY: tf-unlock tf-import tf-taint tf-untaint

tf-help: ## Show terraform help
	@echo ""
	@echo "Terraform Infrastructure Management"
	@echo "===================================="
	@echo ""
	@echo "Usage: make tf-<target> [TF_ENV=environment]"
	@echo ""
	@echo "Targets:"
	@echo "  tf-init           Initialize Terraform"
	@echo "  tf-validate       Validate configuration"
	@echo "  tf-plan           Create execution plan"
	@echo "  tf-apply          Apply the saved plan"
	@echo "  tf-destroy        Destroy infrastructure"
	@echo "  tf-fmt            Format Terraform files"
	@echo "  tf-lint           Run TFLint"
	@echo "  tf-security-scan  Run security scans"
	@echo "  tf-state-list     List resources in state"
	@echo "  tf-output         Show all outputs"
	@echo "  tf-cost-estimate  Estimate costs (requires infracost)"
	@echo ""
	@echo "Environments: development, staging, production"
	@echo ""

# Initialization
tf-init: ## Initialize Terraform
	@echo "$(TF_BLUE)Initializing Terraform for $(TF_ENV)...$(TF_NC)"
	@cd $(TF_DIR) && terraform init -upgrade
	@mkdir -p $(TF_DIR)/.terraform/plans

tf-init-backend: ## Initialize with backend configuration
	@echo "$(TF_BLUE)Initializing Terraform with backend for $(TF_ENV)...$(TF_NC)"
	@cd $(TF_DIR) && terraform init -backend-config=backends/$(TF_ENV).conf -upgrade

# Validation
tf-validate: ## Validate Terraform configuration
	@echo "$(TF_BLUE)Validating Terraform configuration...$(TF_NC)"
	@cd $(TF_DIR) && terraform validate
	@echo "$(TF_GREEN)Validation successful!$(TF_NC)"

tf-fmt: ## Format Terraform files
	@echo "$(TF_BLUE)Formatting Terraform files...$(TF_NC)"
	@cd $(TF_DIR) && terraform fmt -recursive
	@echo "$(TF_GREEN)Formatting complete!$(TF_NC)"

tf-fmt-check: ## Check Terraform formatting (CI/CD)
	@echo "$(TF_BLUE)Checking Terraform formatting...$(TF_NC)"
	@cd $(TF_DIR) && terraform fmt -check -recursive

tf-lint: ## Run TFLint
	@echo "$(TF_BLUE)Running TFLint...$(TF_NC)"
	@cd $(TF_DIR) && tflint --init && tflint --recursive
	@echo "$(TF_GREEN)Linting complete!$(TF_NC)"

tf-security-scan: ## Run security scans (tfsec + checkov)
	@echo "$(TF_BLUE)Running security scans...$(TF_NC)"
	@echo "$(TF_YELLOW)Running tfsec...$(TF_NC)"
	@cd $(TF_DIR) && tfsec . --soft-fail || true
	@echo ""
	@echo "$(TF_YELLOW)Running checkov...$(TF_NC)"
	@cd $(TF_DIR) && checkov -d . --quiet || true
	@echo "$(TF_GREEN)Security scan complete!$(TF_NC)"

# Planning
tf-plan: tf-init tf-validate ## Create execution plan
	@echo "$(TF_BLUE)Creating plan for $(TF_ENV)...$(TF_NC)"
	@cd $(TF_DIR) && terraform plan -var-file=$(TF_VAR_FILE) -out=$(TF_PLAN_FILE)
	@echo "$(TF_GREEN)Plan created: $(TF_PLAN_FILE)$(TF_NC)"

tf-plan-destroy: tf-init ## Create destruction plan
	@echo "$(TF_YELLOW)Creating destruction plan for $(TF_ENV)...$(TF_NC)"
	@cd $(TF_DIR) && terraform plan -destroy -var-file=$(TF_VAR_FILE) -out=$(TF_PLAN_FILE).destroy

tf-show-plan: ## Show the current plan
	@cd $(TF_DIR) && terraform show $(TF_PLAN_FILE)

# Deployment
tf-apply: ## Apply the saved plan
	@echo "$(TF_YELLOW)Applying plan for $(TF_ENV)...$(TF_NC)"
	@echo "$(TF_RED)WARNING: This will modify infrastructure!$(TF_NC)"
	@read -p "Continue? [y/N] " confirm && [ "$$confirm" = "y" ]
	@cd $(TF_DIR) && terraform apply $(TF_PLAN_FILE)
	@echo "$(TF_GREEN)Apply complete!$(TF_NC)"

tf-apply-auto: ## Apply without confirmation (CI/CD)
	@echo "$(TF_YELLOW)Auto-applying plan for $(TF_ENV)...$(TF_NC)"
	@cd $(TF_DIR) && terraform apply -auto-approve $(TF_PLAN_FILE)
	@echo "$(TF_GREEN)Apply complete!$(TF_NC)"

tf-destroy: ## Destroy infrastructure
	@echo "$(TF_RED)WARNING: This will DESTROY all $(TF_ENV) infrastructure!$(TF_NC)"
	@read -p "Type '$(TF_ENV)' to confirm: " confirm && [ "$$confirm" = "$(TF_ENV)" ]
	@cd $(TF_DIR) && terraform destroy -var-file=$(TF_VAR_FILE)
	@echo "$(TF_GREEN)Destruction complete!$(TF_NC)"

# State Management
tf-state-list: ## List resources in state
	@cd $(TF_DIR) && terraform state list

tf-state-show: ## Show state for a resource (TF_RESOURCE=resource_name)
	@cd $(TF_DIR) && terraform state show $(TF_RESOURCE)

tf-state-pull: ## Pull remote state
	@cd $(TF_DIR) && terraform state pull > state-backup-$(shell date +%Y%m%d-%H%M%S).json
	@echo "$(TF_GREEN)State backed up$(TF_NC)"

tf-refresh: ## Refresh state from actual infrastructure
	@echo "$(TF_BLUE)Refreshing state for $(TF_ENV)...$(TF_NC)"
	@cd $(TF_DIR) && terraform refresh -var-file=$(TF_VAR_FILE)
	@echo "$(TF_GREEN)State refreshed!$(TF_NC)"

# Workspace Management
tf-workspace-list: ## List workspaces
	@cd $(TF_DIR) && terraform workspace list

tf-workspace-new: ## Create new workspace (TF_WS=workspace_name)
	@cd $(TF_DIR) && terraform workspace new $(TF_WS)

tf-workspace-select: ## Select workspace
	@cd $(TF_DIR) && terraform workspace select $(TF_ENV)

# Output
tf-output: ## Show all outputs
	@cd $(TF_DIR) && terraform output

tf-output-json: ## Show outputs as JSON
	@cd $(TF_DIR) && terraform output -json

# Utilities
tf-cost-estimate: ## Estimate costs (requires infracost)
	@echo "$(TF_BLUE)Estimating costs for $(TF_ENV)...$(TF_NC)"
	@cd $(TF_DIR) && infracost breakdown --path . --terraform-var-file=$(TF_VAR_FILE)

tf-graph: ## Generate resource graph
	@echo "$(TF_BLUE)Generating resource graph...$(TF_NC)"
	@cd $(TF_DIR) && terraform graph | dot -Tpng > infrastructure-graph.png
	@echo "$(TF_GREEN)Graph saved to $(TF_DIR)/infrastructure-graph.png$(TF_NC)"

tf-docs: ## Generate documentation (requires terraform-docs)
	@echo "$(TF_BLUE)Generating documentation...$(TF_NC)"
	@cd $(TF_DIR) && terraform-docs markdown table . > TERRAFORM.md
	@echo "$(TF_GREEN)Documentation saved to $(TF_DIR)/TERRAFORM.md$(TF_NC)"

tf-clean: ## Clean up temporary files
	@echo "$(TF_BLUE)Cleaning up...$(TF_NC)"
	@rm -rf $(TF_DIR)/.terraform/plans/*.tfplan
	@rm -rf $(TF_DIR)/.terraform/plans/*.tfplan.destroy
	@rm -f $(TF_DIR)/*.tfplan
	@rm -f $(TF_DIR)/crash.log
	@rm -f $(TF_DIR)/state-backup-*.json
	@echo "$(TF_GREEN)Cleanup complete!$(TF_NC)"

# CI/CD Targets
tf-ci-init: ## CI/CD initialization
	@cd $(TF_DIR) && terraform init -input=false

tf-ci-validate: tf-ci-init tf-fmt-check ## CI/CD validation
	@cd $(TF_DIR) && terraform validate

tf-ci-plan: tf-ci-validate ## CI/CD plan
	@cd $(TF_DIR) && terraform plan -var-file=$(TF_VAR_FILE) -input=false -out=$(TF_PLAN_FILE)

tf-ci-apply: ## CI/CD apply
	@cd $(TF_DIR) && terraform apply -input=false -auto-approve $(TF_PLAN_FILE)

# Emergency Procedures
tf-unlock: ## Unlock state (TF_LOCK_ID required)
	@echo "$(TF_RED)WARNING: Unlocking state can cause issues!$(TF_NC)"
	@cd $(TF_DIR) && terraform force-unlock $(TF_LOCK_ID)

tf-import: ## Import existing resource (TF_RESOURCE, TF_ID required)
	@echo "$(TF_BLUE)Importing $(TF_RESOURCE) with ID $(TF_ID)...$(TF_NC)"
	@cd $(TF_DIR) && terraform import -var-file=$(TF_VAR_FILE) $(TF_RESOURCE) $(TF_ID)

tf-taint: ## Taint a resource for recreation (TF_RESOURCE required)
	@echo "$(TF_YELLOW)Tainting $(TF_RESOURCE) for recreation...$(TF_NC)"
	@cd $(TF_DIR) && terraform taint $(TF_RESOURCE)

tf-untaint: ## Remove taint from resource (TF_RESOURCE required)
	@echo "$(TF_BLUE)Removing taint from $(TF_RESOURCE)...$(TF_NC)"
	@cd $(TF_DIR) && terraform untaint $(TF_RESOURCE)

# Validation
tf-test: ## Run full terraform validation suite
	@echo "$(TF_BLUE)Running terraform validation suite...$(TF_NC)"
	@cd $(TF_DIR) && chmod +x tests/validate.sh && ./tests/validate.sh --all

tf-test-quick: ## Run quick terraform validation
	@echo "$(TF_BLUE)Running quick terraform validation...$(TF_NC)"
	@cd $(TF_DIR) && chmod +x tests/validate.sh && ./tests/validate.sh --quick
