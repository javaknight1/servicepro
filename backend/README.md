# ServicePro Backend

## Project Overview

ServicePro is an **all-in-one operations platform** for small to medium-sized service businesses (1-50 employees). Built with Go, it's designed to replace 2-3 existing tools (scheduling + CRM + invoicing) with one integrated, affordable solution.

**Target Market**: HVAC, plumbing, electrical services, maintenance companies, and other field service businesses.

**Value Proposition**:

- 20-30% more affordable than competitors (Jobber, Housecall Pro, FieldPulse)
- Simpler interface with all features included (no expensive add-ons)
- Complete workflow automation from quote to payment

**Pricing Strategy**:

- Starter: $29/month (up to 2 users, 50 jobs/month)
- Professional: $59/month (up to 5 users, unlimited jobs)
- Business: $99/month (up to 15 users, advanced features)

**6-Month MVP Development Timeline**: Single developer, part-time (20 hours/week)

## Tech Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Database**: PostgreSQL 15+ with GORM ORM
- **Cache/Sessions**: Redis
- **Authentication**: JWT with bcrypt password hashing
- **Email**: AWS SES
- **Architecture**: Clean Architecture with repository pattern

## Project Structure

```
backend/
├── cmd/api/              # Application entry point
├── config/               # Configuration management
├── internal/
│   ├── api/
│   │   ├── handlers/     # HTTP request handlers
│   │   ├── middleware/   # HTTP middleware (auth, rate limiting)
│   │   └── routes/       # Route definitions
│   ├── models/           # Domain models with GORM tags
│   ├── repository/       # Data access layer (GORM)
│   └── services/         # Business logic layer
├── pkg/
│   ├── auth/             # Authentication utilities (JWT, password hashing)
│   ├── database/         # Database connections (GORM, Redis)
│   └── email/            # Email service (AWS SES)
├── migrations/           # SQL database migrations
└── docs/                 # Technical documentation
```

## Core Features

### ✅ Implemented Features

1. **Authentication & Authorization**

   - User registration with email verification
   - JWT-based authentication
   - Password reset flow
   - Rate limiting on auth endpoints
   - Role-Based Access Control (RBAC)

2. **Email System**

   - Email verification on registration
   - Password reset emails
   - Reminder emails for unverified accounts
   - AWS SES bounce handling
   - Configurable email templates

3. **Role-Based Access Control**

   - Hierarchical role system
   - Resource-based permissions (resource.action)
   - Permission inheritance
   - Temporary role assignments
   - Default roles: super_admin, admin, manager, user, guest

4. **Security**
   - Bcrypt password hashing
   - JWT token management
   - Account locking after failed login attempts
   - Email verification
   - Rate limiting
   - Soft deletes for data retention

### 🚧 MVP Features (In Development)

**Customer Relationship Management (CRM)**:

- Customer database with contact info, service/billing addresses
- Service history tracking with photos, parts used, labor hours
- Communication logs (phone, email, SMS) with timestamps
- Customer ratings and feedback (1-5 stars)
- Customer status tracking (Active/Inactive/Prospect)

**Job Management & Scheduling**:

- Job creation with service type, priority, estimated duration
- Drag-and-drop calendar scheduling (week/day views)
- Technician assignment with skills-based matching
- Job status workflow (New → Scheduled → In Progress → Completed → Invoiced)
- Recurring jobs support (weekly, monthly, quarterly, annually)
- Photo uploads at each job stage

**Quoting & Invoicing System**:

- Line-item quotes with parts/labor breakdown
- Tax calculation based on customer location
- Electronic signature capture for quote approval
- One-click invoice generation from completed jobs
- Payment terms (Net 15/30/45/60)
- PDF generation with company branding

**Reporting Dashboard**:

- Revenue analytics (MRR, YoY comparison, by service type)
- Operational metrics (jobs per technician, completion rates)
- Customer analytics (LTV, churn, geographic distribution)
- Chart.js visualizations with date range filtering
- Export to PDF and Excel

**Automated Communications**:

- Email notifications (quotes, invoices, appointment reminders, overdue payments)
- SMS notifications (appointment confirmations, technician arrival)
- Customizable templates with company branding
- Delivery tracking and failure notifications

**Payment Processing**:

- Stripe integration for online payments
- Multiple payment methods
- Partial payment handling
- Automatic payment confirmation emails

### ❌ NOT in MVP (Future Features)

Deliberately excluded to stay within 6-month timeline:

- Mobile applications (iOS/Android)
- Advanced routing/GPS optimization
- Marketing automation campaigns
- Inventory management system
- QuickBooks integration
- Multi-location/multi-tenancy support
- Public API for third-party integrations
- Real-time technician tracking
- Customer self-service portal

## Success Metrics & Business Goals

### 12-Month MVP Targets (Conservative)

**Customer Acquisition**:

- Target: 10-15 paying customers by month 12
- Trial → Paid conversion: 10-20%
- Monthly churn rate: <15%

**Revenue Goals**:

- Month 6: $100-200 MRR
- Month 12: $500-1,000 MRR
- Average Revenue Per User (ARPU): $35-45
- Break-even timeline: Months 15-18

**Product Adoption**:

- Daily Active Users: 40-60% of customers
- Jobs created per customer: 20+ per month
- Invoice generation: 80%+ of completed jobs

**System Performance**:

- Application uptime: 99.0% minimum (target 99.5%)
- API response time: <500ms for 95% of requests
- Page load times: <3 seconds

### Validation Criteria

**Proceed with continued development if**:

- ✅ 8+ active paying customers by month 9
- ✅ Sustained 10%+ monthly growth for 3 consecutive months
- ✅ Customer retention >65% monthly
- ✅ Customers using 4+ core features regularly

**Pivot or discontinue if**:

- ❌ <5 paying customers by month 9
- ❌ Monthly churn rate >25% consistently
- ❌ Major technical scalability issues

### Operating Costs

**Monthly Infrastructure** (AWS):

- ECS Fargate + Load Balancer: ~$68/month
- RDS PostgreSQL + ElastiCache Redis: ~$40/month
- S3 + CloudFront + Route 53: ~$14/month
- CloudWatch monitoring: ~$15/month
- SES + SNS (email/SMS): ~$20/month
- **Total: ~$157/month**

**Year 1 Total Investment**: ~$6,100 (infrastructure + tools + marketing)

## Development Conventions

### Database

- **ORM**: GORM (not raw SQL)
- **Migrations**: SQL files in `migrations/` directory
- **Primary Keys**: UUID (not auto-increment integers)
- **Soft Deletes**: All main entities use `gorm.DeletedAt`
- **Timestamps**: All tables have `created_at` and `updated_at`

### Code Style

- **Error Handling**: Always return errors, don't panic
- **Context**: Use context.Context for cancellation
- **Logging**: Use `log.Printf` for now (structured logging later)
- **Testing**: Unit tests for all business logic
- **Validation**: Validate at service layer, not handlers

### API Design

- **Versioning**: `/api/v1/*` prefix
- **REST**: Follow RESTful conventions
- **JSON**: All requests/responses use JSON
- **Error Format**: `{"error": "code", "message": "description"}`
- **Success Format**: Return relevant DTOs (not raw models)

### Authentication Flow

```
Registration → Email Verification → Login → JWT Token
```

- Registration creates unverified user
- Email verification marks account as verified
- Login requires verified email (when verification service is enabled)
- JWT tokens expire after configured time
- Refresh tokens not yet implemented

### Permission System

- **Format**: `resource.action` (e.g., `users.create`)
- **Resources**: users, roles, permissions, orders, etc.
- **Actions**: create, read, update, delete, list, manage, execute, approve, assign, grant
- **Wildcards**: Supported (`users.*` or `*.*`)
- **Hierarchy**: Roles inherit permissions from parent roles

## Environment Variables

```bash
# Server
PORT=8080
SERVER_ENV=development  # or production

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/servicepro

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h

# AWS SES
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret

# Frontend URLs (for emails)
VERIFICATION_URL=http://localhost:5173/verify-email
RESET_URL=http://localhost:5173/reset-password
```

## Running the Application

```bash
# Install dependencies
go mod download

# Run migrations
psql -d servicepro -f migrations/001_create_users_table.sql
psql -d servicepro -f migrations/002_add_password_reset_fields.sql
psql -d servicepro -f migrations/003_add_email_verification.sql
psql -d servicepro -f migrations/004_create_roles_and_permissions.sql

# Run the application
go run cmd/api/main.go

# Run tests
go test ./...

# Run specific tests
go test ./internal/services/... -v
```

## API Endpoints

### Authentication

- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login with email/password
- `POST /api/v1/auth/verify` - Verify email address
- `POST /api/v1/auth/resend-verification` - Resend verification email

### Password Reset

- `POST /api/v1/auth/reset-request` - Request password reset
- `POST /api/v1/auth/reset-password` - Reset password with token

### Webhooks

- `POST /api/v1/webhooks/ses-bounce` - AWS SES bounce notifications
- `POST /api/v1/webhooks/ses-notification` - AWS SES general notifications

### Health

- `GET /health` - Health check endpoint

## Testing

- **Unit Tests**: All services and models have comprehensive tests
- **Test Framework**: testify/assert and testify/mock
- **Test Redis**: Uses miniredis for in-memory Redis testing
- **Coverage Goal**: >80% for business logic

## Common Tasks

### Adding a New Feature

1. Create migration in `migrations/` if DB changes needed
2. Add/update models in `internal/models/`
3. Create repository methods in `internal/repository/`
4. Implement business logic in `internal/services/`
5. Add HTTP handlers in `internal/api/handlers/`
6. Register routes in `internal/api/routes/`
7. Write unit tests for services
8. Update this README if public API changes

### Adding a New Permission

1. Add permission to migration `004_create_roles_and_permissions.sql`
2. Assign to appropriate roles
3. Use in middleware/handlers via `PermissionChecker`

## Architecture Decisions

### Why GORM?

- Rapid development with type-safe queries
- Automatic migrations and schema management
- Built-in soft deletes and hooks
- Less boilerplate than raw SQL
- Trade-off: Small performance overhead vs developer productivity

### Why JWT?

- Stateless authentication (no session storage)
- Easy to scale horizontally
- Works well with SPAs and mobile apps
- Can include claims (user ID, roles, etc.)

### Why Redis?

- Fast in-memory storage for:
  - Rate limiting counters
  - Password reset tokens (with TTL)
  - Email verification tokens (with TTL)
- Future: Session storage, caching

### Why Repository Pattern?

- Separation of concerns
- Easy to test (mock repositories)
- Database implementation can be swapped
- Business logic doesn't depend on GORM

## Troubleshooting

### Database Connection Issues

- Ensure PostgreSQL is running: `pg_ctl status`
- Check DATABASE_URL in environment
- Verify database exists: `psql -l`

### Redis Connection Issues

- Ensure Redis is running: `redis-cli ping`
- Check REDIS_ADDR in environment

### Migration Issues

- Migrations must be run in order
- Check for UUID extension: `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`

### Email Not Sending

- Check AWS SES credentials
- Verify sender email is verified in SES
- In development, mock email service is used (check logs)

## Resources

- [Email Verification Documentation](./docs/EMAIL_VERIFICATION.md)
- [GORM Docs](https://gorm.io/docs/)
- [Gin Docs](https://gin-gonic.com/docs/)
- [Go Best Practices](https://golang.org/doc/effective_go)
