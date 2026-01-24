#!/usr/bin/env bash

# =============================================================================
# ServicePro Development Environment Setup Script
# =============================================================================
# This script sets up the complete development environment for ServicePro.
# Run this script after cloning the repository.
#
# Usage: ./scripts/setup.sh [options]
#   --skip-docker    Skip Docker setup (use if running services locally)
#   --skip-deps      Skip dependency installation
#   --skip-db        Skip database initialization
#   --reset          Reset everything (WARNING: destructive)
#   --help           Show this help message
# =============================================================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default options
SKIP_DOCKER=false
SKIP_DEPS=false
SKIP_DB=false
RESET=false

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
}

check_command() {
    if ! command -v "$1" &> /dev/null; then
        return 1
    fi
    return 0
}

# =============================================================================
# Parse Arguments
# =============================================================================

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-docker)
            SKIP_DOCKER=true
            shift
            ;;
        --skip-deps)
            SKIP_DEPS=true
            shift
            ;;
        --skip-db)
            SKIP_DB=true
            shift
            ;;
        --reset)
            RESET=true
            shift
            ;;
        --help)
            head -20 "$0" | tail -14
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# =============================================================================
# Pre-flight Checks
# =============================================================================

print_header "ServicePro Development Setup"

log_info "Checking prerequisites..."

# Check required tools
MISSING_TOOLS=()

if ! check_command "go"; then
    MISSING_TOOLS+=("go")
fi

if ! check_command "node"; then
    MISSING_TOOLS+=("node")
fi

if ! check_command "npm"; then
    MISSING_TOOLS+=("npm")
fi

if ! check_command "docker"; then
    MISSING_TOOLS+=("docker")
fi

if ! check_command "docker-compose" && ! docker compose version &> /dev/null; then
    MISSING_TOOLS+=("docker-compose")
fi

if ! check_command "git"; then
    MISSING_TOOLS+=("git")
fi

if [ ${#MISSING_TOOLS[@]} -ne 0 ]; then
    log_error "Missing required tools: ${MISSING_TOOLS[*]}"
    log_info "Please install the missing tools and run this script again."
    log_info "See SETUP.md for installation instructions."
    exit 1
fi

log_success "All prerequisites found"

# Check versions
log_info "Checking tool versions..."

GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
NODE_VERSION=$(node --version | sed 's/v//')

log_info "Go version: $GO_VERSION"
log_info "Node version: $NODE_VERSION"

# Version checks (warn only, don't fail)
if [[ "$(echo "$GO_VERSION 1.21" | tr ' ' '\n' | sort -V | head -1)" != "1.21" ]]; then
    log_warning "Go version $GO_VERSION is below recommended 1.21+"
fi

if [[ "$(echo "$NODE_VERSION 18.0.0" | tr ' ' '\n' | sort -V | head -1)" != "18.0.0" ]]; then
    log_warning "Node version $NODE_VERSION is below recommended 18+"
fi

# =============================================================================
# Reset (if requested)
# =============================================================================

if [ "$RESET" = true ]; then
    print_header "Resetting Environment"

    log_warning "This will delete all local data including databases!"
    read -p "Are you sure you want to continue? (y/N) " -n 1 -r
    echo

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Reset cancelled"
        exit 0
    fi

    log_info "Stopping containers..."
    cd "$PROJECT_ROOT"
    docker compose down -v 2>/dev/null || true

    log_info "Removing node_modules..."
    rm -rf "$PROJECT_ROOT/frontend/node_modules"

    log_info "Removing Go module cache..."
    go clean -modcache 2>/dev/null || true

    log_info "Removing .env files..."
    rm -f "$PROJECT_ROOT/backend/.env"
    rm -f "$PROJECT_ROOT/frontend/.env"

    log_success "Environment reset complete"
fi

# =============================================================================
# Environment Files
# =============================================================================

print_header "Setting Up Environment Files"

# Backend .env
if [ ! -f "$PROJECT_ROOT/backend/.env" ]; then
    log_info "Creating backend/.env from template..."
    cp "$PROJECT_ROOT/backend/.env.example" "$PROJECT_ROOT/backend/.env"

    # Generate a random JWT secret
    if check_command "openssl"; then
        JWT_SECRET=$(openssl rand -base64 32)
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s/your-secret-key-change-this-in-production-min-32-chars/$JWT_SECRET/" "$PROJECT_ROOT/backend/.env"
        else
            sed -i "s/your-secret-key-change-this-in-production-min-32-chars/$JWT_SECRET/" "$PROJECT_ROOT/backend/.env"
        fi
        log_success "Generated random JWT secret"
    fi

    log_success "Created backend/.env"
else
    log_info "backend/.env already exists, skipping"
fi

# Frontend .env
if [ ! -f "$PROJECT_ROOT/frontend/.env" ]; then
    log_info "Creating frontend/.env from template..."
    cp "$PROJECT_ROOT/frontend/.env.example" "$PROJECT_ROOT/frontend/.env"
    log_success "Created frontend/.env"
else
    log_info "frontend/.env already exists, skipping"
fi

# =============================================================================
# Docker Setup
# =============================================================================

if [ "$SKIP_DOCKER" = false ]; then
    print_header "Setting Up Docker Services"

    log_info "Checking Docker daemon..."
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running. Please start Docker and try again."
        exit 1
    fi

    log_info "Pulling Docker images..."
    cd "$PROJECT_ROOT"
    docker compose pull

    log_info "Starting Docker services..."
    docker compose up -d postgres redis

    log_info "Waiting for services to be healthy..."
    "$SCRIPT_DIR/wait-for-services.sh"

    log_success "Docker services started"
fi

# =============================================================================
# Dependencies Installation
# =============================================================================

if [ "$SKIP_DEPS" = false ]; then
    print_header "Installing Dependencies"

    # Backend dependencies
    log_info "Installing Go dependencies..."
    cd "$PROJECT_ROOT/backend"
    go mod download
    go mod tidy
    log_success "Go dependencies installed"

    # Frontend dependencies
    log_info "Installing Node dependencies..."
    cd "$PROJECT_ROOT/frontend"
    npm install
    log_success "Node dependencies installed"
fi

# =============================================================================
# Database Initialization
# =============================================================================

if [ "$SKIP_DB" = false ] && [ "$SKIP_DOCKER" = false ]; then
    print_header "Initializing Database"

    log_info "Waiting for PostgreSQL to be ready..."
    until docker compose exec -T postgres pg_isready -U postgres &> /dev/null; do
        sleep 1
    done

    log_info "Creating databases..."
    docker compose exec -T postgres psql -U postgres -c "CREATE DATABASE servicepro;" 2>/dev/null || true
    docker compose exec -T postgres psql -U postgres -c "CREATE DATABASE servicepro_test;" 2>/dev/null || true

    log_info "Running migrations..."
    cd "$PROJECT_ROOT/backend"

    # Check if migrate command exists
    if [ -f "cmd/main.go" ]; then
        # Try to run migrations using the app
        go run cmd/main.go migrate up 2>/dev/null || {
            log_warning "Migration command not available, running SQL files directly..."
            for migration in migrations/*.sql; do
                if [ -f "$migration" ]; then
                    log_info "Applying $migration..."
                    docker compose exec -T postgres psql -U postgres -d servicepro < "$migration" 2>/dev/null || true
                fi
            done
        }
    fi

    log_success "Database initialized"

    # Seed data
    log_info "Seeding development data..."
    go run cmd/main.go seed 2>/dev/null || {
        log_warning "Seed command not available, skipping"
    }
fi

# =============================================================================
# Git Hooks Setup
# =============================================================================

print_header "Setting Up Git Hooks"

if [ -d "$PROJECT_ROOT/.git" ]; then
    log_info "Installing Git hooks..."

    # Pre-commit hook
    cat > "$PROJECT_ROOT/.git/hooks/pre-commit" << 'EOF'
#!/bin/bash
# Pre-commit hook for ServicePro

# Run Go formatting check
cd backend
go fmt ./... > /dev/null 2>&1
CHANGED=$(git diff --name-only)
if [ -n "$CHANGED" ]; then
    echo "Go files were reformatted. Please add them to your commit."
    echo "$CHANGED"
    exit 1
fi

# Run frontend linting
cd ../frontend
npm run lint --silent 2>/dev/null || true

exit 0
EOF
    chmod +x "$PROJECT_ROOT/.git/hooks/pre-commit"

    log_success "Git hooks installed"
else
    log_warning "Not a git repository, skipping hooks setup"
fi

# =============================================================================
# Validation
# =============================================================================

print_header "Validating Setup"

"$SCRIPT_DIR/validate.sh" --quiet || {
    log_warning "Some validation checks failed. See above for details."
}

# =============================================================================
# Complete
# =============================================================================

print_header "Setup Complete!"

echo -e "
${GREEN}ServicePro development environment is ready!${NC}

${BLUE}Quick Start:${NC}
  Backend:   cd backend && go run cmd/cmd/main.go
  Frontend:  cd frontend && npm run dev

  Or use Docker Compose:
  docker compose up

${BLUE}Useful URLs:${NC}
  Frontend:  http://localhost:5173
  Backend:   http://localhost:8080
  API Docs:  http://localhost:8080/swagger
  MailHog:   http://localhost:8025 (if running)

${BLUE}Default Credentials:${NC}
  Email:     admin@servicepro.local
  Password:  admin123

${BLUE}Next Steps:${NC}
  1. Review the configuration in backend/.env and frontend/.env
  2. Start the development servers
  3. Open http://localhost:5173 in your browser

${YELLOW}For troubleshooting, see TROUBLESHOOTING.md${NC}
"
