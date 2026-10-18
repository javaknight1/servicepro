# ServicePro — Roadmap

**Last Updated**: 2026-10-18 (TODO reformatted from the P0–P3/sprint layout to League Looper's indexed phase format: Task Index with ID/Phase/Category/Hours, one detailed section per open task, a v1 Launch Gate and Verified Complete. A full audit of the repo (backend security, backend correctness, frontend, infra — stored in `docs/audits/2026-10-16/`) is filed as Phases 1–3 (T102–T152, T223–T226). The stack-convergence plan towards League Looper's stack is Phase 4 (T153–T166). Competitive research (`docs/product/competitive-research-2026-10.md`) became product Phases 5–16, merging the open T027–T101 tasks with new T167–T222. Hosting is deferred: ServicePro stays local-only until the production database decision (T202). Old IDs T001–T101 are kept; T011 never existed. Same day: Phase 3 platform work T139–T152 landed — golang-migrate, /version + schema-aware readiness, internal job endpoints, optional Redis, Cloud Run image, LL-style CI/release/hooks, Cloudflare Worker edge (not deployed); frontend fixes T130–T138; all of Phase 1 security (T102–T117) — tenant isolation plugin, per-tenant authz, invite-only registration, token/login/webhook/CSRF hardening, no known Go vulns; T227 added for the frontend follow-through. Same day: Phase 2 correctness work landed (T118–T129, T223–T239 — money path, numbering, status machines, refunds, quotes, tenant settings, error envelope + request IDs, ~39k lines of dead backend code removed, stricter golangci-lint); follow-ups T241–T244 filed. 2026-10-18: T231 (org deletion) deferred; Dependabot security-fix PRs enabled; SQLite ruled out for OD-1; Go deps bumped, T247 filed; T202 decided — Neon; OD-9 (auth provider) opened; T164 frontend majors done (React 19, Router 7, Vite 8, ESLint 10, zod 4, Headless UI 2), T248/T249 filed; Tailwind 4 engine landed (T163 tokens + dark mode remain); 0 open Dependabot alerts.)

> **How to read this file.** Every task has an ID (T###). The **Task Index** lists all of them: completed rows
> (struck through) first by ID, then open rows by Phase → ID. Each open task has exactly one detailed section under
> its phase in the **Phase Roadmap** (template: What it is / What it does / Why we need it / Notes). **Phase 0** means
> work finished before the 2026-10-16 reformat; its history is in **Verified Complete**. The **v1 Launch Gate** is the
> subset that must be done before real customers. Hours are rough estimates. Audit references like "security audit
> F-3" or "correctness audit 7.C1" point into `docs/audits/2026-10-16/`. To work a task, see "TODO Task Management"
> in CLAUDE.md.

## Product framing

- **Positioning**: for small commercial service contractors (1–10 techs) who bill businesses and property managers
  on terms, ServicePro is the field service platform that gets you paid — scheduling, work orders, multi-site
  customers and automated collections at one flat monthly price, unlimited users, no contract, and you can leave with
  all your data any time.
- **Beachhead**: small commercial/B2B service contractors (1–10 techs) billing on terms — commercial cleaning,
  property maintenance/handyman for property managers, small commercial HVAC. Secondary hypothesis: inspection and
  compliance trades (backflow, fire, hood cleaning) — validate by interviews (T218, T222).
- **Pricing hypothesis**: Solo $39 / Team $99 (unlimited users) / Pro $199, month-to-month; +0.4% platform fee on
  Stripe Connect volume; founding offer: Team at $59/mo for life for the first 25 tenants (T210).
- **Moat**: lock-in through money flow (Connect, saved payment methods) plus recurring schedules in an under-served
  B2B niche, reinforced by a no-contract, easy-exit trust brand that per-seat incumbents can't copy without hurting
  their revenue. AI is not the moat.
- Full research: `docs/product/competitive-research-2026-10.md`.

## Status at a glance

- **Local-only.** Not hosted anywhere; the old Fly.io/Pages workflows were replaced (Phase 3, done 2026-10-16). The
  database host is decided (Neon, T202, 2026-10-18); provisioning is Phase 14 (T203+).
- **Backend has critical tenant-isolation and auth-escalation flaws** (security audit F-1–F-4). Phase 1 blocks any
  hosting, even invite-only.
- **Money flows are partially broken** (recording a payment doesn't update the invoice, reminders have never sent,
  Checkout charges the wrong amount, invoice numbering wedges) — Phase 2.
- The back office and A/R side is strong for a pre-launch product; the field side, customer-facing side and
  integrations are thin (Phases 5–11).

## Architecture decisions (recorded 2026-10-16)

1. **D1 — Stay Go API + React (Vite) SPA; not Next.js.** The 95K-line React app is healthy; field service is job-
   and worker-heavy (reminders, recurring jobs, exports, webhooks), which suits a long-running Go service better than
   serverless; a technician mobile app needs a standalone API anyway; the broken part is the data layer, which Phase 4
   rebuilds.
2. **D2 — Converge on League Looper's stack incrementally (strangler; the app keeps working).** chi; pgx/sqlc;
   golang-migrate with always-additive migrations; Supabase Auth with our own `users` table keyed by `auth_user_id`;
   Postgres RLS backstop on `tenant_id`; Resend; Cloudflare R2; Upstash (optional Redis); Sentry; PostHog;
   Better Stack.
3. **D3 — Hosting target.** Frontend on Cloudflare Workers static assets with an edge Worker proxying `/api/*` and
   `/version` to Cloud Run at `servicepro.sharkfins.xyz`; API on Cloud Run; database **Neon Postgres** (owner decision
   2026-10-18, T202; free tier to start). Cloud Run runs in the Neon project's region (OD-8). Still local-only until
   Phase 14 provisions it.
4. **D4 — Commit convention = League Looper's.** `feat:`/`fix:` are user-facing; `deps`, `chore`, `docs`, `ci`,
   `build`, `test`, `style`, `refactor`, `perf`, `release` are filtered from release notes. Commits are the changelog;
   releases are tag-driven from `VERSION` (T146, T148). The historic `minor:`/`major:` prefixes are retired.
5. **D5 — Tenancy model.** One database, shared schema, `tenant_id` on every tenant-owned row, enforced in
   repositories (T102, T153) **and** by RLS (T154).
6. **D6 — Money as decimal (shopspring) today.** Evaluate integer cents during the pgx port (the Travelify rule) —
   open question OD-5 / T166.

---

## v1 Launch Gate

The minimum subset that must be done before real (paying) customers: **39 tasks, ~320h**. Everything else is post-launch growth or depth. The invite-gated preview (T205) needs only Phase 1 plus T106/T204/T206; the gate below is for opening to paying customers. Phase 4's gating subset is the parts of the stack that touch auth, tenancy, email and storage; the chi and pgx ports are not gating.

| Phase                                                | Gating tasks                                                                                                                                                                                                       |
|------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Phase 1 — Security Remediation                       | **All** — ~~T102~~, ~~T103~~, ~~T104~~, ~~T105~~, ~~T106~~, ~~T107~~, ~~T108~~, ~~T109~~, ~~T110~~, ~~T111~~, ~~T112~~, ~~T113~~, ~~T114~~, ~~T115~~, ~~T116~~, ~~T117~~                                           |
| Phase 2 — Correctness & Cleanup                      | ~~T118~~, ~~T119~~, ~~T120~~, ~~T121~~, ~~T122~~, ~~T123~~, ~~T124~~, ~~T126~~, ~~T130~~, ~~T131~~, ~~T132~~, ~~T133~~, ~~T136~~, ~~T224~~, ~~T225~~, ~~T227~~, ~~T232~~, ~~T234~~, ~~T238~~, ~~T239~~, T241, T243 |
| Phase 3 — Platform & Delivery                        | ~~T139~~, ~~T140~~, ~~T141~~, ~~T142~~, ~~T143~~, ~~T144~~, ~~T145~~, ~~T146~~, ~~T150~~, ~~T151~~, ~~T245~~                                                                                                       |
| Phase 4 — Stack Convergence (LL parity; incremental) | T153, T154, T158, T159, T160                                                                                                                                                                                       |
| Phase 5 — Core Workflow & Quoting                    | T028, T029, T030, T035, T167                                                                                                                                                                                       |
| Phase 6 — Payments to the Business                   | T169, T170, T061, T063                                                                                                                                                                                             |
| Phase 7 — Dispatch & Field                           | T173, T174, T176                                                                                                                                                                                                   |
| Phase 8 — Customer Communications                    | T048, T049, T050                                                                                                                                                                                                   |
| Phase 9 — Customer Self-Service                      | T183, T081                                                                                                                                                                                                         |
| Phase 10 — Commercial/B2B Wedge                      | T186                                                                                                                                                                                                               |
| Phase 11 — Integrations & Migration                  | T192, T193                                                                                                                                                                                                         |
| Phase 14 — Hosting & Launch Readiness                | ~~T202~~, T203, T204, T205, T206, T207, T208, T209, T210, T211, T212, T213                                                                                                                                         |
| Phase 16 — Go-to-Market                              | T218                                                                                                                                                                                                               |

**Explicitly NOT in the launch gate** (ship without them):

- Phase 2 remainder (T125, T127–T129, T134, T135, T137, T138, T223, T226) — fix soon after launch
- Phase 3 remainder (T101, T147–T149, T152) — delivery polish
- Phase 4 ports: Gin → chi (T155), GORM → pgx/sqlc (T156), GORM/sqlite removal (T157), production Redis (T161 — T142 makes Redis optional, which is gated), frontend modernisation (T162–T164), T165, T166
- Phase 5 remainder (T033, T041, T042, T051, T052, T058, T060, T062, T076, T080, T082, T168)
- Phase 6: saved methods/autopay (T171), Tap to Pay (T172), late fees (T083)
- Phase 7 remainder (checklists, time clock, map, offline, store apps, scheduling extras)
- Phase 8 remainder (templates, delivery tracking, CRM log/timeline, two-way SMS, reviews)
- Phase 9: customer portal (T184), booking widget (T185)
- Phase 10 remainder (POs, consolidated invoicing, statements, PM approvals, agreements autopay, collections suite)
- Phase 11 remainder (Zapier, Wisetack, Gusto, AI receptionist, webhooks, public API)
- Phases 12, 13 and 15 (reporting, roles/compliance/trust, AI)
- Phase 16 remainder (T219–T222) — T218 interviews are gated, the marketing build-out is not

---

## Task Index

| ID       | Phase  | Category       | Hours  | Task                                                                                                  |
|----------|--------|----------------|--------|-------------------------------------------------------------------------------------------------------|
| ~~T001~~ | ~~0~~  | ~~Backend~~    | ~~—~~  | ~~Recovery middleware Sentry integration~~                                                            |
| ~~T002~~ | ~~0~~  | ~~Frontend~~   | ~~—~~  | ~~Centralize direct fetch() calls~~                                                                   |
| ~~T003~~ | ~~0~~  | ~~Frontend~~   | ~~—~~  | ~~Add CSP/HSTS headers to nginx.conf~~                                                                |
| ~~T004~~ | ~~0~~  | ~~CI~~         | ~~—~~  | ~~Create GitHub Actions CI/CD workflows~~                                                             |
| ~~T005~~ | ~~0~~  | ~~Monitoring~~ | ~~—~~  | ~~Expose /metrics endpoint for Prometheus~~                                                           |
| ~~T006~~ | ~~0~~  | ~~Backend~~    | ~~—~~  | ~~Structured JSON logging (replace fmt.Printf)~~                                                      |
| ~~T007~~ | ~~0~~  | ~~Frontend~~   | ~~—~~  | ~~Fix `any` types (50 instances)~~                                                                    |
| ~~T008~~ | ~~0~~  | ~~Frontend~~   | ~~—~~  | ~~Remove @ts-nocheck directives~~                                                                     |
| ~~T009~~ | ~~0~~  | ~~Frontend~~   | ~~—~~  | ~~Enable noUnusedLocals/noUnusedParameters~~                                                          |
| ~~T010~~ | ~~0~~  | ~~Backend~~    | ~~—~~  | ~~Apply error tracking middleware~~                                                                   |
| ~~T012~~ | ~~0~~  | ~~CI~~         | ~~—~~  | ~~Bundle size monitoring in CI~~                                                                      |
| ~~T013~~ | ~~0~~  | ~~Testing~~    | ~~—~~  | ~~Increase frontend test coverage to 70%+~~                                                           |
| ~~T014~~ | ~~0~~  | ~~Testing~~    | ~~—~~  | ~~Integration tests for critical workflows~~                                                          |
| ~~T015~~ | ~~0~~  | ~~Docs~~       | ~~—~~  | ~~Generate OpenAPI/Swagger documentation~~                                                            |
| ~~T016~~ | ~~0~~  | ~~Perf~~       | ~~—~~  | ~~Frontend bundle analysis + optimization~~                                                           |
| ~~T017~~ | ~~0~~  | ~~Perf~~       | ~~—~~  | ~~Query performance monitoring~~                                                                      |
| ~~T018~~ | ~~0~~  | ~~Cleanup~~    | ~~—~~  | ~~Dead code elimination~~                                                                             |
| ~~T019~~ | ~~0~~  | ~~Feature~~    | ~~—~~  | ~~Integrate calendar view for job scheduling~~                                                        |
| ~~T020~~ | ~~0~~  | ~~Feature~~    | ~~—~~  | ~~Build conflict detection API endpoint~~                                                             |
| ~~T021~~ | ~~0~~  | ~~Cleanup~~    | ~~—~~  | ~~Extract duplicate file download utility~~                                                           |
| ~~T022~~ | ~~0~~  | ~~Cleanup~~    | ~~—~~  | ~~Extract duplicate URLSearchParams builder~~                                                         |
| ~~T023~~ | ~~0~~  | ~~Cleanup~~    | ~~—~~  | ~~Consolidate cache hooks (useLocalCache/useSession)~~                                                |
| ~~T024~~ | ~~0~~  | ~~Monitoring~~ | ~~—~~  | ~~Full Sentry integration (frontend + backend)~~                                                      |
| ~~T025~~ | ~~0~~  | ~~Monitoring~~ | ~~—~~  | ~~Integrate product analytics (PostHog)~~                                                             |
| ~~T026~~ | ~~0~~  | ~~Reporting~~  | ~~—~~  | ~~Set up business KPI dashboard (Grafana)~~                                                           |
| ~~T031~~ | ~~0~~  | ~~Reporting~~  | ~~—~~  | ~~Build A/R aging buckets report~~                                                                    |
| ~~T032~~ | ~~0~~  | ~~Reporting~~  | ~~—~~  | ~~Build "Who Owes What" report~~                                                                      |
| ~~T034~~ | ~~0~~  | ~~Email~~      | ~~—~~  | ~~Wire payment reminder automation~~                                                                  |
| ~~T043~~ | ~~0~~  | ~~Email~~      | ~~—~~  | ~~Wire payment reminder notifications~~                                                               |
| ~~T053~~ | ~~0~~  | ~~Feature~~    | ~~—~~  | ~~Add "Repeat Job" button on customer detail~~                                                        |
| ~~T100~~ | ~~0~~  | ~~Feature~~    | ~~—~~  | ~~Add business hours & timezone to org settings~~                                                     |
| ~~T102~~ | ~~1~~  | ~~Security~~   | ~~16~~ | ~~Tenant-scope all business data~~                                                                    |
| ~~T103~~ | ~~1~~  | ~~Security~~   | ~~6~~  | ~~Authorize tenant member, invitation and role endpoints; no self-escalation~~                        |
| ~~T104~~ | ~~1~~  | ~~Security~~   | ~~4~~  | ~~Evaluate permissions per tenant~~                                                                   |
| ~~T105~~ | ~~1~~  | ~~Security~~   | ~~3~~  | ~~Remove or lock the raw Stripe passthrough endpoints~~                                               |
| ~~T106~~ | ~~1~~  | ~~Security~~   | ~~5~~  | ~~Backend-enforced invite-only registration; reject non-edge traffic in production~~                  |
| ~~T107~~ | ~~1~~  | ~~Security~~   | ~~2~~  | ~~Edge client-IP middleware and a safe trusted-proxy default~~                                        |
| ~~T108~~ | ~~1~~  | ~~Security~~   | ~~2~~  | ~~Escape PDF export HTML; drop wkhtmltopdf local file access~~                                        |
| ~~T109~~ | ~~1~~  | ~~Security~~   | ~~5~~  | ~~Stripe webhook reliability: persistent dedupe, accept retries, body limit, idempotent payment~~     |
| ~~T110~~ | ~~1~~  | ~~Security~~   | ~~4~~  | ~~Token hardening: access vs refresh type, revoke on logout and password reset~~                      |
| ~~T111~~ | ~~1~~  | ~~Security~~   | ~~3~~  | ~~Login hardening: no lockout DoS, uniform errors and timing~~                                        |
| ~~T112~~ | ~~1~~  | ~~Security~~   | ~~1~~  | ~~Neutralise CSV/Excel formula injection in exports~~                                                 |
| ~~T113~~ | ~~1~~  | ~~Security~~   | ~~1~~  | ~~Fail closed when no email or storage provider in production; never log tokens~~                     |
| ~~T114~~ | ~~1~~  | ~~Security~~   | ~~2~~  | ~~Lock down /health detail, /metrics, Swagger and debug mode in production~~                          |
| ~~T115~~ | ~~1~~  | ~~Security~~   | ~~3~~  | ~~Secure cookies by default and CSRF protection~~                                                     |
| ~~T116~~ | ~~1~~  | ~~Security~~   | ~~3~~  | ~~Low-severity security sweep~~                                                                       |
| ~~T117~~ | ~~1~~  | ~~Security~~   | ~~3~~  | ~~Remediate dependency vulnerabilities and bump the Go toolchain~~                                    |
| ~~T118~~ | ~~2~~  | ~~Bug~~        | ~~3~~  | ~~Recording a payment updates the invoice balance and status~~                                        |
| ~~T119~~ | ~~2~~  | ~~Payments~~   | ~~3~~  | ~~Stripe Checkout charges the true balance due~~                                                      |
| ~~T120~~ | ~~2~~  | ~~Bug~~        | ~~2~~  | ~~Invoice and job numbering survives soft-delete and concurrency~~                                    |
| ~~T121~~ | ~~2~~  | ~~Bug~~        | ~~2~~  | ~~Payment reminders actually send~~                                                                   |
| ~~T122~~ | ~~2~~  | ~~Bug~~        | ~~2~~  | ~~Fix /webhooks/stripe panicking on subscription and invoice events~~                                 |
| ~~T123~~ | ~~2~~  | ~~Bug~~        | ~~3~~  | ~~Add the missing SQL functions and views behind customer search and reports~~                        |
| ~~T124~~ | ~~2~~  | ~~Bug~~        | ~~4~~  | ~~Enforce invoice status transitions; unify the two job state machines~~                              |
| ~~T125~~ | ~~2~~  | ~~Payments~~   | ~~3~~  | ~~Handle refunds in the ledger~~                                                                      |
| ~~T126~~ | ~~2~~  | ~~Monitoring~~ | ~~2~~  | ~~Backend Sentry actually initialised; wire or delete the Loki/BetterStack/OTel stubs~~               |
| ~~T127~~ | ~~2~~  | ~~Backend~~    | ~~3~~  | ~~Backend reads tenant settings: currency, number prefixes, defaults~~                                |
| ~~T128~~ | ~~2~~  | ~~Backend~~    | ~~4~~  | ~~Exports safe on multiple instances (object storage + persisted progress)~~                          |
| ~~T129~~ | ~~2~~  | ~~Cleanup~~    | ~~4~~  | ~~Backend dead-code removal and a stricter linter~~                                                   |
| ~~T130~~ | ~~2~~  | ~~Bug~~        | ~~1~~  | ~~Login error swallowed by the refresh interceptor~~                                                  |
| ~~T131~~ | ~~2~~  | ~~Bug~~        | ~~1~~  | ~~Public payment result pages for the /pay/:token flow~~                                              |
| ~~T132~~ | ~~2~~  | ~~Security~~   | ~~2~~  | ~~Clear tenant and user data on logout and tenant switch~~                                            |
| ~~T133~~ | ~~2~~  | ~~Bug~~        | ~~2~~  | ~~Job times entered and shown in the tenant's timezone, stored as true UTC~~                          |
| ~~T134~~ | ~~2~~  | ~~Security~~   | ~~1~~  | ~~Strip sensitive query params from analytics and error breadcrumbs~~                                 |
| ~~T135~~ | ~~2~~  | ~~Frontend~~   | ~~4~~  | ~~Frontend medium-severity sweep~~                                                                    |
| ~~T136~~ | ~~2~~  | ~~Testing~~    | ~~3~~  | ~~Green frontend test suite and lint~~                                                                |
| ~~T137~~ | ~~2~~  | ~~Cleanup~~    | ~~3~~  | ~~Frontend dead-code removal (~18.5k unreachable lines)~~                                             |
| ~~T138~~ | ~~2~~  | ~~UX~~         | ~~2~~  | ~~Preview polish: honest contact form, legal pages marked draft, favicon~~                            |
| ~~T139~~ | ~~3~~  | ~~DB~~         | ~~4~~  | ~~golang-migrate baseline (000001), cmd/migrate, dev-only startup migrations~~                        |
| ~~T140~~ | ~~3~~  | ~~Backend~~    | ~~2~~  | ~~/version endpoint and schema-aware readiness~~                                                      |
| ~~T141~~ | ~~3~~  | ~~Backend~~    | ~~4~~  | ~~Background tickers → internal job endpoints + scheduled-jobs workflow~~                             |
| ~~T142~~ | ~~3~~  | ~~Backend~~    | ~~2~~  | ~~Redis optional at boot~~                                                                            |
| ~~T143~~ | ~~3~~  | ~~Infra~~      | ~~2~~  | ~~Cloud Run-ready Dockerfile and .dockerignore~~                                                      |
| ~~T144~~ | ~~3~~  | ~~DB~~         | ~~1~~  | ~~DB pool sizing and pooler (simple protocol) support~~                                               |
| ~~T145~~ | ~~3~~  | ~~CI~~         | ~~6~~  | ~~CI rewrite: lint, tests with Postgres+Redis, govulncheck, OpenAPI drift, frontend, Docker + Trivy~~ |
| ~~T146~~ | ~~3~~  | ~~Deploy~~     | ~~5~~  | ~~Tag-driven release-deploy pipeline, VERSION scripts and RELEASING.md~~                              |
| ~~T147~~ | ~~3~~  | ~~CI~~         | ~~2~~  | ~~Dependabot deps() prefixes, automerge and weekly security scan~~                                    |
| ~~T148~~ | ~~3~~  | ~~CI~~         | ~~2~~  | ~~Versioned bash git hooks replace python pre-commit; commit-msg prefix contract~~                    |
| ~~T149~~ | ~~3~~  | ~~Cleanup~~    | ~~2~~  | ~~Repo hygiene: tracked artifacts, Fly/nginx/cliff removal, .gitignore, single Makefile~~             |
| ~~T150~~ | ~~3~~  | ~~Deploy~~     | ~~6~~  | ~~Cloudflare Workers static assets + /api edge proxy Worker + _headers~~                              |
| ~~T151~~ | ~~3~~  | ~~Docs~~       | ~~3~~  | ~~Ops docs: Cloud Run setup, deploy/rollback, backups + nightly pg_dump → R2 workflow~~               |
| ~~T152~~ | ~~3~~  | ~~CI~~         | ~~1~~  | ~~Commit generated OpenAPI and TS types for drift gating~~                                            |
| ~~T164~~ | ~~4~~  | ~~Frontend~~   | ~~16~~ | ~~React 19, React Router 7, Vite major, ESLint 9, Headless UI 2~~                                     |
| ~~T202~~ | ~~14~~ | ~~Research~~   | ~~1~~  | ~~Decide production database hosting (OWNER DECISION)~~                                               |
| ~~T223~~ | ~~2~~  | ~~Backend~~    | ~~4~~  | ~~Request IDs and a standard API error envelope~~                                                     |
| ~~T224~~ | ~~2~~  | ~~Bug~~        | ~~4~~  | ~~Quote correctness: lost customer edits, accept race, expiry, editable sent quotes~~                 |
| ~~T225~~ | ~~2~~  | ~~Bug~~        | ~~4~~  | ~~Invoice totals: one server-side recalculation, tax/discount order, no double invoicing~~            |
| ~~T226~~ | ~~2~~  | ~~Bug~~        | ~~3~~  | ~~Tenant creation and invitation acceptance in transactions; per-tenant uniqueness~~                  |
| ~~T227~~ | ~~2~~  | ~~Bug~~        | ~~3~~  | ~~Frontend follow-through for the Phase 1 auth changes (invite codes, 429, logout-everywhere)~~       |
| ~~T228~~ | ~~2~~  | ~~Bug~~        | ~~2~~  | ~~Quotes list: backend search (q) and sort params~~                                                   |
| ~~T229~~ | ~~2~~  | ~~Bug~~        | ~~2~~  | ~~Invite and invitation API polish: distinct invalid-code error, encoded links, message field~~       |
| ~~T230~~ | ~~13~~ | ~~Auth~~       | ~~3~~  | ~~Expose the caller's effective permissions per tenant; decide who can delete an organization~~       |
| ~~T232~~ | ~~2~~  | ~~Bug~~        | ~~2~~  | ~~Frontend money calls match the API: record payment URL, cancel reason, refunds, quote revise~~      |
| ~~T233~~ | ~~2~~  | ~~Bug~~        | ~~1~~  | ~~JobMaterial.Billable default:true stores false as true~~                                            |
| ~~T234~~ | ~~2~~  | ~~Bug~~        | ~~2~~  | ~~Invoice/quote forms use the API's fields: tax_rate_id, terms_and_conditions, default tax prefill~~  |
| ~~T237~~ | ~~2~~  | ~~Backend~~    | ~~3~~  | ~~Tax rates API + tax-rate picker; terms on invoice create; keep line discounts on edit~~             |
| ~~T238~~ | ~~2~~  | ~~Security~~   | ~~1~~  | ~~Job assignment checks the technician belongs to the tenant~~                                        |
| ~~T239~~ | ~~2~~  | ~~Security~~   | ~~1~~  | ~~Access log drops query strings~~                                                                    |
| ~~T245~~ | ~~3~~  | ~~Testing~~    | ~~2~~  | ~~Repair the Postgres repository tests so CI go-test is green~~                                       |
| T241     | 2      | Bug            | 3      | CSV imports are queued but never processed                                                            |
| T242     | 2      | Bug            | 1      | PDF header/footer files are deleted before wkhtmltopdf reads them                                     |
| T243     | 2      | Security       | 1      | A/R aging CSV export: proper CSV writer + formula neutralisation                                      |
| T244     | 2      | Bug            | 3      | Handle the discarded GORM .Error results                                                              |
| T101     | 3      | Monitoring     | 2      | Upload source maps to Sentry for production builds                                                    |
| T247     | 3      | Platform       | 3      | Move S3 uploads from the deprecated feature/s3/manager to feature/s3/transfermanager                  |
| T153     | 4      | Backend        | 24     | Repositories take ctx + tenantID; services stop taking *gorm.DB; transaction manager                  |
| T154     | 4      | Security       | 12     | Postgres RLS backstop keyed on a per-transaction tenant GUC + isolation tests                         |
| T155     | 4      | Backend        | 40     | Gin → chi                                                                                             |
| T156     | 4      | Backend        | 120    | GORM → pgx/sqlc, one repository at a time                                                             |
| T157     | 4      | Testing        | 16     | Drop GORM and sqlite; backend tests on Postgres via testcontainers                                    |
| T158     | 4      | Auth           | 40     | Supabase Auth: own sign-in screens, JWKS verification, users.auth_user_id, invite flow                |
| T159     | 4      | Email          | 6      | Email → Resend only (remove SES/SNS/CloudWatch and the AWS email SDKs)                                |
| T160     | 4      | Infra          | 4      | Storage → Cloudflare R2 (fix presign host rewrite)                                                    |
| T161     | 4      | Infra          | 4      | Production Redis: Upstash for rate limits/lockout/cache, or move them to Postgres                     |
| T162     | 4      | Frontend       | 48     | Frontend server state → TanStack Query v5 (~26 pages)                                                 |
| T163     | 4      | Frontend       | 32     | Tailwind v4 + semantic theme tokens + dark mode                                                       |
| T165     | 4      | Monitoring     | 4      | Better Stack uptime/status + PostHog server-side events parity                                        |
| T166     | 4      | Research       | 2      | Decide money representation (decimal vs integer cents) before the pgx port                            |
| T248     | 4      | Monitoring     | 3      | Upgrade @sentry/react to 11 with an explicit data-collection config                                   |
| T249     | 4      | Frontend       | 2      | Drop Vite's legacy.inconsistentCjsInterop flag and ready the config for Vite's native loader          |
| T028     | 5      | Feature        | 3      | Add "Convert Quote to Job" button                                                                     |
| T029     | 5      | Feature        | 2      | Add "Generate Invoice from Job" button                                                                |
| T030     | 5      | Feature        | 4      | Auto-populate invoice from job data                                                                   |
| T033     | 5      | Reporting      | 2      | Add "Jobs Needing Invoice" dashboard widget                                                           |
| T035     | 5      | Auth           | 4      | Create Technician role                                                                                |
| T041     | 5      | Feature        | 6      | Implement quote auto follow-ups                                                                       |
| T042     | 5      | UX             | 4      | Build follow-up queue UI                                                                              |
| T051     | 5      | Email          | 4      | Implement quote follow-up email sequence                                                              |
| T052     | 5      | Auth           | 3      | Create Dispatcher role                                                                                |
| T058     | 5      | Feature        | 4      | Add required fields validation by job stage                                                           |
| T060     | 5      | Feature        | 8      | Create quote template system (good/better/best)                                                       |
| T062     | 5      | Feature        | 2      | Add quote aging tracking                                                                              |
| T076     | 5      | Feature        | 1      | Add "Recreate Last Quote" button                                                                      |
| T080     | 5      | Feature        | 3      | Create terms & conditions templates                                                                   |
| T082     | 5      | Feature        | 2      | Add "Create Quote from Job" button                                                                    |
| T167     | 5      | Feature        | 12     | Recurring service agreements UI wired to the recurrence engine                                        |
| T168     | 5      | Feature        | 12     | Price book (services and materials catalog with flat-rate pricing)                                    |
| T235     | 5      | Feature        | 4      | Tenant settings UI for currency, number prefixes, default payment term and tax rate; validate on save |
| T236     | 5      | Feature        | 3      | Currency-aware money display in the UI, emails and PDFs                                               |
| T240     | 5      | UX             | 3      | Forms show the API's per-field validation errors                                                      |
| T061     | 6      | Payments       | 6      | Add required deposit to accept quote                                                                  |
| T063     | 6      | Payments       | 6      | Add deposit tracking and application                                                                  |
| T083     | 6      | Payments       | 4      | Implement late fee calculation                                                                        |
| T169     | 6      | Payments       | 24     | Stripe Connect Express onboarding; invoice payments routed to the tenant                              |
| T170     | 6      | Payments       | 6      | Card + ACH invoice payments, ACH preferred for B2B                                                    |
| T171     | 6      | Payments       | 10     | Saved payment methods + autopay                                                                       |
| T172     | 6      | Payments       | 16     | Tap to Pay via Stripe Terminal                                                                        |
| T027     | 7      | Feature        | 8      | Add drag-and-drop rescheduling to job calendar                                                        |
| T038     | 7      | Feature        | 6      | Add technician availability management UI                                                             |
| T039     | 7      | Feature        | 8      | Create emergency job insertion workflow                                                               |
| T040     | 7      | SMS            | 3      | Add auto-notify on emergency insertion                                                                |
| T056     | 7      | Feature        | 3      | Add workload capacity warnings                                                                        |
| T057     | 7      | Feature        | 2      | Add double-booking override with reason                                                               |
| T059     | 7      | Feature        | 3      | Add technician skill tags                                                                             |
| T077     | 7      | Feature        | 4      | Add time-off request UI                                                                               |
| T078     | 7      | Feature        | 4      | Add skill-based job matching suggestions                                                              |
| T079     | 7      | Feature        | 1      | Add travel time notes field                                                                           |
| T099     | 7      | Backend        | 4      | Add tenant timezone support for conflict detection                                                    |
| T173     | 7      | Mobile         | 24     | Technician "My Day" mobile-first view                                                                 |
| T174     | 7      | Feature        | 8      | Job photos and attachments (R2)                                                                       |
| T175     | 7      | Feature        | 12     | Job checklists / forms                                                                                |
| T176     | 7      | Feature        | 8      | Customer signature on completion + service report PDF emailed                                         |
| T177     | 7      | Feature        | 12     | Time clock per job + timesheet export                                                                 |
| T178     | 7      | Feature        | 8      | Map / route view of the day                                                                           |
| T179     | 7      | Mobile         | 40     | Offline-capable tech app (queue + sync)                                                               |
| T180     | 7      | Mobile         | 24     | Capacitor-wrapped iOS/Android apps in the stores                                                      |
| T036     | 8      | Feature        | 3      | Add manual Call Log entry                                                                             |
| T037     | 8      | Feature        | 4      | Auto-log system messages to customer activity                                                         |
| T048     | 8      | Email          | 3      | Add appointment confirmation emails                                                                   |
| T049     | 8      | SMS            | 4      | Add appointment reminder emails/SMS                                                                   |
| T050     | 8      | SMS            | 2      | Add "Tech on the way" manual trigger                                                                  |
| T054     | 8      | UX             | 4      | Create Communication Log component                                                                    |
| T055     | 8      | UX             | 8      | Unified Customer Timeline view                                                                        |
| T072     | 8      | Feature        | 6      | Create message template management UI                                                                 |
| T073     | 8      | Feature        | 4      | Add rich variable system for templates                                                                |
| T089     | 8      | Email          | 6      | Add email/SMS delivery tracking                                                                       |
| T090     | 8      | Monitoring     | 3      | Add communication failure alerts                                                                      |
| T091     | 8      | Feature        | 4      | Add notification preferences per customer                                                             |
| T181     | 8      | SMS            | 20     | Two-way SMS inbox (Twilio + 10DLC)                                                                    |
| T182     | 8      | Marketing      | 4      | Review request automation                                                                             |
| T081     | 9      | Feature        | 4      | Add e-signature for quote acceptance                                                                  |
| T183     | 9      | Feature        | 10     | Public quote approval page (approve/decline, e-sign, optional deposit)                                |
| T184     | 9      | Feature        | 24     | Customer portal / client hub                                                                          |
| T185     | 9      | Feature        | 12     | Online booking / request widget embeddable on websites                                                |
| T044     | 10     | Feature        | 8      | Build collections dashboard                                                                           |
| T064     | 10     | Feature        | 2      | Add last contact date tracking on invoices                                                            |
| T065     | 10     | Feature        | 3      | Add promise-to-pay notes                                                                              |
| T186     | 10     | Feature        | 16     | Multi-site customers (service locations; invoice per site or consolidated)                            |
| T187     | 10     | Feature        | 6      | PO capture and matching on jobs and invoices                                                          |
| T188     | 10     | Feature        | 8      | Consolidated monthly invoicing                                                                        |
| T189     | 10     | Feature        | 6      | Statements of account (portal + email)                                                                |
| T190     | 10     | Feature        | 24     | Property-manager approval workflow (NTE limits, multi-approver, email intake)                         |
| T191     | 10     | Feature        | 12     | Maintenance agreements with autopay + renewal reminders                                               |
| T095     | 11     | Integration    | 12     | Build outgoing webhooks system                                                                        |
| T096     | 11     | Backend        | 4      | Build event log table                                                                                 |
| T097     | 11     | Integration    | 4      | Add webhook retry logic                                                                               |
| T098     | 11     | Docs           | 3      | Document public API (OpenAPI/Swagger)                                                                 |
| T192     | 11     | Integration    | 32     | QuickBooks Online sync (customers, invoices, payments; one-way first)                                 |
| T193     | 11     | Integration    | 8      | Import mappers for Jobber / Housecall Pro exports + generic CSV                                       |
| T194     | 11     | Integration    | 12     | Zapier app                                                                                            |
| T195     | 11     | Integration    | 8      | Wisetack consumer financing                                                                           |
| T196     | 11     | Integration    | 12     | Gusto payroll / timesheets                                                                            |
| T197     | 11     | Integration    | 8      | AI receptionist partner integration                                                                   |
| T045     | 12     | Reporting      | 2      | Add aging quotes dashboard widget                                                                     |
| T046     | 12     | Reporting      | 2      | Add follow-up queue dashboard widget                                                                  |
| T047     | 12     | Reporting      | 4      | Build "stuck work" report                                                                             |
| T066     | 12     | Reporting      | 4      | Add technician utilization widget                                                                     |
| T067     | 12     | Reporting      | 2      | Add quote conversion rate metric                                                                      |
| T068     | 12     | Reporting      | 3      | Build job-completion-to-invoice time report                                                           |
| T069     | 12     | Reporting      | 3      | Build invoice-to-paid time report (DSO)                                                               |
| T070     | 12     | Reporting      | 3      | Build revenue by service type report                                                                  |
| T071     | 12     | Reporting      | 3      | Build collections rate report                                                                         |
| T084     | 12     | Reporting      | 3      | Build quote-to-job conversion report                                                                  |
| T085     | 12     | Reporting      | 3      | Build lead-to-quote time report                                                                       |
| T086     | 12     | Reporting      | 2      | Build quote-to-acceptance time report                                                                 |
| T087     | 12     | Reporting      | 3      | Build revenue by technician report                                                                    |
| T088     | 12     | Reporting      | 4      | Add comparative analysis (vs prior period)                                                            |
| T198     | 12     | Reporting      | 12     | Job costing / profit per job                                                                          |
| T246     | 12     | Perf           | 3      | Lazy-load / trim Sentry and PostHog (109 KB gzipped on every page)                                    |
| T074     | 13     | Auth           | 3      | Create Accountant role                                                                                |
| T075     | 13     | Auth           | 12     | Add field-level permission controls                                                                   |
| T092     | 13     | Auth           | 6      | Implement 2FA (TOTP)                                                                                  |
| T093     | 13     | Backend        | 12     | Implement comprehensive audit logging                                                                 |
| T094     | 13     | Auth           | 0      | Add Google OAuth sign-in (Won't Do)                                                                   |
| T199     | 13     | Feature        | 6      | One-click full data export (trust promise)                                                            |
| T200     | 13     | Legal          | 6      | Account/tenant deletion + data retention policy                                                       |
| T201     | 13     | Legal          | 3      | Privacy/CCPA handling SOP                                                                             |
| T231     | 13     | Auth           | 1      | Owner decision: may a tenant's owner/admin delete their own organization? (Deferred)                  |
| T203     | 14     | Deploy         | 4      | GCP project + WIF + Artifact Registry + Cloud Run service                                             |
| T204     | 14     | Deploy         | 2      | Deploy the Worker + DNS for servicepro.sharkfins.xyz                                                  |
| T205     | 14     | Deploy         | 2      | Invite-gated preview launch                                                                           |
| T206     | 14     | Email          | 2      | Production email domain on Resend (SPF/DKIM/DMARC)                                                    |
| T207     | 14     | Legal          | 4      | Terms of Service + Privacy Policy legal review                                                        |
| T208     | 14     | Ops            | 2      | Support inbox + Better Stack status page                                                              |
| T209     | 14     | Ops            | 2      | Activate nightly backups to R2 + restore drill                                                        |
| T210     | 14     | Billing        | 12     | ServicePro plans & billing: Solo/Team/Pro flat, unlimited users, founding offer                       |
| T211     | 14     | Docs           | 1      | Update Fleet and the blog when ServicePro goes live                                                   |
| T212     | 14     | Ops            | 2      | Production config review                                                                              |
| T213     | 14     | Ops            | 1      | Cost guardrails (Cloud Run max instances × DB pool, budget alerts)                                    |
| T214     | 15     | AI             | 24     | AI collections agent                                                                                  |
| T215     | 15     | AI             | 20     | AI quote drafting from photos, voice or notes against the price book                                  |
| T216     | 15     | AI             | 12     | "Ask your business" chat over reporting views                                                         |
| T217     | 15     | AI             | 12     | MCP server for ServicePro                                                                             |
| T218     | 16     | Research       | 10     | Customer discovery: interview 10 commercial service contractors                                       |
| T219     | 16     | Marketing      | 12     | Marketing site with positioning + transparent pricing                                                 |
| T220     | 16     | Marketing      | 4      | White-glove migration offer + playbook                                                                |
| T221     | 16     | Marketing      | 8      | Comparison / alternative pages (vs Jobber, Housecall Pro, ServiceTitan)                               |
| T222     | 16     | Research       | 6      | Validate secondary beachhead (inspection/compliance trades)                                           |

---

## Phase Roadmap

### Phase 0 — Before 2026-10-16

Work completed under the old P0–P3/sprint TODO. Details are in Verified Complete.

- [x] T001 — Recovery middleware Sentry integration
- [x] T002 — Centralize direct fetch() calls
- [x] T003 — Add CSP/HSTS headers to nginx.conf
- [x] T004 — Create GitHub Actions CI/CD workflows
- [x] T005 — Expose /metrics endpoint for Prometheus
- [x] T006 — Structured JSON logging (replace fmt.Printf)
- [x] T007 — Fix `any` types (50 instances)
- [x] T008 — Remove @ts-nocheck directives
- [x] T009 — Enable noUnusedLocals/noUnusedParameters
- [x] T010 — Apply error tracking middleware
- [x] T012 — Bundle size monitoring in CI
- [x] T013 — Increase frontend test coverage to 70%+
- [x] T014 — Integration tests for critical workflows
- [x] T015 — Generate OpenAPI/Swagger documentation
- [x] T016 — Frontend bundle analysis + optimization
- [x] T017 — Query performance monitoring
- [x] T018 — Dead code elimination
- [x] T019 — Integrate calendar view for job scheduling
- [x] T020 — Build conflict detection API endpoint
- [x] T021 — Extract duplicate file download utility
- [x] T022 — Extract duplicate URLSearchParams builder
- [x] T023 — Consolidate cache hooks (useLocalCache/useSession)
- [x] T024 — Full Sentry integration (frontend + backend)
- [x] T025 — Integrate product analytics (PostHog)
- [x] T026 — Set up business KPI dashboard (Grafana)
- [x] T031 — Build A/R aging buckets report
- [x] T032 — Build "Who Owes What" report
- [x] T034 — Wire payment reminder automation
- [x] T043 — Wire payment reminder notifications
- [x] T053 — Add "Repeat Job" button on customer detail
- [x] T100 — Add business hours & timezone to org settings

---

### Phase 1 — Security Remediation (T102, T103, T104, T105, T106, T107, T108, T109, T110, T111, T112, T113, T114, T115, T116, T117)

**Blocks any hosting, even invite-only.** From the 2026-10-16 security audit (`docs/audits/2026-10-16/backend-security.md`): there is no tenant isolation and any user can make themselves super_admin. Nothing gets a URL until every task here is done. **0 open tasks, ~0h.** 16 done.

- [x] T102 — Tenant-scope all business data ✓ (2026-10-16)
- [x] T103 — Authorize tenant member, invitation and role endpoints; no self-escalation ✓ (2026-10-16)
- [x] T104 — Evaluate permissions per tenant ✓ (2026-10-16)
- [x] T105 — Remove or lock the raw Stripe passthrough endpoints ✓ (2026-10-16)
- [x] T106 — Backend-enforced invite-only registration; reject non-edge traffic in production ✓ (2026-10-16)
- [x] T107 — Edge client-IP middleware and a safe trusted-proxy default ✓ (2026-10-16)
- [x] T108 — Escape PDF export HTML; drop wkhtmltopdf local file access ✓ (2026-10-16)
- [x] T109 — Stripe webhook reliability: persistent dedupe, accept retries, body limit, idempotent payment ✓ (2026-10-16)
- [x] T110 — Token hardening: access vs refresh type, revoke on logout and password reset ✓ (2026-10-16)
- [x] T111 — Login hardening: no lockout DoS, uniform errors and timing ✓ (2026-10-16)
- [x] T112 — Neutralise CSV/Excel formula injection in exports ✓ (2026-10-16)
- [x] T113 — Fail closed when no email or storage provider in production; never log tokens ✓ (2026-10-16)
- [x] T114 — Lock down /health detail, /metrics, Swagger and debug mode in production ✓ (2026-10-16)
- [x] T115 — Secure cookies by default and CSRF protection ✓ (2026-10-16)
- [x] T116 — Low-severity security sweep ✓ (2026-10-16)
- [x] T117 — Remediate dependency vulnerabilities and bump the Go toolchain ✓ (2026-10-16)

### Phase 2 — Correctness & Cleanup (T118, T119, T120, T121, T122, T123, T124, T125, T126, T127, T128, T129, T130, T131, T132, T133, T134, T135, T136, T137, T138, T223, T224, T225, T226, T227, T228, T229, T232, T233, T234, T237, T238, T239, T241, T242, T243, T244)

Money flows that are wired but broken (payments don't update invoices, reminders never sent, numbering wedges, a panicking webhook), frontend bugs users would hit in the preview, and dead code. From `docs/audits/2026-10-16/backend-correctness.md` and `docs/audits/2026-10-16/frontend.md`. **4 open tasks, ~8h.** 34 done.

- [x] T118 — Recording a payment updates the invoice balance and status ✓ (2026-10-16)
- [x] T119 — Stripe Checkout charges the true balance due ✓ (2026-10-16)
- [x] T120 — Invoice and job numbering survives soft-delete and concurrency ✓ (2026-10-16)
- [x] T121 — Payment reminders actually send ✓ (2026-10-16)
- [x] T122 — Fix /webhooks/stripe panicking on subscription and invoice events ✓ (2026-10-16)
- [x] T123 — Add the missing SQL functions and views behind customer search and reports ✓ (2026-10-16)
- [x] T124 — Enforce invoice status transitions; unify the two job state machines ✓ (2026-10-16)
- [x] T125 — Handle refunds in the ledger ✓ (2026-10-16)
- [x] T126 — Backend Sentry actually initialised; wire or delete the Loki/BetterStack/OTel stubs ✓ (2026-10-16)
- [x] T127 — Backend reads tenant settings: currency, number prefixes, defaults ✓ (2026-10-16)
- [x] T128 — Exports safe on multiple instances (object storage + persisted progress) ✓ (2026-10-16)
- [x] T129 — Backend dead-code removal and a stricter linter ✓ (2026-10-16)
- [x] T130 — Login error swallowed by the refresh interceptor ✓ (2026-10-16)
- [x] T131 — Public payment result pages for the /pay/:token flow ✓ (2026-10-16)
- [x] T132 — Clear tenant and user data on logout and tenant switch ✓ (2026-10-16)
- [x] T133 — Job times entered and shown in the tenant's timezone, stored as true UTC ✓ (2026-10-16)
- [x] T134 — Strip sensitive query params from analytics and error breadcrumbs ✓ (2026-10-16)
- [x] T135 — Frontend medium-severity sweep ✓ (2026-10-16)
- [x] T136 — Green frontend test suite and lint ✓ (2026-10-16)
- [x] T137 — Frontend dead-code removal (~18.5k unreachable lines) ✓ (2026-10-16)
- [x] T138 — Preview polish: honest contact form, legal pages marked draft, favicon ✓ (2026-10-16)
- [x] T223 — Request IDs and a standard API error envelope ✓ (2026-10-16)
- [x] T224 — Quote correctness: lost customer edits, accept race, expiry, editable sent quotes ✓ (2026-10-16)
- [x] T225 — Invoice totals: one server-side recalculation, tax/discount order, no double invoicing ✓ (2026-10-16)
- [x] T226 — Tenant creation and invitation acceptance in transactions; per-tenant uniqueness ✓ (2026-10-16)
- [x] T227 — Frontend follow-through for the Phase 1 auth changes (invite codes, 429, logout-everywhere) ✓ (2026-10-16)
- [x] T228 — Quotes list: backend search (q) and sort params ✓ (2026-10-16)
- [x] T229 — Invite and invitation API polish: distinct invalid-code error, encoded links, message field ✓ (2026-10-16)
- [x] T232 — Frontend money calls match the API: record payment URL, cancel reason, refunds, quote revise ✓ (2026-10-16)
- [x] T233 — JobMaterial.Billable default:true stores false as true ✓ (2026-10-16)
- [x] T234 — Invoice/quote forms use the API's fields: tax_rate_id, terms_and_conditions, default tax prefill ✓ (2026-10-16)
- [x] T237 — Tax rates API + tax-rate picker; terms on invoice create; keep line discounts on edit ✓ (2026-10-16)
- [x] T238 — Job assignment checks the technician belongs to the tenant ✓ (2026-10-16)
- [x] T239 — Access log drops query strings ✓ (2026-10-16)
- [ ] T241 — CSV imports are queued but never processed
- [ ] T242 — PDF header/footer files are deleted before wkhtmltopdf reads them
- [ ] T243 — A/R aging CSV export: proper CSV writer + formula neutralisation
- [ ] T244 — Handle the discarded GORM .Error results

#### T241 — CSV imports are queued but never processed

**Phase**: 2 · **Category**: Bug · **Hours**: 3 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T102

**What it is** — An uploaded import actually runs and reports per-row results.

**What it does** — The import handler enqueues jobs but nothing calls `ImportService.ProcessNextJob`/`ProcessImport`, so every import stays pending. Run imports via an internal job (`POST /internal/jobs/imports`, idempotent claim, added to scheduled-jobs.yml) or in-request for small files; cap size/rows (F-23, `MaxBytesReader`); tenant-scoped (already tenant-aware since T102); tests.

**Why we need it** — Customers can't bring their data in, which blocks the white-glove migration (T193/T220).

**Notes** — Found during T129 (2026-10-16). T193 depends on this.

---

#### T242 — PDF header/footer files are deleted before wkhtmltopdf reads them

**Phase**: 2 · **Category**: Bug · **Hours**: 1 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Export PDFs get their header/footer (page numbers).

**What it does** — `PDFService.buildWkhtmltopdfArgs` writes the header/footer HTML and `defer os.Remove`s them inside the builder, so they're gone before wkhtmltopdf runs. Return a cleanup func and remove after the command finishes; test that the files exist while the command runs.

**Why we need it** — Export PDFs silently lose page-number footers.

**Notes** — Found during T129 (2026-10-16).

---

#### T243 — A/R aging CSV export: proper CSV writer + formula neutralisation

**Phase**: 2 · **Category**: Security · **Hours**: 1 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T112

**What it is** — The A/R aging export is valid CSV and safe to open in Excel.

**What it does** — It builds lines by string concatenation: commas/quotes in customer names break columns and it skips the `csvsafe` formula-injection helper from T112. Use `encoding/csv` + `csvsafe`; test with a name containing `,`, `"` and a leading `=`.

**Why we need it** — Formula injection (T112) is still open on this one export.

**Notes** — Found during T129 (2026-10-16).

---

#### T244 — Handle the discarded GORM .Error results

**Phase**: 2 · **Category**: Bug · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — No database write silently fails.

**What it does** — ~16 sites call GORM chains without checking `.Error` (correctness audit §6); errcheck can't see them. Check each: propagate, or log with context where best-effort is intended; tests where behaviour changes. Also retire the legacy `UserRepositoryInterface` once auth/registration move to the GORM repository.

**Why we need it** — A failed write currently reports success.

**Notes** — Left over from T129 (2026-10-16).

---

### Phase 3 — Platform & Delivery (T101, T139, T140, T141, T142, T143, T144, T145, T146, T147, T148, T149, T150, T151, T152, T245, T247)

Migrations, versioning, CI, the release pipeline and the edge setup — League Looper's delivery pattern applied to ServicePro. From `docs/audits/2026-10-16/infra.md`. Deploy jobs stay skipped until Phase 14 provisions GCP. **2 open tasks, ~5h.** 15 done.

- [x] T139 — golang-migrate baseline (000001), cmd/migrate, dev-only startup migrations ✓ (2026-10-16)
- [x] T140 — /version endpoint and schema-aware readiness ✓ (2026-10-16)
- [x] T141 — Background tickers → internal job endpoints + scheduled-jobs workflow ✓ (2026-10-16)
- [x] T142 — Redis optional at boot ✓ (2026-10-16)
- [x] T143 — Cloud Run-ready Dockerfile and .dockerignore ✓ (2026-10-16)
- [x] T144 — DB pool sizing and pooler (simple protocol) support ✓ (2026-10-16)
- [x] T145 — CI rewrite: lint, tests with Postgres+Redis, govulncheck, OpenAPI drift, frontend, Docker + Trivy ✓ (2026-10-16)
- [x] T146 — Tag-driven release-deploy pipeline, VERSION scripts and RELEASING.md ✓ (2026-10-16)
- [x] T147 — Dependabot deps() prefixes, automerge and weekly security scan ✓ (2026-10-16)
- [x] T148 — Versioned bash git hooks replace python pre-commit; commit-msg prefix contract ✓ (2026-10-16)
- [x] T149 — Repo hygiene: tracked artifacts, Fly/nginx/cliff removal, .gitignore, single Makefile ✓ (2026-10-16)
- [x] T150 — Cloudflare Workers static assets + /api edge proxy Worker + _headers ✓ (2026-10-16)
- [x] T151 — Ops docs: Cloud Run setup, deploy/rollback, backups + nightly pg_dump → R2 workflow ✓ (2026-10-16)
- [x] T152 — Commit generated OpenAPI and TS types for drift gating ✓ (2026-10-16)
- [x] T245 — Repair the Postgres repository tests so CI go-test is green ✓ (2026-10-16)
- [ ] T101 — Upload source maps to Sentry for production builds
- [ ] T247 — Move S3 uploads from the deprecated feature/s3/manager to feature/s3/transfermanager

#### T101 — Upload source maps to Sentry for production builds

**Phase**: 3 · **Category**: Monitoring · **Hours**: 2 · **Status**: Open · **v1 Gate**: No · **Depends on**: T146

**What it is** — Production frontend errors in Sentry show original file and line instead of minified code.

**What it does** — Add `@sentry/vite-plugin`; build with `build.sourcemap: 'hidden'`; upload during the web job of `release-deploy.yml` with `SENTRY_AUTH_TOKEN`, release = git SHA (matching the SDK's `release`); delete `.map` files from `dist/` before `wrangler deploy`; verify one trace. Original sub-tasks (plugin, CI auth token, upload in the deploy workflow, verify) stand — the workflow is now T146's, not `deploy-release.yml`.

**Why we need it** — Minified stack traces make frontend errors from real users hard to act on.

**Notes** — Old task (P2, Feb 2026). GlitchTip stays for local dev; production is Sentry SaaS. infra audit §5 CHANGE `vite.config.ts`.

---

#### T247 — Move S3 uploads from the deprecated feature/s3/manager to feature/s3/transfermanager

**Phase**: 3 · **Category**: Platform · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Object storage uploads use the AWS SDK's supported transfer manager instead of the deprecated `manager.Uploader`.

**What it does** — Replace `manager.NewUploader` / `Uploader.Upload` in `pkg/clients/storage/s3/s3.go` with `feature/s3/transfermanager` (multipart behaviour, part size and concurrency kept), drop the three `//nolint:staticcheck` SA1019 suppressions, and check the upload against MinIO locally (and R2 when T155 lands).

**Why we need it** — aws-sdk-go-v2 deprecated `feature/s3/manager` (discussions/3306); it will stop receiving fixes, and the lint suppressions hide any new deprecations in that file.

**Notes** — Found when applying the 2026-10-18 Go minor/patch group (Dependabot #62).

---

### Phase 4 — Stack Convergence (LL parity; incremental) (T153, T154, T155, T156, T157, T158, T159, T160, T161, T162, T163, T164, T165, T166, T248, T249)

D2: move to League Looper's stack one strangler step at a time while the app keeps working. Gating subset: the repository/tenant boundary, RLS, Supabase Auth, Resend, R2 and production Redis must be on the target stack before real customers; the chi and pgx ports are not gating. **15 open tasks, ~357h.** 1 done.

- [x] T164 — React 19, React Router 7, Vite major, ESLint 9, Headless UI 2 ✓ (2026-10-18)
- [ ] T153 — Repositories take ctx + tenantID; services stop taking *gorm.DB; transaction manager
- [ ] T154 — Postgres RLS backstop keyed on a per-transaction tenant GUC + isolation tests
- [ ] T155 — Gin → chi
- [ ] T156 — GORM → pgx/sqlc, one repository at a time
- [ ] T157 — Drop GORM and sqlite; backend tests on Postgres via testcontainers
- [ ] T158 — Supabase Auth: own sign-in screens, JWKS verification, users.auth_user_id, invite flow
- [ ] T159 — Email → Resend only (remove SES/SNS/CloudWatch and the AWS email SDKs)
- [ ] T160 — Storage → Cloudflare R2 (fix presign host rewrite)
- [ ] T161 — Production Redis: Upstash for rate limits/lockout/cache, or move them to Postgres
- [ ] T162 — Frontend server state → TanStack Query v5 (~26 pages)
- [ ] T163 — Tailwind v4 + semantic theme tokens + dark mode
- [ ] T165 — Better Stack uptime/status + PostHog server-side events parity
- [ ] T166 — Decide money representation (decimal vs integer cents) before the pgx port
- [ ] T248 — Upgrade @sentry/react to 11 with an explicit data-collection config
- [ ] T249 — Drop Vite's legacy.inconsistentCjsInterop flag and ready the config for Vite's native loader

#### T153 — Repositories take ctx + tenantID; services stop taking *gorm.DB; transaction manager

**Phase**: 4 · **Category**: Backend · **Hours**: 24 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T102

**What it is** — Every data access goes through a repository method that takes `context.Context` and the tenant ID, and multi-step writes run through one `TxManager`.

**What it does** — Add `ctx` + `tenantID` to every repository interface (`internal/repository/interface.go`; today e.g. `CustomerRepositoryInterface.GetByID(id uuid.UUID)` has neither). Move the services that take `*gorm.DB` directly behind repositories: Invoice (936 lines), Revenue, CustomerReport, ARAging, UnifiedExport, PaymentTerms, Invitation, PaymentReminder, Import, and Job (takes both). Add `TxManager.WithinTx(ctx, func(ctx) error)` carrying the tx in the context, replacing the 4 ad-hoc `Transaction(func(tx *gorm.DB))` sites.

**Why we need it** — It's the boundary everything after depends on: tenant filtering can't be forgotten, request cancellation propagates, the transactions needed by T118/T225/T226 have one shape, RLS (T154) needs a per-transaction hook, and the pgx port (T156) can happen one repository at a time behind it.

**Notes** — correctness audit §8 "Is there a repository boundary already?" and "Suggested sequencing" step 2. D2, D5. Much smaller after T129 deletes the dead repositories and services — do that first if time allows.

---

#### T154 — Postgres RLS backstop keyed on a per-transaction tenant GUC + isolation tests

**Phase**: 4 · **Category**: Security · **Hours**: 12 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T153, T139

**What it is** — Postgres itself refuses to return or modify another tenant's rows, even if application code forgets a filter.

**What it does** — Enable RLS on every tenant-owned table with a policy `tenant_id = current_setting('app.tenant_id')::uuid`; a `withTenant(ctx)` hook in the TxManager runs `SET LOCAL app.tenant_id = …` at the start of each transaction (works with transaction poolers). The app connects as a role without `BYPASSRLS`; migrations and backups use a separate owner role. Platform-admin and scheduled jobs that span tenants use an explicit, audited path. Tests: queries without the GUC return 0 rows; tenant B's GUC can't see tenant A.

**Why we need it** — Defence in depth for the bug class that F-3 showed is catastrophic here; it's how League Looper is set up.

**Notes** — security audit F-3 "Defence in depth"; D5. As a migration (T139), not an edit to the baseline.

---

#### T155 — Gin → chi

**Phase**: 4 · **Category**: Backend · **Hours**: 40 · **Status**: Open · **v1 Gate**: No · **Depends on**: T129

**What it is** — The HTTP layer runs on chi and `net/http` handlers instead of Gin.

**What it does** — Write a tiny adapter (`bind`, `param`, `query`, `respondJSON`, typed context-key helpers), convert the 31 handlers and 13 middleware mechanically (`func(http.Handler) http.Handler`), then routes. Swag annotations carry over. Gin is imported by 66 non-test files today.

**Why we need it** — LL parity (D2) and no Gin-specific traps (`ShouldBindJSON` ignoring Content-Type, `ClientIP` proxy trust). Not gating: the security fixes land on Gin first.

**Notes** — correctness audit §8 coupling counts and estimate (4–6 days; ~30% less after T129).

---

#### T156 — GORM → pgx/sqlc, one repository at a time

**Phase**: 4 · **Category**: Backend · **Hours**: 120 · **Status**: Open · **v1 Gate**: No · **Depends on**: T153, T166

**What it is** — Repositories use pgx with sqlc-generated queries instead of GORM, ported incrementally behind the T153 interfaces.

**What it does** — Order: Tenant, Permission, Membership → Customer, Job → Invoice, Quote and the reports (where the 111 `Preload` calls live). Reproduce soft-delete scoping explicitly in every query; move the logic in GORM hooks (13 model files) into services or DB sequences first; money via the T166 decision (shopspring through `pgx-shopspring-decimal`, or integer cents); JSONB `Value`/`Scan` on `TenantSettings` already exists.

**Why we need it** — GORM hides the bugs the audit found (association saves resetting FKs, `default:true` flips, discarded errors) and blocks LL parity (D2).

**Notes** — correctness audit §8 (3–5 weeks estimate, sequencing step 4). Keep pgx at ≥ v5.9.2 (T117).

---

#### T157 — Drop GORM and sqlite; backend tests on Postgres via testcontainers

**Phase**: 4 · **Category**: Testing · **Hours**: 16 · **Status**: Open · **v1 Gate**: No · **Depends on**: T156

**What it is** — No GORM or sqlite left in the module, and backend tests run against real Postgres, including end-to-end tests of the critical workflows.

**What it does** — Remove `gorm.io/*`, `gorm/driver/sqlite` (test-only, used with AutoMigrate) and the ~902 `gorm:"…"` struct tags. Switch test setup to testcontainers Postgres with the T139 migrations. Add backend integration tests for: register/invite → verify → login; customer → job → complete; quote → accept → job → invoice → paid (incl. Stripe webhook); team invite → accept → permission check; email sends via the mock provider.

**Why we need it** — sqlite + AutoMigrate tests can't catch the Postgres-only bugs the audit found (missing functions, enum NOT NULL, ambiguous columns).

**Notes** — Folds in old P1 "Integration Tests for Critical Workflows" (backend list), old P2 "Email Service Tests" and "Stripe/Payment Flow Tests", and drops "Target 80% Backend Test Coverage" as a metric in favour of these flows. correctness audit §8 step 5.

---

#### T158 — Supabase Auth: own sign-in screens, JWKS verification, users.auth_user_id, invite flow

**Phase**: 4 · **Category**: Auth · **Hours**: 40 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T202, T153

**What it is** — Authentication moves to Supabase Auth while ServicePro keeps its own `users` table (keyed by `auth_user_id`), its own sign-in screens and its invite-only rules.

**What it does** — Backend: verify Supabase JWTs via JWKS in the auth middleware; add `users.auth_user_id` and provision the `users` row on first authenticated request — only if a matching invitation/sign-up invite exists (carries T106's rule); delete `pkg/auth` JWT issuance, password reset/verification services, Redis tokens and lockout (T110/T111 retire). Frontend: Login, Register, Forgot/Reset, Verify, AcceptInvitation pages on `@supabase/supabase-js`, Bearer token to the API, authStore + interceptor + ProtectedRoute rewritten. Google sign-in (supersedes T094) and TOTP MFA (T092) come from Supabase. Branded auth email templates.

**Why we need it** — D2: one auth system across the fleet, no in-house password and token code to keep secure, and Google sign-in/MFA for free.

**Notes** — frontend audit §6 "Supabase Auth client" (3–5 days frontend; supabase-js stores the session in `localStorage` by default, a regression vs httpOnly cookies — decide knowingly or use the cookie helpers). LL `docs/auth-migration.md` is the reference. 2026-10-18: the database went to Neon (T202), so whether this stays Supabase Auth is open again (OD-9). Backups must then include `auth.*` (T151).

---

#### T159 — Email → Resend only (remove SES/SNS/CloudWatch and the AWS email SDKs)

**Phase**: 4 · **Category**: Email · **Hours**: 6 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T113

**What it is** — Resend is the only email provider (Mailpit/SMTP for local dev), with bounce/complaint webhooks and retries.

**What it does** — Delete `pkg/clients/email/ses`, `pkg/clients/sms/sns`, the CloudWatch logging/metrics providers and the `aws-sdk-go-v2/service/{ses,sns,cloudwatch,cloudwatchlogs}` modules (S3 SDK stays for R2). Replace the SES bounce/notification routes (`routes/v1/auth_routes.go:29-30`) with a signed Resend webhook (`email.bounced`, `email.complained`) that marks addresses undeliverable. Retry transient failures (429/5xx) with capped exponential backoff and jitter; fail fast on invalid addresses.

**Why we need it** — SES is configured by accident (default env) and unused; one provider means one set of webhooks, one domain setup (T206) and fewer vulnerable modules.

**Notes** — correctness audit §2 "Resend migration notes", §5.5 step 3. Folds in old P1 "Email Retry Logic with Exponential Backoff"; the old "Email Client Enhancements" (SES) section is dropped.

---

#### T160 — Storage → Cloudflare R2 (fix presign host rewrite)

**Phase**: 4 · **Category**: Infra · **Hours**: 4 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T113

**What it is** — Files (PDFs, avatars, job photos, exports) live in R2, and presigned links work.

**What it does** — Configure `S3_COMPATIBLE_ENDPOINT=https://<acct>.r2.cloudflarestorage.com`, region `auto`, bucket and keys (override already at `s3.go:105-108`). Fix `s3.go:328-329`, which rewrites the host of presigned URLs to `PublicURL` — SigV4 signs the Host header, so the links fail with `SignatureDoesNotMatch`; only rewrite behind `S3_COMPATIBLE_PRESIGN_REWRITE` for local MinIO, and use `PublicURL` only for genuinely public objects. Fix `.env.example:121,132`, which sets both. Enforce content-type allow-list and size limits on upload.

**Why we need it** — Following `.env.example` today breaks every invoice and quote PDF link (`document_pdf_service.go:250`).

**Notes** — correctness audit §2 "R2 migration notes". D2. Drops the old "Storage Client Enhancements" section except MIME/size validation, folded here.

---

#### T161 — Production Redis: Upstash for rate limits/lockout/cache, or move them to Postgres

**Phase**: 4 · **Category**: Infra · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T142

**What it is** — A decided, working production setup for whatever still uses Redis after T142.

**What it does** — Per OD-3: either provision Upstash (`rediss://`, free tier) and set `REDIS_URL` on Cloud Run, or move the remaining uses (rate limits, permission cache) to Postgres/in-memory and remove Redis from production. Document the choice in `docs/ops/cloud-run-setup.md` and Fleet.

**Why we need it** — Per-instance in-memory rate limits are only acceptable with a small `--max-instances`; the choice affects cost and the Fleet entry.

**Notes** — infra audit §8 decision 1. Folds in old P2 "Redis Caching for Frequent Queries" — not needed at current scale; add caching only for a measured hot query.

---

#### T162 — Frontend server state → TanStack Query v5 (~26 pages)

**Phase**: 4 · **Category**: Frontend · **Hours**: 48 · **Status**: Open · **v1 Gate**: No · **Depends on**: T137

**What it is** — All server data on live pages comes from `useQuery`/`useMutation` with tenant-scoped key factories; Zustand holds only UI and session state.

**What it does** — Build `queries/` per resource with keys like `[tenantId, 'customers', filters]` (1 d); convert list pages — Customers, Jobs, Quotes, Invoices, Reports (2–3 d); detail/edit pages with mutations and invalidation (2–3 d); move server data out of `tenantStore`, `billingStore`, `membershipStore` (1 d). No persister.

**Why we need it** — Fixes errors-as-empty-states, double requests and out-of-order responses (3.5, 3.7) at the root and matches LL (D2).

**Notes** — frontend audit §6 (5–8 days). Today React Query is installed but used almost only by dead code. 2026-10-18: when a page moves, delete its setState-in-useEffect loader; once the last one goes, turn `react-hooks/set-state-in-effect` back on in `frontend/eslint.config.mjs` (off since T164: 41 violations in 28 files).

---

#### T163 — Tailwind v4 + semantic theme tokens + dark mode

**Phase**: 4 · **Category**: Frontend · **Hours**: 32 · **Status**: Open · **v1 Gate**: No · **Depends on**: T137, T164

**What it is** — Styling uses Tailwind v4 with ~15 semantic CSS-variable tokens defined in one file, switching light/dark from `prefers-color-scheme` or `[data-theme]`.

**What it does** — Run `npx @tailwindcss/upgrade` and fix renamed utilities (`shadow-sm`→`shadow-xs`, ring width, `currentColor` default border vs the `* { @apply border-neutral-200 }` base rule). Define tokens (`--color-surface`, `--color-fg`, `--color-muted`, `--color-border`, `--color-accent`, status colours) in `@theme inline`; codemod the ~2,820 hard palette classes (`neutral-*` 1,036, `gray-*` 834, `blue-*` 352, `primary-*` 349); manual review of ~120 component files; dark QA. Delete `theme/colors.ts` and the `tailwind.config.js` palette.

**Why we need it** — One-file theming like LL, consistent greys (two are mixed today), and dark mode (0 `dark:` classes today).

**Notes** — frontend audit §6 (0.5–1 d upgrade + 4–6 d tokens). Folds in old P3 "Standardize Component Patterns". 2026-10-18: the engine half is done — Tailwind 4.3 with CSS-first `@theme` in `src/index.css`, `@tailwindcss/vite`, PostCSS/autoprefixer removed, v3-compat base CSS (border/ring/placeholder/cursor), `space-y` margin fixes; login/register pixel-identical (c92d925). What remains is the ~15 semantic tokens and dark mode.

---

#### T165 — Better Stack uptime/status + PostHog server-side events parity

**Phase**: 4 · **Category**: Monitoring · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T204

**What it is** — External uptime checks on the user path and the API, a public status page, and server-side PostHog events for the business milestones.

**What it does** — Better Stack monitors on `https://servicepro.sharkfins.xyz/version` (edge path) and the run.app `/health/ready`; status page (published in T208). PostHog server events from `posthog-go` (today only `payment_received`, `routes.go:155-163`): `tenant_created`, `invoice_sent`, `invoice_paid`, `quote_accepted`, `job_completed`, `connect_onboarded`, keyed by tenant.

**Why we need it** — Frontend-only analytics miss webhook-driven events (payments), and nothing tells the owner the API is down.

**Notes** — D2 (Better Stack, PostHog). LL T371–T376 observability plan is the reference.

---

#### T166 — Decide money representation (decimal vs integer cents) before the pgx port

**Phase**: 4 · **Category**: Research · **Hours**: 2 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — A recorded decision (new D-item) on how money is stored and passed — shopspring decimal as today, or integer cents (the Travelify rule) — with a conversion plan.

**What it does** — Inventory today's state: `shopspring/decimal` + `DECIMAL(12,2)` with `ConvertDecimalToCents` for Stripe; Job/JobMaterial money as `float64` in Go vs DECIMAL in SQL (`models/job.go:65-68,144-147`); ten frontend `formatCurrency` helpers mixing cents and dollars. Weigh: sqlc type mapping, Stripe's integer minor units, multi-currency exponents, tax rounding rules. Write the decision into this file's Architecture decisions and the plan into T156.

**Why we need it** — sqlc generates types from the schema, so this must be settled before T156 ports the money tables.

**Notes** — OD-5; D6. correctness audit 7.M14; frontend audit 3.4.

---

#### T248 — Upgrade @sentry/react to 11 with an explicit data-collection config

**Phase**: 4 · **Category**: Monitoring · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — The frontend is on the current Sentry SDK, and what it collects is a written decision instead of the SDK default.

**What it does** — Sentry 11 turns data collection on by default and replaces `sendDefaultPii` with `dataCollection`. Bump `@sentry/react` (and `@sentry/vite-plugin` if used), set `dataCollection` so no emails, tokens, cookies, request bodies or query strings are sent (the existing `beforeSend`/URL scrubbing stays), and add a unit test that a captured event carries none of them. Pair with T246 (lazy-load and trim Sentry).

**Why we need it** — Without it the SDK falls behind on fixes, and a plain bump would start sending data CLAUDE.md forbids sending to analytics.

**Notes** — Held back from the T164 upgrade wave (2026-10-18) on purpose.

---

#### T249 — Drop Vite's legacy.inconsistentCjsInterop flag and ready the config for Vite's native loader

**Phase**: 4 · **Category**: Frontend · **Hours**: 2 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — The Vite 8 build no longer relies on the legacy CommonJS default-import behaviour, and `vite.config.ts` loads under Vite's native config loader.

**What it does** — Find the CommonJS default imports that broke without the flag (`react-big-calendar/lib/addons/dragAndDrop` gave `L.default is not a function` in the Jobs and calendar chunks), import them the Vite 8 way, then delete `legacy.inconsistentCjsInterop` from `vite.config.ts`. Replace `__dirname` with `import.meta.dirname` and add extensions to relative imports in the config. Verify with a production preview of the calendar pages.

**Why we need it** — The flag is a migration crutch Vite will remove; when it goes, the calendar breaks only in production builds.

**Notes** — Follow-up from the Vite 8 upgrade (T164, 2026-10-18).

---

### Phase 5 — Core Workflow & Quoting (T028, T029, T030, T033, T035, T041, T042, T051, T052, T058, T060, T062, T076, T080, T082, T167, T168, T235, T236, T240)

The quote → job → invoice loop, roles for field and office staff, recurring agreements and the price book. **20 open tasks, ~86h.**

- [ ] T028 — Add "Convert Quote to Job" button
- [ ] T029 — Add "Generate Invoice from Job" button
- [ ] T030 — Auto-populate invoice from job data
- [ ] T033 — Add "Jobs Needing Invoice" dashboard widget
- [ ] T035 — Create Technician role
- [ ] T041 — Implement quote auto follow-ups
- [ ] T042 — Build follow-up queue UI
- [ ] T051 — Implement quote follow-up email sequence
- [ ] T052 — Create Dispatcher role
- [ ] T058 — Add required fields validation by job stage
- [ ] T060 — Create quote template system (good/better/best)
- [ ] T062 — Add quote aging tracking
- [ ] T076 — Add "Recreate Last Quote" button
- [ ] T080 — Create terms & conditions templates
- [ ] T082 — Add "Create Quote from Job" button
- [ ] T167 — Recurring service agreements UI wired to the recurrence engine
- [ ] T168 — Price book (services and materials catalog with flat-rate pricing)
- [ ] T235 — Tenant settings UI for currency, number prefixes, default payment term and tax rate; validate on save
- [ ] T236 — Currency-aware money display in the UI, emails and PDFs
- [ ] T240 — Forms show the API's per-field validation errors

#### T028 — Add "Convert Quote to Job" button

**Phase**: 5 · **Category**: Feature · **Hours**: 3 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T224

**What it is** — One-click conversion from an accepted quote to a job ready to schedule.

**What it does** — Button on `QuoteDetailPage` only when status is `accepted`; creates a job with the quote's customer (and site, once T186 exists), service type and line items; links `jobs.quote_id`; navigates to the job detail or scheduling view. A quote converts once (button becomes "View job").

**Why we need it** — The core loop — quote accepted → job scheduled → work done → invoice sent — is what every competitor has; the office must schedule the work the moment the customer says yes.

**Notes** — Old task (Feb 2026 TODO P0 "Quoting – Critical"). Acceptance: button visible only on accepted quotes; job carries all relevant quote data; user can schedule immediately. Correct quote states depend on T224 (accept race, customer edits reverting). `docs/product/competitive-research-2026-10.md` §2 row 6, §5 MVP #2.

---

#### T029 — Add "Generate Invoice from Job" button

**Phase**: 5 · **Category**: Feature · **Hours**: 2 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T225, T124

**What it is** — One-click invoice generation from a completed job.

**What it does** — Button on `JobDetailPage` when status is `completed` and the job has no invoice; creates the invoice pre-populated per T030, links it to the job, moves the job to `invoiced` (T124), and opens the invoice for review/send.

**Why we need it** — Techs complete work and the office forgets to bill; completed-but-unbilled work is the #1 cash-flow leak for small service businesses.

**Notes** — Old task (Feb 2026 TODO P0 "Invoicing – Critical"). Acceptance: visible only on completed, uninvoiced jobs; invoice linked to job; user lands on the invoice. Must use T225's double-invoicing guard (correctness audit 7.M5: today `invoice_handler.go:302-303` accepts any `job_id`).

---

#### T030 — Auto-populate invoice from job data

**Phase**: 5 · **Category**: Feature · **Hours**: 4 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T029

**What it is** — The invoice generated from a job already contains the labor, materials, notes and photo links the tech recorded.

**What it does** — Map `job.materials` → invoice line items; labor from actual time (`ActualStartAt`/`ActualEndAt` today, the T177 time clock later) × the tenant's labor rate or price-book item (T168); job notes into the invoice description; photos (T174) linked from the invoice. Totals computed by the server (T225).

**Why we need it** — Eliminates re-typing ("2 hours labor, replaced capacitor $45, customer approved additional work") and makes sure nothing done in the field goes unbilled.

**Notes** — Old task (Feb 2026 TODO P0). Acceptance: all job materials appear as lines; labor from actual time; notes/photos reachable from the invoice. Job money is `float64` in Go (correctness audit 7.M14) — convert via decimal until T166 decides.

---

#### T033 — Add "Jobs Needing Invoice" dashboard widget

**Phase**: 5 · **Category**: Reporting · **Hours**: 2 · **Status**: Open · **v1 Gate**: No · **Depends on**: T029

**What it is** — A dashboard count and list of completed jobs without an invoice.

**What it does** — Query jobs where `status = completed` and no linked invoice (tenant-scoped, T102); widget with count, click-through to the filtered job list, and a quick "Generate invoice" action (T029).

**Why we need it** — "You have 7 jobs to bill!" — money left on the table is the #1 cash-flow killer for small service businesses.

**Notes** — Old task (Feb 2026 TODO P0 "Dashboard – Critical"). Acceptance: accurate count, one-click navigation, prominent placement. T047 ("stuck work") reuses the query.

---

#### T035 — Create Technician role

**Phase**: 5 · **Category**: Auth · **Hours**: 4 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T104

**What it is** — A role for field workers that sees and updates only their own assigned jobs and never sees customer financials.

**What it does** — Follow CLAUDE.md's permission steps (`permissions/constants.go`, `AllPermissions()`, `DefaultRolePermissions()`, `PermissionsByResource()`, seed grants in `001_schema.sql` or a T139 migration). Include: view assigned jobs, update job status, add notes/photos, log time/materials. Exclude: other techs' jobs, customer financial data, create/edit customers, reports. "Assigned jobs only" needs a row filter in the job repository (assignment-based), not just a permission flag; hide financial fields in job/customer responses for this role (full field-level controls are T075).

**Why we need it** — Techs need their jobs, notes, photos and status buttons in the field (T173), but owners won't hand them "customer owes $5,000".

**Notes** — Old task (Feb 2026 TODO P0 "Roles – Critical"). Acceptance: role created with correct permissions; financial data hidden; tech sees only their assignments. Per-tenant permission evaluation (T104) must land first or the role leaks across tenants.

---

#### T041 — Implement quote auto follow-ups

**Phase**: 5 · **Category**: Feature · **Hours**: 6 · **Status**: Open · **v1 Gate**: No · **Depends on**: T141, T159

**What it is** — Automated follow-up messages for sent quotes that haven't been answered.

**What it does** — A scheduled job (T141) finds `sent` quotes by age and sends follow-ups at configurable intervals (e.g. day 3, 7, 14 "expiring soon"); records each send; stops on accept/decline/expiry.

**Why we need it** — Consistent follow-up lifts close rates (the old TODO cited 20–30%).

**Notes** — Old task (Feb 2026 TODO P1). Acceptance: automatic, configurable timing, none after response, history kept. T051 defines the sequence content; consider merging the two when building.

---

#### T042 — Build follow-up queue UI

**Phase**: 5 · **Category**: UX · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T062

**What it is** — A list of quotes needing a manual follow-up today.

**What it does** — Pending quotes sorted by age/value/customer with last contact date; quick actions to log a call or email (T036) which updates last contact.

**Why we need it** — "Call these 5 customers today" — an organised view for sales follow-up.

**Notes** — Old task (Feb 2026 TODO P1). Acceptance: shows all pending quotes, sortable/filterable, logs follow-up actions. T046 is the dashboard widget for it.

---

#### T051 — Implement quote follow-up email sequence

**Phase**: 5 · **Category**: Email · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T041, T072

**What it is** — The content and rules of the multi-touch quote follow-up sequence.

**What it does** — Templates per touch (day 3 check-in, day 7 address concerns, day 14 expiry warning) using T072/T073 templates and variables; stops when the quote is resolved.

**Why we need it** — Automated nurturing of pending quotes without staff effort.

**Notes** — Old task (Feb 2026 TODO P1). Acceptance: runs automatically, customizable templates, stops when resolved. Overlaps T041 (engine) — merge if simpler.

---

#### T052 — Create Dispatcher role

**Phase**: 5 · **Category**: Auth · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T104

**What it is** — A role for office staff who schedule and assign but don't see money.

**What it does** — Include: view all jobs, create/edit schedules, assign technicians, view customer contact info. Exclude: invoice amounts, payment history, financial reports, pricing changes. Same CLAUDE.md permission steps as T035.

**Why we need it** — Dispatchers need the whole schedule but not financial visibility.

**Notes** — Old task (Feb 2026 TODO P1 "Roles – High Priority"). Acceptance: role created; financial data hidden from dispatchers.

---

#### T058 — Add required fields validation by job stage

**Phase**: 5 · **Category**: Feature · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T124

**What it is** — Configurable data that must exist before a job can move to a given status (e.g. completion requires notes, photos and time logged).

**What it does** — Per-tenant rules keyed by target status, evaluated inside the T124 transition; clear error listing what's missing; T175 checklists can be one of the required items.

**Why we need it** — Stops techs marking jobs complete without the documentation commercial customers expect as proof of service.

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: configurable required fields per status; validation on status change; clear messages. Original confidence: medium — exact requirements per stage unclear; validate in T218 interviews.

---

#### T060 — Create quote template system (good/better/best)

**Phase**: 5 · **Category**: Feature · **Hours**: 8 · **Status**: Open · **v1 Gate**: No · **Depends on**: T168

**What it is** — Reusable quote templates with tiered options the customer chooses between.

**What it does** — Templates hold line items (from the price book, T168), terms (T080) and optional tiers ("Basic tune-up $149 / Premium $249 / Complete $399"); one-click apply to a new quote; the public approval page (T183) lets the customer pick a tier.

**Why we need it** — Upsell pattern that lifts average ticket; competitors (HCP, Jobber optional line items) all have it.

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: create/edit templates with line items; one-click apply. The old "Orphaned Features → Quote Template System" spec (variable substitution, categories, versioning, import/export) and the unreachable `frontend/src/components/templates/*` (deleted in T137) are reference only — build the smaller version.

---

#### T062 — Add quote aging tracking

**Phase**: 5 · **Category**: Feature · **Hours**: 2 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Every pending quote shows how long it has been waiting.

**What it does** — Days since `sent_at` shown on the quote list, sortable, filterable by age range.

**Why we need it** — Prioritises follow-ups; feeds T045 and T042.

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: age displayed, sortable, filter by range.

---

#### T076 — Add "Recreate Last Quote" button

**Phase**: 5 · **Category**: Feature · **Hours**: 1 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — One click on the customer page clones that customer's most recent quote.

**What it does** — Customer detail button calling the existing clone-quote action (shipped 2025-01-30) on the latest quote; opens the new draft.

**Why we need it** — Customers call back months later wanting the same work.

**Notes** — Old task (Feb 2026 TODO P3).

---

#### T080 — Create terms & conditions templates

**Phase**: 5 · **Category**: Feature · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Reusable legal text blocks selectable on quotes and invoices.

**What it does** — Tenant-level CRUD for T&C blocks; default per document type; chosen block rendered on PDFs and the public approval page (T183).

**Why we need it** — Consistent terms across documents; commercial customers expect them.

**Notes** — Old task (Feb 2026 TODO P3). Invoices already carry a T&C field (`docs/product/competitive-research-2026-10.md` §0) — this adds reusable templates.

---

#### T082 — Add "Create Quote from Job" button

**Phase**: 5 · **Category**: Feature · **Hours**: 2 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Generate a new quote from a job, e.g. "how much to replace the whole unit?" after a repair visit.

**What it does** — Job detail action creating a draft quote with the job's customer/site and optionally its materials as starting lines.

**Why we need it** — Turns service visits into replacement or upsell quotes.

**Notes** — Old task (Feb 2026 TODO P3; original confidence medium — less common workflow).

---

#### T167 — Recurring service agreements UI wired to the recurrence engine

**Phase**: 5 · **Category**: Feature · **Hours**: 12 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T141, T102

**What it is** — Customers can have service agreements (e.g. "nightly cleaning Mon–Fri", "quarterly HVAC PM") that generate jobs automatically, managed from the UI.

**What it does** — Agreement model: customer (+ site, T186), cadence, line items/price, start/end, assigned tech, auto-invoice option. A scheduled job (T141) generates the next occurrences idempotently (unique per agreement+date). Decide between reviving `backend/internal/services/recurring` (never linked into the binary; known bugs: Jan 31 + 1 month skips February, time of day treated as UTC and not DST-aware, occurrence cap counted per window not per series, nil `JobTemplateID` panic — `recurring/generator.go:286,373-376,180-182,380`) or rewriting it small. The old frontend `components/recurring/` was deleted in T018.

**Why we need it** — Commercial cleaning and maintenance work is recurring by nature; recurring schedules are half of the lock-in moat.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 8, §4.2, §5 MVP #7; correctness audit 5.1, 7.L5. Autopay agreements are T191.

---

#### T168 — Price book (services and materials catalog with flat-rate pricing)

**Phase**: 5 · **Category**: Feature · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — A per-tenant catalog of services and materials with unit, cost, flat-rate price and tax flag, picked into quotes, jobs and invoices.

**What it does** — Catalog CRUD + CSV import; line-item pickers on quotes/invoices/job materials; cost stored for job costing (T198); categories; inactive items hidden but kept for history.

**Why we need it** — Consistent pricing across staff, faster quoting, and the substrate for good/better/best (T060) and AI quote drafting (T215).

**Notes** — `docs/product/competitive-research-2026-10.md` §3 "Price book / flat-rate pricing".

---

#### T235 — Tenant settings UI for currency, number prefixes, default payment term and tax rate; validate on save

**Phase**: 5 · **Category**: Feature · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T127, T237

**What it is** — Owners can set the business settings the backend now honours, and invalid values are rejected when saved.

**What it does** — Add controls to Org Settings for currency (ISO 4217 list), invoice/quote/job prefixes (≤20 chars, no control characters), default payment term and default tax rate (tenant-scoped pickers), a tax-rates management screen on top of the T237 CRUD API (`tax_rates.manage`), and date format (or remove it — nothing reads it). Backend: validate the tenant settings payload on save with field errors (today it stores anything and sanitises on read). Tests both sides.

**Why we need it** — T127 made the backend read these settings, but there is no UI to set them, so every tenant is stuck on USD / INV- / no defaults.

**Notes** — Follow-up from the T127 report (2026-10-16).

---

#### T236 — Currency-aware money display in the UI, emails and PDFs

**Phase**: 5 · **Category**: Feature · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T127

**What it is** — Amounts show in the document's currency everywhere, not a hard-coded "$".

**What it does** — Use the invoice/quote `currency` field (T127) in `frontend/src/utils/money.ts` formatting (Intl.NumberFormat with the currency; respect zero/three-decimal currencies), email templates and the invoice/quote PDFs.

**Why we need it** — A non-USD tenant's customers would see dollar signs on their invoices.

**Notes** — Follow-up from the T127 report (2026-10-16). Low priority while the beachhead is US contractors.

---

#### T240 — Forms show the API's per-field validation errors

**Phase**: 5 · **Category**: UX · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T223

**What it is** — When the API rejects a form, each problem appears next to its input.

**What it does** — T223's envelope carries `fields: {name: problem}` on validation errors and the frontend has `getFieldErrors`; wire it into the customer, job, quote, invoice, payment and settings forms (map JSON field names to inputs), keeping the summary message.

**Why we need it** — Users currently see one generic "validation failed" message and have to guess which field is wrong.

**Notes** — Follow-up from T223 (2026-10-16).

---

### Phase 6 — Payments to the Business (T061, T063, T083, T169, T170, T171, T172)

Stripe Connect so tenants actually receive the money, ACH for B2B, deposits, autopay and in-person payments. **7 open tasks, ~72h.**

- [ ] T061 — Add required deposit to accept quote
- [ ] T063 — Add deposit tracking and application
- [ ] T083 — Implement late fee calculation
- [ ] T169 — Stripe Connect Express onboarding; invoice payments routed to the tenant
- [ ] T170 — Card + ACH invoice payments, ACH preferred for B2B
- [ ] T171 — Saved payment methods + autopay
- [ ] T172 — Tap to Pay via Stripe Terminal

#### T061 — Add required deposit to accept quote

**Phase**: 6 · **Category**: Payments · **Hours**: 6 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T169, T183

**What it is** — A quote can require a deposit (fixed amount or percentage), paid by the customer as part of accepting it.

**What it does** — Deposit settings on the quote/template; the public approval page (T183) collects payment via the tenant's connected account before the quote becomes `accepted`; deposit recorded separately (T063).

**Why we need it** — Common for big jobs ("pay $200 to confirm") and secures commitment before materials are ordered.

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: configurable amount/percentage; acceptance triggers payment; deposit tracked separately.

---

#### T063 — Add deposit tracking and application

**Phase**: 6 · **Category**: Payments · **Hours**: 6 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T061, T118

**What it is** — Deposits are tracked on the customer/quote and applied to the final invoice automatically.

**What it does** — Deposit ledger entries; when the job's invoice is generated (T029) the deposit appears as a credit ("$2,000 total, $500 deposit, $1,500 due"); refund path via T125.

**Why we need it** — Correct accounting for deposits and a clear invoice for the customer.

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: tracked separately; applied automatically; clear on the invoice.

---

#### T083 — Implement late fee calculation

**Phase**: 6 · **Category**: Payments · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T141

**What it is** — Automatic late fees on overdue invoices according to tenant rules.

**What it does** — Per-tenant rule (flat or % after N days past due, cap, grace period); a scheduled job (T141) adds a late-fee line once per period; reminder templates mention it; can be waived.

**Why we need it** — Enforces payment terms; part of the collections suite.

**Notes** — Old task (Feb 2026 TODO P3; rules vary by business and jurisdiction — default off). The old "Payment Notification System" spec listed `late_fee_applied` notifications; reminder templates cover that.

---

#### T169 — Stripe Connect Express onboarding; invoice payments routed to the tenant

**Phase**: 6 · **Category**: Payments · **Hours**: 24 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T105, T109, T118, T119

**What it is** — Each tenant onboards a Stripe Connect Express account, and their customers' invoice payments go to that account, with ServicePro's platform fee taken automatically.

**What it does** — Onboarding via Account Links from org settings; `tenants.stripe_account_id` + capability status; `account.updated` webhook on a Connect endpoint. Invoice Checkout/PaymentIntents created for the connected account per the charge-model decision (OD-4: direct charges make the tenant merchant of record and liable for disputes; destination charges keep liability on the platform). `application_fee_amount` = platform fee (0.4% default, configurable per plan, T210). Replace the platform-account Checkout in `invoice_payment_service.go`. Payouts, disputes and refunds visible in the tenant's Express dashboard.

**Why we need it** — Today invoice payments land in the platform's Stripe account — no tenant can actually get paid. It blocks launch, and holding the money flow is the core of the moat and the business model (platform fee).

**Notes** — `docs/product/competitive-research-2026-10.md` §0, §2 row 9, §4.2 (1, 4), §5 MVP #1. Use the `stripe:connect-recommend` / `stripe-best-practices` skills when designing. Supersedes the old "Removed Features → Payment System" restore list.

---

#### T170 — Card + ACH invoice payments, ACH preferred for B2B

**Phase**: 6 · **Category**: Payments · **Hours**: 6 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T169

**What it is** — Customers can pay invoices by card or US bank account (ACH), with ACH presented first for business customers.

**What it does** — Enable `us_bank_account` alongside card on the connected account; handle ACH's async settlement (`payment_intent.processing` → `succeeded`/`payment_failed`) with a "payment pending" invoice state that reminders respect; optional lower platform fee on ACH; surcharge pass-through deferred (state-by-state legality).

**Why we need it** — B2B customers pay large invoices by ACH; card fees on a $5k invoice are a reason to keep paying by check.

**Notes** — `docs/product/competitive-research-2026-10.md` §4.5 "Payments" (lower ACH fee to push B2B to ACH).

---

#### T171 — Saved payment methods + autopay

**Phase**: 6 · **Category**: Payments · **Hours**: 10 · **Status**: Open · **v1 Gate**: No · **Depends on**: T169, T170

**What it is** — End customers can save a card or bank account with the tenant and opt into automatic payment of invoices on their due date.

**What it does** — Stripe Customer per end customer on the connected account; SetupIntent (with ACH mandate); autopay flag per customer/agreement; a scheduled job (T141) charges due invoices and handles failures (notify, retry once, fall back to reminders).

**Why we need it** — Stored payment methods are a major switching cost and the basis for maintenance-agreement autopay (T191).

**Notes** — `docs/product/competitive-research-2026-10.md` §4.2 (1). The dead `components/payments/methods/AddPaymentMethod.tsx` (raw `fetch`, frontend audit — deleted in T137) is not the base; build on Stripe Elements.

---

#### T172 — Tap to Pay via Stripe Terminal

**Phase**: 6 · **Category**: Payments · **Hours**: 16 · **Status**: Open · **v1 Gate**: No · **Depends on**: T180, T169

**What it is** — Techs collect in-person card payments on their phone with Tap to Pay on iPhone/Android.

**What it does** — Stripe Terminal SDK in the Capacitor app (T180); connection-token endpoint; a Terminal location per tenant; payment recorded against the invoice like any other (T118).

**Why we need it** — In-person collection at job completion; competitors offer it.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 10. Needs the native app — judgment call: the brief said "depends on store app T181", but the store app is T180.

---

### Phase 7 — Dispatch & Field (T027, T038, T039, T040, T056, T057, T059, T077, T078, T079, T099, T173, T174, T175, T176, T177, T178, T179, T180)

The technician experience (the product's weakest area) and the remaining scheduling features. **19 open tasks, ~182h.**

- [ ] T027 — Add drag-and-drop rescheduling to job calendar
- [ ] T038 — Add technician availability management UI
- [ ] T039 — Create emergency job insertion workflow
- [ ] T040 — Add auto-notify on emergency insertion
- [ ] T056 — Add workload capacity warnings
- [ ] T057 — Add double-booking override with reason
- [ ] T059 — Add technician skill tags
- [ ] T077 — Add time-off request UI
- [ ] T078 — Add skill-based job matching suggestions
- [ ] T079 — Add travel time notes field
- [ ] T099 — Add tenant timezone support for conflict detection
- [ ] T173 — Technician "My Day" mobile-first view
- [ ] T174 — Job photos and attachments (R2)
- [ ] T175 — Job checklists / forms
- [ ] T176 — Customer signature on completion + service report PDF emailed
- [ ] T177 — Time clock per job + timesheet export
- [ ] T178 — Map / route view of the day
- [ ] T179 — Offline-capable tech app (queue + sync)
- [ ] T180 — Capacitor-wrapped iOS/Android apps in the stores

#### T027 — Add drag-and-drop rescheduling to job calendar

**Phase**: 7 · **Category**: Feature · **Hours**: 8 · **Status**: Open · **v1 Gate**: No · **Depends on**: T099

**What it is** — Drag a job to a new slot (or resize it) on the calendar to reschedule, with conflict warnings before saving.

**What it does** — Re-enable `onEventDrop`/`onEventResize` in `frontend/src/components/calendar/Calendar.tsx` (commented out since T019) and add handlers in `pages/Jobs/JobCalendarPage.tsx`; run the conflict check (`POST /v1/conflicts/check`, T020) before confirming; call `jobService.updateJob()` with new `scheduled_start_at`/`scheduled_end_at`; confirmation or undo; loading state; toasts.

**Why we need it** — Quick rescheduling is the daily driver for office staff.

**Notes** — Old task (Feb 2026 TODO P2; depended on T020, done). Acceptance: drag updates schedule; resize adjusts duration; conflict warnings first; confirm/undo; loading state. Conflict detection currently can't see real jobs (nothing writes `schedules`, correctness audit 7.M10) — fixed in T099. jsdom DnD tests need T136's `elementFromPoint` stub. Use typed `withDragAndDrop<Event>` instead of the `any` cast at `Calendar.tsx:61`.

---

#### T038 — Add technician availability management UI

**Phase**: 7 · **Category**: Feature · **Hours**: 6 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Techs (or managers) set working hours; dispatchers see availability when scheduling.

**What it does** — Weekly working hours per technician (simple pattern first; shift rotations later); shown on the calendar; conflict detection warns outside them.

**Why we need it** — "Mike works M–F 8–5, Sarah T–Sat 7–4" — dispatchers need to know who's available.

**Notes** — Old task (Feb 2026 TODO P1; confidence medium — simple hours vs shift patterns). Acceptance: techs set hours; dispatchers view them; warning outside hours. T077 adds time off.

---

#### T039 — Create emergency job insertion workflow

**Phase**: 7 · **Category**: Feature · **Hours**: 8 · **Status**: Open · **v1 Gate**: No · **Depends on**: T099

**What it is** — A flow for urgent jobs that finds the soonest slot and can bump non-emergency work.

**What it does** — "Emergency" priority; suggested slots accounting for travel time and current load; option to reschedule existing non-emergency jobs, notifying affected customers (T040).

**Why we need it** — Pipe burst, cooler down in a restaurant — the customer needs someone today.

**Notes** — Old task (Feb 2026 TODO P1; confidence medium). Acceptance: mark as emergency; system suggests slots; reschedule existing jobs with notification. Less central for the commercial beachhead than residential — validate in T218.

---

#### T040 — Add auto-notify on emergency insertion

**Phase**: 7 · **Category**: SMS · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T039, T049

**What it is** — Customers and techs affected by an emergency reshuffle are told automatically.

**What it does** — When T039 bumps a job, send the affected customer a reschedule message and the affected tech an update, on their preferred channel; option to call instead.

**Why we need it** — Keeps everyone informed when schedules change.

**Notes** — Old task (Feb 2026 TODO P1). Acceptance: affected parties notified; clear message; option to call instead.

---

#### T056 — Add workload capacity warnings

**Phase**: 7 · **Category**: Feature · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T099

**What it is** — A warning when a technician would exceed their daily capacity.

**What it does** — Configurable threshold (default 8h) per tenant or tech; warn on scheduling; override with acknowledgment. The detector's workload check uses a UTC "day" (`conflict/detector.go:291-293`) — tenant-local days come with T099.

**Why we need it** — Prevents overloading techs ("John already has 8 hours today").

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: warning over threshold; configurable; override with acknowledgment.

---

#### T057 — Add double-booking override with reason

**Phase**: 7 · **Category**: Feature · **Hours**: 2 · **Status**: Open · **v1 Gate**: No · **Depends on**: T099

**What it is** — Schedulers can deliberately override a conflict, with a required reason that's logged.

**What it does** — Override option on the conflict dialog; mandatory reason; stored and shown in job history / audit log (T093).

**Why we need it** — Sometimes two techs really are needed, or an emergency justifies it.

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: override offered on conflict; reason required; logged for audit.

---

#### T059 — Add technician skill tags

**Phase**: 7 · **Category**: Feature · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Technicians carry skill tags (HVAC, refrigeration, electrical, floor care…).

**What it does** — Tenant-defined tags; add/remove on the tech profile; filter techs by skill in assignment UI.

**Why we need it** — Basis for smart assignment (T078).

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: add/remove tags; visible on profile; filter by skill.

---

#### T077 — Add time-off request UI

**Phase**: 7 · **Category**: Feature · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T038

**What it is** — Techs request time off; managers approve; approved time off blocks scheduling.

**What it does** — Request/approve/deny flow; approved time off shown on the calendar and treated as unavailability by conflict detection.

**Why we need it** — Vacation and sick-day management.

**Notes** — Old task (Feb 2026 TODO P3; confidence medium — simple block vs approval workflow).

---

#### T078 — Add skill-based job matching suggestions

**Phase**: 7 · **Category**: Feature · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T059

**What it is** — When assigning a job, suggest techs whose skills match the job type and who are available.

**What it does** — Job types carry required skills; assignment UI ranks available techs by skill match and load.

**Why we need it** — Smart assignment recommendations.

**Notes** — Old task (Feb 2026 TODO P3; confidence medium).

---

#### T079 — Add travel time notes field

**Phase**: 7 · **Category**: Feature · **Hours**: 1 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — A field on scheduled jobs for expected travel time.

**What it does** — Number/text field on the job schedule; displayed on the calendar and My Day.

**Why we need it** — Helps scheduling accuracy until the map view (T178) gives real distances.

**Notes** — Old task (Feb 2026 TODO P3). May be superseded by T178 — drop if the map lands first.

---

#### T099 — Add tenant timezone support for conflict detection

**Phase**: 7 · **Category**: Backend · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Conflict detection and other date logic use the tenant's timezone and configured business hours, and actually see scheduled jobs.

**What it does** — Read the tenant timezone and business hours (stored by T100) once per request; convert schedule times to tenant-local before comparing; replace the hard-coded 8–18 Mon–Fri UTC check and re-enable `checkBusinessHours` (`conflict/detector.go:254`, currently unused); tenant-local day boundaries for the workload check (`:291-293`) and for days-overdue in reminders (`payment_reminder_service.go:113,148`). Make the detector see real work: nothing writes `schedules`, so technician-overlap never fires — write schedule rows on job create/update or have the detector read jobs directly; fix the containment range queries to overlap (`schedule_repository.go:64-69,111`). Default to UTC / 8–18 Mon–Fri when unset.

**Why we need it** — Business-hours validation compares UTC against hard-coded hours, producing false conflicts (1:24 PM local flagged as out of hours), and overlaps are never detected.

**Notes** — Old task (Feb 2026 TODO P1; depended on T100, now done in 3df4280). correctness audit 3.5, 7.M9, 7.M10. Acceptance: tenant timezone and hours used; no false positives inside local hours; sensible defaults. Frontend half is T133.

---

#### T173 — Technician "My Day" mobile-first view

**Phase**: 7 · **Category**: Mobile · **Hours**: 24 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T035, T124, T133

**What it is** — A phone-first screen where a technician sees today's assigned jobs and works them: directions, call, status, notes.

**What it does** — Route for the Technician role: today's jobs in time order (tenant timezone, T133), customer/site, tap-to-call, maps deep link for directions, big status buttons (en route → triggers T050, start, complete) using T124 transitions, notes, materials; photos (T174), checklist (T175) and signature (T176) slot in. Works on 375px screens.

**Why we need it** — Techs must use it in the field or owners won't buy; the field side is the product's weakest area.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 15, §5 MVP #3. Offline support is T179; store-listed app is T180.

---

#### T174 — Job photos and attachments (R2)

**Phase**: 7 · **Category**: Feature · **Hours**: 8 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T160

**What it is** — Techs and office staff attach photos and files to jobs (and quotes/invoices), stored in R2.

**What it does** — Presigned PUT uploads from the browser/app; server-side resize + thumbnail and `DecodeConfig` dimension check (security audit F-19); content-type/size allow-list; attachments listed on the job, linked from the invoice (T030) and included in the service report (T176).

**Why we need it** — Before/after photos are proof of service for commercial customers and settle disputes.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 16. Folds in old P3 "Image Optimization Strategy" (resize on upload, WebP, lazy loading).

---

#### T175 — Job checklists / forms

**Phase**: 7 · **Category**: Feature · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T173

**What it is** — Per-job-type checklists and simple forms techs complete on the job.

**What it does** — Tenant-defined templates (checkbox, text, number, photo-required, pass/fail) attached by job type or agreement; answers stored per job; required items can block completion (T058); included in the service report (T176).

**Why we need it** — Commercial cleaning and maintenance clients want proof each item was done; the basis for inspection/compliance work (T222).

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 17, §5 v1.

---

#### T176 — Customer signature on completion + service report PDF emailed

**Phase**: 7 · **Category**: Feature · **Hours**: 8 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T173, T174, T159

**What it is** — At completion the customer signs on the tech's device, and a service report PDF is emailed to the customer's contacts.

**What it does** — Signature canvas capturing name, timestamp, IP/device; gofpdf service report (work done, materials, time, photos, checklist answers, signature); emailed via Resend to the job's contact(s) and stored with the job.

**Why we need it** — Commercial customers and property managers need proof of service before they approve an invoice.

**Notes** — `docs/product/competitive-research-2026-10.md` §5 v1 "Job checklists and service report PDF".

---

#### T177 — Time clock per job + timesheet export

**Phase**: 7 · **Category**: Feature · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T173

**What it is** — Techs clock in/out per job (and for the day), and owners export timesheets.

**What it does** — Time entries per user/job with breaks; edits audited (T093); weekly timesheet view and CSV export; feeds invoice labor (T030), payroll (T196) and job costing (T198).

**Why we need it** — Payroll and job-costing basics; a common reason shops buy FSM software.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 18.

---

#### T178 — Map / route view of the day

**Phase**: 7 · **Category**: Feature · **Hours**: 8 · **Status**: Open · **v1 Gate**: No · **Depends on**: T173

**What it is** — A map of the day's jobs per technician for office and tech.

**What it does** — Geocode customer/site addresses on save; map view (Mapbox or Google) with pins coloured by status and tech; ordered route line; no optimization in v1.

**Why we need it** — Dispatchers see geography at a glance; techs see their route.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 24. Route optimization (Google Route Optimization API) is a later add-on.

---

#### T179 — Offline-capable tech app (queue + sync)

**Phase**: 7 · **Category**: Mobile · **Hours**: 40 · **Status**: Open · **v1 Gate**: No · **Depends on**: T173

**What it is** — The tech view keeps working without signal and syncs when back online.

**What it does** — Cache today's jobs in IndexedDB; queue status changes, notes, time entries, photos and signatures; replay with idempotency keys; conflict rules (server wins on schedule, client wins on notes); clear offline indicator. A deliberately built service worker — the existing never-registered one is deleted in T137.

**Why we need it** — Basements and rural sites have no signal; offline is a known Jobber complaint and a marketing point.

**Notes** — `docs/product/competitive-research-2026-10.md` §1.3 (Jobber "app doesn't work offline"), §5 v2.

---

#### T180 — Capacitor-wrapped iOS/Android apps in the stores

**Phase**: 7 · **Category**: Mobile · **Hours**: 24 · **Status**: Open · **v1 Gate**: No · **Depends on**: T173

**What it is** — ServicePro is installable from the App Store and Google Play as a Capacitor wrapper around the web app.

**What it does** — Capacitor project; native camera and push notifications; deep links; store listings, screenshots, privacy labels; Apple Developer ($99/yr) and Google Play ($25 one-time) accounts — add to Fleet costs.

**Why we need it** — Buyers search the app stores, and Tap to Pay (T172) needs a native app.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 28.

---

### Phase 8 — Customer Communications (T036, T037, T048, T049, T050, T054, T055, T072, T073, T089, T090, T091, T181, T182)

Confirmations, reminders, on-my-way, two-way SMS, templates and the customer activity record. **14 open tasks, ~75h.**

- [ ] T036 — Add manual Call Log entry
- [ ] T037 — Auto-log system messages to customer activity
- [ ] T048 — Add appointment confirmation emails
- [ ] T049 — Add appointment reminder emails/SMS
- [ ] T050 — Add "Tech on the way" manual trigger
- [ ] T054 — Create Communication Log component
- [ ] T055 — Unified Customer Timeline view
- [ ] T072 — Create message template management UI
- [ ] T073 — Add rich variable system for templates
- [ ] T089 — Add email/SMS delivery tracking
- [ ] T090 — Add communication failure alerts
- [ ] T091 — Add notification preferences per customer
- [ ] T181 — Two-way SMS inbox (Twilio + 10DLC)
- [ ] T182 — Review request automation

#### T036 — Add manual Call Log entry

**Phase**: 8 · **Category**: Feature · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Staff log phone calls with customers (date, duration, notes, outcome).

**What it does** — `call_logs` (customer, user, timestamp, duration, notes, outcome, tenant); form on customer detail; appears in the timeline (T055).

**Why we need it** — Team coordination — no two people calling the same customer, history for everyone.

**Notes** — Old task (Feb 2026 TODO P1). Acceptance: log from customer page; appears in timeline; shows who and when.

---

#### T037 — Auto-log system messages to customer activity

**Phase**: 8 · **Category**: Feature · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T096

**What it is** — Every outbound message, payment and status change is automatically logged to the customer's activity.

**What it does** — Hook the email and SMS senders, payment recording (T118) and quote/invoice/job transitions to write activity entries (via the T096 event log).

**Why we need it** — Ends "did the quote go out?" questions.

**Notes** — Old task (Feb 2026 TODO P1). Acceptance: all outbound comms, payments and status changes logged and shown in the timeline.

---

#### T048 — Add appointment confirmation emails

**Phase**: 8 · **Category**: Email · **Hours**: 3 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T159

**What it is** — An email to the customer when a job is scheduled or rescheduled.

**What it does** — Trigger on schedule set/changed; template with date, time window (tenant timezone), service, site, tech name, and a reschedule/cancel contact link; uses T072/T073 templates when they exist.

**Why we need it** — Sets expectations and reduces no-shows — the cheapest visible "wow".

**Notes** — Old task (Feb 2026 TODO P1). Acceptance: sent when scheduled; contains date, time, service; professional template. `docs/product/competitive-research-2026-10.md` §5 MVP #5. Folds in the old "Email Templates Expansion" item's job-completion and payment-received emails as sibling templates.

---

#### T049 — Add appointment reminder emails/SMS

**Phase**: 8 · **Category**: SMS · **Hours**: 4 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T141, T048

**What it is** — Automatic reminders before appointments by email and SMS.

**What it does** — Scheduled job (T141) finds upcoming jobs and sends at configurable offsets (24h, 2h), on the customer's channel, honouring opt-out (T091); claim rows so nothing double-sends. SMS provider per OD-6 (the existing TextBelt client works for one-way; SNS goes away in T159).

**Why we need it** — Reduces no-shows (the old TODO cited 25–30%).

**Notes** — Old task (Feb 2026 TODO P1). Acceptance: automatic, configurable timing, customer can opt out.

---

#### T050 — Add "Tech on the way" manual trigger

**Phase**: 8 · **Category**: SMS · **Hours**: 2 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T049

**What it is** — A button the tech taps to text the customer that they're on the way.

**What it does** — Button in My Day / job view (T173) that also moves the job to `en_route`; SMS (email fallback) with tech name and ETA; logged in job activity (T037).

**Why we need it** — Customers know to be ready; on-site contacts at commercial sites can unlock doors.

**Notes** — Old task (Feb 2026 TODO P1). Acceptance: one-click send; SMS delivered; logged in job history.

---

#### T054 — Create Communication Log component

**Phase**: 8 · **Category**: UX · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T037

**What it is** — A unified, chronological view of all emails, SMS and calls with a customer.

**What it does** — Component on customer detail listing messages and calls with delivery status (T089), linked to related records.

**Why we need it** — "Did anyone call Mrs. Smith?" in one place.

**Notes** — Old task (Feb 2026 TODO P2; confidence medium — threaded vs simple list).

---

#### T055 — Unified Customer Timeline view

**Phase**: 8 · **Category**: UX · **Hours**: 8 · **Status**: Open · **v1 Gate**: No · **Depends on**: T037

**What it is** — One scrollable timeline of a customer's jobs, quotes, invoices, payments and communications.

**What it does** — Merged feed from the T096 event log; filter by type; items link to detail pages.

**Why we need it** — At-a-glance customer history during conversations.

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: all activity in one timeline, clickable, filterable.

---

#### T072 — Create message template management UI

**Phase**: 8 · **Category**: Feature · **Hours**: 6 · **Status**: Open · **v1 Gate**: No · **Depends on**: T073

**What it is** — Tenants edit their own email/SMS templates for every automated message.

**What it does** — CRUD per message type (confirmation, reminder, on-my-way, quote follow-up, payment reminder, receipt); categories; preview with sample data; reset to default.

**Why we need it** — Customizable, on-brand communications.

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: create/edit/delete, categories, preview. The unlinked `templates/email` engine (correctness audit 5.1) is deleted in T129; build on whatever T159 uses.

---

#### T073 — Add rich variable system for templates

**Phase**: 8 · **Category**: Feature · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Templates use documented variables like `{{customer_first_name}}`, `{{job_date}}`, `{{tech_name}}`, `{{invoice_balance}}`.

**What it does** — A typed variable registry per message type; formatting by type (dates in tenant timezone, money in tenant currency); preview with sample data; error on unknown or missing variables.

**Why we need it** — Personalized messages without breaking when data is missing.

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: documented variables, preview, error if missing. The old "Quote Template System" spec's variable rules are reference.

---

#### T089 — Add email/SMS delivery tracking

**Phase**: 8 · **Category**: Email · **Hours**: 6 · **Status**: Open · **v1 Gate**: No · **Depends on**: T159, T181

**What it is** — Every outbound message shows sent/delivered/bounced/failed.

**What it does** — Store provider message IDs; Resend webhooks (delivered, bounced, complained) and Twilio status callbacks update a `messages` table; status shown in the customer timeline (T055).

**Why we need it** — Know whether reminders and confirmations actually reached the customer.

**Notes** — Old task (Feb 2026 TODO P3). Folds in old P3 "Email Analytics (Open/Click Tracking)" — opens/clicks only if cheap via Resend; no pixel/link-wrapping of our own.

---

#### T090 — Add communication failure alerts

**Phase**: 8 · **Category**: Monitoring · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T089

**What it is** — Admins are alerted when messages fail.

**What it does** — Bounce/failure events create an in-app alert on the customer/invoice and a daily digest to the tenant admin; repeated failures flag the contact as unreachable.

**Why we need it** — Prevents silent failures (e.g. reminders bouncing for months).

**Notes** — Old task (Feb 2026 TODO P3).

---

#### T091 — Add notification preferences per customer

**Phase**: 8 · **Category**: Feature · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Customers (and their contacts) opt in or out of each notification type and channel.

**What it does** — Per-contact preferences by type (reminders, confirmations, marketing/reviews) and channel; unsubscribe links in email and STOP handling in SMS (T181) update them; every sender checks them. Builds on the existing `PreferredContactMethod` and DoNot flags the reminder service already reads.

**Why we need it** — Respects preferences and keeps SMS/email compliant (TCPA, CAN-SPAM).

**Notes** — Old task (Feb 2026 TODO P3). Folds in the old "Future Features → Notification Preferences" item (in-app and digest settings are for staff users — later).

---

#### T181 — Two-way SMS inbox (Twilio + 10DLC)

**Phase**: 8 · **Category**: SMS · **Hours**: 20 · **Status**: Open · **v1 Gate**: No · **Depends on**: T091

**What it is** — Each tenant gets a business number and a shared inbox to text customers and see replies.

**What it does** — Twilio number per tenant; A2P 10DLC brand + campaign registration per tenant (registration fees and timelines per Twilio's current pricing — record in Fleet); outbound/inbound webhooks (signature-verified); threads per customer contact; STOP/HELP handling; unread counts; assign to staff.

**Why we need it** — Small shops expect to text customers from the business number, not personal phones.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 14, §5 v1. Provider choice is OD-6 (Twilio vs Telnyx).

---

#### T182 — Review request automation

**Phase**: 8 · **Category**: Marketing · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T048

**What it is** — After a job is completed (or paid) the customer gets a request to leave a Google review.

**What it does** — Tenant's Google review link in settings; send after completion/payment with delay; at most once per customer per N days; respects preferences (T091).

**Why we need it** — Cheap growth lever for customers.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 23.

---

### Phase 9 — Customer Self-Service (T081, T183, T184, T185)

Public quote approval with e-sign and deposit, the customer portal and online booking. Run T218 interviews before building beyond the gated items. **4 open tasks, ~50h.**

- [ ] T081 — Add e-signature for quote acceptance
- [ ] T183 — Public quote approval page (approve/decline, e-sign, optional deposit)
- [ ] T184 — Customer portal / client hub
- [ ] T185 — Online booking / request widget embeddable on websites

#### T081 — Add e-signature for quote acceptance

**Phase**: 9 · **Category**: Feature · **Hours**: 4 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T183

**What it is** — The customer signs to accept a quote, producing a legally useful acceptance record.

**What it does** — Typed name and/or drawn signature on the approval page; store signer name, email, timestamp, IP, user agent and a hash of the quote PDF version accepted; ESIGN/UETA consent text; signature shown on the accepted quote PDF.

**Why we need it** — A legal acceptance record for larger commercial work.

**Notes** — Old task (Feb 2026 TODO P3; confidence medium — typed name vs signature pad).

---

#### T183 — Public quote approval page (approve/decline, e-sign, optional deposit)

**Phase**: 9 · **Category**: Feature · **Hours**: 10 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T224

**What it is** — Customers open a tokenized link to view a quote, pick an option, approve or decline (with reason), sign, and pay a deposit if required.

**What it does** — Public route (e.g. `/q/:token`, like `/pay/:token`) with no app chrome; rate-limited token lookup; approve/decline transitions via T224's race-safe update; e-signature (T081); deposit via Connect (T061); notifies the office and logs activity.

**Why we need it** — Gets quotes approved without phone tag and is table stakes (`docs/product/competitive-research-2026-10.md` §2 rows 3–5).

**Notes** — `docs/product/competitive-research-2026-10.md` §5 MVP #4. Customer portal (T184) later wraps the same page.

---

#### T184 — Customer portal / client hub

**Phase**: 9 · **Category**: Feature · **Hours**: 24 · **Status**: Open · **v1 Gate**: No · **Depends on**: T183, T169

**What it is** — A per-customer portal to view and approve quotes, view and pay invoices, request work and see job history.

**What it does** — Magic-link sign-in per customer contact (no passwords); quotes (T183), invoices + pay (T169/T170), saved methods/autopay (T171), statements (T189), request-work form, job history with service reports (T176); tenant branding (logo, colours) on the portal, PDFs and emails.

**Why we need it** — Property managers expect self-serve, and it cuts A/R phone calls.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 20, §5 v1. Folds in the old "Future Features → White-label / Custom Branding" item at the logo/colours level (custom domains later).

---

#### T185 — Online booking / request widget embeddable on websites

**Phase**: 9 · **Category**: Feature · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T184

**What it is** — A widget tenants embed on their website so prospects can request work or book.

**What it does** — Embeddable script/iframe; request form creating a lead/request; optional available-window picker; Cloudflare Turnstile against spam; notification to the office.

**Why we need it** — Lead capture straight into the system.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 21.

---

### Phase 10 — Commercial/B2B Wedge (T044, T064, T065, T186, T187, T188, T189, T190, T191)

What residential-first tools do poorly and the beachhead needs: multi-site customers, POs, consolidated invoicing, statements, property-manager approvals and the collections suite. **9 open tasks, ~85h.**

- [ ] T044 — Build collections dashboard
- [ ] T064 — Add last contact date tracking on invoices
- [ ] T065 — Add promise-to-pay notes
- [ ] T186 — Multi-site customers (service locations; invoice per site or consolidated)
- [ ] T187 — PO capture and matching on jobs and invoices
- [ ] T188 — Consolidated monthly invoicing
- [ ] T189 — Statements of account (portal + email)
- [ ] T190 — Property-manager approval workflow (NTE limits, multi-approver, email intake)
- [ ] T191 — Maintenance agreements with autopay + renewal reminders

#### T044 — Build collections dashboard

**Phase**: 10 · **Category**: Feature · **Hours**: 8 · **Status**: Open · **v1 Gate**: No · **Depends on**: T064, T065

**What it is** — A single place to work overdue accounts.

**What it does** — Overdue invoices grouped by customer with contact info, last contact (T064), promise-to-pay (T065), reminder history (T034), one-click call log (T036) and send-statement (T189); sortable by amount/age.

**Why we need it** — Collections is the wedge; this is its home screen.

**Notes** — Old task (Feb 2026 TODO P1; confidence medium). Acceptance: all overdue accounts visible; contact info; log collection activity.

---

#### T064 — Add last contact date tracking on invoices

**Phase**: 10 · **Category**: Feature · **Hours**: 2 · **Status**: Open · **v1 Gate**: No · **Depends on**: T037

**What it is** — Each invoice shows when the customer was last contacted about it.

**What it does** — Updated by reminders, calls (T036) and manual notes; visible and filterable in the collections view.

**Why we need it** — Prevents pestering and ensures follow-up.

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: updated on contact, visible, filterable.

---

#### T065 — Add promise-to-pay notes

**Phase**: 10 · **Category**: Feature · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Record "I'll pay Friday" commitments with date and amount.

**What it does** — Promise date + amount + note on the invoice; shown in collections; reminders pause until the date; alert when it passes unpaid.

**Why we need it** — Turns phone promises into tracked follow-ups; the AI collections agent (T214) writes these too.

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: date and amount, visible in collections, alert when passed.

---

#### T186 — Multi-site customers (service locations; invoice per site or consolidated)

**Phase**: 10 · **Category**: Feature · **Hours**: 16 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T102

**What it is** — A customer can have many service locations, and jobs, quotes, agreements and invoices reference a site.

**What it does** — `service_locations` (customer, address, access notes, site contact, tenant); migrate each existing customer address into a default site; site pickers on job/quote/agreement/invoice; billing per site or consolidated per customer (T188); site-level history.

**Why we need it** — Commercial and property-manager customers have many properties — one address per customer (today) is a non-starter for the beachhead.

**Notes** — `docs/product/competitive-research-2026-10.md` §0 (gaps), §4.3, §5 MVP #8.

---

#### T187 — PO capture and matching on jobs and invoices

**Phase**: 10 · **Category**: Feature · **Hours**: 6 · **Status**: Open · **v1 Gate**: No · **Depends on**: T186

**What it is** — Purchase-order numbers captured on jobs/quotes, required where the customer demands it, carried to invoices and tracked against PO amounts.

**What it does** — PO field on jobs/quotes/agreements (invoices already have one); per-customer "PO required" flag blocking invoice send without it; PO amount and remaining balance across invoices.

**Why we need it** — B2B customers reject invoices without the right PO.

**Notes** — `docs/product/competitive-research-2026-10.md` §4.3, §5 v1.

---

#### T188 — Consolidated monthly invoicing

**Phase**: 10 · **Category**: Feature · **Hours**: 8 · **Status**: Open · **v1 Gate**: No · **Depends on**: T186

**What it is** — One invoice per customer (or site) per period covering all completed jobs.

**What it does** — Billing preference per customer (per job / monthly consolidated); a scheduled run (or button) rolls up completed, uninvoiced jobs into one invoice with per-job/site sections; works with agreements (T167).

**Why we need it** — Property managers and facilities want one invoice a month, not dozens.

**Notes** — `docs/product/competitive-research-2026-10.md` §5 v1.

---

#### T189 — Statements of account (portal + email)

**Phase**: 10 · **Category**: Feature · **Hours**: 6 · **Status**: Open · **v1 Gate**: No · **Depends on**: T184

**What it is** — A statement per customer listing open invoices, payments and aging, sent monthly and available in the portal.

**What it does** — Statement PDF (gofpdf) from the A/R data (T031); monthly email run; portal view (T184).

**Why we need it** — Standard B2B collections document.

**Notes** — `docs/product/competitive-research-2026-10.md` §5 v1.

---

#### T190 — Property-manager approval workflow (NTE limits, multi-approver, email intake)

**Phase**: 10 · **Category**: Feature · **Hours**: 24 · **Status**: Open · **v1 Gate**: No · **Depends on**: T186, T184

**What it is** — Property managers set not-to-exceed limits, approve work above them (with multiple approvers if needed), and can send work orders by email.

**What it does** — NTE per customer/site/job; jobs or quotes over NTE require approval through the portal; approver lists and order; audit trail; inbound email address per tenant that parses PM work-order emails into requests (Resend inbound or Cloudflare Email Routing).

**Why we need it** — Deep B2B workflow residential tools don't do — part of the niche moat.

**Notes** — `docs/product/competitive-research-2026-10.md` §4.3, §5 v2.

---

#### T191 — Maintenance agreements with autopay + renewal reminders

**Phase**: 10 · **Category**: Feature · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T167, T171

**What it is** — Service agreements bill automatically and remind customers before they renew.

**What it does** — Agreement billing schedule (per visit or periodic) charged via autopay (T171); renewal reminder N days before end; renew/upgrade flow in the portal.

**Why we need it** — Recurring revenue for customers; sticky for ServicePro.

**Notes** — `docs/product/competitive-research-2026-10.md` §5 v2.

---

### Phase 11 — Integrations & Migration (T095, T096, T097, T098, T192, T193, T194, T195, T196, T197)

QuickBooks (the #1 B2B buying gate), import mappers, and partner integrations instead of builds. **10 open tasks, ~103h.**

- [ ] T095 — Build outgoing webhooks system
- [ ] T096 — Build event log table
- [ ] T097 — Add webhook retry logic
- [ ] T098 — Document public API (OpenAPI/Swagger)
- [ ] T192 — QuickBooks Online sync (customers, invoices, payments; one-way first)
- [ ] T193 — Import mappers for Jobber / Housecall Pro exports + generic CSV
- [ ] T194 — Zapier app
- [ ] T195 — Wisetack consumer financing
- [ ] T196 — Gusto payroll / timesheets
- [ ] T197 — AI receptionist partner integration

#### T095 — Build outgoing webhooks system

**Phase**: 11 · **Category**: Integration · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T096

**What it is** — Tenants subscribe URLs to ServicePro events.

**What it does** — Endpoint registry per tenant; event types from the T096 event log; HMAC-signed payloads; delivery log; SSRF guard (block private/link-local IPs, resolve-then-connect) since this creates the first user-configurable outbound calls.

**Why we need it** — Integration foundation (Zapier T194, customer systems).

**Notes** — Old task (Feb 2026 TODO P3). An unlinked `internal/api/handlers/webhooks` package exists (correctness audit 5.1) — evaluate before T129 deletes it. security audit "SSRF" (no user-configurable outbound webhooks exist today). Retry is T097.

---

#### T096 — Build event log table

**Phase**: 11 · **Category**: Backend · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T102

**What it is** — A tenant-scoped append-only table of significant domain events.

**What it does** — `events (id, tenant_id, type, entity, entity_id, actor, payload, created_at)` written in the same transaction as the change; consumed by the timeline (T037/T055), webhooks (T095) and audit log (T093).

**Why we need it** — Debugging and integration foundation.

**Notes** — Old task (Feb 2026 TODO P3). Don't revive the unlinked `internal/analytics` event store (broken `ON CONFLICT` target, correctness audit 7.L).

---

#### T097 — Add webhook retry logic

**Phase**: 11 · **Category**: Integration · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T095

**What it is** — Failed webhook deliveries are retried with backoff and surfaced to the tenant.

**What it does** — Exponential backoff with jitter over ~24h via the scheduled-jobs runner (T141); disable an endpoint after repeated failures and notify; manual redeliver from the log.

**Why we need it** — Reliable event delivery.

**Notes** — Old task (Feb 2026 TODO P3). Folds in the old "Future Features → Webhooks for Customers" item.

---

#### T098 — Document public API (OpenAPI/Swagger)

**Phase**: 11 · **Category**: Docs · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T152

**What it is** — Published API docs and API keys for integrators.

**What it does** — Partly done by T015 (swag → `swagger.json`, Swagger UI at `/api/docs`). Remaining: per-tenant API keys with scopes and rate limits; mark the public subset; publish static docs (Swagger UI is locked down in production by T114); versioning policy.

**Why we need it** — Enables Zapier, the MCP server (T217) and partner integrations.

**Notes** — Old task (Feb 2026 TODO P3). Folds in the old "Future Features → Public API for Integrations" item (OAuth2 for third-party apps later).

---

#### T192 — QuickBooks Online sync (customers, invoices, payments; one-way first)

**Phase**: 11 · **Category**: Integration · **Hours**: 32 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T118, T102

**What it is** — Customers, invoices and payments created in ServicePro appear in the tenant's QuickBooks Online.

**What it does** — Intuit OAuth2 per tenant; mapping tables (customer, item/service, tax code, account); one-way push on create/update/payment with an error queue and retry; manual "resync"; Intuit app review. Two-way later.

**Why we need it** — The #1 buying gate for B2B shops — their accountant lives in QuickBooks.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 22, §5 MVP #6.

---

#### T193 — Import mappers for Jobber / Housecall Pro exports + generic CSV

**Phase**: 11 · **Category**: Integration · **Hours**: 8 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T186, T241

**What it is** — New tenants import customers, sites, jobs and open invoices from Jobber or Housecall Pro exports, or any CSV.

**What it does** — Column mappers for each export format on top of the existing CSV import (`import_service.go`); preview + validation; tenant-scoped; size/row caps (security audit F-23) and the unchecked `json.Unmarshal` at `import_service.go:341` fixed.

**Why we need it** — Switching friction is the main barrier; the white-glove migration offer (T220) relies on it.

**Notes** — `docs/product/competitive-research-2026-10.md` §2 row 26, §5 MVP #9.

---

#### T194 — Zapier app

**Phase**: 11 · **Category**: Integration · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T095, T098

**What it is** — A Zapier integration with ServicePro triggers and actions.

**What it does** — Triggers: new customer, job completed, invoice sent, invoice paid; actions: create customer, create job/request. Auth via API keys (T098); triggers via outgoing webhooks (T095).

**Why we need it** — Covers the long tail of integrations without building each.

**Notes** — `docs/product/competitive-research-2026-10.md` §5 v2 "Integrations rather than builds".

---

#### T195 — Wisetack consumer financing

**Phase**: 11 · **Category**: Integration · **Hours**: 8 · **Status**: Open · **v1 Gate**: No · **Depends on**: T183

**What it is** — Customers can finance larger quotes through Wisetack from the approval page.

**What it does** — Wisetack partner API; "pay over time" option on quotes above a threshold; status webhooks; payout reconciled to the invoice.

**Why we need it** — Matters for big-ticket replacements; competitors have it.

**Notes** — `docs/product/competitive-research-2026-10.md` §3 (about 3.9% merchant fee, 3rd-party figure). Less relevant to the B2B beachhead — confirm in T218 before building.

---

#### T196 — Gusto payroll / timesheets

**Phase**: 11 · **Category**: Integration · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T177

**What it is** — Approved timesheets flow into Gusto payroll.

**What it does** — Gusto OAuth; map ServicePro users to Gusto employees; push approved hours per pay period.

**Why we need it** — Payroll is a common add-on in competitors; partner, don't build.

**Notes** — `docs/product/competitive-research-2026-10.md` §3, §5 v2.

---

#### T197 — AI receptionist partner integration

**Phase**: 11 · **Category**: Integration · **Hours**: 8 · **Status**: Open · **v1 Gate**: No · **Depends on**: T098, T185

**What it is** — An AI answering/receptionist vendor can create requests and book jobs in ServicePro.

**What it does** — Public booking/request API (T098 keys, T185 request model) and a partner guide; integrate with one vendor first.

**Why we need it** — AI receptionists became a $29/mo commodity in ~12 months — integrate rather than build.

**Notes** — `docs/product/competitive-research-2026-10.md` §1.2, §3, §4.1.

---

### Phase 12 — Reporting & Dashboards (T045, T046, T047, T066, T067, T068, T069, T070, T071, T084, T085, T086, T087, T088, T198, T246)

Dashboard widgets and reports from the old TODO plus job costing. All tenant-scoped and in tenant timezone/currency. **16 open tasks, ~56h.**

- [ ] T045 — Add aging quotes dashboard widget
- [ ] T046 — Add follow-up queue dashboard widget
- [ ] T047 — Build "stuck work" report
- [ ] T066 — Add technician utilization widget
- [ ] T067 — Add quote conversion rate metric
- [ ] T068 — Build job-completion-to-invoice time report
- [ ] T069 — Build invoice-to-paid time report (DSO)
- [ ] T070 — Build revenue by service type report
- [ ] T071 — Build collections rate report
- [ ] T084 — Build quote-to-job conversion report
- [ ] T085 — Build lead-to-quote time report
- [ ] T086 — Build quote-to-acceptance time report
- [ ] T087 — Build revenue by technician report
- [ ] T088 — Add comparative analysis (vs prior period)
- [ ] T198 — Job costing / profit per job
- [ ] T246 — Lazy-load / trim Sentry and PostHog (109 KB gzipped on every page)

#### T045 — Add aging quotes dashboard widget

**Phase**: 12 · **Category**: Reporting · **Hours**: 2 · **Status**: Open · **v1 Gate**: No · **Depends on**: T062

**What it is** — A widget counting quotes pending longer than a threshold.

**What it does** — Count of sent quotes older than 7/14/30 days (configurable); click-through to the filtered list.

**Why we need it** — Quick visibility into a stale pipeline.

**Notes** — Old task (Feb 2026 TODO). P1. Depends on quote aging (T062). All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T046 — Add follow-up queue dashboard widget

**Phase**: 12 · **Category**: Reporting · **Hours**: 2 · **Status**: Open · **v1 Gate**: No · **Depends on**: T042

**What it is** — A widget showing quotes and invoices needing follow-up today.

**What it does** — Count from the follow-up queue (T042) and collections (T044); links to each.

**Why we need it** — Daily action list at a glance.

**Notes** — Old task (Feb 2026 TODO). P1. All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T047 — Build "stuck work" report

**Phase**: 12 · **Category**: Reporting · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T033

**What it is** — A report of work falling through the cracks.

**What it does** — Three lists: completed-not-invoiced (T033 query), quotes without follow-up, overdue invoices without recent contact (T064); quick actions to unstick.

**Why we need it** — Catches what slips between office and field.

**Notes** — Old task (Feb 2026 TODO). P1; "stuck" definitions to confirm. All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T066 — Add technician utilization widget

**Phase**: 12 · **Category**: Reporting · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T038

**What it is** — Scheduled hours vs capacity per tech for today.

**What it does** — Per-tech scheduled/available hours (T038) with under/over highlighting.

**Why we need it** — Capacity planning.

**Notes** — Old task (Feb 2026 TODO). P2; historical trend later. All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T067 — Add quote conversion rate metric

**Phase**: 12 · **Category**: Reporting · **Hours**: 2 · **Status**: Open · **v1 Gate**: No · **Depends on**: T102

**What it is** — Quote-to-job conversion rate on the dashboard.

**What it does** — Accepted / sent over a configurable period with trend; the `v_quote_conversion` Grafana view (T026) has the logic.

**Why we need it** — Business health indicator.

**Notes** — Old task (Feb 2026 TODO). P2. All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T068 — Build job-completion-to-invoice time report

**Phase**: 12 · **Category**: Reporting · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T102

**What it is** — Average days from job completed to invoice sent.

**What it does** — From job and invoice timestamps; trend; period filter.

**Why we need it** — Measures billing efficiency.

**Notes** — Old task (Feb 2026 TODO). P2. All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T069 — Build invoice-to-paid time report (DSO)

**Phase**: 12 · **Category**: Reporting · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T118

**What it is** — Days Sales Outstanding — average days to collect.

**What it does** — DSO from invoice sent and payment dates (T118 makes payments trustworthy); trend over time.

**Why we need it** — Key collections metric for the wedge.

**Notes** — Old task (Feb 2026 TODO). P2. The old "industry comparison" criterion is dropped (no data source). All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T070 — Build revenue by service type report

**Phase**: 12 · **Category**: Reporting · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T118

**What it is** — Revenue broken down by job/service type.

**What it does** — Aggregate from paid invoice lines joined to job type / price-book category; chart; period filter. Compute from invoices/payments — `revenue_transactions` is never written, so the existing revenue reports always show $0.

**Why we need it** — Shows where money comes from.

**Notes** — Old task (Feb 2026 TODO). P2. Correctness audit 7.M11. All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T071 — Build collections rate report

**Phase**: 12 · **Category**: Reporting · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T118

**What it is** — Percentage of billed revenue collected.

**What it does** — Collected / billed per period with trend and breakdown by age bucket.

**Why we need it** — Tracks bad debt.

**Notes** — Old task (Feb 2026 TODO). P2. All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T084 — Build quote-to-job conversion report

**Phase**: 12 · **Category**: Reporting · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T067

**What it is** — Detailed conversion analysis over time.

**What it does** — Extends T067 with breakdown by service type, staff and value band.

**Why we need it** — Sales effectiveness.

**Notes** — Old task (Feb 2026 TODO). P3. All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T085 — Build lead-to-quote time report

**Phase**: 12 · **Category**: Reporting · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T102

**What it is** — Average time from new customer/request to first quote.

**What it does** — From customer/request creation to first quote sent.

**Why we need it** — Measures sales speed.

**Notes** — Old task (Feb 2026 TODO). P3; needs request tracking (T185) for real leads. All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T086 — Build quote-to-acceptance time report

**Phase**: 12 · **Category**: Reporting · **Hours**: 2 · **Status**: Open · **v1 Gate**: No · **Depends on**: T102

**What it is** — Average time for quotes to be accepted.

**What it does** — From `sent_at` to `accepted_at`.

**Why we need it** — Measures quote effectiveness.

**Notes** — Old task (Feb 2026 TODO). P3. All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T087 — Build revenue by technician report

**Phase**: 12 · **Category**: Reporting · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T102

**What it is** — Revenue attributed to each technician.

**What it does** — Paid revenue by assigned tech(s) with split rules for multi-tech jobs; restricted to admin roles.

**Why we need it** — Performance tracking.

**Notes** — Old task (Feb 2026 TODO). P3; privacy considerations noted in the original. All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T088 — Add comparative analysis (vs prior period)

**Phase**: 12 · **Category**: Reporting · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T102

**What it is** — Period-over-period comparison on dashboard metrics and reports.

**What it does** — "Revenue this month vs last month: +12%"; customizable date ranges; CSV export.

**Why we need it** — Trend visibility.

**Notes** — Old task (Feb 2026 TODO). P3. Folds in the old "Future Features → Dashboard KPI Improvements" item (ranges, export, comparison). All reports are tenant-scoped (T102) and use tenant timezone/currency (T099/T127).

---

#### T198 — Job costing / profit per job

**Phase**: 12 · **Category**: Reporting · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T168, T177

**What it is** — Profit per job: revenue minus labor and materials cost.

**What it does** — Labor cost from time entries (T177) × per-tech cost rate; material cost from price-book costs (T168); job and agreement profitability report; margin by service type.

**Why we need it** — Owners don't know which jobs or contracts lose money; a known Jobber weakness.

**Notes** — `docs/product/competitive-research-2026-10.md` §1.3 (Jobber "weak job costing"), §3.

---

#### T246 — Lazy-load / trim Sentry and PostHog (109 KB gzipped on every page)

**Phase**: 12 · **Category**: Perf · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — Error tracking and analytics stop costing ~109 KB gzipped of JavaScript on first load.

**What it does** — Load PostHog after first paint (dynamic import, queue early events), drop PostHog features we don't use (session recording/surveys bundle), and trim Sentry (no Replay unless enabled; `@sentry/browser` tree-shaking). Lower the `vendor-observability` budget in `frontend/scripts/analyze-bundle.cjs` to the new size; aim for total JS under the 400 KB target.

**Why we need it** — Field techs on phones on cell data pay for this on every page load; total JS is 471 KB against a 400 KB target.

**Notes** — Found while repairing CI (2026-10-16), when the catch-all vendor chunk failed its budget.

---

### Phase 13 — Roles, Compliance & Trust (T074, T075, T092, T093, T094, T199, T200, T201, T230, T231)

Remaining roles, audit logging, MFA, and the export/deletion promises that back the trust brand. **9 open tasks, ~49h.** 1 done.

- [x] T230 — Expose the caller's effective permissions per tenant; decide who can delete an organization ✓ (2026-10-16)
- [ ] T074 — Create Accountant role
- [ ] T075 — Add field-level permission controls
- [ ] T092 — Implement 2FA (TOTP)
- [ ] T093 — Implement comprehensive audit logging
- [~] T094 — Add Google OAuth sign-in (Won't Do — superseded by T158)
- [ ] T199 — One-click full data export (trust promise)
- [ ] T200 — Account/tenant deletion + data retention policy
- [ ] T201 — Privacy/CCPA handling SOP
- [~] T231 — Owner decision: may a tenant's owner/admin delete their own organization? (Deferred)

#### T074 — Create Accountant role

**Phase**: 13 · **Category**: Auth · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: T104

**What it is** — A view-only role for bookkeepers/CPAs covering all financial data.

**What it does** — Include: view invoices, payments, financial reports, export. Exclude: create/edit customers, modify jobs, change settings. CLAUDE.md permission steps.

**Why we need it** — Lets owners give their bookkeeper access safely.

**Notes** — Old task (Feb 2026 TODO P2). Acceptance: role created; all financial data visible; no modification.

---

#### T075 — Add field-level permission controls

**Phase**: 13 · **Category**: Auth · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T035

**What it is** — Specific fields (prices, balances, costs) are hidden from roles without financial permission, in every API response.

**What it does** — Response shaping by permission (serializers per role or a field-permission map) applied to customers, jobs, quotes, invoices; configurable visibility per tenant; tests proving no leakage via list/detail/export.

**Why we need it** — "Tech can't see 'customer owes $5,000 overdue'" must hold everywhere, not just in the UI.

**Notes** — Old task (Feb 2026 TODO P2; confidence medium). T035 does the minimum for technicians.

---

#### T092 — Implement 2FA (TOTP)

**Phase**: 13 · **Category**: Auth · **Hours**: 6 · **Status**: Open · **v1 Gate**: No · **Depends on**: T158

**What it is** — Users can protect their account with an authenticator app; admins can require it.

**What it does** — After T158, enable Supabase Auth MFA (TOTP) in the sign-in screens, recovery codes, and a per-tenant "require MFA for admins" setting checked by the API (AAL claim). Before T158 don't build in-house TOTP.

**Why we need it** — Security for accounts that control money flows.

**Notes** — Old task (Feb 2026 TODO P3). Folds in the old "Future Features → Two-Factor Authentication" item. Effort likely drops below the estimate with Supabase.

---

#### T093 — Implement comprehensive audit logging

**Phase**: 13 · **Category**: Backend · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T096

**What it is** — Who changed what, when, and from where, for all business data — searchable and exportable.

**What it does** — Audit entries (actor, tenant, entity, before/after diff, IP via T107, request ID via T223) written in the change transaction; admin search and CSV export; retention policy (T200).

**Why we need it** — Compliance, dispute resolution and debugging.

**Notes** — Old task (Feb 2026 TODO P3; confidence medium). Folds in the old "Future Features → Audit Logging" item.

---

#### T094 — Add Google OAuth sign-in

**Phase**: 13 · **Category**: Auth · **Hours**: 0 · **Status**: Won't Do · **v1 Gate**: No · **Depends on**: none

**Why we won't do this** — Superseded by T158 — Supabase Auth provides Google sign-in. Revisit only if T158 is abandoned.

---

#### T199 — One-click full data export (trust promise)

**Phase**: 13 · **Category**: Feature · **Hours**: 6 · **Status**: Open · **v1 Gate**: No · **Depends on**: T128

**What it is** — A tenant owner downloads all their data (CSV/JSON per entity plus PDFs and attachments) in one ZIP.

**What it does** — Background export (T128 infra) of every tenant-owned table and R2 object; emailed link; documented format; available on every plan.

**Why we need it** — "Your data leaves with one click" is a core brand promise against lock-in-heavy incumbents.

**Notes** — `docs/product/competitive-research-2026-10.md` §4.2 (3), §4.5.

---

#### T200 — Account/tenant deletion + data retention policy

**Phase**: 13 · **Category**: Legal · **Hours**: 6 · **Status**: Open · **v1 Gate**: No · **Depends on**: T199

**What it is** — Owners can delete their tenant; data is purged on a documented schedule.

**What it does** — Owner-initiated deletion with export offered first (T199) and a 30-day grace period; cascade delete of rows and R2 objects; Connect account detached; backups age out per the backup retention; written retention policy linked from the privacy policy.

**Why we need it** — Required for trust and privacy compliance; the no-contract promise means easy exit.

**Notes** — LL D6/T237 pattern.

---

#### T201 — Privacy/CCPA handling SOP

**Phase**: 13 · **Category**: Legal · **Hours**: 3 · **Status**: Open · **v1 Gate**: No · **Depends on**: none

**What it is** — A written procedure for privacy requests, for ServicePro's own users and for end-customer data held on tenants' behalf.

**What it does** — privacy@ inbox; 45-day response SLA; steps for access/deletion; processor role for tenants' customer data (DPA template); log of requests.

**Why we need it** — Required for any US users; manual handling is permitted.

**Notes** — LL `docs/ops/ccpa-sop.md` is the template.

---

#### T231 — Owner decision: may a tenant's owner/admin delete their own organization?

**Phase**: 13 · **Category**: Auth · **Hours**: 1 · **Status**: Deferred · **v1 Gate**: No · **Depends on**: T230

**What it is** — A deliberate rule for who can delete an organization, implemented as a permission grant.

**Why it's deferred** — Deferred 2026-10-18 (owner): a future feature — few owners want to delete their organization. Until then only `super_admin` can. Revisit when a paying customer asks to close their account, or with T200's deletion/retention work. Split out of T230 (2026-10-16), which shipped the effective-permissions endpoint and UI gating.

---

### Phase 14 — Hosting & Launch Readiness (T202, T203, T204, T205, T206, T207, T208, T209, T210, T211, T212, T213)

Provision and launch. Database: Neon (T202, decided 2026-10-18). ServicePro stays local-only until these land. **11 open tasks, ~34h.** 1 done.

- [x] T202 — Decide production database hosting (OWNER DECISION) ✓ (2026-10-18)
- [ ] T203 — GCP project + WIF + Artifact Registry + Cloud Run service
- [ ] T204 — Deploy the Worker + DNS for servicepro.sharkfins.xyz
- [ ] T205 — Invite-gated preview launch
- [ ] T206 — Production email domain on Resend (SPF/DKIM/DMARC)
- [ ] T207 — Terms of Service + Privacy Policy legal review
- [ ] T208 — Support inbox + Better Stack status page
- [ ] T209 — Activate nightly backups to R2 + restore drill
- [ ] T210 — ServicePro plans & billing: Solo/Team/Pro flat, unlimited users, founding offer
- [ ] T211 — Update Fleet and the blog when ServicePro goes live
- [ ] T212 — Production config review
- [ ] T213 — Cost guardrails (Cloud Run max instances × DB pool, budget alerts)

#### T203 — GCP project + WIF + Artifact Registry + Cloud Run service

**Phase**: 14 · **Category**: Deploy · **Hours**: 4 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T202, T143

**What it is** — A GCP project with keyless GitHub deploys and a capped Cloud Run service `servicepro-api`.

**What it does** — Project, region next to the DB; Artifact Registry `servicepro` with a keep-5 cleanup policy; Workload Identity Federation pool pinned to this repo + `gh-deploy` SA; runtime SA; Secret Manager secrets (DATABASE_URL, Stripe, Resend, R2, EDGE_PROXY_SECRET, INTERNAL_JOBS_SECRET, Sentry, PostHog); Cloud Run with `--max-instances` cap and min 0; budget with email alerts at 50/90/100%; GitHub vars for T146. Follow `docs/ops/cloud-run-setup.md` (T151).

**Why we need it** — The API needs somewhere to run; LL's pattern is proven and cheap at this scale.

**Notes** — D3; infra audit §5. Cost caps detailed in T213.

---

#### T204 — Deploy the Worker + DNS for servicepro.sharkfins.xyz

**Phase**: 14 · **Category**: Deploy · **Hours**: 2 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T150, T203, T107

**What it is** — `https://servicepro.sharkfins.xyz` serves the frontend and proxies `/api/*` and `/version` to Cloud Run.

**What it does** — Custom domain route on the `sharkfins.xyz` zone; `API_ORIGIN` = run.app URL; `EDGE_PROXY_SECRET` set as a Worker secret and in Secret Manager (same value); first `wrangler deploy` via T146; Stripe webhook endpoint updated to the new host; backend `FRONTEND_URL` and CORS updated.

**Why we need it** — The user-facing address and the edge path T106/T107 rely on.

**Notes** — D3. Per CLAUDE.md, adding the domain triggers T211.

---

#### T205 — Invite-gated preview launch

**Phase**: 14 · **Category**: Deploy · **Hours**: 2 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T106, T204, T206

**What it is** — The first invited tenants can sign up and use ServicePro at the production URL.

**What it does** — Confirm all of Phase 1 is done; create the first sign-up invites (super_admin path from T106); smoke test sign-up → tenant → customer → quote → job → invoice → pay; watch Sentry for a week.

**Why we need it** — Real usage from the T218 interviewees is what validates the beachhead.

**Notes** — Requires every Phase 1 task.

---

#### T206 — Production email domain on Resend (SPF/DKIM/DMARC)

**Phase**: 14 · **Category**: Email · **Hours**: 2 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T159

**What it is** — Transactional email sends from a verified ServicePro domain with SPF, DKIM and DMARC passing.

**What it does** — Verify a sending subdomain in Resend; add DNS records on Cloudflare; DMARC at `p=none` with reporting, tighten later; set `RESEND_FROM_EMAIL`; test deliverability to Gmail/Outlook.

**Why we need it** — Reminders and invoices that land in spam defeat the collections wedge.

**Notes** — D2 (Resend).

---

#### T207 — Terms of Service + Privacy Policy legal review

**Phase**: 14 · **Category**: Legal · **Hours**: 4 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: none

**What it is** — Final Terms and Privacy Policy reviewed and published, replacing the draft placeholders.

**What it does** — Draft from a reputable template covering: payments via Stripe Connect (ServicePro not merchant of record if direct charges, OD-4), SMS consent, data processing for tenants' customers, export/deletion (T199/T200), subprocessors list; review; publish with a fixed effective date.

**Why we need it** — Required before charging anyone or processing their customers' data.

**Notes** — frontend audit §7 (placeholders with "Last updated: today"). T138 marks the current pages as drafts in the meantime.

---

#### T208 — Support inbox + Better Stack status page

**Phase**: 14 · **Category**: Ops · **Hours**: 2 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T204

**What it is** — A support address that reaches the owner and a public status page.

**What it does** — support@ (and privacy@, T201) forwarding via Cloudflare Email Routing; Better Stack status page linked in the app footer, built on the monitors from T165 (create the two uptime monitors here if T165 hasn't run yet).

**Why we need it** — "You email me directly" is part of the trust promise; a status page answers "is it down?".

**Notes** — `docs/product/competitive-research-2026-10.md` §4.2 (3).

---

#### T209 — Activate nightly backups to R2 + restore drill

**Phase**: 14 · **Category**: Ops · **Hours**: 2 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T151, T202

**What it is** — Nightly database backups run, are verified, and a restore has been rehearsed.

**What it does** — Create R2 bucket `servicepro-backups` with a 30-day lifecycle rule and a bucket-scoped token; GitHub secrets; enable `db-backup.yml` (T151); first run checked with `pg_restore --list`; restore into a scratch database and record timings in `docs/ops/backups.md`.

**Why we need it** — Provider backups alone aren't enough for a system holding customers' invoices and payment history.

**Notes** — infra audit §5 `db-backup.yml`.

---

#### T210 — ServicePro plans & billing: Solo/Team/Pro flat, unlimited users, founding offer

**Phase**: 14 · **Category**: Billing · **Hours**: 12 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T122, T169

**What it is** — ServicePro's own subscription plans match the pricing hypothesis and are sold self-serve.

**What it does** — Adapt the existing membership/subscription system (tiers, Stripe Billing on the platform account, proration preview): Solo $39 / Team $99 (unlimited users) / Pro $199, month-to-month, annual options; founding offer — Team at $59/mo for life for the first 25 tenants (Stripe coupon or grandfathered price); plan → platform fee mapping for Connect (T169); feature gating per plan kept minimal; trial length decided.

**Why we need it** — Flat, unlimited-user, no-contract pricing is the attack on incumbents' #1 complaint, and the business needs revenue.

**Notes** — `docs/product/competitive-research-2026-10.md` §4.5. Relies on the webhook fix in T122.

---

#### T211 — Update Fleet and the blog when ServicePro goes live

**Phase**: 14 · **Category**: Docs · **Hours**: 1 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T204

**What it is** — Fleet and the public blog reflect ServicePro's hosting, vendors, costs and URL.

**What it does** — Run `/fleet-sync` (Cloud Run, Workers, DB host, R2, Resend, Twilio, Sentry, PostHog, Better Stack, costs). Blog (`~/repositories/blogs/rob-blog`): add ServicePro to `src/pages/apps.astro` if it's public, and the Worker name/account/hosting to `INFRA.md`; commit, ask before pushing.

**Why we need it** — Required by the owner's global instructions whenever hosting, vendors, costs or domains change.

**Notes** — Owner CLAUDE.md "Fleet" and "Blog" sections.

---

#### T212 — Production config review

**Phase**: 14 · **Category**: Ops · **Hours**: 2 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T203

**What it is** — A checklist-driven review that production runs with safe settings.

**What it does** — `ENV=production`, `GIN_MODE=release`, Swagger and debug off (T114), `COOKIE_SECURE=true`/SameSite (T115), `REGISTRATION_MODE=invite_only` (T106), all secrets from Secret Manager (none as plain env), CORS limited to the production origin, email/storage not mock (T113), `FRONTEND_URL` correct, Sentry/PostHog DSNs set, edge secret enforced.

**Why we need it** — Most of the security fixes are config-dependent ("conditional" findings in the security audit).

**Notes** — security audit F-14, F-15, F-16.

---

#### T213 — Cost guardrails (Cloud Run max instances × DB pool, budget alerts)

**Phase**: 14 · **Category**: Ops · **Hours**: 1 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: T203, T144

**What it is** — Hard ceilings on spend and DB connections.

**What it does** — Pin `--max-instances` in `scripts/deploy-cloudrun.sh` and pool size (T144) so instances × pool stays under the DB limit with room for migrate/backup/psql; budget alerts at 50/90/100%; Artifact Registry cleanup; document "raise instances and pool together".

**Why we need it** — Without caps a bot or retry loop can exhaust DB connections exactly when traffic is good, or scale the bill silently.

**Notes** — LL T368 is the pattern.

---

### Phase 15 — AI Differentiators (T214, T215, T216, T217)

AI as accelerator and marketing hook, not the moat — aimed at the collections wedge first. **4 open tasks, ~68h.**

- [ ] T214 — AI collections agent
- [ ] T215 — AI quote drafting from photos, voice or notes against the price book
- [ ] T216 — "Ask your business" chat over reporting views
- [ ] T217 — MCP server for ServicePro

#### T214 — AI collections agent

**Phase**: 15 · **Category**: AI · **Hours**: 24 · **Status**: Open · **v1 Gate**: No · **Depends on**: T121, T065, T181

**What it is** — An agent that runs collections: drafts and sends escalating reminders, reads replies, logs promises-to-pay and suggests calls.

**What it does** — LLM drafts per-invoice messages from history and tone rules (owner approves or auto-sends within limits); inbound email/SMS replies classified ("paid", "send W-9", "dispute", "promise Friday") and turned into actions — promise-to-pay (T065), task for staff, pause reminders; daily summary of who to call.

**Why we need it** — Aligned with the wedge; ServiceTitan only lists it as "coming soon".

**Notes** — `docs/product/competitive-research-2026-10.md` §3, §5 v2. Use the `claude-api` skill for model choice.

---

#### T215 — AI quote drafting from photos, voice or notes against the price book

**Phase**: 15 · **Category**: AI · **Hours**: 20 · **Status**: Open · **v1 Gate**: No · **Depends on**: T168, T174

**What it is** — Techs or owners describe the job (photos, voice memo, notes) and get a draft quote built from the tenant's price book.

**What it does** — Multimodal LLM call with the price book as tools/context; produces line items with quantities and options; always a draft for human review.

**Why we need it** — Strong demo value; matches Markate, QuoteIQ and Jobber.

**Notes** — `docs/product/competitive-research-2026-10.md` §3, §5 v2.

---

#### T216 — "Ask your business" chat over reporting views

**Phase**: 15 · **Category**: AI · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T102

**What it is** — Owners ask questions in plain language ("who owes me the most over 60 days?") and get answers from their data.

**What it does** — LLM with read-only, tenant-scoped query tools over curated reporting views (the Grafana KPI views, A/R); answers cite the numbers; no free-form SQL against raw tables.

**Why we need it** — Cheap given the existing data layer; incumbents ship it (ServiceTitan Atlas, Jobber AI chat).

**Notes** — `docs/product/competitive-research-2026-10.md` §3. Tenant scoping (T102/T154) is a hard prerequisite.

---

#### T217 — MCP server for ServicePro

**Phase**: 15 · **Category**: AI · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T098

**What it is** — Owners connect ServicePro to Claude or ChatGPT through an MCP server.

**What it does** — Remote MCP server exposing scoped tools (look up customer, A/R summary, create job/request, draft quote) authenticated with per-user OAuth or API keys (T098); tenant-scoped; audit-logged (T093).

**Why we need it** — Cheap, forward-looking differentiator; few FSMs have one.

**Notes** — `docs/product/competitive-research-2026-10.md` §3, §5 v2.

---

### Phase 16 — Go-to-Market (T218, T219, T220, T221, T222)

Validate the beachhead before the Phase 9–10 build-out, then publish positioning and pricing. **5 open tasks, ~40h.**

- [ ] T218 — Customer discovery: interview 10 commercial service contractors
- [ ] T219 — Marketing site with positioning + transparent pricing
- [ ] T220 — White-glove migration offer + playbook
- [ ] T221 — Comparison / alternative pages (vs Jobber, Housecall Pro, ServiceTitan)
- [ ] T222 — Validate secondary beachhead (inspection/compliance trades)

#### T218 — Customer discovery: interview 10 commercial service contractors

**Phase**: 16 · **Category**: Research · **Hours**: 10 · **Status**: Open · **v1 Gate**: Yes · **Depends on**: none

**What it is** — Ten structured interviews with owners/office managers of 1–10-tech commercial cleaning, property maintenance and small commercial HVAC shops.

**What it does** — Script covering current tools and spend, billing on terms, POs, multi-site, collections pain, QuickBooks dependence, willingness to pay $99 flat, switching triggers; recruit locally and via property-manager contacts; write up findings and update the beachhead, pricing and Phase 9–10 scope.

**Why we need it** — The beachhead, pricing and collections wedge are hypotheses from desk research. Do this BEFORE the Phase 9–10 build-out.

**Notes** — `docs/product/competitive-research-2026-10.md` §4.3, §4.5. OD-7.

---

#### T219 — Marketing site with positioning + transparent pricing

**Phase**: 16 · **Category**: Marketing · **Hours**: 12 · **Status**: Open · **v1 Gate**: No · **Depends on**: T218

**What it is** — Public pages that state the positioning and publish the prices.

**What it does** — Landing page around "field service software that gets you paid" for commercial contractors; pricing page (Solo/Team/Pro, no contract, unlimited users, platform fee disclosed); founding-offer CTA; trust promises (export, migration, direct support).

**Why we need it** — Transparent pricing is part of the attack on incumbents' hidden/quote-only pricing.

**Notes** — `docs/product/competitive-research-2026-10.md` §4.4, §4.5.

---

#### T220 — White-glove migration offer + playbook

**Phase**: 16 · **Category**: Marketing · **Hours**: 4 · **Status**: Open · **v1 Gate**: No · **Depends on**: T193

**What it is** — A documented, repeatable "we migrate you for free" service.

**What it does** — Checklist per source system (Jobber, HCP, spreadsheets) using T193 mappers; data-validation steps; customer communication templates; time budget per migration.

**Why we need it** — Switching friction is the main barrier; a solo founder can credibly offer this.

**Notes** — `docs/product/competitive-research-2026-10.md` §4.2 (3), §4.5.

---

#### T221 — Comparison / alternative pages (vs Jobber, Housecall Pro, ServiceTitan)

**Phase**: 16 · **Category**: Marketing · **Hours**: 8 · **Status**: Open · **v1 Gate**: No · **Depends on**: T219

**What it is** — Honest comparison pages for buyers searching for alternatives.

**What it does** — One page per competitor: who each is for, pricing (sourced and dated), where ServicePro fits (commercial, terms, flat price); no trademark misuse.

**Why we need it** — High-intent search traffic.

**Notes** — `docs/product/competitive-research-2026-10.md` §1.

---

#### T222 — Validate secondary beachhead (inspection/compliance trades)

**Phase**: 16 · **Category**: Research · **Hours**: 6 · **Status**: Open · **v1 Gate**: No · **Depends on**: T218

**What it is** — Interviews with backflow testing, fire extinguisher/sprinkler inspection and hood-cleaning businesses.

**What it does** — 5+ interviews after the first 10 customers; assess recurring-inspection needs, certificate/authority report formats and current software.

**Why we need it** — Recurring compliance work with mandatory reports creates strong lock-in, but there's no solid market data yet.

**Notes** — `docs/product/competitive-research-2026-10.md` §4.3 "Secondary".

---

## Open Decisions

- **OD-1 — Production database hosting** (T202). **Decided 2026-10-18: Neon** (free tier to start). SQLite was
  considered and rejected the same day: the schema relies on Postgres (JSONB, plpgsql triggers, tsvector/pg_trgm
  search, row locks on the payment path), Cloud Run has no persistent disk for a database file, and T154's RLS
  backstop needs Postgres.
- **OD-2 — Make the GitHub repo private?** It is **public** and contains the 2026-10-16 audit and the product
  strategy/pricing docs. The audit's flaws are fixed (Phase 1). Owner, 2026-10-18: leave it public for now.
- **OD-3 — Redis in production** (T142, T161). Upstash (free tier, TLS) vs moving rate limits, tokens and caches to
  Postgres/in-memory and dropping Redis in production.
- **OD-4 — Stripe Connect charge model** (T169). Direct charges (tenant is merchant of record and carries dispute
  liability and Stripe fees) vs destination charges (platform carries dispute liability, more control). Affects
  Terms (T207) and fee display.
- **OD-5 — Money representation** (T166). Keep shopspring decimal or move to integer cents before the pgx port.
- **OD-6 — SMS provider and 10DLC** (T049, T181). Twilio vs Telnyx; per-tenant A2P 10DLC brand/campaign registration
  cost and lead time; whether ServicePro registers on tenants' behalf.
- **OD-7 — Beachhead confirmation** (T218). Confirm or change the commercial/B2B beachhead and the $99 Team price
  after 10 interviews; re-scope Phases 9–10 accordingly.
- **OD-8 — Cloud Run region.** Follows the Neon project's region (OD-1); pick it when creating the Neon project.
- **OD-9 — Auth provider now that the database is on Neon** (T158). D2 assumed Supabase Auth bundled with a Supabase
  database. Options: (a) **Neon's managed Better Auth** — included on Neon Free up to 60k MAU, data in our own Neon
  database (neon.com/pricing, checked 2026-10-18); Fleet decision 3 already names Better Auth as the self-owned
  option. Recommended. (b) Supabase Auth alone in a Supabase project (the free tier is full with 2 active projects,
  so Pro at $25/mo unless a slot frees up). (c) keep ServicePro's own JWT auth — ruled out by Fleet decision 3
  ("never hand-rolled auth"). Blocks T158 and T092.


---

## Dropped from the old TODO (2026-10-16 triage)

Untagged checklist items from the Feb 2026 TODO that were not carried forward. Items still worth doing were folded
into numbered tasks instead (named in that task's Notes): request correlation IDs, standardized API errors and input
validation → T223; email retry → T159; secrets rotation runbook → T151; Redis caching → T161 (not needed yet);
expanded health checks → T140; backend workflow/email/Stripe tests → T157; image optimization → T174; webhook
handling improvements → T109; Loki/BetterStack → T126; footer links → T138; barrel exports → T137; component
patterns → T163; useCallback/useMemo audit → T164; dependency scanning → T147; payment tracking TODOs → T118; quote
template spec → T060; job state machine remainder → T124; quote_status_machine.go:118 → T224; late-fee notification
→ T083; notification preferences → T091; audit logging → T093; 2FA → T092; public API / customer webhooks → T098,
T097; white-label branding → T184; mobile app/PWA → T173, T179, T180; dashboard KPI improvements → T088; email
templates expansion → T048; per-user rate limiting → T116; email open/click tracking → T089.

- **Email Client Enhancements (SES token bucket, batch, Redis queue, SES templates, CloudWatch metrics, SES domain
  management)** — moot: email moves to Resend only (T159), which handles rate limits, batching and domains.
- **Storage Client Enhancements (S3 manager multipart, semaphores, checksums, key strategy, SSE/KMS, bulk delete,
  versioning, client stats)** — moot at current scale with R2 (T160); only MIME/size validation kept.
- **Metrics Client Enhancements (histograms, summaries, runtime metrics, registry)** — the Prometheus client library
  already provides these; Cloud Run supplies request/runtime metrics.
- **HTTP Request Metrics** — Cloud Run request metrics + Sentry performance cover it.
- **Business Metrics Dashboard (Prometheus KPIs)** — done differently by T026 (Grafana over SQL views).
- **Infrastructure as Code (Terraform)** — not the fleet pattern; setup is scripted `gcloud` + runbooks (T151, T203).
- **Redis-Backed Email Queue / Email Batch Sending** — unnecessary with Resend and the scheduled-jobs runner (T141).
- **Frontend Service Layer Tests / Frontend Store Tests (membershipApi, billingApi, stores)** — the stores are being
  replaced by TanStack Query (T162); test the queries there.
- **Target 80% Backend Test Coverage** — replaced by workflow tests on Postgres (T157); coverage % is not a goal.
- **Implement GeoIP Lookup** — the access-control layer it served is dead code (deleted in T129); no feature needs it.
- **Customer Email Validation (MX/disposable checks)** — format validation comes with T223; MX/disposable checks add
  false rejections for business domains.
- **"Removed Features to Restore": Payment System files** — superseded by Stripe Connect (T169) and saved methods
  (T171).
- **"Removed Features": Invoice template management UI** — the backing tables don't exist and the service is dead
  (correctness audit 5.2); T080 covers reusable terms.
- **"Removed Features": Error Tracking (Sentry) middleware** — Sentry linkage is fixed in T126; the middleware exists.
- **"Removed Features": Health Checks – Advanced, Analytics System, Performance Monitoring packages** — `/health`,
  `/health/live`, `/health/ready`, `/metrics` and PostHog already exist; the custom packages would duplicate them.
- **"Orphaned Features": Payment Notification System** — replaced by the shipped reminder system (T034/T043, fixed in
  T121).
- **"Orphaned Features": Job Scheduling System (schedules CRUD API)** — the remaining real gap (nothing writes
  `schedules`) is fixed in T099.
- **"Orphaned Features": Tax Calculation Service (US state rate tables)** — hand-maintained state tables are a
  liability; tenants already define tax rates. Revisit with Stripe Tax if multi-jurisdiction customers appear.
- **"Orphaned Features": WebSocket Real-time Notifications** — no backend WS route; the frontend client is gated and
  deleted as dead code (T137). Revisit only with a concrete real-time need.
- **Old "Fix @ts-nocheck in Routes", "Remove Console.log", "Structured JSON Logging", "E2E Testing Setup"** — already
  complete (see Verified Complete).


---

## Verified Complete

### Before 2026-10-16

- **T001** — Recovery middleware Sentry integration. Panics captured with stack trace, request context, user ID; `RecoveryMiddleware` accepts `errortracking.Client`; `gin.Default()` → `gin.New()`; applied first in route setup. (Note 2026-10-16: the Sentry provider is not linked into the binary, so nothing is actually reported — T126.)
- **T002** — Centralized fetch() calls. `pdfGenerator.tsx`, `useQuoteCalculations.ts` now use the api service; no direct `fetch('/api/...')` calls remained at the time.
- **T003** — nginx security headers. CSP, HSTS (`max-age=31536000; includeSubDomains`), Permissions-Policy, Referrer-Policy `strict-origin-when-cross-origin` added to nginx.conf. (Ported to `_headers` for Workers in T150.)
- **T004** — GitHub Actions CI/CD. `checks.yml` (pre-commit lint, backend tests, frontend tests), `release.yml` (GitHub Release with changelog on tags), `deploy-release.yml` (tests + Fly.io backend + Cloudflare Pages frontend on tags). (Replaced by T145/T146.)
- **T005** — Prometheus /metrics endpoint. Enabled via `PROMETHEUS_ENABLED=true`, metrics client initialized in `cmd/main.go`, uses `pkg/clients/metrics/prometheus/`.
- **T006** — Structured JSON logging. Custom logging client at `pkg/clients/logging/` (Mock, BetterStack, CloudWatch, Loki providers), global logger (`SetDefault()`/`L()`), all `fmt.Printf`/`log.Printf` migrated; JSON in production, human-readable in development; standard fields timestamp, level, message, request_id, user_id, tenant_id. Exceptions: config/config.go (circular import), logging package files.
- **T007** — Fix `any` types. Replaced 50+ `any` types, created `src/utils/error.ts`, fixed type definition files (template.ts, chart.ts, recurring/types.ts, calendar/types.ts), pages, components, stores and all test mocks (e.g. `MockCardElementProps`).
- **T008** — Remove @ts-nocheck. Removed from `routes/index.tsx` (35+ `@ts-expect-error` comments on dynamic imports) and `utils/performance.ts`; fixed `loadable()` signature; added browser API interfaces (`PerformanceEventTiming`, `LayoutShiftEntry`, `ExtendedPerformanceObserverInit`, `DocumentWithPrerendering`).
- **T009** — Enable noUnusedLocals/noUnusedParameters. 17 issues fixed: 7 unused `React` imports, unused variables (`_chartRef`, `_handleItemsChange`, `_bulkAssignMutation`, `_formatValue`, `_navigate`, `_API_BASE_URL`), dead `_offlineFallback` route, `_`-prefixed unused params.
- **T010** — Error tracking middleware. Already implemented via `RecoveryMiddleware` in `error_handler.go` (panics → error tracking with method, path, query, client IP, user context; first in chain). HTTPMiddleware not needed since nothing else calls `CaptureException`.
- **T012** — Bundle size monitoring in CI. Markdown reports in CI step summaries (`analyze-bundle.cjs`), pre-push hook validation, release script gate. Thresholds 500 KB warn / 750 KB fail (total JS gzipped); per-chunk budgets for vendor splits.
- **T013** — Frontend test coverage 70%+ (52 test files, 1251 tests; 69.87% statements, 72.49% functions, 70.99% lines). (Suite is red as of 2026-10-16 — T136.)
- **T014** — Integration tests for critical workflows (auth, customer CRUD, job lifecycle, quote workflow, invoice & payment) in `frontend/tests/integration/`.
- **T015** — OpenAPI/Swagger documentation. Backend generates `swagger.json` via swag, Swagger UI at `/api/docs`, frontend generates TypeScript types via `npm run generate:api`.
- **T016** — Frontend bundle optimization. Gzipped bundle 720 KB → 322 KB (-55%): lazy-loaded `@react-pdf/renderer`, fixed `manualChunks` boundary matching (react-big-calendar was landing in the catch-all vendor chunk), dedicated `vendor-calendar`/`vendor-pdf`/`vendor-table` chunks, fixed `analyze-bundle.cjs` ESM issue.
- **T017** — Query performance monitoring. Custom GORM logger (`pkg/database/logger.go`) with slow-query detection (≥100 ms WARN, ≥1 s ERROR); Prometheus `db_query_duration_seconds`, `db_queries_total`, `db_query_errors_total` with operation/table labels; pool gauges (`pkg/database/pool_metrics.go`: open/idle/in-use/max/wait count/wait duration); `DB_SLOW_QUERY_THRESHOLD`, `DB_QUERY_ALERT_THRESHOLD`, `DB_LOG_ALL_QUERIES`; deferred wiring via `SetMetricsClient()`.
- **T018** — Dead code elimination. Removed ~2,700 lines (`components/recurring/`, `components/subscriptions/`, `components/health/`, `utils/performance.ts`, `hooks/useErrorTracking.ts`, `hooks/useAnalytics.ts`, `hooks/useKPI.ts`); ts-prune with a baseline of 116 known unused exports, `npm run dead-code` / `dead-code:update`, pre-commit hook and CI job.
- **T019** — Calendar view for job scheduling. Read-only `JobCalendarPage` at `/jobs/calendar` using `GET /v1/jobs/scheduled?start=&end=`, status colour mapping, click to job detail; drag-drop disabled (T027). Files: `JobCalendarPage.tsx`, `jobService.getScheduledJobs()`, `calendar/Calendar.tsx`, `calendar/types.ts`, routes.
- **T020** — Conflict detection API endpoint. `POST /v1/conflicts/check` wires the existing `ConflictDetector`; `conflict_handler.go` with Swagger annotations, `conflicts_routes.go` with auth + tenant + permission middleware; tenant scoping fixed in `GetByTechnicianAndDateRange()`/`GetByDateRange()`; `TenantID` set from auth context; detects technician overlap, business hours, workload excess with resolution suggestions; Swagger + TS types regenerated.
- **T021** — Extract duplicate file download utility. `src/utils/fileDownload.ts` with `downloadFile()`, `downloadCSV()`, `downloadJSON()`, `downloadPDF()`, `openBlobInNewTab()`, `downloadFromDataURL()`; 10 files refactored.
- **T022** — Extract duplicate URLSearchParams builder. `src/utils/queryParams.ts` (`buildQueryParams()`, `buildQueryString()`, `appendQueryParams()`); refactored customerReportService (7), revenueService (6), invoiceService (1), customerService (1).
- **T023** — Consolidate cache hooks. Generic `useStorageCache` with a `CacheAdapter` interface in `src/hooks/useCache.ts`; `useLocalCache` and `useSessionCache` share the core.
- **T024** — Full Sentry integration. GlitchTip (Sentry-compatible) in docker-compose with its own Postgres + Celery worker; frontend Sentry SDK in `main.tsx`, `ErrorBoundary`, user context on login/logout/checkAuth in `authStore.ts`; backend via factory pattern; no DSN = no-op. Source maps deferred to T101.
- **T025** — PostHog product analytics. Frontend `posthog-js` analytics service (`initAnalytics`, `trackEvent`, `identifyUser`, `resetUser`), typed event constants, `PageViewTracker`, `useAnalytics`; workflow events quote_created/sent/accepted/rejected, job_created/completed/status_changed, invoice_created/sent, export_downloaded. Backend `posthog-go` pluggable client (PostHog + mock); Stripe webhook → `payment_received`. Env `POSTHOG_API_KEY`, `POSTHOG_HOST`, `VITE_POSTHOG_*`.
- **T026** — Grafana KPI dashboard. Grafana OSS replaced Metabase in docker-compose (port 3001); provisioned Postgres datasource and `grafana/dashboards/platform-overview.json`; 8 views in `001_schema.sql` (`v_platform_overview`, `v_tenant_growth_daily`, `v_feature_adoption`, `v_job_metrics_daily`, `v_invoice_metrics_daily`, `v_quote_conversion`, `v_revenue_daily`, `v_ar_aging_summary`); 12 panels.
- **T031** — A/R aging buckets report. Backend `models/ar_aging_report.go`, `services/ar_aging_service.go`, `handlers/ar_aging_handler.go`; frontend `types/arAging.ts`, `services/arAgingService.ts`, `pages/Reports/ARAgingReportPage.tsx`. Buckets Current, 1-30, 31-60, 61-90, 90+, drill-down to invoices, CSV export. (Returns empty for API-created data until T102 writes `tenant_id`.)
- **T032** — "Who Owes What" report. Customer filter dropdown on the A/R Aging report (customer summary endpoint `/api/v1/reports/ar-aging/customers`, `min_amount` filter); fixed Gin UUID query binding (`ShouldBindQuery` can't bind `*uuid.UUID` → string field + `ParseCustomerID()`).
- **T034 + T043** — Payment reminder automation (merged). `payment_reminders` table, `PaymentReminderService` with hourly background worker, escalating tone (friendly → firm → final), email/SMS with customer preference resolution, dedup, manual trigger; API `GET/POST /invoices/:id/reminders`, `GET/PUT /settings/payment-reminders`; Notifications tab in Org Settings; reminder history + manual send on Invoice Detail. (2026-10-16: the reminder query has an ambiguous column and has never sent; SMS client never wired — T121.)
- **T053** — "Repeat Job" button on customer detail. `?clone=` param on JobDetailPage pre-fills from the source job (title, type, priority, duration, notes, instructions, materials); customer detail lists recent jobs with "Repeat Job"/"View"; JobActionsMenu "Repeat Job"; "New Job" from the customer page pre-selects the customer.
- **T100** — Business hours & timezone in org settings (commit 3df4280 "added business hours and timezone"). Admins set an IANA timezone and business hours/days in org settings. The backend does not read these settings yet — see T099 (conflict detection, days overdue) and T127 (other settings).

Untagged items completed before the indexed system:

- Rate limiting — multi-tier with Redis backing, applied globally via `DynamicRateLimit()`.
- PDF generation — quotes, invoices, receipts with S3 storage and email attachments (`document_pdf_service.go`, `services/pdf/templates.go`; download endpoints `GET /api/v1/quotes/:id/pdf`, `/invoices/:id/pdf`, `/invoices/:id/receipt/pdf`).
- SMS integration — TextBelt, SNS, Mock providers with pluggable architecture.
- httpOnly cookies — SameSite=Lax for CSRF protection.
- Security headers (backend) — CSP, HSTS, X-Frame-Options in middleware.
- Error tracking client — Sentry integration at `pkg/clients/errortracking`.
- `.env` in `.gitignore` — properly excluded.
- Console.log stripping — Terser `drop_console: true` in production build (`frontend/config/optimization.ts:64`).
- Migrate `db.Raw()` to GORM (SQL injection risk).
- Cypress E2E testing — auth, customers, jobs, invoices, quotes, payments.
- Graceful shutdown — SIGTERM/SIGINT handling with 30 s timeout in `cmd/main.go`.
- HTTP server timeouts — ReadTimeout, WriteTimeout, IdleTimeout, ReadHeaderTimeout via `http.Server`.
- MaxMultipartMemory — configurable via `SERVER_MAX_MULTIPART_MEMORY`, applied to the router.
- TrustedProxies — configurable via `SERVER_TRUSTED_PROXIES`, applied to the router. (Unsafe empty default — T107.)
- Deep health checks — `/health`, `/health/live`, `/health/ready` with DB + Redis connectivity checks.
- localStorage token fix — `ConflictChecker.tsx` uses the api service with httpOnly cookies.
- Dependency vulnerability scanning — Dependabot for Go and npm, weekly, auto-merge for patch-level security updates. (Prefixes and blocking issues — T147.)
- Job status state machine — `POST /api/v1/jobs/:id/transition`, `GET /api/v1/jobs/:id/status-history` (`models/job_status.go`, `job_service.go`, `job_handler.go`).

Dated history from the original "Completed Items" list:

- Initial project setup (2024-01-15)
- Basic authentication flow (2024-01-20)
- Customer CRUD operations (2024-01-25)
- Membership system with Stripe integration (2025-01-25)
- Subscription proration preview (2025-01-25)
- Local email development setup with Mailpit (2025-01-25)
- Security headers middleware (2025-01-25)
- CORS configuration with environment-specific origins (2025-01-25)
- Job status state machine with transition validation (2025-01-29)
- Job status history tracking (2025-01-29)
- PDF generation for quotes, invoices, and receipts (2025-01-30)
- PDF attachments in quote/invoice/receipt emails (2025-01-30)
- PDF download endpoints for quotes, invoices, receipts (2025-01-30)
- Clone quote functionality (2025-01-30)
- Quote accept/decline actions in UI (2025-01-30)
- Dropdown menus with portal rendering (fixes table overflow clipping) (2025-01-30)
- Cypress E2E testing setup with tests for auth, customers, jobs, invoices, quotes, payments


### 2026-10-16 — audit remediation session

- **T102** — Tenant-scope all business data. New `internal/tenancy` GORM plugin scopes every read/update/delete/upsert on tenant tables to the context tenant, sets `tenant_id` on create and errors when no tenant is present; raw SQL rewritten or explicitly filtered; children reached only via parents; cross-tenant customer references rejected; exports tenant-checked and downloadable only by the requester; A/R aging returns data. Migration `000003`: `tenant_id NOT NULL`, revenue tables get `tenant_id`, job/quote/invoice numbers + customer email unique per tenant, composite tenant indexes. SQLite isolation tests (customers, jobs, quotes, invoices, payments, exports). Customer search rewritten in GORM (its SQL functions never existed); cohort analysis returns 'unavailable' (missing view, T123).
- **T103** — Authorize tenant member, invitation and role endpoints; no self-escalation. Tenant/member/invitation/membership-tier endpoints require active membership of `:id` (`RequireTenantFromPath`) + `tenants.read/manage/update` / `membership.update`; super_admin never assignable (incl. invitations, rechecked on accept); no granting above your own role, managing someone who outranks you, or changing your own role.
- **T104** — Evaluate permissions per tenant. Permission checks take the tenant and count only the user's role in that tenant plus platform `user_roles`; cache keyed by user+tenant; `RequireTenant` on every business route group; removed `AccessControlMiddleware`, permission decorators and the `jwt_tenant_id` fallback.
- **T105** — Remove or lock the raw Stripe passthrough endpoints. Raw Stripe passthrough endpoints (`/stripe/payment-intents*`, `/stripe/customers*`, `/stripe/refunds`, webhook stats) removed — the frontend never called them; only `/stripe/webhooks` remains.
- **T106** — Backend-enforced invite-only registration; reject non-edge traffic in production. `REGISTRATION_MODE` (`open` dev / `invite_only` elsewhere), hashed invite codes (migration `000004`), `/api/v1/admin/invite-codes` (platform super admin), `go run ./cmd/invite` / `make invite-code`; invitation registration requires a matching email; in production `/api/*` without the edge secret → 403 (health/version/internal jobs exempt). Frontend follow-through: T227.
- **T107** — Edge client-IP middleware and a safe trusted-proxy default. Edge client IP (`middleware.EdgeClientIP` / `ClientIP`, trusts `X-SP-Client-IP` only with a valid `X-SP-Edge-Secret`) landed with T142's platform work; Gin now calls `SetTrustedProxies(nil)` when `SERVER_TRUSTED_PROXIES` is empty; unwired `RATE_LIMIT_TRUSTED_PROXIES` deleted.
- **T108** — Escape PDF export HTML; drop wkhtmltopdf local file access. PDF export HTML built with `html/template`; wkhtmltopdf runs with local file access and JavaScript disabled; unused SSRF-prone `ProcessImage` removed.
- **T109** — Stripe webhook reliability: persistent dedupe, accept retries, body limit, idempotent payment. Persistent Stripe webhook dedupe (`stripe_webhook_events`, migration `000006`: claim with insert-on-conflict, release on failure); 5-minute age rejection removed; 1 MiB body cap; `MarkInvoiceAsPaid` locks the invoice row and is idempotent per checkout session (unique index). Legacy `/webhooks/stripe` still pending (T122).
- **T110** — Token hardening: access vs refresh type, revoke on logout and password reset. Only access tokens authenticate API calls; JWTs carry `jti` + token version (migration `000005`); logout and password reset bump the version (logout = every device).
- **T111** — Login hardening: no lockout DoS, uniform errors and timing. Failed logins throttled per (account, IP) with exponential backoff, 5-minute account cool-down after 50 failures, 100 attempts / 15 min per IP; dummy bcrypt for unknown emails; unknown / wrong password / locked all return the same 401; 'not verified' only after a correct password; throttling → 429.
- **T112** — Neutralise CSV/Excel formula injection in exports. Cells starting with `= + - @ \\t \\r` get a leading `'` in every CSV/XLSX export, including hand-built report CSVs; tests.
- **T113** — Fail closed when no email or storage provider in production; never log tokens. Production startup fails without a real email provider (`RESEND_API_KEY`); the mock provider never logs token links.
- **T114** — Lock down /health detail, /metrics, Swagger and debug mode in production. Outside development: `/health` + `/health/ready` return only `{status}` unless loopback or `INTERNAL_JOBS_SECRET`; `/metrics` requires `METRICS_TOKEN` (not registered without it); Swagger UI only with `ENABLE_SWAGGER=true`; Gin release mode.
- **T115** — Secure cookies by default and CSRF protection. `COOKIE_SECURE` defaults true outside development; refresh cookie always SameSite=Strict; cookie-authenticated mutations require `X-Requested-With: XMLHttpRequest` (sent by `frontend/src/services/api.ts`); CORS allowed headers include `X-Requested-With`, `X-Tenant-ID`.
- **T116** — Low-severity security sweep. F-17: central `publicError()` replaces raw `err.Error()` at 146 handler sites; F-18: SES webhook routes removed; F-19: profile pictures over 4000×4000 rejected before decode; F-21: public invoice-by-number route removed (T104); F-22: global limiter recognises signed-in users. F-24: emails no longer logged (user_id instead), `QueryLogger.ParamsFilter` keeps bound values out of logged SQL; F-25: unrouted `download_handler.go` and `AccessControlMiddleware` deleted. F-23 (import size/row caps) moved to the import task (T193).
- **T117** — Remediate dependency vulnerabilities and bump the Go toolchain. Bumped x/net, x/text, excelize, grpc, otel, quic-go, AWS SDK modules; `toolchain go1.27.2`; Dockerfile builder `golang:1.27-alpine`; `govulncheck ./...` reports no vulnerabilities.
- **T118** — Recording a payment updates the invoice balance and status. Recording a payment updates the invoice's paid amount, balance due and status in the same transaction (commit fbb1c5f).
- **T119** — Stripe Checkout charges the true balance due. `ToMinorUnits`/`FromMinorUnits` round and honour zero/three-decimal currencies; Checkout charges total − amount paid as one line with an idempotency key (invoice+amount+hour); partially paid/viewed/overdue invoices payable online; webhook converts with the session currency (4b42f0a, 3a32e63). Currency fixed to `usd` behind `invoiceCurrency()` until T127.
- **T120** — Invoice and job numbering survives soft-delete and concurrency. Per-tenant `tenant_counters` (migration `000008`, backfilled; `models/numbering.go`) allocate INV-/QUO-/JOB-NNNNNN inside the create transaction — never reused, skips numbers taken by old schemes; dead invoice-number trigger and quote sequence dropped (0d4caa6). Prefixes from tenant settings remain in T127.
- **T121** — Payment reminders actually send. Each run sends only the latest due reminder per invoice; failed or stale (>1h) pending claims retried; ignored errors checked; SMS client wired through `SetupServices`; worker `Stop()` idempotent; manual reminders numbered separately (migration `000010`, `payment_reminders.is_manual`, unique `(invoice_id, is_manual, reminder_number)`); frontend shows "Manual #n" (81fab9f).
- **T122** — Fix /webhooks/stripe panicking on subscription and invoice events. `customer.subscription.*`, `invoice.paid`, `invoice.payment_failed` handled on `POST /api/v1/stripe/webhooks` (`services/billing_webhook_handlers.go`; unknown customer acked, lookup failure → 503 so Stripe retries); legacy `/webhooks/stripe` handler + route, dead `services/stripe/webhook_handler.go` and unwired `handlers/webhooks` deleted (c57f10d). Point the Stripe dashboard at the single endpoint when hosting (T204).
- **T123** — Add the missing SQL functions and views behind customer search and reports. Migration `000011` trigram indexes for customer search/autocomplete; customer metrics computed in Go from tenant-scoped loads (`services/customer_metrics.go`) and refreshed when missing/stale, so top/at-risk/segment reports fill; filtered-summary SQL error and retention rate fixed; tenant-scoped cohort retention report implemented in Go (4eeb05a, 6c02870, 829f143).
- **T124** — Enforce invoice status transitions; unify the two job state machines. Invoice transition table (paid/cancelled/refunded read-only → 409; cancel voids the pay link; changes audited); one job state machine via `JobService.applyTransition` + `models.ValidateTransition` (duplicate `pkg/validators/job.go` rules deleted); status + history written in one transaction; `GET /jobs/{id}/allowed-transitions`; jobs follow their invoice completed→invoiced→paid and back (8cc6579, f3ca03a, 104f4ea, 9a5d692, a0fea72). Frontend: JobDetailPage status dropdown removed; cancel/hold need a reason.
- **T125** — Handle refunds in the ledger. `invoice_refunds` table (migration `000009`) + `POST /invoices/{id}/refunds` (`payments.refund`): refund lowers amount paid under the invoice lock → partially paid / refunded, paid job back to invoiced; Stripe `charge.refunded` applied once per payment intent on the deduplicated processor (c9c4e8e, c66d173). UI action: T232.
- **T126** — Backend Sentry actually initialised; wire or delete the Loki/BetterStack/OTel stubs. Sentry provider linked into the binary (blank imports + test); outside development logs are JSON lines on stdout with Cloud Logging keys; Better Stack/Loki/CloudWatch log shippers, CloudWatch metrics and OTel tracing stubs deleted with their modules/config; `main` → `run() error` so Sentry/PostHog flush on startup failure (9c49364, 40bf2c2, bede549, 4f43fa4).
- **T127** — Backend reads tenant settings: currency, number prefixes, defaults. Tenant prefixes drive INV-/QUO-/JOB- numbering (sanitised, fallback defaults); migration `000013` adds `currency` to invoices/quotes (set from the tenant at create, quote→invoice inherits; Stripe Checkout and the public invoice use it); invoice/quote create apply the tenant's default payment term and tax rate (tenant-scoped, active rows only; quote `tax_rate` now optional); calendar-day dates + tenant timezone (earlier); business hours labelled 'Coming soon' (b4c2d6f, a4c3bbe, 2a6f3ab, 432935d, 6039d8f, cbf9d94). Settings UI → T235; currency display → T236.
- **T128** — Exports safe on multiple instances (object storage + persisted progress). Exports run in the request (2-min limit) with progress/status persisted on `export_jobs` (cancel sticks; `/progress` + SSE read the row); files stored via `ExportFileStore` in object storage at `exports/<tenant>/<job>/<file>` (local `./exports` only without a storage client), download streams the stored file, cleanup deletes it; tenant context required (da7d052, ff8c0c8). Very large exports may hit `SERVER_WRITE_TIMEOUT` — revisit with a job runner.
- **T129** — Backend dead-code removal and a stricter linter. ~39k lines of unreachable backend code deleted across 21 commits (unlinked packages: templates, analytics, api/validators, services/recurring, services/subscriptions, pkg/search, pkg/notifications; dead models.Payment/PaymentRefund + webhook records; unrouted assignment handler/service (8 documented-but-never-served /assignments paths dropped from the spec); unused middleware, limiters, Stripe wrappers, health/PDF/model helpers; legacy database/sql user repo); golangci-lint now runs unused, ineffassign, sqlclosecheck, staticcheck, gocritic, errorlint, errcheck, noctx, rowserrcheck and gosec (narrow justified nolints; server-written files 0600/0750); real bug fixed: A/R aging CSV days-overdue ≥10 written as punctuation (42136cb); golangci-lint pinned v2.14.0 (65c8dd3 … 017304c, 7b31816). Follow-ups T241–T244.
- **T130** — Login error swallowed by the refresh interceptor. The 401 refresh/redirect interceptor skips auth endpoints (login, register, refresh, logout, reset, verify) so login errors show; concurrent 401s share one refresh; tests (`src/services/__tests__/api.interceptor.test.ts`).
- **T131** — Public payment result pages for the /pay/:token flow. `/invoices/payment-success` and `/invoices/payment-cancelled` (the URLs the backend already sends to Stripe) moved outside the auth guard; test.
- **T132** — Clear tenant and user data on logout and tenant switch. Persisted React Query cache keyed by app version + user + tenant; logout, session expiry and a different user's login clear the in-memory and persisted cache, tenant state, `current_tenant_id` and `X-Tenant-ID`; tenant switch clears the query cache; tests.
- **T133** — Job times entered and shown in the tenant's timezone, stored as true UTC. Job form, limits, list/detail/history/conflict displays and calendar use the tenant timezone (`settings.timezone`, browser fallback) via `Intl`; storage stays UTC; tests for EDT/EST/DST/Pacific/Honolulu + two job integration tests.
- **T134** — Strip sensitive query params from analytics and error breadcrumbs. Query-param allowlist applied to PostHog page views + `before_send` and to Sentry events, transactions and breadcrumbs — tokens, codes, emails and Stripe session ids no longer leave the browser; tests.
- **T135** — Frontend medium-severity sweep. Error + retry states on Customers/Jobs/Quotes/Invoices; customer pickers page past the 100 cap; one money utility (`src/utils/money.ts`: `formatDollars` for API decimals, `formatCents` for Stripe cents); tax rate display rounded; Sentry trace targets same-origin `/api` + localhost; `AddPaymentMethod` (raw fetch) removed as dead code. Not fixed: the quotes list search box is ignored by `getQuotes` (follow-up).
- **T136** — Green frontend test suite and lint. In-memory storage stubs make the suite pass on Node 20/22 and ≥25; `document.elementFromPoint` polyfill; broken tests repaired, integration tests rewritten to the current UI (many silently passed behind `if (element)` guards); `retry: 1` removed; lint at zero warnings. 850 passed, 1 skipped (invalid-email login test blocked by native validation).
- **T137** — Frontend dead-code removal (~18.5k unreachable lines). ~19k lines of unreachable source (~43k with tests/stories) removed after tracing imports from `src/main.tsx`: templates, payments, charts, kpi, filters, export, skeletons, realtime/WebSocket, unregistered service worker + PWA plugin; dropped `@react-pdf/renderer`, workbox, `vite-plugin-pwa`, `@tanstack/react-table`; dead-code baseline regenerated.
- **T138** — Preview polish: honest contact form, legal pages marked draft, favicon. Contact form opens a mailto to `VITE_SUPPORT_EMAIL` (default support@sharkfins.xyz) instead of faking success; placeholder phone/address removed; Privacy/Terms marked 'Draft — pending legal review' (fixed date 2026-10-16); SVG favicon + apple-touch icon.
- **T139** — golang-migrate baseline (000001), cmd/migrate, dev-only startup migrations. golang-migrate baseline `000001_initial_schema` (snapshot of the old 001_schema.sql) + `000002` (fixes `v_ar_aging_summary`, which never applied), embedded via `migrations/embed.go`; `cmd/migrate` (up/down/version/force; down refused in production); legacy DBs adopted at v1; `RUN_MIGRATIONS_ON_STARTUP` (dev default on). Dev seed moved to `backend/seeds/dev_seed.sql`.
- **T140** — /version endpoint and schema-aware readiness. Root `GET /version` `{git_sha, app_version}` from `APP_VERSION`/`GIT_SHA`; hardcoded `/api/v1/version` removed; `/health` + `/health/ready` include version; readiness reports schema ok/ahead/behind/dirty and returns 503 on behind/dirty (rechecked every 30s).
- **T141** — Background tickers → internal job endpoints + scheduled-jobs workflow. `POST /internal/jobs/payment-reminders` (header `X-Internal-Jobs-Token` = `INTERNAL_JOBS_SECRET`, 404 when unset); reminders claimed with `INSERT … ON CONFLICT DO NOTHING` before sending (re-run + concurrent tests); `RUN_BACKGROUND_WORKERS` gates in-process tickers (dev default on); `.github/workflows/scheduled-jobs.yml` hourly, inert until `vars.API_URL`. The analytics aggregator is never started and the webhook 'cleanup' tickers only evict per-instance caches, so they got no endpoints. Also fixed the ambiguous `status` column that stopped every automatic reminder (part of T121).
- **T142** — Redis optional at boot. Redis optional: unset/unreachable at boot → warning + embedded in-memory store; readiness reports `not_configured`/`in_memory_fallback`. Caveat: reset/verification tokens and rate limits are then per-instance — production needs T161.
- **T143** — Cloud Run-ready Dockerfile and .dockerignore. `backend/Dockerfile`: golang:1.26-alpine builder, CGO off, trimpath; alpine:3.22 runtime, non-root uid 10001, ca-certs + tzdata, `GIT_SHA`/`APP_VERSION` build args → env, builds `server` + `migrate`, HEALTHCHECK `/health/live`; new `.dockerignore` (the old one excluded `pkg/`). Not yet built under Docker locally (daemon was down) — CI's docker-build + Trivy job is the gate.
- **T144** — DB pool sizing and pooler (simple protocol) support. `DB_MAX_OPEN_CONNS` default 10 outside development (25 in dev); `DB_SIMPLE_PROTOCOL=true` or a :6543 URL enables pgx simple protocol for Supabase's transaction pooler; migrations always use simple protocol.
- **T145** — CI rewrite: lint, tests with Postgres+Redis, govulncheck, OpenAPI drift, frontend, Docker + Trivy. `.github/workflows/ci.yml`: golangci-lint (pinned action), go test -race with Postgres 16 + Redis 7 services, db-integration (continue-on-error until the suite compiles), govulncheck, openapi-drift, frontend typecheck/lint/vitest/build/npm audit/dead-code/bundle, docker build + Trivy; `contents: read`, timeouts, PR-only cancellation. Several gates are red on existing code until Phase 1/2 land (T117, T129, T136).
- **T146** — Tag-driven release-deploy pipeline, VERSION scripts and RELEASING.md. `release-deploy.yml` (guard VERSION==tag → migrate → Cloud Run → Worker → GitHub Release; deploy jobs skipped until `vars.GCP_PROJECT_ID`), `migrate.yml`, `scripts/bump-version.sh` (understands legacy `minor:`/`major:`), `scripts/gh-release.sh`, `scripts/deploy-cloudrun.sh`, `RELEASING.md`, `make bump`/`release-check`.
- **T147** — Dependabot deps() prefixes, automerge and weekly security scan. Dependabot `deps(go)`/`deps(npm)`/`deps(actions)` prefixes, `dependabot-automerge.yml`, weekly `security-scan.yml`.
- **T148** — Versioned bash git hooks replace python pre-commit; commit-msg prefix contract. Python pre-commit framework replaced by versioned `scripts/git-hooks/{pre-commit,commit-msg}` + `make install-hooks`: golangci-lint, Go tests/build and OpenAPI drift (via `go tool swag`) on staged backend changes; eslint + tsc on staged frontend changes; commit-msg enforces the prefix contract.
- **T149** — Repo hygiene: tracked artifacts, Fly/nginx/cliff removal, .gitignore, single Makefile. Untracked `frontend/bundle-report.html` and test-export CSVs; deleted `backend/fly.toml`, `backend/tests/Makefile`, `cliff.toml`, `scripts/release.sh`, `scripts/run-tests.sh`, `scripts/setup.sh`, `frontend/{Dockerfile,nginx.conf,security-headers.conf}`; tightened `.gitignore`; CHANGELOG.md frozen at v0.7.0; compose images pinned (MinIO left on :latest — no longer publicly pullable, see T160).
- **T150** — Cloudflare Workers static assets + /api edge proxy Worker + _headers. `frontend/wrangler.jsonc` (worker `servicepro-web`, SPA fallback, `run_worker_first` for `/api/*` + `/version`, custom domain servicepro.sharkfins.xyz, workers.dev/previews off, `EDGE_PROXY_SECRET` required), `frontend/worker/{index,proxy}.ts` (503 when `API_ORIGIN` unset, raw body streamed, `X-SP-Client-IP`/`X-SP-Edge-Secret` set and spoofed copies stripped) with 22 tests, `public/_headers` (CSP/HSTS/cache) + `.assetsignore`; same-origin `/api` default; verified with `wrangler deploy --dry-run` and `wrangler dev`. Not deployed (T204).
- **T151** — Ops docs: Cloud Run setup, deploy/rollback, backups + nightly pg_dump → R2 workflow. `docs/ops/cloud-run-setup.md`, `docs/ops/deploy.md` (rollback incl. Workers versions), `docs/ops/backups.md`; `.github/workflows/db-backup.yml` nightly pg_dump → R2 `servicepro-backups`, skipped until the `MIGRATION_DATABASE_URL` secret exists (activation + restore drill: T209).
- **T152** — Commit generated OpenAPI and TS types for drift gating. `backend/docs/{swagger.json,swagger.yaml,docs.go}` and `frontend/src/types/api.generated.ts` committed; swag pinned via go.mod `tool` directive (`go tool swag`); `make openapi-check`, pre-commit hook and CI `openapi-drift` gate on them.
- **T164** — React 19, React Router 7, Vite major, ESLint 9, Headless UI 2. All frontend majors landed in small commits: React 19.3 + react-dom 19 (a381e00), React Router 7.18 (`react-router-dom` removed), Headless UI 2 (Modal on v2 components), Vite 8 / Rolldown with the same vendor chunks (`codeSplitting.groups`, b38c6b6), @vitejs/plugin-react 6, Vitest 5, jsdom 30, jest-dom 7, ESLint 10 + typescript-eslint 8 + react-hooks 7 + react-refresh 0.5 (2cf082c), zod 4 via `zod/mini` + @hookform/resolvers 5 (c7f502c), Stripe.js 10 / react-stripe-js 7, date-fns 4, react-easy-crop 6, lucide-react 1.x; GitHub Actions checkout/setup-go/setup-node 7, wrangler-action 4; Node 24 in CI. Bundle budgets re-based with reasons (vendor-react 85 KB, vendor-observability 195 KB — CI now measures the real Sentry chunk); `analyze-bundle.cjs` chunk-name bug fixed. Follow-ups: T248 (Sentry 11), T249 (CJS interop flag), `react-hooks/set-state-in-effect` with T162.
- **T202** — Decide production database hosting (OWNER DECISION). Owner chose **Neon** Postgres (free tier to start; move to a paid plan when it outgrows the free limits). Recorded in D3 and OD-1; Cloud Run goes in the Neon region (OD-8). Neon has no bundled auth, so the T158 auth plan is reopened as OD-9.
- **T223** — Request IDs and a standard API error envelope. Request-ID middleware mounted first (inbound X-Request-ID kept if 8–128 safe chars, else UUID) → response header, context, every log line, access log, Sentry tag; one envelope `{error: code, message, request_id, fields?, details?}` written through `internal/api/apierror` with typed domain errors in `internal/errors` (409 conflicts incl. quote/job status, 404 not-found, 400 validation with fields / malformed_json, generic 500 for infra); all handler groups + middleware converted, JSON 404 for unknown /api routes, `docs/api/errors.md`; frontend reads the envelope and shows 'Reference: <id>' on 5xx (67cce6a … 6ad0516). Follow-ups T238–T240.
- **T224** — Quote correctness: lost customer edits, accept race, expiry, editable sent quotes. Quote update keeps customer changes (same-tenant check); accept/decline are single conditional updates (send/accept/decline never worked before — validation hook ran on an empty model); subtotal = sum of rounded item totals; only drafts editable + `POST /quotes/{id}/revise`; `POST /internal/jobs/quote-expiry` hourly in scheduled-jobs.yml (aeaef27, e04a7c3, 5ddeb72, fc15df2, 9a29a32).
- **T225** — Invoice totals: one server-side recalculation, tax/discount order, no double invoicing. Removed GORM `default:true` on 5 models (46b6211); single totals calculation (`services/invoice_totals.go`: line discounts, invoice discount spread before tax) recalculated in create/update/add-line transactions with partial updates and total ≥ amount paid; PDF lines sum to subtotal; no double invoicing (row lock, accepted quote, matching customer) (a7634b5, 1d47ac7). `JobMaterial.Billable` → T233.
- **T226** — Tenant creation and invitation acceptance in transactions; per-tenant uniqueness. Migration `000012`: tenant slug unique among live tenants, one pending invitation per `(tenant, LOWER(email))`; `TenantRepository.CreateWithOwner` creates tenant + owner membership + default subscription in one transaction (owner role by name; debug query and hard-coded UUID gone); invitation accept runs in one transaction with the invitation row locked, persists expiry, reactivates an existing member with the invited role (458209c, 591a6ed, 1489c79).
- **T227** — Frontend follow-through for the Phase 1 auth changes (invite codes, 429, logout-everywhere). Registration form has an `invite_code` field (prefilled from `?invite=`/`?code=`/`?invite_code=`, hidden for tenant invitations) and distinct messages for invite-only / invalid code / invalid invitation and 429; login shows a generic 429 message and honours a safe same-origin `returnUrl` (`getSafeReturnUrl`); logout says it signs out every device; Cypress cases for the removed Stripe endpoints deleted; tenant settings/members/membership/billing pages show a permission message on 403 and hide actions built-in roles can't use (`src/utils/tenantPermissions.ts`); accept-invitation offers sign-up. Quotes search needs backend support (T228); invite-code/invitation API gaps → T229; effective permissions + org deletion → T230.
- **T228** — Quotes list: backend search (q) and sort params. `GET /api/v1/quotes` takes `q` (case-insensitive quote number / customer first, last, company or full name; `%`/`_` escaped; customer subquery tied to `quotes.tenant_id`), allowlisted `sort_by` (created_at, updated_at, quote_number, status, total, valid_until) and `sort_order` (bad values → 400); QuotesPage sends a debounced `q` and only API sort columns; tenant-isolation tests (f24876c, 05fc0e4). Verified on SQLite only — Postgres run pending.
- **T229** — Invite and invitation API polish: distinct invalid-code error, encoded links, message field. Invite-only registration returns distinct 403 codes `registration_closed` (nothing supplied) / `invalid_invite_code` / `invalid_invitation`, and RegisterPage maps them directly; invitation links built with `url.Values` (`inviteAcceptURL`/`inviteRegisterURL`) so emails with `+`/`&` survive; invitation handler errors are `{error: <code>, message: <text>}` (43ba167, 302f8e4, e9cf6fc).
- **T230** — Expose the caller's effective permissions per tenant; decide who can delete an organization. `GET /api/v1/tenants/{id}/me` → `{tenant_id, role_id, role, role_level, permissions[]}` from `PermissionChecker.EffectivePermissionNames` (membership-gated, cross-tenant test); frontend `useTenantPermissions` (React Query) gates Team Members, Org Settings and Membership/Billing on specific permissions, delete button only with `tenants.delete` (6e0b27c, e159e51). Owner decision on org deletion split to T231.
- **T232** — Frontend money calls match the API: record payment URL, cancel reason, refunds, quote revise. Record Payment / Cancel Invoice (reason required) / Record Refund dialogs and a Payments + Refunds section on the invoice page against `/invoices/{id}/payments`, `/cancel`, `/refunds` (refund gated on `payments.refund`); API 400/409 reasons shown; Refunded badge; quote detail shows number + status and the API's refusal reason, QuoteActionsMenu reports errors; integration tests assert every step (6ddd208, 8aac090, 74ec760). Field mismatches → T234.
- **T233** — JobMaterial.Billable default:true stores false as true. `JobMaterial.Billable` `default:true` removed; `AddMaterial` defaults omitted `billable` to true explicitly; regression tests (ced0464).
- **T234** — Invoice/quote forms use the API's fields: tax_rate_id, terms_and_conditions, default tax prefill. Invoice form shows the stored tax rate (API returns the rate object, so edits used to zero the tax) and keeps per-line taxable flags; `terms_and_conditions` read/sent; typed create/update request bodies; new invoices/quotes prefill the tenant's default tax rate (left to the server) and payment term (due date from `days_until_due`); QuotesPage shows refused list actions in an inline banner (3cf699c, 9df0441, 228fd35). Tax-rate picker + create terms field → T237.
- **T237** — Tax rates API + tax-rate picker; terms on invoice create; keep line discounts on edit. Tenant-scoped tax rates API (`GET/POST/PUT/DELETE /api/v1/tax-rates`, own + global rows; global read-only 403, in-use delete 409) with `tax_rates.read` (all roles) / `tax_rates.manage` (super_admin, admin) via migration `000014`; `CreateInvoiceRequest.terms_and_conditions` (follow-up PUT removed); direct `tax_rate` on update clears `tax_rate_id`; `TaxRatePicker` on invoice (sends `tax_rate_id`) and quote forms; invoice form round-trips per-line discounts; Cypress selectors refreshed; cross-tenant + Postgres grant tests (98d7df6, 115dcb0, 4826dbe, 7db9374, 8072299, d1fc6b6). Tax-rate management screen → T235.
- **T238** — Job assignment checks the technician belongs to the tenant. Every assignment path (create job with assignees, `POST /jobs/:id/assign`, `POST /assignments`, bulk) requires an active `tenant_users` membership in the job's tenant; unknown/inactive/other-tenant users get a typed 404 (`assignee_not_found` / `technician_not_found`); CreateJob checks before writing; repository backstop `IsActiveTenantMember`; cross-tenant tests (cef9489).
- **T239** — Access log drops query strings. Access log logs the path plus sorted query KEYS only (`middleware.QueryKeys`); same for the unused request loggers, `AccessLogger` and the panic handler's Sentry extra; test asserts `?token=secret123` never reaches a log line (9cc9f5e).
- **T245** — Repair the Postgres repository tests so CI go-test is green. Repository tests run in a seeded tenant against the migrated DB: shared fixture (`openTestDB` with the tenancy plugin, `createTestTenant`, `addMember`, cleanup), valid fixtures, exact counts, new cross-tenant and non-member assignment tests (18 pass); customer bench seeds its own tenant; CI migrates TEST_DATABASE_URL first and runs the Postgres race tests (4e7e3c8, ae193e0).
