# ServicePro Quick Start Guide

Get up and running with ServicePro locally in 5 minutes. (Production lives at
<https://servicepro.sharkfins.xyz> and ships only from `v*` tags — see
[RELEASING.md](../../RELEASING.md).)

## Prerequisites

- Docker & Docker Compose
- Go matching `backend/go.mod` (`go 1.26`, toolchain go1.27.3)
- Node.js 24 (`frontend/.nvmrc`; `nvm use` in `frontend/`)
- Make

Redis is optional — without `REDIS_URL` the API falls back to an in-memory,
per-instance store. The local compose stack starts one anyway.

## Fastest Way to Start

### Option 1: One Command Setup (Recommended)

```bash
cp backend/.env.example backend/.env   # first time only
make setup
```

This will:

- Install all dependencies (backend & frontend)
- Install the versioned hooks (`make install-hooks`)
- Start PostgreSQL, Redis, MinIO and Mailpit containers
- Run database migrations and load the dev seed (login `dev@servicepro.local` / `password123`)

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
cp backend/.env.example backend/.env   # first time only
make docker-dev                        # foreground; make docker-dev-d to detach
make seed                              # optional test data
```

This starts PostgreSQL (5432), Redis (6379), Mailpit (8025), MinIO, the backend
(8080, migrates its own database on startup), the frontend (3000) and Grafana
(3001).

Open http://localhost:3000 in your browser.

---

### Option 3: Step-by-Step Setup

```bash
# 1. Start databases (PostgreSQL, Redis, MinIO, Mailpit)
make dev-db

# 2. Install dependencies
make install-deps

# 3. Create backend .env (if not exists)
cp backend/.env.example backend/.env

# 4. Run migrations and load seed data
make migrate
make seed

# 5. Start backend (in one terminal)
make dev-backend

# 6. Start frontend (in another terminal)
make dev-frontend
```

Open http://localhost:3000 in your browser.

---

## Useful Make Commands

```bash
make help          # Show all available commands

# Development
make dev           # Start databases (then use dev-backend and dev-frontend)
make dev-db        # Start PostgreSQL, Redis, MinIO and Mailpit
make dev-backend   # Run backend server
make dev-frontend  # Run frontend dev server
make migrate       # Run database migrations
make seed          # Load backend/seeds/dev_seed.sql

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
make ci-local      # The core CI gates

# Database
make db-reset      # Reset database (WARNING: deletes all data)
make db-fresh-seed # Recreate the local db and load the seed
```

## Testing the Application

### 1. Log in with the seeded account

After `make seed`, log in at http://localhost:3000/login as
`dev@servicepro.local` / `password123`.

### 2. Or register a new account

Local development defaults to open registration (`REGISTRATION_MODE`; deployed
environments default to `invite_only`).

1. Go to http://localhost:3000
2. Click "Get Started" or "Sign Up"
3. Enter email and password
4. Click "Create Account"

Locally, email goes to Mailpit: open http://localhost:8025 and click the
verification link.

### 3. Explore

Explore the dashboard, customers, jobs, quotes, invoices and settings.

## Check Service Status

```bash
# All services
make ps

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
lsof -i :8025   # Mailpit

# Stop local services if needed
brew services stop postgresql  # macOS
brew services stop redis       # macOS
```

### Database Connection Failed?

```bash
# Check PostgreSQL is running
docker compose ps postgres

# View logs
docker compose logs postgres

# Restart PostgreSQL
docker compose restart postgres
```

### Frontend Can't Reach Backend?

1. Verify backend is running: `curl http://localhost:8080/health`
2. Check CORS in `backend/.env`:
   ```
   CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
   ```
3. Check the dev proxy in `frontend/vite.config.ts` (the SPA calls same-origin `/api`)

See [Troubleshooting](./troubleshooting.md) for more solutions.

## Project Structure

```
servicepro/
├── backend/           # Go backend API
│   ├── cmd/          # main.go (server), migrate/, invite/
│   ├── internal/     # Private application code
│   ├── pkg/          # Clients, auth, database
│   ├── migrations/   # golang-migrate migrations (always additive)
│   └── seeds/        # dev_seed.sql
├── frontend/          # React frontend + Cloudflare Worker
│   ├── src/
│   │   ├── components/  # UI components
│   │   ├── pages/       # Page components
│   │   ├── services/    # API services
│   │   └── store/       # State management
│   ├── worker/          # /api + /version edge proxy
│   └── public/
└── docker-compose.yml # Docker services
```

## Service URLs

| Service        | URL                            |
| -------------- | ------------------------------ |
| Frontend       | http://localhost:3000          |
| Backend API    | http://localhost:8080          |
| Backend Health | http://localhost:8080/health   |
| Swagger UI     | http://localhost:8080/api/docs |
| Mailpit        | http://localhost:8025          |
| Grafana        | http://localhost:3001          |
| PostgreSQL     | localhost:5432                 |
| Redis          | localhost:6379                 |

## Default Credentials

**App (after `make seed`):** `dev@servicepro.local` / `password123`

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
