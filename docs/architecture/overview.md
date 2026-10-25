# Architecture Overview

ServicePro system architecture, services, and design patterns.

## System Architecture

Production (live since v0.9.0, 2026-10-24):

```
Browser ── https://servicepro.sharkfins.xyz
   │
   ▼
Cloudflare Worker "servicepro-web"
   │  static assets (React SPA, frontend/dist) + SPA fallback
   │  /api/* and /version ──► proxied with X-SP-Client-IP + X-SP-Edge-Secret
   ▼
Cloud Run "servicepro-api" (us-east5) — Go / Gin
   │  Middleware (request ID, edge check, CORS, CSRF, auth, tenant, permissions, rate limits)
   │  Handlers ──► Services ──► Repositories (GORM)
   │
   ├──► Neon Postgres 18 (AWS us-east-2)        primary database
   ├──► Neon Auth (managed Better Auth)          JWTs verified via JWKS (SPA switch: T158)
   ├──► Redis (optional: Upstash or none)        rate limits, tokens; in-memory fallback
   ├──► Email: Resend (T159); SMTP/Mailpit locally
   ├──► Object storage: S3 API (Cloudflare R2 target; MinIO locally)
   └──► Stripe, Sentry, PostHog

GitHub Actions: v* tag ──► migrate (Neon) ──► deploy API ──► deploy Worker
                hourly ──► POST /internal/jobs/* on the Cloud Run URL
```

## Technology Stack

### Frontend

| Technology           | Purpose                      |
| -------------------- | ---------------------------- |
| React 19             | UI framework                 |
| TypeScript (strict)  | Type safety                  |
| Vite 8               | Build tool                   |
| React Router 7       | Routing                      |
| Zustand              | Client state                 |
| TanStack React Query | Server state                 |
| Tailwind CSS 4       | Styling                      |
| Cloudflare Worker    | Static assets + `/api` proxy |

### Backend

| Technology                 | Purpose                                                |
| -------------------------- | ------------------------------------------------------ |
| Go 1.26 (toolchain 1.27.3) | Language                                               |
| Gin                        | HTTP framework                                         |
| GORM                       | ORM (pgx/sqlc is the target)                           |
| golang-migrate             | Always-additive migrations, embedded in the binary     |
| JWT (httpOnly cookies)     | Own authentication today; Neon Auth JWTs also accepted |
| bcrypt                     | Password hashing (cost 12)                             |
| shopspring/decimal         | Money                                                  |

### Data and services

| Technology            | Purpose                                                        |
| --------------------- | -------------------------------------------------------------- |
| PostgreSQL 18         | Primary database (Neon in production, Docker locally)          |
| Redis (optional)      | Rate limits, short-lived tokens; in-memory fallback without it |
| Resend                | Email (target, T159); Mailpit locally                          |
| S3-compatible storage | Files (Cloudflare R2 target; MinIO locally)                    |
| Stripe                | Payments (Stripe Connect planned, T169)                        |

## Backend Architecture

### Layer Structure

```
backend/
├── cmd/              # main.go (server), migrate/ (CLI), invite/ (invite codes)
├── internal/
│   ├── api/
│   │   ├── handlers/     # HTTP handlers (controllers)
│   │   ├── middleware/   # Auth, CORS, logging
│   │   └── routes/       # Route definitions
│   ├── models/           # Data models, DTOs
│   ├── repository/       # Data access layer
│   └── services/         # Business logic
├── pkg/
│   ├── auth/             # JWT, cookies, passwords, Neon Auth verifier
│   ├── clients/          # email, storage, sms, analytics, logging, metrics, error tracking
│   └── database/         # Postgres + Redis connections
└── migrations/           # Database migrations
```

### Request Flow

```
HTTP Request
    │
    ▼
┌─────────────────┐
│   Middleware    │  (CORS, Auth, Logging)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│    Handler      │  (Input validation, HTTP response)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│    Service      │  (Business logic, validation)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Repository    │  (Database operations)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│    Database     │
└─────────────────┘
```

## Frontend Architecture

### Structure

```
frontend/
├── src/
│   ├── components/       # Reusable UI components
│   ├── pages/            # Page components
│   ├── hooks/            # Custom React hooks
│   ├── services/         # API clients (axios, same-origin /api)
│   ├── store/            # Zustand stores
│   ├── providers/        # React providers
│   ├── routes/           # Route definitions
│   ├── types/            # TypeScript types (api.generated.ts from OpenAPI)
│   └── utils/            # Utility functions
├── public/               # Static assets, _headers (CSP/HSTS/cache)
├── worker/               # Cloudflare Worker: /api/* + /version proxy
├── wrangler.jsonc        # Worker config (servicepro-web)
└── vite.config.ts        # Build configuration
```

### State Management

- **Zustand** for client state (UI, preferences)
- **React Query** for server state (API data)

## Key Design Patterns

### Backend

- **Repository Pattern** - Abstracts data access
- **Service Layer** - Business logic isolation
- **Dependency Injection** - Service composition
- **Middleware Chain** - Cross-cutting concerns

### Frontend

- **Container/Presentational** - Logic/UI separation
- **Custom Hooks** - Reusable logic
- **Context Providers** - Shared state
- **Error Boundaries** - Error handling

## API Design

### RESTful Endpoints

```
GET    /api/v1/{resource}           # List
GET    /api/v1/{resource}/:id       # Get one
POST   /api/v1/{resource}           # Create
PUT    /api/v1/{resource}/:id       # Update
DELETE /api/v1/{resource}/:id       # Delete
```

### Response Format

Single resources are returned as the object itself. Lists return a named array
plus paging fields, e.g. `GET /api/v1/customers`:

```json
{ "customers": [ ... ], "total": 100, "page": 1, "page_size": 20, "total_pages": 5 }
```

Errors use one envelope `{error, message, request_id, fields?}` — see
[docs/api/errors.md](../api/errors.md). The OpenAPI spec (`backend/docs/`,
Swagger UI at `/api/docs`) is the source of truth for shapes.

## Security

### Authentication

- Own JWTs in httpOnly cookies (access + refresh), bcrypt password hashing (cost 12)
- Mutating cookie-authenticated requests must send `X-Requested-With: XMLHttpRequest` (CSRF)
- Neon Auth (managed Better Auth) bearer JWTs are verified via JWKS; the SPA moves to it in T158

### Authorization

- Per-tenant roles and permissions (a role in tenant A grants nothing in tenant B)
- Tenant comes from the authenticated membership (`X-Tenant-ID` only selects among the caller's tenants)
- Every tenant-owned query filters `tenant_id`; cross-tenant reads return 404

### Data Protection

- Input validation on all endpoints
- Parameterized queries (SQL injection prevention)
- XSS prevention via JSON encoding
- CORS configuration
- Rate limiting

## Performance

### Caching Strategy

- Redis (when configured) for rate limits and short-lived tokens; in-memory per-instance fallback otherwise
- Report caches with explicit refresh endpoints
- HTTP caching headers on static assets (`frontend/public/_headers`)

### Database Optimization

- Connection pooling
- Indexed queries
- Pagination for lists

## Scalability

### Horizontal Scaling

- Stateless API on Cloud Run (scales to zero; `max-instances × DB_MAX_OPEN_CONNS` stays under the DB limit)
- Background work as idempotent `POST /internal/jobs/<name>` calls from a scheduler, not in-process tickers
- Static assets served from Cloudflare's edge

### Planned (see TODO.md Phase 4)

- chi + pgx/sqlc, Postgres RLS as a tenant-isolation backstop
- TanStack Query for all server state
