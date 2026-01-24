#!/usr/bin/env bash

# =============================================================================
# ServicePro Diagnostic Script
# =============================================================================
# Collects diagnostic information for troubleshooting issues.
#
# Usage: ./scripts/diagnose.sh [options]
#   --output FILE    Save output to file (default: stdout)
#   --full           Include sensitive information (logs, env vars)
#   --help           Show this help message
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Options
OUTPUT_FILE=""
FULL_MODE=false

# =============================================================================
# Helper Functions
# =============================================================================

section() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

subsection() {
    echo ""
    echo -e "${BLUE}▸ $1${NC}"
    echo ""
}

run_cmd() {
    local cmd="$1"
    local description="$2"

    if [ -n "$description" ]; then
        echo -e "${YELLOW}$ $description${NC}"
    else
        echo -e "${YELLOW}$ $cmd${NC}"
    fi

    eval "$cmd" 2>&1 || echo -e "${RED}Command failed${NC}"
    echo ""
}

check_command() {
    if command -v "$1" &> /dev/null; then
        return 0
    fi
    return 1
}

# =============================================================================
# Parse Arguments
# =============================================================================

while [[ $# -gt 0 ]]; do
    case $1 in
        --output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        --full)
            FULL_MODE=true
            shift
            ;;
        --help)
            head -14 "$0" | tail -8
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# =============================================================================
# Output Handling
# =============================================================================

if [ -n "$OUTPUT_FILE" ]; then
    exec > >(tee "$OUTPUT_FILE")
fi

# =============================================================================
# Diagnostic Collection
# =============================================================================

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║        ServicePro Diagnostic Report                         ║${NC}"
echo -e "${GREEN}║        Generated: $(date '+%Y-%m-%d %H:%M:%S')                       ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"

# -----------------------------------------------------------------------------
# System Information
# -----------------------------------------------------------------------------

section "System Information"

subsection "Operating System"
run_cmd "uname -a"

if [[ "$OSTYPE" == "darwin"* ]]; then
    run_cmd "sw_vers"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    run_cmd "cat /etc/os-release 2>/dev/null | head -5"
fi

subsection "CPU & Memory"
if [[ "$OSTYPE" == "darwin"* ]]; then
    run_cmd "sysctl -n machdep.cpu.brand_string"
    run_cmd "sysctl -n hw.memsize | awk '{print \$0/1024/1024/1024 \" GB\"}'"
else
    run_cmd "grep 'model name' /proc/cpuinfo | head -1"
    run_cmd "free -h 2>/dev/null | head -2"
fi

subsection "Disk Space"
run_cmd "df -h . | head -2"

# -----------------------------------------------------------------------------
# Development Tools
# -----------------------------------------------------------------------------

section "Development Tools"

subsection "Go"
if check_command "go"; then
    run_cmd "go version"
    run_cmd "go env GOROOT GOPATH"
else
    echo -e "${RED}Go is not installed${NC}"
fi

subsection "Node.js"
if check_command "node"; then
    run_cmd "node --version"
    run_cmd "npm --version"
else
    echo -e "${RED}Node.js is not installed${NC}"
fi

subsection "Docker"
if check_command "docker"; then
    run_cmd "docker --version"
    run_cmd "docker compose version 2>/dev/null || docker-compose --version"
    run_cmd "docker info 2>/dev/null | grep -E '(Server Version|Storage Driver|Operating System)'"
else
    echo -e "${RED}Docker is not installed${NC}"
fi

subsection "Git"
if check_command "git"; then
    run_cmd "git --version"
    run_cmd "git config user.name 2>/dev/null || echo 'Not configured'"
    run_cmd "git config user.email 2>/dev/null || echo 'Not configured'"
fi

# -----------------------------------------------------------------------------
# Docker Status
# -----------------------------------------------------------------------------

section "Docker Status"

if docker info &> /dev/null; then
    subsection "Running Containers"
    run_cmd "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"

    subsection "Container Resource Usage"
    run_cmd "docker stats --no-stream --format 'table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}' 2>/dev/null || echo 'No containers running'"

    subsection "Docker Networks"
    run_cmd "docker network ls --filter name=servicepro"

    subsection "Docker Volumes"
    run_cmd "docker volume ls --filter name=servicepro"

    subsection "Docker System Info"
    run_cmd "docker system df"
else
    echo -e "${RED}Docker daemon is not running${NC}"
fi

# -----------------------------------------------------------------------------
# Service Health
# -----------------------------------------------------------------------------

section "Service Health"

cd "$PROJECT_ROOT"

subsection "PostgreSQL"
if docker compose ps 2>/dev/null | grep -q postgres; then
    run_cmd "docker compose exec -T postgres pg_isready -U postgres"
    run_cmd "docker compose exec -T postgres psql -U postgres -c 'SELECT version();' 2>/dev/null | head -3"
    run_cmd "docker compose exec -T postgres psql -U postgres -c '\\l' 2>/dev/null | grep servicepro"
    run_cmd "docker compose exec -T postgres psql -U postgres -c 'SELECT count(*) FROM pg_stat_activity;' 2>/dev/null"
else
    echo -e "${YELLOW}PostgreSQL container is not running${NC}"
fi

subsection "Redis"
if docker compose ps 2>/dev/null | grep -q redis; then
    run_cmd "docker compose exec -T redis redis-cli ping"
    run_cmd "docker compose exec -T redis redis-cli info server 2>/dev/null | grep -E '(redis_version|uptime)'"
    run_cmd "docker compose exec -T redis redis-cli info memory 2>/dev/null | grep -E '(used_memory_human|maxmemory_human)'"
else
    echo -e "${YELLOW}Redis container is not running${NC}"
fi

subsection "Backend API"
if curl -s --connect-timeout 2 http://localhost:8080/health &> /dev/null; then
    run_cmd "curl -s http://localhost:8080/health | head -20"
else
    echo -e "${YELLOW}Backend is not responding${NC}"
fi

subsection "Frontend"
if curl -s --connect-timeout 2 http://localhost:5173 &> /dev/null; then
    echo -e "${GREEN}Frontend is responding${NC}"
else
    echo -e "${YELLOW}Frontend is not responding${NC}"
fi

# -----------------------------------------------------------------------------
# Port Status
# -----------------------------------------------------------------------------

section "Port Status"

subsection "Ports in Use"
run_cmd "lsof -i :3000 -i :5173 -i :8080 -i :5432 -i :6379 2>/dev/null | grep LISTEN || echo 'No relevant ports in use'"

# -----------------------------------------------------------------------------
# Project Status
# -----------------------------------------------------------------------------

section "Project Status"

subsection "Git Status"
cd "$PROJECT_ROOT"
run_cmd "git status --short 2>/dev/null | head -20"
run_cmd "git log --oneline -5 2>/dev/null"
run_cmd "git branch -v 2>/dev/null | head -5"

subsection "Environment Files"
echo "backend/.env: $([ -f backend/.env ] && echo 'exists' || echo 'missing')"
echo "frontend/.env: $([ -f frontend/.env ] && echo 'exists' || echo 'missing')"
echo "docker/.env: $([ -f docker/.env ] && echo 'exists' || echo 'missing')"

subsection "Dependencies"
echo "backend/go.sum: $([ -f backend/go.sum ] && echo 'exists' || echo 'missing')"
echo "frontend/node_modules: $([ -d frontend/node_modules ] && echo 'exists' || echo 'missing')"

# -----------------------------------------------------------------------------
# Logs (Full Mode Only)
# -----------------------------------------------------------------------------

if [ "$FULL_MODE" = true ]; then
    section "Recent Logs"

    subsection "Docker Compose Logs (last 50 lines)"
    cd "$PROJECT_ROOT"
    run_cmd "docker compose logs --tail=50 2>/dev/null" "docker compose logs --tail=50"

    subsection "Environment Variables (redacted)"
    echo "Backend .env (keys only):"
    if [ -f "$PROJECT_ROOT/backend/.env" ]; then
        grep -v '^#' "$PROJECT_ROOT/backend/.env" | grep '=' | cut -d'=' -f1 | sort
    fi
    echo ""
    echo "Frontend .env (keys only):"
    if [ -f "$PROJECT_ROOT/frontend/.env" ]; then
        grep -v '^#' "$PROJECT_ROOT/frontend/.env" | grep '=' | cut -d'=' -f1 | sort
    fi
fi

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------

section "Diagnostic Summary"

echo "The diagnostic report has been generated."
echo ""

if [ -n "$OUTPUT_FILE" ]; then
    echo -e "${GREEN}Report saved to: $OUTPUT_FILE${NC}"
    echo ""
fi

echo "Common issues to check:"
echo "  1. Is Docker daemon running?"
echo "  2. Are all required containers running?"
echo "  3. Do .env files exist with correct values?"
echo "  4. Are ports 5173, 8080, 5432, 6379 available?"
echo "  5. Are dependencies installed (node_modules, go.sum)?"
echo ""
echo "For help, see TROUBLESHOOTING.md or run:"
echo "  ./scripts/validate.sh --fix"
