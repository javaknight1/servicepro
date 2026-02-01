# ServicePro - Technical Debt & Future Work

This document tracks technical improvements that should be implemented but are deferred for future development cycles.

---

## Sprint Roadmap

### Sprint 1 - Production Readiness

- [ ] Fly.io deployment setup
- [x] Migrate `db.Raw()` to GORM (SQL injection risk)
- [ ] Add general API rate limiting
- [ ] Enable Sentry error tracking
- [ ] Remove console.log statements from frontend

### Sprint 2 - Observability & Email

- [ ] Structured logging + correlation IDs
- [ ] Email retry logic with exponential backoff
- [ ] Expanded health checks (DB, Redis, S3)

### Sprint 3 - Testing & Security

- [ ] Integration tests for 5 critical workflows
- [ ] Input validation improvements
- [ ] N+1 query fixes

### Sprint 4 - Performance & Cleanup

- [ ] Bundle analysis + optimization
- [ ] Frontend type safety fixes
- [ ] Query performance monitoring
- [ ] Dead code elimination

---

## P0 - Critical (Blocks Production)

### Infrastructure & Deployment

- [ ] **Fly.io Backend Deployment**
  - **What**: Create `fly.toml` configuration for backend service deployment
  - **Why**: No deployment configuration exists; README says "coming soon"
  - **Expected Result**: Backend deployable to Fly.io with single command
  - **Acceptance Criteria**:
    - `fly.toml` created with proper service configuration
    - Environment variables documented
    - Health check endpoint configured
    - Auto-scaling rules defined

- [ ] **Fly.io Deployment GitHub Action**
  - **What**: Add deployment workflow to `.github/workflows/`
  - **Why**: Automate deployments on merge to main
  - **Expected Result**: Automatic staging deployment on main branch, manual production deployment
  - **Acceptance Criteria**:
    - Workflow triggers on main branch push
    - Staging deploys automatically
    - Production requires manual approval
    - Rollback procedure documented

- [ ] **Container Image Scanning (Trivy)**
  - **What**: Add Trivy container scanning to CI pipeline
  - **Why**: Detect vulnerabilities in Docker images before deployment
  - **Expected Result**: CI fails if critical/high vulnerabilities found
  - **Acceptance Criteria**:
    - Trivy integrated into GitHub Actions
    - Threshold set for blocking vulnerabilities
    - Scan results visible in PR checks

### Security - Critical

- [ ] **Profile and Fix N+1 Queries**
  - **What**: Audit list endpoints for N+1 query patterns
  - **Why**: 58 Preload calls found; list endpoints likely have performance issues
  - **Expected Result**: All list endpoints optimized with proper eager loading
  - **Acceptance Criteria**:
    - Customer list: single query + 1 for counts
    - Job list: single query + 1 for related data
    - Query count logged in development mode
    - No endpoint exceeds 5 queries for list operations

### Observability - Critical

- [ ] **Enable Sentry Error Tracking**
  - **What**: Configure and enable Sentry in production environment
  - **Why**: Sentry client exists but may not be active; need production error visibility
  - **Expected Result**: All unhandled errors reported to Sentry with context
  - **Acceptance Criteria**:
    - Sentry DSN configured in production
    - Source maps uploaded for frontend errors
    - User context attached to errors
    - Release tracking enabled
    - Alert rules configured for critical errors

---

## P1 - High Priority

### Infrastructure

- [ ] **Staging Environment Configuration**
  - **What**: Create separate configuration for staging environment
  - **Why**: Need environment parity for testing before production
  - **Expected Result**: Isolated staging environment with production-like setup
  - **Acceptance Criteria**:
    - Separate Fly.io app for staging
    - Staging database (isolated from production)
    - Staging-specific environment variables
    - Accessible at staging.servicepro.com (or similar)

- [x] **Dependency Vulnerability Scanning**
  - **What**: Add Dependabot or Snyk to scan for vulnerable dependencies
  - **Why**: Go modules and npm packages need continuous security monitoring
  - **Expected Result**: Automatic PRs for security updates
  - **Acceptance Criteria**:
    - Dependabot configured for Go and npm ✓
    - Weekly security scans ✓
    - Critical vulnerabilities create blocking issues
    - Auto-merge for patch-level security updates ✓

- [ ] **Bundle Size Monitoring in CI**
  - **What**: Add bundle analysis to CI pipeline with size limits
  - **Why**: Prevent frontend bundle bloat over time
  - **Expected Result**: CI warns/fails if bundle exceeds threshold
  - **Acceptance Criteria**:
    - Bundle size reported in PR comments
    - Warning at 500KB gzipped
    - Failure at 750KB gzipped
    - Chunk breakdown visible

### Email Infrastructure

- [ ] **Email Rate Limiting for SES**
  - **What**: Implement token bucket rate limiter for SES sandbox limits
  - **Why**: SES sandbox limit is 14 emails/second; need to prevent throttling
  - **Expected Result**: Emails queued and sent within rate limits
  - **Acceptance Criteria**:
    - Configurable tokens per second (default: 14)
    - Automatic queuing when limit reached
    - Metrics for queue depth and wait times
    - Graceful handling of rate limit errors

- [ ] **Email Retry Logic with Exponential Backoff**
  - **What**: Add retry mechanism for transient email failures
  - **Why**: Network issues and temporary provider errors cause lost emails
  - **Expected Result**: Transient failures automatically retried
  - **Acceptance Criteria**:
    - Max 3 retries with exponential backoff
    - Base delay: 1 second, max delay: 30 seconds
    - Jitter added to prevent thundering herd
    - Non-retryable errors (invalid email) fail immediately
    - Failed emails logged with reason

### Logging & Observability

- [ ] **Request Correlation IDs**
  - **What**: Add unique request ID to all log entries for tracing
  - **Why**: Cannot trace requests across services/logs currently
  - **Expected Result**: Every request has traceable ID in all logs
  - **Acceptance Criteria**:
    - X-Request-ID header generated if not provided
    - ID propagated to all service calls
    - ID included in all log entries
    - ID returned in error responses
    - Frontend includes ID in error reports

- [ ] **Structured JSON Logging**
  - **What**: Convert log output to structured JSON format
  - **Why**: Better parsing in log aggregation systems
  - **Expected Result**: All logs in JSON format with consistent fields
  - **Acceptance Criteria**:
    - JSON format in production
    - Human-readable format in development
    - Standard fields: timestamp, level, message, request_id, user_id
    - Context fields for domain-specific data

- [ ] **Expanded Health Checks**
  - **What**: Add health checks for all dependencies (DB, Redis, S3)
  - **Why**: Current /health only checks if server is running
  - **Expected Result**: Comprehensive health status with dependency details
  - **Acceptance Criteria**:
    - `/health/live` - Is the server running? (for k8s liveness)
    - `/health/ready` - Are all dependencies healthy? (for k8s readiness)
    - Individual checks: database, redis, s3, stripe
    - Response includes latency for each check
    - Degraded state if non-critical dependency unhealthy

- [ ] **HTTP Request Metrics**
  - **What**: Add Prometheus metrics for HTTP requests
  - **Why**: Need visibility into request latency and error rates
  - **Expected Result**: Prometheus-compatible metrics endpoint
  - **Acceptance Criteria**:
    - `http_requests_total` counter by method, path, status
    - `http_request_duration_seconds` histogram
    - `http_requests_in_flight` gauge
    - Metrics exposed at `/metrics`

### Security

- [ ] **Comprehensive Input Validation**
  - **What**: Add request validators for all API endpoints
  - **Why**: Inconsistent validation across endpoints
  - **Expected Result**: All inputs validated with clear error messages
  - **Acceptance Criteria**:
    - Use `go-playground/validator` consistently
    - Custom validators for business rules (email format, phone, etc.)
    - Standardized validation error response format
    - Validation rules documented

### Testing

- [ ] **Integration Tests for Critical Workflows**
  - **What**: Add integration tests for 5 critical user workflows
  - **Why**: Only 9 integration tests exist currently
  - **Expected Result**: End-to-end testing of critical paths
  - **Workflows to test**:
    1. User registration → email verification → login
    2. Create customer → create job → complete job
    3. Create quote → convert to invoice → mark paid
    4. Stripe payment method → subscription change
    5. Team member invite → accept → permissions check
  - **Acceptance Criteria**:
    - Tests run against real database (test container)
    - Tests clean up after themselves
    - Tests run in CI pipeline
    - <30 second execution time

- [x] **E2E Testing Setup (Cypress/Playwright)**
  - **What**: Set up end-to-end testing framework for frontend
  - **Why**: No E2E tests exist; critical flows untested
  - **Expected Result**: Automated browser tests for critical flows
  - **Acceptance Criteria**:
    - Framework installed and configured ✓
    - Tests for: login, create customer, create job ✓ (auth, customers, jobs, invoices, quotes, payments)
    - Tests run in CI with video recording
    - Test environment seeded with data

### Frontend Cleanup

- [ ] **Remove Console.log Statements**
  - **What**: Remove or replace 134 console.log/debugger statements
  - **Why**: Debug logging in production; clutters browser console
  - **Expected Result**: No debug logging in production builds
  - **Acceptance Criteria**:
    - All console.log removed or replaced with proper logger
    - ESLint rule to prevent future additions
    - Logger that only outputs in development

- [ ] **Fix @ts-nocheck in Routes**
  - **What**: Fix type inference issues in `/src/routes/index.tsx`
  - **Why**: Entire file has TypeScript checking disabled
  - **Expected Result**: Full type safety in routing
  - **Acceptance Criteria**:
    - Remove @ts-nocheck comment
    - Fix all type errors
    - Dynamic imports properly typed
    - No regression in routing behavior

---

## P2 - Medium Priority

### Infrastructure

- [ ] **Infrastructure as Code (Terraform)**
  - **What**: Add Terraform for cloud infrastructure management
  - **Why**: Infrastructure changes should be versioned and reproducible
  - **Expected Result**: All infrastructure defined in code
  - **Acceptance Criteria**:
    - Terraform modules for: database, redis, S3
    - State stored in S3 with locking
    - Separate workspaces for staging/production
    - CI/CD integration for terraform apply

### Email Infrastructure

- [ ] **Redis-Backed Email Queue**
  - **What**: Implement async email processing with persistence
  - **Why**: Decouple email sending from request handling
  - **Expected Result**: Emails queued and processed asynchronously
  - **Acceptance Criteria**:
    - Job priorities: low, normal, high, critical
    - Job statuses: pending, processing, completed, failed, retrying
    - Dead letter queue for failed emails
    - Worker pool with configurable concurrency
    - Scheduled sends (future dated emails)
    - Dashboard for queue monitoring

- [ ] **Email Batch Sending**
  - **What**: Implement concurrent batch operations for bulk emails
  - **Why**: Efficient handling of newsletters, notifications
  - **Expected Result**: Bulk emails sent with controlled concurrency
  - **Acceptance Criteria**:
    - Configurable concurrency limit (default: 10)
    - Aggregate results with success/failure counts
    - Progress tracking for large batches
    - Cancellation support

### Logging & Observability

- [ ] **Query Execution Time Logging**
  - **What**: Log database query execution times
  - **Why**: Identify slow queries in development and production
  - **Expected Result**: Slow queries logged with context
  - **Acceptance Criteria**:
    - Log queries exceeding threshold (default: 100ms)
    - Include query, parameters, duration
    - Prometheus histogram for query times
    - Alert on queries >1 second

- [ ] **Business Metrics Dashboard**
  - **What**: Track business metrics (customers, jobs, revenue)
  - **Why**: Need visibility into business health
  - **Expected Result**: Prometheus metrics for business KPIs
  - **Acceptance Criteria**:
    - Metrics: customers_total, jobs_created, jobs_completed
    - Metrics: invoices_generated, revenue_total
    - Metrics: quotes_sent, quotes_converted
    - Grafana dashboard template provided

### Testing

- [ ] **Email Service Tests**
  - **What**: Add unit tests for email sending service
  - **Why**: Email sending currently untested
  - **Expected Result**: Full test coverage for email service
  - **Acceptance Criteria**:
    - Tests for all email types (welcome, reset, verification)
    - Mock provider used in tests
    - Template rendering tested
    - Error handling tested

- [ ] **Stripe/Payment Flow Tests**
  - **What**: Add tests for payment processing flows
  - **Why**: Critical business logic untested
  - **Expected Result**: Payment flows tested with Stripe test mode
  - **Acceptance Criteria**:
    - Setup intent creation tested
    - Payment method attachment tested
    - Subscription creation/change tested
    - Webhook handling tested

- [ ] **Frontend Service Layer Tests**
  - **What**: Add tests for frontend services and API calls
  - **Why**: Service layer has minimal test coverage
  - **Expected Result**: Service methods tested with mocked API
  - **Acceptance Criteria**:
    - All membershipApi methods tested
    - All billingApi methods tested
    - Error handling tested
    - Loading states tested

- [ ] **Frontend Store Tests**
  - **What**: Add tests for Zustand store actions
  - **Why**: Store logic untested
  - **Expected Result**: Store actions tested in isolation
  - **Acceptance Criteria**:
    - membershipStore actions tested
    - billingStore actions tested
    - State transitions verified
    - Error states tested

### Frontend Cleanup

- [ ] **Audit useCallback/useMemo Usage**
  - **What**: Review 203+ hook instances for necessity
  - **Why**: May be over-optimizing; adds complexity
  - **Expected Result**: Hooks used only where needed
  - **Acceptance Criteria**:
    - Document when to use memoization
    - Remove unnecessary memoization
    - No performance regression
    - Guidelines in CONTRIBUTING.md

- [ ] **Review Barrel Export Files**
  - **What**: Audit 1978 lines of index.ts exports for unused exports
  - **Why**: May have orphaned exports; adds to bundle
  - **Expected Result**: Clean exports with no dead code
  - **Acceptance Criteria**:
    - Remove unused exports
    - Add lint rule for unused exports
    - Document export patterns

- [ ] **Add Dead Code Detection**
  - **What**: Add dead code detection to pre-commit hooks
  - **Why**: Prevent future orphaned code
  - **Expected Result**: CI/pre-commit catches unused code
  - **Acceptance Criteria**:
    - ts-prune or similar configured
    - Pre-commit hook added
    - CI job added
    - Baseline established

- [ ] **Complete Footer Links**
  - **What**: Create pages and routes for all footer links currently pointing to "/"
  - **Why**: Footer links (Features, Pricing, About, Contact, Privacy Policy, Terms of Service) are placeholders leading to homepage
  - **Expected Result**: All footer links navigate to proper pages with appropriate content
  - **Acceptance Criteria**:
    - Features page created with product feature highlights
    - Pricing page linked correctly (may already exist)
    - About page created with company information
    - Contact page created with contact form/information
    - Privacy Policy page created with legal content
    - Terms of Service page created with legal content
    - All links in Footer.tsx updated to correct routes
    - SEO meta tags added to each page

### Backend Cleanup

- [ ] **Standardize API Error Responses**
  - **What**: Create consistent error response format across all endpoints
  - **Why**: Inconsistent error formats make client handling difficult
  - **Expected Result**: All errors follow same structure
  - **Acceptance Criteria**:
    - Standard format: `{error: string, message: string, details?: object}`
    - HTTP status codes used correctly
    - Validation errors include field-level details
    - Internal errors don't leak implementation details

- [ ] **Resolve Payment Tracking TODOs**
  - **What**: Implement 7 TODOs in payment tracking model
  - **Why**: Incomplete payment tracking features
  - **Expected Result**: Full payment tracking functionality
  - **Acceptance Criteria**:
    - All TODOs resolved or converted to issues
    - Payment tracking working end-to-end
    - Tests added for new functionality

### Performance

- [ ] **Frontend Bundle Analysis**
  - **What**: Run bundle analysis and optimize large chunks
  - **Why**: Need to understand and optimize bundle size
  - **Expected Result**: Bundle size reduced; large dependencies identified
  - **Acceptance Criteria**:
    - Run `npm run build:analyze`
    - Document findings
    - Split large chunks
    - Lazy load heavy components

- [ ] **Redis Caching for Frequent Queries**
  - **What**: Add caching for frequently accessed data
  - **Why**: Reduce database load for common queries
  - **Expected Result**: Faster response times for cached data
  - **Queries to cache**:
    - Customer counts per tenant
    - Job statistics (pending, completed)
    - User permissions
  - **Acceptance Criteria**:
    - Cache TTL configurable (default: 5 minutes)
    - Cache invalidation on data changes
    - Cache hit/miss metrics

### Security

- [ ] **Document Secrets Rotation Strategy**
  - **What**: Create runbook for rotating all secrets
  - **Why**: No documented procedure for secret rotation
  - **Expected Result**: Clear procedure for rotating each secret
  - **Secrets to document**:
    - Database credentials
    - JWT signing key
    - Stripe API keys
    - AWS credentials
    - Redis password
  - **Acceptance Criteria**:
    - Runbook in docs/security/
    - Zero-downtime rotation procedure
    - Tested in staging

---

## P3 - Low Priority (Nice to Have)

### Infrastructure

- [ ] **Advanced Observability Setup**
  - **What**: Complete Loki/BetterStack integration for log aggregation
  - **Why**: Clients exist but may not be fully configured
  - **Expected Result**: Centralized log aggregation with search
  - **Acceptance Criteria**:
    - Logs shipped to aggregation service
    - Log retention configured
    - Search and alerting enabled

### Email

- [ ] **Email Analytics (Open/Click Tracking)**
  - **What**: Track email opens and link clicks
  - **Why**: Measure email engagement for marketing
  - **Expected Result**: Analytics for email campaigns
  - **Acceptance Criteria**:
    - Tracking pixel injection for opens
    - Link wrapping for click tracking
    - Opt-out management
    - Analytics dashboard

### Frontend

- [ ] **Standardize Component Patterns**
  - **What**: Create consistent patterns across all pages
  - **Why**: Different pages use different patterns
  - **Expected Result**: Consistent, maintainable components
  - **Acceptance Criteria**:
    - Component template documented
    - Example component provided
    - Existing components refactored gradually

### Backend

- [ ] **Implement GeoIP Lookup**
  - **What**: Complete GeoIP stub in access control
  - **Why**: Currently placeholder; needed for location-based features
  - **Expected Result**: IP addresses resolved to locations
  - **Acceptance Criteria**:
    - MaxMind or similar integrated
    - Database auto-updated
    - Location available in request context

- [ ] **Customer Email Validation**
  - **What**: Complete TODO for customer email validation
  - **Why**: Customer emails not validated
  - **Expected Result**: Invalid emails rejected at creation
  - **Acceptance Criteria**:
    - Email format validation
    - Optional MX record check
    - Disposable email detection (optional)

### Testing

- [ ] **Target 80% Backend Test Coverage**
  - **What**: Increase test coverage to 80%+
  - **Why**: Long-term code quality goal
  - **Expected Result**: Comprehensive test suite
  - **Acceptance Criteria**:
    - Coverage reported in CI
    - Coverage gate at 80%
    - Critical paths at 90%+

### Performance

- [ ] **Image Optimization Strategy**
  - **What**: Implement image optimization for user uploads
  - **Why**: Large images slow down pages
  - **Expected Result**: Images automatically optimized
  - **Acceptance Criteria**:
    - Images resized on upload
    - WebP format generated
    - CDN integration
    - Lazy loading implemented

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

## Future Features

### Near-term Features

- [ ] **Webhook Handling Improvements**
  - Stripe event retry logic
  - Webhook signature verification
  - Event deduplication
  - Webhook delivery status tracking

- [x] **Invoice PDF Generation**
  - Template-based PDF generation
  - Email attachment support
  - _Note: Multiple languages and custom branding not yet implemented_

- [ ] **Email Templates Expansion**
  - Job completion notification
  - Quote expiration reminder
  - Payment received confirmation
  - Account activity summary

- [ ] **Dashboard KPI Improvements**
  - Real-time metrics updates
  - Customizable date ranges
  - Export to CSV/PDF
  - Comparative analysis (vs last period)

### Medium-term Features

- [ ] **API Rate Limiting per User/Tenant**
  - Tier-based rate limits
  - Burst allowance
  - Rate limit headers in responses
  - Admin override capability

- [ ] **Audit Logging**
  - Track all data modifications
  - Who, what, when, from where
  - Audit log search and export
  - Retention policy

- [ ] **Two-Factor Authentication**
  - TOTP (Google Authenticator)
  - SMS backup codes
  - Recovery codes
  - Per-user enforcement

- [ ] **Notification Preferences**
  - Email notification settings
  - In-app notification settings
  - Frequency controls (immediate, daily digest, weekly)
  - Channel preferences per notification type

### Long-term Features

- [ ] **Public API for Integrations**
  - API key management
  - Rate limiting per key
  - Webhook subscriptions
  - OAuth2 support

- [ ] **Webhooks for Customers**
  - Event subscriptions
  - Retry with exponential backoff
  - Webhook logs
  - Testing tools

- [ ] **White-label / Custom Branding**
  - Custom logo and colors
  - Custom domain support
  - Email template customization
  - Pro tier feature

- [ ] **Mobile App / PWA**
  - Job status updates
  - Customer information access
  - Push notifications
  - Offline support

---

## Orphaned Features to Integrate

These features exist in the codebase but are not wired into the application routes. They represent completed or near-complete implementations that should be integrated.

### Job Status State Machine (High Priority) - MOSTLY COMPLETE

A robust job status management system with state machine validation, concurrency protection, and audit trail.

**What it provides:**

- State machine pattern with validated transitions (prevents invalid status changes)
- Pessimistic locking (`SELECT ... FOR UPDATE`) for concurrent update protection
- Complete status change history with audit trail
- Bulk status changes for batch operations
- Automatic timestamp updates (ActualStartAt, ActualEndAt, ActualDuration)
- Notification dispatch on status changes
- Transition metrics for monitoring

**API Endpoints (implemented):**

- [x] `POST /api/v1/jobs/:id/transition` - Change job status
- [x] `GET /api/v1/jobs/:id/status-history` - Get status change history
- [ ] `GET /api/v1/jobs/:id/status/transitions` - Get allowed transitions
- [ ] `POST /api/v1/jobs/:id/status/validate` - Validate without applying
- [ ] `POST /api/v1/jobs/status/bulk` - Bulk status change
- [ ] `GET /api/v1/jobs/status/statistics` - Status statistics

**Files implemented:**

- `internal/models/job_status.go` - JobStatusTransition model
- `internal/services/job_service.go` - Status transition logic
- `internal/api/handlers/job_handler.go` - TransitionStatus and GetStatusHistory handlers

**Remaining integration requirements:**

- [ ] Connect to existing notification service for status change alerts
- [ ] Add allowed transitions endpoint
- [ ] Add bulk status change endpoint

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

### Invoice Templates / PDF Generation (Partially Complete)

PDF generation for invoices, quotes, and receipts is now working via `document_pdf_service.go`. Template CRUD management UI is not yet implemented.

**Completed:**

- [x] PDF generation for quotes, invoices, receipts (`backend/internal/services/document_pdf_service.go`)
- [x] PDF download endpoints (`GET /api/v1/quotes/:id/pdf`, `/invoices/:id/pdf`, `/invoices/:id/receipt/pdf`)
- [x] PDF attachments in emails
- [x] PDF templates (`backend/internal/services/pdf/templates.go`)

**Remaining (template management UI):**

- [ ] `backend/internal/api/handlers/invoice_template_handler.go` - Template CRUD
- [ ] `backend/internal/api/routes/invoice_template_routes.go` - Template route definitions
- [ ] Frontend template management UI

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
- [x] Membership system with Stripe integration (2025-01-25)
- [x] Subscription proration preview (2025-01-25)
- [x] Local email development setup with Mailpit (2025-01-25)
- [x] Security headers middleware (2025-01-25)
- [x] CORS configuration with environment-specific origins (2025-01-25)
- [x] Job status state machine with transition validation (2025-01-29)
- [x] Job status history tracking (2025-01-29)
- [x] PDF generation for quotes, invoices, and receipts (2025-01-30)
- [x] PDF attachments in quote/invoice/receipt emails (2025-01-30)
- [x] PDF download endpoints for quotes, invoices, receipts (2025-01-30)
- [x] Clone quote functionality (2025-01-30)
- [x] Quote accept/decline actions in UI (2025-01-30)
- [x] Dropdown menus with portal rendering (fixes table overflow clipping) (2025-01-30)
- [x] Cypress E2E testing setup with tests for auth, customers, jobs, invoices, quotes, payments

---

## Notes

### Priority Definitions

- **P0 (Critical)**: Blocks production deployment or poses security risk
- **P1 (High)**: Required for 1.0 release; significant user impact
- **P2 (Medium)**: Improves developer experience or code quality
- **P3 (Low)**: Nice-to-have improvements

### Adding New Items

When adding new items, include:

1. **What**: Clear description of what needs to be done
2. **Why**: Why it's important / what problem it solves
3. **Expected Result**: What success looks like
4. **Acceptance Criteria**: Specific, testable requirements

### Sprint Planning

- Sprints are 2 weeks
- Each sprint should have 1-2 P0/P1 items max
- Fill remaining capacity with P2/P3 items
- Always include at least one testing task

---

_Last updated: 2025-01-30_
