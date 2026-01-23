# ServicePro Troubleshooting Guide

Common issues and solutions when developing with ServicePro.

## Quick Diagnostics

```bash
# Run diagnostic script
./scripts/diagnose.sh

# Or manually check
docker compose ps
docker compose logs --tail=50
lsof -i :3000 -i :8080 -i :5432 -i :6379
```

## Installation Issues

### Go not found or wrong version

```bash
# macOS
brew install go
# or update
brew upgrade go

# Ubuntu
sudo rm -rf /usr/local/go
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz

# Add to PATH
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin

# Verify
go version
```

### Node.js version mismatch

```bash
# Using nvm (recommended)
nvm install 20
nvm use 20
nvm alias default 20

# Or update directly (macOS)
brew upgrade node
```

### Docker permission denied

```bash
# Linux - add user to docker group
sudo usermod -aG docker $USER
newgrp docker

# Verify
docker ps
```

### Docker Desktop not running (macOS)

```bash
open -a Docker
# Wait for Docker to start
docker info
```

## Docker Issues

### Containers not starting

```bash
# Check for port conflicts
lsof -i :5432
lsof -i :6379

# Stop conflicting services
brew services stop postgresql
brew services stop redis

# Restart Docker services
docker compose down
docker compose up -d
```

### Out of disk space

```bash
# Clean up Docker resources
docker system prune -a --volumes

# Check disk usage
docker system df
```

### Container keeps restarting

```bash
# Check logs
docker compose logs postgres

# Reset volume (WARNING: deletes data)
docker compose down -v
docker compose up -d
```

## Database Issues

### Cannot connect to database

```bash
# 1. Check if PostgreSQL is running
docker compose ps postgres

# 2. Check connection settings
cat backend/.env | grep DB_

# 3. Test connection manually
docker compose exec postgres psql -U postgres -d servicepro -c "SELECT 1"

# 4. Reset database container
docker compose down postgres
docker volume rm servicepro_postgres_data
docker compose up -d postgres
```

### Migration errors

```bash
# Check migration status
cd backend && go run cmd/api/cmd/main.go migrate status

# Reset migrations (WARNING: deletes all data)
go run cmd/api/cmd/main.go migrate down --all
go run cmd/api/cmd/main.go migrate up

# Or reset entire database
make db-reset
```

### Database connection pool exhausted

```bash
# Check current connections
docker compose exec postgres psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# Adjust pool settings in backend/.env
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
```

## Backend Issues

### Go module errors

```bash
cd backend

# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download

# Tidy modules
go mod tidy

# Verify
go mod verify
```

### Port already in use

```bash
# Find process using port
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or use a different port
APP_PORT=8081 go run cmd/api/cmd/main.go
```

### Environment variables not loading

```bash
# Check .env file exists
ls -la backend/.env

# Copy from template if missing
cp backend/.env.example backend/.env

# Verify file format (no spaces around =)
cat backend/.env
```

### JWT token errors

```bash
# Ensure JWT_SECRET is set consistently
echo $JWT_SECRET

# Clear browser storage and re-login
# In browser console:
localStorage.clear()
sessionStorage.clear()
```

## Frontend Issues

### npm install fails

```bash
cd frontend

# Clear npm cache
npm cache clean --force

# Delete node_modules and lock file
rm -rf node_modules package-lock.json

# Reinstall
npm install

# If still failing
npm install --legacy-peer-deps
```

### Vite build errors

```bash
# Clear Vite cache
rm -rf frontend/node_modules/.vite

# Restart dev server
npm run dev
```

### TypeScript errors

```bash
# Check TypeScript version
npx tsc --version

# Clear TypeScript cache
rm -rf frontend/node_modules/.cache/typescript

# Restart TS server in VS Code
# Cmd/Ctrl + Shift + P > "TypeScript: Restart TS Server"
```

### API requests failing (CORS)

```bash
# 1. Check API URL in frontend/.env
VITE_API_URL=http://localhost:8080/api/v1

# 2. Verify backend is running
curl http://localhost:8080/health

# 3. Check CORS settings in backend/.env
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

### Hot reload not working

```bash
# Linux - increase file watchers
echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf
sudo sysctl -p

# Restart dev server
npm run dev
```

## Authentication Issues

### Login not working

```bash
# Verify user exists in database
docker compose exec postgres psql -U postgres -c \
  "SELECT email, email_verified FROM users WHERE email = 'admin@servicepro.local';"

# Reset password or create user
cd backend && go run cmd/api/cmd/main.go seed
```

### Token not persisting

```bash
# Check localStorage in browser
# Developer Tools > Application > Local Storage

# In browser console:
console.log(localStorage.getItem('auth_token'))

# Check token expiration
# Decode JWT at jwt.io to check exp claim
```

### Email verification not working

```bash
# Check MailHog for local development
open http://localhost:8025

# Check verification token in database
docker compose exec postgres psql -U postgres -c \
  "SELECT token, expires_at FROM email_verifications ORDER BY created_at DESC LIMIT 1;"
```

## Performance Issues

### Slow API responses

```bash
# Enable query logging
LOG_LEVEL=debug

# Check database queries
docker compose exec postgres psql -U postgres -c \
  "SELECT query, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 5;"

# Check Redis caching
docker compose exec redis redis-cli INFO stats
```

### High memory usage

```bash
# Check container stats
docker stats

# Increase container memory limits in docker-compose.yml
services:
  backend:
    deploy:
      resources:
        limits:
          memory: 1G
```

## Testing Issues

### Tests failing locally but passing in CI

```bash
# Check environment variables
env | grep -E "(DB_|TEST_)"

# Use same database for tests
TEST_DATABASE_URL=postgresql://postgres:password@localhost:5432/servicepro_test

# Run tests in isolation
go test -count=1 -p=1 ./...
```

### E2E tests timing out

```javascript
// Increase timeout in cypress.config.ts
export default defineConfig({
  defaultCommandTimeout: 10000,
  pageLoadTimeout: 60000,
});
```

## Deployment Issues

### Docker build fails

```bash
# Build with no cache
docker build --no-cache -t servicepro-backend ./backend

# Check .dockerignore
cat .dockerignore
```

### Environment variables not available in container

```bash
# Check env vars in container
docker compose exec backend env | grep DB_

# Rebuild container
docker compose up -d --build backend
```

## Getting Help

If you're still stuck:

1. **Check logs thoroughly:**

   ```bash
   docker compose logs --tail=100 | grep -i error
   ```

2. **Search existing issues on GitHub**

3. **Create a new issue with:**
   - Error message
   - Steps to reproduce
   - Environment details (OS, versions)
   - Relevant logs

## Health Check Commands

```bash
# Full system check
curl -s http://localhost:8080/health | jq .
curl -s http://localhost:8080/ready | jq .
docker compose exec postgres pg_isready
docker compose exec redis redis-cli ping

# View all logs
docker compose logs -f --tail=50
```
