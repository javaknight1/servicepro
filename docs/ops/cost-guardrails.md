# Cost and connection guardrails (T213)

Hard ceilings so a bot, a retry loop or a good traffic day can't exhaust the
database's connections or grow the bill silently. Companion docs:
[`cloud-run-setup.md`](./cloud-run-setup.md), [`monitoring.md`](./monitoring.md).

| Guardrail                 | Value                                     | Where                                      | Kind         |
| ------------------------- | ----------------------------------------- | ------------------------------------------ | ------------ |
| Cloud Run max instances   | 4 (min 0)                                 | `scripts/deploy-cloudrun.sh`, every deploy | hard cap     |
| API DB pool per instance  | `DB_MAX_OPEN_CONNS=5`                     | `scripts/deploy-cloudrun.sh`, every deploy | hard cap     |
| Connection budget check   | instances × pool ≤ 40, else deploy fails  | `scripts/deploy-cloudrun.sh`               | release gate |
| GCP monthly budget        | $10, emails at 50/90/100% + 100% forecast | `scripts/setup-cost-guardrails.sh`         | alert only   |
| Artifact Registry cleanup | keep 20 newest images, delete > 90 days   | `scripts/setup-cost-guardrails.sh`         | storage cap  |

Live values checked 2026-11-04: `maxScale=4`, `DB_MAX_OPEN_CONNS=5`, no
min-instances.

## Connection math

Neon's `max_connections` depends on compute size
([Neon docs](https://neon.com/docs/connect/connection-pooling),
[formula](https://neon.com/docs/reference/compatibility)): 0.25 CU → **104**,
of which 7 are reserved for Neon's superuser → **97 usable**. With autoscaling
it is computed from `min(max CU, 8 × min CU)`, so Free's default 0.25–2 CU
range gives 839. We size against the 0.25 CU floor, so a change to the compute
settings can't break us.

`DATABASE_URL` (the API) uses the **pooled** host (`-pooler`, PgBouncer in
transaction mode, up to 10,000 client connections; per user+database pool =
0.9 × `max_connections` = **93** at 0.25 CU). `MIGRATION_DATABASE_URL`
(migrate job, nightly `pg_dump`) uses the **direct** host and counts against
`max_connections` directly.

| Consumer                                       | Connections (worst case) |
| ---------------------------------------------- | ------------------------ |
| API: 4 instances × `DB_MAX_OPEN_CONNS=5`       | 20 (pooled)              |
| Release `migrate` job (golang-migrate)         | 1–2 (direct)             |
| Nightly backup (`pg_dump`, `psql` probe)       | 1–2 (direct)             |
| Neon Auth (managed Better Auth), console, psql | ~5–10                    |
| **Total**                                      | **≈ 35 of 97** usable    |

The deploy script refuses a deploy where instances × pool > **40**
(`DB_CONNECTION_BUDGET`), which keeps the API under half the 0.25 CU limit and
under the 93-connection pooled-user pool. Note the API's built-in non-dev
default pool is 10, so an instance without `DB_MAX_OPEN_CONNS` would use
4 × 10 = 40 — still inside the budget; the deploy pins 5 anyway.

Per instance, `--concurrency=40` requests share 5 connections; requests wait
for a free connection rather than opening more. Watch
`db_connections_*` on `/metrics` and Neon → Monitoring → Connections.

## Raising the limits — instances and pool together

1. Check the real ceiling: `SHOW max_connections;` (Neon SQL editor) and the
   compute's autoscaling range.
2. Choose instances × pool that leaves at least ~50% of usable connections for
   migrate, backup, Neon Auth and people.
3. Change `MAX_INSTANCES` / `DB_MAX_OPEN_CONNS` defaults (and
   `DB_CONNECTION_BUDGET` if the compute grew) in `scripts/deploy-cloudrun.sh`
   in one commit, update the tables above, and release.

Moving to Neon's Launch plan (D14) doesn't change the math unless the minimum
compute size grows.

## Budget alert + Artifact Registry cleanup (one-time)

Needs a Billing Account Administrator (budgets) and Artifact Registry admin
(Owner works):

```bash
gcloud auth login
ALERT_EMAIL=<your alert address> scripts/setup-cost-guardrails.sh   # BUDGET_AMOUNT=10 default
```

Expected GCP spend today is a few dollars a month (Cloud Run inside the free
tier, Secret Manager versions, Artifact Registry storage), so $10 is an early
warning, not a forecast. A budget never stops spend — the Cloud Run
max-instances cap does. Change the amount later with
`gcloud billing budgets update <budget-id> --budget-amount=<N>USD`. Re-running
the script is safe: an existing budget is left alone and the cleanup policy is
re-applied.
