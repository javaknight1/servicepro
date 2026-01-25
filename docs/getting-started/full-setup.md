# ServicePro Full Setup Guide

Complete instructions for setting up the ServicePro development environment with IDE configuration, all dependencies, and production-ready tooling.

## Prerequisites

### Required Software

| Software       | Minimum Version | Recommended | Purpose                       |
| -------------- | --------------- | ----------- | ----------------------------- |
| Go             | 1.21.0          | 1.22+       | Backend development           |
| Node.js        | 18.0.0          | 20.x LTS    | Frontend development          |
| npm            | 9.0.0           | 10.x        | Package management            |
| Docker         | 24.0.0          | Latest      | Containerization              |
| Docker Compose | 2.20.0          | Latest      | Multi-container orchestration |
| PostgreSQL     | 15.0            | 15.x        | Database (or use Docker)      |
| Redis          | 7.0             | 7.x         | Caching (or use Docker)       |
| Git            | 2.40.0          | Latest      | Version control               |
| Make           | 4.0             | Latest      | Build automation              |

### Optional Software

| Software      | Version | Purpose                |
| ------------- | ------- | ---------------------- |
| AWS CLI       | 2.x     | AWS deployment         |
| golangci-lint | 1.55+   | Go linting             |
| k6            | Latest  | Load testing           |
| mkcert        | Latest  | Local SSL certificates |

## Software Installation

### macOS (using Homebrew)

```bash
# Install Homebrew if not installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install required software
brew install go node docker docker-compose postgresql@15 redis git make

# Install optional tools
brew install awscli golangci-lint k6 mkcert

# Start Docker Desktop
open -a Docker

# Verify installations
go version          # Should show go1.21+
node --version      # Should show v18+
docker --version    # Should show 24+
docker compose version
```

### Ubuntu/Debian

```bash
# Update package list
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y curl wget git make build-essential

# Install Go
GO_VERSION=1.22.0
wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Node.js (via NodeSource)
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
newgrp docker

# Install Docker Compose
sudo apt install -y docker-compose-plugin

# Verify installations
go version
node --version
docker --version
docker compose version
```

### Windows (WSL2 Recommended)

```powershell
# Enable WSL2
wsl --install -d Ubuntu

# Then follow Ubuntu instructions inside WSL2
```

## IDE Configuration

### Visual Studio Code (Recommended)

#### Required Extensions

```json
{
  "recommendations": [
    "golang.go",
    "dbaeumer.vscode-eslint",
    "esbenp.prettier-vscode",
    "bradlc.vscode-tailwindcss",
    "ms-vscode.vscode-typescript-next",
    "mikestead.dotenv",
    "eamodio.gitlens",
    "ms-azuretools.vscode-docker",
    "redhat.vscode-yaml"
  ]
}
```

#### Workspace Settings

Create `.vscode/settings.json`:

```json
{
  "editor.formatOnSave": true,
  "editor.defaultFormatter": "esbenp.prettier-vscode",
  "editor.codeActionsOnSave": {
    "source.fixAll.eslint": "explicit",
    "source.organizeImports": "explicit"
  },
  "go.useLanguageServer": true,
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "[go]": {
    "editor.defaultFormatter": "golang.go"
  },
  "[typescript]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  },
  "[typescriptreact]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  }
}
```

## Repository Setup

```bash
# Clone the repository
git clone https://github.com/your-org/servicepro.git
cd servicepro

# Configure Git (if not already done)
git config user.name "Your Name"
git config user.email "your.email@example.com"
```

### Branch Strategy

We follow GitHub Flow:

```
master (main)
├── feature/SP-123-user-authentication
├── feature/SP-124-invoice-system
├── fix/SP-125-login-error
└── hotfix/SP-126-critical-bug
```

## Environment Configuration

### Backend Environment

Create `backend/.env` from template:

```bash
cp backend/.env.example backend/.env
```

Key settings:

```bash
# Application
APP_ENV=development
APP_PORT=8080

# Database
DATABASE_URL=postgresql://postgres:password@localhost:5432/servicepro?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379/0

# JWT
JWT_SECRET=your-super-secret-jwt-key-change-in-production
JWT_EXPIRY=3600

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# Logging
LOG_LEVEL=debug
```

### Frontend Environment

Create `frontend/.env`:

```bash
cp frontend/.env.example frontend/.env
```

Key settings:

```bash
VITE_API_URL=http://localhost:8080/api/v1
VITE_APP_ENV=development
```

## Database Setup

### Using Docker (Recommended)

```bash
# Start database services
docker compose up -d postgres redis

# Verify containers are running
docker compose ps

# Run migrations
make migrate
```

### Using Local PostgreSQL

```bash
# Create database user
sudo -u postgres createuser --interactive --pwprompt servicepro

# Create database
sudo -u postgres createdb -O servicepro servicepro_dev

# Run migrations
cd backend && go run cmd//main.go migrate up
```

## Running the Application

### Using Make (Recommended)

```bash
# Complete setup (first time)
make setup

# Start backend
make dev-backend

# Start frontend (separate terminal)
make dev-frontend
```

### Using Docker Compose

```bash
# Build and start all services
docker compose up --build

# Run in background
docker compose up -d

# View logs
docker compose logs -f
```

## Validation

### Check All Services

```bash
# 1. Check software versions
go version
node --version
docker --version

# 2. Check Docker services
docker compose ps

# 3. Check database connection
docker compose exec postgres pg_isready -U postgres

# 4. Check Redis connection
docker compose exec redis redis-cli ping

# 5. Check backend health
curl -s http://localhost:8080/health | jq .

# 6. Run backend tests
cd backend && go test ./... -count=1

# 7. Run frontend tests
cd frontend && npm test -- --run
```

### Health Check Endpoints

| Service        | URL                          | Expected Response       |
| -------------- | ---------------------------- | ----------------------- |
| Backend Health | http://localhost:8080/health | `{"status": "healthy"}` |
| Frontend       | http://localhost:3000        | HTML page               |
| PostgreSQL     | localhost:5432               | Connection success      |
| Redis          | localhost:6379               | `PONG`                  |

## SSL Certificates (Optional)

For local HTTPS development:

```bash
# Install mkcert
brew install mkcert  # macOS

# Install local CA
mkcert -install

# Generate certificates
mkdir -p certs && cd certs
mkcert localhost 127.0.0.1 ::1
```

## Quick Reference

### Useful Commands

```bash
# Development
make dev                # Start all services
make test              # Run all tests
make lint              # Run linters
make format            # Format code

# Database
make migrate           # Run migrations
make db-reset          # Reset database

# Docker
make docker-dev        # Start dev containers
make docker-down       # Stop containers
make docker-logs       # View logs

# Cleanup
make clean             # Clean build artifacts
```

### Default URLs

| Service     | URL                          |
| ----------- | ---------------------------- |
| Frontend    | http://localhost:3000        |
| Backend API | http://localhost:8080/api/v1 |
| PostgreSQL  | localhost:5432               |
| Redis       | localhost:6379               |

### Default Credentials

| Service    | Username               | Password  |
| ---------- | ---------------------- | --------- |
| PostgreSQL | postgres               | password  |
| Admin User | admin@servicepro.local | Admin123! |

## Next Steps

1. Review [API Documentation](../api/README.md)
2. Check [Architecture Overview](../architecture/overview.md)
3. Read [Contributing Guide](/CONTRIBUTING.md)
