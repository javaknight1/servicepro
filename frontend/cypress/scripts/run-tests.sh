#!/bin/bash
# =============================================================================
# Cypress Test Runner Script
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
ENVIRONMENT="development"
BROWSER="chrome"
SPEC=""
HEADED=false
PARALLEL=false

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -e|--environment)
      ENVIRONMENT="$2"
      shift 2
      ;;
    -b|--browser)
      BROWSER="$2"
      shift 2
      ;;
    -s|--spec)
      SPEC="$2"
      shift 2
      ;;
    --headed)
      HEADED=true
      shift
      ;;
    --parallel)
      PARALLEL=true
      shift
      ;;
    -h|--help)
      echo "Usage: $0 [options]"
      echo ""
      echo "Options:"
      echo "  -e, --environment  Environment to run tests against (development|staging|production)"
      echo "  -b, --browser      Browser to use (chrome|firefox|electron)"
      echo "  -s, --spec         Specific spec file or pattern to run"
      echo "  --headed           Run tests in headed mode"
      echo "  --parallel         Run tests in parallel (requires Cypress Cloud)"
      echo "  -h, --help         Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

echo -e "${GREEN}==================================================${NC}"
echo -e "${GREEN}  ServicePro E2E Test Runner${NC}"
echo -e "${GREEN}==================================================${NC}"
echo ""
echo -e "${YELLOW}Environment:${NC} $ENVIRONMENT"
echo -e "${YELLOW}Browser:${NC} $BROWSER"
echo -e "${YELLOW}Headed:${NC} $HEADED"
if [ -n "$SPEC" ]; then
  echo -e "${YELLOW}Spec:${NC} $SPEC"
fi
echo ""

# Clean up old reports
echo -e "${YELLOW}Cleaning old reports...${NC}"
rm -rf cypress/reports/mochawesome/*.json
rm -rf cypress/reports/mochawesome/*.html
rm -rf cypress/reports/merged-report.html

# Build Cypress command
CYPRESS_CMD="npx cypress run"
CYPRESS_CMD="$CYPRESS_CMD --browser $BROWSER"
CYPRESS_CMD="$CYPRESS_CMD --env environment=$ENVIRONMENT"

if [ "$HEADED" = true ]; then
  CYPRESS_CMD="$CYPRESS_CMD --headed"
fi

if [ -n "$SPEC" ]; then
  CYPRESS_CMD="$CYPRESS_CMD --spec '$SPEC'"
fi

if [ "$PARALLEL" = true ]; then
  CYPRESS_CMD="$CYPRESS_CMD --parallel --record"
fi

# Run tests
echo -e "${YELLOW}Running tests...${NC}"
echo "Command: $CYPRESS_CMD"
echo ""

eval $CYPRESS_CMD || TEST_EXIT_CODE=$?

# Generate merged report
echo ""
echo -e "${YELLOW}Generating merged report...${NC}"
npx mochawesome-merge cypress/reports/mochawesome/*.json -o cypress/reports/merged-report.json
npx marge cypress/reports/merged-report.json --reportDir cypress/reports --reportFilename merged-report --inline

echo ""
if [ -z "$TEST_EXIT_CODE" ] || [ "$TEST_EXIT_CODE" -eq 0 ]; then
  echo -e "${GREEN}==================================================${NC}"
  echo -e "${GREEN}  All tests passed!${NC}"
  echo -e "${GREEN}==================================================${NC}"
else
  echo -e "${RED}==================================================${NC}"
  echo -e "${RED}  Some tests failed. Exit code: $TEST_EXIT_CODE${NC}"
  echo -e "${RED}==================================================${NC}"
fi

echo ""
echo -e "${YELLOW}Reports available at:${NC}"
echo "  - HTML Report: cypress/reports/merged-report.html"
echo "  - JSON Report: cypress/reports/merged-report.json"
echo "  - Screenshots: cypress/screenshots/"
echo "  - Videos: cypress/videos/"

exit ${TEST_EXIT_CODE:-0}
