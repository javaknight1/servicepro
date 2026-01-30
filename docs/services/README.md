# External Services

This directory contains documentation for all external services used by ServicePro.

## Production Stack Overview ($0/month for ~50 MAU)

ServicePro uses a cost-optimized production stack that's completely free for early-stage validation:

| Service                                 | Purpose             | Free Tier Limits                 | Upgrade Point              |
| --------------------------------------- | ------------------- | -------------------------------- | -------------------------- |
| [Fly.io](fly-io.md)                     | Backend hosting     | 3 shared VMs, 160GB outbound     | ~200 MAU                   |
| [Cloudflare Pages](cloudflare-pages.md) | Frontend hosting    | Unlimited requests, global CDN   | Never (generous free tier) |
| [Neon](neon.md)                         | PostgreSQL database | 0.5GB storage, auto-suspend      | ~100 MAU or 0.5GB data     |
| [Upstash](upstash.md)                   | Redis cache         | 10K commands/day                 | ~200 MAU                   |
| [Cloudflare R2](cloudflare-r2.md)       | File storage        | 10GB storage, 10M reads/mo       | ~500 MAU                   |
| [Resend](resend.md)                     | Transactional email | 3K emails/month                  | ~100 MAU                   |
| [Stripe](stripe.md)                     | Payment processing  | No monthly fee, 2.9% + $0.30/txn | N/A (pay per use)          |

## When You'll Start Paying

| MAU Range | Expected Monthly Cost | What Changes                       |
| --------- | --------------------- | ---------------------------------- |
| 0-50      | $0                    | Everything free                    |
| 50-200    | $0-5                  | Might hit Upstash limits           |
| 200-500   | $10-25                | Upgrade Postgres ($7), Redis ($10) |
| 500-1000  | $30-60                | Scale Fly.io VMs, more storage     |
| 1000+     | $100+                 | Dedicated resources, HA setup      |

## Service Documentation

Each service has detailed documentation covering:

- What the service is and why we use it
- Setup via CLI and web browser
- Configuration and environment variables
- Common operations and management
- Troubleshooting and debugging

### Current Services

| Service          | Documentation                              | Status |
| ---------------- | ------------------------------------------ | ------ |
| Fly.io           | [fly-io.md](fly-io.md)                     | Active |
| Cloudflare Pages | [cloudflare-pages.md](cloudflare-pages.md) | Active |
| Neon             | [neon.md](neon.md)                         | Active |
| Upstash          | [upstash.md](upstash.md)                   | Active |
| Cloudflare R2    | [cloudflare-r2.md](cloudflare-r2.md)       | Active |
| Resend           | [resend.md](resend.md)                     | Active |
| Stripe           | [stripe.md](stripe.md)                     | Active |

### Future Services (Coming Soon)

| Service            | Purpose                     | Priority | Notes                         |
| ------------------ | --------------------------- | -------- | ----------------------------- |
| Sentry             | Error tracking & monitoring | High     | Free tier: 5K errors/month    |
| PostHog            | Product analytics           | High     | Free tier: 1M events/month    |
| Cloudflare Workers | Edge functions              | Medium   | Free tier: 100K requests/day  |
| Twilio             | SMS notifications           | Medium   | Pay-as-you-go                 |
| OpenAI             | AI features                 | Low      | Pay-as-you-go                 |
| Algolia            | Advanced search             | Low      | Free tier: 10K searches/month |

## Quick Reference: Environment Variables

```bash
# Fly.io (Backend)
FLY_API_TOKEN=xxx                    # For CI/CD deployment

# Neon (PostgreSQL)
DATABASE_URL=postgresql://user:pass@host/db?sslmode=require

# Upstash (Redis)
REDIS_URL=rediss://default:xxx@host:6379

# Cloudflare R2 (Storage)
R2_ACCESS_KEY_ID=xxx
R2_SECRET_ACCESS_KEY=xxx
R2_BUCKET_NAME=servicepro-uploads
R2_ENDPOINT=https://xxx.r2.cloudflarestorage.com
S3_FORCE_PATH_STYLE=true

# Resend (Email)
RESEND_API_KEY=re_xxx
EMAIL_FROM=noreply@servicepro.com

# Stripe (Payments)
STRIPE_SECRET_KEY=sk_live_xxx
STRIPE_PUBLISHABLE_KEY=pk_live_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx
```

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         USERS                                    │
└─────────────────────────────────┬───────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Cloudflare (CDN + DNS)                        │
│  ┌─────────────────────┐    ┌─────────────────────────────┐    │
│  │  Cloudflare Pages   │    │      Cloudflare R2          │    │
│  │  (React Frontend)   │    │    (File Storage)           │    │
│  └─────────────────────┘    └─────────────────────────────┘    │
└─────────────────────────────────┬───────────────────────────────┘
                                  │ API Calls
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Fly.io                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Go Backend (Gin)                            │   │
│  │  • REST API                                              │   │
│  │  • Authentication (JWT)                                  │   │
│  │  • Business Logic                                        │   │
│  └─────────────────────────────────────────────────────────┘   │
└───────────┬─────────────────────────────────┬───────────────────┘
            │                                 │
            ▼                                 ▼
┌───────────────────────┐       ┌─────────────────────────────────┐
│        Neon           │       │           Upstash               │
│   (PostgreSQL)        │       │           (Redis)               │
│                       │       │                                 │
│  • User data          │       │  • Session cache                │
│  • Business entities  │       │  • Rate limiting                │
│  • Transactions       │       │  • Temporary data               │
└───────────────────────┘       └─────────────────────────────────┘

┌───────────────────────┐       ┌─────────────────────────────────┐
│       Resend          │       │           Stripe                │
│   (Email)             │       │         (Payments)              │
│                       │       │                                 │
│  • Verification       │       │  • Subscriptions                │
│  • Notifications      │       │  • Invoicing                    │
│  • Password reset     │       │  • Payment methods              │
└───────────────────────┘       └─────────────────────────────────┘
```

## Deployment Checklist

Before deploying to production, ensure:

- [ ] All services are created and configured
- [ ] Environment variables are set as Fly.io secrets
- [ ] Database migrations have been run
- [ ] DNS is configured for custom domains
- [ ] SSL certificates are active
- [ ] Health checks are passing
- [ ] Monitoring is configured
- [ ] Backup strategy is in place

See [../deployment/README.md](../deployment/README.md) for full deployment guide.
