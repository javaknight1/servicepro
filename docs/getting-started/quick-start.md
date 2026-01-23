# ServicePro Quick Start Guide

Get up and running with ServicePro in 5 minutes.

## Prerequisites

- Docker & Docker Compose
- Go 1.23+
- Node.js 18+
- Make (optional but recommended)

## Fastest Way to Start

### Option 1: One Command Setup (Recommended)

```bash
make setup
```

This will:

- Install all dependencies (backend & frontend)
- Set up pre-commit hooks
- Start PostgreSQL and Redis
- Run database migrations

Then in **two separate terminals**:

**Terminal 1 - Backend:**

```bash
make dev-backend
```

**Terminal 2 - Frontend:**

```bash
make dev-frontend
```

Open http://localhost:3000 in your browser.

---

### Option 2: Docker Compose (Everything in Docker)

```bash
# Start all services at once
docker-compose up

# Or run in background
docker-compose up -d
```

This starts:

- PostgreSQL (port 5432)
- Redis (port 6379)
- Backend (port 8080)
- Frontend (port 5173)

Open http://localhost:5173 in your browser.

---

### Option 3: Step-by-Step Setup

```bash
# 1. Start databases
docker-compose up -d postgres redis

# 2. Install dependencies
cd frontend && npm install && cd ..
cd backend && go mod download && cd ..

# 3. Create backend .env (if not exists)
cd backend && cp .env.example .env && cd ..

# 4. Run migrations
make migrate

# 5. Start backend (in one terminal)
cd backend && go run cmd/api/cmd/main.go

# 6. Start frontend (in another terminal)
cd frontend && npm run dev
```

Open http://localhost:3000 in your browser.

---

## Useful Make Commands

```bash
make help          # Show all available commands

# Development
make dev           # Start databases (then use dev-backend and dev-frontend)
make dev-db        # Start only PostgreSQL and Redis
make dev-backend   # Run backend server
make dev-frontend  # Run frontend dev server
make migrate       # Run database migrations

# Docker
make up            # Start all services with Docker Compose
make down          # Stop all services
make logs          # View all logs
make ps            # Show running containers

# Setup
make setup         # Complete first-time setup
make install-deps  # Install all dependencies

# Quality
make test          # Run all tests
make lint          # Run all linters
make format        # Format all code

# Database
make db-reset      # Reset database (WARNING: deletes all data)
```

## Testing the Application

### 1. Register a New Account

1. Go to http://localhost:3000
2. Click "Get Started" or "Sign Up"
3. Enter email and password
4. Click "Create Account"

### 2. Verify Email (Development Mode)

In development, verification emails aren't sent. To verify manually:

```bash
# Connect to database
docker exec -it servicepro-postgres psql -U postgres -d servicepro

# Get the verification token
SELECT email, token FROM email_verifications ORDER BY created_at DESC LIMIT 1;

# Copy the token and visit:
# http://localhost:3000/verify-email?token=<TOKEN>
```

### 3. Login and Explore

1. Go to http://localhost:3000/login
2. Enter your credentials
3. Explore the dashboard, settings, and features

## Check Service Status

```bash
# All services
docker-compose ps

# Backend health check
curl http://localhost:8080/health

# PostgreSQL
docker exec servicepro-postgres pg_isready -U postgres

# Redis
docker exec servicepro-redis redis-cli ping
```

## Common Issues

### Port Already in Use?

```bash
# Check what's using each port
lsof -i :3000   # Frontend
lsof -i :8080   # Backend
lsof -i :5432   # PostgreSQL
lsof -i :6379   # Redis

# Stop local services if needed
brew services stop postgresql  # macOS
brew services stop redis       # macOS
```

### Database Connection Failed?

```bash
# Check PostgreSQL is running
docker-compose ps postgres

# View logs
docker-compose logs postgres

# Restart PostgreSQL
docker-compose restart postgres
```

### Frontend Can't Reach Backend?

1. Verify backend is running: `curl http://localhost:8080/health`
2. Check CORS in `backend/.env`:
   ```
   CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
   ```
3. Check frontend proxy in `frontend/vite.config.ts`

See [Troubleshooting](./troubleshooting.md) for more solutions.

## Project Structure

```
servicepro/
├── backend/           # Go backend API
│   ├── cmd/          # Main applications
│   ├── internal/     # Private application code
│   ├── pkg/          # Public libraries
│   └── migrations/   # Database migrations
├── frontend/          # React frontend
│   ├── src/
│   │   ├── components/  # UI components
│   │   ├── pages/       # Page components
│   │   ├── services/    # API services
│   │   └── store/       # State management
│   └── public/
└── docker-compose.yml # Docker services
```

## Service URLs

| Service        | URL                          |
| -------------- | ---------------------------- |
| Frontend       | http://localhost:3000        |
| Backend API    | http://localhost:8080        |
| Backend Health | http://localhost:8080/health |
| PostgreSQL     | localhost:5432               |
| Redis          | localhost:6379               |

## Default Credentials

**Database:**

- Host: localhost
- Port: 5432
- User: postgres
- Password: password
- Database: servicepro

**Redis:**

- Host: localhost
- Port: 6379
- No password

## Next Steps

1. Review the [Full Setup Guide](./full-setup.md) for IDE configuration
2. Explore the [API Documentation](../api/README.md)
3. Check the [Architecture Overview](../architecture/overview.md)
