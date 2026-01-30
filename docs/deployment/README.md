# Deployment Guide

This guide walks you through deploying ServicePro to production using our $0/month stack.

## Overview

| Component    | Service          | URL                      |
| ------------ | ---------------- | ------------------------ |
| Backend API  | Fly.io           | `servicepro-api.fly.dev` |
| Frontend App | Cloudflare Pages | `servicepro.pages.dev`   |
| Database     | Neon             | Managed PostgreSQL       |
| Cache        | Upstash          | Managed Redis            |
| File Storage | Cloudflare R2    | S3-compatible            |
| Email        | Resend           | Transactional email      |
| Payments     | Stripe           | Subscriptions            |

## Prerequisites

Before starting, ensure you have:

- [ ] Git repository with ServicePro code
- [ ] Node.js 20+ installed
- [ ] Go 1.24+ installed
- [ ] A credit card (for service verification, won't be charged on free tiers)

---

## Step-by-Step Setup

### Phase 1: Install CLI Tools (5 minutes)

```bash
# 1. Fly.io CLI (backend hosting)
brew install flyctl
# Verify: fly version

# 2. Wrangler CLI (Cloudflare tools)
npm install -g wrangler
# Verify: wrangler --version

# 3. Neon CLI (optional, for database management)
npm install -g neonctl
# Verify: neonctl --version

# 4. Stripe CLI (optional, for webhook testing)
brew install stripe/stripe-cli/stripe
# Verify: stripe --version
```

### Phase 2: Create Accounts & Authenticate (15 minutes)

#### 2.1 Fly.io

```bash
# Create account (opens browser)
fly auth signup

# Or login to existing
fly auth login

# Verify
fly auth whoami
```

**Web alternative**: Go to [fly.io](https://fly.io), click "Sign Up"

#### 2.2 Cloudflare

```bash
# Login (opens browser)
wrangler login

# Verify
wrangler whoami
```

**Web alternative**: Go to [cloudflare.com](https://cloudflare.com), click "Sign Up"

#### 2.3 Neon

**Web only** (required for initial setup):

1. Go to [neon.tech](https://neon.tech)
2. Sign up with GitHub/Google/email
3. Verify email

Then optionally authenticate CLI:

```bash
neonctl auth
```

#### 2.4 Stripe

**Web only**:

1. Go to [stripe.com](https://stripe.com)
2. Create account
3. Complete business verification

Then optionally authenticate CLI:

```bash
stripe login
```

#### 2.5 Resend

**Web only**:

1. Go to [resend.com](https://resend.com)
2. Sign up
3. Get API key from dashboard

---

### Phase 3: Create Database (Neon) (10 minutes)

#### Via Web (Recommended)

1. Go to [console.neon.tech](https://console.neon.tech)
2. Click "Create Project"
3. Configure:
   - Name: `servicepro`
   - PostgreSQL version: `16`
   - Region: `US East (Ohio)` (closest to Fly.io `iad`)
4. Click "Create Project"
5. Copy the connection string:
   ```
   postgresql://user:pass@ep-xxx.us-east-2.aws.neon.tech/neondb?sslmode=require
   ```
6. Go to "Databases" tab, create database named `servicepro`
7. Update connection string to use `/servicepro` instead of `/neondb`

#### Via CLI

```bash
neonctl projects create --name servicepro --region-id aws-us-east-2
neonctl databases create --name servicepro --project-id <project-id>
neonctl connection-string --project-id <project-id> --database-name servicepro
```

**Save your DATABASE_URL** - you'll need it later.

---

### Phase 4: Create Redis (Upstash) (5 minutes)

#### Via Fly.io (Easiest)

```bash
fly redis create
# Name: servicepro-redis
# Region: iad
# Eviction: No
```

This gives you the `REDIS_URL`.

#### Via Web

1. Go to [console.upstash.com](https://console.upstash.com)
2. Click "Create Database"
3. Configure:
   - Name: `servicepro-redis`
   - Type: `Regional`
   - Region: `US-East-1`
   - TLS: Enabled
4. Copy the Redis URL (starts with `rediss://`)

**Save your REDIS_URL** - you'll need it later.

---

### Phase 5: Create Storage (Cloudflare R2) (10 minutes)

#### Via Web

1. Go to [Cloudflare Dashboard](https://dash.cloudflare.com)
2. Click "R2" in sidebar
3. Click "Purchase R2" (free to start, needs payment method)
4. Click "Create bucket"
   - Name: `servicepro-uploads`
   - Location: `Automatic`
5. Click "Manage R2 API Tokens"
6. Click "Create API Token"
   - Permissions: `Object Read & Write`
   - Bucket: `servicepro-uploads`
7. **Copy immediately** (shown once):
   - Access Key ID
   - Secret Access Key
8. Note your endpoint:
   ```
   https://<account-id>.r2.cloudflarestorage.com
   ```

**Save your R2 credentials** - you'll need them later.

---

### Phase 6: Set Up Email (Resend) (5 minutes)

#### Via Web

1. Go to [resend.com](https://resend.com)
2. Copy your API key from dashboard (starts with `re_`)
3. **For production**: Add and verify your domain
   - Click "Domains"
   - Add `servicepro.com`
   - Add the DNS records shown
   - Wait for verification

**Save your RESEND_API_KEY** - you'll need it later.

---

### Phase 7: Set Up Payments (Stripe) (15 minutes)

#### Via Web

1. Go to [dashboard.stripe.com/apikeys](https://dashboard.stripe.com/apikeys)
2. Copy:
   - **Secret key** (`sk_test_...` for testing)
   - **Publishable key** (`pk_test_...` for testing)

3. Create Products:
   - Go to [dashboard.stripe.com/products](https://dashboard.stripe.com/products)
   - Create subscription products (Free, Basic, Pro)
   - Copy the Price IDs (`price_xxx`)

4. Set up Webhooks:
   - Go to [dashboard.stripe.com/webhooks](https://dashboard.stripe.com/webhooks)
   - Click "Add endpoint"
   - URL: `https://servicepro-api.fly.dev/api/v1/webhooks/stripe`
   - Select events: `customer.subscription.*`, `invoice.*`
   - Copy webhook secret (`whsec_xxx`)

**Save your Stripe credentials** - you'll need them later.

---

### Phase 8: Deploy Backend (Fly.io) (15 minutes)

#### Step 1: Create App

```bash
cd backend

# Create the app (uses existing fly.toml)
fly apps create servicepro-api
```

#### Step 2: Set All Secrets

```bash
# Required secrets
fly secrets set DATABASE_URL="postgresql://user:pass@host/servicepro?sslmode=require" --app servicepro-api
fly secrets set REDIS_URL="rediss://default:xxx@host:6379" --app servicepro-api
fly secrets set JWT_SECRET="$(openssl rand -base64 32)" --app servicepro-api
fly secrets set FRONTEND_URL="https://servicepro.pages.dev" --app servicepro-api
fly secrets set CORS_ALLOWED_ORIGINS="https://servicepro.pages.dev" --app servicepro-api

# Storage (R2)
fly secrets set AWS_ACCESS_KEY_ID="xxx" --app servicepro-api
fly secrets set AWS_SECRET_ACCESS_KEY="xxx" --app servicepro-api
fly secrets set S3_ENDPOINT="https://xxx.r2.cloudflarestorage.com" --app servicepro-api
fly secrets set S3_BUCKET_NAME="servicepro-uploads" --app servicepro-api

# Email
fly secrets set RESEND_API_KEY="re_xxx" --app servicepro-api
fly secrets set EMAIL_FROM="noreply@servicepro.com" --app servicepro-api

# Payments (optional initially)
fly secrets set STRIPE_SECRET_KEY="sk_live_xxx" --app servicepro-api
fly secrets set STRIPE_WEBHOOK_SECRET="whsec_xxx" --app servicepro-api
fly secrets set STRIPE_PRICE_BASIC_MONTHLY="price_xxx" --app servicepro-api
fly secrets set STRIPE_PRICE_PRO_MONTHLY="price_xxx" --app servicepro-api

# Verify secrets are set
fly secrets list --app servicepro-api
```

#### Step 3: Deploy

```bash
# Deploy (builds with Dockerfile)
fly deploy --app servicepro-api

# Watch the deployment
fly logs --app servicepro-api
```

#### Step 4: Run Migrations

```bash
# SSH into the machine
fly ssh console --app servicepro-api

# Or run migration directly (if you have a migrate command)
# The app should auto-migrate on startup based on your setup
```

#### Step 5: Verify

```bash
# Check status
fly status --app servicepro-api

# Test health endpoint
curl https://servicepro-api.fly.dev/health

# Should return: {"status":"healthy",...}
```

---

### Phase 9: Deploy Frontend (Cloudflare Pages) (10 minutes)

#### Via Web (Recommended)

1. Go to [Cloudflare Dashboard](https://dash.cloudflare.com)
2. Click "Workers & Pages" > "Create application" > "Pages"
3. Click "Connect to Git"
4. Select your GitHub repository
5. Configure:
   ```
   Project name: servicepro
   Production branch: master
   Build command: npm run build
   Build output directory: dist
   Root directory: frontend
   ```
6. Add Environment Variables:
   ```
   VITE_API_URL = https://servicepro-api.fly.dev
   VITE_STRIPE_PUBLISHABLE_KEY = pk_live_xxx
   ```
7. Click "Save and Deploy"

#### Via CLI

```bash
cd frontend

# Build
npm ci
npm run build

# Deploy
wrangler pages deploy dist --project-name=servicepro
```

#### Verify

Visit `https://servicepro.pages.dev` - your app should load!

---

### Phase 10: Custom Domains (Optional) (15 minutes)

#### Backend (Fly.io)

```bash
# Add custom domain
fly certs add api.servicepro.com --app servicepro-api

# Add DNS record (at your DNS provider)
# CNAME: api -> servicepro-api.fly.dev
```

#### Frontend (Cloudflare Pages)

1. Go to Pages project > Custom domains
2. Add `app.servicepro.com`
3. Add DNS record:
   ```
   CNAME: app -> servicepro.pages.dev
   ```

---

## CI/CD Setup

### Backend: GitHub Actions

Create `.github/workflows/deploy-backend.yml`:

```yaml
name: Deploy Backend

on:
  push:
    branches: [master]
    paths:
      - 'backend/**'
      - '.github/workflows/deploy-backend.yml'
  workflow_dispatch:

jobs:
  deploy:
    name: Deploy to Fly.io
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Fly
        uses: superfly/flyctl-actions/setup-flyctl@master

      - name: Deploy
        working-directory: ./backend
        run: flyctl deploy --remote-only
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

**Get Fly deploy token:**

```bash
fly tokens create deploy --app servicepro-api
# Add as FLY_API_TOKEN in GitHub Secrets
```

### Frontend: Auto-Deploy

Cloudflare Pages automatically deploys on push to `master`. No additional setup needed!

For manual GitHub Actions deployment, create `.github/workflows/deploy-frontend.yml`:

```yaml
name: Deploy Frontend

on:
  push:
    branches: [master]
    paths:
      - 'frontend/**'
      - '.github/workflows/deploy-frontend.yml'
  workflow_dispatch:

jobs:
  deploy:
    name: Deploy to Cloudflare Pages
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies
        working-directory: ./frontend
        run: npm ci

      - name: Build
        working-directory: ./frontend
        run: npm run build
        env:
          VITE_API_URL: https://servicepro-api.fly.dev
          VITE_STRIPE_PUBLISHABLE_KEY: ${{ vars.VITE_STRIPE_PUBLISHABLE_KEY }}

      - name: Deploy
        uses: cloudflare/pages-action@v1
        with:
          apiToken: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          accountId: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
          projectName: servicepro
          directory: frontend/dist
```

---

## Post-Deployment Checklist

- [ ] Backend health check passes (`/health`)
- [ ] Frontend loads and connects to API
- [ ] User registration works
- [ ] Email verification emails are sent
- [ ] Login/logout works
- [ ] Database operations work (create customer, etc.)
- [ ] File uploads work (if applicable)
- [ ] Stripe checkout works (if applicable)

---

## Troubleshooting

### Backend won't start

```bash
# Check logs
fly logs --app servicepro-api

# Common issues:
# - Missing DATABASE_URL secret
# - Database not accessible
# - Port mismatch (should be 8080)
```

### Frontend shows API errors

```bash
# Check CORS settings
fly secrets list --app servicepro-api | grep CORS

# Ensure CORS_ALLOWED_ORIGINS includes your frontend URL
fly secrets set CORS_ALLOWED_ORIGINS="https://servicepro.pages.dev" --app servicepro-api
```

### Database connection fails

```bash
# Test from Fly machine
fly ssh console --app servicepro-api -C "wget -qO- $DATABASE_URL"

# Check Neon is not suspended (auto-wake takes 1-3s)
```

For more troubleshooting, see individual service docs in [docs/services/](../services/).

---

## Cost Summary

| Service          | Free Tier             | When You Pay         |
| ---------------- | --------------------- | -------------------- |
| Fly.io           | 3 VMs, 160GB transfer | > 200 MAU            |
| Cloudflare Pages | Unlimited             | Never (for this use) |
| Neon             | 0.5GB storage         | > 0.5GB data         |
| Upstash          | 10K commands/day      | > 200 MAU            |
| Cloudflare R2    | 10GB                  | > 10GB files         |
| Resend           | 3K emails/month       | > 100 MAU            |
| Stripe           | $0 monthly            | 2.9% + $0.30/txn     |

**Total at 50 MAU: $0/month** (excluding Stripe transaction fees)
