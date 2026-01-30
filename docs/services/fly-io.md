# Fly.io (Backend Hosting)

## Overview

**Fly.io** is a platform that runs your Docker containers on lightweight VMs called "Machines" in data centers around the world. It's designed for applications that need to be close to users.

### Why We Use Fly.io

| Feature                | Benefit                              |
| ---------------------- | ------------------------------------ |
| **Free Tier**          | 3 shared VMs, 160GB outbound/month   |
| **Global Edge**        | Deploy close to users (30+ regions)  |
| **Docker Native**      | Just bring your Dockerfile           |
| **Auto-scaling**       | Scale to zero or up based on traffic |
| **Built-in SSL**       | Free automatic HTTPS                 |
| **Private Networking** | Secure inter-service communication   |

### How ServicePro Uses Fly.io

- **Backend API**: Go/Gin REST API hosting
- **Health Checks**: Built-in health monitoring
- **Auto-restart**: Automatic recovery from crashes
- **Secrets Management**: Encrypted environment variables

---

## Free Tier Limits

| Resource           | Limit                    |
| ------------------ | ------------------------ |
| Shared VMs         | 3 (shared-cpu-1x, 256MB) |
| Outbound Transfer  | 160 GB/month             |
| Persistent Volumes | None (paid feature)      |
| SSL Certificates   | Unlimited                |

**When to Upgrade**: Around 200 MAU or when you need more memory/CPU.

**Pricing**: Pay-as-you-go after free tier. ~$5/month for shared-cpu-1x with 512MB.

---

## Setup

### Option A: CLI Setup (Recommended)

#### Step 1: Install Fly CLI

```bash
# macOS (Homebrew)
brew install flyctl

# macOS/Linux (curl)
curl -L https://fly.io/install.sh | sh

# Windows (PowerShell)
powershell -Command "iwr https://fly.io/install.ps1 -useb | iex"

# Verify installation
fly version
```

#### Step 2: Create Account & Login

```bash
# Create new account (opens browser)
fly auth signup

# Or login to existing account
fly auth login

# Verify authentication
fly auth whoami
```

#### Step 3: Create Application

```bash
# Navigate to backend directory
cd backend

# Option 1: Use existing fly.toml
fly apps create servicepro-api

# Option 2: Interactive setup (creates fly.toml)
fly launch --no-deploy
# Follow prompts:
# - App name: servicepro-api
# - Region: iad (Ashburn, Virginia)
# - No Postgres (we use Neon)
# - No Redis (we use Upstash)
```

#### Step 4: Set Secrets

```bash
# Required secrets
fly secrets set DATABASE_URL="postgresql://..." --app servicepro-api
fly secrets set REDIS_URL="rediss://..." --app servicepro-api
fly secrets set JWT_SECRET="$(openssl rand -base64 32)" --app servicepro-api
fly secrets set FRONTEND_URL="https://servicepro.pages.dev" --app servicepro-api

# Optional secrets
fly secrets set RESEND_API_KEY="re_xxx" --app servicepro-api
fly secrets set STRIPE_SECRET_KEY="sk_live_xxx" --app servicepro-api

# View set secrets (values hidden)
fly secrets list --app servicepro-api
```

#### Step 5: Deploy

```bash
# Deploy from current directory
fly deploy --app servicepro-api

# Watch deployment
fly status --app servicepro-api
fly logs --app servicepro-api
```

### Option B: Web Dashboard Setup

The web dashboard is useful for monitoring but most setup requires CLI:

1. Go to [fly.io](https://fly.io) and sign up
2. After CLI deployment, view app at [fly.io/apps](https://fly.io/apps)
3. Use dashboard for:
   - Viewing metrics and logs
   - Managing team access
   - Viewing billing

---

## Configuration

### fly.toml Explained

```toml
# App name - becomes servicepro-api.fly.dev
app = "servicepro-api"

# Primary region (iad = Ashburn, Virginia)
primary_region = "iad"

# Build using Dockerfile
[build]
  dockerfile = "Dockerfile"

# Non-secret environment variables
[env]
  ENV = "production"
  PORT = "8080"
  GIN_MODE = "release"

# HTTP service configuration
[http_service]
  internal_port = 8080          # Port your app listens on
  force_https = true            # Redirect HTTP to HTTPS
  auto_stop_machines = "stop"   # Stop when idle (saves money)
  auto_start_machines = true    # Start on request
  min_machines_running = 1      # Keep at least 1 running

  [http_service.concurrency]
    type = "connections"
    hard_limit = 250            # Max connections before rejecting
    soft_limit = 200            # Start scaling at this point

# Health check configuration
[[http_service.checks]]
  interval = "30s"              # Check every 30 seconds
  timeout = "5s"                # Timeout for health check
  grace_period = "15s"          # Wait after startup before checking
  method = "GET"
  path = "/health"

# Machine (VM) configuration
[[vm]]
  cpu_kind = "shared"           # shared or performance
  cpus = 1                      # Number of CPUs
  memory = "512mb"              # Memory (256, 512, 1024, etc.)
```

### Environment Variables

These go in `[env]` section (non-secret):

```toml
[env]
  ENV = "production"
  PORT = "8080"
  GIN_MODE = "release"
  LOG_LEVEL = "info"
  LOG_FORMAT = "json"
```

### Secrets

Set via CLI (encrypted at rest):

```bash
# Database
fly secrets set DATABASE_URL="..." --app servicepro-api

# Redis
fly secrets set REDIS_URL="..." --app servicepro-api

# Auth
fly secrets set JWT_SECRET="..." --app servicepro-api

# Email
fly secrets set RESEND_API_KEY="..." --app servicepro-api

# Storage
fly secrets set AWS_ACCESS_KEY_ID="..." --app servicepro-api
fly secrets set AWS_SECRET_ACCESS_KEY="..." --app servicepro-api
fly secrets set S3_ENDPOINT="..." --app servicepro-api

# Payments
fly secrets set STRIPE_SECRET_KEY="..." --app servicepro-api
fly secrets set STRIPE_WEBHOOK_SECRET="..." --app servicepro-api
```

---

## Common Operations

### Deployment

```bash
# Deploy (builds and deploys)
fly deploy --app servicepro-api

# Deploy specific image
fly deploy --image myregistry/myapp:tag --app servicepro-api

# Deploy with build args
fly deploy --build-arg VERSION=1.0.0 --app servicepro-api
```

### Logs

```bash
# Stream logs
fly logs --app servicepro-api

# Last 100 lines
fly logs -n 100 --app servicepro-api

# Filter by instance
fly logs --instance <instance-id> --app servicepro-api
```

### Status & Monitoring

```bash
# App status
fly status --app servicepro-api

# Machine status
fly machine list --app servicepro-api

# Detailed machine info
fly machine status <machine-id> --app servicepro-api
```

### SSH Access

```bash
# SSH into running machine
fly ssh console --app servicepro-api

# Run a command
fly ssh console --app servicepro-api -C "ls -la"

# SSH with specific machine
fly ssh console --app servicepro-api --select
```

### Scaling

```bash
# Scale horizontally (more machines)
fly scale count 2 --app servicepro-api

# Scale vertically (more resources)
fly scale memory 1024 --app servicepro-api
fly scale vm shared-cpu-2x --app servicepro-api

# Add regions
fly regions add lhr --app servicepro-api  # London

# View current scale
fly scale show --app servicepro-api
```

### Restart/Redeploy

```bash
# Restart all machines
fly apps restart servicepro-api

# Redeploy
fly deploy --app servicepro-api
```

---

## Management

### View Metrics (Web Dashboard)

1. Go to [fly.io/apps/servicepro-api](https://fly.io/apps/servicepro-api)
2. Click "Metrics"
3. View:
   - Request rate
   - Response times
   - Error rate
   - CPU/Memory usage

### Certificates

```bash
# List certificates
fly certs list --app servicepro-api

# Add custom domain
fly certs add api.servicepro.com --app servicepro-api

# Check certificate status
fly certs show api.servicepro.com --app servicepro-api
```

### Custom Domains

```bash
# Add domain
fly certs add api.servicepro.com --app servicepro-api

# Configure DNS (add CNAME or A record)
# CNAME: api.servicepro.com -> servicepro-api.fly.dev
# Or A record to Fly.io IP

# Verify
fly certs check api.servicepro.com --app servicepro-api
```

### IP Addresses

```bash
# List IPs
fly ips list --app servicepro-api

# Allocate dedicated IPv4 (costs extra)
fly ips allocate-v4 --app servicepro-api

# Allocate IPv6 (free)
fly ips allocate-v6 --app servicepro-api
```

---

## Troubleshooting

### Deployment Fails

**Symptom**: `fly deploy` fails

**Debugging**:

```bash
# Check build logs
fly logs --app servicepro-api

# Try local build
docker build -t servicepro-api .
docker run -p 8080:8080 servicepro-api
```

**Common Causes**:

1. **Dockerfile error**
   - Test locally first
   - Check build stage names

2. **Health check fails**
   - Ensure `/health` endpoint works
   - Check `grace_period` is long enough

3. **Missing secrets**
   - App crashes on startup
   - Check logs for missing env vars

### Health Check Fails

**Symptom**: Machine marked unhealthy, no traffic routed

**Solutions**:

1. **Check endpoint**

   ```bash
   # Test locally
   curl http://localhost:8080/health
   ```

2. **Increase timeouts**

   ```toml
   [[http_service.checks]]
     timeout = "10s"
     grace_period = "30s"
   ```

3. **Check logs**
   ```bash
   fly logs --app servicepro-api
   ```

### Out of Memory

**Symptom**: App restarts, OOM in logs

**Solutions**:

1. **Increase memory**

   ```bash
   fly scale memory 1024 --app servicepro-api
   ```

2. **Check for memory leaks**
   - Profile your application
   - Check connection pool sizes

### Connection Timeouts

**Symptom**: Database or Redis connections fail

**Solutions**:

1. **Check secrets**

   ```bash
   fly secrets list --app servicepro-api
   ```

2. **Check connectivity**

   ```bash
   fly ssh console --app servicepro-api -C "curl -I https://api.neon.tech"
   ```

3. **Check for DNS issues**
   - Fly uses internal DNS
   - External services should work

### App Not Starting

**Symptom**: Machine starts but app doesn't respond

**Debugging**:

```bash
# Check logs
fly logs --app servicepro-api

# SSH and check process
fly ssh console --app servicepro-api
ps aux
./main  # Try running manually
```

---

## Regions

Available regions closest to your users:

| Code  | Location          | Use For           |
| ----- | ----------------- | ----------------- |
| `iad` | Ashburn, Virginia | US East (primary) |
| `lax` | Los Angeles       | US West           |
| `lhr` | London            | Europe            |
| `nrt` | Tokyo             | Asia              |
| `syd` | Sydney            | Australia         |
| `gru` | Sao Paulo         | South America     |

```bash
# Add region
fly regions add lhr --app servicepro-api

# List regions
fly regions list --app servicepro-api

# Set primary region
fly regions set iad --app servicepro-api
```

---

## Deploying Releases

### Manual Release

```bash
# Tag your commit
git tag v1.0.0
git push origin v1.0.0

# Deploy with version info
fly deploy --app servicepro-api --build-arg VERSION=v1.0.0
```

### Rollback

```bash
# List releases
fly releases --app servicepro-api

# Rollback to previous
fly deploy --image registry.fly.io/servicepro-api:deployment-xxx --app servicepro-api

# Or redeploy from git
git checkout v0.9.0
fly deploy --app servicepro-api
```

---

## Security

### Secrets Management

- Never put secrets in `fly.toml`
- Use `fly secrets set` for all sensitive values
- Secrets are encrypted at rest

### Network Security

- All traffic is HTTPS by default
- Internal services can use private networking
- No public SSH access (use `fly ssh console`)

### Access Control

```bash
# View organization members
fly orgs list

# Invite team member
fly orgs invite email@example.com --org personal
```

---

## Cost Optimization

### Auto-stop Idle Machines

```toml
[http_service]
  auto_stop_machines = "stop"   # Stop when idle
  auto_start_machines = true    # Start on request
  min_machines_running = 0      # Allow all to stop
```

### Use Shared CPUs

```toml
[[vm]]
  cpu_kind = "shared"  # Much cheaper than "performance"
```

### Right-size Memory

```bash
# Monitor actual usage
fly status --app servicepro-api

# Start small, scale up as needed
fly scale memory 256 --app servicepro-api  # Minimum
```

---

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/deploy-backend.yml
name: Deploy Backend

on:
  push:
    branches: [master]
    paths: ['backend/**']

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - run: flyctl deploy --remote-only
        working-directory: ./backend
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

### Get Deploy Token

```bash
# Create deploy token
fly tokens create deploy --app servicepro-api

# Add to GitHub Secrets as FLY_API_TOKEN
```

---

## Useful Links

- [Fly.io Documentation](https://fly.io/docs/)
- [Fly.io CLI Reference](https://fly.io/docs/flyctl/)
- [Fly.io Pricing](https://fly.io/docs/about/pricing/)
- [Fly.io Status](https://status.fly.io/)
- [Fly.io Community](https://community.fly.io/)
