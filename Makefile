.PHONY: help setup-hooks lint lint-fix test format

help:
	@echo "Available targets:"
	@echo "  setup-hooks  - Install pre-commit hooks"
	@echo "  lint         - Run all linters"
	@echo "  lint-fix     - Run linters with auto-fix"
	@echo "  format       - Format all code"
	@echo "  test         - Run all tests"
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
