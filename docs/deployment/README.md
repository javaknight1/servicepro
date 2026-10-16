# Deployment

> **Not hosted yet — local only.** The hosting target is decided and the
> pipeline is wired, but inert until the one-time setup is done.

| Piece          | Target                                                                                            |
| -------------- | ------------------------------------------------------------------------------------------------- |
| API            | Google Cloud Run service `servicepro-api` (image from `backend/Dockerfile`)                       |
| Web            | Cloudflare Worker `servicepro-web` (static assets + `/api/*` proxy) on `servicepro.sharkfins.xyz` |
| Database       | Postgres — host undecided (Supabase likely)                                                       |
| Redis          | Optional; Upstash later                                                                           |
| Object storage | Cloudflare R2 later (MinIO locally)                                                               |
| Backups        | Nightly `pg_dump` to R2 bucket `servicepro-backups`                                               |

## Where to look

- [RELEASING.md](../../RELEASING.md) — versioning, commit prefixes, cutting a release
- [docs/ops/cloud-run-setup.md](../ops/cloud-run-setup.md) — one-time GCP, Worker and GitHub setup
- [docs/ops/deploy.md](../ops/deploy.md) — what a release does, rollback
- [docs/ops/backups.md](../ops/backups.md) — database backups and restore

## CI/CD at a glance

| Workflow                   | Trigger                    | Does                                                                                                 |
| -------------------------- | -------------------------- | ---------------------------------------------------------------------------------------------------- |
| `ci.yml`                   | push to master, PRs        | lint, tests, govulncheck, OpenAPI drift, frontend build/audit, Docker build + Trivy. Deploys nothing |
| `release-deploy.yml`       | `v*` tag                   | guard → migrate → api → web → GitHub Release (deploy jobs skipped until `GCP_PROJECT_ID` is set)     |
| `migrate.yml`              | manual                     | `version` / `up` against production                                                                  |
| `scheduled-jobs.yml`       | hourly                     | POST `/internal/jobs/*` (no-op until `API_URL` is set)                                               |
| `db-backup.yml`            | nightly                    | `pg_dump` to R2 (skipped until `MIGRATION_DATABASE_URL` is set)                                      |
| `security-scan.yml`        | weekly                     | govulncheck + npm audit                                                                              |
| `dependabot-automerge.yml` | after CI on Dependabot PRs | merge patch/minor, bump, tag, release                                                                |

## Local development

See [Getting Started](../getting-started/quick-start.md) or run `make help`.
