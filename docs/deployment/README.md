# Deployment

> **Live since v0.9.0 (2026-10-24)** at <https://servicepro.sharkfins.xyz>.
> Every `v*` tag deploys through `release-deploy.yml`.

| Piece          | Production                                                                                                                       |
| -------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| Web            | Cloudflare Worker `servicepro-web` (static assets + `/api/*` and `/version` proxy) on `servicepro.sharkfins.xyz`                 |
| API            | Google Cloud Run service `servicepro-api` in **us-east5** (image from `backend/Dockerfile`)                                      |
| Database       | Neon Postgres 18, AWS **us-east-2** (Free plan; Launch plan before the first paying tenant)                                      |
| Auth           | Neon's managed Better Auth — the API verifies its JWTs; the SPA still signs in with the API's own JWT cookies until T158 lands   |
| Redis          | Optional (Upstash or none; without it the API uses an in-memory, per-instance store)                                            |
| Email          | Resend (T159)                                                                                                                    |
| Object storage | Cloudflare R2 (target; MinIO locally)                                                                                            |
| Backups        | Nightly `pg_dump` to R2 bucket `servicepro-backups` — wired, not activated yet (T209)                                            |

## Where to look

- [RELEASING.md](../../RELEASING.md) — versioning, commit prefixes, cutting a release
- [docs/ops/cloud-run-setup.md](../ops/cloud-run-setup.md) — one-time GCP, Worker and GitHub setup
- [docs/ops/deploy.md](../ops/deploy.md) — what a release does, rollback
- [docs/ops/backups.md](../ops/backups.md) — database backups and restore

## CI/CD at a glance

| Workflow                   | Trigger                    | Does                                                                                                 |
| -------------------------- | -------------------------- | ---------------------------------------------------------------------------------------------------- |
| `ci.yml`                   | push to master, PRs        | lint, tests, govulncheck, OpenAPI drift, frontend build/audit, Docker build + Trivy. Deploys nothing |
| `release-deploy.yml`       | `v*` tag                   | guard → migrate → api → web → GitHub Release                                                         |
| `migrate.yml`              | manual                     | `version` / `up` against production                                                                  |
| `scheduled-jobs.yml`       | hourly                     | POST `/internal/jobs/{payment-reminders,quote-expiry}` to the Cloud Run URL (`API_URL`)              |
| `db-backup.yml`            | nightly                    | `pg_dump` to R2 (fails until the R2 bucket + secrets exist — T209)                                   |
| `security-scan.yml`        | weekly                     | govulncheck + npm audit                                                                              |
| `dependabot-automerge.yml` | after CI on Dependabot PRs | merge patch/minor, bump, tag, release                                                                |

## Local development

See [Getting Started](../getting-started/quick-start.md) or run `make help`.
