# ServicePro

A modern service management platform built with React and Go, designed for efficient service delivery and customer management.

## Overview

ServicePro is a comprehensive platform that streamlines service operations, customer interactions, and business workflows. The platform combines a responsive React frontend with a robust Go backend, deployed on AWS infrastructure for scalability and reliability.

## Technology Stack

### Frontend

- **React** - Modern UI framework
- **TypeScript** - Type-safe JavaScript
- **Vite** - Fast build tool and dev server
- **React Router** - Client-side routing
- **Tailwind CSS** - Utility-first CSS framework
- **Axios** - HTTP client

### Backend

- **Go 1.21+** - High-performance backend language
- **Gorilla Mux** - HTTP router and dispatcher
- **PostgreSQL** - Primary database
- **Redis** - Caching layer
- **JWT** - Authentication

### Infrastructure

- **AWS ECS** - Container orchestration
- **AWS RDS** - Managed PostgreSQL
- **AWS ElastiCache** - Managed Redis
- **AWS CloudFront** - CDN
- **AWS S3** - Static asset storage
- **Terraform** - Infrastructure as Code
- **Docker** - Containerization

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go** 1.21 or higher
- **Node.js** 18.x or higher
- **npm** or **yarn** or **pnpm**
- **Docker** and **Docker Compose**
- **PostgreSQL** 14+ (for local development)
- **Redis** 7+ (for local development)
- **AWS CLI** (for deployment)
- **Terraform** 1.5+ (for infrastructure management)

## Setup Instructions

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/servicepro.git
cd servicepro
```

### 2. Backend Setup

```bash
# Navigate to the backend directory
cd backend

# Install Go dependencies
go mod download

# Copy environment template
cp .env.example .env

# Update .env with your local configuration
# Edit database connection strings, API keys, etc.

# Run database migrations
go run cmd/migrate/main.go up

# Start the backend server
go run cmd/api/main.go
```

The backend API will be available at `http://localhost:8080`

### 3. Frontend Setup

```bash
# Navigate to the frontend directory
cd frontend

# Install dependencies
npm install
# or
yarn install
# or
pnpm install

# Copy environment template
cp .env.example .env.local

# Update .env.local with your API endpoints

# Start the development server
npm run dev
# or
yarn dev
# or
pnpm dev
```

The frontend will be available at `http://localhost:5173`

### 4. Docker Setup (Alternative)

```bash
# From the root directory
docker-compose up -d

# This will start:
# - PostgreSQL database
# - Redis cache
# - Backend API
# - Frontend dev server
```

## Development Workflow

### Running Tests

**Backend Tests:**

```bash
cd backend
go test ./... -v
go test ./... -cover
```

**Frontend Tests:**

```bash
cd frontend
npm test
# or
npm run test:coverage
```

### Code Quality

**Backend:**

```bash
# Format code
go fmt ./...

# Lint
golangci-lint run

# Vet
go vet ./...
```

**Frontend:**

```bash
# Format code
npm run format

# Lint
npm run lint

# Type check
npm run type-check
```

### Building for Production

**Backend:**

```bash
cd backend
go build -o bin/api cmd/api/main.go
```

**Frontend:**

```bash
cd frontend
npm run build
```

### Database Migrations

```bash
# Create a new migration
go run cmd/migrate/main.go create migration_name

# Run migrations
go run cmd/migrate/main.go up

# Rollback last migration
go run cmd/migrate/main.go down

# Check migration status
go run cmd/migrate/main.go status
```

## Project Structure

```
servicepro/
├── backend/                # Go backend application
│   ├── cmd/               # Command-line applications
│   │   ├── api/          # Main API server
│   │   └── migrate/      # Database migration tool
│   ├── internal/         # Private application code
│   │   ├── api/         # API handlers and routes
│   │   ├── models/      # Data models
│   │   ├── services/    # Business logic
│   │   └── repository/  # Data access layer
│   ├── pkg/             # Public libraries
│   ├── migrations/      # Database migrations
│   └── config/          # Configuration files
│
├── frontend/            # React frontend application
│   ├── src/
│   │   ├── components/  # Reusable UI components
│   │   ├── pages/       # Page components
│   │   ├── services/    # API services
│   │   ├── hooks/       # Custom React hooks
│   │   ├── utils/       # Utility functions
│   │   ├── types/       # TypeScript types
│   │   └── assets/      # Static assets
│   ├── public/          # Public assets
│   └── tests/           # Test files
│
├── infrastructure/      # Infrastructure as Code
│   ├── terraform/      # Terraform configurations
│   │   ├── environments/
│   │   ├── modules/
│   │   └── backend.tf
│   └── docker/         # Docker configurations
│
├── docs/               # Documentation
├── scripts/            # Utility scripts
└── .github/            # GitHub workflows and templates
```

## Environment Variables

### Backend (.env)

```
DATABASE_URL=postgresql://user:password@localhost:5432/servicepro
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-secret-key
AWS_REGION=us-east-1
PORT=8080
```

### Frontend (.env.local)

```
VITE_API_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080/ws
```

## Deployment

### AWS Deployment

1. **Configure AWS credentials:**

```bash
aws configure
```

2. **Initialize Terraform:**

```bash
cd infrastructure/terraform
terraform init
```

3. **Plan infrastructure changes:**

```bash
terraform plan -var-file="environments/production.tfvars"
```

4. **Apply infrastructure:**

```bash
terraform apply -var-file="environments/production.tfvars"
```

5. **Deploy application:**

```bash
# Build and push Docker images
./scripts/build-and-push.sh

# Update ECS services
./scripts/deploy.sh production
```

## API Documentation

API documentation is available at:

- Development: `http://localhost:8080/api/docs`
- Production: `https://api.servicepro.com/docs`

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details on:

- Code of Conduct
- Development process
- Pull request process
- Coding standards
- Commit message conventions

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For support and questions:

- Create an issue in the GitHub repository
- Check the [documentation](docs/)
- Contact the team at support@servicepro.com

## Authors

- Development Team - [Your Organization]

## Acknowledgments

- Thanks to all contributors
- Built with modern best practices and industry standards
