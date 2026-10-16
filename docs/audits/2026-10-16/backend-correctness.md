# ServicePro Backend Audit: Correctness, Code Health, Cleanup

Scope: `/Users/robavery/repositories/projects/saas/servicepro/backend` (HEAD `3df4280`, branch master). Read-only audit; nothing in the repo was edited (see the note under "Tooling results" about ignored test artifacts).
Severity scale: **Critical** (blocks deploy or corrupts money/data), **High** (user-visible wrong behaviour likely in prod), **Medium** (wrong under specific conditions, or a significant ops risk), **Low** (cleanup/hygiene).

---

## 0. Tooling results (run 2026-10-16, Go 1.27.1 toolchain, module `go 1.24.0`)

| Check | Result |
|---|---|
| `go build ./...` | OK |
| `go vet ./...` | clean |
| `go test ./... -count=1 -skip Integration` | **all 24 test packages pass** |
| `go test -race ./... -skip Integration` (what CI runs) | all pass, no races reported |
| `staticcheck` (v0.8.2) | 29 findings: 14 unused symbols (U1000), 5 deprecated `strings.Title` (SA1019), 7 capitalised error strings (ST1005), 1 S1009. Nothing correctness-critical. |
| `golangci-lint run` with the repo's `.golangci.yml` | **fails with 2 goimports issues**: `internal/analytics/aggregator.go:10`, `internal/repository/schedule_repository.go:4`. CI `checks.yml` runs lint, so CI will be red if it pins a similar version. |
| `golangci-lint --enable-only errcheck,ineffassign` | 105 errcheck + 3 ineffassign. Most are excelize `SetCellValue` calls and `Close` calls. The meaningful ones are listed in section 6. |
| GORM calls whose `.Error` is discarded (errcheck can't see these) | 16 sites, listed in section 6. |
| `go mod tidy -diff` | `github.com/posthog/posthog-go` should move from `// indirect` to direct. Otherwise tidy. |
| `deadcode ./cmd` | 476 unreachable functions. 21 whole packages are never linked into the binary (section 5). |

Side effect: the test `internal/services/unified_export_service_test.go:142` and siblings write into `./test_exports` inside the source tree. There are already about 400 leftover UUID directories in `internal/services/test_exports/`, and my run added 5 more. They are git-ignored, but two CSVs under that path are tracked by git: `internal/services/test_exports/{0e168f6f…,5a323d8a…}/customers_20260115_215151.csv`. **Fix:** use `t.TempDir()`, `git rm` the two tracked CSVs, and delete the directory. Also untracked: an 81 MB `backend/main` binary (ignored) and `backend/node_modules/` (ignored).

The `.golangci.yml` is very permissive: only govet (with `printf` and `copylocks` disabled), bodyclose, unconvert and misspell are enabled. errcheck, staticcheck, unused and ineffassign are all off, which is why the dead code and swallowed errors below have built up. **Fix:** enable `errcheck` (excluding Close and excelize), `staticcheck`, `unused` and `ineffassign`.

---

## 1. Deploy blockers (Cloud Run)

### 1.1 CRITICAL: the Docker image cannot build, because `.dockerignore` excludes source packages
- `backend/.dockerignore` has `pkg/` (under "# Binaries") and `docs/` (under "# Documentation").
- `pkg/` holds `pkg/auth`, `pkg/clients/*`, `pkg/database` and others, which are core code. `docs/docs.go` is the generated swag package, blank-imported at `internal/api/routes/routes.go:15`.
- **Verified:** I copied the module to my scratch directory applying the same exclusions and ran `go build ./cmd`. It fails with `no required module provides package .../backend/docs` and `.../pkg/clients/logging`, `.../pkg/auth`, `.../pkg/clients/errortracking`, and so on. (The Docker daemon wasn't running, so I simulated the build context.)
- CI (`.github/workflows/deploy-release.yml:41`) only runs `go build` on the checkout, so it never notices. `flyctl deploy` / `docker build` would fail.
- **Fix:** remove `pkg/` and `docs/` from `.dockerignore`. To exclude docs markdown, list it explicitly (e.g. `*.md`).

### 1.2 HIGH: `Dockerfile` COPY lines use shell syntax
- `backend/Dockerfile`: `COPY --from=builder /app/migrations ./migrations 2>/dev/null || true` and the same for `config`. COPY has no shell, so `2>/dev/null`, `||` and `true` are treated as extra source and destination paths. With more than one source the destination must end in `/`, so the build errors. Even if parsed, it would try to copy files named `2>/dev/null` and `||`.
- **Fix:** use a plain `COPY --from=builder /app/migrations ./migrations`. The `config` copy is pointless (only Go source lives there), so drop it.

### 1.3 HIGH: the runtime image lacks `wkhtmltopdf` and the `templates/` directory
- `internal/services/pdf_service.go:50-53,286` runs the `wkhtmltopdf` binary from PATH. It is wired at `internal/api/routes/services.go:95` (`svc.PDF`) and used by the unified export service for PDF exports (`unified_export_service.go:761-784`). `alpine:latest` doesn't include it, so every PDF export fails at runtime.
- `templates/email/template_engine.go:252` defaults to `TemplateDir: "templates/email"`, and `internal/services/pdf/config.go:229` to `./templates/pdf`. The Dockerfile copies only the binary. Today `templates`/`templates/email` aren't linked into the binary (section 5), so this is latent.
- **Fix:** replace wkhtmltopdf exports with the existing gofpdf generator (`internal/services/pdf`), or install it in the image. Better still, consolidate onto one PDF engine (see 5.3).

### 1.4 HIGH: local-disk state breaks with multiple Cloud Run instances
- Export jobs write files to `./exports/<jobID>/` (`unified_export_service.go:107,169,202`). Progress is held in process memory (`initProgress`, `:116`). Downloads are served from local disk (`:944`, `os.Open(job.FilePath)`).
- Export processing runs in a detached goroutine with `context.Background()` (`unified_export_service.go:121`). On Cloud Run, with "CPU only during requests" (the default), that goroutine is CPU-throttled after the POST returns, and the file lands on one instance's in-memory filesystem. A download request routed to another instance returns 404, and a scale-down loses it.
- **Fix:** upload exports to R2 through the storage client, return presigned URLs, and keep job state in the DB (it already is) rather than in memory. Alternatively, generate synchronously for small exports.

### 1.5 HIGH: in-process background workers on Cloud Run
Every instance starts these goroutines with `context.Background()`, and none is stopped on SIGTERM:
- the payment reminder worker, `internal/api/routes/routes.go:149-152` (interval 1h, and it **runs immediately on start**, `payment_reminder_worker.go:41`)
- the health runner, `routes.go:78`
- the pool metrics and KPI collectors, `cmd/main.go:173-178`
- the API rate-limiter cleanup, `middleware/ratelimit_api.go:64`

Consequences:
- With request-based CPU allocation, the hourly reminder tick may not run at all.
- With N instances (or each cold start), `ProcessAllTenants` runs N times concurrently. The dedupe check-then-send has no lock (see the money/reminder section).
- Shutdown doesn't wait for an in-flight reminder send.

**Fix:** move reminders to a Cloud Scheduler → authenticated HTTP endpoint (or a Cloud Run Job). Make the send idempotent with a unique `(invoice_id, reminder_number)` constraint and an insert-before-send "claim" row. Pass a cancellable root context from `main` and cancel it before `srv.Shutdown`.

### 1.6 MEDIUM: Supabase pooler compatibility
- `pkg/database/gorm.go:28` uses `postgres.Open(cfg.URL)` (pgx, extended protocol with a prepared-statement cache). Supabase's **transaction pooler** (Supavisor, port 6543) doesn't support named prepared statements across transactions, which causes `prepared statement "stmtcache_…" already exists` / `does not exist` errors under load.
- **Fix:** connect through the session pooler (port 5432) or direct connection, or use `postgres.New(postgres.Config{DSN: url, PreferSimpleProtocol: true})` (or `default_query_exec_mode=simple_protocol`). Also size `DB_MAX_OPEN_CONNS` (default 25, `config/config.go:393`) times the max instance count below the Supabase connection limit.
- `001_schema.sql` calls both `uuid_generate_v4()` (29 tables) and `gen_random_uuid()` (18). On Supabase `uuid-ossp` lives in the `extensions` schema. It resolves through the default search_path, but standardise on `gen_random_uuid()` (core PG13+) and drop the extension dependency.
- `001_schema.sql` isn't idempotent (`CREATE TYPE ...` without guards), and nothing applies it automatically. The app has **no migrations at startup** and no `AutoMigrate`; the only path is the `Makefile:223-230` `psql` via `docker exec`. That's fine for a first Supabase load, but every later schema change needs a manual process. Plan golang-migrate before the first production schema change (see section 8).

### 1.7 MEDIUM: Redis URL DB index is overwritten
- `pkg/database/redis.go:24`: `opt.DB = cfg.DB` always overwrites the DB number parsed from `REDIS_URL` (e.g. `redis://host:6379/2`) with `REDIS_DB` (default 0). Upstash/Memorystore usually use DB 0, so this is low impact, but it's surprising.
- **Fix:** only override when `REDIS_DB` is explicitly set.

### 1.8 MEDIUM: shutdown timeout exceeds Cloud Run's grace period
- `SERVER_SHUTDOWN_TIMEOUT` defaults to 30s (`config/config.go:385`). Cloud Run sends SIGKILL 10s after SIGTERM.
- `WriteTimeout` is 30s (`:382`), while synchronous PDF/report generation can be slow.
- **Fix:** set the shutdown timeout to about 8s on Cloud Run.

---

## 2. Boot requirements (what's needed for `./main` to start)

Order follows `cmd/main.go`. "Fatal" means the process exits.

| # | Dependency | Required? | Where | What it's used for |
|---|---|---|---|---|
| 1 | `JWT_SECRET` (≥32 chars, not the placeholder) | **Fatal if missing/short**, in all envs | `config/config.go:596-602,618` | Access/refresh JWT signing (`pkg/auth/jwt.go`) |
| 2 | `STRIPE_WEBHOOK_SECRET` | **Fatal only if `STRIPE_SECRET_KEY` is set** | `config/config.go:605` | Webhook signature verification |
| 3 | Postgres `DATABASE_URL` (reachable, ping succeeds) | **Fatal** | `cmd/main.go:114-118`, `pkg/database/gorm.go:28,45` | Everything |
| 4 | Redis `REDIS_URL` (reachable, PING succeeds) | **Fatal** | `cmd/main.go:130-134`, `pkg/database/redis.go:30` | Auth login rate limit and lockout (`middleware/ratelimit.go`), API rate limiting (`middleware/ratelimit_api.go`, which has an in-memory fallback), permission cache (`services/permissions/checker.go`), email-verification and password-reset tokens (`services/email_verification_service.go`, `services/password_reset_service.go`), CSV import progress (`services/import_service.go`), health check (`internal/health/checks.go`) |
| 5 | Email client | **Fatal only if the chosen provider's constructor errors** | `cmd/main.go:138-142`; selection `pkg/clients/email/factory.go:75-93` | Verification, password reset, invitations, quote/invoice send, payment reminders |
| 6 | Storage client | Optional (warn) | `cmd/main.go:146-151`; selection `pkg/clients/storage/factory.go:71-79` | Quote/invoice PDFs (`services/document_pdf_service.go`, presigned URLs at `:250`), profile pictures (`services/profile_picture_service.go`) |
| 7 | SMS client | Optional (warn) | `cmd/main.go:154-159` | **Nothing.** It's never passed to the reminder service (`routes/services.go:161` hard-codes `SMSClient: nil`) |
| 8 | Error tracking (Sentry) | Optional, **always fails**, see 3.1 | `cmd/main.go:91-97` | Panic capture in `middleware/error_handler.go:126-145` |
| 9 | Logging (BetterStack/Loki/CloudWatch) | Optional, **always fails in prod**, see 3.2 | `cmd/main.go:77-83` | Remote log shipping |
| 10 | Analytics (PostHog) | Optional | `cmd/main.go:100-106`, `pkg/clients/analytics/factory.go:61` | One `payment_received` event (`routes.go:155-163`) plus whatever handlers call |
| 11 | Metrics (Prometheus) | Optional; CloudWatch is selected and fails if `PROMETHEUS_ENABLED=false` in non-dev | `cmd/main.go:162-179`, `pkg/clients/metrics/factory.go:74-92` | `/metrics`, DB pool and KPI gauges |
| 12 | Stripe `STRIPE_SECRET_KEY` | Optional. Billing, Stripe, public-invoice-payment and webhook routes are only registered when it's set | `routes/services.go:112-145`, `routes/handlers.go:48,85-93` | Subscriptions and billing, invoice checkout links |
| 13 | AWS creds / SES / SNS | **Not required to boot**, but see the trap below | `config/config.go:435-459` | SES email (fallback), SNS SMS (unused), SES bounce webhooks (`routes/v1/auth_routes.go:29-30`) |

**Minimum to boot:** `JWT_SECRET`, a reachable Postgres, and a reachable Redis. That's all. Without Redis the server **will not start**, even though rate limiting has an in-memory fallback. If you want Redis optional, make `NewRedisClient` failure non-fatal and give the permission checker and token services a DB- or memory-backed fallback. Otherwise provision Upstash or Memorystore.

**Email provider trap.** The detection order is SMTP_HOST → RESEND_API_KEY → (AWS_REGION and AWS_SES_FROM_EMAIL) → mock. `AWS_REGION` defaults to `us-east-1` (`config.go:436`) and `AWS_SES_FROM_EMAIL` defaults to `noreply@servicepro.com` (`:440`), so **with no `RESEND_API_KEY` the app silently picks SES**. `awsconfig.LoadDefaultConfig` (`pkg/clients/email/ses/ses.go:83`) doesn't fail without credentials, so boot succeeds and every send fails at runtime. For Resend, set `RESEND_API_KEY` and `RESEND_FROM_EMAIL`, and make sure `SMTP_HOST` is empty. **Fix:** remove the SES defaults, or require an explicit `EMAIL_PROVIDER`.

**Storage trap.** If `S3_COMPATIBLE_ACCESS_KEY_ID` is empty, the factory returns the in-memory **mock** storage (`storage/factory.go:73-78`), even in production. PDFs and profile pictures then "upload" successfully into RAM and vanish. **Fix:** in `ENV=production`, return an error instead of the mock.

### R2 migration notes (storage)
- Config already supports R2: `S3_COMPATIBLE_ENDPOINT=https://<acct>.r2.cloudflarestorage.com`, `S3_COMPATIBLE_REGION=auto`, and the bucket and keys. Endpoint override is at `pkg/clients/storage/s3/s3.go:105-108`.
- **Bug for R2 (High once you set `S3_COMPATIBLE_PUBLIC_URL`):** `s3.go:328-329` rewrites the host of a **presigned** URL from the endpoint to `PublicURL`. SigV4 signs the Host header, so the rewritten URL fails with `SignatureDoesNotMatch`. R2 custom domains and r2.dev also don't accept S3 presigned query auth. `.env.example:121,132` sets both, so following the example breaks invoice and quote PDF links (`document_pdf_service.go:250`).
  - **Fix:** only rewrite in local MinIO dev (e.g. behind a `S3_COMPATIBLE_PRESIGN_REWRITE` flag). In prod, return presigned URLs untouched and use `PublicURL` only for objects that are genuinely public (avatars).
- `UsePathStyle` is only applied when an Endpoint is set. That's fine for R2 (either style works).

### Resend migration notes (email)
- `pkg/clients/email/resend` already exists and is registered (`cmd/main.go:44`).
- SES-specific bounce/complaint webhooks (`routes/v1/auth_routes.go:29-30` → `EmailVerificationHandler.HandleSESBounce/HandleSESNotification`) need Resend webhook equivalents (`email.bounced`, `email.complained`), or they should be deleted.
- After the migration, `aws-sdk-go-v2/service/{ses,sns,cloudwatch,cloudwatchlogs}` and `pkg/clients/{email/ses,sms/sns,logging/cloudwatch,metrics/cloudwatch}` can be removed. The S3 SDK stays for R2.

---

## 3. Integrations that are wired but non-functional

### 3.1 HIGH: Sentry error tracking never initialises
- `pkg/clients/errortracking/factory.go:59-66` looks providers up in a registry populated by `init()` in `errortracking/mock` and `errortracking/sentry`. **Neither package is imported anywhere in the binary.** I confirmed with `go list -deps ./cmd`: both are absent.
- So `errortracking.NewClient` always returns "provider not registered", `main.go:93` logs a warning, and `errorTrackingClient` is nil. `RecoveryMiddleware` (`middleware/error_handler.go:126`) nil-checks it, so panics are recovered but **never reported**.
- The recent "Integrated Sentry logs" commit (`d15346e`) only touched the frontend.
- **Fix:** add `_ ".../pkg/clients/errortracking/sentry"` and `/mock` blank imports in `cmd/main.go`.

### 3.2 MEDIUM: remote logging providers are not linked in
- `pkg/clients/logging/factory.go` picks BetterStack → Loki → CloudWatch (when `AWS_REGION` is set, which it always is by default) → mock. Only `logging/mock` is imported (`cmd/main.go:69`). `logging/betterstack`, `logging/loki` and `logging/cloudwatch` aren't in the binary.
- In production, `logging.NewClient` fails ("cloudwatch not registered") and the app falls back to the default logger. On Cloud Run, stdout goes to Cloud Logging anyway, which is probably what you want.
- **Fix:** delete the CloudWatch provider, and default to a structured JSON stdout logger in prod.
- `metrics/cloudwatch` is likewise unlinked. In prod with `PROMETHEUS_ENABLED=false`, the metrics client fails, so the pool and KPI collectors never start.

### 3.3 MEDIUM: OpenTelemetry tracing is never initialised
- `pkg/clients/tracing/{,otel,sentry,mock}` aren't imported by the binary, so the `OTEL_*` config (`config.go:542-548`) is dead.
- **Fix:** wire it or delete it, along with the `go.opentelemetry.io/*` deps.

### 3.4 MEDIUM: SMS payment reminders can never be sent
- `routes/services.go:159-164` constructs `PaymentReminderService` with `SMSClient: nil`. The comment says it "will be set there if needed", but nothing ever sets it.
- `routes.Setup` receives `sms` and only stores it in `Clients.SMS`, which nobody reads.
- Customers whose preferred channel is SMS get no reminder. What happens depends on `resolveChannel` and `SendReminder`; see the reminder findings.
- **Fix:** pass `sms` into `SetupServices`, or remove SMS.

### 3.5 MEDIUM: tenant settings that the backend ignores
`TenantSettings` (`internal/models/tenant.go:48-67`) exposes `Timezone`, `DateFormat`, `Currency`, `InvoicePrefix`, `QuotePrefix`, `JobPrefix`, `DefaultPaymentTermID`, `DefaultTaxRateID` and the new `BusinessHours` (commit `3df4280`). **None is read anywhere in backend code** (0 references outside the model). So:
- Business hours aren't enforced in scheduling. `conflict/detector.go:254 checkBusinessHours` exists, is **unused** (staticcheck U1000), and hard-codes 8:00–18:00 Mon–Fri in UTC epoch hours.
- Tenant timezone isn't used for "days overdue", due dates, schedules or reports. All date math is server-time/UTC.
- Currency and prefixes set in the UI have no effect on invoices or quotes.

**Fix:** either implement them (load the tenant's `time.Location` once per request or job and use it for day boundaries), or hide them in the UI.

---

## 4. Correctness bugs found in the code paths I reviewed

### 4.1 HIGH: legacy Stripe webhook handler panics on its main events
- `internal/api/handlers/webhook_handler.go:108,130,165`: `bgCtx := ctx.(interface{ Request() *http.Request }).Request().Context()`.
- `ctx` comes from `c.Request.Context()` (`:71`), which is a `context.Context`, not a `*gin.Context`, and has no `Request()` method. This unchecked type assertion **panics on every** `customer.subscription.created/updated/deleted`, `invoice.paid` and `invoice.payment_failed` event.
- `RecoveryMiddleware` turns the panic into a 500, so Stripe retries for 3 days and subscription state is never updated through this endpoint.
- The route is `POST /api/v1/webhooks/stripe` (`routes/v1/auth_routes.go:33-34`), registered whenever `STRIPE_SECRET_KEY` is set.
- It also calls `webhook.ConstructEvent` (`:62`) without `IgnoreAPIVersionMismatch`, so if the Stripe account's API version differs from stripe-go v76's pinned version, every event fails signature construction.
- **Fix:** change the helper signatures to `ctx context.Context` and use it directly. Better, delete this handler and point Stripe at the newer `POST /api/v1/stripe/webhooks` (`internal/services/stripe/webhook.go:224`, which already uses `ConstructEventWithOptions{IgnoreAPIVersionMismatch: true}`), porting the subscription-state handling into that event processor. Two webhook endpoints that both handle `invoice.paid` is itself a double-processing hazard if both are configured in the Stripe dashboard.

### 4.2 MEDIUM: tenant creation is a compensating "transaction" that permanently burns the slug
- `internal/services/tenant_service.go:109-136` creates the tenant, then adds the owner membership. On failure it calls `tenantRepo.Delete` (`:135`), which is a **soft delete** (`models.Tenant.DeletedAt`, `repository/tenant_repository.go:97-100`).
- `tenants.slug` is a **global** `UNIQUE` (`001_schema.sql:1036`), not a partial unique index, so the user can never retry with the same slug.
- The default membership tier assignment failure (`:154-158`) is only logged, which leaves a tenant with no tier.
- **Fix:** wrap tenant, owner membership and tier in one `db.Transaction` (the repo needs a `WithTx`), and use `UNIQUE (slug) WHERE deleted_at IS NULL`. Also drop the leftover debug verification query (`:145-150`) and the hard-coded admin role UUID (`:119`).

### 4.3 MEDIUM: accepting an invitation is non-transactional and doesn't reactivate removed members
- `internal/services/invitation_service.go:195-257`: the tenant_user create/update and the invitation status update are separate writes. A crash between them leaves an accepted membership with a still-pending invitation (reusable token).
- If a `tenant_users` row already exists (e.g. the user was previously removed, `is_active=false`), the code only sets `AcceptedAt` (`:226-229`). **It doesn't set `IsActive=true` or the invitation's `RoleID`**, so a re-invited user stays deactivated with their old role.
- `:207` and `:286` discard the error from saving the expired status.
- **Fix:** do it in one transaction with `SELECT … FOR UPDATE` on the invitation, and set `IsActive`, `RoleID` and `AcceptedAt` on the existing row.

### 4.4 LOW: user repository has methods that always return errors
- `repository/user_repository_gorm.go:66-71,86-89,102-105`: `IncrementFailedLoginCount(int)`, `LockAccount(int, …)` and `ResetFailedLoginCount(int)` always return `errors.New("use …ByUUID instead")`. They exist only to satisfy the legacy `UserRepositoryInterface` (`repository/interface.go:16-22`).
- No production caller uses them, but anyone calling them through the interface fails silently in logic.
- **Fix:** delete the legacy interface and the dead `repository/user_repository.go`, which is a `database/sql` implementation with `int` IDs and `NewUserRepository` that nothing calls.

### 4.5 LOW: repository lookups leak the raw GORM not-found error
- `repository/payment_repository.go:36,46` (`GetPaymentByID`, `GetPaymentByStripeID`) and `permission_repository.go:120` return the raw `gorm.ErrRecordNotFound`.
- Most services map not-found correctly. These two payment lookups currently have **no callers** (dead).
- **Fix:** either define `ErrNotFound` per repo or delete them.

### 4.6 LOW: `/api/v1/version` is hard-coded
- `routes/v1/auth_routes.go:40-45` returns a hard-coded `"1.0.0"`. Meanwhile `internal/version` and `handlers/version_handler.go:26 GetVersion` exist but are unreachable.

---

## 5. Dead code, duplicates and stale integrations (cleanup)

### 5.1 Packages never linked into the binary
Found by comparing `go list ./...` against `go list -deps ./cmd`. These 21 packages are never linked into the binary:
- `internal/analytics`, the in-house event store. It also has a broken `ON CONFLICT` target, see 7.L.
- `internal/api/handlers/webhooks`
- `internal/api/validators`
- `internal/errors`, the custom error types that CLAUDE.md says to use. They're unused.
- `internal/models/payment` (its tables don't exist)
- `internal/services/recurring` (its bugs are in 7.L5)
- `internal/services/subscriptions` (its tables don't exist)
- `pkg/clients/errortracking/{mock,sentry}`, which **should** be linked (see 3.1)
- `pkg/clients/logging/{betterstack,loki,cloudwatch}`
- `pkg/clients/metrics/cloudwatch`
- `pkg/clients/tracing/{,mock,otel,sentry}`
- `pkg/search`
- `pkg/utils/permissions`
- `templates`, `templates/email`

### 5.2 Wired-up code that nothing reaches
These are the larger chunks among the 476 unreachable functions:
- `handlers/assignment_handler.go`: the whole handler is unrouted. `services/assignment_service.go` and `pkg/notifications` are therefore dead too.
- Download stack: `handlers/download_handler.go`, `handlers/rate_limit.go` (a fourth rate-limiter implementation) and `handlers/tracking.go`.
- Middleware:
  - `middleware/access_control.go` and `config/access_control.go`, the entire access-control layer
  - `middleware/access_logging.go`
  - `middleware/auth.go` (`AuthMiddleware`, `RequireRole`, `OptionalAuth`, superseded by `PermissionMiddleware`)
  - `middleware/logger.go`. Its `RequestIDMiddleware` isn't installed, yet CORS exposes `X-Request-ID`.
  - `middleware/error_handler.go` `ErrorHandler`, `NotFoundHandler` and `MethodNotAllowedHandler`
- Services:
  - `services/quote_status_machine.go`, the entire quote state machine (see 7.M4)
  - `services/stripe/webhook_handler.go`, the DB-backed webhook service with 53 unreachable functions
  - `services/invoice_template_service.go` (its tables don't exist)
  - `conflict/validator.go` and `conflict/resolver.go`
- Repositories: `repository/user_repository.go` (legacy `database/sql`) and `pkg/database/postgres.go` (`NewPostgresDB`). They're the only users of `lib/pq`.

### 5.3 Duplicate implementations
- User repository: `user_repository.go` (int IDs, dead) and `user_repository_gorm.go` (live), plus the legacy `UserRepositoryInterface` (4.4).
- Two Stripe webhook endpoints: `/api/v1/webhooks/stripe` (panics, 4.1) and `/api/v1/stripe/webhooks`. A third, DB-backed implementation is dead.
- Two PDF engines: gofpdf in `services/pdf` (used by `DocumentPDFService`) and the wkhtmltopdf CLI in `services/pdf_service.go` (used by exports). Keep gofpdf.
- Four rate limiters:
  - `middleware/ratelimit.go` (auth)
  - `middleware/ratelimit_api.go`
  - `middleware/ratelimit_invoice.go`
  - `handlers/rate_limit.go` (dead)
- Two job status validators that disagree: `pkg/validators/job.go` and `models/job_status.go` (7.H7).
- `gen_random_uuid()` and `uuid_generate_v4()` are mixed throughout the schema.
- `log.Printf` is used alongside the `logging` package. `routes/services.go` uses `log.Printf`, against the CLAUDE.md convention.

### 5.4 Smaller items from staticcheck
- Unused: `analytics/tracker.go:323`, `middleware/ratelimit_api.go:36` (`mu`), `assignment_service.go:113`, `permissions/checker.go:23`, `recurring/pattern.go:312`, `stripe/client.go:33`, `pkg/csv/parser.go:99`, `pkg/notifications/assignment_notification.go:111`, `templates/renderer.go:192`.
- Deprecated `strings.Title`: `unified_export_service.go:811`, `pkg/search/customer_search.go:144`, `templates/email/template_engine.go:308`, `templates/renderer.go:114`.
- `invoice_template_service_test.go:223` has a test disabled with a `_` prefix.

### 5.5 Suggested deletion order (low risk)
1. Delete the dead packages and files in 5.1 and 5.2, except errortracking and its sentry provider, which you should **link in** instead.
2. Delete `tests/Makefile`, which conflicts with the "single root Makefile" rule in CLAUDE.md.
3. After the Resend and R2 migration, delete the SES, SNS and CloudWatch providers and their AWS SDK modules.
4. Run `go mod tidy`.

This removes roughly 10-12k lines and `lib/pq`, `otel`, `cloudwatch*`, `sns`, `ses` and `gorm/driver/sqlite`. sqlite is a test-only dependency, used by `*_test.go` with AutoMigrate.

---

## 6. Swallowed errors (selected, all verified)

Unchecked GORM writes (`.Error` discarded):
- `unified_export_service.go:141,158,225,981,1038`: export job status saves. A failed save leaves the job stuck in `pending`/`processing`.
- `payment_reminder_service.go:121` (`Count`, the dedupe check, ignores DB errors and treats an error as count=0), `:180`, `:187` (reminder status updates) and `:229`.
- `invitation_service.go:207,286`
- `invoice_template_service.go:557`
- `customer_report_service.go:570`
- `subscriptions/notifications.go:448,526,562` (dead)

Other unchecked errors:
- `job_service.go:158,480,529,944-946`: `recordStatusTransition` results discarded with `_ =`. The insert at :158 always fails (see 7.M1).
- `import_service.go:341`: `json.Unmarshal` unchecked.
- `export_handler.go:238`: `io.Copy` unchecked.
- `cmd/main.go`: `logging.Fatal(...)` followed by `os.Exit(1)` skips the deferred `Close`/flush for logging, Sentry and PostHog. Use `log.Fatal` only after flushing, or return from a `run() error`.

---

## 7. Money, state machines, concurrency, reminders, time zones, schema drift
(Two parallel sub-audits produced this section. I re-checked C1, C3, the reminder `status` ambiguity, the missing trigger, the missing `TenantID` and the missing search functions myself.)

### Critical
- **7.C1 Recording a payment never updates the invoice.**
  - Where: `invoice_service.go:412-415` says "triggers will update invoice", but `001_schema.sql` has no such trigger. The only invoice/payment triggers are at :614 (updated_at), :634 (number) and :719 (updated_at). `amount_due` is generated from `amount_paid`, which is never incremented.
  - Effect: the invoice stays `sent`, A/R still shows it as owed, reminders keep firing, the online pay link charges the full amount, and the overpayment guard never trips.
  - Fix: in one transaction, `SELECT … FOR UPDATE` the invoice, insert the payment, then update `amount_paid`, `status` and `paid_date`. Reject `draft`, `cancelled` and `refunded`.
- **7.C2 Stripe checkout amount doesn't equal the invoice balance.**
  - Where: `invoice_payment_service.go:130-167`.
  - Problems:
    - `Quantity.IntPart()` (line 133) truncates fractional quantities.
    - Line and invoice discounts are ignored.
    - `amount_paid` is ignored.
    - Currency is hard-coded to `"usd"`.
  - Then `MarkInvoiceAsPaid` adds `AmountTotal`, which can overshoot (negative `amount_due`). A `partially_paid` invoice can't be paid online (`invoice_service.go:664` accepts only `sent`).
  - Fix: one line item for `total_amount - amount_paid`, rounded to cents.
- **7.C3 Invoice numbering wedges.**
  - Where: `invoice_service.go:530-548` uses `MAX(invoice_number)+1` through the soft-delete scope, while the UNIQUE constraint (`schema:571`) includes deleted rows.
  - Effect: deleting the newest draft leads to a unique violation on **every** later create. It also races under concurrency and is global across tenants.
  - Fix: use a per-tenant sequence or counter row with `FOR UPDATE`. The DB trigger at :618-635 never fires because Go always sets the number.
- **7.C4 Job numbering wedges.**
  - Where: `models/job.go:100-110` uses `COUNT(today)+1` excluding soft-deleted jobs, with a UNIQUE `job_number`.
  - Effect: after any deletion, job creation fails until midnight. It also races.
- **7.C5 (schema) Business tables never get `tenant_id`.**
  - Where: the `Customer`, `Job`, `Quote`, `Invoice`, `Schedule`, export/import, payment-term, tax-rate and template models have no `TenantID`. Schema adds nullable `tenant_id` at :1107-1141, filled only by the dev seed.
  - Effects:
    - The A/R aging report (`ar_aging_service.go:91,206,263,296`) is always empty for API-created data.
    - Reminders (`payment_reminder_service.go:101`, `customers.tenant_id = ?`) never match API-created customers.
    - **Repositories, exports and reports apply no tenant filter**, so tenants can read each other's data. That is a security issue; flagging it for the security audit.
  - Fix: add `TenantID` to the models, set it from the request context, filter in every repository, then make the columns NOT NULL.

### High
- **7.H1 Payment reminder query always errors.** `payment_reminder_service.go:93` has an unqualified `Where("status IN ?")` while joining `customers`, and both tables have `status`. The result is `column reference "status" is ambiguous` for every tenant on every run. **Reminders have never been sent.** Use `invoices.status` and `invoices.due_date`.
- **7.H2 Generic invoice `PUT` can set any status.** `invoice_handler.go:421-423` → `invoice_service.go:194,209`. Only `paid` is protected. `{"status":"paid"}` marks an invoice paid without money, and cancelled or refunded invoices can be revived or edited. Remove `status` from the update DTO and use a transition table.
- **7.H3 Totals drift on partial edits.** `invoice_handler.go:445-505` recalculates only when `lines` is sent.
  - A discount or tax-rate change alone leaves `total_amount` stale.
  - `lines` without `tax_rate` zeroes every line's tax.
  - Fix: a single server-side recalculation inside the update transaction.
- **7.H4 Webhook idempotency lives only in memory.** `stripe/webhook.go:34,106,127` keeps a per-process map. `MarkInvoiceAsPaid` (`invoice_service.go:672-707`) does read-then-write with no lock and no `invoice_payments` row.
  - Effect: replays on another instance or after a restart double-apply partial payments, and a cancelled invoice can flip to paid.
  - Fix: persist event IDs with a unique constraint, and key payments on a unique `stripe_payment_intent_id`.
- **7.H5 Refunds are ignored.** `stripe/events.go:416-427` only logs `charge.refunded`. `refunded` status is never set and `amount_paid` is never decreased.
- **7.H6 Quote update silently reverts the customer change.** `quote_repository.go:85` calls `tx.Save(quote)` with preloaded `Customer`, `CreatedByUser` and `UpdatedByUser`. GORM's save-associations resets the belongs-to foreign keys from the preloaded structs, so `CustomerID` and `UpdatedBy` changes are lost. Use `tx.Omit(clause.Associations).Save(quote)`.
- **7.H7 Two disagreeing job state machines.** `pkg/validators/job.go:247-285` vs `models/job_status.go:89-240`.
  - The validator has no `new`, `en_route`, `invoiced` or `paid`. Jobs are created as `new` (`job_service.go:128`), so `CancelJob` (:556) fails for new jobs.
  - `StartJob` rejects `en_route`.
  - `UpdateJob` (:262-264) changes status without history or `ActualStartAt`, so a later `CompleteJob` fails at `validators/job.go:334`.
  - Status and history writes aren't in a transaction.
- **7.H8 Customer search, autocomplete and stats always 500.**
  - `customer_repository.go:268,291` call `search_customers()`, `:315` calls `autocomplete_customers()`, and `:330` reads the view `customer_search_stats`. None exist in `001_schema.sql` (verified).
  - Endpoints affected: `GET /customers/advanced-search`, `/autocomplete` and `/stats`.
- **7.H9 Customer report endpoints are broken.**
  - `customer_report_service.go:502` reads `v_customer_cohorts` and `:544` calls `refresh_customer_metrics_cache()`. Neither exists.
  - `customer_metrics_cache` is never populated, so the top, at-risk and segment reports are empty.
  - `:116,140` use `Table("customers")` while filters reference alias `c.` (`:742-757`), so any filtered request fails with "missing FROM-clause entry for table c".

### Medium
- **7.M1 Job creation's first history row always fails.** `job_service.go:158` passes `from_status=""` into a `job_status NOT NULL` enum (`schema:334`). The error is swallowed, so job history is missing its creation entry.
- **7.M2 Assignments get a 1970 timestamp.** `job_service.go:163,711` don't set `AssignedAt`, and the model has no default tag, so the column gets epoch 0 (1970).
- **7.M3 `default:true` tags flip `false` to `true` on insert.** Affected: `InvoiceLine.Taxable` (`models/invoice.go:209`), so non-taxable lines get taxed; also `TaxRate.IsActive`, `PaymentTerm.IsActive`, `Schedule.RemindersEnabled` and `TenantUser.IsActive`. Use `*bool` or drop the tag.
- **7.M4 Quote expiry doesn't exist.**
  - `QuoteStatusMachine` is never constructed and nothing expires quotes.
  - Even if it were wired:
    - its `→expired` transition is only allowed from `accepted`;
    - `ValidateTransition` checks un-preloaded `Items`;
    - notifications are sent before the commit.
  - Sent quotes are still editable (`models/quote.go:198`).
- **7.M5 Nothing stops double invoicing.** `invoice_handler.go:302-303` accepts any `job_id` or `quote_id`, including declined or draft quotes and already-invoiced jobs. There's no uniqueness and no status change, and jobs never move to `invoiced` or `paid`.
- **7.M6 Quote accept and reject race.** `quote_repository.go:240-286` does check-then-update. Add `WHERE status='sent'` and check `RowsAffected`.
- **7.M7 Tax and discount ordering.**
  - Line tax is computed before the invoice-level discount (`invoice_handler.go:337-365`).
  - `DiscountPercentage` is stored but never applied.
  - The invoice PDF lines don't sum to the totals (`document_pdf_service.go:360`), and the quote PDF re-sums in float64.
- **7.M8 Reminder logic** (once 7.H1 is fixed):
  - A single pass sends #1, #2 and #3 at once for long-overdue invoices (`payment_reminder_service.go:116-141`).
  - Manual reminders use `MAX+1` numbering (:228-233), which permanently skips automatic #1.
  - `pending` or `failed` rows block retries forever (:164-190).
  - Cross-instance duplicate sends are prevented by `UNIQUE(invoice_id, reminder_number)` (`schema:1824`).
  - `Worker.Stop()` panics if called twice (`payment_reminder_worker.go:57`).
- **7.M9 Time zones.**
  - Due dates are DATE values read as UTC midnight, then `.In(loc)`, which shifts them a day early in US zones (`payment_terms_service.go:255-264,313-360`). The `Hours()/24` truncation is off by one across DST.
  - `daysOverdue` uses UTC (`payment_reminder_service.go:113,148`).
  - The conflict workload "day" is UTC (`conflict/detector.go:291-293`).
  - Tenant `Timezone` and `BusinessHours` are never read (3.5).
- **7.M10 Schedule range queries drop events that overlap the boundary.** `schedule_repository.go:64-69,111` use containment (`start>=from AND end<=to`); they should use overlap. Also, nothing writes `schedules`, so `/conflicts/check` never finds technician conflicts.
- **7.M11 Revenue reports are always $0.** Nothing writes `revenue_transactions`. The table also lacks `tenant_id`, and growth-rate calculation ignores the filters.
- **7.M12 Invitation uniqueness ignores status.** `UNIQUE (tenant_id, email)` (`schema:1095`) means re-inviting after a cancelled, accepted or expired invite fails with a duplicate-key error. Make it a partial unique index `WHERE status='pending'`.
- **7.M13 Global uniqueness that should be per tenant.** Customer email (`schema:221`), the one-default payment term (:567) and the one-default invoice template (:769) are unique system-wide instead of per tenant.
- **7.M14 Job money uses floats.** Job and JobMaterial money fields are `float64` in Go but `DECIMAL` in SQL (`models/job.go:65-68,144-147`). `Job.TotalAmount` is never computed.
- **7.M15 Overpayment and delete/cancel holes.** The overpayment check races (no lock), and `DeleteInvoice` and `CancelInvoice` allow `partially_paid` invoices that have payments (`invoice_service.go:249,482`).

### Low
- **7.L1 Quote rounding.** Item totals are rounded but the subtotal sums unrounded products (`models/quote.go:232-247`), so they can differ by a cent.
- **7.L3 Cents conversion.** `stripe/client.go:638-640` uses `IntPart()` truncation and assumes 2-decimal currencies. `stripe_handler.go:63` takes the amount as `float64`. A new UUID idempotency key per request means client retries can duplicate payment intents.
- **7.L5 Recurring generator** (dead code): Jan 31 + 1 month skips February; the time of day is treated as UTC and isn't DST-aware; the occurrence cap counts per window, not per series; a nil `JobTemplateID` panics (`recurring/generator.go:286,373-376,180-182,380`).
- **7.L Missing schema objects referenced by dead code.** Includes `invoice_summary`, `revenue_by_month`, `quote_status_history`, `quote_templates`, `invoice_template_*`, `webhook_events`, `payment_records`, the `subscriptions/*` tables and the `models/payment/*` tables. `models.Payment` doesn't match the `payments` table (12 missing columns), and `payment_refunds` doesn't exist. `analytics/store.go:354` has an `ON CONFLICT` target with no matching unique index. `ConflictType` values violate the CHECK at `schema:419`.

### Indexes worth adding
1. `customers (tenant_id, LOWER(email)) WHERE deleted_at IS NULL`. Lookups use `LOWER(email)` on every create (`customer_repository.go:60,175`), which is a sequential scan today.
2. GIN `gin_trgm_ops` indexes on customer name and email. pg_trgm is installed but unused, and the `%term%` searches at `customer_repository.go:123,201` scan.
3. `invoices (tenant_id, status, due_date) WHERE deleted_at IS NULL`, for reminders and A/R aging.
4. Drop `idx_invoices_deleted_at` (`schema:611`), which indexes `deleted_at` with `WHERE deleted_at IS NULL`.
5. `jobs (tenant_id, status)` and `(tenant_id, scheduled_start_at)`, partial on `deleted_at IS NULL`.

N+1 queries (minor): `permission_repository.go:90` (one association query per role) and `invitation_service.go:328` (one user lookup per invitation).

---

## 8. Gin/GORM coupling and the chi + pgx + golang-migrate migration

### Coupling counts (non-test files; tests in parentheses)
- `github.com/gin-gonic/gin` is imported by **66 files** (14):
  - 31 handlers
  - 13 middleware
  - routes, routeconfigs and `v1/*`
  - 2 in `pkg`
  - `templates/preview_handler.go`
- `gorm.io/gorm` is imported by **54 files** (14):
  - 18 services: `invoice_service`, `job_service`, `payment_reminder_service`, the reports, `unified_export`, `invitation`, `import`, `payment_terms`, `assignment`, `quote_status_machine`, `invoice_template`, `subscriptions/*` and `stripe/webhook_handler`
  - 9 repositories
  - 18 models (hooks such as `BeforeCreate`, `gorm.DeletedAt`, and 902 `gorm:"…"` struct tags)
  - 1 handler (`job_handler.go:103`, an `ErrRecordNotFound` check)
  - `pkg/database`
- 111 `Preload(` calls, 16 `Raw`/`Exec` calls.
- Business logic in 13 model files lives in GORM hooks, for example job-number generation in `models/job.go:100-110`.

### Is there a repository boundary already?
Only partly.
- `internal/repository/interface.go` defines interfaces for User, Customer, Job, Quote, Tenant, Permission, Membership, Payment and Schedule.
- **The heaviest services bypass repositories and take `*gorm.DB` directly** (`routes/services.go`):
  - Invoice (936 lines), Revenue, CustomerReport, ARAging, UnifiedExport, PaymentTerms, Invitation, PaymentReminder, Import
  - Job, which takes both a repo and the db
- Repository methods take **no `context.Context`**, for example `CustomerRepositoryInterface.GetByID(id uuid.UUID)`. So request cancellation and timeouts don't propagate.
- There is no transaction abstraction: `Transaction(func(tx *gorm.DB))` appears in 4 places.

### Effort estimate (one engineer familiar with the code)
- **Gin → chi:** about 4-6 days. Handlers use `c.ShouldBindJSON`, `c.Param`, `c.Query`, `c.JSON`, `c.Get/Set` (auth context) and `gin.H`. Write a tiny adapter (`bind`, `param`, `respondJSON`, a typed context-key helper), then convert mechanically. Middleware (13 files) needs `func(http.Handler) http.Handler`. Swag annotations carry over unchanged. About 30% less work if you first delete the dead handlers and middleware from section 5.
- **GORM → pgx (+ sqlc recommended):** about 3-5 weeks. This is the bulk of the work:
  - 111 Preloads to replace with explicit joins or batch loads
  - soft-delete scoping (`gorm.DeletedAt`) to reproduce by hand in every query
  - hooks to move into services
  - `decimal.Decimal` support (pgx works with shopspring via `pgx-shopspring-decimal`)
  - JSONB `Value`/`Scan` already implemented on `TenantSettings`
  - tests currently use sqlite + AutoMigrate; switch them to testcontainers Postgres
- **golang-migrate:** about 1 day.
  - Baseline `001_schema.sql` as `000001_init.up.sql`, after fixing the drift in section 7.
  - Split `002_seed_dev.sql` out as a dev-only seed.
  - Run migrations as a separate Cloud Run Job, or a `migrate` sub-command, before deploying. Don't run them in the server boot path: concurrent instances would race on the advisory lock and slow cold start.

### Suggested sequencing
1. **Now, before Cloud Run:**
   - fix the deploy blockers (1.1-1.5) and Sentry linkage (3.1)
   - fix the critical money bugs (7.C1-C4, 7.H1)
   - add `TenantID` (7.C5)
   - delete the dead code (5.5)
   - adopt golang-migrate with the corrected schema as the baseline. Every later fix then ships as an incremental migration, which CLAUDE.md anticipates for production.
2. **Introduce the boundary while still on GORM:**
   - add `ctx context.Context` and `tenantID` to every repository method;
   - move the `*gorm.DB`-taking services (Invoice, Payments, Reports, Exports, Reminders, Invitations) behind repository interfaces;
   - add a `TxManager` interface (`WithinTx(ctx, func(ctx) error)`) that carries the tx in the context.
   This step is valuable on its own: it delivers the tenant filtering and the transactions needed by C1, H3 and 4.2.
3. **Gin → chi:** independent of the DB layer and low risk. Do it after step 1's deletions.
4. **GORM → pgx/sqlc one repository at a time**, behind the interfaces from step 2:
   - start with the simple ones (Tenant, Permission, Membership);
   - then Customer and Job;
   - then Invoice, Quote and the reports, where the Preload-heavy code is.
   - Move hook logic (job numbers and so on) into services or DB sequences first.
5. Drop GORM, sqlite and the remaining `gorm:` tags, then switch the test suite to Postgres via testcontainers.
