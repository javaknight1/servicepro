# ServicePro

A comprehensive service management platform for field service businesses. Built with Go (backend) and React (frontend).

## Overview

ServicePro provides tools for managing:

- **Customers** - CRM with full-text search, import/export
- **Jobs** - Work order management with scheduling
- **Quotes** - Quote generation with templates
- **Invoices** - Invoice generation with payment tracking
- **Scheduling** - Technician scheduling with conflict detection
- **Reports** - Revenue and customer analytics

## Tech Stack

### Backend

- **Language**: Go 1.21+
- **Framework**: Gin
- **Database**: PostgreSQL 15+ with GORM
- **Cache**: Redis
- **Authentication**: JWT with bcrypt

### Frontend

- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite
- **State**: Zustand + React Query
- **Styling**: Tailwind CSS

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for backend development)
- Node.js 18+ (for frontend development)

### Start with Docker

```bash
# Start all services
make dev

# View logs
make logs

# Stop services
make stop
```

### Access the Application

- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- API Health: http://localhost:8080/health

## Project Structure

```
servicepro/
├── backend/              # Go backend service
├── frontend/             # React frontend app
├── docs/                 # Developer documentation
│   ├── getting-started/  # Setup guides
│   ├── api/              # API reference
│   ├── architecture/     # System design
│   ├── features/         # Feature docs
│   └── observability/    # Monitoring docs
├── help/                 # End-user documentation
├── training/             # Training materials
├── docker-compose.yml    # Local development
└── Makefile              # Build commands
```

## Documentation

| Topic             | Location                                                                   |
| ----------------- | -------------------------------------------------------------------------- |
| **Quick Start**   | [docs/getting-started/quick-start.md](docs/getting-started/quick-start.md) |
| **Full Setup**    | [docs/getting-started/full-setup.md](docs/getting-started/full-setup.md)   |
| **API Reference** | [docs/api/](docs/api/)                                                     |
| **Architecture**  | [docs/architecture/](docs/architecture/)                                   |
| **Help Center**   | [help/](help/)                                                             |

## Development

### Backend

```bash
cd backend
make run        # Start server
make test       # Run tests
make lint       # Lint code
```

### Frontend

```bash
cd frontend
npm run dev     # Development server
npm run build   # Production build
npm run test    # Run tests
```

### Useful Make Commands

```bash
make dev        # Start all services with Docker
make stop       # Stop all services
make logs       # View service logs
make test       # Run all tests
make lint       # Run linters
make clean      # Clean build artifacts
```

## Deployment

> **Coming Soon**: Deployment documentation will be added once hosting infrastructure is finalized.

## Contributing

1. Create a feature branch from `master`
2. Make your changes
3. Run tests and linting
4. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

Proprietary - All rights reserved.

## Support

- **Help Center**: [/help](help/)
- **Email**: support@servicepro.com
