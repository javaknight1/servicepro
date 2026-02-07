# Product Analytics Guide

A comprehensive guide to product analytics, user behavior tracking, and business intelligence for ServicePro.

---

## Table of Contents

- [What is Product Analytics?](#what-is-product-analytics)
- [Product Analytics vs Application Metrics](#product-analytics-vs-application-metrics)
- [What Product Analytics Tracks](#what-product-analytics-tracks)
- [Tool Landscape](#tool-landscape)
  - [Google Analytics (GA4)](#google-analytics-ga4)
  - [PostHog](#posthog)
  - [Mixpanel](#mixpanel)
  - [Amplitude](#amplitude)
  - [Business Intelligence (Metabase)](#business-intelligence-metabase)
- [What ServicePro Needs](#what-servicepro-needs)
- [Implementation Guide: PostHog](#implementation-guide-posthog)
- [Event Taxonomy for ServicePro](#event-taxonomy-for-servicepro)
- [Privacy and Compliance](#privacy-and-compliance)
- [The Full Picture: What Goes Where](#the-full-picture-what-goes-where)
- [Key Concepts Glossary](#key-concepts-glossary)

---

## What is Product Analytics?

Product analytics answers the question: **"How are people using my product, and how can I improve their experience?"**

If application metrics are the car's dashboard gauges (engine health), product analytics is the GPS trip log — where did the driver go, how long did they stop at each place, what route did they take, and did they reach their destination?

Product analytics tracks **user behavior**: who clicked what, which features get used, where users drop off in a workflow, and how behavior changes over time.

Examples of questions product analytics answers:

- "What percentage of quotes get converted to jobs?"
- "How many days does it take the average user to send their first invoice?"
- "Which features are office managers using that technicians aren't?"
- "Where do users drop off in the job creation workflow?"
- "Has the new calendar view increased job scheduling frequency?"

---

## Product Analytics vs Application Metrics

These are fundamentally different tools that solve different problems. You need both.

|                        | Application Metrics (Prometheus)           | Product Analytics (PostHog/GA)                                      |
| ---------------------- | ------------------------------------------ | ------------------------------------------------------------------- |
| **Question**           | "Is my system healthy?"                    | "How do users behave?"                                              |
| **Audience**           | Developers, DevOps                         | Product managers, founders, marketing                               |
| **Data shape**         | Numbers over time (time-series)            | Events with properties (event stream)                               |
| **Identity**           | Doesn't care about individual users        | Tracks individual users and sessions                                |
| **Example**            | `http_requests_total = 50,000`             | "Sarah clicked 'Create Invoice' at 2:30pm from the Job Detail page" |
| **Granularity**        | Aggregated (counts, averages, percentiles) | Individual events tied to individual users                          |
| **Retention**          | 15-30 days at full resolution              | Months to years                                                     |
| **Real-time**          | Yes (seconds)                              | Usually minutes to hours                                            |
| **Alerting**           | Core feature                               | Not typically                                                       |
| **Cost of missing it** | Downtime goes undetected                   | Bad product decisions, wasted dev time                              |

### Concrete ServicePro Example

**Application metric (Prometheus):**

```
invoices_created_total{tenant="acme-hvac"} 847
```

> "847 invoices have been created by Acme HVAC's tenant. The counter went up by 12 in the last hour."

**Product analytics event (PostHog):**

```json
{
  "event": "invoice_created",
  "distinct_id": "user-sarah-456",
  "properties": {
    "invoice_total": 1250.0,
    "line_items": 3,
    "created_from": "job_detail_page",
    "time_since_job_completed_days": 2,
    "tenant": "acme-hvac",
    "user_role": "office_manager"
  }
}
```

> "Sarah (office manager at Acme HVAC) created a $1,250 invoice with 3 line items from the job detail page, 2 days after the job was completed."

The metric tells you **how many**. The analytics event tells you **who, what, where, when, and why**.

---

## What Product Analytics Tracks

### Core Capabilities

**1. Event Tracking**
Every meaningful user action is recorded as an event: page views, button clicks, form submissions, feature usage.

**2. User Identification**
Events are tied to specific users, allowing you to see an individual's journey and aggregate behavior by user properties (role, tenant, plan).

**3. Funnels**
Track conversion through multi-step workflows. Example: "Of users who viewed a quote, what % clicked Edit, then clicked Send, then saw acceptance?"

```
View Quote (100%) → Edit Quote (72%) → Send Quote (58%) → Quote Accepted (31%)
```

**4. Retention**
"Of users who signed up in January, what % are still active in February? March?"

**5. Cohort Analysis**
Group users by a shared characteristic and compare behavior. "Do users who use the calendar view schedule more jobs?"

**6. Session Replay (PostHog)**
Watch recordings of actual user sessions to see where they get confused or stuck.

**7. Feature Flags (PostHog)**
Roll out features to a percentage of users and measure impact before full release.

---

## Tool Landscape

### Google Analytics (GA4)

**What it is:** Google's free web analytics platform, primarily designed for marketing websites.

**Strengths:**

- Free for most use cases
- Excellent traffic source analysis (organic, paid, referral, social)
- Built-in e-commerce tracking
- Huge ecosystem of tutorials and integrations
- Audience demographics

**Weaknesses for SaaS apps like ServicePro:**

- Designed for anonymous website visitors, not authenticated SaaS users
- Limited custom event support (compared to dedicated product analytics)
- Data is owned by Google (privacy implications)
- Complex setup for single-page applications (SPAs)
- Reporting is marketing-oriented, not product-oriented
- Session model doesn't map well to SaaS usage patterns

**When to use it:** If ServicePro has a public marketing website (e.g., `www.servicepro.com` with pricing, features, blog), GA4 is great for understanding marketing traffic. For the app itself (`app.servicepro.com`), use a dedicated product analytics tool.

**Verdict for ServicePro:** Only needed if you build a separate marketing site. Not useful for the application itself.

### PostHog

**What it is:** An open-source, all-in-one product analytics platform. This is the recommended tool for ServicePro (as noted in TODO.md T025).

**Strengths:**

- Self-hostable (full data ownership) or cloud-hosted
- Generous free tier: 1 million events/month
- Event tracking + funnels + retention + session replay + feature flags + A/B testing
- Designed specifically for SaaS products
- Privacy-friendly (can self-host, EU hosting available)
- Auto-capture mode (tracks clicks/pageviews without code)
- Open source

**Weaknesses:**

- Self-hosting requires infrastructure (ClickHouse + Kafka + Redis + Postgres)
- Cloud version can get expensive at scale
- Newer than Mixpanel/Amplitude (some features less mature)

**Cost:**

- Cloud: Free up to 1M events/mo, then $0.00031/event
- Self-hosted: Free (you pay for infrastructure)

### Mixpanel

**What it is:** A mature SaaS product analytics platform focused on event-based tracking.

**Strengths:**

- Very mature funnel and retention analysis
- Excellent query builder for non-technical users
- Strong mobile analytics
- Free tier: 20M events/month (very generous)

**Weaknesses:**

- Not self-hostable (data lives on Mixpanel servers)
- No session replay
- No feature flags
- Paid tiers get expensive quickly

### Amplitude

**What it is:** Enterprise-grade product analytics, popular with larger companies.

**Strengths:**

- Powerful behavioral analytics and segmentation
- Good for complex product analytics at scale
- Free tier: 10M events/month

**Weaknesses:**

- Complex to set up
- Enterprise-oriented pricing
- Overkill for early-stage products

### Business Intelligence (Metabase)

**What it is:** An open-source BI tool that queries your database directly to build reports and dashboards. This is different from product analytics — it answers business questions from your existing data.

**Strengths:**

- Queries your existing PostgreSQL database directly — no separate event tracking needed
- Non-technical users can build reports with a visual query builder
- Self-hostable and free
- Great for business KPIs: revenue, customer counts, aging reports

**Weaknesses:**

- Not real-time (queries the database)
- No user behavior tracking (page views, clicks, funnels)
- Can put load on your production database (use a read replica)
- Not a replacement for product analytics

**ServicePro relevance:** TODO.md T026 recommends Metabase for business KPI dashboards. This would complement PostHog — PostHog tracks user behavior, Metabase reports on business data.

### Comparison Summary

| Tool                 | Free Tier     | Self-Host | Session Replay | Funnels | Feature Flags | Best For                    |
| -------------------- | ------------- | --------- | -------------- | ------- | ------------- | --------------------------- |
| **Google Analytics** | Unlimited     | No        | No             | Basic   | No            | Marketing websites          |
| **PostHog**          | 1M events/mo  | Yes       | Yes            | Yes     | Yes           | SaaS products (recommended) |
| **Mixpanel**         | 20M events/mo | No        | No             | Yes     | No            | Mature product analytics    |
| **Amplitude**        | 10M events/mo | No        | Yes (paid)     | Yes     | Yes (paid)    | Enterprise analytics        |
| **Metabase**         | Unlimited     | Yes       | No             | No      | No            | Business reporting from SQL |

---

## What ServicePro Needs

### Decision Tree

```
"Is my server healthy?"               → Application Metrics (Prometheus)  DONE (T005)
"Which database query is slow?"        → Application Metrics (T017)       NEXT
"Did my deploy break something?"       → Error Tracking (Sentry)          PLANNED (T024)
"What features do users actually use?" → Product Analytics (PostHog)      PLANNED (T025)
"Where does my web traffic come from?" → Google Analytics                  ONLY IF marketing site
"What's our monthly revenue trend?"    → Business Intelligence (Metabase)  PLANNED (T026)
```

### Recommended Implementation Order

1. **Application Metrics** (done) — T005 Prometheus + T006 structured logging
2. **Query Performance Monitoring** (next) — T017, builds on existing Prometheus
3. **Error Tracking** — T024 Sentry, catches errors before users report them
4. **Product Analytics** — T025 PostHog, understand how users use the app
5. **Business Intelligence** — T026 Metabase, revenue and operational reporting
6. **Google Analytics** — Only if/when you build a marketing website

### You Need Multiple Tools

This is normal and expected. Each tool has a distinct purpose:

| Layer          | Tool                 | What It Tells You                        |
| -------------- | -------------------- | ---------------------------------------- |
| Infrastructure | Prometheus + Grafana | System health, performance, alerting     |
| Application    | Sentry               | Errors, crashes, performance traces      |
| Product        | PostHog              | User behavior, feature adoption, funnels |
| Business       | Metabase             | Revenue, KPIs, operational reports       |
| Marketing      | Google Analytics     | Traffic sources, SEO, ad performance     |

---

## Implementation Guide: PostHog

When ServicePro is ready to implement product analytics (T025), here's how PostHog integration works.

### Frontend Integration (React)

```typescript
// Install
// npm install posthog-js

// Initialize (e.g., in main.tsx or a provider)
import posthog from 'posthog-js';

posthog.init('phc_your_project_key', {
  api_host: 'https://app.posthog.com', // or your self-hosted URL
  capture_pageview: true, // auto-track page views
  capture_pageleave: true, // track when users leave pages
  autocapture: true, // auto-track clicks, inputs, form submits
});

// Identify user after login
posthog.identify(user.id, {
  email: user.email,
  name: user.name,
  role: user.role, // 'office_manager', 'technician', 'dispatcher'
  tenant_id: user.tenantId,
  tenant_name: user.tenantName,
});

// Track custom events
posthog.capture('invoice_created', {
  invoice_total: 1250.0,
  line_items: 3,
  created_from: 'job_detail_page',
});

posthog.capture('quote_sent', {
  quote_total: 3500.0,
  customer_type: 'residential',
  service_type: 'hvac_repair',
});

// Reset on logout
posthog.reset();
```

### Backend Integration (Go)

```go
// For server-side events (e.g., automated emails sent, cron jobs, webhooks)
// Use PostHog's Go SDK or HTTP API

// POST https://app.posthog.com/capture
// {
//   "api_key": "phc_your_project_key",
//   "event": "payment_received",
//   "distinct_id": "user-123",
//   "properties": {
//     "amount": 1250.00,
//     "payment_method": "credit_card",
//     "tenant_id": "tenant-456"
//   }
// }
```

### React Provider Pattern

```tsx
// PostHogProvider.tsx
import { PostHogProvider as PHProvider } from 'posthog-js/react';
import posthog from 'posthog-js';

export function PostHogProvider({ children }: { children: React.ReactNode }) {
  return <PHProvider client={posthog}>{children}</PHProvider>;
}

// Using hooks in components
import { usePostHog } from 'posthog-js/react';

function CreateInvoiceButton() {
  const posthog = usePostHog();

  const handleClick = () => {
    posthog.capture('create_invoice_clicked', {
      source: 'job_detail_page',
    });
    // ... actual create logic
  };

  return <button onClick={handleClick}>Create Invoice</button>;
}
```

---

## Event Taxonomy for ServicePro

When implementing product analytics, plan your events upfront. Here's a recommended taxonomy:

### Naming Convention

Use `object_action` format in `snake_case`:

```
job_created          (not "created job" or "createJob")
quote_sent           (not "send quote")
invoice_paid         (not "payment received")
customer_searched    (not "search")
```

### Core Events

#### Authentication

| Event             | Properties                     | Why Track                  |
| ----------------- | ------------------------------ | -------------------------- |
| `user_signed_up`  | `method` (email/oauth), `role` | Measure signup conversion  |
| `user_logged_in`  | `method`                       | Measure daily active users |
| `user_logged_out` | `session_duration_minutes`     | Understand session length  |

#### Customers

| Event               | Properties                                 | Why Track               |
| ------------------- | ------------------------------------------ | ----------------------- |
| `customer_created`  | `type` (residential/commercial)            | Measure growth          |
| `customer_searched` | `query_length`, `results_count`            | Understand search usage |
| `customer_viewed`   | `customer_id`, `source` (list/search/link) | Measure engagement      |

#### Quoting

| Event                    | Properties                             | Why Track                   |
| ------------------------ | -------------------------------------- | --------------------------- |
| `quote_created`          | `total`, `line_items`, `service_type`  | Measure quoting activity    |
| `quote_sent`             | `delivery_method` (email/sms), `total` | Track the sales pipeline    |
| `quote_accepted`         | `total`, `days_since_sent`             | Measure conversion rate     |
| `quote_rejected`         | `total`, `days_since_sent`, `reason`   | Understand lost deals       |
| `quote_converted_to_job` | `quote_total`, `days_since_accepted`   | Measure workflow completion |

#### Jobs

| Event           | Properties                                     | Why Track                     |
| --------------- | ---------------------------------------------- | ----------------------------- |
| `job_created`   | `source` (quote/manual/repeat), `service_type` | Track job creation patterns   |
| `job_scheduled` | `days_until_scheduled`, `technician_id`        | Measure scheduling efficiency |
| `job_started`   | `on_time` (boolean)                            | Track punctuality             |
| `job_completed` | `duration_hours`, `on_budget`                  | Measure job execution         |

#### Invoicing

| Event             | Properties                                    | Why Track                      |
| ----------------- | --------------------------------------------- | ------------------------------ |
| `invoice_created` | `total`, `source` (job/manual), `line_items`  | Track invoicing activity       |
| `invoice_sent`    | `delivery_method`, `total`                    | Measure billing follow-through |
| `invoice_paid`    | `total`, `days_since_sent`, `payment_method`  | Measure collections speed      |
| `invoice_overdue` | `total`, `days_overdue`, `bucket` (30/60/90+) | Track A/R health               |

#### Navigation & Feature Usage

| Event               | Properties                         | Why Track                      |
| ------------------- | ---------------------------------- | ------------------------------ |
| `page_viewed`       | `page_name`, `referrer`            | Understand navigation patterns |
| `feature_used`      | `feature_name`, `context`          | Measure feature adoption       |
| `report_generated`  | `report_type`, `filters_used`      | Understand reporting needs     |
| `export_downloaded` | `format` (csv/pdf), `record_count` | Track export usage             |

---

## Privacy and Compliance

### General Principles

1. **Only track what you need.** Don't track everything "just in case."
2. **Anonymize when possible.** Use user IDs, not emails, as distinct IDs in events.
3. **Respect consent.** If required by jurisdiction, implement consent management.
4. **Document what you track.** Maintain the event taxonomy above.
5. **Provide opt-out.** Users should be able to disable analytics tracking.

### GDPR Considerations (if serving EU users)

- Use PostHog Cloud EU (eu.posthog.com) or self-host
- Implement cookie consent banner
- Support data deletion requests (`posthog.reset()` + API deletion)
- Include analytics in your privacy policy

### CCPA Considerations (if serving California users)

- Provide "Do Not Sell My Information" opt-out
- Include analytics disclosure in privacy policy
- Support data deletion requests

### Implementation

```typescript
// Respect user consent
const analyticsEnabled = getUserConsent(); // check cookie/preference

if (analyticsEnabled) {
  posthog.init('phc_key', { api_host: 'https://app.posthog.com' });
} else {
  // Initialize in disabled mode — no data sent
  posthog.init('phc_key', {
    api_host: 'https://app.posthog.com',
    opt_out_capturing_by_default: true,
  });
}
```

---

## The Full Picture: What Goes Where

Here's a complete map of which tool answers which question for ServicePro:

```
┌─────────────────────────────────────────────────────────────────┐
│                        QUESTION                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  "Is the server up?"                    → Prometheus (T005)      │
│  "Is the database slow?"                → Prometheus (T017)      │
│  "Are there errors in production?"      → Sentry (T024)          │
│  "Which features do users love?"        → PostHog (T025)         │
│  "What's our monthly revenue?"          → Metabase (T026)        │
│  "Where does website traffic come from?"→ Google Analytics       │
│                                                                  │
│  "The app feels slow" (user report)                              │
│    ├── Check Prometheus → is latency actually high?              │
│    ├── Check Sentry → are there errors causing retries?          │
│    ├── Check PostHog → which page/feature is the user on?        │
│    └── Check Metabase → is the dataset unusually large?          │
│                                                                  │
│  "Users aren't converting quotes to jobs"                        │
│    ├── Check PostHog → where do users drop off in the funnel?    │
│    ├── Check Sentry → are there errors on the conversion page?   │
│    └── Check Metabase → what's the actual conversion rate?       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Key Concepts Glossary

| Term                   | Definition                                                                                          |
| ---------------------- | --------------------------------------------------------------------------------------------------- |
| **Event**              | A discrete user action tracked with a name and properties (e.g., "invoice_created" with total=$500) |
| **Distinct ID**        | A unique identifier for a user, used to tie events to individuals                                   |
| **Funnel**             | A sequence of events representing a workflow, measured by conversion rate at each step              |
| **Retention**          | The percentage of users who return after their first visit, measured over time                      |
| **Cohort**             | A group of users who share a characteristic (e.g., "signed up in January" or "office managers")     |
| **Session**            | A continuous period of user activity, typically bounded by 30 minutes of inactivity                 |
| **MAU/DAU**            | Monthly/Daily Active Users — the count of unique users who performed any action                     |
| **Conversion rate**    | The percentage of users who complete a desired action (e.g., quote sent → quote accepted)           |
| **Feature adoption**   | The percentage of active users who have used a specific feature                                     |
| **Auto-capture**       | Automatic tracking of all clicks, page views, and form submissions without custom code              |
| **Session replay**     | A recording of a user's screen during a session, allowing you to watch their experience             |
| **Feature flag**       | A toggle that enables/disables a feature for specific users or a percentage of traffic              |
| **A/B test**           | An experiment comparing two versions of a feature to measure which performs better                  |
| **Product-market fit** | When users consistently get value from your product — measurable via retention curves               |

---

## Further Reading

- [PostHog documentation](https://posthog.com/docs)
- [PostHog event taxonomy guide](https://posthog.com/blog/event-naming-conventions)
- [Mixpanel implementation guide](https://docs.mixpanel.com/)
- [Metabase documentation](https://www.metabase.com/docs/)
- [Google Analytics 4 documentation](https://developers.google.com/analytics)
