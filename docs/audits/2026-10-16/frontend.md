# ServicePro Frontend Audit

**Date:** 2026-10-16. **Scope:** `frontend/`, read-only. **Repo state:** `master` @ 3df4280, clean.
**Toolchain used locally:** Node v26.10.0 (CI uses Node 20 and the Dockerfile uses Node 22. Neither is pinned: there is no `.nvmrc` and no `engines` field).

Paths are relative to `frontend/` unless noted. Severity scale: **Critical / High / Medium / Low**. Every finding below was checked against the code. Anything inferred rather than reproduced is marked *(inferred)*.

---

## 0. Executive summary

| Area | Status |
|---|---|
| `tsc --noEmit` | PASS (0 errors) |
| `npm run lint` | **FAIL**: 0 errors, 8 warnings, and the script runs with `--max-warnings 0` |
| `vitest run` | **FAIL**: 16/64 files and 87/1347 tests fail, with 142 unhandled errors and 2 skipped. `retry: 1` is configured |
| `vite build` | PASS. Initial JS ≈ 760 KB raw / ≈ 245 KB gzip modulepreloaded on every page |
| `npm audit --omit=dev` | 19 vulns: 1 critical (protobufjs, via posthog-js → OpenTelemetry), 2 high (axios <1.19 direct dependency, form-data via axios), 16 moderate. `npm audit fix` resolves them |
| Token storage | Good: access and refresh tokens are httpOnly cookies, and no JWT is in JS storage. **But** the whole React Query cache (customer, quote and role PII) is persisted to `localStorage` and never cleared on logout |
| Raw HTML | No `dangerouslySetInnerHTML`, `innerHTML` or `srcdoc` anywhere |
| Same-origin API | Yes. axios `baseURL: '/api'` is hard-coded, so it is Cloudflare-friendly. `VITE_API_URL` is dead config |
| Target-convention gap | Large: Tailwind v3 with 2,800+ hard palette classes and no dark mode, React 18, server state mostly in `useEffect` plus Zustand, and roughly 18k LOC of unreachable components |

**Top 5 to fix before an invite-only preview**
1. **High**: a wrong password on `/login` triggers the 401 → refresh → `window.location.href='/login'` hard reload, so the error is never shown (`src/services/api.ts:56-77`).
2. **High**: the Stripe invoice-payment success and cancel URLs point at `/invoices/payment-success|cancelled`, which sit behind `ProtectedRoute`. End customers who pay through the public `/pay/:token` link get bounced to `/login` (`src/routes/index.tsx:444-452`; `backend/internal/api/routes/services.go:129-130`).
3. **High**: the persisted React Query cache plus non-tenant-scoped query keys plus no clear on logout or tenant switch leaks data across tenants and users on a shared browser (`src/providers/QueryProvider.tsx:85-140`, `src/hooks/useRoles.ts:14-22`).
4. **High**: job times are entered as "UTC wall-clock" but the calendar renders browser-local time. The same job shows at 09:00 on the form and 05:00 on the calendar for an EDT user (`src/utils/epoch.ts:5-26`, `src/pages/Jobs/JobCalendarPage.tsx:34`).
5. **Medium**: password-reset, email-verify and invitation tokens are sent to PostHog in `$search` on every page view (`src/App.tsx:9-17`).

---

## 1. Build / typecheck / lint / test

### 1.1 Lint fails the 0-warning gate (Low)
`npm run lint` exits 1 because of 8 warnings, all trivial:
- `src/components/shared/DropdownMenu/__tests__/DropdownMenu.test.tsx:1` unused `afterEach`
- `src/hooks/__tests__/useCache.test.ts:2` unused `waitFor`
- `src/pages/Invoices/InvoiceDetailPage.tsx:107` `react-hooks/exhaustive-deps` (missing `loadInvoice`). This one is real: see the race note in 3.7.
- `src/services/__tests__/api.test.ts:1` unused `afterEach`
- `tests/integration/auth.integration.test.tsx:63` unused `mockUser`
- `tests/integration/customer.integration.test.tsx:13` unused `within`
- `tests/integration/invoice.integration.test.tsx:883` unused `user`
- `tests/integration/setup.ts:10` unused `Route`

**Fix:** delete the unused imports and wrap `loadInvoice` in `useCallback`, or move it to `useQuery`.

### 1.2 Test suite is red (High: CI signal is meaningless)
`npx vitest run`: **16 failed files, 87 failed tests, 142 errors, 2 skipped.** The failures fall into five groups:

| Cause | Files | Verdict |
|---|---|---|
| `TypeError: document.elementFromPoint is not a function` (142×). react-big-calendar drag and drop in jsdom | `tests/components/calendar/Calendar.test.tsx` (11 fail), `Calendar.integration.test.tsx` (14 fail), `CalendarEvent.test.tsx` (1) | Test rot. Add `document.elementFromPoint = () => null` to `src/test/setup.ts`, or mock the DnD addon |
| `Failed to resolve import ../../../src/components/subscriptions/*`. The components were deleted but the tests remain | `tests/components/subscriptions/{BillingSchedule,CancellationFlow,PlanSelector,SubscriptionManager}.test.tsx` | Dead tests. Delete them, along with `src/types/subscription.ts`, which is also unused |
| `TypeError: checkAuth is not a function`. The store mock predates `checkAuth` | `src/components/layout/__tests__/ProtectedRoute.test.tsx` (3), and parts of the integration suites | Test rot. Update the mock |
| `localStorage` undefined / `Cannot read properties of undefined (reading 'length'/'setItem')` | `src/store/__tests__/tenantStore.test.ts` (25), `sidebarStore.test.ts` (10), `src/utils/__tests__/cache.test.ts` (file fails to load) | **Environment-specific.** Node ≥25 ships an experimental global `localStorage` that shadows jsdom's ("localStorage is not available because --localstorage-file was not provided"). These probably pass on CI's Node 20. **Fix:** pin Node (`.nvmrc` = 22, `"engines"`) and/or stub `localStorage` explicitly in `src/test/setup.ts` |
| Text and role queries no longer match the UI (`Found multiple elements…`, `Unable to find … Install HVAC System`, `$1,080`, `/first name/i`, role `application`) | `tests/integration/{auth,customer,job,quote,invoice}.integration.test.tsx` (3/4/5/3/8 fail) | Test rot after UI changes. CLAUDE.md requires these to be maintained |

Other items:
- **Flakiness masking:** `vite.config.ts:166` sets `retry: 1`, so every failing test runs twice (most of the 22 s runtime) and genuinely flaky tests are hidden. Remove it, or set it only in CI with reporting.
- **Skipped tests:** `src/components/roles/__tests__/RoleForm.test.tsx:142` (`validates hierarchy level range`) and `src/pages/Login/__tests__/LoginPage.test.tsx:99` (`shows error for invalid email format`).
- `.github/workflows/deploy-release.yml:59` runs `npm test -- --run --passWithNoTests`, so on the release path a red suite blocks deploys only if it actually runs. Check whether `checks.yml` gates on frontend tests at all: it runs `npm ci`, then pre-commit.
- `vite.config.ts:142-145` uses the deprecated `test.deps.inline`, and `poolOptions` is deprecated in Vitest 4.

### 1.3 Build (OK, with notes)
- The build passes. **Initial-load JS** (modulepreloaded from `index.html` on *every* route, including the marketing landing page): `vendor` 340 KB, `vendor-react` 142 KB, `vendor-forms` 77 KB, `vendor-ui` 51 KB, `vendor-data` 40 KB, `page-auth` 57 KB, `page-settings` 66 KB and `vendor-stripe` 11 KB. That is about 785 KB raw / about 245 KB gzip.
- `vendor` (340 KB) is mostly **posthog-js** plus **@sentry/react** (with Replay), imported eagerly in `src/main.tsx:5-18`. Lazy-load both, or at least the Replay integration.
- `page-settings` is preloaded for anonymous visitors because Settings pages are imported via the shared `@pages/Settings` barrel, and `preloadEagerRoutes()` (`src/routes/index.tsx:355`) eagerly preloads dashboard and settings.
- `vendor-charts` is 163 KB. It is only used by `pages/Reports/{Revenue,Customer}ReportPage.tsx`, where it is correctly lazy.
- `vite-plugin-compression` writes `.gz` and `.br` copies of every asset. These are useless on Cloudflare, which compresses at the edge, and inflate the asset count.
- `@sentry/react` is in **devDependencies** (`package.json:52`) but imported at runtime. It works only because every install is a full install.

---

## 2. Security

### 2.1 Auth token storage (OK)
- `src/services/api.ts:20-27`: `withCredentials: true`, with tokens in httpOnly cookies. No token in `localStorage`. `VITE_TOKEN_KEY` and `VITE_REFRESH_TOKEN_KEY` in `.env.example` are dead config and should be removed to avoid confusion.
- `src/store/authStore.ts:156-162` persists only `{isAuthenticated, user}` (`auth-storage`). The user object includes email and profile; that is acceptable but XSS-readable.

### 2.2 React Query cache persisted to localStorage, never cleared (High)
- `src/providers/QueryProvider.tsx:85-140` uses `persistQueryClient` with a sync `localStorage` persister (`sp_`-prefixed via `utils/cache.ts`). Every successful query is persisted for 1 h (`maxAge: CACHE_TIMES.EXTENDED`): roles, permissions, audit logs, AR aging, quotes and templates.
- **Query keys are not tenant-scoped** (`['quotes', filters]`, `roleKeys.all…`, `['templates', …]`).
- `logout()` (`src/store/authStore.ts:70-87`) does **not** clear the query cache, `tenant-storage` or `current_tenant_id`. Only the refresh-failure path in `api.ts:71-73` clears the tenant keys, and even it leaves the query cache.
- **Scenario:** user A logs out on a shared or office PC and user B logs in. B sees A's restored cached data until it goes stale (staleTime 5 min, `config/cache-config.ts:16`). Likewise, after `TenantSwitcher` → `window.location.reload()` (`components/tenants/TenantSwitcher.tsx:42-44`), the previous tenant's roles and AR data are restored from cache and treated as fresh.
- **Fix:** drop persistence entirely, since there is no real offline use case. Otherwise include `tenantId` in every key, and on logout or tenant switch call `queryClient.clear()`, `persister.removeClient()`, `useTenantStore.getState().reset()` and `localStorage.removeItem('current_tenant_id')`.

### 2.3 Stale tenant header after logout (Medium)
- `src/services/api.ts:33-45` sends `X-Tenant-ID` from `localStorage.current_tenant_id`, which survives logout (see 2.2). `loadTenants()` (`src/store/tenantStore.ts:31-42`) keeps the persisted `currentTenant` if it is set and never re-validates it against the new user's list.
- **Scenario:** user B logs in on the same browser and gets A's tenant header. If the backend validates membership, B sees 403s and a wrong org name. If it does not, it is a cross-tenant read. The backend `RequireTenant` (`backend/internal/api/middleware/tenant.go:73-110`) trusts the header first.
- **Related bug:** on a fresh browser, `loadTenants` picks `tenants[0]` with `set({currentTenant})` but does **not** write `current_tenant_id` (`tenantStore.ts:42`, unlike `setCurrentTenant` at :53). The UI shows tenant[0] while requests carry no header, so the backend falls back to the JWT or "default" tenant, which may differ for a multi-org user.
- **Fix:** derive the header from the store, not a separate key. Validate the persisted tenant against the freshly loaded list. Reset on logout.

### 2.4 Secrets in auth tokens sent to analytics (Medium)
- `src/App.tsx:9-17` `PageViewTracker` sends `{path, search: location.search}` to PostHog on every navigation. `/reset-password?token=…` (`pages/ResetPassword/ResetPasswordPage.tsx:40`), `/verify-email?token=…` and `/invitations/accept?token=…` therefore ship single-use secrets to a third party. Sentry navigation breadcrumbs likely capture the full URL too *(inferred: `sanitizeData` in `services/errorTracking/sentry.ts:204` redacts by key name, not URL query)*.
- **Fix:** strip `search`, or allow-list params, before `trackEvent`, and add a Sentry `beforeBreadcrumb` that drops `token=`.

### 2.5 HTML injection / open redirect (OK)
- There is no `dangerouslySetInnerHTML`, `innerHTML`, `srcDoc` or `iframe` in `src/`. Template previews (`components/templates/TemplatePreview.tsx`) render as React text, and that component is unreachable anyway (see 4.2).
- There is no open redirect. The only `window.location.href` assignment is the constant `/login` (`api.ts:74`). `receiptUrl` in `components/payments/PaymentResult.tsx:222` comes from Stripe, and the component is unused.
- Minor: `authApi.verifyEmail` builds `?token=${data.token}` without `encodeURIComponent` (`api.ts:111`). Use `params: {token}`.

### 2.6 Hard-coded secrets (OK)
- No secrets in tracked source. `frontend/.env` is gitignored (`.gitignore:47`) and contains a `pk_test_` Stripe publishable key, which is public by design. All `VITE_*` values are client-exposed by definition. Nothing secret-shaped is in `.env.example`.

### 2.7 CSP incompatible with what the app loads (Medium for nginx, and a porting concern)
The `nginx.conf:27` and `security-headers.conf:9` policy is `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; connect-src 'self'; frame-ancestors 'self'`. It blocks:
- **Google Fonts** (`index.html:8-12`): the stylesheet from `fonts.googleapis.com` and the fonts from `fonts.gstatic.com`. Inter silently falls back to the system font. Self-host Inter, for example with `@fontsource-variable/inter`.
- **Stripe.js** (`providers/StripeProvider.tsx:47`, `loadStripe` → `https://js.stripe.com`) is blocked by `script-src` and `frame-src`. It wraps the whole app, so it loads on every page.
- **PostHog** (`us.i.posthog.com`, plus asset host) and **Sentry** ingest are blocked by `connect-src 'self'`. Errors and analytics therefore never arrive under the production nginx image *(inferred: not run in a browser)*.
- `Permissions-Policy: payment=()` disables the Payment Request API (Apple Pay / Google Pay in Stripe Elements).
- `X-XSS-Protection` is obsolete and can be dropped.
- **Fix:** write the CSP for the actual third parties, or stop loading them on the marketing pages, and port it to `_headers` (section 5).

### 2.8 Dependency vulnerabilities (Medium)
`npm audit --omit=dev`: **protobufjs ≤7.6.4 (critical, multiple RCE/DoS advisories)**, pulled in via `posthog-js` → `@opentelemetry/otlp-transformer`. **axios 1.13.4 (high, the range 1.0.0-1.19.0 is affected)** is a direct dependency, so bump to ≥1.20. form-data 4.0.5 (high, CRLF injection) comes via axios. There are 16 moderate. `npm audit fix` and an axios bump resolve them without majors.

---

## 3. Bugs

### 3.1 Login error swallowed by a hard reload (High)
`src/services/api.ts:56-77`: any 401 not yet retried triggers `axios.post('/api/v1/auth/refresh')`. On failure it clears storage and sets `window.location.href = '/login'`. The login endpoint itself returns **401 for bad credentials** (`backend/internal/api/handlers/auth_handler.go:68,126,140`). So a mistyped password → refresh (no cookie → 401) → full page reload → `LoginPage`'s `setError(...)` (`pages/Login/LoginPage.tsx:44-49`) never renders. The user just sees the form blank out.
- **Fix:** skip refresh for `/v1/auth/*` URLs, and navigate via the router (or an auth-store event) instead of a hard reload.
- **Also:** there is no single-flight refresh. N concurrent 401s (the Dashboard fires 6 parallel requests, `pages/Dashboard/DashboardPage.tsx:83-105`) cause N refresh calls. This is harmless today because the backend does not rotate refresh tokens, but it breaks as soon as rotation (or Supabase) arrives. Use a shared `refreshPromise`.

### 3.2 Customer payment return pages are behind auth (High)
`src/routes/index.tsx:444-452` puts `/invoices/payment-success` and `/invoices/payment-cancelled` inside `<ProtectedRoute>`. The backend sets the Checkout success and cancel URLs to exactly those paths (`backend/internal/api/routes/services.go:129-130`), and the checkout is started from the **public** `GET /pay/:token` route (`backend/internal/api/routes/v1/invoices_routes.go:96`). End customers are not app users, so after paying they are redirected to `/login`.
- **Fix:** move these to public routes, for example `/pay/success` and `/pay/cancelled`, with no tenant chrome. Keep the deep-link on the protected side for staff.

### 3.3 Scheduling timezone inconsistency (High)
- `src/utils/epoch.ts:5-26`: `datetimeLocalToEpoch` treats the `<input type="datetime-local">` value **as UTC** (`value + ':00Z'`), and `epochToDatetimeLocal` renders back in UTC. `JobDetailPage.tsx:184,256,289,318,563` use these, and the min/max bounds use `toISOString()` (UTC) at `:91-102`.
- `src/pages/Jobs/JobCalendarPage.tsx:34-38` uses `epochToDate()`, and react-big-calendar with `dateFnsLocalizer` (`components/calendar/Calendar.tsx:50`) renders in **browser-local** time.
- **Scenario:** an EDT dispatcher creates a job for "09:00" and stores 09:00Z. The job detail shows 09:00, the calendar shows it at 05:00, and any backend logic (reminders, conflict detection, the new tenant business hours in `America/New_York`) sees 09:00Z = 05:00 local.
- **Fix:** pick one model. The usual one: inputs are in the **tenant timezone**, converting with `date-fns-tz` / `@date-fns/tz` using `currentTenant.settings.timezone`, and storage is true UTC epoch. Render the calendar in the tenant timezone as well.
- **Related (Low):** default invoice and quote dates use `new Date().toISOString().split('T')[0]` (`pages/Invoices/InvoiceDetailPage.tsx:103-104`, `pages/Quotes/QuoteDetailPage.tsx:82,147`). After roughly 7-8 pm in US timezones the default issue date is *tomorrow*. Due and valid-until dates are sent as `T23:59:59Z` (`InvoiceDetailPage.tsx:265-268`, `QuoteDetailPage.tsx:232`), so "overdue" flips at end of day UTC, not tenant-local.

### 3.4 Money formatting and rounding (Medium)
- **Float artefact in the tax-rate input:** `(invoice.tax_rate * 100).toString()` (`InvoiceDetailPage.tsx:127-129`, `QuoteDetailPage.tsx:105-107`). Verified: `0.07*100` gives `"7.000000000000001"` and `0.029*100` gives `"2.9000000000000004"`, which then appear in the edit form. Use `Number((rate*100).toFixed(4)).toString()`.
- **Client totals use unrounded float math** (`parseFloat(qty) * parseFloat(unit_price)`, with tax applied to the subtotal) at `InvoiceDetailPage.tsx:236-252,682-685` and `QuoteDetailPage.tsx:203-218,622-625`. There is no per-line rounding to cents, so the preview can differ by a cent from the server total. Compute in integer cents, or display the server's totals after save.
- **Ten separate `formatCurrency` implementations:** `types/invoice.ts:97` (USD, dollars), `types/payment.ts:295` (**divides by 100**, cents), `types/chart.ts:372`, `types/membership.ts:130` (cents), `types/kpi.ts:184`, `components/quotes/LineItemRow.tsx:151`, `components/quotes/TotalCalculator.tsx:48`, `components/quotes/utils/pdfGenerator.tsx:153`, `pages/Reports/{CustomerReport,ARAging,RevenueReport}Page.tsx`. Mixing cents and dollars helpers with the same name is a latent 100× bug. USD and `en-US` are hard-coded everywhere, and there is no tenant currency. Consolidate into `utils/money.ts` with an explicit `formatCents` / `formatDollars`.

### 3.5 Lists swallow errors and show "empty" (Medium)
For example, `pages/Customers/CustomersPage.tsx:34-48`: errors go to `console.error` only, so an API failure renders the **"No customers yet" empty state** (`:143`). The same pattern appears across the `useEffect`-based pages (26 page files use `useEffect` fetching).
- `handleSearch` (`:50-54`) calls `setFilters(...)`, which re-triggers the effect, **and** `loadCustomers()` with the old page. That means double requests with no abort or ordering guard, so out-of-order responses can show stale results.
- **Fix:** move to `useQuery` (`placeholderData: keepPreviousData`, `isError` UI). This is the TanStack migration in section 6.

### 3.6 Customer pickers capped at 100 (Medium)
`InvoiceDetailPage.tsx:112`, `QuoteDetailPage.tsx:90` and `JobDetailPage.tsx:162` use `customerService.getCustomers({ page_size: 100 })`. A tenant with more than 100 customers cannot select the 101st customer when creating an invoice, quote or job. Replace with an async search combobox.

### 3.7 Missing aborts / effect deps (Low)
`InvoiceDetailPage.tsx:93-107` loads by `id` without cancellation (the lint warning). Navigating quickly between invoices can render the previous invoice's data. React Query would fix this.

### 3.8 Deep links lost on login (Low)
`components/layout/ProtectedRoute.tsx:20-22` redirects to `/login` without `state.from`. `LoginPage.tsx:43` always goes to `/dashboard`, and `getPostLoginRedirect()` (`routes/index.tsx:547`) exists but is unused. An emailed `/invoices/:id` link loses its target after login.

### 3.9 Missing static assets (Low, but visible)
`public/` contains only `.gitkeep`. `index.html:5` references `/vite.svg`, and the PWA manifest references `/icon-192.png` and `/icon-512.png` (`vite.config.ts:35-45`). All 404, or with SPA fallback they return HTML, so there is no favicon and the manifest is broken.

---

## 4. Cleanup

### 4.1 Artifacts in git / working tree (Low)
- **Tracked:** `bundle-report.html` (2.7 MB, committed in 948583d), `SETUP_SUMMARY.md`, and `TEMPLATE_SYSTEM.md` (describes an unreachable feature). The component-level docs (`components/quotes/{README,QUOTELIST,QUOTEDETAIL}.md`, `components/roles/IMPLEMENTATION_SUMMARY.md`, and others) are AI-generated scaffolding notes. Remove them, or move the useful parts to `docs/`.
- **Untracked but present:** `frontend/~/.npm/_logs/…debug-0.log` and `_update-notifier-last-checked`. This came from an npm run with `HOME`/cache set to a literal `~` (for example `npm_config_cache=~/.npm` inside quotes in a Docker or script context). It is **ignored, not tracked** (`git status --ignored` shows `!! ~/`). Delete it and add `/frontend/~` to `.gitignore`. Also present and correctly ignored: `coverage/`, `dist/`, `.env`.
- `.dead-code-baseline.json` and the `ts-prune` scripts maintain a baseline of known-dead exports. Prefer deleting the code (4.2).

### 4.2 Unreachable code: about 18.5k LOC (Medium for maintenance and test noise)
Nothing reachable from `routes/index.tsx` imports the following. Verified by grep for importers outside tests and stories:
- `components/templates/*`: the entire template system, including TemplateList, Editor, Preview, CategoryManager and ImportExport. **There is no route to it.**
- `components/payments/*` (PaymentForm, PaymentSummary, PaymentResult, CreditCardInput, BillingAddress, methods/*), which only have stories and tests. Invoice payment uses Stripe Checkout.
- `components/charts/*`, `components/kpi/*`, `store/kpiStore.tsx`, `components/skeletons/*`, `components/filters/*` (only used by the dead TemplateList), and `components/OptimizedImage.tsx`.
- `components/quotes/{QuoteDetail,QuoteList,QuoteStatusActions,QuoteStatusTimeline,LineItem*,TotalCalculator}`, `components/quotes/hooks/*` and `components/quotes/utils/pdfGenerator.tsx`. Only `QuoteActionsMenu` is used. `@react-pdf/renderer` is a dependency only for this dead path, and it is tree-shaken out of the build.
- `hooks/useWebSocket.ts`, `services/realtime/*` (exported from `services/index.ts:46` but never mounted) and `hooks/useCache.ts`.
- `utils/registerServiceWorker.tsx` and `src/service-worker.ts`: the SW is built (`injectManifest`) but never registered (`injectRegister: false`, and the register util is not imported). Workbox (6 runtime deps) and `vite-plugin-pwa` are therefore dead weight.
- `types/subscription.ts`, plus the orphan tests in `tests/components/subscriptions/`.
- Almost all of the app's React Query usage lives in this dead code. The live pages use `useEffect` instead.

**Fix:** delete it, or move it to a `wip/` branch. That cuts a large share of the failing tests and the misleading docs.

### 4.3 Duplicate utilities (Low)
- 10× `formatCurrency` (3.4). `Skeleton` is defined in both `components/Loadable.tsx:296` and `components/skeletons/Skeleton.tsx`.
- Two `DateRangePicker`s (`components/filters/` and `components/reports/`).
- `theme/colors.ts` duplicates `tailwind.config.js`.
- Two loading/preload systems (`config/splitting.ts`, `config/loading.ts` and `components/Loadable.tsx` with many unused exports).
- `webpackChunkName` magic comments in `routes/index.tsx` are no-ops under Vite.

### 4.4 `any` usage (OK)
Only 3 occurrences outside generated code: `config/splitting.ts:17`, `components/Loadable.tsx:188` (`ComponentType<any>` for lazy modules, which is acceptable) and `components/calendar/Calendar.tsx:61` (a `BigCalendar as React.ComponentType<any>` cast, so use the typed `withDragAndDrop<Event>`). There is none in tests.

### 4.5 Outdated majors (from `npm outdated`)
| Package | Current | Latest | Note |
|---|---|---|---|
| react / react-dom / @types | 18.3.1 | 19.3.1 | see 6 |
| vite | 5.4.21 | 8.3.4 | plus `@vitejs/plugin-react` 4.7 → 6.1 |
| tailwindcss | 3.4.19 | 4.3.3 | see 6 |
| react-router-dom | 6.30 | 7.18 | v7 future-flag warnings already print in tests |
| @headlessui/react | 1.7.19 | 2.2.10 | v1 does not support React 19. Only used in `shared/Modal/Modal.tsx` |
| eslint | 8.57 (EOL) | 10.12 | flat config already exists. `eslint-plugin-react-hooks` 4 → 7 |
| zod / @hookform/resolvers | 3.25 / 3.10 | 4.6 / 5.9 | |
| date-fns | 3.6 | 4.4 | v4 adds first-class tz (`@date-fns/tz`), which is useful for 3.3 |
| lucide-react | 0.323 | 1.58 | very old. Check icon renames |
| @stripe/stripe-js / react-stripe-js | 8.7 / 5.6 | 10.1 / 7.0 | |
| @sentry/react | 10.38 | 11.8 | move to `dependencies` |
| typescript | 5.9.3 | 7.0.2 | stay on 5.9 until the tooling catches up |
| vitest / coverage-v8 | 4.0 | 5.0 | |
| vite-plugin-pwa | 0.21 | 2.0 | delete rather than upgrade (4.2) |

---

## 5. Cloudflare static-assets deployment readiness

**Good news:** the SPA already calls the API same-origin (`api.ts:21` `baseURL: '/api'`, refresh at `/api/v1/auth/refresh`), cookies are first-party, and the dev proxy mirrors this (`vite.config.ts:120-127`). `VITE_API_URL` (Dockerfile ARG, `.env.example:13`) is read nowhere, so delete it.

**What to change:**
1. **Worker with static assets** (`wrangler.jsonc`): `assets: { directory: "./dist", not_found_handling: "single-page-application", run_worker_first: ["/api/*"] }`. The Worker `fetch` proxies `/api/*` to the Go API origin, replacing `nginx.conf:34-46` `location /api { proxy_pass ${BACKEND_URL} … }`, and forwards `X-Forwarded-*`. WebSocket upgrade passthrough (nginx sets `Upgrade` and `Connection`) only matters if realtime is revived. Alternatively, a Cloudflare route on the same zone can send `/api/*` straight to the origin and skip the Worker.
2. **SPA fallback:** `nginx.conf:49-51` `try_files … /index.html` becomes `not_found_handling: "single-page-application"`.
3. **Headers:** `nginx.conf:16-30` plus `security-headers.conf` move to `public/_headers`, with `/*` getting the security headers and the corrected CSP (2.7), and `/assets/*` getting `Cache-Control: public, max-age=31536000, immutable`. Drop the nginx `.html` no-store rule: Cloudflare already serves HTML revalidated. Note that `_headers` applies only to asset responses. `/api/*` responses from the Worker need headers set in code.
4. **Remove** `vite-plugin-compression` (it would upload `.gz` and `.br` duplicates), the `/health` nginx route (use the API's), and the Dockerfile, `nginx.conf` and `security-headers.conf` once migrated.
5. **WebSocket URL:** `hooks/useWebSocket.ts:50` defaults to `ws://localhost:8080/api/v1/ws`. If realtime is kept, derive it from `location` (`wss://${location.host}/api/v1/ws`).
6. **Env at build time:** `VITE_STRIPE_PUBLISHABLE_KEY`, `VITE_SENTRY_DSN`, `VITE_POSTHOG_*` and `VITE_APP_VERSION` must be set in the CF build (Workers Builds env vars), since Vite inlines them. `VITE_APP_VERSION` is the React Query persist buster, so it is moot if persistence is removed.
7. **Backend coupling:** the backend `FRONTEND_URL` (Stripe success URLs, email links) and CORS need to point at the CF hostname. Cookie `Domain`/`SameSite` must suit same-origin, which it already does.
8. **Assets:** add a real favicon and icons (3.9).

**Effort:** about 0.5-1 day, plus about 0.5 day for CSP tuning.

---

## 6. Gaps vs target conventions (League Looper) with rough effort

| Gap | Current | Target | Effort |
|---|---|---|---|
| **Tailwind v3 → v4** | `tailwind.config.js` + PostCSS + `@tailwind` directives (`src/index.css:1-3`), `@apply` in base styles, `styles/marketing/*.css` | `@import "tailwindcss"`, `@theme inline`, `@tailwindcss/vite` | **0.5-1 d** with `npx @tailwindcss/upgrade`, plus fixing renamed utilities (`shadow-sm`→`shadow-xs`, `ring` width, default border colour now `currentColor`, which matters for the `* { @apply border-neutral-200 }` base rule) |
| **Semantic CSS-variable tokens + light/dark from one file** | **No dark mode at all** (0 `dark:` classes). About 2,820 hard palette classes: `neutral-*` 1,036, `gray-*` 834 (two greys mixed), `blue-*` 352 (off-brand vs orange `primary`), `primary-*` 349 | `--color-surface`, `--color-fg`, `--color-muted`, `--color-border`, `--color-accent`, … in `@theme inline`, switched with `prefers-color-scheme` / `[data-theme]` | **4-6 d.** Define the about 15 tokens (0.5 d), then a scripted codemod for the mechanical mappings (`bg-white`→`bg-surface`, `text-neutral-900`→`text-fg`, `border-neutral-200`→`border-border`) followed by manual review of about 120 component files (3-4 d). Status colours (badges and charts) need a decision. Then dark QA (1 d). `theme/colors.ts` and the `tailwind.config.js` palette are deleted |
| **React 18 → 19** | 18.3.1 | 19.x | **1-1.5 d.** Blockers: `@headlessui/react` v1 (one file, `shared/Modal/Modal.tsx`; upgrade to v2 or replace with native `<dialog>`), and the react-big-calendar 1.20 peer range. Codemod `forwardRef` / `defaultProps` usages. `@testing-library/react` 16 is fine |
| **Server state → TanStack Query v5** | RQ v5 is installed but used mainly in dead code. Live pages use `useEffect` + `useState`, with about 26 page files. Zustand holds server data in `tenantStore` (tenants and members), `billingStore` and `membershipStore`. `kpiStore` is dead | `useQuery`/`useMutation` with a tenant-scoped key factory, invalidation on mutation, Zustand only for UI state (sidebar, auth session) | **5-8 d.** Build `queries/` per resource with key factories `[tenantId, 'customers', filters]` (1 d), then convert the list pages (Customers, Jobs, Quotes, Invoices, Reports: 2-3 d), detail and edit pages with mutations (2-3 d), and the tenant, billing and membership stores (1 d). Drop the persister. Fixes 2.2, 3.5, 3.7 |
| **Vite 5 → 7/8** | 5.4 + plugin-react 4.7 | current | **0.5 d.** Remove `vite-plugin-pwa`, `vite-plugin-compression` and the custom `manualChunks` function (Rolldown in Vite 7+/8 has its own chunking; re-measure) |
| **React Router 6 → 7** | 6.30 | 7.x (library mode) | **0.5 d.** Enable the v7 future flags first. Imports move to `react-router` |
| **ESLint 8 → 9/10, react-hooks 7** | 8.57 (EOL) | current | **0.5 d.** Flat config already exists. react-hooks 7 adds compiler-style rules that will flag the `eslint-disable exhaustive-deps` sites |
| **Supabase Auth client (future)** | httpOnly cookie JWT via the Go API, with an axios refresh interceptor | `@supabase/supabase-js` session, Bearer to the Go API | **3-5 d frontend** (Login, Register, Forgot/Reset, Verify, AcceptInvitation pages, `authStore`, interceptor, `ProtectedRoute`), plus backend JWKS verification. Note that Supabase stores the session in `localStorage` by default, which is a regression vs the current httpOnly cookies, so accept it knowingly or use the SSR cookie helpers |
| **Cloudflare static assets + `/api/*` proxy** | nginx Docker image | Worker + assets | **0.5-1 d** (section 5) |
| **Node pinning** | none (CI 20, Docker 22, local 26) | `.nvmrc` + `engines` | **0.1 d** |

**Total to reach parity (excluding Supabase):** roughly **13-19 dev-days**. Doing the dead-code deletion (4.2, about 0.5 d) first shrinks every one of these.

---

## 7. UX quick wins for an invite-only preview (noted, not fixed)

1. **Contact form fakes success** (`pages/Contact/ContactPage.tsx:23-27`, a `setTimeout` followed by "success"). Messages from prospects are silently dropped. Wire it to an API or use a `mailto:`.
2. **Legal pages are placeholders** with TODO comments (`pages/Legal/PrivacyPolicyPage.tsx:44`, `TermsOfServicePage.tsx:44`) and show "Last updated: {today}" (`:40`), which changes daily. The About page has a placeholder company story (`pages/About/AboutPage.tsx:62`).
3. **No favicon or app icons** (3.9). The browser tab shows a generic icon.
4. **Errors render as empty states** (3.5): a backend hiccup looks like "you have no customers".
5. **Login appears to do nothing on a wrong password** (3.1).
6. **Paying customers are dumped on the staff login page** (3.2).
7. **Onboarding:** Dashboard only loads data when `currentTenant` is set (`pages/Dashboard/DashboardPage.tsx:72-79`). Check that a new user with zero orgs gets a clear "Create your organization" CTA rather than an empty dashboard. There is no guided first-run flow (add customer → create quote → send) or sample data.
8. **Mobile:** the sidebar has a working off-canvas mode (`components/layout/Sidebar.tsx:176-195`). The hamburger is `fixed top-4 left-4` and probably overlaps page titles on narrow screens *(inferred, not checked in a browser)*. `DataTable` scrolls horizontally (`shared/DataTable/DataTable.tsx:110`), so wide tables are usable but not card-ified on phones. The job and invoice detail forms are long single-column pages and need a phone pass.
9. **Customer pickers stop at 100** (3.6).
10. **Fonts:** Inter does not load under the production CSP (2.7), so the marketing pages render in the fallback font.
11. **No dark mode** (6), which the League Looper apps have.
12. The **"Last updated"** and **"$" / USD-only** formatting signal "template" to anyone outside the US.

---

## Appendix: commands run
```
npx tsc --noEmit                      → exit 0
npm run lint                          → exit 1 (8 warnings, --max-warnings 0)
npx vitest run                        → 16 files / 87 tests failed, 142 errors, 2 skipped (Node 26)
npx vite build --outDir <scratchpad>  → exit 0 (repo dist/ untouched)
npm audit --omit=dev                  → 19 vulns (1 critical, 2 high, 16 moderate)
npm outdated                          → see 4.5
```
