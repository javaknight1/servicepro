#!/usr/bin/env bash

# =============================================================================
# ServicePro Environment Validation Script
# =============================================================================
# Validates that the development environment is correctly configured.
#
# Usage: ./scripts/validate.sh [options]
#   --quiet    Only show errors
#   --fix      Attempt to fix issues automatically
#   --help     Show this help message
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Options
QUIET=false
FIX=false

# Counters
CHECKS_PASSED=0
CHECKS_FAILED=0
CHECKS_WARNED=0

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    if [ "$QUIET" = false ]; then
        echo -e "${BLUE}[INFO]${NC} $1"
    fi
}

log_success() {
    if [ "$QUIET" = false ]; then
        echo -e "${GREEN}[PASS]${NC} $1"
    fi
    ((CHECKS_PASSED++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((CHECKS_WARNED++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((CHECKS_FAILED++))
}

check_command() {
    if command -v "$1" &> /dev/null; then
        return 0
    fi
    return 1
}

check_version() {
    local current="$1"
    local required="$2"

    if [[ "$(printf '%s\n' "$required" "$current" | sort -V | head -n1)" == "$required" ]]; then
        return 0
    fi
    return 1
}

# =============================================================================
# Parse Arguments
# =============================================================================

while [[ $# -gt 0 ]]; do
    case $1 in
        --quiet)
            QUIET=true
            shift
            ;;
        --fix)
            FIX=true
            shift
            ;;
        --help)
            head -15 "$0" | tail -9
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# =============================================================================
# Header
# =============================================================================

if [ "$QUIET" = false ]; then
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE} ServicePro Environment Validation${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
fi

# =============================================================================
# Required Tools
# =============================================================================

log_info "Checking required tools..."

# Go
if check_command "go"; then
    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    if check_version "$GO_VERSION" "1.21"; then
        log_success "Go $GO_VERSION installed"
    else
        log_warning "Go $GO_VERSION installed (1.21+ recommended)"
    fi
else
    log_error "Go is not installed"
fi

# Node.js
if check_command "node"; then
    NODE_VERSION=$(node --version | sed 's/v//')
    if check_version "$NODE_VERSION" "18.0.0"; then
        log_success "Node.js $NODE_VERSION installed"
    else
        log_warning "Node.js $NODE_VERSION installed (18+ recommended)"
    fi
else
    log_error "Node.js is not installed"
fi

# npm
if check_command "npm"; then
    NPM_VERSION=$(npm --version)
    log_success "npm $NPM_VERSION installed"
else
    log_error "npm is not installed"
fi

# Docker
if check_command "docker"; then
    DOCKER_VERSION=$(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    log_success "Docker $DOCKER_VERSION installed"
else
    log_error "Docker is not installed"
fi

# Docker Compose
if docker compose version &> /dev/null; then
    COMPOSE_VERSION=$(docker compose version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    log_success "Docker Compose $COMPOSE_VERSION installed"
elif check_command "docker-compose"; then
    COMPOSE_VERSION=$(docker-compose --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    log_success "Docker Compose $COMPOSE_VERSION installed"
else
    log_error "Docker Compose is not installed"
fi

# Git
if check_command "git"; then
    GIT_VERSION=$(git --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')
    log_success "Git $GIT_VERSION installed"
else
    log_error "Git is not installed"
fi

# =============================================================================
# Environment Files
# =============================================================================

log_info "Checking environment files..."

# Backend .env
if [ -f "$PROJECT_ROOT/backend/.env" ]; then
    log_success "backend/.env exists"

    # Check for default/empty values
    if grep -q "your-secret-key-change-this" "$PROJECT_ROOT/backend/.env" 2>/dev/null; then
        log_warning "backend/.env contains default JWT_SECRET (should be changed for production)"
    fi

    # Check required variables
    REQUIRED_VARS=(DATABASE_URL JWT_SECRET)
    for var in "${REQUIRED_VARS[@]}"; do
        if grep -q "^${var}=" "$PROJECT_ROOT/backend/.env"; then
            VALUE=$(grep "^${var}=" "$PROJECT_ROOT/backend/.env" | cut -d'=' -f2-)
            if [ -n "$VALUE" ]; then
                log_success "backend/.env: $var is set"
            else
                log_error "backend/.env: $var is empty"
            fi
        else
            log_error "backend/.env: $var is missing"
        fi
    done
else
    log_error "backend/.env does not exist"
    if [ "$FIX" = true ]; then
        log_info "Creating backend/.env from template..."
        cp "$PROJECT_ROOT/backend/.env.example" "$PROJECT_ROOT/backend/.env"
        log_success "Created backend/.env"
    fi
fi

# Frontend .env
if [ -f "$PROJECT_ROOT/frontend/.env" ]; then
    log_success "frontend/.env exists"

    # Check required variables
    if grep -q "^VITE_API_URL=" "$PROJECT_ROOT/frontend/.env"; then
        log_success "frontend/.env: VITE_API_URL is set"
    else
        log_error "frontend/.env: VITE_API_URL is missing"
    fi
else
    log_error "frontend/.env does not exist"
    if [ "$FIX" = true ]; then
        log_info "Creating frontend/.env from template..."
        cp "$PROJECT_ROOT/frontend/.env.example" "$PROJECT_ROOT/frontend/.env"
        log_success "Created frontend/.env"
    fi
fi

# =============================================================================
# Docker Services
# =============================================================================

log_info "Checking Docker services..."

# Check if Docker is running
if docker info &> /dev/null; then
    log_success "Docker daemon is running"

    # Check if containers are running
    cd "$PROJECT_ROOT"

    # PostgreSQL
    if docker compose ps --status running 2>/dev/null | grep -q postgres; then
        log_success "PostgreSQL container is running"

        # Test connection
        if docker compose exec -T postgres pg_isready -U postgres &> /dev/null; then
            log_success "PostgreSQL is accepting connections"
        else
            log_error "PostgreSQL is not accepting connections"
        fi
    else
        log_warning "PostgreSQL container is not running"
        if [ "$FIX" = true ]; then
            log_info "Starting PostgreSQL container..."
            docker compose up -d postgres
        fi
    fi

    # Redis
    if docker compose ps --status running 2>/dev/null | grep -q redis; then
        log_success "Redis container is running"

        # Test connection
        if docker compose exec -T redis redis-cli ping 2>/dev/null | grep -q PONG; then
            log_success "Redis is responding to ping"
        else
            log_error "Redis is not responding"
        fi
    else
        log_warning "Redis container is not running"
        if [ "$FIX" = true ]; then
            log_info "Starting Redis container..."
            docker compose up -d redis
        fi
    fi
else
    log_warning "Docker daemon is not running"
fi

# =============================================================================
# Dependencies
# =============================================================================

log_info "Checking dependencies..."

# Backend dependencies
if [ -d "$PROJECT_ROOT/backend" ]; then
    cd "$PROJECT_ROOT/backend"

    if [ -f "go.sum" ]; then
        log_success "Go dependencies are downloaded (go.sum exists)"
    else
        log_warning "Go dependencies may not be downloaded (go.sum missing)"
        if [ "$FIX" = true ]; then
            log_info "Downloading Go dependencies..."
            go mod download
            go mod tidy
        fi
    fi
fi

# Frontend dependencies
if [ -d "$PROJECT_ROOT/frontend" ]; then
    cd "$PROJECT_ROOT/frontend"

    if [ -d "node_modules" ]; then
        log_success "Node dependencies are installed (node_modules exists)"
    else
        log_warning "Node dependencies are not installed"
        if [ "$FIX" = true ]; then
            log_info "Installing Node dependencies..."
            npm install
        fi
    fi
fi

# =============================================================================
# Port Availability
# =============================================================================

log_info "Checking port availability..."

check_port() {
    local port=$1
    local name=$2

    if lsof -Pi :$port -sTCP:LISTEN -t &> /dev/null; then
        # Port is in use, check if it's our service
        PROCESS=$(lsof -Pi :$port -sTCP:LISTEN | tail -1 | awk '{print $1}')
        if [[ "$PROCESS" == "docker"* ]] || [[ "$PROCESS" == "com.docke"* ]]; then
            log_success "Port $port is in use by Docker ($name)"
        else
            log_warning "Port $port is in use by $PROCESS (expected for $name)"
        fi
    else
        log_success "Port $port is available ($name)"
    fi
}

check_port 3000 "Frontend (alternate)"
check_port 5173 "Frontend (Vite)"
check_port 8080 "Backend API"
check_port 5432 "PostgreSQL"
check_port 6379 "Redis"

# =============================================================================
# Database
# =============================================================================

log_info "Checking database..."

if docker compose ps --status running 2>/dev/null | grep -q postgres; then
    cd "$PROJECT_ROOT"

    # Check if database exists
    if docker compose exec -T postgres psql -U postgres -lqt 2>/dev/null | cut -d \| -f 1 | grep -qw servicepro; then
        log_success "Database 'servicepro' exists"
    else
        log_warning "Database 'servicepro' does not exist"
        if [ "$FIX" = true ]; then
            log_info "Creating database..."
            docker compose exec -T postgres psql -U postgres -c "CREATE DATABASE servicepro;"
        fi
    fi

    # Check for test database
    if docker compose exec -T postgres psql -U postgres -lqt 2>/dev/null | cut -d \| -f 1 | grep -qw servicepro_test; then
        log_success "Test database 'servicepro_test' exists"
    else
        log_warning "Test database 'servicepro_test' does not exist"
        if [ "$FIX" = true ]; then
            log_info "Creating test database..."
            docker compose exec -T postgres psql -U postgres -c "CREATE DATABASE servicepro_test;"
        fi
    fi
fi

# =============================================================================
# Network Connectivity
# =============================================================================

log_info "Checking network connectivity..."

# Check if backend is reachable (if running)
if curl -s --connect-timeout 2 http://localhost:8080/health &> /dev/null; then
    log_success "Backend health endpoint is reachable"
else
    log_info "Backend is not running (this is OK during setup)"
fi

# Check if frontend is reachable (if running)
if curl -s --connect-timeout 2 http://localhost:5173 &> /dev/null; then
    log_success "Frontend is reachable"
else
    log_info "Frontend is not running (this is OK during setup)"
fi

# =============================================================================
# Summary
# =============================================================================

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE} Validation Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "  ${GREEN}Passed:${NC}  $CHECKS_PASSED"
echo -e "  ${YELLOW}Warnings:${NC} $CHECKS_WARNED"
echo -e "  ${RED}Failed:${NC}  $CHECKS_FAILED"
echo ""

if [ $CHECKS_FAILED -gt 0 ]; then
    echo -e "${RED}Some checks failed. Please fix the issues above.${NC}"
    echo -e "Run with --fix to attempt automatic fixes."
    exit 1
elif [ $CHECKS_WARNED -gt 0 ]; then
    echo -e "${YELLOW}All critical checks passed, but there are warnings.${NC}"
    exit 0
else
    echo -e "${GREEN}All checks passed! Environment is ready.${NC}"
    exit 0
fi
