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

## External Services

ServicePro uses a cost-optimized production stack that's **completely free** for early-stage validation (~50 MAU):

| Service                                               | Purpose             | Free Tier                    | Documentation                                                          |
| ----------------------------------------------------- | ------------------- | ---------------------------- | ---------------------------------------------------------------------- |
| [Fly.io](https://fly.io)                              | Backend hosting     | 3 shared VMs, 160GB outbound | [docs/services/fly-io.md](docs/services/fly-io.md)                     |
| [Cloudflare Pages](https://pages.cloudflare.com)      | Frontend hosting    | Unlimited requests           | [docs/services/cloudflare-pages.md](docs/services/cloudflare-pages.md) |
| [Neon](https://neon.tech)                             | PostgreSQL database | 0.5GB storage                | [docs/services/neon.md](docs/services/neon.md)                         |
| [Upstash](https://upstash.com)                        | Redis cache         | 10K commands/day             | [docs/services/upstash.md](docs/services/upstash.md)                   |
| [Cloudflare R2](https://developers.cloudflare.com/r2) | File storage        | 10GB, zero egress            | [docs/services/cloudflare-r2.md](docs/services/cloudflare-r2.md)       |
| [Resend](https://resend.com)                          | Transactional email | 3K emails/month              | [docs/services/resend.md](docs/services/resend.md)                     |
| [Stripe](https://stripe.com)                          | Payment processing  | No monthly fee               | [docs/services/stripe.md](docs/services/stripe.md)                     |

### Future Services (Coming Soon)

| Service                                              | Purpose                     | Status     |
| ---------------------------------------------------- | --------------------------- | ---------- |
| [Sentry](https://sentry.io)                          | Error tracking & monitoring | Planned    |
| [PostHog](https://posthog.com)                       | Product analytics           | Planned    |
| [Cloudflare Workers](https://workers.cloudflare.com) | Edge functions              | Planned    |
| [Twilio](https://twilio.com)                         | SMS notifications           | Planned    |
| [OpenAI](https://openai.com)                         | AI features                 | Evaluating |

For detailed service documentation, see [docs/services/](docs/services/).

## Deployment

See [docs/deployment/](docs/deployment/) for complete deployment guides.

### Quick Deploy (Backend)

```bash
# Install Fly CLI
brew install flyctl

# Login and deploy
fly auth login
cd backend
fly deploy --app servicepro-api
```

### Quick Deploy (Frontend)

Frontend auto-deploys via Cloudflare Pages when pushing to `master`.

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
