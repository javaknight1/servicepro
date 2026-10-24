# ServicePro Backend

Go backend service for the ServicePro platform.

## Tech Stack

- **Language**: Go 1.26+
- **Framework**: Gin
- **Database**: PostgreSQL 15+ with GORM
- **Cache**: Redis (optional; in-memory fallback)
- **Auth**: JWT with bcrypt
- **Email**: AWS SES

## Project Structure

```
backend/
├── cmd/                  # API server entry point
│   └── migrate/          # golang-migrate CLI (up/down/version/force)
├── config/               # Configuration management
├── internal/
│   ├── api/
│   │   ├── handlers/     # HTTP request handlers
│   │   ├── middleware/   # Auth, rate limiting, CORS
│   │   └── routes/       # Route definitions
│   ├── models/           # Domain models (GORM)
│   ├── repository/       # Data access layer
│   └── services/         # Business logic
├── pkg/
│   ├── auth/             # JWT, password utilities
│   ├── database/         # DB connections
│   └── email/            # Email service
├── migrations/           # Numbered golang-migrate migrations (embedded)
└── seeds/                # Dev-only seed data (not a migration)
```

## Quick Start

### Prerequisites

- Go 1.26+
- PostgreSQL 15+
- Redis 7+
- Docker (optional)

### With Docker (Recommended)

```bash
# From project root
make dev          # Start all services
make logs         # View logs
make stop         # Stop services
```

### Manual Setup

```bash
# Install dependencies
go mod download

# Set environment variables (see .env.example)
export DATABASE_URL=postgresql://postgres:password@localhost:5432/servicepro?sslmode=disable
export REDIS_URL=redis://localhost:6379
export JWT_SECRET=dev-secret-key-min-32-characters-long

# Run migrations (also applied automatically at startup in development)
go run ./cmd/migrate up
go run ./cmd/migrate version

# Run the application
go run cmd/main.go
```

## Environment Variables

| Variable     | Required | Description                        |
| ------------ | -------- | ---------------------------------- |
| PORT         | No       | Server port (default: 8080)        |
| ENV          | No       | development/production             |
| DATABASE_URL | Yes      | PostgreSQL connection string       |
| REDIS_URL    | No       | Redis URL (in-memory fallback)     |
| JWT_SECRET   | Yes      | Secret for JWT signing (32+ chars) |
| FRONTEND_URL | No       | Frontend URL for CORS              |

See `.env.example` for full list.

## API Endpoints

Base URL: `http://localhost:8080/api/v1`

| Method | Endpoint              | Description            |
| ------ | --------------------- | ---------------------- |
| POST   | /auth/register        | User registration      |
| POST   | /auth/login           | User login             |
| POST   | /auth/forgot-password | Request password reset |
| GET    | /customers            | List customers         |
| POST   | /customers            | Create customer        |
| GET    | /jobs                 | List jobs              |
| POST   | /jobs                 | Create job             |
| GET    | /invoices             | List invoices          |
| POST   | /invoices             | Create invoice         |

For full API documentation, see [API Reference](/docs/api/).

## Development

### Running Tests

```bash
make test         # Run all tests
make test-cover   # Run with coverage
```

### Code Style

- Use `gofmt` for formatting
- Follow standard Go conventions
- All errors must be handled (no ignored returns)
- Use context.Context for cancellation

### Database Conventions

- Primary Keys: UUID
- Soft Deletes: `deleted_at` column
- Timestamps: `created_at`, `updated_at`

## Documentation

| Topic           | Location                                         |
| --------------- | ------------------------------------------------ |
| Architecture    | [/docs/architecture/](/docs/architecture/)       |
| API Reference   | [/docs/api/](/docs/api/)                         |
| Getting Started | [/docs/getting-started/](/docs/getting-started/) |
| Features        | [/docs/features/](/docs/features/)               |

## Migrations

Schema changes are numbered, additive golang-migrate files in `migrations/`
(`NNNNNN_name.up.sql` + `.down.sql`), embedded into both binaries.
`000001_initial_schema` is the whole pre-launch schema (the former
`001_schema.sql` plus 000002–000014, squashed 2026-10-24). A database built
from the old `001_schema.sql`, or a local one migrated before the squash, is
refused or out of step — rebuild it with `make db-fresh-seed`.

```bash
go run ./cmd/migrate up          # apply pending migrations
go run ./cmd/migrate version     # applied version + dirty flag
go run ./cmd/migrate force N     # repair a dirty schema after fixing it by hand
go run ./cmd/migrate down        # drop everything (refused when ENV=production)
```

`RUN_MIGRATIONS_ON_STARTUP` (default true only in development) applies them
at server start. In deployed environments the release pipeline runs
`migrate up` before deploying; `/health/ready` returns 503 while the schema is
behind or dirty.

## System endpoints

| Method | Path                               | Notes                                                    |
| ------ | ---------------------------------- | -------------------------------------------------------- |
| GET    | /version                           | `{"git_sha","app_version"}` from GIT_SHA / APP_VERSION   |
| GET    | /health, /health/live, /health/ready | readiness includes schema and redis state              |
| POST   | /internal/jobs/payment-reminders   | header `X-Internal-Jobs-Token: $INTERNAL_JOBS_SECRET`   |
