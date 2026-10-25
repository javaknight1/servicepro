# ServicePro

Field service management for small commercial service businesses — the field service platform that
gets you paid. Built with Go (backend) and React (frontend).

> **Live since 2026-10-24 (v0.9.0)** at <https://servicepro.sharkfins.xyz>: Cloudflare Worker
> `servicepro-web` (static assets + `/api/*` and `/version` proxy) → Cloud Run `servicepro-api`
> (us-east5) → Neon Postgres 18 (AWS us-east-2). Deploys run on `v*` tags. See
> [docs/deployment](docs/deployment/README.md).
>
> **MVP focus (decided 2026-10-25):** commercial cleaning and facility-maintenance companies
> (1–10 techs) billing property managers on net terms. Plan: [TODO.md](TODO.md) and
> [docs/product/mvp-plan-2026-10.md](docs/product/mvp-plan-2026-10.md).

## Overview

ServicePro provides tools for managing:

- **Customers** - CRM with full-text search, import/export
- **Jobs** - Work order management with scheduling
- **Quotes** - Quotes with send/accept/reject/revise, PDF and expiry
- **Invoices** - Invoices with payment terms, payment tracking, reminders and AR aging
- **Scheduling** - Technician assignment, calendar and conflict checks
- **Reports** - Revenue and customer analytics

## Tech Stack

### Backend

- **Language**: Go (version in `backend/go.mod`, currently `go 1.26`, toolchain go1.27.3)
- **Framework**: Gin
- **Database**: PostgreSQL 18 (Neon in production) with GORM; golang-migrate migrations in `backend/migrations/`
- **Cache**: Redis — optional (Upstash or none in production)
- **Authentication**: own JWT in httpOnly cookies + bcrypt today; Neon's managed Better Auth JWTs are
  already verified by the API and the SPA moves to it in T158
- **Email**: Resend (target, T159); SMTP/Mailpit locally
- **Storage**: S3-compatible (MinIO locally; Cloudflare R2 target)

### Frontend

- **Runtime**: Node.js 24 (`frontend/.nvmrc`)
- **Framework**: React 19 with TypeScript (strict)
- **Build Tool**: Vite 8
- **State**: Zustand + React Query
- **Styling**: Tailwind CSS 4

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go matching `backend/go.mod` (for running the backend on the host)
- Node.js 24 (`nvm use` in `frontend/`)
- `golangci-lint` (optional locally; `make install-golangci-lint`)

### Everything in Docker

```bash
cp backend/.env.example backend/.env   # first time only
make docker-dev                        # postgres, redis, mailpit, minio, api, web, grafana
```

The API container migrates its own database on startup
(`RUN_MIGRATIONS_ON_STARTUP=true`). Load test data with `make seed`
(login `dev@servicepro.local` / `password123`).

### Backend and frontend on the host

```bash
make setup          # npm ci + go mod download, git hooks, db containers, migrate, seed
make dev-backend    # terminal 1 — http://localhost:8080
make dev-frontend   # terminal 2 — http://localhost:3000
```

### Access the Application

- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- API Health: http://localhost:8080/health
- Swagger UI: http://localhost:8080/api/docs
- Mailpit: http://localhost:8025
- Grafana: http://localhost:3001 (admin/admin)

## Project Structure

```
servicepro/
├── backend/              # Go API (cmd/, internal/, migrations/, seeds/, docs/ = generated OpenAPI)
├── frontend/             # React app + Cloudflare Worker (wrangler.jsonc, worker/)
├── docs/                 # Developer documentation
│   ├── getting-started/  # Setup guides
│   ├── api/              # API reference
│   ├── architecture/     # System design
│   ├── deployment/       # Hosting overview
│   ├── ops/              # Cloud Run setup, deploy/rollback, backups
│   └── observability/    # Monitoring docs
├── scripts/              # bump-version, gh-release, deploy-cloudrun, git-hooks/
├── .github/workflows/    # CI, release, scheduled jobs, backups
├── help/                 # End-user documentation
├── training/             # Training materials
├── docker-compose.yml    # Local development stack
├── Makefile              # All build tasks (`make help`)
├── RELEASING.md          # Versioning + release process
└── VERSION               # Current release version
```

## Documentation

| Topic             | Location                                                                   |
| ----------------- | -------------------------------------------------------------------------- |
| **Quick Start**   | [docs/getting-started/quick-start.md](docs/getting-started/quick-start.md) |
| **Full Setup**    | [docs/getting-started/full-setup.md](docs/getting-started/full-setup.md)   |
| **API Reference** | [docs/api/](docs/api/)                                                     |
| **Architecture**  | [docs/architecture/](docs/architecture/)                                   |
| **Releasing**     | [RELEASING.md](RELEASING.md)                                               |
| **Deployment**    | [docs/deployment/](docs/deployment/README.md), [docs/ops/](docs/ops/)      |
| **Help Center**   | [help/](help/)                                                             |

## Development

Every task goes through the single root Makefile — run `make help` for the full list.

```bash
make install-hooks   # pre-commit (lint/test/build/OpenAPI drift) + commit-msg hooks
make test            # backend + frontend unit tests
make lint            # golangci-lint + eslint
make ci-local        # the core CI gates
make docs            # regenerate swagger + TypeScript API types (commit the result)
make migrate         # apply migrations to the local db
make migrate-create NAME=add_foo
make seed            # load backend/seeds/dev_seed.sql
```

## Releases

Commits use type prefixes (`feat:`, `fix:`, `deps:`, `chore:`, ...) — they are
the changelog. Releases ship from a `v*` tag; `make bump` picks the version.
History up to v0.7.0 is in [CHANGELOG.md](CHANGELOG.md); later releases are
GitHub Releases. See [RELEASING.md](RELEASING.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Proprietary - All rights reserved.
