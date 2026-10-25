# Recurring Jobs

> **Status: not built.** ServicePro has no recurring-job feature today: no API
> endpoints, no UI and no scheduler that generates jobs from a pattern. It is
> planned as **recurring service agreements (T167)** — the core of the MVP for
> commercial cleaning and facility-maintenance companies (see
> [docs/product/mvp-plan-2026-10.md](../product/mvp-plan-2026-10.md) and
> TODO.md).

## What exists today

- **Schema only.** The baseline migration (`000001_initial_schema`) has a
  `recurring_schedules` table (recurrence type `daily`/`weekly`/`monthly`/
  `yearly`/`custom`, an optional RRULE string, interval, days of week, day of
  month, time window, assigned technicians, `job_template_id`) plus
  `schedules`, `schedule_exceptions` and `holidays`. Nothing creates rows in
  `recurring_schedules`; the repository methods for it
  (`backend/internal/repository/schedule_repository.go`) have no callers in the
  API. Note the table has no `tenant_id` — it must get one (or be replaced)
  before any code uses it.
- **One-off scheduling.** Jobs have a scheduled start/end and assigned
  technicians; `GET /api/v1/jobs/scheduled` lists them for the calendar and
  `POST /api/v1/conflicts/check` checks a technician/time slot for overlaps.
- The old recurrence engine (`backend/internal/services/recurring`) was never
  wired into the binary and has been removed, as was the old frontend
  `components/recurring/` (T018).

## Planned: service agreements (T167)

From TODO.md T167 (Phase 5, v1 gate):

- An **agreement** per customer (and site, T186): cadence (e.g. "nightly
  Mon–Fri", "quarterly PM"), line items/price, start/end, assigned tech, and
  an auto-invoice option.
- A scheduled job (`POST /internal/jobs/<name>`, T141) generates the next
  occurrences **idempotently** — unique per agreement + date — so a late or
  repeated run never double-books.
- Managed from the UI.
- Autopay on agreements is a separate task (T191).

Known pitfalls to design for (from the removed engine's audit): month-end
dates (Jan 31 + 1 month must not skip February), time of day in the tenant's
time zone with DST (not UTC), occurrence caps counted per series rather than
per generation window, and a missing template must not panic.

Until T167 ships, describe recurring work to users as "create the jobs
individually" — don't document recurrence options in help content.
