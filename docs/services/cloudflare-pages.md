# Cloudflare Pages (Frontend Hosting)

## Overview

**Cloudflare Pages** is a JAMstack platform for deploying static sites and single-page applications with global CDN distribution at the edge.

### Why We Use Cloudflare Pages

| Feature            | Benefit                          |
| ------------------ | -------------------------------- |
| **Always Free**    | Unlimited requests, bandwidth    |
| **Global CDN**     | 300+ edge locations              |
| **Auto Deploy**    | Git integration, preview deploys |
| **Free SSL**       | Automatic HTTPS                  |
| **Custom Domains** | Unlimited domains                |
| **Preview URLs**   | Per-branch deployments           |

### How ServicePro Uses Cloudflare Pages

- **React Frontend**: SPA hosting
- **Preview Environments**: PR previews
- **CDN**: Global asset distribution
- **Custom Domain**: `app.servicepro.com`

---

## Free Tier Limits

| Resource          | Limit          |
| ----------------- | -------------- |
| Requests          | Unlimited      |
| Bandwidth         | Unlimited      |
| Builds            | 500/month      |
| Concurrent builds | 1 (20 on paid) |
| Sites             | Unlimited      |
| Custom domains    | Unlimited      |

**When to Upgrade**: When you need more concurrent builds or advanced features like Web Analytics.

---

## Setup

### Option A: Web Browser Setup (Recommended)

1. **Create Cloudflare Account**
   - Go to [cloudflare.com](https://cloudflare.com)
   - Sign up / Log in

2. **Create Pages Project**
   - Go to "Workers & Pages" in sidebar
   - Click "Create application"
   - Select "Pages"
   - Click "Connect to Git"

3. **Connect GitHub Repository**
   - Authorize Cloudflare to access GitHub
   - Select `servicepro` repository
   - Click "Begin setup"

4. **Configure Build Settings**

   ```
   Project name: servicepro
   Production branch: master
   Build command: npm run build
   Build output directory: dist
   Root directory: frontend
   ```

5. **Set Environment Variables**
   - Click "Add variable"
   - Add:
     ```
     VITE_API_URL = https://servicepro-api.fly.dev
     VITE_STRIPE_PUBLISHABLE_KEY = pk_live_xxx
     ```
   - Click "Save and Deploy"

6. **Custom Domain** (after first deploy)
   - Go to project > Custom domains
   - Click "Set up a custom domain"
   - Enter: `app.servicepro.com`
   - Add DNS records as shown

### Option B: CLI Setup (Wrangler)

```bash
# Install Wrangler
npm install -g wrangler

# Login
wrangler login

# Navigate to frontend
cd frontend

# Create project
wrangler pages project create servicepro

# Build locally
npm run build

# Deploy to Pages
wrangler pages deploy dist --project-name=servicepro
```

### Option C: Direct Upload (No Git)

For manual deployments without Git:

```bash
# Build
cd frontend
npm run build

# Deploy via Wrangler
wrangler pages deploy dist --project-name=servicepro

# Or via dashboard:
# 1. Go to Pages project
# 2. Click "Upload assets"
# 3. Drag and drop dist folder
```

---

## Configuration

### Build Settings

In Cloudflare Dashboard or `wrangler.toml`:

```toml
# wrangler.toml (in frontend directory)
name = "servicepro"
pages_build_output_dir = "dist"

[env.production]
vars = { VITE_API_URL = "https://servicepro-api.fly.dev" }

[env.preview]
vars = { VITE_API_URL = "https://servicepro-api-staging.fly.dev" }
```

### Environment Variables

Set in Cloudflare Dashboard > Pages > Settings > Environment variables:

| Variable                      | Production                       | Preview                  |
| ----------------------------- | -------------------------------- | ------------------------ |
| `VITE_API_URL`                | `https://servicepro-api.fly.dev` | `https://staging-api...` |
| `VITE_STRIPE_PUBLISHABLE_KEY` | `pk_live_xxx`                    | `pk_test_xxx`            |
| `VITE_APP_ENV`                | `production`                     | `preview`                |

### Build Commands

Common build configurations:

```bash
# React/Vite (ServicePro uses this)
Build command: npm run build
Output directory: dist

# Create React App
Build command: npm run build
Output directory: build

# Next.js (static export)
Build command: npm run build && npm run export
Output directory: out
```

---

## Common Operations

### Deploy

```bash
# Automatic: Push to master
git push origin master

# Manual: Wrangler
cd frontend
npm run build
wrangler pages deploy dist --project-name=servicepro

# Manual: Dashboard
# Upload dist folder in Pages dashboard
```

### Preview Deployments

Every PR/branch gets a preview URL:

```bash
# Push feature branch
git push origin feature/new-design

# Cloudflare auto-creates:
# https://feature-new-design.servicepro.pages.dev
```

### Rollback

```bash
# Via dashboard:
# 1. Go to Deployments
# 2. Find previous deployment
# 3. Click "..." > "Rollback to this deployment"

# Via Wrangler (redeploy old version)
wrangler pages deploy --project-name=servicepro --deployment-id=xxx
```

### View Deployments

```bash
# List recent deployments
wrangler pages deployment list --project-name=servicepro

# Via dashboard:
# Go to Pages project > Deployments
```

---

## Management

### Custom Domains

```bash
# Add domain via dashboard:
# 1. Pages project > Custom domains
# 2. Add domain
# 3. Configure DNS

# DNS configuration:
# CNAME: app -> servicepro.pages.dev
# Or if using Cloudflare DNS, it's automatic
```

### DNS Setup (Non-Cloudflare DNS)

If your domain DNS is not on Cloudflare:

```
Type: CNAME
Name: app (or @ for root)
Value: servicepro.pages.dev
TTL: Auto
```

For root domain (apex):

```
Type: A
Name: @
Value: 192.0.2.1  # Cloudflare will handle this
```

### SSL/TLS

SSL is automatic. To verify:

1. Go to Pages project
2. Click "Custom domains"
3. Check certificate status (should show "Active")

---

## Troubleshooting

### Build Fails

**Symptom**: Build fails in Cloudflare

**Debugging**:

1. Check build logs in dashboard
2. Test locally:
   ```bash
   cd frontend
   npm ci
   npm run build
   ```

**Common Causes**:

1. **Wrong Node version**
   - Set `NODE_VERSION` in environment variables
   - Example: `NODE_VERSION = 20`

2. **Missing dependencies**
   - Check `package-lock.json` is committed
   - Run `npm ci` not `npm install`

3. **Build command error**
   - Verify build command in settings
   - Check for TypeScript errors

### Environment Variables Not Working

**Symptom**: `VITE_API_URL` is undefined in app

**Solutions**:

1. **Prefix with VITE\_**
   - Vite only exposes variables starting with `VITE_`
   - `API_URL` won't work, use `VITE_API_URL`

2. **Rebuild required**
   - Environment variables are baked in at build time
   - Changing them requires new deployment

3. **Check variable scope**
   - Production vs Preview environments

### 404 on Page Refresh

**Symptom**: Direct URL access returns 404

**Cause**: SPA routes aren't real files

**Solution**: Add redirect rule

Create `public/_redirects`:

```
/*    /index.html   200
```

Or use `_headers` and routing:

```
# _routes.json in build output
{
  "version": 1,
  "include": ["/*"],
  "exclude": []
}
```

### Slow Builds

**Symptom**: Builds take > 5 minutes

**Solutions**:

1. **Cache dependencies**
   - Cloudflare caches `node_modules` automatically
   - Ensure `package-lock.json` is committed

2. **Reduce build scope**
   - Don't rebuild on non-frontend changes
   - Use build filters (paid feature)

3. **Optimize bundle**
   - Check for large dependencies
   - Use dynamic imports

### Custom Domain Not Working

**Symptom**: Domain shows error or wrong site

**Solutions**:

1. **Check DNS propagation**

   ```bash
   dig app.servicepro.com
   nslookup app.servicepro.com
   ```

2. **Verify CNAME target**
   - Should point to `servicepro.pages.dev`

3. **Check SSL certificate**
   - May take up to 24 hours to provision

4. **Clear browser cache**
   - Try incognito mode

---

## SPA Configuration

### React Router Setup

For React Router (client-side routing):

1. Create `public/_redirects`:

   ```
   /*    /index.html   200
   ```

2. Or create `public/_routes.json`:
   ```json
   {
     "version": 1,
     "include": ["/*"],
     "exclude": ["/assets/*", "/favicon.ico"]
   }
   ```

### Headers Configuration

Create `public/_headers`:

```
/*
  X-Frame-Options: DENY
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin
  Permissions-Policy: camera=(), microphone=(), geolocation=()

/assets/*
  Cache-Control: public, max-age=31536000, immutable
```

---

## Performance Optimization

### Caching Strategy

Cloudflare Pages caches automatically:

- HTML: Short cache, revalidate
- Assets (JS/CSS/images): Long cache with hash-based URLs

### Bundle Optimization

```bash
# Analyze bundle
cd frontend
npm run build -- --analyze

# Or add to package.json
"scripts": {
  "build:analyze": "vite build --mode analyze"
}
```

### Image Optimization

Use Cloudflare Images or optimize at build time:

```bash
# Install optimization tools
npm install -D vite-plugin-imagemin

# Configure in vite.config.ts
```

---

## Preview Environments

### How It Works

1. Push to any branch (not master)
2. Cloudflare builds and deploys
3. Get unique URL: `https://<branch>.servicepro.pages.dev`

### Branch-Specific Variables

Set different env vars for preview:

```
Environment: Preview
Variable: VITE_API_URL
Value: https://staging-api.servicepro.com
```

### Protect Preview Deployments

For private previews:

1. Go to Settings > Access Policy
2. Enable "Cloudflare Access"
3. Configure who can access preview URLs

---

## Integration with Backend

### API Configuration

Frontend talks to backend via environment variable:

```typescript
// src/config.ts
export const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

// src/services/api.ts
import axios from 'axios';
import { API_URL } from '../config';

export const api = axios.create({
  baseURL: API_URL,
  withCredentials: true,
});
```

### CORS Configuration

Backend must allow frontend origin:

```bash
# In Fly.io secrets
fly secrets set CORS_ALLOWED_ORIGINS="https://servicepro.pages.dev,https://app.servicepro.com" --app servicepro-api
```

---

## CI/CD Integration

### GitHub Actions (Alternative to Built-in)

```yaml
name: Deploy Frontend

on:
  push:
    branches: [master]
    paths: ['frontend/**']

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies
        working-directory: frontend
        run: npm ci

      - name: Build
        working-directory: frontend
        run: npm run build
        env:
          VITE_API_URL: ${{ vars.VITE_API_URL }}

      - name: Deploy to Cloudflare Pages
        uses: cloudflare/pages-action@v1
        with:
          apiToken: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          accountId: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
          projectName: servicepro
          directory: frontend/dist
```

### Get Cloudflare API Token

1. Go to [Cloudflare API Tokens](https://dash.cloudflare.com/profile/api-tokens)
2. Create token with "Edit Cloudflare Pages" permission
3. Add to GitHub Secrets as `CLOUDFLARE_API_TOKEN`

---

## Useful Links

- [Cloudflare Pages Documentation](https://developers.cloudflare.com/pages/)
- [Pages Build Configuration](https://developers.cloudflare.com/pages/platform/build-configuration/)
- [Pages Redirects](https://developers.cloudflare.com/pages/platform/redirects/)
- [Pages Headers](https://developers.cloudflare.com/pages/platform/headers/)
- [Wrangler CLI](https://developers.cloudflare.com/workers/wrangler/)
- [Cloudflare Status](https://www.cloudflarestatus.com/)
