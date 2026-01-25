# Architecture Overview

ServicePro system architecture, services, and design patterns.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                         │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────────┐ │
│  │  Pages  │  │ Comps   │  │ Hooks   │  │    State (Zustand)  │ │
│  └────┬────┘  └────┬────┘  └────┬────┘  └──────────┬──────────┘ │
│       └────────────┴────────────┴──────────────────┘            │
│                              │                                   │
│                    ┌─────────┴─────────┐                        │
│                    │   API Services    │                        │
│                    └─────────┬─────────┘                        │
└──────────────────────────────┼──────────────────────────────────┘
                               │ HTTP/REST
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Backend (Go/Gin)                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Handlers   │──│  Services   │──│      Repositories       │  │
│  │   (API)     │  │  (Business) │  │       (Data Access)     │  │
│  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘  │
│         │                │                     │                 │
│  ┌──────┴──────┐  ┌──────┴──────┐             │                 │
│  │ Middleware  │  │  External   │             │                 │
│  │ Auth, CORS  │  │  Services   │             │                 │
│  └─────────────┘  └─────────────┘             │                 │
└───────────────────────────────────────────────┼─────────────────┘
                                                │
                    ┌───────────────────────────┼───────────────┐
                    │                           │               │
                    ▼                           ▼               ▼
             ┌──────────┐               ┌──────────┐     ┌──────────┐
             │PostgreSQL│               │  Redis   │     │   AWS    │
             │ Database │               │  Cache   │     │   SES    │
             └──────────┘               └──────────┘     └──────────┘
```

## Technology Stack

### Frontend

| Technology   | Purpose          |
| ------------ | ---------------- |
| React 18     | UI framework     |
| TypeScript   | Type safety      |
| Vite         | Build tool       |
| React Router | Routing          |
| Zustand      | State management |
| React Query  | Server state     |
| Tailwind CSS | Styling          |

### Backend

| Technology         | Purpose                |
| ------------------ | ---------------------- |
| Go 1.21+           | Language               |
| Gin                | HTTP framework         |
| GORM               | ORM                    |
| JWT                | Authentication         |
| bcrypt             | Password hashing       |
| shopspring/decimal | Financial calculations |

### Data Layer

| Technology    | Purpose           |
| ------------- | ----------------- |
| PostgreSQL 15 | Primary database  |
| Redis 7       | Caching, sessions |
| AWS SES       | Email delivery    |

## Backend Architecture

### Layer Structure

```
backend/
├── cmd//          # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/     # HTTP handlers (controllers)
│   │   ├── middleware/   # Auth, CORS, logging
│   │   └── routes/       # Route definitions
│   ├── models/           # Data models, DTOs
│   ├── repository/       # Data access layer
│   └── services/         # Business logic
├── pkg/
│   ├── auth/             # Authentication utilities
│   ├── database/         # Database connections
│   └── email/            # Email services
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
│   ├── services/         # API clients
│   ├── store/            # State management
│   ├── contexts/         # React contexts
│   ├── types/            # TypeScript types
│   └── utils/            # Utility functions
├── public/               # Static assets
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

```json
{
  "data": { ... },
  "message": "Success",
  "meta": {
    "page": 1,
    "total": 100
  }
}
```

## Security

### Authentication

- JWT tokens with short expiry
- Refresh token rotation
- bcrypt password hashing (cost 12)

### Authorization

- Role-based access control (RBAC)
- Permission-based endpoints
- Resource-level authorization

### Data Protection

- Input validation on all endpoints
- Parameterized queries (SQL injection prevention)
- XSS prevention via JSON encoding
- CORS configuration
- Rate limiting

## Performance

### Caching Strategy

- Redis for session data
- Redis for tax rates, permissions
- HTTP caching headers

### Database Optimization

- Connection pooling
- Indexed queries
- Pagination for lists

## Scalability

### Horizontal Scaling

- Stateless backend (JWT auth)
- Redis for shared state
- Database connection pooling

### Future Considerations

- Message queue for async operations
- CDN for static assets
- Read replicas for reporting
