# ServicePro - Technical Debt & Future Work

This document tracks technical improvements that should be implemented but are deferred for future development cycles.

---

## High Priority

### Testing

- [ ] **Integration Tests** - Create comprehensive API integration tests with test database

  - Test all CRUD operations for each entity (customers, jobs, quotes, invoices)
  - Test authentication flows (login, register, password reset, token refresh)
  - Test authorization (permission checks, role-based access)
  - Test webhook handlers (Stripe, SES)
  - Framework: Use `testify` with `sqlmock` and `miniredis` for mocking

- [ ] **Frontend E2E Tests** - Set up Cypress or Playwright for end-to-end testing
  - Critical user flows: login, create customer, create job, generate invoice
  - Payment flow testing with Stripe test mode

### CI/CD Pipeline

- [ ] **GitHub Actions Workflows** - Create CI/CD pipelines

  ```yaml
  # Suggested workflow stages:
  - lint (golangci-lint, eslint)
  - test (unit tests, integration tests)
  - build (Docker images)
  - security-scan (trivy, snyk)
  - deploy-staging (auto on main branch merge)
  - deploy-preprod (manual approval)
  - deploy-prod (manual approval + version tag)
  ```

- [ ] **Automated Security Scanning**
  - Container image scanning with Trivy
  - Dependency vulnerability scanning
  - SAST scanning for code vulnerabilities

### Request Validation

- [ ] **Comprehensive Input Validation** - Add request validators for all endpoints
  - Use `go-playground/validator` for struct validation
  - Create custom validators for business rules
  - Standardize validation error responses
  - Document validation rules in OpenAPI spec

---

## Email Client Enhancements

The following advanced email features were identified in legacy code and should be integrated into the unified email client factory (`pkg/clients/email/`). These features should be implemented for all email providers (SES, Resend, Mock).

### Rate Limiting & Throttling

- [ ] **Token Bucket Rate Limiter** - Prevent exceeding provider send limits
  - Configurable tokens per second (e.g., SES sandbox = 14/sec)
  - Automatic token refill based on elapsed time
  - Context-aware waiting (respects cancellation)
  - **Benefit**: Prevents API throttling errors and account suspension

### Retry Logic with Exponential Backoff

- [ ] **Smart Retry System** - Handle transient failures gracefully
  - Configurable max retries, base delay, max delay, multiplier
  - Jitter to prevent thundering herd
  - Distinguish retryable vs non-retryable errors
  - **Benefit**: Improved delivery reliability without manual intervention

### Batch Sending

- [ ] **Concurrent Batch Operations** - Send multiple emails efficiently
  - Configurable concurrency limit (semaphore-based)
  - Aggregate results with success/failure counts
  - Context cancellation support
  - **Benefit**: Efficient bulk email operations (newsletters, notifications)

### Email Queue System

- [ ] **Redis-Backed Email Queue** - Async email processing with persistence
  - Job priorities (low, normal, high, critical)
  - Job statuses (pending, processing, completed, failed, retrying, dead)
  - Scheduled sends (future dated emails)
  - Dead letter queue for failed emails
  - Worker pool with configurable concurrency
  - **Benefit**: Decouples email sending from request handling, improves reliability

### Email Tracking & Analytics

- [ ] **Open/Click Tracking** - Monitor email engagement

  - Tracking pixel injection for opens
  - Link wrapping for click tracking
  - Opt-out management
  - **Benefit**: Measure email campaign effectiveness

- [ ] **Analytics Dashboard** - Aggregate email metrics
  - Send/delivery/bounce/complaint rates
  - Time-series data with configurable aggregation
  - Caching for performance
  - **Benefit**: Business insights into email performance

### Template Support

- [ ] **SES Template Integration** - Use pre-defined templates
  - Template prefix configuration
  - Dynamic template data injection
  - Template versioning support
  - **Benefit**: Consistent branding, easier template management

### CloudWatch Metrics Integration

- [ ] **Email Metrics to CloudWatch** - Operational visibility
  - Send success/failure counts
  - Latency histograms
  - Custom dimensions (environment, email type)
  - **Benefit**: Alerting and dashboards for email health

### Domain Verification

- [ ] **SES Domain Management** - Programmatic domain setup
  - Domain verification status checking
  - DKIM setup and verification
  - MAIL FROM domain configuration
  - **Benefit**: Automated email infrastructure setup

---

## Storage Client Enhancements

The following advanced storage features were identified in legacy code and should be integrated into the unified storage client factory (`pkg/clients/storage/`). These features should be implemented for all storage providers (S3, R2, Mock).

### Multipart Upload/Download Management

- [ ] **AWS S3 Manager Integration** - Efficient large file handling
  - Configurable part size and concurrency
  - Automatic multipart uploads for large files
  - Parallel download with byte ranges
  - **Benefit**: Faster uploads/downloads for large files, automatic retry of failed parts

### Concurrency Control

- [ ] **Semaphore-Based Rate Limiting** - Prevent resource exhaustion
  - Configurable max concurrent uploads/downloads
  - Context-aware semaphore acquisition
  - **Benefit**: Prevents overwhelming the storage provider or local resources

### Content Validation

- [ ] **Checksum Verification** - Data integrity assurance
  - SHA256 checksum calculation on upload
  - Checksum stored in metadata for verification on download
  - MD5 content integrity headers
  - **Benefit**: Detect corruption during transfer

### MIME Type Detection

- [ ] **Automatic Content Type Detection** - Proper file handling
  - Extension-based detection
  - Content sniffing for unknown extensions
  - Configurable allowed/blocked MIME types
  - **Benefit**: Correct content-type headers, security filtering

### File Organization

- [ ] **Key Generation Strategy** - Organized storage structure
  - Pattern: `{document_type}/{year}/{month}/{uuid}_{filename}`
  - Filename sanitization
  - Entity association metadata
  - **Benefit**: Easy browsing, logical organization, deduplication

### Encryption Support

- [ ] **Server-Side Encryption Options** - Data at rest security
  - AES-256 encryption
  - KMS key support
  - Per-upload encryption override
  - **Benefit**: Compliance, data protection

### Batch Operations

- [ ] **Bulk Delete** - Efficient multi-object operations
  - Delete multiple objects in single API call
  - Quiet mode (suppress individual responses)
  - **Benefit**: Faster cleanup operations

### Object Versioning

- [ ] **Version Management** - Object history
  - Delete specific versions
  - Download specific versions
  - Version ID tracking
  - **Benefit**: Accidental deletion recovery, audit trail

### Client Statistics

- [ ] **Upload/Download Metrics** - Operational visibility
  - Total uploads/downloads count
  - Total bytes transferred
  - Failed operation tracking
  - **Benefit**: Usage monitoring, capacity planning

---

## Metrics Client Enhancements

The following advanced metrics features were identified in legacy code and should be integrated into the unified metrics client factory (`pkg/clients/metrics/`).

### Histogram Implementation

- [ ] **Distribution Tracking** - Latency and size distributions
  - Configurable bucket boundaries
  - Percentile calculations (p50, p90, p99)
  - Count and sum tracking
  - **Benefit**: Understanding latency distributions, SLO monitoring

### Summary Implementation

- [ ] **Sliding Window Statistics** - Recent trend analysis
  - Configurable window size
  - Mean calculation over window
  - Memory-efficient ring buffer
  - **Benefit**: Real-time trend monitoring

### Runtime Metrics Collection

- [ ] **Go Runtime Metrics** - Application health
  - Goroutine count
  - Heap allocation and objects
  - GC pause times and cycles
  - CPU count
  - Configurable collection interval
  - **Benefit**: Memory leak detection, performance profiling

### Metrics Registry

- [ ] **Centralized Metric Management** - Organized metrics
  - Get-or-create pattern for metrics
  - Label-based metric identification
  - Export all metrics as array
  - **Benefit**: Prevents duplicate metrics, easy enumeration

---

## Medium Priority

### Code Quality

- [ ] **Linting Configuration**

  - Backend: Set up `golangci-lint` with custom rules
    ```yaml
    # .golangci.yml suggested linters:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    ```
  - Frontend: Configure ESLint + Prettier with strict rules
  - Add lint checks to CI pipeline

- [ ] **Pre-commit Hooks** - Set up pre-commit framework
  ```yaml
  # .pre-commit-config.yaml
  repos:
    - repo: https://github.com/golangci/golangci-lint
      hooks:
        - id: golangci-lint
    - repo: https://github.com/pre-commit/mirrors-eslint
      hooks:
        - id: eslint
    - repo: https://github.com/pre-commit/pre-commit-hooks
      hooks:
        - id: trailing-whitespace
        - id: end-of-file-fixer
        - id: check-yaml
        - id: check-json
  ```

### Developer Experience

- [ ] **Developer Environment Setup** - Improve local development experience

  - Create `make dev` command for one-click setup
  - Add hot-reload for both backend and frontend in Docker
  - Create seed data scripts for local development
  - Document common development workflows

- [ ] **ArgoCD Dev Environments** - Enable developer-specific deployments
  - Create ArgoCD ApplicationSet for dynamic dev environments
  - Template: `dev-{developer-name}` namespaces
  - Auto-cleanup after inactivity
  - Integrate with PR workflows

### Infrastructure

- [ ] **Terraform State Migration** - Enable remote state management

  - Uncomment and configure S3 backend in `main.tf`
  - Set up DynamoDB table for state locking
  - Migrate existing state to S3
  - Document state management procedures

- [ ] **Terraform Modules** - Complete empty module implementations
  - `modules/eks/` - EKS cluster configuration
  - `modules/rds/` - RDS PostgreSQL setup
  - `modules/elasticache/` - Redis cluster
  - `modules/s3/` - S3 buckets with policies

---

## Low Priority

### Documentation

- [ ] **API Documentation** - Generate OpenAPI/Swagger specification

  - Use `swaggo/swag` for auto-generation from Go comments
  - Host interactive API docs at `/api/docs`
  - Keep spec in sync with code changes

- [ ] **Architecture Decision Records (ADRs)**
  - Document key architectural decisions
  - Include context, decision, and consequences
  - Store in `docs/adr/` directory

### Observability

- [ ] **Distributed Tracing** - Implement request tracing

  - Add OpenTelemetry instrumentation
  - Configure trace export to AWS X-Ray or Jaeger
  - Add trace IDs to all log entries

- [ ] **Custom Metrics** - Extend application metrics
  - Business metrics (jobs created, invoices generated, etc.)
  - Performance metrics (request latency percentiles)
  - Error rate tracking by endpoint

### Security

- [ ] **Security Hardening**

  - Implement CSP headers for frontend
  - Add rate limiting per user (not just per IP)
  - Implement request signing for webhooks
  - Add audit logging for sensitive operations

- [ ] **Secret Rotation** - Automate secret rotation
  - Database credentials rotation
  - JWT signing key rotation
  - API key rotation procedures

---

## Orphaned Features to Integrate

These features exist in the codebase but are not wired into the application routes. They represent completed or near-complete implementations that should be integrated.

### Job Status State Machine (High Priority)

A robust job status management system with state machine validation, concurrency protection, and audit trail.

**What it provides:**

- State machine pattern with validated transitions (prevents invalid status changes)
- Pessimistic locking (`SELECT ... FOR UPDATE`) for concurrent update protection
- Complete status change history with audit trail
- Bulk status changes for batch operations
- Automatic timestamp updates (ActualStartAt, ActualEndAt, ActualDuration)
- Notification dispatch on status changes
- Transition metrics for monitoring

**API Endpoints to wire up:**

- `POST /api/v1/jobs/:id/status` - Change job status
- `GET /api/v1/jobs/:id/status/history` - Get status change history
- `GET /api/v1/jobs/:id/status/transitions` - Get allowed transitions
- `POST /api/v1/jobs/:id/status/validate` - Validate without applying
- `POST /api/v1/jobs/status/bulk` - Bulk status change
- `GET /api/v1/jobs/status/statistics` - Status statistics

**Files that were removed (rebuild from this spec):**

- `internal/services/status_service.go` - StatusService with state machine logic
- `internal/api/handlers/status_handler.go` - HTTP handlers

**Integration requirements:**

- [ ] Wire StatusHandler into routes.go
- [ ] Add `JobStatusTransition` model and migration
- [ ] Connect to existing notification service for status change alerts
- [ ] Add status validation to existing job update endpoints

---

### Payment Notification System (High Priority)

Comprehensive payment reminder and notification system with multi-channel delivery.

**What it provides:**

- Multi-channel notifications: Email, SMS, Push, In-App, Webhook
- Notification types:
  - `payment_due_soon` - Reminder before due date
  - `payment_overdue` - Past due notification
  - `payment_in_grace_period` - Grace period warning
  - `early_payment_discount_available` - Discount reminder
  - `late_fee_applied` - Late fee notification
  - `payment_received` - Confirmation
  - `partial_payment_received` - Partial payment confirmation
- Configurable notification rules (e.g., "send reminder 3 days before due")
- Delivery history tracking (sent, delivered, opened, clicked)
- Batch processing with configurable retries
- Scheduled notifications
- Statistics and metrics

**Database tables needed:**

- `payment_notifications` - Notification records
- `payment_notification_rules` - Rule configurations
- `payment_notification_history` - Delivery tracking

**Files that were removed (rebuild from this spec):**

- `internal/services/payment_notification_service.go` - Full service implementation

**Integration requirements:**

- [ ] Create database migration for notification tables
- [ ] Create NotificationProcessor worker (background job)
- [ ] Wire up to invoice service (trigger on invoice creation/update)
- [ ] Add SMS provider integration (Twilio or similar)
- [ ] Create admin UI for managing notification rules

---

### Job Scheduling System (Medium Priority)

Calendar-style job scheduling with conflict detection and technician availability.

**What it provides:**

- Job scheduling with start/end times
- Conflict detection (overlapping schedules)
- Technician schedule management
- Recurrence support (daily, weekly, monthly)
- Reminders with configurable lead time
- Color coding for calendar display

**API Endpoints to wire up:**

- `POST /api/v1/schedules` - Create schedule
- `GET /api/v1/schedules/:id` - Get schedule
- `PUT /api/v1/schedules/:id` - Update schedule
- `DELETE /api/v1/schedules/:id` - Delete schedule
- `GET /api/v1/schedules` - List with filters
- `GET /api/v1/schedules/technician/:tech_id` - Technician's schedule

**Files that were removed (rebuild from this spec):**

- `internal/services/schedule_service.go` - ScheduleService
- `internal/api/handlers/schedule_handler.go` - HTTP handlers

**Integration requirements:**

- [ ] Create Schedule model and migration
- [ ] Create ScheduleRepository
- [ ] Wire ScheduleHandler into routes.go
- [ ] Add conflict detection algorithm
- [ ] Integrate with job creation flow (auto-create schedule)
- [ ] Add calendar view to frontend

---

### Tax Calculation Service (Medium Priority)

State-based tax calculation with Redis caching for performance.

**What it provides:**

- State-based tax rate lookups (all US states configured)
- Zip code to state mapping
- Tax exemption validation:
  - Government entities
  - Educational institutions
  - Non-profit organizations
  - Resale certificates
- Redis caching with 1-hour TTL
- Configurable default tax rates per state

**Files that were removed (rebuild from this spec):**

- `internal/services/tax_service.go` - TaxService
- `internal/services/tax_types.go` - Tax-related types

**Integration requirements:**

- [ ] Wire into invoice creation flow
- [ ] Wire into quote calculation
- [ ] Add tax exemption fields to Customer model
- [ ] Create tax rate admin UI (override defaults)
- [ ] Add tax breakdown to invoice/quote responses

---

### Quote Template System (Medium Priority)

Dynamic quote templating with variable substitution and validation.

**What it provides:**

- Variable substitution: `{{customer_name}}`, `{{quote_total}}`, etc.
- Type-aware formatting:
  - Dates → "January 2, 2006"
  - Decimals → proper precision
  - Booleans → "Yes/No"
- Variable validation:
  - Required/optional variables
  - Min/max length for text
  - Regex pattern matching
  - Allowed values (dropdown)
  - Min/max for numbers
- Template categories (HVAC, Plumbing, Electrical, etc.)
- Template versioning (tracks changes)
- Usage tracking (counts template usage)
- Import/export templates as JSON
- Duplicate template feature
- Statistics (most used, recently updated)

**Template sections:**

- Content (main body)
- Line items (with variable descriptions)
- Payment terms
- Delivery info
- Warranty info
- Default tax rate
- Valid days

**Files that were removed (rebuild from this spec):**

- `internal/services/template_engine.go` - Core rendering engine
- `internal/services/quote_template_service.go` - Business logic

**Integration requirements:**

- [ ] Create QuoteTemplate model and migration
- [ ] Create TemplateCategory model and migration
- [ ] Create QuoteTemplateHandler
- [ ] Wire into routes.go
- [ ] Integrate with quote creation (create from template)
- [ ] Add template management UI

---

### WebSocket Real-time Notifications (Low Priority)

Real-time push notifications via WebSocket for instant updates.

**What it provides:**

- WebSocket hub pattern for connection management
- Real-time status updates (quote status, job status)
- Broadcast to all connected clients
- Per-user message targeting

**Files that were removed (rebuild from this spec):**

- `internal/services/websocket_notification_service.go` - WebSocket hub

**Integration requirements:**

- [ ] Set up WebSocket endpoint (`/ws`)
- [ ] Add authentication for WebSocket connections
- [ ] Integrate with status change events
- [ ] Add frontend WebSocket client
- [ ] Handle reconnection logic

---

## Removed Features to Restore

These features were removed during build fixes due to missing dependencies or method mismatches. They need to be restored.

### Payment System (High Priority)

Stripe payment integration was removed due to model/service mismatches.

**Files to restore:**

- [ ] `backend/internal/models/payment_method.go` - Saved payment methods model (cards, bank accounts)
- [ ] `backend/internal/services/stripe/payment_methods.go` - Stripe payment methods service
- [ ] `backend/internal/api/handlers/payment_handler.go` - Payment processing endpoints
- [ ] `backend/internal/api/handlers/payment_methods.go` - Saved payment methods endpoints
- [ ] `backend/internal/api/routes/payment_routes.go` - Payment route definitions

**Issues to fix:**

- `PaymentMethodType` and `PaymentMethodResponse` duplicated in `payment.go` and `payment_method.go` - consolidate
- `User` model needs `StripeCustomerID` field
- Method signature mismatches between handler and service
- `PaymentIntentResult` missing `ReceiptURL` field

### Invoice Templates / PDF Generation (High Priority)

Invoice template system for generating PDF invoices was removed due to service method mismatches.

**Files to restore:**

- [ ] `backend/internal/api/handlers/invoice_template_handler.go` - Template CRUD and PDF generation
- [ ] `backend/internal/api/routes/invoice_template_routes.go` - Template route definitions

**Issues to fix:**

- `SetDefaultTemplate` returns 1 value but handler expects 2
- `GeneratePDF` method signature mismatch
- `GeneratePreviewFromContent` method signature mismatch
- `DeleteAsset` method signature mismatch
- `stringPtr` function redeclared

### Error Tracking - Sentry (Medium Priority)

Sentry integration for production error monitoring.

**Files to restore:**

- [ ] `backend/internal/api/middleware/error_tracking.go` - Sentry middleware
- [ ] `backend/internal/api/middleware/error_tracking_test.go` - Tests

**Dependencies:** `go get github.com/getsentry/sentry-go`

**Environment variables:** `SENTRY_DSN`, `SENTRY_ENVIRONMENT`

### Health Checks - Advanced (Medium Priority)

Advanced health check system with uptime tracking and dashboard.

**Files to restore:**

- [ ] `backend/internal/health/` - Health check package (needs to be created)
- [ ] `backend/internal/api/handlers/health_handler.go` - Health endpoints
- [ ] `backend/internal/api/routes/health_routes.go` - Health route definitions

**Features:** Detailed health per dependency, uptime tracking, health dashboard, k8s probes

### Analytics System (Low Priority)

Custom analytics for tracking API usage and feature adoption.

**Files to restore:**

- [ ] `backend/internal/analytics/` - Analytics package (needs to be created)
- [ ] `backend/internal/api/middleware/analytics.go` - Analytics middleware
- [ ] `backend/internal/api/handlers/analytics_handler.go` - Analytics API endpoints

### Performance Monitoring (Low Priority)

Request performance, database query, and cache metrics.

**Files to restore:**

- [ ] `backend/internal/monitoring/` - Monitoring package (needs to be created)
- [ ] `backend/internal/api/middleware/performance.go` - Performance middleware
- [ ] `backend/internal/api/handlers/metrics_handler.go` - Metrics endpoints
- [ ] `backend/internal/api/handlers/metrics_handler_test.go` - Tests
- [ ] `backend/internal/api/routes/metrics_routes.go` - Metrics route definitions

**Features:** Request duration histograms, error rate tracking, slow request detection, Prometheus `/metrics` endpoint

### Other Fixes Needed

- [ ] `backend/internal/services/quote_status_machine.go:118` - Customer email validation commented out; need to add `CustomerEmail` to Quote model or load Customer relationship

---

## Completed Items

_Move items here when completed with date and PR/commit reference_

- [x] Initial project setup (2024-01-15)
- [x] Basic authentication flow (2024-01-20)
- [x] Customer CRUD operations (2024-01-25)

---

## Notes

### Priority Definitions

- **High**: Blocks production deployment or poses security risk
- **Medium**: Improves developer experience or code quality
- **Low**: Nice-to-have improvements

### Adding New Items

When adding new items, include:

1. Clear description of what needs to be done
2. Why it's important
3. Any relevant context or links
4. Suggested implementation approach if known

---

_Last updated: 2024-01-19_
