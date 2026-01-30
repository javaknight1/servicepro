# Neon (PostgreSQL Database)

## Overview

**Neon** is a serverless PostgreSQL database service that provides a fully managed, auto-scaling PostgreSQL database with a generous free tier.

### Why We Use Neon

| Feature                | Benefit                                |
| ---------------------- | -------------------------------------- |
| **Serverless**         | Auto-suspends when idle, saving costs  |
| **Free Tier**          | 0.5GB storage, perfect for early stage |
| **PostgreSQL 16**      | Latest features, full compatibility    |
| **Branching**          | Create database branches for testing   |
| **Auto-scaling**       | Scales compute automatically           |
| **Connection Pooling** | Built-in PgBouncer                     |

### How ServicePro Uses Neon

- **Primary Database**: All application data (users, customers, jobs, invoices)
- **GORM ORM**: Go's GORM library for database operations
- **Migrations**: Schema managed via SQL migration files
- **Connection**: SSL-required connections for security

---

## Free Tier Limits

| Resource          | Limit                      |
| ----------------- | -------------------------- |
| Storage           | 0.5 GB                     |
| Compute hours     | 191.9 hours/month (shared) |
| Branches          | 10                         |
| Projects          | 1                          |
| History retention | 7 days                     |

**When to Upgrade**: Around 100 MAU or when you exceed 0.5GB of data.

---

## Setup

### Option A: Web Browser Setup (Recommended for First Time)

1. **Create Account**
   - Go to [neon.tech](https://neon.tech)
   - Click "Sign Up" (GitHub, Google, or email)
   - Verify your email

2. **Create Project**
   - Click "Create Project"
   - Project name: `servicepro`
   - PostgreSQL version: `16`
   - Region: `US East (Ohio)` (closest to Fly.io iad region)
   - Click "Create Project"

3. **Get Connection String**
   - After creation, you'll see the connection string
   - Copy the "Connection string" (looks like):
     ```
     postgresql://username:password@ep-xxx.us-east-2.aws.neon.tech/neondb?sslmode=require
     ```

4. **Create Database**
   - Go to "Databases" tab
   - Click "New Database"
   - Name: `servicepro`
   - Owner: `neondb_owner` (default)

5. **Update Connection String**
   - Change `/neondb` to `/servicepro` in your connection string:
     ```
     postgresql://username:password@ep-xxx.us-east-2.aws.neon.tech/servicepro?sslmode=require
     ```

### Option B: CLI Setup

Neon has a CLI, but it requires the web dashboard for initial account creation.

```bash
# Install Neon CLI
npm install -g neonctl

# Or with Homebrew
brew install neonctl

# Authenticate (opens browser)
neonctl auth

# List projects
neonctl projects list

# Create a new project
neonctl projects create --name servicepro --region-id aws-us-east-2

# Get connection string
neonctl connection-string servicepro

# Create a database
neonctl databases create --name servicepro --project-id <project-id>

# List databases
neonctl databases list --project-id <project-id>
```

---

## Configuration

### Environment Variables

```bash
# Full connection URL (recommended)
DATABASE_URL=postgresql://user:pass@ep-xxx.us-east-2.aws.neon.tech/servicepro?sslmode=require

# Individual settings (alternative)
DB_HOST=ep-xxx.us-east-2.aws.neon.tech
DB_PORT=5432
DB_USER=neondb_owner
DB_PASSWORD=your-password
DB_NAME=servicepro
DB_SSL_MODE=require
```

### Setting in Fly.io

```bash
# Set as a secret (encrypted)
fly secrets set DATABASE_URL="postgresql://user:pass@ep-xxx.us-east-2.aws.neon.tech/servicepro?sslmode=require" --app servicepro-api
```

### Connection Pool Settings

Neon provides built-in connection pooling. For Go applications:

```bash
# Recommended settings for serverless
DB_MAX_OPEN_CONNS=10      # Neon free tier has limited connections
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
DB_CONN_MAX_IDLE_TIME=5m
```

### Pooled vs Direct Connections

Neon provides two connection types:

| Type       | Use Case                     | Connection String                            |
| ---------- | ---------------------------- | -------------------------------------------- |
| **Pooled** | Web apps, APIs (recommended) | `ep-xxx.us-east-2.aws.neon.tech`             |
| **Direct** | Migrations, admin tasks      | `ep-xxx.us-east-2.aws.neon.tech` (port 5432) |

For migrations, use the direct connection. For the app, use pooled.

---

## Common Operations

### Run Migrations

```bash
# Connect via psql
psql "postgresql://user:pass@ep-xxx.us-east-2.aws.neon.tech/servicepro?sslmode=require"

# Run migration file
psql "postgresql://user:pass@ep-xxx.us-east-2.aws.neon.tech/servicepro?sslmode=require" < backend/migrations/001_schema.sql

# Or via Makefile
make migrate-prod
```

### Query Database

```bash
# Interactive psql session
psql $DATABASE_URL

# Run a query
psql $DATABASE_URL -c "SELECT COUNT(*) FROM users;"

# Export data
psql $DATABASE_URL -c "COPY customers TO STDOUT WITH CSV HEADER" > customers.csv
```

### Create Database Branch (Testing)

```bash
# Via CLI
neonctl branches create --name feature-branch --project-id <project-id>

# Get branch connection string
neonctl connection-string --branch feature-branch
```

### Reset Database

```bash
# Drop and recreate (DANGER: destroys all data)
psql $DATABASE_URL -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# Re-run migrations
psql $DATABASE_URL < backend/migrations/001_schema.sql
```

---

## Management

### View Usage (Web Dashboard)

1. Go to [console.neon.tech](https://console.neon.tech)
2. Select your project
3. Click "Usage" in sidebar
4. View storage, compute hours, and data transfer

### View Usage (CLI)

```bash
# Project details including usage
neonctl projects get <project-id>
```

### Scale Compute

```bash
# Via CLI (requires paid plan)
neonctl projects update <project-id> --cu 0.5

# Or via dashboard: Project Settings > Compute
```

### Enable Auto-suspend

Auto-suspend is enabled by default on free tier. The database suspends after 5 minutes of inactivity.

```bash
# Check auto-suspend status
neonctl projects get <project-id>

# Disable auto-suspend (paid plans only)
neonctl projects update <project-id> --suspend-timeout 0
```

---

## Troubleshooting

### Connection Refused

**Symptom**: `connection refused` or `timeout`

**Causes & Solutions**:

1. **Database is suspended** (auto-suspend feature)
   - First connection after idle period takes 1-3 seconds
   - Add retry logic in your app
   - Solution: The connection will wake the database automatically

2. **Wrong region**
   - Ensure Fly.io app and Neon are in nearby regions
   - US East (Fly: iad) ↔ US East (Neon: aws-us-east-2)

3. **SSL not enabled**
   - Neon requires SSL
   - Ensure `?sslmode=require` in connection string

### Too Many Connections

**Symptom**: `too many connections for role`

**Solutions**:

1. **Reduce connection pool size**

   ```bash
   DB_MAX_OPEN_CONNS=5
   ```

2. **Use pooled connection endpoint**
   - Check you're using the pooled endpoint, not direct

3. **Check for connection leaks**
   - Ensure all database connections are properly closed

### Slow Queries

**Symptom**: Queries taking longer than expected

**Debugging**:

```bash
# Check slow queries in Neon dashboard
# Go to: Project > Monitoring > Queries

# Or run EXPLAIN
psql $DATABASE_URL -c "EXPLAIN ANALYZE SELECT * FROM customers WHERE email = 'test@example.com';"
```

**Solutions**:

1. **Add indexes**

   ```sql
   CREATE INDEX idx_customers_email ON customers(email);
   ```

2. **Check if database is waking from suspend**
   - First query after idle takes longer

3. **Upgrade compute** (paid plan)
   - More CPU/memory for complex queries

### Database Suspended During Deploy

**Symptom**: Deployment fails with connection timeout

**Solution**: Add retry logic or pre-warm the database:

```bash
# Pre-warm before deploy
psql $DATABASE_URL -c "SELECT 1;"
```

### Check Connection Status

```bash
# Test connection
psql $DATABASE_URL -c "SELECT version();"

# Check active connections
psql $DATABASE_URL -c "SELECT count(*) FROM pg_stat_activity;"

# View connection details
psql $DATABASE_URL -c "SELECT usename, client_addr, state FROM pg_stat_activity;"
```

---

## Backup & Recovery

### Automatic Backups

Neon automatically handles:

- **Point-in-time recovery** (7 days on free tier)
- **Continuous backup** (no manual backups needed)

### Manual Backup

```bash
# Export entire database
pg_dump $DATABASE_URL > backup.sql

# Export specific table
pg_dump $DATABASE_URL -t customers > customers_backup.sql

# Export with compression
pg_dump $DATABASE_URL | gzip > backup.sql.gz
```

### Restore from Backup

```bash
# Restore from SQL file
psql $DATABASE_URL < backup.sql

# Restore from compressed backup
gunzip -c backup.sql.gz | psql $DATABASE_URL
```

### Point-in-Time Recovery (Dashboard)

1. Go to Neon Dashboard
2. Select project
3. Click "Branches"
4. Click "Restore" on main branch
5. Select timestamp to restore to

---

## Security

### Connection Security

- Always use `sslmode=require`
- Never commit credentials to git
- Use Fly.io secrets for production

### IP Allowlist (Paid Feature)

On paid plans, you can restrict access by IP:

1. Go to Project Settings
2. Click "IP Allow"
3. Add Fly.io IP ranges

### Rotate Credentials

```bash
# Via CLI
neonctl roles reset-password --project-id <project-id> --role-name neondb_owner

# Update Fly.io secret with new password
fly secrets set DATABASE_URL="new-connection-string" --app servicepro-api
```

---

## Useful Links

- [Neon Documentation](https://neon.tech/docs)
- [Neon CLI Reference](https://neon.tech/docs/reference/cli)
- [Connection Pooling Guide](https://neon.tech/docs/connect/connection-pooling)
- [Branching Guide](https://neon.tech/docs/introduction/branching)
- [Neon Status Page](https://neonstatus.com)
