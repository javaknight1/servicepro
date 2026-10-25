# Cloud Run + Workers setup (one-time)

Everything needed to stand up the ServicePro API on Cloud Run, the web app on
Cloudflare Workers, and give GitHub Actions keyless permission to deploy them.
Run this once; after that [`RELEASING.md`](../../RELEASING.md) is the
day-to-day process.

> **Status: done 2026-10-24 (T203/T204), live since v0.9.0.** GCP project
> `service-pro-512609`, Cloud Run `servicepro-api` in **us-east5**, Worker
> `servicepro-web` on `servicepro.sharkfins.xyz`, database **Neon Postgres 18**
> in AWS **us-east-2** (Free plan; Launch plan before the first paying tenant).
> `GCP_PROJECT_ID` is set, so `v*` tags deploy. Still open: Redis is optional
> (Upstash or none), object storage moves to Cloudflare R2, and nightly backups
> to R2 aren't activated yet (T209). Keep this doc as the rebuild recipe.

Companion docs: [`deploy.md`](./deploy.md) (deploy + rollback),
[`backups.md`](./backups.md).

> Google's CLI drifts. If a command errors on an unknown flag, check
> `gcloud <group> <command> --help` before assuming this doc is right.

---

## What we're building

```
Browser ── https://servicepro.sharkfins.xyz ──► Cloudflare Worker "servicepro-web"
                                                 │  static assets (frontend/dist)
                                                 │  /api/*, /version ──► proxied
                                                 ▼
GitHub Actions (v* tag)                    Cloud Run "servicepro-api" (runtime SA)
   │ OIDC, no stored key                         │  secrets from Secret Manager
   ▼                                             ▼
Workload Identity Federation ──► deploy SA   Neon Postgres (us-east-2)   Redis (optional)
        ├─► Artifact Registry (push image)
        └─► Cloud Run (deploy revision)
```

Two service accounts on purpose: the **deploy SA** is what CI impersonates
(push images, roll revisions); the **runtime SA** is what the container runs as
(read its secrets, nothing more).

---

## 0. Decisions (made)

| Decision       | Choice                                       | Notes                                                                                                   |
| -------------- | -------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| Postgres host  | Neon Postgres 18, AWS `us-east-2` (T202)     | Free plan today; Launch plan before the first paying tenant                                             |
| Region         | `us-east5` (Columbus)                        | Closest Cloud Run region to Neon's `aws-us-east-2` (Ohio)                                               |
| Auth           | Neon's managed Better Auth (OD-9)            | The API verifies its JWTs (`NEON_AUTH_BASE_URL`); the SPA still uses the API's own JWT cookies (T158)   |
| Redis          | Optional: Upstash (`rediss://`) or none      | Without `REDIS_URL` the API boots with an in-memory, per-instance store                                 |
| Object storage | Cloudflare R2 (S3 API)                       | Local dev uses MinIO                                                                                    |
| Email          | Resend (T159)                                | `RESEND_API_KEY` secret                                                                                 |

---

## 1. Shell variables

```bash
export PROJECT_ID="service-pro-512609"       # globally unique (the live project)
export REGION="us-east5"                     # next to Neon aws-us-east-2 — see §0
export AR_REPO="servicepro"
export SERVICE="servicepro-api"
export GITHUB_REPO="javaknight1/servicepro"  # owner/repo, exactly
```

## 2. Project + APIs

```bash
gcloud projects create "$PROJECT_ID"
gcloud config set project "$PROJECT_ID"
gcloud billing projects link "$PROJECT_ID" --billing-account=XXXXXX-XXXXXX

gcloud services enable \
  run.googleapis.com artifactregistry.googleapis.com \
  iamcredentials.googleapis.com secretmanager.googleapis.com sts.googleapis.com

export PROJECT_NUMBER="$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')"
```

Keep `PROJECT_NUMBER` — the WIF principal needs the _number_, not the id.

## 3. Artifact Registry

```bash
gcloud artifacts repositories create "$AR_REPO" \
  --repository-format=docker --location="$REGION" \
  --description="ServicePro container images"

gcloud artifacts repositories set-cleanup-policies "$AR_REPO" --location="$REGION" \
  --policy=<(cat <<'JSON'
[
  {"name": "keep-recent", "action": {"type": "Keep"}, "mostRecentVersions": {"keepCount": 20}},
  {"name": "delete-old", "action": {"type": "Delete"}, "condition": {"olderThan": "90d"}}
]
JSON
)
```

Images land at `${REGION}-docker.pkg.dev/${PROJECT_ID}/${AR_REPO}/api:<git-sha>`
(and `:vX.Y.Z`).

## 4. Service accounts

```bash
gcloud iam service-accounts create servicepro-api --display-name="ServicePro API runtime"
export RUNTIME_SA="servicepro-api@${PROJECT_ID}.iam.gserviceaccount.com"

gcloud iam service-accounts create gh-deploy --display-name="GitHub Actions deployer"
export DEPLOY_SA="gh-deploy@${PROJECT_ID}.iam.gserviceaccount.com"

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${DEPLOY_SA}" --role="roles/run.admin"
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${DEPLOY_SA}" --role="roles/artifactregistry.writer"
# deploy-cloudrun.sh lists secret versions to decide which optional secrets to mount
# (metadata only, no payload access).
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${DEPLOY_SA}" --role="roles/secretmanager.viewer"
# Deploying a service that runs AS the runtime SA requires acting as it.
gcloud iam service-accounts add-iam-policy-binding "$RUNTIME_SA" \
  --member="serviceAccount:${DEPLOY_SA}" --role="roles/iam.serviceAccountUser"
```

## 5. Secrets

Required (attached once in §7): `DATABASE_URL`, `JWT_SECRET`.
Optional (attached by `scripts/deploy-cloudrun.sh` on each release **only if
they have an enabled version**): `SENTRY_DSN`, `RESEND_API_KEY`,
`STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, `POSTHOG_API_KEY`, `REDIS_URL`,
`TEXTBELT_API_KEY`, `INTERNAL_JOBS_SECRET`, `EDGE_PROXY_SECRET`.

```bash
for s in DATABASE_URL JWT_SECRET SENTRY_DSN RESEND_API_KEY STRIPE_SECRET_KEY \
         STRIPE_WEBHOOK_SECRET POSTHOG_API_KEY REDIS_URL TEXTBELT_API_KEY \
         INTERNAL_JOBS_SECRET EDGE_PROXY_SECRET; do
  gcloud secrets create "$s" --replication-policy=automatic
  gcloud secrets add-iam-policy-binding "$s" \
    --member="serviceAccount:${RUNTIME_SA}" --role="roles/secretmanager.secretAccessor"
done

# Values from stdin so they never land in shell history:
printf '%s' 'postgresql://...' | gcloud secrets versions add DATABASE_URL --data-file=-
openssl rand -hex 32 | gcloud secrets versions add JWT_SECRET --data-file=-
```

- `DATABASE_URL` is the **runtime** connection — Neon's pooled (`-pooler`)
  host suits a scale-to-zero service. If prepared statements fail behind the
  pooler, set `DB_SIMPLE_PROTOCOL=true` (a URL on port 6543 turns it on
  automatically; Neon's pooler is on 5432). Keep `max-instances ×
  DB_MAX_OPEN_CONNS` under the database's limit.
- `MIGRATION_DATABASE_URL` (GitHub secret, §8) is a **different** value: Neon's
  direct (non-pooler) host, port 5432 — golang-migrate and `pg_dump` need a
  session.
- `INTERNAL_JOBS_SECRET` must match the GitHub secret of the same name used by
  `scheduled-jobs.yml`. `EDGE_PROXY_SECRET` must match the Worker secret (§9).

## 6. Workload Identity Federation

```bash
gcloud iam workload-identity-pools create github --location=global --display-name="GitHub Actions"

gcloud iam workload-identity-pools providers create-oidc servicepro \
  --location=global --workload-identity-pool=github \
  --display-name="ServicePro repo" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.repository_owner=assertion.repository_owner" \
  --attribute-condition="assertion.repository_owner == '${GITHUB_REPO%%/*}'"

gcloud iam service-accounts add-iam-policy-binding "$DEPLOY_SA" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/github/attribute.repository/${GITHUB_REPO}"

echo "projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/github/providers/servicepro"
```

The `--attribute-condition` is not optional — without it the provider trusts
tokens from any GitHub repository.

## 7. Create the service

Releases only swap the image (`gcloud run deploy --image` preserves env vars
and secrets), so runtime configuration is set here, once. Bootstrap with a
placeholder image:

```bash
gcloud run deploy "$SERVICE" --region="$REGION" \
  --image="us-docker.pkg.dev/cloudrun/container/hello" \
  --service-account="$RUNTIME_SA" --port=8080 \
  --cpu=1 --memory=512Mi --min-instances=0 --max-instances=4 \
  --concurrency=40 --timeout=60s --allow-unauthenticated

gcloud run services update "$SERVICE" --region="$REGION" \
  --set-env-vars="ENV=production,GIN_MODE=release,FRONTEND_URL=https://servicepro.sharkfins.xyz,BACKEND_URL=https://servicepro.sharkfins.xyz,CORS_ALLOWED_ORIGINS=https://servicepro.sharkfins.xyz,DB_MAX_OPEN_CONNS=5,RUN_MIGRATIONS_ON_STARTUP=false,RUN_BACKGROUND_WORKERS=false" \
  --set-secrets="DATABASE_URL=DATABASE_URL:latest,JWT_SECRET=JWT_SECRET:latest"
```

`--set-env-vars` splits on commas — escape a comma inside a value as `\,`.

| Setting                   | Why                                                                               |
| ------------------------- | --------------------------------------------------------------------------------- |
| `--min-instances=0`       | Scale to zero, stays in the free tier; costs a cold start                         |
| `--max-instances=4`       | Cost cap and DB-connection cap (4 × `DB_MAX_OPEN_CONNS=5`)                        |
| `--timeout=60s`           | Above the server's own write timeout                                              |
| `--allow-unauthenticated` | The API does its own JWT auth; this only disables Cloud Run's IAM check           |
| `RUN_*=false`             | Migrations run in the release `migrate` job; tickers run via `scheduled-jobs.yml` |

Startup probe: point it at the API's readiness endpoint (`/health/ready`) so a
revision that can't reach the database never takes traffic.

## 8. GitHub configuration

Repo **variables** (Settings → Secrets and variables → Actions → Variables):

| Variable                      | Value                                                                                  | Used by                        |
| ----------------------------- | -------------------------------------------------------------------------------------- | ------------------------------ |
| `GCP_PROJECT_ID`              | `service-pro-512609` — **setting this switches deploys on**                            | release-deploy                 |
| `GCP_REGION`                  | `us-east5`                                                                             | release-deploy                 |
| `GCP_WIF_PROVIDER`            | `projects/<NUMBER>/locations/global/workloadIdentityPools/github/providers/servicepro` | release-deploy                 |
| `GCP_DEPLOY_SA`               | `gh-deploy@<project>.iam.gserviceaccount.com`                                          | release-deploy                 |
| `AR_REPOSITORY`               | `servicepro` (default if unset)                                                        | release-deploy                 |
| `CLOUD_RUN_SERVICE`           | `servicepro-api` (default if unset)                                                    | release-deploy                 |
| `API_URL`                     | the service's run.app URL — **setting this switches scheduled jobs on**                | release-deploy, scheduled-jobs |
| `SITE_URL`                    | `https://servicepro.sharkfins.xyz` (default if unset)                                  | release-deploy                 |
| `VITE_STRIPE_PUBLISHABLE_KEY` | `pk_live_...` (public)                                                                 | release-deploy (web build)     |

Repo **secrets**:

| Secret                                      | Value                                                   | Used by                            |
| ------------------------------------------- | ------------------------------------------------------- | ---------------------------------- |
| `MIGRATION_DATABASE_URL`                    | Neon direct (non-pooler) URL, port **5432**             | release-deploy, migrate, db-backup |
| `INTERNAL_JOBS_SECRET`                      | same value as the Secret Manager secret                 | scheduled-jobs                     |
| `CLOUDFLARE_API_TOKEN`                      | token with Workers Scripts:Edit (+ routes for the zone) | release-deploy (web)               |
| `CLOUDFLARE_ACCOUNT_ID`                     | Cloudflare account id                                   | release-deploy, db-backup          |
| `R2_ACCESS_KEY_ID` / `R2_SECRET_ACCESS_KEY` | R2 token scoped to `servicepro-backups`                 | db-backup                          |
| `SENTRY_AUTH_TOKEN`, `SENTRY_ORG`           | optional; Sentry steps skip without them                | release-deploy                     |
| `SENTRY_DSN_WEB`, `POSTHOG_API_KEY_WEB`     | optional; browser DSN / PostHog key                     | release-deploy (web build)         |

## 9. Cloudflare Worker (`servicepro-web`)

`frontend/wrangler.jsonc` defines the Worker: static assets from `./dist` with
SPA fallback, and a small Worker in `frontend/worker/` that proxies `/api/*` and
`/version` to the Cloud Run URL. One-time:

```bash
cd frontend
npx wrangler login
# The API origin (run.app URL) goes in wrangler.jsonc vars (API_ORIGIN).
# Shared client-IP secret — same value as the Secret Manager EDGE_PROXY_SECRET:
openssl rand -hex 32 | tee >(gcloud secrets versions add EDGE_PROXY_SECRET --data-file=-) \
  | npx wrangler secret put EDGE_PROXY_SECRET
npm run build && npx wrangler deploy     # first deploy; then releases do it
```

The custom domain `servicepro.sharkfins.xyz` is attached via the Worker's
`routes` (`custom_domain: true`), so Cloudflare creates the DNS record and cert.
No Cloud Run domain mapping is needed — the browser only ever talks to the
Worker.

Stripe: point the webhook endpoint at
`https://servicepro.sharkfins.xyz/api/v1/stripe/webhooks` (the Worker passes the
raw body through untouched).

Edge enforcement: with `ENV=production` and `EDGE_PROXY_SECRET` set, the API
rejects every `/api/*` request that lacks the Worker's `X-SP-Edge-Secret`
header (403), so the public `*.run.app` URL can't be used to bypass
Cloudflare. `/health/*`, `/version` and `/internal/jobs/*` (own token) stay
reachable directly. The Stripe webhook is deliberately **not** exempt: Stripe
calls the public domain, which goes through the Worker.

Registration: deployed environments default to `REGISTRATION_MODE=invite_only`.
Create invite codes with `make invite-code` (pointing `DATABASE_URL` at the
target database) or `POST /api/v1/admin/invite-codes` as a platform super admin.

## 10. First release

```bash
make bump ARGS="--dry-run"
make bump
git add VERSION && git commit -m "release: v0.8.0"
git push origin master
git tag v0.8.0 && git push origin v0.8.0
```

Expected job order: `guard` → `migrate` → `api` → `web` → `release`. Then:

```bash
curl -s https://servicepro.sharkfins.xyz/version | jq
# {"git_sha":"<tagged commit>","app_version":"0.8.0"}
```

## Troubleshooting

| Symptom                                       | Cause                                                                                                         |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| Release run shows migrate/api/web **skipped** | `vars.GCP_PROJECT_ID` unset (it is set for production; check it wasn't removed)                               |
| `unauthorized_client` on the auth step        | `GCP_WIF_PROVIDER` uses the project **id** instead of the **number**                                          |
| Auth succeeds, deploy 403s                    | Deploy SA lacks `roles/iam.serviceAccountUser` on the runtime SA                                              |
| `cannot list versions of secret`              | Deploy SA lacks `roles/secretmanager.viewer`                                                                  |
| Revision never becomes ready                  | Missing `DATABASE_URL`/`JWT_SECRET`, unreachable DB or Redis — `gcloud run services logs read servicepro-api` |
| `/api/*` returns the SPA's `index.html`       | Worker `run_worker_first` doesn't include `/api/*`, or the Worker wasn't deployed from `frontend/`            |
