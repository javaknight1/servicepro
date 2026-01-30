# Upstash (Redis)

## Overview

**Upstash** is a serverless Redis service that provides a fully managed Redis database with pay-per-request pricing and a generous free tier.

### Why We Use Upstash

| Feature            | Benefit                                  |
| ------------------ | ---------------------------------------- |
| **Serverless**     | No idle costs, pay per command           |
| **Free Tier**      | 10K commands/day, plenty for early stage |
| **Global**         | Low latency with regional endpoints      |
| **REST API**       | Works everywhere, even edge functions    |
| **TLS by Default** | Secure connections out of the box        |
| **Durable**        | Data persisted to disk                   |

### How ServicePro Uses Upstash

- **Session Cache**: JWT token validation caching
- **Rate Limiting**: Redis-backed rate limiter
- **Temporary Data**: OTP codes, verification tokens
- **Job Queue**: Background job processing (future)

---

## Free Tier Limits

| Resource         | Limit            |
| ---------------- | ---------------- |
| Commands         | 10,000/day       |
| Storage          | 256 MB           |
| Max Request Size | 1 MB             |
| Max Connections  | 1,000 concurrent |
| Databases        | 1 per account    |

**When to Upgrade**: Around 200 MAU or when you exceed 10K commands/day consistently.

---

## Setup

### Option A: Web Browser Setup (Recommended for First Time)

1. **Create Account**
   - Go to [upstash.com](https://upstash.com)
   - Click "Sign Up" (GitHub, Google, or email)
   - Verify your email

2. **Create Database**
   - Click "Create Database"
   - Name: `servicepro-redis`
   - Type: `Regional`
   - Region: `US-East-1` (closest to Fly.io iad)
   - Enable TLS: `Yes` (recommended)
   - Enable Eviction: `No` (we want data to persist)
   - Click "Create"

3. **Get Connection Details**
   - After creation, you'll see the dashboard
   - Copy the "Redis URL" (looks like):
     ```
     rediss://default:xxx@us1-xxx.upstash.io:6379
     ```
   - Note: `rediss://` means TLS-enabled (the extra 's')

### Option B: CLI Setup (via Fly.io)

Fly.io has native Upstash integration:

```bash
# Create Upstash Redis via Fly.io
fly redis create

# Follow prompts:
# ? Choose a Redis database name: servicepro-redis
# ? Choose a primary region: iad (Ashburn, Virginia)
# ? Would you like to enable eviction? No
# ? Optionally, choose one or more replica regions: (skip for free tier)

# This automatically:
# 1. Creates Upstash account linked to Fly
# 2. Creates Redis database
# 3. Gives you the connection URL
```

### Option C: Upstash CLI

```bash
# Install Upstash CLI
npm install -g @upstash/cli

# Or with Homebrew (if available)
brew tap upstash/tap
brew install upstash

# Login
upstash auth login

# Create database
upstash redis create --name servicepro-redis --region us-east-1

# Get connection URL
upstash redis list
```

---

## Configuration

### Environment Variables

```bash
# Full Redis URL (recommended)
REDIS_URL=rediss://default:xxx@us1-xxx.upstash.io:6379

# Individual settings (alternative)
REDIS_HOST=us1-xxx.upstash.io
REDIS_PORT=6379
REDIS_PASSWORD=xxx
REDIS_DB=0

# TLS is required for Upstash
REDIS_TLS=true
```

### Setting in Fly.io

```bash
# Set as a secret (encrypted)
fly secrets set REDIS_URL="rediss://default:xxx@us1-xxx.upstash.io:6379" --app servicepro-api

# If created via fly redis create, it's automatically attached
```

### Connection Settings

```bash
# Recommended settings for Upstash
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=2
REDIS_DIAL_TIMEOUT=5s
REDIS_READ_TIMEOUT=3s
REDIS_WRITE_TIMEOUT=3s
```

---

## Common Operations

### Connect via CLI

```bash
# Using redis-cli (if installed)
redis-cli -u $REDIS_URL

# Note: May need to disable TLS verification for some redis-cli versions
redis-cli -u $REDIS_URL --tls --insecure
```

### Basic Commands

```bash
# Set a value
redis-cli -u $REDIS_URL SET mykey "myvalue"

# Get a value
redis-cli -u $REDIS_URL GET mykey

# Delete a key
redis-cli -u $REDIS_URL DEL mykey

# Check if key exists
redis-cli -u $REDIS_URL EXISTS mykey

# Set with expiration (seconds)
redis-cli -u $REDIS_URL SETEX mykey 3600 "expires in 1 hour"

# Get all keys (use sparingly)
redis-cli -u $REDIS_URL KEYS "*"
```

### Using Upstash REST API

Upstash provides a REST API that works anywhere:

```bash
# Get REST API credentials from Upstash dashboard
# UPSTASH_REDIS_REST_URL=https://us1-xxx.upstash.io
# UPSTASH_REDIS_REST_TOKEN=xxx

# Set a value
curl "$UPSTASH_REDIS_REST_URL/set/mykey/myvalue" \
  -H "Authorization: Bearer $UPSTASH_REDIS_REST_TOKEN"

# Get a value
curl "$UPSTASH_REDIS_REST_URL/get/mykey" \
  -H "Authorization: Bearer $UPSTASH_REDIS_REST_TOKEN"
```

### Clear All Data

```bash
# WARNING: Deletes everything
redis-cli -u $REDIS_URL FLUSHALL

# Or via REST API
curl "$UPSTASH_REDIS_REST_URL/flushall" \
  -H "Authorization: Bearer $UPSTASH_REDIS_REST_TOKEN"
```

---

## Management

### View Usage (Web Dashboard)

1. Go to [console.upstash.com](https://console.upstash.com)
2. Select your database
3. View:
   - **Daily Commands**: How many commands used today
   - **Storage**: Current data size
   - **Connections**: Active connections

### Check Usage via CLI

```bash
# Get database info
redis-cli -u $REDIS_URL INFO

# Memory usage
redis-cli -u $REDIS_URL INFO memory

# Number of keys
redis-cli -u $REDIS_URL DBSIZE

# Command stats
redis-cli -u $REDIS_URL INFO commandstats
```

### Monitor Real-time

```bash
# Watch commands in real-time (exit with Ctrl+C)
redis-cli -u $REDIS_URL MONITOR

# Note: MONITOR itself counts as commands, use sparingly
```

---

## Troubleshooting

### Connection Refused

**Symptom**: `connection refused` or `timeout`

**Causes & Solutions**:

1. **Wrong URL format**
   - Ensure using `rediss://` (with TLS) not `redis://`
   - Check password is correct

2. **TLS issues**
   - Upstash requires TLS
   - Ensure your client supports TLS

3. **Network issues**
   - Check if you can reach `upstash.io` from your network
   - Try the REST API as alternative

### Authentication Failed

**Symptom**: `NOAUTH Authentication required` or `invalid password`

**Solutions**:

1. **Check password**
   - Copy password directly from Upstash dashboard
   - Ensure no extra spaces or characters

2. **URL encoding**
   - If password has special characters, URL-encode them
   - Or use individual settings instead of URL

### Rate Limited (Free Tier)

**Symptom**: `ERR max daily request limit exceeded`

**Solutions**:

1. **Check usage**
   - Go to Upstash dashboard
   - View daily command count

2. **Reduce commands**
   - Add application-level caching
   - Batch operations where possible
   - Use pipelining

3. **Upgrade plan**
   - Pay-as-you-go: $0.20 per 100K commands
   - Pro: $10/month for 10M commands/month

### Slow Operations

**Symptom**: Redis commands taking longer than expected

**Debugging**:

```bash
# Check latency
redis-cli -u $REDIS_URL --latency

# Check slow log
redis-cli -u $REDIS_URL SLOWLOG GET 10
```

**Solutions**:

1. **Check region**
   - Ensure Redis and app are in same region
   - US-East-1 (Upstash) ↔ iad (Fly.io)

2. **Optimize commands**
   - Use pipelines for multiple commands
   - Avoid KEYS \* in production

### Memory Full

**Symptom**: `OOM command not allowed` or writes failing

**Solutions**:

1. **Check memory usage**

   ```bash
   redis-cli -u $REDIS_URL INFO memory
   ```

2. **Clear unused data**

   ```bash
   # Delete keys by pattern
   redis-cli -u $REDIS_URL --scan --pattern "session:*" | xargs redis-cli -u $REDIS_URL DEL
   ```

3. **Set TTL on keys**
   - Ensure temporary data has expiration
   - Rate limit counters should expire

4. **Enable eviction** (not recommended for cache)
   - Go to Upstash dashboard
   - Enable eviction policy

---

## Rate Limiting Implementation

ServicePro uses Redis for rate limiting. Here's how it works:

```go
// Key format: ratelimit:{ip}:{window}
// Example: ratelimit:192.168.1.1:1704067200

// Increment counter for current window
INCR ratelimit:192.168.1.1:1704067200

// Set expiration on first request
EXPIRE ratelimit:192.168.1.1:1704067200 60
```

### Monitor Rate Limiting

```bash
# View rate limit keys
redis-cli -u $REDIS_URL KEYS "ratelimit:*"

# Check specific IP's count
redis-cli -u $REDIS_URL GET "ratelimit:192.168.1.1:1704067200"

# Clear rate limits (if needed)
redis-cli -u $REDIS_URL --scan --pattern "ratelimit:*" | xargs redis-cli -u $REDIS_URL DEL
```

---

## Session Management

ServicePro may cache JWT token validation in Redis:

```bash
# View session keys
redis-cli -u $REDIS_URL KEYS "session:*"

# Invalidate a session
redis-cli -u $REDIS_URL DEL "session:user-uuid-here"

# Invalidate all sessions
redis-cli -u $REDIS_URL --scan --pattern "session:*" | xargs redis-cli -u $REDIS_URL DEL
```

---

## Security

### Connection Security

- Always use `rediss://` (TLS enabled)
- Never commit credentials to git
- Use Fly.io secrets for production

### Rotate Password

1. Go to Upstash Dashboard
2. Select database
3. Click "Reset Password"
4. Update Fly.io secret:
   ```bash
   fly secrets set REDIS_URL="new-url-with-new-password" --app servicepro-api
   ```

### IP Allowlist (Optional)

1. Go to Upstash Dashboard
2. Select database
3. Click "Settings"
4. Add allowed IP addresses/ranges

---

## Useful Links

- [Upstash Documentation](https://docs.upstash.com)
- [Upstash Redis REST API](https://docs.upstash.com/redis/features/restapi)
- [Redis Commands Reference](https://redis.io/commands)
- [Upstash Status Page](https://status.upstash.com)
- [Fly.io Redis Integration](https://fly.io/docs/reference/redis)
