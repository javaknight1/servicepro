# ServicePro Project Context

This file provides consistent context for Claude Code sessions.

## What We're Building

**ServicePro** is an all-in-one operations platform for small to medium-sized service businesses (1-50 employees). The goal is to replace 2-3 existing tools with one integrated solution.

**Target Market**: HVAC, plumbing, electrical services, maintenance companies, and other field service businesses.

**Core Value Proposition**:
- Replace multiple tools (scheduling + CRM + invoicing) with one platform
- Reduce operational complexity for service businesses
- Priced 20-30% below major competitors (Jobber, Housecall Pro)
- Simple, easy-to-use interface vs. complex enterprise systems

**Competitive Positioning**:
- **vs Jobber** ($50-100/month): We're simpler, more affordable, better automation
- **vs Housecall Pro** ($80-150/month): No expensive add-ons, all features included
- **vs FieldPulse** ($225/month): Fraction of the cost, no complex integrations needed

**Pricing Strategy**:
- Starter: $29/month (up to 2 users, 50 jobs/month)
- Professional: $59/month (up to 5 users, unlimited jobs)
- Business: $99/month (up to 15 users, advanced features)

## Current Development Stage

**6-Month MVP Timeline** (Single developer, part-time, 20 hours/week = 480 total hours)

**Phase 1: Foundation (Months 1-2) - IN PROGRESS**
- ✅ Project setup & AWS infrastructure
- ✅ Database design & core APIs
- ✅ User authentication (JWT)
- ✅ User registration with email verification
- ✅ Password reset flow
- ✅ Role-based access control (RBAC)
- 🚧 Customer management APIs (CURRENT)
- ⏸️ Basic frontend framework (React + Tailwind)

**Phase 2: Core Features (Months 3-4) - UPCOMING**
- ⏸️ Customer management interface
- ⏸️ Job management system (create, assign, track)
- ⏸️ Calendar scheduling with drag-and-drop
- ⏸️ Quote management (create, send, approve)
- ⏸️ Invoice generation and tracking

**Phase 3: Business Logic & Integrations (Months 5-6) - FUTURE**
- ⏸️ Payment processing (Stripe integration)
- ⏸️ Communication systems (AWS SES for email, SNS for SMS)
- ⏸️ Reporting & analytics dashboard
- ⏸️ Polish & launch preparation

## MVP Feature Specifications (What We're Building)

### 1. Customer Relationship Management (CRM)

**Customer Database Requirements**:
- **Required fields**: first_name, last_name, company_name, email, primary_phone
- **Optional fields**: secondary_phone, billing_address, service_address (if different)
- **Customer type**: Residential/Commercial (dropdown)
- **Status**: Active/Inactive/Prospect
- **Notes field**: Unlimited text with timestamps for each entry
- **Audit**: created_at, updated_at, deleted_at (soft delete)

**Service History Tracking**:
- Complete job history with dates, services performed, technician assigned
- Job photos and before/after documentation
- Parts used and labor hours for each service
- Customer feedback and ratings (1-5 star system)
- Payment history with invoice links

**Communication Log Requirements**:
- All inbound/outbound phone calls with timestamps
- Email history (sent/received) with read receipts
- SMS message history with delivery status
- Notes from each interaction with employee attribution
- Automatic logging of system-generated communications

### 2. Job Management & Scheduling

**Job Creation Requirements**:
- **Required**: job_title, customer_id, service_type, estimated_duration
- **Job title**: 200 character limit
- **Service type**: Predefined categories (HVAC, Plumbing, Electrical, etc.)
- **Priority**: Low/Medium/High/Emergency
- **Estimated duration**: 15-minute increments, 30 minutes to 8 hours
- **Special instructions**: 500 characters
- **Required parts/materials**: List with quantities
- **Job value estimate**: Optional dollar amount

**Calendar Scheduling System**:
- Drag-and-drop interface for job assignment
- Week view and day view options
- Color coding by job type and priority
- Technician availability blocking (vacation, sick days, training)
- Double-booking prevention with warning alerts
- Recurring job support (weekly, monthly, quarterly, annually)

**Job Status Workflow**:
- New → Scheduled → In Progress → Completed → Invoiced
- Status change timestamps with employee attribution
- Automatic customer notifications on status changes
- Photo upload capability at each stage

**Technician Assignment**:
- Skills-based matching (each technician has skill tags)
- Availability checking before assignment
- Travel time calculations between jobs
- Workload balancing across team members
- Reassignment capability with notification to all parties

### 3. Quoting & Invoicing System

**Quote Creation Requirements**:
- Line-item entry with description, quantity, unit price, total
- Parts and labor are separate line items
- Tax calculation based on customer zip code
- Discount application (dollar amount or percentage)
- Terms and conditions inclusion (editable template)
- Expiration date setting (default 30 days)
- Quote notes and special instructions section

**Quote Management Features**:
- Quote status tracking (Draft/Sent/Viewed/Accepted/Declined/Expired)
- Customer signature capture (electronic)
- Automated follow-up email scheduling
- Quote comparison for returning customers
- Clone quote functionality for similar jobs

**Invoice Generation**:
- One-click conversion from completed jobs
- Automatic population of job details and time tracking
- Payment terms selection (Net 15/30/45/60)
- Multiple payment method options
- Late fee calculation and application
- Partial payment handling

**PDF Generation Requirements**:
- Company logo and branding inclusion
- Professional formatting with consistent styling
- Digital signature fields
- Terms and conditions footer
- Itemized breakdown with tax details
- Payment instructions and methods accepted

### 4. Basic Reporting Dashboard

**Revenue Analytics**:
- Monthly Recurring Revenue (MRR) calculation and trending
- Year-over-year revenue comparison
- Revenue by service type breakdown
- Payment collection rates and aging reports
- Profit margin analysis by job type

**Operational Metrics**:
- Jobs completed per technician per day/week/month
- Average job duration vs. estimated time
- Customer satisfaction scores and trends
- Quote-to-job conversion rates
- Schedule utilization rates (billable hours vs. available hours)

**Customer Analytics**:
- New customer acquisition trends
- Customer lifetime value calculations
- Repeat customer identification and frequency
- Geographic distribution of customers
- Customer churn analysis

**Visual Requirements**:
- Chart.js implementation for all graphs
- Responsive design for mobile viewing
- Export capability to PDF and Excel
- Date range filtering (last 7/30/90 days, custom ranges)
- Real-time data updates (refresh every 5 minutes)

### 5. Automated Communications

**Email Notification System (AWS SES)**:
- Quote delivery with professional email template
- Invoice delivery with payment link inclusion
- Payment confirmation and receipt
- Appointment reminders (24 hours and 2 hours before)
- Job completion notifications with satisfaction survey
- Overdue payment reminders (automated series: 7, 14, 30 days)

**SMS Notification System (AWS SNS)**:
- Appointment confirmations within 1 hour of booking
- Technician arrival notifications ("On my way" messages)
- Emergency service availability alerts
- Payment due reminders for large invoices (>$500)

**Notification Customization**:
- Company branding in all email templates
- Customizable message content for each notification type
- Opt-out capability for customers
- Delivery tracking and failure notifications
- A/B testing capability for message effectiveness

### 6. User Management & Security

**User Role Definitions**:
- **Owner Role**: Full system access, financial reports, user management, settings
- **Admin Role**: All features except user management and financial settings
- **Technician Role**: Job management, customer interaction, time tracking only
- **View-Only Role**: Dashboard and reports access only (for accountants/partners)

**Authentication Requirements**:
- Email + password login with complexity requirements
- Two-factor authentication option (SMS or app-based)
- Password reset via email verification
- Session timeout after 8 hours of inactivity
- Login attempt limiting (5 attempts, then 30-minute lockout)

**Permission System**:
- Granular permissions for each feature area
- Customer data access controls
- Financial data restriction capabilities
- Audit trail for all user actions
- IP address restrictions for admin accounts (optional)

### 7. Core Integrations

**Required for MVP**:
- ✅ Stripe for payment processing
- ✅ AWS SES for email notifications
- ✅ AWS SNS for SMS notifications
- ⏸️ Google SSO Authentication (optional)

## What We Will NOT Build (Deliberately Excluded from MVP)

These features would extend development beyond 6 months and add significant complexity:

❌ **Mobile applications** (iOS/Android) - Web-first approach
❌ **Advanced routing/GPS optimization** - Manual scheduling only
❌ **Marketing campaigns and automation** - Basic notifications only
❌ **Inventory management** - Track parts used, but no inventory system
❌ **Advanced reporting and analytics** - Basic dashboard only
❌ **QuickBooks integration** - Manual export only
❌ **Multi-location support** - Single business location only
❌ **API for third-party integrations** - Closed platform for MVP
❌ **Real-time technician tracking** - Status updates only
❌ **Customer portal** - Admin-only interface for MVP
❌ **Multi-tenancy/workspaces** - Single business per instance

**Why excluded?**: Focus on core workflow automation that immediately reduces manual processes. These features can be added post-MVP based on customer demand.

## Tech Stack Decisions

### Backend (This Repo)
- **Go 1.21+** - Type safety, performance, easy deployment
- **Gin** - Fast, minimal HTTP framework
- **GORM** - ORM for rapid development (chosen over raw SQL)
- **PostgreSQL 15+** - Primary data store on AWS RDS
- **Redis** - Caching, rate limiting, temporary tokens (ElastiCache)
- **AWS SES** - Transactional emails (~$0.10 per 1,000 emails)
- **AWS SNS** - SMS notifications (~$0.0075 per SMS)
- **JWT** - Stateless authentication

### Frontend (Separate Repo)
- **React 18** with TypeScript for type safety
- **Vite** build tool for fast development
- **React Router** for navigation
- **TanStack Query** (React Query) for data fetching and caching
- **Tailwind CSS** for rapid, consistent UI development
- **Chart.js** for dashboard analytics

### Infrastructure (AWS)
- **ECS with Fargate** - Containerized application hosting (~$45/month)
- **Application Load Balancer** - Traffic distribution (~$23/month)
- **RDS PostgreSQL** - Database (db.t3.micro, ~$25/month)
- **ElastiCache Redis** - Caching (cache.t3.micro, ~$15/month)
- **S3** - File storage for documents, invoices (~$3/month for 100GB)
- **CloudFront CDN** - Global performance (~$10/month)
- **Route 53** - DNS management (~$1/month)
- **CloudWatch** - Monitoring, logging, metrics (~$15/month)
- **Certificate Manager** - SSL/TLS certificates (free)

**Total Infrastructure Cost**: ~$137/month + ~$20/month for SES/SNS = **$157/month operating costs**

### Analytics & Monitoring
- **AWS CloudWatch** - Application performance and uptime monitoring
- **Google Analytics** - Website traffic and user behavior (free)
- **Custom analytics system** - Built into application for business KPIs
- **Native database tracking** - Customer metrics, feature usage, revenue

### Development & Deployment
- **Docker** - Containerization
- **GitHub Actions** - CI/CD pipelines
- **Terraform** - Infrastructure as code
- **AWS ECR** - Container registry

## Code Conventions (IMPORTANT)

### Always Use GORM
❌ Don't write raw SQL queries in code
✅ Use GORM methods and queries

### Always Use UUIDs
❌ Don't use auto-increment integers for primary keys
✅ Use UUID for all entity IDs

### Always Use Soft Deletes
❌ Don't hard delete records
✅ Use `gorm.DeletedAt` for soft deletes

### Always Write Tests
❌ Don't skip unit tests for services
✅ Write comprehensive tests with testify

### Always Use Repository Pattern
```
Handler → Service → Repository → Database
```
- Handlers: Parse HTTP, call services, return HTTP
- Services: Business logic, validation, orchestration
- Repositories: Database operations only

### Error Handling Pattern
```go
// ✅ Good
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}

// ❌ Bad
if err != nil {
    panic(err)  // Never panic in production code
}
```

### API Response Pattern
```go
// ✅ Good - Return DTOs
func (h *Handler) GetUser(c *gin.Context) {
    user, err := h.service.GetUser(id)
    c.JSON(200, user.ToResponse())  // Convert to DTO
}

// ❌ Bad - Return raw models
func (h *Handler) GetUser(c *gin.Context) {
    user, err := h.service.GetUser(id)
    c.JSON(200, user)  // Exposes internal fields
}
```

## Business Goals & Success Metrics

### MVP Success Criteria (12-Month Goals)

**Conservative Targets** (Base Case Scenario):
- **Customer Acquisition**: 10-15 paying customers by month 12
- **Monthly Recurring Revenue (MRR)**: $500-1,000 by month 12
- **User Retention**: 70% monthly retention rate
- **Break-even Timeline**: Months 15-18
- **Total Year 1 Investment**: ~$6,100 (infrastructure + tools + marketing)

**Validation Criteria - Proceed if**:
- ✅ 8+ active paying customers by month 9
- ✅ Sustained 10%+ monthly growth for 3 consecutive months
- ✅ Customer retention >65% monthly
- ✅ Organic word-of-mouth referrals (20%+ of new customers)
- ✅ Customers using 4+ core features regularly

**Pivot or Discontinue if**:
- ❌ <5 paying customers by month 9
- ❌ Monthly churn rate >25% consistently
- ❌ Uptime <95% or major scalability problems
- ❌ Consistent feedback that core value proposition is insufficient

### Primary Metrics to Track

**Customer Acquisition**:
- Website visitors → Trial sign-ups: Target 2-5% conversion
- Trial → Paid: Target 10-20% conversion
- Monthly churn rate: <15% target
- Customer Lifetime Value (LTV): $400-600 target

**Revenue Metrics**:
- Average Revenue Per User (ARPU): $35-45
- Month 6 Goal: $100-200 MRR
- Month 12 Goal: $500-1,000 MRR
- Revenue growth rate: 15-25% monthly target

**Product Adoption**:
- Daily Active Users: 40-60% of total customers
- Session Duration: Average 15+ minutes
- Jobs Created Per Customer: 20+ jobs/month (active users)
- Invoice Generation: 80%+ of completed jobs generate invoices

**System Performance**:
- Application Uptime: 99.0% minimum (target 99.5%)
- Page Load Times: <3 seconds for all pages
- API Response Times: <500ms for 95% of requests
- Database Query Performance: <100ms for standard queries

### Development Roadmap

**Phase 1: MVP (Current - Month 6)**
- Core CRM, job management, scheduling, quoting, invoicing
- Basic reporting dashboard
- Stripe payments, email/SMS notifications
- Single-tenant, web-only

**Phase 2: Mobile & Advanced Features (Months 7-12)**
- Mobile apps (iOS/Android) for technicians
- Advanced scheduling with route optimization
- Marketing automation (email campaigns, review requests)
- Inventory management (parts tracking)
- Enhanced reporting and analytics

**Phase 3: Enterprise & Integrations (Year 2)**
- QuickBooks integration (two-way sync)
- Multi-location support for franchises
- Public API for third-party integrations
- Advanced CRM (lead scoring, sales pipeline)
- White-label solution for partner channel

**Long-term Vision (Years 3-5)**
- AI-powered features (predictive scheduling, demand forecasting)
- Industry-specific solutions (HVAC-specific, plumbing workflows)
- Marketplace integration (parts suppliers, contractor networks)
- International expansion (multi-currency, localization)

## Common Patterns

### Creating a New Feature
1. **Migration**: `migrations/00X_feature_name.sql`
2. **Model**: `internal/models/feature.go` with GORM tags
3. **Repository**: `internal/repository/feature_repository.go`
4. **Service**: `internal/services/feature_service.go`
5. **Handler**: `internal/api/handlers/feature_handler.go`
6. **Routes**: Update `internal/api/routes/routes.go`
7. **Tests**: `*_test.go` files for services

### Database Migration Pattern
```sql
-- Always include rollback comments
-- Migration: Description
-- Rollback: DROP TABLE IF EXISTS table_name;

CREATE TABLE IF NOT EXISTS table_name (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP  -- Soft delete
);

CREATE INDEX IF NOT EXISTS idx_name ON table_name(column);
```

### GORM Model Pattern
```go
type Entity struct {
    ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key"`
    Name      string         `json:"name" gorm:"type:varchar(255);not null"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

func (Entity) TableName() string {
    return "entities"
}
```

## Database Schema

### ✅ Implemented Tables

**users**
- id (UUID), email, password_hash
- email_verified, verification_sent_at
- failed_login_count, last_failed_login_at, locked_until
- created_at, updated_at, deleted_at

**roles**
- id (UUID), name, description
- parent_role_id, hierarchy_level
- created_at, updated_at, deleted_at
- Default roles: super_admin(100), admin(80), manager(60), user(40), guest(20)

**permissions**
- id (UUID), name, description
- resource, action (resource-based permissions like "users.create")
- created_at, updated_at, deleted_at

**role_permissions** (junction table)
- role_id, permission_id
- granted_at, granted_by

**user_roles** (junction table)
- user_id, role_id
- assigned_at, assigned_by, expires_at

### 🚧 Planned Tables (Next to Implement)

**customers**
- id (UUID), first_name, last_name, company_name
- email, primary_phone, secondary_phone
- billing_address (street, city, state, zip)
- service_address (street, city, state, zip) - if different
- customer_type (Residential/Commercial)
- status (Active/Inactive/Prospect)
- notes (text field with timestamps)
- created_at, updated_at, deleted_at

**jobs**
- id (UUID), customer_id, assigned_technician_id
- job_title (200 chars), description
- service_type (HVAC, Plumbing, Electrical, etc.)
- priority (Low/Medium/High/Emergency)
- status (New/Scheduled/InProgress/Completed/Invoiced)
- estimated_duration (minutes)
- actual_duration (minutes)
- scheduled_start, scheduled_end
- actual_start, actual_end
- special_instructions (500 chars)
- job_value_estimate (decimal)
- created_at, updated_at, deleted_at

**job_parts** (materials required for job)
- id (UUID), job_id
- part_name, quantity, unit_price
- created_at, updated_at

**quotes**
- id (UUID), customer_id, job_id (optional)
- quote_number (auto-generated)
- status (Draft/Sent/Viewed/Accepted/Declined/Expired)
- subtotal, tax_amount, discount_amount, total
- terms_and_conditions (text)
- expiration_date
- customer_signature (base64), signed_at
- notes
- created_at, updated_at, deleted_at

**quote_line_items**
- id (UUID), quote_id
- description, quantity, unit_price, total
- item_type (Part/Labor)
- created_at, updated_at

**invoices**
- id (UUID), customer_id, job_id
- invoice_number (auto-generated)
- status (Draft/Sent/Paid/Overdue/Cancelled)
- subtotal, tax_amount, discount_amount, total
- amount_paid, amount_due
- payment_terms (Net15/Net30/Net45/Net60)
- due_date, paid_date
- notes
- created_at, updated_at, deleted_at

**invoice_line_items**
- id (UUID), invoice_id
- description, quantity, unit_price, total
- item_type (Part/Labor)
- created_at, updated_at

**payments**
- id (UUID), invoice_id, customer_id
- amount, payment_method
- stripe_payment_intent_id
- status (Pending/Succeeded/Failed/Refunded)
- paid_at, refunded_at
- created_at, updated_at

**communication_logs**
- id (UUID), customer_id, user_id (employee)
- communication_type (Phone/Email/SMS/Note)
- direction (Inbound/Outbound/Internal)
- subject, body
- status (Sent/Delivered/Failed/Read)
- created_at

**technician_schedules** (availability blocking)
- id (UUID), technician_id (user_id)
- schedule_type (Vacation/Sick/Training/Blocked)
- start_date, end_date
- notes
- created_at, updated_at, deleted_at

**recurring_jobs**
- id (UUID), job_id (template job)
- frequency (Weekly/Monthly/Quarterly/Annually)
- next_occurrence_date
- end_date (optional)
- created_at, updated_at, deleted_at

**analytics_events** (for tracking user actions and KPIs)
- id (UUID), user_id
- event_type, event_data (JSON)
- created_at

## Environment Setup

### Required Services
- PostgreSQL 15+ (primary database)
- Redis 7+ (caching, tokens)
- AWS SES (email sending)

### Environment Variables
See README.md for complete list.

## Testing Philosophy

### What to Test
✅ Service layer business logic (always)
✅ Model validation methods (always)
✅ Repository queries (when complex)
✅ Handlers (when complex logic)

### What Not to Test
❌ Simple getters/setters
❌ GORM auto-generated methods
❌ Third-party libraries

### Test Pattern
```go
func TestServiceMethod_Success(t *testing.T) {
    // Setup
    mockRepo := new(MockRepository)
    service := NewService(mockRepo)

    // Expectations
    mockRepo.On("Method", args).Return(result, nil)

    // Execute
    result, err := service.DoSomething(args)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
    mockRepo.AssertExpectations(t)
}
```

## Performance Targets

- API response time: <100ms (p95)
- Database queries: <50ms (p95)
- Email sending: Async (don't block requests)
- Rate limiting: Redis-based, <5ms overhead
- Application uptime: 99.0% minimum (target 99.5%)
- Page load times: <3 seconds for all pages

## Analytics & Metrics Tracking

### How We'll Track Business Metrics

**Customer Acquisition Metrics**:
- Technology: Custom event tracking in PostgreSQL + Google Analytics
- Track: Sign-ups, trial-to-paid conversions, referral sources
- Dashboard: Real-time counters in admin panel
- Cost: $0 (built into application)

**Revenue Metrics**:
- Technology: Native database calculations + Stripe webhook integration
- Dashboard: Revenue charts using Chart.js
- Real-time updates: Updated daily via automated jobs

**User Engagement Tracking**:
- Technology: Custom middleware logging user actions
- Metrics tracked: Login frequency, feature adoption, session duration
- Storage: Dedicated `analytics_events` table in PostgreSQL

**System Performance Monitoring**:
- AWS CloudWatch: Application uptime, API response times, error rates
- Custom health checks: Database connectivity, external service status
- Alerts: Automated notifications if uptime drops below 99%

**Data Flow**:
```
User Action → Go Middleware → Analytics Event → PostgreSQL
     ↓
Business Logic → KPI Calculations → React Dashboard
     ↓
AWS CloudWatch ← System Metrics ← Application Performance
```

**Communication Tracking**:
- Email Analytics (AWS SES): Delivery rates, basic open rates, failed deliveries
- SMS Analytics (AWS SNS): Delivery confirmation, cost tracking
- Cost: ~$0.10 per 1,000 emails, ~$0.0075 per SMS

**Privacy & Data Handling**:
- Analytics data retention: 12 months
- Business metrics: Indefinite (anonymized after 24 months)
- GDPR compliance: User data deletion on request
- No PII in tracking events

**What We'll Track (MVP)**:
- Basic business KPIs (revenue, customers, churn)
- System performance and uptime
- Core feature usage (jobs created, invoices sent)
- Customer lifecycle events (sign-up, first job, first payment)

**What We Won't Track (Future)**:
- Advanced user journey mapping
- A/B testing frameworks
- Detailed session recordings
- Predictive analytics

## Security Requirements

- All passwords hashed with bcrypt (cost 12)
- JWT tokens with reasonable expiry (1 hour)
- Rate limiting on all auth endpoints
- Input validation at service layer
- SQL injection prevention (GORM handles this)
- XSS prevention (frontend handles this)
- CSRF protection (future: when using cookies)

## Development Approach & Risk Management

### Iterative Development Strategy

**Weekly Deployments**: Every Friday to staging environment
**Bi-weekly Demos**: Feature demonstrations to stakeholders
**Customer Feedback Integration**: Early beta testing with 2-3 customers
**Risk Mitigation**: Parallel development tracks to avoid blocking dependencies

### Quality Assurance Process

**Automated Testing**:
- Unit tests for all business logic functions
- Integration testing of API endpoints with realistic data
- Service layer testing with mocks (testify/mock)

**Manual Testing**:
- User acceptance testing of complete workflows
- Browser compatibility verification
- Mobile responsiveness testing

**Security Testing**:
- Regular vulnerability scans
- Dependency security updates
- Input validation testing

### Documentation Requirements

**Technical Documentation**:
- API documentation (endpoint specifications)
- Deployment guides (AWS setup, infrastructure)
- Troubleshooting guides (common issues, solutions)

**User Documentation**:
- Feature guides (how to use each feature)
- Onboarding tutorials (getting started)
- FAQ (frequently asked questions)

**Business Documentation**:
- Requirements tracking (feature specifications)
- Change management (version history)
- Decision logs (architectural decisions)

### Risk Assessment

**Technical Risks**:
- ⚠️ Single developer bottleneck → Mitigation: Comprehensive documentation, code reviews
- ⚠️ AWS cost spiral → Mitigation: Cost monitoring/alerts, reserved instances
- ⚠️ Scalability issues → Mitigation: Regular architecture reviews, load testing

**Market Risks**:
- ⚠️ Competition from established players → Mitigation: Focus on underserved SMB segment
- ⚠️ Slow customer adoption → Mitigation: Beta program, referral incentives, SEO
- ⚠️ Insufficient differentiation → Mitigation: Emphasize simplicity and affordability

**Business Risks**:
- ⚠️ Feature scope creep → Mitigation: Strict adherence to MVP scope, regular reviews
- ⚠️ Insufficient marketing budget → Mitigation: Organic growth (SEO), customer referrals
- ⚠️ Break-even timeline too long → Mitigation: Conservative projections, milestone reviews

### Critical Decision Points

**Month 6**: Evaluate early customer traction and user feedback
**Month 9**: Assess whether growth trajectory supports continued investment
**Month 12**: Make go/no-go decision on expanded platform development

## When In Doubt

1. Check existing code patterns
2. Follow Go best practices
3. Prioritize readability over cleverness
4. Write tests first if complex logic
5. Use GORM, don't write raw SQL
6. Return errors, don't panic
7. Keep handlers thin, services fat
8. Stick to MVP scope - no feature creep
9. Focus on customer value, not feature completeness
10. Validate with real customers before building
