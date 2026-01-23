# ServicePro Backend

Go backend service for the ServicePro platform.

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin
- **Database**: PostgreSQL 15+ with GORM
- **Cache**: Redis
- **Auth**: JWT with bcrypt
- **Email**: AWS SES

## Project Structure

```
backend/
├── cmd/api/              # Application entry point
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
└── migrations/           # SQL migrations
```

## Quick Start

### Prerequisites

- Go 1.21+
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

# Run migrations
make migrate-up

# Run the application
go run cmd/api/main.go
```

## Environment Variables

| Variable     | Required | Description                        |
| ------------ | -------- | ---------------------------------- |
| PORT         | No       | Server port (default: 8080)        |
| ENV          | No       | development/production             |
| DATABASE_URL | Yes      | PostgreSQL connection string       |
| REDIS_URL    | Yes      | Redis connection string            |
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
