# ServicePro Documentation

Welcome to the ServicePro documentation. This guide covers everything you need to develop, deploy, and operate the ServicePro platform.

## Quick Links

- [Quick Start Guide](./getting-started/quick-start.md) - Get up and running in 5 minutes
- [Full Setup Guide](./getting-started/full-setup.md) - Complete development environment setup
- [API Reference](./api/README.md) - REST API documentation

## Documentation Structure

### Getting Started

| Document                                                | Description                           |
| ------------------------------------------------------- | ------------------------------------- |
| [Quick Start](./getting-started/quick-start.md)         | Fastest way to run ServicePro locally |
| [Full Setup](./getting-started/full-setup.md)           | Complete setup with IDE configuration |
| [Troubleshooting](./getting-started/troubleshooting.md) | Common issues and solutions           |

### API Reference

| Document                                  | Description                           |
| ----------------------------------------- | ------------------------------------- |
| [API Overview](./api/README.md)           | API conventions and authentication    |
| [Authentication](./api/authentication.md) | Registration, login, password reset   |
| [Customers](./api/customers.md)           | Customer management endpoints         |
| [Jobs](./api/jobs.md)                     | Jobs, status, assignments, conflicts  |
| [Invoices](./api/invoices.md)             | Invoice management and billing        |
| [Payments](./api/payments.md)             | Invoice payments, Stripe, billing     |
| [Quotes](./api/quotes.md)                 | Quote status and workflows            |
| [Errors](./api/errors.md)                 | Error envelope and status codes       |

### Architecture

| Document                               | Description                           |
| -------------------------------------- | ------------------------------------- |
| [Overview](./architecture/overview.md) | System architecture and services      |
| [Database](./architecture/DATABASE.md) | Database schema and design            |
| [Security](./architecture/SECURITY.md) | Security practices and implementation |

### Features

| Document                                       | Description               |
| ---------------------------------------------- | ------------------------- |
| [Recurring Jobs](./features/recurring-jobs.md) | Not built yet — T167 plan |
| [Tax Rates](./features/tax-service.md)         | Flat per-tenant tax rates |
| [Import/Export](./features/import-export.md)   | Data import and export    |

### Observability

| Document                                            | Description                    |
| --------------------------------------------------- | ------------------------------ |
| [Analytics](./observability/analytics.md)           | Business analytics and metrics |
| [Error Tracking](./observability/error-tracking.md) | Error monitoring and alerts    |
| [Health Checks](./observability/health-checks.md)   | Service health monitoring      |
| [Performance](./observability/performance.md)       | Performance monitoring         |

### Deployment

Live since v0.9.0 (2026-10-24) at <https://servicepro.sharkfins.xyz>: Cloudflare Worker `servicepro-web` → Cloud Run `servicepro-api` (us-east5) → Neon Postgres 18 (AWS us-east-2). Hosting and the release pipeline are documented in [Deployment](./deployment/README.md), [docs/ops/](./ops/) and [RELEASING.md](/RELEASING.md).

## Related Documentation

- [End-User Help](/help/README.md) - User guides and tutorials
- [Training Materials](/training/README.md) - Onboarding and reference guides
- [Contributing Guide](/CONTRIBUTING.md) - How to contribute to ServicePro
- [Releasing](/RELEASING.md) - Versioning and the release pipeline
- [Changelog](/CHANGELOG.md) - Release history up to v0.7.0 (later: GitHub Releases)

## Service URLs (Development)

| Service     | URL                          |
| ----------- | ---------------------------- |
| Frontend    | http://localhost:3000        |
| Backend API | http://localhost:8080/api/v1 |
| API Health  | http://localhost:8080/health |
| PostgreSQL  | localhost:5432               |
| Redis       | localhost:6379               |

## Getting Help

1. Check [Troubleshooting](./getting-started/troubleshooting.md) for common issues
2. Search existing documentation
3. Review the [API Reference](./api/README.md)
4. Open a GitHub issue for bugs or questions
