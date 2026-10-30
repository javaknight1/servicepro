# CLAUDE.md — ServicePro

Read this every session before writing code. `TODO.md` is the *what to do, in order* (indexed tasks,
phases, launch gate, decisions). This file pins the rules an agent will otherwise quietly violate —
especially tenant isolation and the money path.

## What this is

Multi-tenant field-service management SaaS: CRM, jobs/work orders, scheduling + dispatch, quotes,
invoices, payments, collections, reports. A **tenant** is a service business (ServicePro's customer);
its **customers** are the people/companies it does work for.

- **Positioning** (TODO.md "Product framing", `docs/product/mvp-plan-2026-10.md`): the field service
  software that gets every site's work invoiced and paid — for commercial cleaning and facility-maintenance
  companies (1–10 techs) billing property managers on net terms (D7). Flat price, unlimited users, no
  contract; Solo $39 / Team $99 / Pro $199, cards Stripe + 0.4%, ACH 1% capped at $10 (D10).
  "ServicePro" is a working name (OD-10).
- **Status (2026-10-25): v0.9.0 live since 2026-10-24** at `servicepro.sharkfins.xyz` (invite-only).
  Phase 1 security is done. Work follows the **MVP Launch Gate** in TODO.md (phases A–G, paid launch
  2027-09-07); the honesty gate (D11: T255, T256, T261) is due 2026-11-08. Database: **Neon** Postgres
  us-east-2 (T202). Auth: **Neon's managed Better Auth** (OD-9) — backend done, SPA still on own JWTs (T158).

## Stack

| Layer    | Today                                                              | Target (TODO Phase 4, League Looper parity; paced by D12)         |
| -------- | ------------------------------------------------------------------ | ----------------------------------------------------------------- |
| API      | Go 1.26, Gin, GORM, Postgres 18, Redis (optional)                  | chi, pgx/sqlc, Postgres RLS backstop, Upstash (if Redis needed)   |
| Auth     | Own JWT (httpOnly cookies) + bcrypt                                | Neon's managed Better Auth, JWKS-verified in the API (T158, OD-9) |
| Web      | React 19, Vite 8, TS strict, Zustand + React Query, Tailwind v4    | React 19, TanStack Query for all server state, Tailwind v4 tokens |
| Email    | Resend / SES / SMTP (Mailpit locally)                              | Resend only                                                       |
| Storage  | S3-compatible (MinIO locally)                                      | Cloudflare R2                                                     |
| Payments | Stripe (platform account; pay links off until T169 — T261)         | Connect Accounts v2, direct charges (D9, T169)                    |
| Obs.     | Sentry, PostHog, Prometheus `/metrics`, Grafana (local only)       | + Better Stack uptime/status                                      |
| Hosting  | Cloudflare Worker (static assets + `/api` proxy) → Cloud Run us-east5 | same                                                            |
| Database | Neon Postgres 18 us-east-2 (Free plan); Docker Postgres 18 locally | Neon Launch plan + nightly R2 backups before paying tenants (D14) |

Decision record: TODO.md "Architecture & product decisions" (D1–D14). **Stay Go API + React SPA — not Next.js**
(D1). Migrate incrementally; the app must keep working at every step. **D12: the chi, pgx/sqlc, GORM-removal,
repository-refactor and TanStack Query ports (T153, T155–T157, T162) are frozen until the month-6 checkpoint** —
don't start them; Neon Auth, Resend, R2 and RLS stay in the MVP. Don't introduce a new vendor or framework without
recording a decision there first.

## Hard rules (the reason this file exists)

1. **Every tenant-owned query is tenant-scoped.** Tenant comes only from the authenticated membership
   context, never from a client-supplied ID alone. Reads by ID filter `tenant_id` too (cross-tenant →
   404). New tenant tables get `tenant_id NOT NULL` + a `(tenant_id, …)` index. Add a cross-tenant test
   for every new repository. (Phase 4 adds RLS as a backstop — it is not a substitute.)
2. **Authorization is per tenant.** Permission checks take the tenant; a role in tenant A grants
   nothing in tenant B. No endpoint may let a caller raise their own role; `super_admin` is never
   assignable through the API.
3. **Money:** amounts are `shopspring/decimal` today (never `float64`); totals are computed
   server-side; payment state changes happen in one transaction. Integer cents is under evaluation
   (T166) — don't mix representations in new code.
4. **Stripe:** payment state is confirmed only from a signature-verified, deduplicated webhook — never
   from a client redirect. Every Stripe write carries an idempotency key. Never accept a raw Stripe ID
   from the client; look up our own tenant-scoped row and use its stored ID.
5. **Background work must be idempotent.** Jobs run as `POST /internal/jobs/<name>` (token-protected)
   triggered by `.github/workflows/scheduled-jobs.yml`; in-process tickers are dev-only
   (`RUN_BACKGROUND_WORKERS`). Claim before acting (e.g. `INSERT … ON CONFLICT DO NOTHING`).
6. **Secrets never in the repo** (the GitHub repo is public). `.env` files are gitignored; never print
   secret values — key names only.

## Project structure

```
servicepro/
├── CLAUDE.md · TODO.md · RELEASING.md · VERSION · Makefile · docker-compose.yml
├── backend/                     # Go API (module github.com/javaknight1/servicepro/backend)
│   ├── cmd/main.go              # server entrypoint
│   ├── cmd/migrate/             # golang-migrate CLI (up/down/version/force)
│   ├── config/config.go         # env-based config
│   ├── migrations/              # 00000N_name.{up,down}.sql, embedded (embed.go)
│   ├── seeds/dev_seed.sql       # dev-only seed data (never a migration)
│   ├── internal/models/         # GORM models
│   ├── internal/repository/     # data access
│   ├── internal/services/       # business logic
│   ├── internal/api/handlers/   # HTTP handlers (swag annotations)
│   ├── internal/api/routes/     # route wiring
│   ├── internal/api/middleware/ # auth, permissions, edge client IP, rate limits
│   ├── internal/permissions/    # permission constants
│   ├── pkg/                     # clients (email, storage, sms, logging, …), database
│   └── docs/                    # GENERATED swagger (committed)
├── frontend/                    # React SPA
│   ├── src/{types,services,store,components,pages,routes}/
│   ├── worker/                  # Cloudflare Worker: /api/* + /version proxy, assets
│   ├── wrangler.jsonc           # Worker servicepro-web, servicepro.sharkfins.xyz (live)
│   └── public/_headers          # CSP/HSTS/cache headers for the static assets
├── docs/                        # ops/, audits/, product/, architecture/, api/, …
├── scripts/                     # bump-version, gh-release, deploy-cloudrun, git-hooks/
└── .github/workflows/           # ci, release-deploy, migrate, db-backup, scheduled-jobs, …
```

## Database migrations — ALWAYS ADDITIVE

golang-migrate, files in `backend/migrations/`, embedded into the binary.

- **Never edit a migration that exists on master.** Every schema change is a new pair:
  `make migrate-create NAME=add_foo` → `00000N_add_foo.up.sql` + `.down.sql`. Use `IF [NOT] EXISTS`.
  (`000001_initial_schema` is the whole pre-launch schema: 000001–000014 were squashed into it on
  2026-10-24, owner's call, before any shared database existed. That was a one-off — once Neon holds
  real data, the rule above is absolute. A local DB from before the squash: `make db-fresh-seed`.)
- Schema change and the code that needs it ship in the same commit. Update `backend/seeds/dev_seed.sql`
  when you add NOT NULL columns so `make seed` keeps working. Seed `INSERT`s use `ON CONFLICT`.
- Local: the server self-migrates (`RUN_MIGRATIONS_ON_STARTUP`, default on in development);
  `make migrate`, `make migrate-version`, `make db-fresh-seed`. Production: the release pipeline's
  `migrate` job owns migrations; the server never migrates at startup. `/health/ready` reports schema
  state and returns 503 when the DB is behind or dirty.

## Coding conventions

### Backend

- GORM today (pgx/sqlc is the target — keep new repository methods behind interfaces that take
  `ctx` and `tenantID`, so the port is mechanical). UUID primary keys, `timestamptz`.
- Errors: return typed domain errors from `internal/errors` (status + snake_case code + safe message
  + optional field errors) and write every error response through `internal/api/apierror`
  (`Respond`, `BindError`) — one envelope `{error, message, request_id, fields?}` (see
  `docs/api/errors.md`). Conflicts are 409, not-found/cross-tenant 404, validation 400; anything
  untyped becomes a logged generic 500. Never return raw `err.Error()` to clients.
- Every request carries an `X-Request-ID` (in logs, responses and Sentry); log through the
  context-aware logger so lines carry it.
- Logging: structured logger with a contextual prefix (e.g. `[INVOICE-SERVICE]`); no PII or tokens.
- Client IP: always `middleware.ClientIP(c)` (trusts `X-SP-Client-IP` only with a valid
  `X-SP-Edge-Secret`) — never read `X-Forwarded-For` directly.

### Frontend

- Zustand for client state; React Query for server state (TanStack Query everywhere is the target).
- API calls only through the axios services (`frontend/src/services/api.ts`); same-origin `/api`.
- Follow `frontend/src/components/shared/` patterns. TypeScript strict.
- **UI follows the MVP mockups** in `docs/product/mockups/` (index in its README: which screen guides
  which task). Match layout, copy tone and colours; sample numbers are not targets.
- **Never use `any`.** Use `unknown` + `getErrorMessage()` (`frontend/src/utils/error.ts`) in catch
  blocks; typed interfaces for third-party callbacks (see `frontend/src/components/calendar/types.ts`);
  proper mock interfaces in tests; if you must cast an incomplete mock, `as unknown as ProperType`.
- Never send tokens/emails/query strings to analytics (PostHog/Sentry) — strip them.

## Permissions

When a feature needs a new permission:

1. Add the constant to `backend/internal/permissions/constants.go`
2. Update `AllPermissions()`
3. Update `DefaultRolePermissions()` for the right roles
4. Update `PermissionsByResource()`
5. Grant it to roles in a **new migration** (and the dev seed if needed)

## Environment variables

When adding a backend env var, update all three in the same change:

1. `backend/.env.example` — descriptive comment; placeholders for secrets (`sk_test_xxxx`); real
   defaults for non-sensitive config.
2. `backend/.env` (gitignored, local) — secrets get `# TODO: Get from …`; non-sensitive defaults
   match `.env.example`.
3. `backend/config/config.go` — struct field + `getEnv()` with a sensible default.

Frontend vars live in `frontend/.env.example` (`VITE_*`, baked in at build time). Worker vars/secrets
live in `frontend/wrangler.jsonc` / `wrangler secret put` (`API_ORIGIN`, `EDGE_PROXY_SECRET`).

## Makefile

**One root `Makefile`; never add Makefiles in subdirectories.** New tasks follow the `backend-*` /
`frontend-*` naming. `make help` lists everything. Common: `make setup`, `make dev`, `make up`,
`make migrate`, `make seed`, `make test`, `make lint`, `make ci-local`, `make docs`, `make bump`.

## API documentation (OpenAPI)

Swag annotations on every handler (`@Summary @Description @Tags @Accept @Produce @Security @Param
@Success @Failure @Router`). Add/modify/remove them with the endpoint. After any annotation change run
**`make docs`** and commit `backend/docs/` + `frontend/src/types/api.generated.ts` — they are committed
on purpose; the pre-commit hook and CI's `openapi-drift` job fail on drift. swag is pinned by the
`tool` directive in `backend/go.mod` (`go tool swag`).

## Testing

- Backend: `make backend-test` (race, integration skipped). Integration: `make test-integration`.
- Frontend: `make frontend-test` (vitest). Integration tests in `frontend/tests/integration/`
  (`auth`, `customer`, `job`, `quote`, `invoice`) — **update them when you change those workflows**
  (new page/component in a flow, API service method changes → mocks, validation, routing, new status
  transitions). Utilities in `frontend/tests/integration/setup.ts` (`renderWithProviders()`, mock
  stores, `createMockApiResponse()`, entity factories).
- Every security fix gets a regression test; every new repository gets a cross-tenant test.
- `make ci-local` runs the core CI gates before you push.

## Git hooks & commit message contract

Install once: `make install-hooks` (symlinks `scripts/git-hooks/*`). pre-commit runs golangci-lint,
Go tests/build and the OpenAPI drift check on staged backend changes, eslint + tsc on staged frontend
changes. commit-msg enforces the prefix list. Don't bypass with `--no-verify`.

**Commits are the changelog** (`scripts/gh-release.sh` builds release notes from subjects;
`scripts/bump-version.sh` reads them to pick the version):

| Prefix                                                              | In release notes? | Bump    |
| ------------------------------------------------------------------- | ----------------- | ------- |
| `feat:`                                                             | yes               | MINOR   |
| `fix:`                                                              | yes               | PATCH   |
| `<type>!:` / `BREAKING CHANGE`                                      | yes               | MINOR pre-1.0, MAJOR after |
| `deps:` / `deps(go):` / `deps(npm):`                                | no                | PATCH   |
| `chore: docs: ci: build: test: style: refactor: perf: release:`     | no                | none    |

Legacy `minor:`/`major:` subjects (≤ v0.7.0) are still understood by the bump script; don't use them.
Format: `type: short description (T0XX)`. One logical change per commit; commit after every task.

## Commit size, worktrees & pushing (standard since 2026-10-16)

- **Small items, one commit each.** Split work into the smallest item that still leaves the app
  working (one bug, one endpoint, one migration + its code, one test fix). Finish the item, then
  commit it as a single commit, so each commit is one complete, passing item. Don't batch several
  items into one commit or leave an item spread across several "WIP" commits. A large TODO task is
  several commits that share its ID (`fix: … (T118)`, `test: … (T118)`).
- **Parallel agents work in git worktrees.** When more than one agent (or session) changes code at
  the same time, each one works in its own worktree and branch (`isolation: "worktree"`, or
  `git worktree add`), never in the main checkout. Give parallel agents disjoint areas and, for
  schema changes, disjoint migration-number ranges. Agents commit on their branch and never push
  `master`; the coordinator merges.
- **Push often.** The coordinator merges finished commits into `master` as they land (not only when
  a whole batch is done), runs the gates, and pushes `master` right away. Nothing finished should
  sit unpushed for long. Remove a worktree and its branch once it is merged.

## Releases & hosting

Tag-driven: `make bump` → commit `VERSION` → `git tag vX.Y.Z && git push origin vX.Y.Z`. Full process
in `RELEASING.md`; one-time setup in `docs/ops/cloud-run-setup.md`; rollback in `docs/ops/deploy.md`;
backups in `docs/ops/backups.md`. `release-deploy.yml` skips every deploy job until the repo variable
`GCP_PROJECT_ID` exists; it does, so a `v*` tag migrates Neon, deploys the API and the Worker, and publishes
release notes.

Production (live since 2026-10-24): `servicepro.sharkfins.xyz` → Worker `servicepro-web` (static assets,
SPA fallback) → `/api/*` and `/version` proxied to Cloud Run `servicepro-api` (us-east5) with
`X-SP-Client-IP` + `X-SP-Edge-Secret` → Neon Postgres (us-east-2). The API rejects non-edge traffic in
production. Cloud Run instances × DB pool must stay under the DB's connection limit.

## TODO task management

`TODO.md` uses indexed tasks (T001…). It is the in-repo source of truth; the Notion **Tasks** database
(app = ServicePro, under Apps → All Apps → ServicePro) mirrors it for tracking.

When implementing a task:

1. Find it in the Task Index, then its `#### T### — …` section under its phase.
2. When done, update **all three places** in TODO.md: strike through every cell of its Task Index
   row (`| ~~T042~~ | ~~5~~ | ~~Feature~~ | ~~4~~ | ~~Title~~ |`), check it off in the phase section,
   and add a line to **Verified Complete** (ID + what was done + where). Update "Last Updated".
3. Commit immediately (`type: description (T0XX)`).
4. Mirror the status in Notion (Status = Done, Completed date, Commit Message) — if the Notion tools
   aren't available in the session, say so in your summary so it can be synced later.

When adding a task: next free ID; a Task Index row (completed rows first by ID, then open by
Phase → ID; keep the columns padded — see the snippet at the top of TODO.md's Task Index or re-pad
with a script); a full section using the template below; cross-references both ways; "Last Updated";
and a matching Notion task.

```markdown
#### T### — Task title

**Phase**: N · **Category**: X · **Hours**: N · **Status**: Open · **MVP Gate**: Yes/No · **Depends on**: T### (or "none")

**What it is** — the deliverable.
**What it does** — implementation gist.
**Why we need it** — what breaks if skipped.
**Notes** — audit refs, decisions, supersession.
```

Deferred / Won't Do tasks get an abbreviated section (status + why + revisit condition). Use
`/analysis T###` (`.claude/commands/analysis.md`) for a deep pre-implementation analysis.

## Fleet & blog

This app is tracked in Fleet as `SP`; run `/fleet-sync` after any change to hosting, vendors, plans,
auth, data providers, domains or monthly costs. `servicepro.sharkfins.xyz` went live 2026-10-24; T211 adds
it to the blog (`src/pages/apps.astro`, `INFRA.md` in `~/repositories/blogs/rob-blog`; commit there,
ask before pushing).
