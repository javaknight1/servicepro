# Production config review — 2026-11 (T212)

A read-only review of every production setting that matters, done
2026-11-04 against live v0.11.0 (`servicepro-api-v0-11-0`, deployed
2026-11-02). Nothing was changed in production. Secret values were never read:
only secret **names**, version metadata and IAM bindings.

Sources: `gcloud run services describe`, `gcloud secrets list` /
`versions list`, IAM policies, `gh variable list` / `gh secret list`,
`wrangler secret list` / `deployments list`, `curl` against the live site,
Cloud Run startup logs, and the Neon API (project `frosty-unit-21038953`), plus
one read-only `SELECT` on the catalog (roles, policies, `pg_stat_activity`).

Companion docs: [`cloud-run-setup.md`](./cloud-run-setup.md),
[`cost-guardrails.md`](./cost-guardrails.md), [`monitoring.md`](./monitoring.md),
[`backups.md`](./backups.md), [`deploy.md`](./deploy.md).

Status key: **OK** · **Missing** (should be set, isn't) · **Wrong** (set, to the
wrong thing) · **Needs Rob** (a decision, a signup or a value only Rob can
check) · **Pending** (deliberately off until a tracked task lands).

## Action list (not-OK items first)

| #   | Item                                         | Status    | Fix                                                                                                                                                                                                                                                                                                                                                                                               |
| --- | -------------------------------------------- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Nightly DB backup                            | Wrong     | `db-backup.yml` has failed every night (2026-11-01 → 03): `R2_ACCESS_KEY_ID` / `R2_SECRET_ACCESS_KEY` GitHub secrets don't exist. Create the `servicepro-backups` R2 bucket + an R2 token scoped to it, then `gh secret set R2_ACCESS_KEY_ID` / `R2_SECRET_ACCESS_KEY` ([`backups.md`](./backups.md)). Until then the only recovery is Neon's 6-hour history window.                              |
| 2   | Plain HTTP not redirected                    | Wrong     | `http://servicepro.sharkfins.xyz/` returns **200** (page and `/api/*`), no redirect. HSTS protects only after a first HTTPS visit, and the domain isn't preloaded. Cloudflare dashboard → zone `sharkfins.xyz` → SSL/TLS → Edge Certificates → **Always Use HTTPS = On** (or a Redirect Rule for this host). Auth cookies are `Secure`, so they are not leaked today.                             |
| 3   | Sentry (API) — `SENTRY_DSN`                  | Missing   | Secret exists with **no version**, so the API runs the mock error tracker: production errors go nowhere. `printf '%s' "$DSN" \| gcloud secrets versions add SENTRY_DSN --data-file=- --project=service-pro-512609`; the next release attaches it ([`monitoring.md`](./monitoring.md)).                                                                                                          |
| 4   | Sentry (web + releases + crons)              | Missing   | GitHub secrets `SENTRY_DSN_WEB`, `SENTRY_AUTH_TOKEN`, `SENTRY_ORG`, `SENTRY_CRONS_DSN` don't exist: the SPA has no error reporting, no source maps / releases are uploaded, and T252 check-ins are silent. `gh secret set` each (values from Sentry → Bitwarden).                                                                                                                                  |
| 5   | PostHog — `POSTHOG_API_KEY` / `POSTHOG_API_KEY_WEB` | Needs Rob | API secret has no version; the web build secret doesn't exist. Both apps run analytics-off. Decide if analytics is wanted before launch; if yes, add a version to `POSTHOG_API_KEY` and `gh secret set POSTHOG_API_KEY_WEB`.                                                                                                                                                                        |
| 6   | Neon Auth: open sign-up                      | Needs Rob | Production branch has `email_password.allow_sign_up = true`, email verification off, while the app is invite-only (`REGISTRATION_MODE=invite_only`). Anyone can create a Neon Auth user by calling the auth URL directly. Confirm with the T158 work that an account with no invite/membership gets nothing; otherwise turn sign-up off (Neon console → Auth → Settings) or require email verification. |
| 7   | Neon Auth: trusted domains                   | Missing   | `trusted_origins` is **empty** and `allow_localhost = true` on the production branch. Before the SPA signs in through Neon Auth (T158), add `https://servicepro.sharkfins.xyz` as a trusted domain and turn localhost off for production (keep it on the `dev` branch).                                                                                                                            |
| 8   | Neon Auth: email + Google                    | Needs Rob | Auth emails come from the shared `auth@mail.myneon.app` ("Neon Auth") sender, and Google sign-in uses Neon's shared OAuth app. For launch: point Neon Auth email at Resend SMTP (custom sender on `servicepro.sharkfins.xyz`) and either register our own Google OAuth client or turn Google off.                                                                                                |
| 9   | Row-level security backstop (T154)           | Pending   | Policies are live (18 policies, RLS on 15 tables) and `servicepro_app` exists with `NOBYPASSRLS`, but **no login role is in it**; the API connects as `neondb_owner` (`BYPASSRLS`) and logs `rls_enforce=false`. RLS currently enforces nothing. Follow [`cloud-run-setup.md` → Row-level security](./cloud-run-setup.md#row-level-security-t154-off-until-switched-on): create `servicepro_api`, repoint the runtime `DATABASE_URL` secret, set `RLS_ENFORCE=true` in the same revision. |
| 10  | Production branch not protected              | Needs Rob | `production` branch has `protected = false`. Protected branches need a paid plan; revisit with the Launch-plan upgrade (D14).                                                                                                                                                                                                                                                                      |
| 11  | Default compute SA has `roles/editor`        | Wrong     | `736486908135-compute@developer.gserviceaccount.com` holds project **Editor**. Nothing of ours runs as it (the API uses `servicepro-api@`), so it's an unused, over-privileged identity. `gcloud projects remove-iam-policy-binding service-pro-512609 --member=serviceAccount:736486908135-compute@developer.gserviceaccount.com --role=roles/editor` (or disable the SA).                    |
| 12  | Redis / shared rate limits                   | Needs Rob | No `REDIS_URL` version: rate limits and login lockouts are in-memory **per instance** (up to 4 instances → limits are effectively up to 4× and reset on scale-to-zero). Acceptable for the invite-only beta; provision Upstash before the paid launch if login brute-force limits must be global.                                                                                                |
| 13  | `DATABASE_URL` uses the pooler               | Needs Rob | Can't verify without reading the secret. Evidence suggests yes (`pg_stat_activity` showed `neondb_owner` arriving via `pgbouncer`), but that row could have been the review's own session. Check in Bitwarden / Neon console that the host is `ep-dawn-art-b53wsdjg-pooler…`. No `DB_SIMPLE_PROTOCOL` needed: Neon's pooler supports protocol-level prepared statements.                     |
| 14  | Storage (R2, T160)                           | Pending   | No `S3_COMPATIBLE_*` secrets; startup logs "no object storage configured in production". Uploads/attachments are off and PDFs are generated on demand (T256). Fix is T160.                                                                                                                                                                                                                        |
| 15  | Stripe (T169 / T261)                         | Pending   | `STRIPE_SECRET_KEY` / `STRIPE_WEBHOOK_SECRET` secrets exist with **no version**; `STRIPE_CONNECT_WEBHOOK_SECRET` / `STRIPE_V2_WEBHOOK_SECRET` secrets don't exist; `vars.VITE_STRIPE_PUBLISHABLE_KEY` doesn't exist. Logs confirm "Stripe not configured, skipping Stripe/billing routes". Test-vs-live mode can't be checked until keys exist. When added: test keys (`sk_test_`) until launch, then live keys + all three webhook secrets together (config refuses a key without `STRIPE_WEBHOOK_SECRET`). Create the two missing secrets first (and grant `servicepro-api@` `secretAccessor` on them), or `deploy-cloudrun.sh` will just skip them. |
| 16  | `RESEND_WEBHOOK_SECRET` (T159)               | Missing   | No such secret in Secret Manager, so `/webhooks/resend` is disabled and bounces/complaints are not suppressed. Create the Resend webhook (email.bounced, email.complained → `https://servicepro.sharkfins.xyz/api/v1/webhooks/resend`), then `gcloud secrets create RESEND_WEBHOOK_SECRET`, add a version with its `whsec_…` value, and grant `servicepro-api@` `roles/secretmanager.secretAccessor` on it. `deploy-cloudrun.sh` already attaches it once it has a version. |
| 17  | `invoice-overdue` scheduled job              | OK (transient) | `scheduled-jobs.yml` calls `/internal/jobs/invoice-overdue`, which v0.11.0 doesn't have yet (HTTP 404, 4 of the last 30 runs failed). Clears with the next release; no config change.                                                                                                                                                                                                            |

## Full checklist

### Cloud Run `servicepro-api` (us-east5) — plain env vars

| Setting                            | Expected / rule                                         | Production has                                            | Status |
| ---------------------------------- | ------------------------------------------------------- | --------------------------------------------------------- | ------ |
| `ENV`                              | `production`                                            | `production`                                              | OK     |
| `GIN_MODE`                         | `release` (code also forces it when ENV≠development)    | `release`                                                 | OK     |
| `FRONTEND_URL`                     | `https://servicepro.sharkfins.xyz`                      | same                                                      | OK     |
| `BACKEND_URL`                      | public origin (links in emails)                         | `https://servicepro.sharkfins.xyz`                        | OK     |
| `CORS_ALLOWED_ORIGINS`             | production origin only, never `*`                       | `https://servicepro.sharkfins.xyz`                        | OK     |
| `REGISTRATION_MODE`                | `invite_only`                                           | `invite_only`                                             | OK     |
| `RUN_MIGRATIONS_ON_STARTUP`        | `false`                                                 | `false` (pinned by `deploy-cloudrun.sh`)                  | OK     |
| `RUN_BACKGROUND_WORKERS`           | `false`                                                 | `false` (pinned)                                          | OK     |
| `DB_MAX_OPEN_CONNS` × max instances | ≤ 40 (T213)                                            | 5 × 4 = 20; Neon `max_connections` = 901 today            | OK     |
| `NEON_AUTH_BASE_URL`               | production branch's auth URL                            | `https://ep-dawn-art-b53wsdjg.neonauth…/neondb/auth` (matches Neon) | OK |
| `RLS_ENFORCE`                      | `true` once a non-owner role is used (T154)             | unset (`rls_enforce=false` in logs)                       | Pending (#9) |
| `COOKIE_SECURE`                    | true                                                    | unset → default `!isDev` = **true**                       | OK     |
| `COOKIE_SAMESITE`                  | `Lax` (refresh cookie always `Strict`)                  | unset → `Lax`                                             | OK     |
| `COOKIE_DOMAIN`                    | empty (host-only)                                       | unset                                                     | OK     |
| `ENABLE_SWAGGER`                   | false                                                   | unset → false; log "Swagger documentation disabled"       | OK     |
| `PROMETHEUS_ENABLED` / `METRICS_TOKEN` | off, or on with a token                             | unset → off; log "Prometheus metrics disabled"            | OK     |
| `RATE_LIMIT_*`                     | enabled; defaults 100 anon / 1000 auth / 50 admin per min | unset → defaults; in-memory per instance                | OK (see #12) |
| `INVOICE_PAYMENTS_ENABLED`         | false until T169 (D11/T261)                             | unset → false                                             | OK     |
| `EXPORT_JOBS_ENABLED`              | false (T259)                                            | unset → false                                             | OK     |
| `SENTRY_*` sample rates / debug    | `SENTRY_DEBUG` false                                    | unset → defaults (errors 1.0, traces 0.1, debug off)      | OK     |
| `SMTP_HOST`                        | unset in production (would override Resend)             | unset                                                     | OK     |
| Log level                          | stdout JSON logger in production                        | stdout provider (no level knob; DEBUG lines not emitted by default) | OK |
| `DB_LOG_ALL_QUERIES`               | false                                                   | unset → false                                             | OK     |

### Cloud Run — secret env vars (Secret Manager, `:latest`)

| Secret                            | Expected                                  | Production has                          | Status        |
| --------------------------------- | ----------------------------------------- | --------------------------------------- | ------------- |
| `DATABASE_URL`                    | pooled URL, runtime role                  | attached, v1 (2026-10-24)               | OK (#13 to confirm pooler) |
| `JWT_SECRET`                      | ≥ 32 chars, not the default (enforced)    | attached, v1                            | OK            |
| `EDGE_PROXY_SECRET`               | set; equals the Worker secret             | attached, v1; Worker has it too         | OK            |
| `INTERNAL_JOBS_SECRET`            | set; equals the GitHub secret             | attached, v1; GitHub secret exists; jobs return 200 | OK |
| `RESEND_API_KEY`                  | set                                       | attached, v1 (2026-10-29)               | OK            |
| `RESEND_WEBHOOK_SECRET`           | set (T159)                                | secret doesn't exist                    | Missing (#16) |
| `SENTRY_DSN`                      | set                                       | secret exists, no version               | Missing (#3)  |
| `POSTHOG_API_KEY`                 | set if analytics wanted                   | secret exists, no version               | Needs Rob (#5) |
| `STRIPE_SECRET_KEY` / `STRIPE_WEBHOOK_SECRET` | set with T169                 | secrets exist, no version               | Pending (#15) |
| `STRIPE_CONNECT_WEBHOOK_SECRET` / `STRIPE_V2_WEBHOOK_SECRET` | set with T169  | secrets don't exist                     | Pending (#15) |
| `S3_COMPATIBLE_ACCESS_KEY_ID` / `…_SECRET_ACCESS_KEY` | R2 token (T160)       | secrets don't exist                     | Pending (#14) |
| `REDIS_URL`                       | Upstash, optional                         | secret exists, no version               | Needs Rob (#12) |
| `TEXTBELT_API_KEY`                | optional; SMS off without it              | secret exists, no version; SMS refused in prod (no mock) | OK (off by design) |
| Secrets as plain env              | none                                      | none: every secret is a `secretKeyRef`  | OK            |

### Cloud Run — service shape and identity

| Setting                                | Expected                                        | Production has                                         | Status |
| -------------------------------------- | ----------------------------------------------- | ------------------------------------------------------ | ------ |
| Max / min instances                    | 4 / 0 (T213)                                    | revision `maxScale=4`, no min                          | OK     |
| Concurrency / CPU / memory / timeout   | 40 / 1 / 512Mi / 60s                            | same                                                   | OK     |
| Ingress                                | `all` (Worker calls `*.run.app`)                | `all`; non-edge `/api/*` → 403 "Direct access is not allowed" (verified) | OK |
| Invoker                                | `allUsers` (edge secret is the gate)            | `allUsers`                                             | OK     |
| Runtime SA                             | dedicated, least privilege                      | `servicepro-api@`: **no project roles**; `secretAccessor` only on the 11 secrets it mounts; no user-managed keys | OK |
| Deploy SA `gh-deploy@`                 | AR writer, Run admin, secret viewer, actAs runtime SA only | `artifactregistry.writer`, `run.admin`, `secretmanager.viewer` (metadata only), `iam.serviceAccountUser` on `servicepro-api@` only; no keys | OK |
| Workload Identity                      | only this repo can impersonate `gh-deploy@`     | pool condition `repository_owner == javaknight1`, binding `attribute.repository/javaknight1/servicepro` | OK |
| Default compute SA                     | no broad roles                                  | `roles/editor`                                         | Wrong (#11) |
| Startup probe                          | present                                         | TCP :8080 (default)                                    | OK     |

### Endpoints (live probes, 2026-11-04)

| Check                                   | Expected          | Result                                                                    | Status |
| --------------------------------------- | ----------------- | ------------------------------------------------------------------------- | ------ |
| `/api/docs/index.html` (site and run.app) | not served      | site 404 (API JSON), run.app 403                                          | OK     |
| `/swagger/index.html`                   | not served        | 404                                                                       | OK     |
| `/metrics` on run.app                   | not served        | 404 (Prometheus off). On the site `/metrics` is just the SPA shell (200 HTML) | OK |
| `/debug/pprof/` (site and run.app)      | not served        | 404; no `net/http/pprof` or gin pprof import in the code                  | OK     |
| `/health/ready` on run.app              | minimal body      | `{"status":"healthy"}` (detail needs the jobs token)                      | OK     |
| Anonymous `/api/v1/...` through Worker  | reaches API       | API JSON with `X-Request-ID` (edge secret matches on both sides)          | OK     |
| `/version` through Worker               | v0.11.0 + SHA     | `0.11.0`, `4933271…`                                                      | OK     |
| `http://` → `https://`                  | 301               | **200, no redirect**                                                      | Wrong (#2) |

### Headers

| Header (static assets, `public/_headers`) | Expected                                    | Live                          | Status |
| ---------------------------------------- | ------------------------------------------- | ----------------------------- | ------ |
| `Content-Security-Policy`                | as in `_headers` (Stripe, Sentry, PostHog)  | matches                       | OK     |
| `Strict-Transport-Security`              | `max-age=31536000; includeSubDomains`       | matches                       | OK (consider `preload` later) |
| `X-Content-Type-Options` / `X-Frame-Options` / `Referrer-Policy` / `Permissions-Policy` | set | all present          | OK     |
| `Cache-Control` on `/`                   | `no-cache`                                  | `no-cache`                    | OK     |
| API responses (`/api/*`)                 | API's own CSP, `X-Frame-Options: DENY`, HSTS, COOP/CORP | all present       | OK     |

### Cloudflare Worker `servicepro-web`

| Setting                 | Expected                                | Live                                              | Status |
| ----------------------- | --------------------------------------- | ------------------------------------------------- | ------ |
| `API_ORIGIN` var        | Cloud Run URL                           | `https://servicepro-api-736486908135.us-east5.run.app` | OK |
| `EDGE_PROXY_SECRET`     | Worker secret                           | present (names-only listing)                      | OK     |
| `workers_dev` / previews | off                                    | `false` / `false`                                 | OK     |
| Latest deploy           | from the release workflow               | 2026-11-02 (v0.11.0)                              | OK     |
| Always Use HTTPS (zone) | on                                      | off (see #2)                                      | Wrong (#2) |

### GitHub Actions

| Name                            | Kind     | Present? | Used by                         | Status |
| ------------------------------- | -------- | -------- | ------------------------------- | ------ |
| `GCP_PROJECT_ID`, `GCP_REGION`, `GCP_WIF_PROVIDER`, `GCP_DEPLOY_SA`, `API_URL`, `SITE_URL` | var | yes | release-deploy, smoke, jobs | OK |
| `AR_REPOSITORY`, `CLOUD_RUN_SERVICE` | var | no (workflow defaults `servicepro` / `servicepro-api` apply) | release-deploy | OK |
| `VITE_STRIPE_PUBLISHABLE_KEY`   | var      | no       | web build                       | Pending (#15) |
| `MIGRATION_DATABASE_URL`        | secret   | yes      | migrate, backup                 | OK (workflow refuses :6543) |
| `INTERNAL_JOBS_SECRET`          | secret   | yes      | scheduled-jobs                  | OK     |
| `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID` | secret | yes | web deploy, backup endpoint | OK |
| `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY` | secret | **no** | db-backup                   | Wrong (#1) |
| `SENTRY_DSN_WEB`, `SENTRY_AUTH_TOKEN`, `SENTRY_ORG`, `SENTRY_CRONS_DSN` | secret | **no** | web build, releases, crons | Missing (#4) |
| `POSTHOG_API_KEY_WEB`           | secret   | no       | web build                       | Needs Rob (#5) |

### Neon (project `frosty-unit-21038953`, Free plan, aws-us-east-2)

| Setting                      | Expected / note                                            | Live                                                        | Status |
| ---------------------------- | ---------------------------------------------------------- | ----------------------------------------------------------- | ------ |
| Branch production uses       | `production` (default, primary)                            | `br-empty-math-b5zd95mf` = `production`; endpoint `ep-dawn-art-b53wsdjg` matches `NEON_AUTH_BASE_URL` | OK |
| Other branches               | dev only                                                   | `dev` (child of production, no compute activity)            | OK     |
| Compute size                 | 0.25–2 CU autoscaling (Free default)                       | 0.25–2 CU                                                   | OK     |
| Autosuspend                  | default (5 min on Free; `0` = plan default)                | `suspend_timeout_seconds = 0`                               | OK     |
| History / PITR retention     | Free max 6 h                                               | 21600 s = 6 h                                               | OK (D14: Launch plan + backups before paying tenants) |
| Snapshot schedule            | none on Free                                               | empty                                                       | OK (backups are the R2 dump, #1) |
| Pooler                       | API on `-pooler` host                                      | see #13                                                     | Needs Rob (#13) |
| Branch protection            | protected                                                  | not protected (paid feature)                                | Needs Rob (#10) |
| IP allow-list / public conn. | n/a on Free                                                | none / public allowed                                       | OK (plan limit) |
| `servicepro_app` role (T154) | `NOLOGIN`, `NOBYPASSRLS`, a login role in it              | exists; no members                                          | Pending (#9) |
| Neon Auth trusted domains    | `https://servicepro.sharkfins.xyz`                         | none; localhost allowed                                     | Missing (#7) |
| Neon Auth sign-up            | closed or harmless (invite-only app)                       | open, no email verification                                 | Needs Rob (#6) |
| Neon Auth email / OAuth      | own sender, own Google client                              | shared Neon sender and shared Google app                    | Needs Rob (#8) |
| Schema version               | equals the deployed release                                | 8 (v0.11.0); master has migrations up to 22 for the next tag | OK    |

### PII scrubbing

| Where                 | Expected                                         | Code today                                                                                   | Status |
| --------------------- | ------------------------------------------------ | -------------------------------------------------------------------------------------------- | ------ |
| Sentry (API)          | no cookies/auth headers/bodies                   | `sentry-go` defaults (`SendDefaultPII=false`: no cookies, `Authorization`, IPs); no custom `BeforeSend` | OK (inactive until #3) |
| Sentry (web)          | replay masked, URLs stripped                     | `maskAllText`, `blockAllMedia`, `beforeSend` / `beforeSendTransaction` scrubbers             | OK (inactive until #4) |
| PostHog (web)         | no autocapture, sanitized events                 | `autocapture: false`, `before_send: sanitizeCaptureResult`                                   | OK (inactive until #5) |

## Code: what fails open when an env var is missing

Startup behaviour in production (`ENV=production`) for each setting, after
this review's two `fix:` commits:

| Missing / wrong                        | Before                                                                 | Now                                         |
| -------------------------------------- | ---------------------------------------------------------------------- | ------------------------------------------- |
| `ENV` unset on Cloud Run               | silently ran as development: open sign-up, Swagger, non-Secure cookies, startup migrations, workers | **refuses to start** (`K_SERVICE` set + ENV=development) |
| `EDGE_PROXY_SECRET`                    | direct `*.run.app` access allowed; every visitor shares the Worker IP for rate limits | **refuses to start**           |
| `INTERNAL_JOBS_SECRET`                 | `/internal/jobs/*` 404 → reminders/overdue/expiry silently stop        | **refuses to start**                        |
| `COOKIE_SECURE=false`                  | auth cookies sent over plain HTTP                                      | **refuses to start**                        |
| `DATABASE_URL`                         | tried localhost (failed anyway)                                        | **refuses to start** with a clear message   |
| `RUN_MIGRATIONS_ON_STARTUP=true`       | migrated on every cold start                                           | **refuses to start**                        |
| `JWT_SECRET` missing/default/short     | already refused                                                        | unchanged                                   |
| `STRIPE_SECRET_KEY` without `STRIPE_WEBHOOK_SECRET` | already refused                                           | unchanged                                   |
| `RESEND_API_KEY`                       | already refused (no mock email in production)                          | unchanged                                   |
| Storage / SMS credentials              | client refused in production (no mock); feature off with a warning     | unchanged (intended until T160)             |
| `SENTRY_DSN`                           | mock tracker, warning only                                             | unchanged — see #3                          |
| `REDIS_URL`                            | in-memory per-instance store, logged                                   | unchanged — see #12                         |
| `ENV=prod` (typo, any non-`production` value) | Gin release mode but none of the production-only checks or the edge gate | unchanged — known gap; only `production` is used |
| `REGISTRATION_MODE=open`, `ENABLE_SWAGGER=true` | honoured if explicitly set                                    | unchanged — explicit opt-ins, not defaults  |

## Not checked

- Secret **values** (by rule): whether `DATABASE_URL` is the pooler (#13),
  whether the Worker and API edge secrets are byte-identical (indirectly
  proven: Worker-proxied `/api/*` passes the edge gate), Stripe key mode.
- Live `Set-Cookie` flags: needs a real sign-in; verified from code defaults
  and the unset env vars instead.
- Cloudflare zone settings other than the observed HTTP behaviour (no
  Cloudflare API session in this review): SSL mode, minimum TLS, WAF/bot rules.
- GCP budget alert existence (T213 doc records it as set up; not re-listed).
- Resend domain/DKIM status and the Resend webhook in Resend's dashboard.
