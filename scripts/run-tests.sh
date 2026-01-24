#!/bin/bash
# =============================================================================
# ServicePro Test Runner
# =============================================================================
# Unified test execution script with multiple test suites

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default values
TEST_TYPE="all"
COVERAGE=false
PARALLEL=false
VERBOSE=false
CI_MODE=false
TIMEOUT=600

# Usage
usage() {
  echo "Usage: $0 [OPTIONS]"
  echo ""
  echo "Options:"
  echo "  -t, --type TYPE       Test type: unit, integration, e2e, all (default: all)"
  echo "  -c, --coverage        Generate coverage report"
  echo "  -p, --parallel        Run tests in parallel (where supported)"
  echo "  -v, --verbose         Verbose output"
  echo "  --ci                  CI mode (non-interactive, stricter)"
  echo "  --timeout SECONDS     Test timeout in seconds (default: 600)"
  echo "  -h, --help            Show this help"
  echo ""
  echo "Examples:"
  echo "  $0 -t unit -c          # Run unit tests with coverage"
  echo "  $0 -t e2e -v           # Run E2E tests verbosely"
  echo "  $0 --ci -p             # Run all tests in CI mode with parallelism"
  exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -t|--type)
      TEST_TYPE="$2"
      shift 2
      ;;
    -c|--coverage)
      COVERAGE=true
      shift
      ;;
    -p|--parallel)
      PARALLEL=true
      shift
      ;;
    -v|--verbose)
      VERBOSE=true
      shift
      ;;
    --ci)
      CI_MODE=true
      shift
      ;;
    --timeout)
      TIMEOUT="$2"
      shift 2
      ;;
    -h|--help)
      usage
      ;;
    *)
      echo "Unknown option: $1"
      usage
      ;;
  esac
done

# Print header
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           ServicePro Test Runner                               ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  Test Type: $TEST_TYPE"
echo "  Coverage: $COVERAGE"
echo "  Parallel: $PARALLEL"
echo "  CI Mode: $CI_MODE"
echo "  Timeout: ${TIMEOUT}s"
echo ""

# Track results
declare -A RESULTS
EXIT_CODE=0

# Run backend unit tests
run_backend_unit_tests() {
  echo -e "${BLUE}━━━ Running Backend Unit Tests ━━━${NC}"
  cd "$PROJECT_ROOT/backend"

  local ARGS="-v -race -timeout ${TIMEOUT}s"

  if [ "$COVERAGE" = true ]; then
    ARGS="$ARGS -coverprofile=coverage.out -covermode=atomic"
  fi

  if [ "$VERBOSE" = true ]; then
    ARGS="$ARGS -v"
  fi

  # Skip integration tests
  ARGS="$ARGS -skip Integration"

  if go test $ARGS ./... 2>&1 | tee backend-unit-test.log; then
    RESULTS["backend-unit"]="passed"
    echo -e "${GREEN}✓ Backend unit tests passed${NC}"

    if [ "$COVERAGE" = true ]; then
      echo ""
      echo "Coverage summary:"
      go tool cover -func=coverage.out | tail -1
    fi
  else
    RESULTS["backend-unit"]="failed"
    echo -e "${RED}✗ Backend unit tests failed${NC}"
    EXIT_CODE=1
  fi

  cd "$PROJECT_ROOT"
}

# Run frontend unit tests
run_frontend_unit_tests() {
  echo -e "${BLUE}━━━ Running Frontend Unit Tests ━━━${NC}"
  cd "$PROJECT_ROOT/frontend"

  local ARGS=""

  if [ "$COVERAGE" = true ]; then
    ARGS="--coverage"
  fi

  if [ "$CI_MODE" = true ]; then
    ARGS="$ARGS --reporter=verbose"
    export CI=true
  fi

  if npm run test $ARGS 2>&1 | tee frontend-unit-test.log; then
    RESULTS["frontend-unit"]="passed"
    echo -e "${GREEN}✓ Frontend unit tests passed${NC}"
  else
    RESULTS["frontend-unit"]="failed"
    echo -e "${RED}✗ Frontend unit tests failed${NC}"
    EXIT_CODE=1
  fi

  cd "$PROJECT_ROOT"
}

# Run backend integration tests
run_integration_tests() {
  echo -e "${BLUE}━━━ Running Integration Tests ━━━${NC}"

  # Check if docker compose is available
  if ! docker compose version &> /dev/null; then
    echo -e "${YELLOW}⚠ docker compose not found, skipping integration tests${NC}"
    RESULTS["integration"]="skipped"
    return
  fi

  cd "$PROJECT_ROOT/backend/tests"

  # Start test services
  echo "Starting test services..."
  docker compose -f docker-compose.test.yml up -d

  # Wait for services
  echo "Waiting for services to be ready..."
  sleep 10

  cd "$PROJECT_ROOT/backend"

  local ARGS="-v -race -tags=integration -timeout ${TIMEOUT}s"

  if [ "$COVERAGE" = true ]; then
    ARGS="$ARGS -coverprofile=coverage-integration.out"
  fi

  if go test $ARGS ./tests/integration/... 2>&1 | tee integration-test.log; then
    RESULTS["integration"]="passed"
    echo -e "${GREEN}✓ Integration tests passed${NC}"
  else
    RESULTS["integration"]="failed"
    echo -e "${RED}✗ Integration tests failed${NC}"
    EXIT_CODE=1
  fi

  # Cleanup
  cd "$PROJECT_ROOT/backend/tests"
  docker compose -f docker-compose.test.yml down -v

  cd "$PROJECT_ROOT"
}

# Run E2E tests
run_e2e_tests() {
  echo -e "${BLUE}━━━ Running E2E Tests ━━━${NC}"
  cd "$PROJECT_ROOT/frontend"

  local ARGS="--browser chrome"

  if [ "$PARALLEL" = true ]; then
    ARGS="$ARGS --parallel"
  fi

  if [ "$CI_MODE" = true ]; then
    ARGS="$ARGS --headless"
  fi

  # Check if servers are running
  if ! curl -s http://localhost:3000 > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠ Frontend server not running at localhost:3000${NC}"
    echo "Starting servers..."

    # Start backend
    cd "$PROJECT_ROOT/backend"
    go build -o server ./cmd
    ./server &
    BACKEND_PID=$!

    # Start frontend
    cd "$PROJECT_ROOT/frontend"
    npm run preview &
    FRONTEND_PID=$!

    # Wait for servers
    npx wait-on http://localhost:3000 http://localhost:8080/api/health --timeout 60000
  fi

  if npx cypress run $ARGS 2>&1 | tee e2e-test.log; then
    RESULTS["e2e"]="passed"
    echo -e "${GREEN}✓ E2E tests passed${NC}"
  else
    RESULTS["e2e"]="failed"
    echo -e "${RED}✗ E2E tests failed${NC}"
    EXIT_CODE=1
  fi

  # Cleanup servers if we started them
  if [ -n "$BACKEND_PID" ]; then
    kill $BACKEND_PID 2>/dev/null || true
  fi
  if [ -n "$FRONTEND_PID" ]; then
    kill $FRONTEND_PID 2>/dev/null || true
  fi

  cd "$PROJECT_ROOT"
}


# Print summary
print_summary() {
  echo ""
  echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
  echo -e "${BLUE}║                       Test Summary                             ║${NC}"
  echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
  echo ""

  for test in "${!RESULTS[@]}"; do
    case ${RESULTS[$test]} in
      passed)
        echo -e "  ${GREEN}✓${NC} $test: ${GREEN}PASSED${NC}"
        ;;
      failed)
        echo -e "  ${RED}✗${NC} $test: ${RED}FAILED${NC}"
        ;;
      skipped)
        echo -e "  ${YELLOW}○${NC} $test: ${YELLOW}SKIPPED${NC}"
        ;;
    esac
  done

  echo ""

  if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}═══ All tests passed! ═══${NC}"
  else
    echo -e "${RED}═══ Some tests failed! ═══${NC}"
  fi

  echo ""
  echo "Logs and reports:"
  echo "  - Backend unit: backend/backend-unit-test.log"
  echo "  - Frontend unit: frontend/frontend-unit-test.log"
  echo "  - Integration: backend/integration-test.log"
  echo "  - E2E: frontend/e2e-test.log"
}

# Main execution
case $TEST_TYPE in
  unit)
    run_backend_unit_tests
    run_frontend_unit_tests
    ;;
  backend)
    run_backend_unit_tests
    ;;
  frontend)
    run_frontend_unit_tests
    ;;
  integration)
    run_integration_tests
    ;;
  e2e)
    run_e2e_tests
    ;;
  all)
    run_backend_unit_tests
    echo ""
    run_frontend_unit_tests
    echo ""
    run_integration_tests
    echo ""
    run_e2e_tests
    ;;
  *)
    echo "Unknown test type: $TEST_TYPE"
    usage
    ;;
esac

print_summary

exit $EXIT_CODE
