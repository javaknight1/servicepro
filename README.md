# ServicePro

A comprehensive service management platform for field service businesses. Built with Go (backend) and React (frontend).

## Table of Contents

- [Overview](#overview)
- [Tech Stack](#tech-stack)
- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [Development](#development)
- [API Documentation](#api-documentation)
- [Deployment](#deployment)
- [Contributing](#contributing)

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
- **Framework**: Gin (HTTP router)
- **ORM**: GORM
- **Database**: PostgreSQL 15+
- **Cache**: Redis
- **Authentication**: JWT with refresh tokens
- **Email**: AWS SES

### Frontend

- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite
- **State Management**: Zustand (client state) + React Query (server state)
- **Styling**: Tailwind CSS
- **HTTP Client**: Axios

### Infrastructure

- **Cloud**: AWS (EKS, RDS, ElastiCache, S3, SES)
- **Container Orchestration**: Kubernetes
- **IaC**: Terraform
- **CI/CD**: GitHub Actions + ArgoCD

## Getting Started

### Prerequisites

- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL 15+ (or use Docker)
- Redis (or use Docker)

### Quick Start

1. **Clone the repository**
   \`\`\`bash
   git clone https://github.com/javaknight1/servicepro.git
   cd servicepro
   \`\`\`

2. **Start dependencies with Docker**
   \`\`\`bash
   docker-compose up -d postgres redis
   \`\`\`

3. **Setup backend**
   \`\`\`bash
   cd backend
   cp .env.example .env

   # Edit .env with your configuration

   go mod download
   make migrate
   make run
   \`\`\`

4. **Setup frontend**
   \`\`\`bash
   cd frontend
   cp .env.example .env
   npm install
   npm run dev
   \`\`\`

5. **Access the application**
   - Frontend: http://localhost:3000
   - Backend API: http://localhost:8080
   - API Health: http://localhost:8080/health

## Project Structure

\`\`\`
servicepro/
├── backend/ # Go backend
│ ├── cmd/api/ # Application entry point
│ ├── config/ # Configuration management
│ ├── internal/
│ │ ├── api/
│ │ │ ├── handlers/ # HTTP handlers
│ │ │ ├── middleware/ # Auth, rate limiting, etc.
│ │ │ ├── routes/ # Route definitions
│ │ │ └── validators/ # Request validation
│ │ ├── models/ # Domain models
│ │ ├── repository/ # Data access layer
│ │ └── services/ # Business logic
│ ├── migrations/ # Database migrations
│ ├── pkg/ # Shared packages
│ └── templates/ # Email/PDF templates
│
├── frontend/ # React frontend
│ ├── src/
│ │ ├── components/ # Reusable UI components
│ │ ├── pages/ # Page components
│ │ ├── services/ # API service layer
│ │ ├── store/ # Zustand stores
│ │ ├── hooks/ # Custom React hooks
│ │ ├── types/ # TypeScript types
│ │ └── utils/ # Utility functions
│ └── public/ # Static assets
│
├── infrastructure/
│ └── terraform/ # AWS infrastructure
│
├── gitops/ # ArgoCD configurations
│ └── values/ # Environment-specific values
│
├── docker-compose.yml # Local development
└── Makefile # Build commands
\`\`\`

## Development

### Backend Commands

\`\`\`bash
cd backend

# Run the server

make run

# Run tests

make test

# Run linting

make lint

# Run migrations

make migrate

# Generate API docs

make docs
\`\`\`

### Frontend Commands

\`\`\`bash
cd frontend

# Development server

npm run dev

# Build for production

npm run build

# Run tests

npm run test

# Lint code

npm run lint

# Type check

npm run typecheck
\`\`\`

### Docker Compose

\`\`\`bash

# Start all services

docker-compose up -d

# Start specific services

docker-compose up -d postgres redis

# View logs

docker-compose logs -f backend

# Stop all services

docker-compose down
\`\`\`

## API Documentation

### Authentication

All authenticated endpoints require a JWT token in the Authorization header:
\`\`\`
Authorization: Bearer <token>
\`\`\`

### Base URL

\`\`\`
Development: http://localhost:8080/api/v1
Production: https://api.servicepro.com/api/v1
\`\`\`

### Key Endpoints

| Method | Endpoint             | Description       |
| ------ | -------------------- | ----------------- |
| POST   | \`/auth/login\`      | User login        |
| POST   | \`/auth/register\`   | User registration |
| GET    | \`/customers\`       | List customers    |
| POST   | \`/customers\`       | Create customer   |
| GET    | \`/jobs\`            | List jobs         |
| POST   | \`/jobs\`            | Create job        |
| GET    | \`/quotes\`          | List quotes       |
| POST   | \`/quotes\`          | Create quote      |
| GET    | \`/invoices\`        | List invoices     |
| POST   | \`/invoices\`        | Create invoice    |
| GET    | \`/reports/revenue\` | Revenue report    |

### Error Responses

\`\`\`json
{
"error": "Error type",
"message": "Human-readable message",
"details": {}
}
\`\`\`

## Deployment

### Environments

| Environment     | Purpose                   |
| --------------- | ------------------------- |
| \`development\` | Local development         |
| \`staging\`     | Testing and QA            |
| \`preprod\`     | Pre-production validation |
| \`production\`  | Live environment          |

### Infrastructure Setup

\`\`\`bash
cd infrastructure/terraform

# Initialize

make init ENV=staging

# Plan changes

make plan ENV=staging

# Apply changes

make apply ENV=staging
\`\`\`

### Kubernetes Deployment

Deployments are managed via ArgoCD with Helm charts.

\`\`\`bash

# Deploy to staging

kubectl apply -f gitops/applications/staging.yaml

# Check deployment status

kubectl get pods -n servicepro-staging
\`\`\`

## Contributing

1. Create a feature branch from \`master\`
2. Make your changes
3. Run tests and linting
4. Submit a pull request

### Code Style

- **Go**: Follow standard Go conventions, use \`gofmt\`
- **TypeScript**: Follow ESLint/Prettier configuration
- **Commits**: Use conventional commit messages

### Pull Request Guidelines

- Include tests for new functionality
- Update documentation as needed
- Ensure CI passes before requesting review

## License

Proprietary - All rights reserved.

## Support

For questions or issues, contact the development team.
