# =============================================================================
# Integration Test Makefile
# =============================================================================

.PHONY: help test test-integration test-unit test-coverage \
        docker-up docker-down docker-logs docker-clean \
        setup-test-env run-api-tests run-db-tests

# Default target
help:
	@echo "ServicePro Integration Test Commands"
	@echo "======================================"
	@echo ""
	@echo "Docker Environment:"
	@echo "  make docker-up       - Start test environment containers"
	@echo "  make docker-down     - Stop test environment containers"
	@echo "  make docker-logs     - View container logs"
	@echo "  make docker-clean    - Remove containers and volumes"
	@echo ""
	@echo "Running Tests:"
	@echo "  make test            - Run all tests (unit + integration)"
	@echo "  make test-unit       - Run unit tests only"
	@echo "  make test-integration- Run integration tests only"
	@echo "  make test-coverage   - Run tests with coverage report"
	@echo ""
	@echo "Specific Test Suites:"
	@echo "  make run-api-tests   - Run API integration tests"
	@echo "  make run-db-tests    - Run database integration tests"
	@echo ""
	@echo "Setup:"
	@echo "  make setup-test-env  - Set up local test environment"

# =============================================================================
# Docker Commands
# =============================================================================

# Start test containers
docker-up:
	@echo "Starting test environment..."
	docker compose -f ../../docker-compose.test.yml up -d
	@echo "Waiting for services to be healthy..."
	@sleep 10
	@echo "Test environment ready!"

# Stop test containers
docker-down:
	@echo "Stopping test environment..."
	docker compose -f ../../docker-compose.test.yml down

# View container logs
docker-logs:
	docker compose -f ../../docker-compose.test.yml logs -f

# Clean up containers and volumes
docker-clean:
	@echo "Cleaning up test environment..."
	docker compose -f ../../docker-compose.test.yml down -v --remove-orphans
	@echo "Cleanup complete!"

# =============================================================================
# Test Commands
# =============================================================================

# Run all tests
test: test-unit test-integration

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	cd .. && go test -v -count=1 ./...

# Run integration tests only
test-integration: docker-up
	@echo "Running integration tests..."
	cd .. && go test -v -tags=integration -count=1 -timeout=10m ./tests/integration/...
	@$(MAKE) docker-down

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	cd .. && go test -v -tags=integration -coverprofile=coverage.out -covermode=atomic ./...
	cd .. && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# =============================================================================
# Specific Test Suites
# =============================================================================

# Run API tests
run-api-tests: docker-up
	@echo "Running API integration tests..."
	cd .. && go test -v -tags=integration -count=1 -timeout=5m ./tests/integration/api/...

# Run database tests
run-db-tests: docker-up
	@echo "Running database integration tests..."
	cd .. && go test -v -tags=integration -count=1 -timeout=5m ./tests/integration/database/...

# =============================================================================
# Local Development Setup
# =============================================================================

# Set up test environment variables for local development
setup-test-env:
	@echo "Setting up test environment..."
	@echo "export TEST_DATABASE_URL=postgresql://test_user:test_password@localhost:5433/servicepro_test?sslmode=disable" > .env.test
	@echo "export REDIS_URL=redis://localhost:6380" >> .env.test
	@echo "export STRIPE_API_BASE=http://localhost:12111" >> .env.test
	@echo "export AWS_ENDPOINT_URL=http://localhost:4566" >> .env.test
	@echo "export MAILHOG_API=http://localhost:8025" >> .env.test
	@echo "export JWT_SECRET=test-jwt-secret-for-integration-tests" >> .env.test
	@echo ""
	@echo "Test environment file created: .env.test"
	@echo "Source it with: source .env.test"

# =============================================================================
# CI/CD Commands
# =============================================================================

# Run tests in CI mode
ci-test:
	@echo "Running tests in CI mode..."
	docker compose -f ../../docker-compose.test.yml run --rm test-runner

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	cd .. && go test -v -tags=integration -race -count=1 ./tests/integration/...

# Run tests in verbose mode with detailed output
test-verbose:
	@echo "Running tests in verbose mode..."
	cd .. && go test -v -tags=integration -count=1 -json ./tests/integration/... | tee test-results.json
