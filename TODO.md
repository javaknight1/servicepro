# ServicePro - Technical Debt & Future Work

This document tracks technical improvements that should be implemented but are deferred for future development cycles.

**Last Updated: 2026-02-03** (MVP tasks added from proposal document)

---

## Task Index

Quick reference for all pending tasks. Use the ID (e.g., "implement T001") to reference a task.

| ID       | Priority | Category       | Confidence | MVP    | Task                                               |
| -------- | -------- | -------------- | ---------- | ------ | -------------------------------------------------- |
| ~~T001~~ | ~~P0~~   | ~~Backend~~    | ~~High~~   | ~~--~~ | ~~Recovery middleware Sentry integration~~ ✓       |
| ~~T002~~ | ~~P0~~   | ~~Frontend~~   | ~~High~~   | ~~--~~ | ~~Centralize direct fetch() calls~~ ✓              |
| ~~T003~~ | ~~P0~~   | ~~Frontend~~   | ~~High~~   | ~~--~~ | ~~Add CSP/HSTS headers to nginx.conf~~ ✓           |
| ~~T004~~ | ~~P0~~   | ~~Infra~~      | ~~High~~   | ~~--~~ | ~~Create GitHub Actions CI/CD workflows~~ ✓        |
| ~~T005~~ | ~~P0~~   | ~~Backend~~    | ~~High~~   | ~~--~~ | ~~Expose /metrics endpoint for Prometheus~~ ✓      |
| ~~T006~~ | ~~P1~~   | ~~Backend~~    | ~~High~~   | ~~--~~ | ~~Structured JSON logging (replace fmt.Printf)~~ ✓ |
| ~~T007~~ | ~~P1~~   | ~~Frontend~~   | ~~High~~   | ~~--~~ | ~~Fix `any` types (50 instances)~~ ✓               |
| ~~T008~~ | ~~P1~~   | ~~Frontend~~   | ~~High~~   | ~~--~~ | ~~Remove @ts-nocheck directives~~ ✓                |
| ~~T009~~ | ~~P1~~   | ~~Frontend~~   | ~~High~~   | ~~--~~ | ~~Enable noUnusedLocals/noUnusedParameters~~ ✓     |
| ~~T010~~ | ~~P1~~   | ~~Backend~~    | ~~High~~   | ~~--~~ | ~~Apply error tracking middleware~~ ✓              |
| T012     | P1       | Infra          | High       | --     | Bundle size monitoring in CI                       |
| T013     | P2       | Testing        | High       | --     | Increase frontend test coverage to 70%+            |
| T014     | P2       | Testing        | High       | --     | Integration tests for critical workflows           |
| T015     | P2       | Docs           | High       | After  | Generate OpenAPI/Swagger documentation             |
| T016     | P2       | Perf           | High       | --     | Frontend bundle analysis + optimization            |
| T017     | P2       | Perf           | High       | --     | Query performance monitoring                       |
| T018     | P2       | Cleanup        | High       | --     | Dead code elimination                              |
| ~~T019~~ | ~~P0~~   | ~~Scheduling~~ | ~~High~~   | ~~--~~ | ~~Integrate calendar view for job scheduling~~ ✓   |
| T020     | P0       | Scheduling     | High       | Before | Build conflict detection API endpoint              |
| T021     | P2       | Refactor       | High       | --     | Extract duplicate file download utility            |
| T022     | P2       | Refactor       | High       | --     | Extract duplicate URLSearchParams builder          |
| T023     | P2       | Refactor       | High       | --     | Consolidate cache hooks (useLocalCache/useSession) |
| T024     | P1       | Observability  | High       | --     | Full Sentry integration (frontend + backend)       |
| T025     | P2       | Analytics      | High       | After  | Integrate product analytics (PostHog recommended)  |
| T026     | P2       | Analytics      | High       | After  | Set up business KPI dashboard (Metabase)           |
| T027     | P2       | Scheduling     | High       | Before | Add drag-and-drop rescheduling to job calendar     |
| T028     | P0       | Quoting        | High       | Before | Add "Convert Quote to Job" button                  |
| T029     | P0       | Invoicing      | High       | Before | Add "Generate Invoice from Job" button             |
| T030     | P0       | Invoicing      | High       | Before | Auto-populate invoice from job data                |
| T031     | P0       | Invoicing      | High       | Before | Build A/R aging buckets report                     |
| T032     | P0       | Invoicing      | High       | Before | Build "Who Owes What" report                       |
| T033     | P0       | Dashboard      | High       | Before | Add "Jobs Needing Invoice" dashboard widget        |
| T034     | P0       | Comms          | High       | Before | Wire payment reminder automation                   |
| T035     | P0       | Roles          | High       | Before | Create Technician role                             |
| T036     | P1       | CRM            | High       | Before | Add manual Call Log entry                          |
| T037     | P1       | CRM            | High       | Before | Auto-log system messages to customer activity      |
| T038     | P1       | Scheduling     | Medium     | Before | Add technician availability management UI          |
| T039     | P1       | Scheduling     | Medium     | Before | Create emergency job insertion workflow            |
| T040     | P1       | Scheduling     | High       | Before | Add auto-notify on emergency insertion             |
| T041     | P1       | Quoting        | Medium     | Before | Implement quote auto follow-ups                    |
| T042     | P1       | Quoting        | High       | Before | Build follow-up queue UI                           |
| T043     | P1       | Invoicing      | High       | Before | Wire payment reminder notifications                |
| T044     | P1       | Invoicing      | Medium     | Before | Build collections dashboard                        |
| T045     | P1       | Dashboard      | High       | Before | Add aging quotes dashboard widget                  |
| T046     | P1       | Dashboard      | High       | Before | Add follow-up queue dashboard widget               |
| T047     | P1       | Reporting      | Medium     | Before | Build "stuck work" report                          |
| T048     | P1       | Comms          | High       | Before | Add appointment confirmation emails                |
| T049     | P1       | Comms          | High       | Before | Add appointment reminder emails/SMS                |
| T050     | P1       | Comms          | High       | Before | Add "Tech on the way" manual trigger               |
| T051     | P1       | Comms          | Medium     | Before | Implement quote follow-up email sequence           |
| T052     | P1       | Roles          | High       | Before | Create Dispatcher role                             |
| T053     | P2       | CRM            | High       | Before | Add "Repeat Job" button on customer detail         |
| T054     | P2       | CRM            | Medium     | Before | Create Communication Log component                 |
| T055     | P2       | CRM            | Medium     | Before | Unified Customer Timeline view                     |
| T056     | P2       | Scheduling     | Medium     | Before | Add workload capacity warnings                     |
| T057     | P2       | Scheduling     | High       | Before | Add double-booking override with reason            |
| T058     | P2       | Scheduling     | Medium     | Before | Add required fields validation by job stage        |
| T059     | P2       | Scheduling     | High       | Before | Add technician skill tags                          |
| T060     | P2       | Quoting        | Medium     | Before | Create quote template system (good/better/best)    |
| T061     | P2       | Quoting        | Medium     | Before | Add required deposit to accept quote               |
| T062     | P2       | Quoting        | High       | Before | Add quote aging tracking                           |
| T063     | P2       | Invoicing      | Medium     | Before | Add deposit tracking and application               |
| T064     | P2       | Invoicing      | High       | Before | Add last contact date tracking on invoices         |
| T065     | P2       | Invoicing      | High       | Before | Add promise-to-pay notes                           |
| T066     | P2       | Dashboard      | Medium     | Before | Add technician utilization widget                  |
| T067     | P2       | Dashboard      | High       | Before | Add quote conversion rate metric                   |
| T068     | P2       | Reporting      | High       | Before | Build job-completion-to-invoice time report        |
| T069     | P2       | Reporting      | High       | Before | Build invoice-to-paid time report (DSO)            |
| T070     | P2       | Reporting      | High       | Before | Build revenue by service type report               |
| T071     | P2       | Reporting      | High       | Before | Build collections rate report                      |
| T072     | P2       | Comms          | High       | Before | Create message template management UI              |
| T073     | P2       | Comms          | High       | Before | Add rich variable system for templates             |
| T074     | P2       | Roles          | High       | Before | Create Accountant role                             |
| T075     | P2       | Roles          | Medium     | Before | Add field-level permission controls                |
| T076     | P3       | CRM            | High       | After  | Add "Recreate Last Quote" button                   |
| T077     | P3       | Scheduling     | Medium     | After  | Add time-off request UI                            |
| T078     | P3       | Scheduling     | Medium     | After  | Add skill-based job matching suggestions           |
| T079     | P3       | Scheduling     | High       | After  | Add travel time notes field                        |
| T080     | P3       | Quoting        | High       | After  | Create terms & conditions templates                |
| T081     | P3       | Quoting        | Medium     | After  | Add e-signature for quote acceptance               |
| T082     | P3       | Quoting        | Medium     | After  | Add "Create Quote from Job" button                 |
| T083     | P3       | Invoicing      | Medium     | After  | Implement late fee calculation                     |
| T084     | P3       | Reporting      | High       | After  | Build quote-to-job conversion report               |
| T085     | P3       | Reporting      | Medium     | After  | Build lead-to-quote time report                    |
| T086     | P3       | Reporting      | High       | After  | Build quote-to-acceptance time report              |
| T087     | P3       | Reporting      | Medium     | After  | Build revenue by technician report                 |
| T088     | P3       | Reporting      | Medium     | After  | Add comparative analysis (vs prior period)         |
| T089     | P3       | Comms          | Medium     | After  | Add email/SMS delivery tracking                    |
| T090     | P3       | Comms          | Medium     | After  | Add communication failure alerts                   |
| T091     | P3       | Comms          | Medium     | After  | Add notification preferences per customer          |
| T092     | P3       | Roles          | High       | After  | Implement 2FA (TOTP)                               |
| T093     | P3       | Roles          | Medium     | After  | Implement comprehensive audit logging              |
| T094     | P3       | Roles          | High       | After  | Add Google OAuth sign-in                           |
| T095     | P3       | Integration    | Medium     | After  | Build outgoing webhooks system                     |
| T096     | P3       | Integration    | High       | After  | Build event log table                              |
| T097     | P3       | Integration    | High       | After  | Add webhook retry logic                            |
| T098     | P3       | Integration    | High       | After  | Document public API (OpenAPI/Swagger)              |

---

## Sprint Roadmap

### Sprint 1 - Production Readiness (CURRENT)

**Backend Critical:**

- [x] Graceful shutdown with SIGTERM handling ✓
- [x] HTTP server timeouts (Read/Write/Idle) ✓
- [x] Apply MaxMultipartMemory to router ✓
- [x] Apply TrustedProxies to router ✓
- [x] Deep health checks (DB + Redis) ✓
- [x] **T001** - Recovery middleware Sentry integration ✓

**Frontend Critical:**

- [x] Fix localStorage token access in ConflictChecker.tsx ✓
- [x] **T002** - Centralize direct fetch() calls ✓
- [x] **T003** - Add CSP/HSTS headers to nginx.conf ✓

**Infrastructure:**

- [x] **T004** - Create GitHub Actions CI/CD workflows ✓

### Sprint 2 - Observability & Type Safety

- [x] **T005** - Expose /metrics endpoint for Prometheus ✓
- [x] **T006** - Structured logging (replace fmt.Printf) ✓
- [x] **T007** - Fix 50 `any` types in frontend ✓
- [x] **T008** - Remove @ts-nocheck directives ✓
- [x] **T009** - Enable noUnusedLocals/noUnusedParameters ✓

### Sprint 3 - Testing & Documentation

- [ ] **T013** - Increase frontend test coverage to 70%+
- [ ] **T014** - Integration tests for critical workflows
- [ ] **T015** - Generate OpenAPI/Swagger documentation

### Sprint 4 - Features & Cleanup

- [x] **T019** - Integrate calendar view for job scheduling ✓
- [ ] **T018** - Dead code elimination (partially complete - see audit)

### Sprint 5 - Performance & Refactoring

- [ ] **T016** - Bundle analysis + optimization
- [ ] **T017** - Query performance monitoring
- [ ] **T021** - Extract duplicate file download utility
- [ ] **T022** - Extract duplicate URLSearchParams builder
- [ ] **T023** - Consolidate cache hooks

### Sprint 6 - Scheduling & Conflict Detection

- [ ] **T020** - Build conflict detection API endpoint
- [ ] **T027** - Add drag-and-drop rescheduling to job calendar (depends on T020)
- [ ] **T038** - Add technician availability management UI
- [ ] **T056** - Add workload capacity warnings
- [ ] **T057** - Add double-booking override with reason
- [ ] **T059** - Add technician skill tags

### Sprint 7 - Core Workflow (Quote → Job → Invoice)

- [ ] **T028** - Add "Convert Quote to Job" button
- [ ] **T029** - Add "Generate Invoice from Job" button
- [ ] **T030** - Auto-populate invoice from job data
- [ ] **T033** - Add "Jobs Needing Invoice" dashboard widget
- [ ] **T053** - Add "Repeat Job" button on customer detail

### Sprint 8 - Collections & A/R

- [ ] **T031** - Build A/R aging buckets report
- [ ] **T032** - Build "Who Owes What" report
- [ ] **T034** - Wire payment reminder automation
- [ ] **T043** - Wire payment reminder notifications
- [ ] **T044** - Build collections dashboard
- [ ] **T064** - Add last contact date tracking on invoices
- [ ] **T065** - Add promise-to-pay notes
- [ ] **T071** - Build collections rate report

### Sprint 9 - Quoting Enhancements

- [ ] **T041** - Implement quote auto follow-ups
- [ ] **T042** - Build follow-up queue UI
- [ ] **T045** - Add aging quotes dashboard widget
- [ ] **T046** - Add follow-up queue dashboard widget
- [ ] **T051** - Implement quote follow-up email sequence
- [ ] **T060** - Create quote template system (good/better/best)
- [ ] **T061** - Add required deposit to accept quote
- [ ] **T062** - Add quote aging tracking

### Sprint 10 - Communications & Notifications

- [ ] **T048** - Add appointment confirmation emails
- [ ] **T049** - Add appointment reminder emails/SMS
- [ ] **T050** - Add "Tech on the way" manual trigger
- [ ] **T072** - Create message template management UI
- [ ] **T073** - Add rich variable system for templates

### Sprint 11 - Roles & Permissions

- [ ] **T035** - Create Technician role
- [ ] **T052** - Create Dispatcher role
- [ ] **T074** - Create Accountant role
- [ ] **T075** - Add field-level permission controls

### Sprint 12 - CRM & Customer Timeline

- [ ] **T036** - Add manual Call Log entry
- [ ] **T037** - Auto-log system messages to customer activity
- [ ] **T054** - Create Communication Log component
- [ ] **T055** - Unified Customer Timeline view

### Sprint 13 - Dashboard & Reporting

- [ ] **T047** - Build "stuck work" report
- [ ] **T066** - Add technician utilization widget
- [ ] **T067** - Add quote conversion rate metric
- [ ] **T068** - Build job-completion-to-invoice time report
- [ ] **T069** - Build invoice-to-paid time report (DSO)
- [ ] **T070** - Build revenue by service type report

### Sprint 14 - Job Workflow Enhancements

- [ ] **T039** - Create emergency job insertion workflow
- [ ] **T040** - Add auto-notify on emergency insertion
- [ ] **T058** - Add required fields validation by job stage
- [ ] **T063** - Add deposit tracking and application

### Sprint 15+ - Post-MVP Enhancements

- [ ] **T076** - Add "Recreate Last Quote" button
- [ ] **T077** - Add time-off request UI
- [ ] **T078** - Add skill-based job matching suggestions
- [ ] **T079** - Add travel time notes field
- [ ] **T080** - Create terms & conditions templates
- [ ] **T081** - Add e-signature for quote acceptance
- [ ] **T082** - Add "Create Quote from Job" button
- [ ] **T083** - Implement late fee calculation
- [ ] **T084** - Build quote-to-job conversion report
- [ ] **T085** - Build lead-to-quote time report
- [ ] **T086** - Build quote-to-acceptance time report
- [ ] **T087** - Build revenue by technician report
- [ ] **T088** - Add comparative analysis (vs prior period)
- [ ] **T089** - Add email/SMS delivery tracking
- [ ] **T090** - Add communication failure alerts
- [ ] **T091** - Add notification preferences per customer
- [ ] **T092** - Implement 2FA (TOTP)
- [ ] **T093** - Implement comprehensive audit logging
- [ ] **T094** - Add Google OAuth sign-in
- [ ] **T095** - Build outgoing webhooks system
- [ ] **T096** - Build event log table
- [ ] **T097** - Add webhook retry logic
- [ ] **T098** - Document public API (OpenAPI/Swagger)

---

## Verified Complete ✓

- [x] Rate limiting - Multi-tier with Redis backing, applied globally via DynamicRateLimit()
- [x] PDF generation - Quotes, Invoices, Receipts with S3 storage and email attachments
- [x] SMS integration - TextBelt, SNS, Mock providers with pluggable architecture
- [x] httpOnly cookies - SameSite=Lax for CSRF protection
- [x] Security headers (backend) - CSP, HSTS, X-Frame-Options in middleware
- [x] Error tracking client - Sentry integration at pkg/clients/errortracking
- [x] .env in .gitignore - Properly excluded (lines 42-50)
- [x] Console.log stripping - Terser `drop_console: true` in production build
- [x] Migrate `db.Raw()` to GORM (SQL injection risk)
- [x] Cypress E2E testing - Auth, customers, jobs, invoices, quotes, payments
- [x] Graceful shutdown - SIGTERM/SIGINT handling with 30s timeout in cmd/main.go
- [x] HTTP server timeouts - ReadTimeout, WriteTimeout, IdleTimeout, ReadHeaderTimeout configured via http.Server
- [x] MaxMultipartMemory - Configurable via SERVER_MAX_MULTIPART_MEMORY env var, applied to router
- [x] TrustedProxies - Configurable via SERVER_TRUSTED_PROXIES env var, applied to router
- [x] Deep health checks - /health, /health/live, /health/ready with DB + Redis connectivity checks
- [x] localStorage token fix - ConflictChecker.tsx now uses api service with httpOnly cookies
- [x] T001: Recovery middleware Sentry integration - Panics captured with stack trace, request context, user ID
- [x] T002: Centralized fetch() calls - pdfGenerator.tsx, useQuoteCalculations.ts now use api service
- [x] T003: nginx security headers - CSP, HSTS, Permissions-Policy added to nginx.conf
- [x] T004: GitHub Actions CI/CD - checks.yml, release.yml, deploy-release.yml already exist
- [x] T005: Prometheus /metrics endpoint - Enabled via PROMETHEUS_ENABLED=true, uses existing prometheus client
- [x] T006: Structured JSON logging - Migrated all fmt.Printf/log.Printf to structured logging client with global logger pattern
- [x] T007: Fix `any` types - Replaced 50+ `any` types with proper TypeScript types, created error utility, fixed all test mocks
- [x] T008: Remove @ts-nocheck - Removed directives from routes/index.tsx and performance.ts, added browser API type declarations
- [x] T009: Enable noUnusedLocals/noUnusedParameters - Enabled strict checks, removed 17 instances of dead code
- [x] T010: Error tracking middleware - Already implemented via RecoveryMiddleware in error_handler.go
- [x] T019: Calendar view for job scheduling - Read-only calendar at `/jobs/calendar`, click to view job details, status color mapping

---

## P0 - Critical (Blocks Production)

### Backend Server Configuration

- [x] **T001: Recovery Middleware Sentry Integration** ✓ COMPLETE
  - Updated `RecoveryMiddleware` to accept `errortracking.Client`
  - Panics are now captured with full context: stack trace, request method/path, user ID
  - Changed `gin.Default()` to `gin.New()` to use custom recovery instead of Gin's default
  - Middleware applied first in route setup to catch all panics

### Frontend Critical

- [x] **T002: Centralize Direct fetch() Calls** ✓ COMPLETE
  - All API calls now go through `frontend/src/services/api.ts`
  - Fixed: `pdfGenerator.tsx`, `useQuoteCalculations.ts`
  - No direct `fetch('/api/...')` calls remain

- [x] **T003: Add Security Headers to nginx.conf** ✓ COMPLETE
  - Added HSTS: `max-age=31536000; includeSubDomains`
  - Added CSP: `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; ...`
  - Added Permissions-Policy to disable unused browser features
  - Updated Referrer-Policy to `strict-origin-when-cross-origin`

### Infrastructure & Deployment

- [x] **T004: Create GitHub Actions CI/CD Workflows** ✓ COMPLETE
  - `checks.yml` - Lint (pre-commit), backend tests, frontend tests on push/PR
  - `release.yml` - Creates GitHub Release with changelog on version tags
  - `deploy-release.yml` - Tests + deploys to Fly.io (backend) and Cloudflare Pages (frontend) on version tags

### Observability - Critical

- [x] **T005: Expose /metrics Endpoint** ✓ COMPLETE
  - Added metrics client initialization in `cmd/main.go`
  - Added `/metrics` endpoint in `routes.go` (enabled via `PROMETHEUS_ENABLED=true`)
  - Uses existing Prometheus client at `pkg/clients/metrics/prometheus/`
  - Ready for Prometheus/Grafana scraping when monitoring is set up

### Scheduling - Critical

- [ ] **T020: Build Conflict Detection API Endpoint**
  - **What**: Implement `POST /v1/conflicts/check` endpoint for scheduling conflict detection
  - **Why**: Calendar view is useless without conflict warnings - prevents double-booking disasters
  - **Confidence**: High - clear requirements, frontend components already exist
  - **Context**: When dispatcher schedules a job, system should warn if tech is already booked, too far away, or overloaded
  - **Backend Requirements**:
    - Create `internal/services/conflict_service.go`
    - Create `internal/api/handlers/conflict_handler.go`
    - Wire endpoint in `routes.go`
  - **Conflict Types to Detect**:
    - Technician already booked (time overlap)
    - Outside business hours
    - Workload exceeds daily limit
  - **Acceptance Criteria**:
    - Endpoint returns conflicts in <100ms
    - All conflict types detected
    - Suggestions provided for resolution

### Quoting - Critical

- [ ] **T028: Add "Convert Quote to Job" Button**
  - **What**: One-click conversion from accepted quote to scheduled job
  - **Why**: Core workflow - customer accepts quote, office manager needs to immediately schedule the work
  - **Confidence**: High - straightforward data mapping between quote and job
  - **Context**: Field service businesses live and die by this flow: quote accepted → job scheduled → work done → invoice sent
  - **Implementation**:
    - Add button on QuoteDetailPage when status is "accepted"
    - Create job with customer, service type, line items from quote
    - Navigate to job detail or scheduling view
  - **Acceptance Criteria**:
    - Button visible only on accepted quotes
    - Job created with all relevant quote data
    - User can immediately schedule the new job

### Invoicing - Critical

- [ ] **T029: Add "Generate Invoice from Job" Button**
  - **What**: One-click invoice generation from completed job
  - **Why**: Core workflow - prevents "completed but not billed" revenue leakage
  - **Confidence**: High - clear data flow from job to invoice
  - **Context**: Techs complete work, office forgets to bill. This button makes billing instant.
  - **Implementation**:
    - Add button on JobDetailPage when status is "completed"
    - Pre-populate invoice with job data (see T030)
  - **Acceptance Criteria**:
    - Button visible only on completed jobs without invoices
    - Invoice created and linked to job
    - User navigated to invoice for review/send

- [ ] **T030: Auto-populate Invoice from Job Data**
  - **What**: Pull labor hours, materials, notes, photos into invoice automatically
  - **Why**: Eliminates manual data entry, ensures nothing is missed
  - **Confidence**: High - models already have the fields
  - **Context**: Tech logged "2 hours labor, replaced capacitor $45, customer approved additional work" → invoice has all that
  - **Implementation**:
    - Map job.materials → invoice line items
    - Calculate labor from job time tracking
    - Include job notes in invoice description
  - **Acceptance Criteria**:
    - All job materials appear as line items
    - Labor calculated from actual time
    - Job notes/photos accessible from invoice

- [ ] **T031: Build A/R Aging Buckets Report**
  - **What**: Report showing receivables by age: 0-30, 31-60, 61-90, 90+ days
  - **Why**: Essential for cash flow management - "how much money is stuck and for how long?"
  - **Confidence**: High - standard accounting report, clear requirements
  - **Context**: Owner opens report, sees "$5,000 current, $2,000 over 30 days, $500 over 60 days" - knows where to focus collection efforts
  - **Implementation**:
    - Query invoices grouped by days since due date
    - Display totals per bucket
    - Drill-down to see individual invoices
  - **Acceptance Criteria**:
    - Buckets: Current, 1-30, 31-60, 61-90, 90+ days
    - Click bucket to see invoices in that range
    - Export to CSV

- [ ] **T032: Build "Who Owes What" Report**
  - **What**: Customer-centric view of outstanding balances
  - **Why**: Collection prioritization - focus on customers with largest balances
  - **Confidence**: High - simple aggregation query
  - **Context**: "ABC Company owes $3,000 across 3 invoices, last payment 45 days ago" - actionable collection info
  - **Implementation**:
    - Group unpaid invoices by customer
    - Show total owed, invoice count, oldest invoice, last payment date
    - Sort by amount descending
  - **Acceptance Criteria**:
    - Shows all customers with outstanding balances
    - Click customer to see their invoices
    - Filter by amount threshold

### Dashboard - Critical

- [ ] **T033: Add "Jobs Needing Invoice" Dashboard Widget**
  - **What**: Count and list of completed jobs that haven't been invoiced
  - **Why**: Prevents revenue leakage - "You have 7 jobs to bill!"
  - **Confidence**: High - simple query: jobs where status=completed AND no linked invoice
  - **Context**: Money left on the table is the #1 cash flow killer for small service businesses
  - **Implementation**:
    - Dashboard widget showing count
    - Click to see list of unbilled jobs
    - Quick action to generate invoice
  - **Acceptance Criteria**:
    - Accurate count of completed-but-unbilled jobs
    - One-click navigation to job list
    - Prominent placement on dashboard

### Communications - Critical

- [ ] **T034: Wire Payment Reminder Automation**
  - **What**: Background job that sends automatic reminders for overdue invoices
  - **Why**: Getting paid is the business - automated reminders improve collection rate 30-40%
  - **Confidence**: High - payment notification models exist, just need scheduler
  - **Context**: Invoice 7 days overdue → auto-email "Your payment is past due." 14 days → another. 30 days → escalation.
  - **Implementation**:
    - Create background worker/cron job
    - Query overdue invoices daily
    - Send reminders at 7, 14, 30 days (configurable)
    - Track which reminders were sent
  - **Acceptance Criteria**:
    - Reminders sent automatically
    - Configurable timing (days overdue)
    - No duplicate reminders
    - Reminder history tracked

### Roles - Critical

- [ ] **T035: Create Technician Role**
  - **What**: Role for field workers with limited access
  - **Why**: Techs need to see their jobs, add notes/photos, update status - but NOT see customer financials or other techs' schedules
  - **Confidence**: High - RBAC system exists, just need role definition
  - **Context**: Tech opens app, sees "Your jobs today: 3", can mark complete, add photos, log time. Cannot see "Customer owes $5,000"
  - **Permissions to include**:
    - View assigned jobs only
    - Update job status
    - Add job notes and photos
    - Log time/materials
  - **Permissions to exclude**:
    - View other technicians' jobs
    - View customer financial data
    - Create/edit customers
    - Access reports
  - **Acceptance Criteria**:
    - Role created with correct permissions
    - Field-level hiding of financial data
    - Tech can only see their assignments

---

## P1 - High Priority

### Frontend Type Safety

- [x] **T007: Fix `any` Types** ✓ COMPLETE
  - Replaced 50+ `any` types with proper TypeScript types
  - Created `src/utils/error.ts` utility for type-safe error handling
  - Fixed type definition files: template.ts, chart.ts, recurring/types.ts, calendar/types.ts
  - Fixed page components: CustomerDetailPage, InvoiceDetailPage, QuoteDetailPage, JobDetailPage, etc.
  - Fixed components: FilterBar.tsx, TemplatePreview.tsx, Calendar.tsx, RecurringForm.tsx, etc.
  - Fixed stores: tenantStore.ts with proper error handling
  - Fixed all test mocks with proper TypeScript interfaces (MockCardElementProps, etc.)

- [x] **T008: Remove @ts-nocheck Directives** ✓ COMPLETE
  - Removed `@ts-nocheck` from both files:
    - [x] `frontend/src/routes/index.tsx` - Removed 35+ `@ts-expect-error` comments from dynamic imports
    - [x] `frontend/src/utils/performance.ts` - Added proper browser API type declarations
  - Fixed `loadable()` function type signature to handle dynamic import union types
  - Added browser API interfaces: `PerformanceEventTiming`, `LayoutShiftEntry`, `ExtendedPerformanceObserverInit`, `DocumentWithPrerendering`
  - All type errors resolved, type-check passes

- [x] **T009: Enable Strict TypeScript Checks** ✓ COMPLETE
  - Enabled `noUnusedLocals: true` and `noUnusedParameters: true` in tsconfig.json
  - Fixed 17 issues:
    - Removed 7 unused `React` imports (modern JSX transform doesn't need them)
    - Removed unused variables: `_chartRef`, `_handleItemsChange`, `_bulkAssignMutation`, `_formatValue`, `_navigate`, `_API_BASE_URL`
    - Removed unused `_offlineFallback` Route (dead code in service-worker.ts)
    - Prefixed unused callback parameters with `_` (TypeScript convention)
  - Type-check passes, dead code will now be caught at compile time

### Backend Observability

- [x] **T006: Structured JSON Logging** ✓ COMPLETE
  - Implemented custom logging client at `pkg/clients/logging/` with multiple providers (Mock, BetterStack, CloudWatch, Loki)
  - Added global logger pattern with `SetDefault()` and `L()` functions
  - Migrated all `fmt.Printf` and `log.Printf` calls to structured logging
  - JSON format in production, human-readable in development
  - Standard fields: timestamp, level, message, request_id, user_id, tenant_id
  - Only exceptions: config/config.go (circular import), logging package files (can't import itself)

- [x] **T010: Apply Error Tracking Middleware** ✓ ALREADY COMPLETE
  - **Analysis**: `RecoveryMiddleware` in `error_handler.go` already handles this
  - **What it does**:
    - Captures panics and sends to error tracking (Sentry)
    - Adds full request context: method, path, query, client IP
    - Attaches user context when available
    - Applied first in middleware chain in `routes.go`
  - **Why HTTPMiddleware not needed**: No application code calls `CaptureException` directly outside of panic recovery. The existing implementation covers the critical use case.

### Infrastructure

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

### Frontend Features

- [x] **T019: Integrate Calendar View for Job Scheduling** ✓ COMPLETE
  - **What**: Wire up the existing Calendar components to display and manage jobs
  - **Why**: Calendar components are 100% complete but not integrated - critical for service business scheduling
  - **Implementation**:
    - Created `JobCalendarPage` at `/jobs/calendar` route
    - Uses existing `GET /v1/jobs/scheduled?start=&end=` backend endpoint
    - Converts Job data to calendar events with status color mapping
    - Click event navigates to job detail page
    - Drag-and-drop disabled for now (see T027)
  - **Files Created/Modified**:
    - `frontend/src/pages/Jobs/JobCalendarPage.tsx` - New calendar page component
    - `frontend/src/services/jobService.ts` - Added `getScheduledJobs()` method
    - `frontend/src/components/calendar/Calendar.tsx` - Disabled drag-drop
    - `frontend/src/components/calendar/types.ts` - Added status mapping function
    - `frontend/src/routes/index.tsx` - Added `/jobs/calendar` route

- [ ] **T024: Full Sentry Integration (Frontend + Backend)**
  - **What**: Complete error tracking setup with Sentry across the entire stack
  - **Why**: Currently backend catches panics but frontend errors are invisible; need unified error visibility
  - **Current State**:
    - Backend: `RecoveryMiddleware` sends panics to Sentry ✓
    - Backend: `@sentry/react` package installed but not initialized
    - Frontend: No error boundaries or user context
  - **Frontend Tasks**:
    - [ ] Initialize Sentry in `main.tsx` with proper config
    - [ ] Add `Sentry.ErrorBoundary` around App component
    - [ ] Create `useErrorTracking` hook for manual error capture
    - [ ] Attach user context on login (`Sentry.setUser`)
    - [ ] Configure source maps upload in build process
    - [ ] Add performance monitoring (optional)
  - **Backend Tasks**:
    - [ ] Add `Sentry.CaptureException` for non-panic errors in services
    - [ ] Add request context to all captured errors
    - [ ] Configure release tracking
  - **Local Development Setup**:

    ```yaml
    # docker-compose.yml addition for self-hosted Sentry (optional)
    # Alternative: Use Sentry.io free tier (10k errors/month)
    sentry:
      image: sentry:latest
      # ... complex setup, see getsentry/self-hosted
    ```

    - Simpler option: Use Sentry.io with development DSN
    - Set `SENTRY_DSN` env var, use `SENTRY_ENVIRONMENT=development`

  - **Testing**:
    - Add test error button in dev mode
    - Verify errors appear in Sentry dashboard
    - Verify stack traces are readable (source maps working)
  - **Acceptance Criteria**:
    - All unhandled frontend errors captured
    - All backend panics and explicit errors captured
    - User ID attached to errors
    - Readable stack traces with source maps
    - Environment separation (dev/staging/prod)

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

- [x] **Structured JSON Logging** ✓ COMPLETE (See T006)

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

- [x] **Remove Console.log Statements** ✓ VERIFIED COMPLETE
  - **Status**: Production build already strips console.log via Terser
  - **Config**: `frontend/config/optimization.ts:64` - `drop_console: process.env.NODE_ENV === 'production'`
  - **Note**: 189 console statements exist in source but are removed in production build

- [ ] **Fix @ts-nocheck in Routes** (MOVED TO P1 - Type Safety)
  - See P1 section for details

### CRM - High Priority

- [ ] **T036: Add Manual Call Log Entry**
  - **What**: Form to log phone calls with customers (date, duration, notes, outcome)
  - **Why**: Team coordination - prevents two people calling same customer, tracks communication history
  - **Confidence**: High - simple CRUD with form
  - **Context**: "Called Mrs. Smith about her quote, left voicemail, will call back tomorrow" - everyone on team can see this
  - **Implementation**:
    - Add CallLog model (customer_id, user_id, timestamp, duration, notes, outcome)
    - Add form on customer detail page
    - Display in customer activity timeline
  - **Acceptance Criteria**:
    - Can log calls from customer detail page
    - Call appears in customer timeline
    - Shows who made the call and when

- [ ] **T037: Auto-log System Messages to Customer Activity**
  - **What**: Automatically create activity log entries when system sends communications
  - **Why**: No more "did the quote go out?" questions - everything is logged
  - **Confidence**: High - hook into existing email/SMS send functions
  - **Context**: Quote sent → activity log shows "Quote #123 sent via email". Invoice paid → "Payment received $500"
  - **Implementation**:
    - Hook email service to log sends
    - Hook SMS service to log sends
    - Log payment events
    - Log quote/invoice status changes
  - **Acceptance Criteria**:
    - All outbound communications logged
    - All payment events logged
    - All status changes logged
    - Shows in customer timeline

### Scheduling - High Priority

- [ ] **T038: Add Technician Availability Management UI**
  - **What**: UI for techs to set working hours and for managers to view availability
  - **Why**: Dispatchers need to know who's available when scheduling
  - **Confidence**: Medium - unclear if this is simple working hours or complex shift patterns
  - **Context**: "Mike works M-F 8-5, Sarah works T-Sat 7-4" - dispatcher sees this when assigning
  - **Implementation**:
    - Add availability settings per technician
    - Display availability on scheduling calendar
    - Warn when scheduling outside available hours
  - **Acceptance Criteria**:
    - Techs can set their working hours
    - Dispatchers can view tech availability
    - Warning when scheduling outside hours

- [ ] **T039: Create Emergency Job Insertion Workflow**
  - **What**: Special flow for high-priority emergency jobs that may bump existing appointments
  - **Why**: Pipe burst, AC died in summer - customer needs someone TODAY
  - **Confidence**: Medium - exact workflow unclear (simple priority flag vs complex rescheduling)
  - **Context**: Emergency call comes in → mark as emergency → system helps find slot, may bump existing job
  - **Implementation**:
    - Add "emergency" priority level
    - Show available slots accounting for travel time
    - Option to bump existing non-emergency jobs
    - Notify affected customers if bumped
  - **Acceptance Criteria**:
    - Can mark job as emergency
    - System suggests available slots
    - Can reschedule existing jobs with notification

- [ ] **T040: Add Auto-notify on Emergency Insertion**
  - **What**: Automatically notify customer and tech when emergency job is inserted
  - **Why**: Keep everyone informed when schedules change
  - **Confidence**: High - straightforward notification trigger
  - **Context**: Emergency bumps existing job → affected customer gets "We need to reschedule" message
  - **Implementation**:
    - Trigger notification when job bumped
    - Send to affected customer
    - Send to affected technician
  - **Acceptance Criteria**:
    - Affected parties notified automatically
    - Clear message about rescheduling
    - Option to call instead of auto-message

### Quoting - High Priority

- [ ] **T041: Implement Quote Auto Follow-ups**
  - **What**: Automated email sequence for quotes that haven't been responded to
  - **Why**: Consistent follow-up improves close rate by 20-30%
  - **Confidence**: Medium - cadence and messaging unclear
  - **Context**: Quote sent Monday, no response → Day 3: "Following up" → Day 7: "Any questions?" → Day 14: "Quote expiring soon"
  - **Implementation**:
    - Background job to check quote ages
    - Send follow-up at configurable intervals
    - Track follow-ups sent
    - Stop when quote accepted/declined
  - **Acceptance Criteria**:
    - Follow-ups sent automatically
    - Configurable timing
    - No follow-ups after response
    - Track follow-up history

- [ ] **T042: Build Follow-up Queue UI**
  - **What**: List view of quotes needing manual follow-up
  - **Why**: Organized view for sales efforts - "Call these 5 customers today"
  - **Confidence**: High - straightforward list with filters
  - **Context**: Manager opens queue, sees quotes sorted by age/value, assigns follow-up tasks
  - **Implementation**:
    - List quotes pending response
    - Sort by age, value, customer
    - Show last contact date
    - Quick actions (call logged, email sent)
  - **Acceptance Criteria**:
    - Shows all pending quotes
    - Sortable/filterable
    - Log follow-up actions

### Invoicing - High Priority

- [ ] **T043: Wire Payment Reminder Notifications**
  - **What**: Connect payment notification models to actual email/SMS sending
  - **Why**: Models exist but aren't wired - reminders don't actually go out
  - **Confidence**: High - infrastructure exists, just needs wiring
  - **Implementation**:
    - Connect notification service to email client
    - Connect to SMS client
    - Use existing notification templates
  - **Acceptance Criteria**:
    - Payment reminders actually sent
    - Both email and SMS working
    - Delivery tracked

- [ ] **T044: Build Collections Dashboard**
  - **What**: Unified view for managing overdue accounts
  - **Why**: Central place for collection efforts
  - **Confidence**: Medium - could be simple list or complex workflow tool
  - **Context**: Collections person opens dashboard, sees all overdue accounts, prioritizes calls
  - **Implementation**:
    - List overdue invoices
    - Show customer contact info
    - Track collection activities
    - Log promises-to-pay
  - **Acceptance Criteria**:
    - All overdue accounts visible
    - Contact info accessible
    - Can log collection activities

### Dashboard - High Priority

- [ ] **T045: Add Aging Quotes Dashboard Widget**
  - **What**: Widget showing count of quotes older than X days
  - **Why**: Quick visibility into stale pipeline
  - **Confidence**: High - simple count query
  - **Context**: "12 quotes over 7 days old" - prompts follow-up action
  - **Implementation**:
    - Query quotes by age
    - Display count in widget
    - Click to see list
  - **Acceptance Criteria**:
    - Shows count of aging quotes
    - Configurable threshold (7, 14, 30 days)
    - Links to quote list

- [ ] **T046: Add Follow-up Queue Dashboard Widget**
  - **What**: Widget showing items needing follow-up today
  - **Why**: Daily action list at a glance
  - **Confidence**: High - simple count with link
  - **Context**: "8 items need follow-up today" - click to see list
  - **Implementation**:
    - Count quotes + invoices needing follow-up
    - Display on dashboard
    - Link to follow-up queue
  - **Acceptance Criteria**:
    - Accurate count
    - Links to queue

- [ ] **T047: Build "Stuck Work" Report**
  - **What**: Report showing work stuck at various stages
  - **Why**: Catches things falling through cracks
  - **Confidence**: Medium - "stuck" definition needs clarification
  - **Context**: Three lists: completed-not-invoiced, quotes-not-followed-up, overdue-not-contacted
  - **Implementation**:
    - Query for each "stuck" category
    - Display counts and lists
    - Quick actions to unstick
  - **Acceptance Criteria**:
    - Shows completed jobs without invoices
    - Shows quotes without follow-up
    - Shows overdue invoices without recent contact

### Communications - High Priority

- [ ] **T048: Add Appointment Confirmation Emails**
  - **What**: Send confirmation email when job is scheduled
  - **Why**: Sets customer expectations, reduces no-shows
  - **Confidence**: High - standard notification
  - **Context**: Job scheduled → "Your appointment is confirmed for Feb 5 at 2pm"
  - **Implementation**:
    - Trigger on job scheduled
    - Template with job details
    - Include reschedule/cancel link
  - **Acceptance Criteria**:
    - Email sent when job scheduled
    - Contains date, time, service type
    - Professional template

- [ ] **T049: Add Appointment Reminder Emails/SMS**
  - **What**: Send reminders before scheduled appointments
  - **Why**: Reduces no-shows by 25-30%
  - **Confidence**: High - standard pattern
  - **Context**: Day before: "Reminder: technician arriving tomorrow 2pm". Morning of: "Tech arriving in 2 hours"
  - **Implementation**:
    - Background job to check upcoming jobs
    - Send at configurable intervals (24hr, 2hr)
    - Support both email and SMS
  - **Acceptance Criteria**:
    - Reminders sent automatically
    - Configurable timing
    - Customer can opt out

- [ ] **T050: Add "Tech on the Way" Manual Trigger**
  - **What**: Button for tech to notify customer they're en route
  - **Why**: Customer knows to be ready, improves experience
  - **Confidence**: High - simple button + notification
  - **Context**: Tech clicks button → customer gets "Your technician is on the way!"
  - **Implementation**:
    - Button on tech's job view
    - Send SMS to customer
    - Log in job activity
  - **Acceptance Criteria**:
    - One-click send
    - SMS delivered to customer
    - Logged in job history

- [ ] **T051: Implement Quote Follow-up Email Sequence**
  - **What**: Multi-email sequence for quote follow-up
  - **Why**: Automated nurturing of pending quotes
  - **Confidence**: Medium - sequence rules need definition
  - **Context**: Day 3: check-in, Day 7: address concerns, Day 14: expiration warning
  - **Implementation**:
    - Define email templates for each touch
    - Schedule sends based on quote age
    - Stop on response
  - **Acceptance Criteria**:
    - Sequence runs automatically
    - Templates customizable
    - Stops when quote resolved

### Roles - High Priority

- [ ] **T052: Create Dispatcher Role**
  - **What**: Role for office staff who manage scheduling
  - **Why**: Dispatchers need schedule access but not financial visibility
  - **Confidence**: High - clear permission boundaries
  - **Context**: Dispatcher can see all jobs, assign techs, reschedule - but can't see invoice amounts or customer payment history
  - **Permissions to include**:
    - View all jobs
    - Create/edit job schedules
    - Assign technicians
    - View customer contact info
  - **Permissions to exclude**:
    - View invoice amounts
    - View payment history
    - Access financial reports
    - Modify pricing
  - **Acceptance Criteria**:
    - Role created with correct permissions
    - Financial data hidden from dispatchers

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

### Frontend Refactoring (Code Deduplication)

- [ ] **T021: Extract Duplicate File Download Utility**
  - **What**: Create `src/utils/fileDownload.ts` utility
  - **Why**: Same download pattern copy-pasted in 7+ locations
  - **Files with Duplicate Code**:
    - `services/exportService.ts`
    - `services/revenueService.ts`
    - `services/customerReportService.ts`
    - `services/invoiceService.ts`
    - `services/templateService.ts`
    - `components/quotes/utils/pdfGenerator.tsx`
    - `hooks/useRoles.ts`
  - **Implementation**:

    ```typescript
    // src/utils/fileDownload.ts
    export function downloadFile(blob: Blob, filename: string): void {
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', filename);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
    }

    export function downloadFromResponse(
      response: Response,
      filename: string
    ): Promise<void>;
    export function downloadCSV(data: string, filename: string): void;
    export function downloadJSON(data: object, filename: string): void;
    ```

  - **Estimated Savings**: ~100 lines of duplicate code

- [ ] **T022: Extract Duplicate URLSearchParams Builder**
  - **What**: Create `src/utils/queryParams.ts` utility
  - **Why**: Same pattern for building query params in 4+ services
  - **Files with Duplicate Code**:
    - `services/customerReportService.ts` (6 instances)
    - `services/revenueService.ts` (4 instances)
    - `services/invoiceService.ts`
    - `services/customerService.ts`
  - **Implementation**:
    ```typescript
    // src/utils/queryParams.ts
    export function buildQueryParams(
      obj: Record<string, unknown>
    ): URLSearchParams {
      const params = new URLSearchParams();
      Object.entries(obj).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          params.append(key, String(value));
        }
      });
      return params;
    }
    ```
  - **Estimated Savings**: ~80 lines of duplicate code

- [ ] **T023: Consolidate Cache Hooks**
  - **What**: Combine `useLocalCache` and `useSessionCache` into generic hook
  - **Why**: Nearly identical implementations, only differ in storage backend
  - **File**: `src/hooks/useCache.ts`
  - **Implementation**:

    ```typescript
    function useStorageCache<T>(
      storage: Storage,
      key: string,
      defaultValue: T
    ): [T, (value: T) => void, () => void];

    export const useLocalCache = <T>(key: string, defaultValue: T) =>
      useStorageCache(localStorage, key, defaultValue);

    export const useSessionCache = <T>(key: string, defaultValue: T) =>
      useStorageCache(sessionStorage, key, defaultValue);
    ```

  - **Estimated Savings**: ~50 lines of duplicate code

### Analytics & Observability

- [ ] **T025: Integrate Product Analytics (PostHog Recommended)**
  - **What**: Add product analytics to understand user behavior
  - **Why**: Know which features users use, where they drop off, what to improve
  - **Who Uses This**: Product managers, founders, growth team (NOT DevOps, NOT customers)
  - **Recommended Service**: PostHog (open source, can self-host, generous free tier)
  - **Alternatives**: Mixpanel, Amplitude, Plausible
  - **Implementation**:

    ```typescript
    // Install: npm install posthog-js
    // In main.tsx:
    import posthog from 'posthog-js';
    posthog.init('your-api-key', { api_host: 'https://app.posthog.com' });

    // Track events:
    posthog.capture('job_created', { job_type: 'hvac', customer_id: '...' });
    ```

  - **What to Track**:
    - Feature usage (which pages, which actions)
    - Funnel completion (onboarding, job creation, invoicing)
    - Search queries
    - Time spent on pages
  - **Acceptance Criteria**:
    - PostHog (or alternative) integrated
    - Key events tracked (job created, invoice sent, quote accepted)
    - Dashboard showing feature usage
    - Funnel visualization for critical flows

- [ ] **T026: Set Up Business KPI Dashboard (Metabase)**
  - **What**: Visual dashboard for business metrics without building custom endpoints
  - **Why**: Track revenue, job counts, customer growth without backend development
  - **Who Uses This**: Business owners, managers (customer-facing or internal)
  - **Recommended Service**: Metabase (open source, connects directly to PostgreSQL)
  - **Alternatives**: Redash, Apache Superset, Cube.js (for embedding)
  - **Setup**:
    ```yaml
    # docker-compose.yml
    metabase:
      image: metabase/metabase:latest
      ports:
        - '3001:3000'
      environment:
        MB_DB_TYPE: postgres
        MB_DB_HOST: db
        MB_DB_PORT: 5432
        MB_DB_DBNAME: servicepro
        MB_DB_USER: ${DB_USER}
        MB_DB_PASS: ${DB_PASSWORD}
      depends_on:
        - db
    ```
  - **Example KPIs to Create** (in Metabase SQL):
    - Revenue this month vs last month
    - Jobs completed per week
    - Average job value
    - Customer acquisition trend
    - Outstanding invoice total
  - **Embedding** (optional): Metabase charts can be embedded in your app
  - **Acceptance Criteria**:
    - Metabase running and connected to database
    - 5+ KPI dashboards created
    - Accessible to business stakeholders
    - (Optional) Key charts embedded in app dashboard

- [ ] **T027: Add Drag-and-Drop Rescheduling to Job Calendar**
  - **What**: Enable drag-and-drop to reschedule jobs on the calendar view
  - **Why**: Allow quick rescheduling without navigating to job detail page
  - **Depends On**: T020 (Conflict Detection API) - should validate conflicts before saving
  - **Current State**: Calendar view (T019) is implemented as read-only; drag-drop code is commented out
  - **Implementation**:
    - Re-enable `onEventDrop` and `onEventResize` handlers in `Calendar.tsx`
    - Integrate `ConflictChecker` component before confirming reschedule
    - Call `jobService.updateJob()` with new `scheduled_start_at`/`scheduled_end_at`
    - Show success/error toast notifications
  - **Files to Modify**:
    - `frontend/src/components/calendar/Calendar.tsx` - Uncomment drag-drop handlers
    - `frontend/src/pages/Jobs/JobCalendarPage.tsx` - Add drop/resize handlers
  - **Acceptance Criteria**:
    - Drag event to new time slot updates job schedule
    - Resize event adjusts job duration
    - Conflict warnings shown before saving (requires T020)
    - Undo option or confirmation dialog before save
    - Loading state during save operation

- [ ] **T018: Dead Code Elimination** (Partially Complete)
  - **What**: Remove unused components, hooks, and utilities identified in audit
  - **Why**: ~2,700 lines of dead code identified
  - **Already Deleted (2026-02-02)**:
    - [x] `src/components/recurring/` - Unused recurring job components
    - [x] `src/components/subscriptions/` - Unused billing/subscription components
    - [x] `src/components/health/` - DevOps tool, no backend, use Prometheus/Grafana instead
    - [x] `src/utils/performance.ts` - Use Sentry Performance or Vercel Analytics instead
    - [x] `src/hooks/useErrorTracking.ts` - Will rebuild with proper Sentry integration (T024)
    - [x] `src/hooks/useAnalytics.ts` - Use PostHog instead (T025)
    - [x] `src/hooks/useKPI.ts` - Use Metabase instead (T026)
  - **Remaining to Delete**: None - all dead code cleaned up!
  - **Components to Keep** (valuable, need integration):
    - `src/components/calendar/` - See T019
    - `src/components/scheduling/` - See T020
  - **Acceptance Criteria**:
    - All dead code removed
    - No TypeScript errors after removal
    - Bundle size reduced

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

### CRM - Medium Priority

- [ ] **T053: Add "Repeat Job" Button on Customer Detail**
  - **What**: Button to create new job from last job for a customer
  - **Why**: Service businesses do recurring work (annual maintenance, quarterly pest control)
  - **Confidence**: High - straightforward clone operation
  - **Context**: Customer calls, wants same service as last time → one click → job created with same details
  - **Acceptance Criteria**:
    - Button on customer detail page
    - Creates job with same service type, notes, duration
    - Opens job for scheduling

- [ ] **T054: Create Communication Log Component**
  - **What**: Unified view of all communications with a customer
  - **Why**: "Did anyone call Mrs. Smith?" - shows all emails, SMS, calls in one place
  - **Confidence**: Medium - unclear if needs threaded view or simple list
  - **Acceptance Criteria**:
    - Shows emails, SMS, calls
    - Chronological order
    - Links to related records

- [ ] **T055: Unified Customer Timeline View**
  - **What**: Single scrollable timeline: jobs, quotes, invoices, payments, communications
  - **Why**: "At a glance" customer history for context in conversations
  - **Confidence**: Medium - complexity depends on richness of interaction
  - **Acceptance Criteria**:
    - All customer activity in one timeline
    - Clickable items link to detail pages
    - Filterable by type

### Scheduling - Medium Priority

- [ ] **T056: Add Workload Capacity Warnings**
  - **What**: Warning when technician exceeds daily capacity
  - **Why**: Prevents overloading techs
  - **Confidence**: Medium - threshold logic unclear (8 hours? configurable?)
  - **Context**: "John already has 8 hours scheduled today" when adding more
  - **Acceptance Criteria**:
    - Warning shown when over threshold
    - Configurable threshold
    - Can override with acknowledgment

- [ ] **T057: Add Double-booking Override with Reason**
  - **What**: Allow scheduling conflicts with mandatory reason
  - **Why**: Sometimes you NEED two techs on same job, or emergency overrides
  - **Confidence**: High - simple modal with text field
  - **Acceptance Criteria**:
    - Override option when conflict detected
    - Reason field required
    - Override logged for audit

- [ ] **T058: Add Required Fields Validation by Job Stage**
  - **What**: Enforce required data at each job status transition
  - **Why**: Techs can't mark "complete" without notes/photos
  - **Confidence**: Medium - exact requirements per stage unclear
  - **Context**: Completion requires notes, photos, time logged. Prevents rushing without documentation.
  - **Acceptance Criteria**:
    - Configurable required fields per status
    - Validation on status change
    - Clear error messages

- [ ] **T059: Add Technician Skill Tags**
  - **What**: Tag techs with skills (HVAC, plumbing, electrical)
  - **Why**: Used for smart assignment suggestions
  - **Confidence**: High - simple tagging system
  - **Acceptance Criteria**:
    - Can add/remove skill tags
    - Skills visible on tech profile
    - Filter techs by skill

### Quoting - Medium Priority

- [ ] **T060: Create Quote Template System (Good/Better/Best)**
  - **What**: Pre-built quote templates with tiered options
  - **Why**: Upsell pattern - "Basic AC Tune-up $149, Premium $249, Complete $399"
  - **Confidence**: Medium - complexity in template builder UI
  - **Acceptance Criteria**:
    - Create/edit templates
    - Templates have line items
    - One-click apply template to quote

- [ ] **T061: Add Required Deposit to Accept Quote**
  - **What**: Option to require payment before quote acceptance
  - **Why**: Common for big jobs - "Pay $200 deposit to confirm"
  - **Confidence**: Medium - integrates with Stripe checkout
  - **Acceptance Criteria**:
    - Configurable deposit amount/percentage
    - Quote acceptance triggers payment
    - Deposit tracked separately

- [ ] **T062: Add Quote Aging Tracking**
  - **What**: Track how long quotes have been pending
  - **Why**: Helps prioritize follow-ups
  - **Confidence**: High - simple date math
  - **Acceptance Criteria**:
    - Age displayed on quote list
    - Sortable by age
    - Filter by age range

### Invoicing - Medium Priority

- [ ] **T063: Add Deposit Tracking and Application**
  - **What**: Track deposits separately, apply to final invoice
  - **Why**: Proper accounting for deposits
  - **Confidence**: Medium - accounting complexity
  - **Context**: Customer paid $500 deposit → Final invoice shows $2000 total, $500 deposit, $1500 due
  - **Acceptance Criteria**:
    - Deposits tracked separately
    - Applied to invoice automatically
    - Clear in invoice display

- [ ] **T064: Add Last Contact Date Tracking on Invoices**
  - **What**: Track when customer was last contacted about invoice
  - **Why**: Prevents pestering, ensures follow-up
  - **Confidence**: High - simple date field
  - **Acceptance Criteria**:
    - Date updated on contact
    - Visible in collections view
    - Filter by last contact

- [ ] **T065: Add Promise-to-Pay Notes**
  - **What**: Field to track customer payment commitments
  - **Why**: Customer says "I'll pay Friday" - log it
  - **Confidence**: High - simple notes field with date
  - **Acceptance Criteria**:
    - Promise date and amount
    - Visible in collections
    - Alert when promise date passes

### Dashboard - Medium Priority

- [ ] **T066: Add Technician Utilization Widget**
  - **What**: Show scheduled hours per tech today
  - **Why**: Helps with capacity planning
  - **Confidence**: Medium - unclear if needs historical trend
  - **Context**: "John: 6/8 hours (75%), Sarah: 8/8 hours (100%)"
  - **Acceptance Criteria**:
    - Shows utilization per tech
    - Today's view
    - Highlights underutilized/overutilized

- [ ] **T067: Add Quote Conversion Rate Metric**
  - **What**: Display quote-to-job conversion rate on dashboard
  - **Why**: Business health indicator
  - **Confidence**: High - simple math: accepted / total
  - **Acceptance Criteria**:
    - Shows percentage
    - Configurable time period
    - Trend indicator

### Reporting - Medium Priority

- [ ] **T068: Build Job-completion-to-invoice Time Report**
  - **What**: Average days from job done to invoice sent
  - **Why**: Measures billing efficiency
  - **Confidence**: High - timestamps exist
  - **Acceptance Criteria**:
    - Average days displayed
    - Trend over time
    - Filter by period

- [ ] **T069: Build Invoice-to-paid Time Report (DSO)**
  - **What**: Days Sales Outstanding - average days to collect payment
  - **Why**: Key financial metric
  - **Confidence**: High - standard calculation
  - **Acceptance Criteria**:
    - DSO calculated
    - Trend over time
    - Industry comparison

- [ ] **T070: Build Revenue by Service Type Report**
  - **What**: Revenue breakdown by job type
  - **Why**: Shows where money comes from
  - **Confidence**: High - straightforward aggregation
  - **Acceptance Criteria**:
    - Revenue per service type
    - Chart visualization
    - Filter by period

- [ ] **T071: Build Collections Rate Report**
  - **What**: Percentage of billed revenue collected
  - **Why**: Tracks bad debt
  - **Confidence**: High - simple ratio
  - **Acceptance Criteria**:
    - Collection rate percentage
    - Trend over time
    - Breakdown by age

### Communications - Medium Priority

- [ ] **T072: Create Message Template Management UI**
  - **What**: CRUD for email/SMS templates per company
  - **Why**: Customizable communications
  - **Confidence**: High - standard template management
  - **Acceptance Criteria**:
    - Create/edit/delete templates
    - Template categories
    - Preview before save

- [ ] **T073: Add Rich Variable System for Templates**
  - **What**: Variable substitution in templates
  - **Why**: Personalized messages
  - **Confidence**: High - standard pattern
  - **Context**: `{{customer_first_name}}`, `{{job_date}}`, `{{tech_name}}`
  - **Acceptance Criteria**:
    - Documented variables
    - Preview with sample data
    - Error if variable missing

### Roles - Medium Priority

- [ ] **T074: Create Accountant Role**
  - **What**: View-only access to all financial data
  - **Why**: For bookkeeper/CPA access
  - **Confidence**: High - clear permission boundaries
  - **Permissions to include**:
    - View all invoices
    - View all payments
    - View financial reports
    - Export data
  - **Permissions to exclude**:
    - Create/edit customers
    - Modify jobs
    - Change settings
  - **Acceptance Criteria**:
    - Role created
    - All financial data visible
    - No modification allowed

- [ ] **T075: Add Field-level Permission Controls**
  - **What**: Hide specific fields based on role
  - **Why**: Tech can't see "customer owes $5,000 overdue"
  - **Confidence**: Medium - more complex than page-level permissions
  - **Acceptance Criteria**:
    - Financial fields hidden from non-financial roles
    - Configurable field visibility
    - No data leakage

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

### CRM - Low Priority

- [ ] **T076: Add "Recreate Last Quote" Button**
  - **What**: One-click to clone customer's most recent quote
  - **Why**: Customer calls back months later wanting same work
  - **Confidence**: High - simple clone operation

### Scheduling - Low Priority

- [ ] **T077: Add Time-off Request UI**
  - **What**: Techs request time off, managers approve
  - **Why**: Vacation/sick day management
  - **Confidence**: Medium - could be simple block or complex approval workflow

- [ ] **T078: Add Skill-based Job Matching Suggestions**
  - **What**: Suggest techs based on job type vs tech skills
  - **Why**: Smart assignment recommendations
  - **Confidence**: Medium - filtering logic complexity

- [ ] **T079: Add Travel Time Notes Field**
  - **What**: Field to note expected travel time between jobs
  - **Why**: Helps with scheduling accuracy
  - **Confidence**: High - simple text/number field

### Quoting - Low Priority

- [ ] **T080: Create Terms & Conditions Templates**
  - **What**: Reusable legal text blocks for quotes
  - **Why**: Consistent terms across quotes
  - **Confidence**: High - simple template CRUD

- [ ] **T081: Add E-signature for Quote Acceptance**
  - **What**: Customer signs digitally to accept quote
  - **Why**: Legal acceptance record
  - **Confidence**: Medium - could be typed name or signature pad

- [ ] **T082: Add "Create Quote from Job" Button**
  - **What**: Generate quote from completed job
  - **Why**: Customer asks "how much to replace the whole unit?"
  - **Confidence**: Medium - less common workflow

### Invoicing - Low Priority

- [ ] **T083: Implement Late Fee Calculation**
  - **What**: Auto-calculate late fees based on rules
  - **Why**: Enforce payment terms
  - **Confidence**: Medium - rules vary by business/jurisdiction

### Reporting - Low Priority

- [ ] **T084: Build Quote-to-job Conversion Report**
  - **What**: Detailed conversion analysis over time
  - **Why**: Track sales effectiveness
  - **Confidence**: High - extends dashboard metric

- [ ] **T085: Build Lead-to-quote Time Report**
  - **What**: Average time from new customer to first quote
  - **Why**: Measures sales speed
  - **Confidence**: Medium - requires tracking customer creation date

- [ ] **T086: Build Quote-to-acceptance Time Report**
  - **What**: Average time for quotes to be accepted
  - **Why**: Measures quote effectiveness
  - **Confidence**: High - timestamps exist

- [ ] **T087: Build Revenue by Technician Report**
  - **What**: Revenue attributed to each tech
  - **Why**: Performance tracking
  - **Confidence**: Medium - privacy considerations

- [ ] **T088: Add Comparative Analysis (vs Prior Period)**
  - **What**: "Revenue this month vs last month: +12%"
  - **Why**: Trend visibility
  - **Confidence**: Medium - period comparison logic

### Communications - Low Priority

- [ ] **T089: Add Email/SMS Delivery Tracking**
  - **What**: Track sent/delivered/failed status
  - **Why**: Know if messages reached customers
  - **Confidence**: Medium - provider webhook integration

- [ ] **T090: Add Communication Failure Alerts**
  - **What**: Alert admin when communications fail
  - **Why**: Prevent silent failures
  - **Confidence**: Medium - alerting infrastructure needed

- [ ] **T091: Add Notification Preferences per Customer**
  - **What**: Customer opts out of certain notifications
  - **Why**: Respect preferences, compliance
  - **Confidence**: Medium - preference storage and checking

### Roles & Security - Low Priority

- [ ] **T092: Implement 2FA (TOTP)**
  - **What**: Google Authenticator compatible 2FA
  - **Why**: Security enhancement
  - **Confidence**: High - standard implementation

- [ ] **T093: Implement Comprehensive Audit Logging**
  - **What**: Track all data changes with who/what/when
  - **Why**: Compliance and debugging
  - **Confidence**: Medium - complex to implement well

- [ ] **T094: Add Google OAuth Sign-in**
  - **What**: "Sign in with Google" option
  - **Why**: Convenience
  - **Confidence**: High - standard OAuth flow

### Integration - Low Priority

- [ ] **T095: Build Outgoing Webhooks System**
  - **What**: Send events to customer-configured URLs
  - **Why**: Integration with external systems
  - **Confidence**: Medium - webhook infrastructure

- [ ] **T096: Build Event Log Table**
  - **What**: Store all significant events
  - **Why**: Debugging and integration foundation
  - **Confidence**: High - straightforward logging

- [ ] **T097: Add Webhook Retry Logic**
  - **What**: Retry failed webhook deliveries
  - **Why**: Reliable event delivery
  - **Confidence**: High - standard retry pattern

- [ ] **T098: Document Public API (OpenAPI/Swagger)**
  - **What**: API documentation for integrations
  - **Why**: Enable third-party integrations
  - **Confidence**: High - documentation generation

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

_Last updated: 2026-02-02_
