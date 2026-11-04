# Production monitoring

What tells us production is broken before a customer does. Companion docs:
[`deploy.md`](./deploy.md) (deploy + rollback), [`backups.md`](./backups.md),
[`cloud-run-setup.md`](./cloud-run-setup.md).

| Layer                     | Catches                                             | Where                                                   |
| ------------------------- | --------------------------------------------------- | ------------------------------------------------------- |
| Release smoke test (T251) | A release that deployed but doesn't work end to end | `smoke` job in `release-deploy.yml`, `scripts/smoke.sh` |
| GCP uptime checks (T251)  | Site, Worker proxy or API down between releases     | `scripts/setup-monitoring.sh` (run once by hand)        |
| Sentry errors             | Exceptions in the API and the browser               | Sentry projects `servicepro-api`, `servicepro-web`      |

## Release smoke test

`scripts/smoke.sh` is read-only (GETs only, no login). The release pipeline runs
it after the `web` job; run it any time with `make smoke`
(`API_URL=https://<run.app host> make smoke` adds the direct checks).

| Check                     | Pass                                                       |
| ------------------------- | ---------------------------------------------------------- |
| `SITE/version`            | 200, `git_sha` = tagged commit, `app_version` = tag        |
| `SITE/`                   | 200 app shell, `Content-Security-Policy` + HSTS headers    |
| `SITE/login`              | 200 app shell (SPA fallback for deep links)                |
| `SITE/api/v1/users/me`    | 401 — Worker → API works, edge secret accepted (403 = not) |
| `API_URL/health/ready`    | 200 (503 = DB unreachable or schema behind/dirty)          |
| `API_URL/api/v1/users/me` | 403 — edge enforcement on (warning only, not a failure)    |

Each check retries with backoff (5 s doubling to 30 s) for `SMOKE_TIMEOUT`
seconds (180 in the pipeline). A failure fails the run, prints the rollback
commands from [`deploy.md`](./deploy.md) with the previous revision filled in,
and holds back the GitHub Release notes until a re-run of the `smoke` job
passes.

## Google Cloud uptime checks + alert

`scripts/setup-monitoring.sh` creates three uptime checks (GET every 5 minutes
from every probe region) and one email alert policy per check. A policy fires
only when **more than one region** fails, so one flaky region doesn't page.

| Uptime check              | Target                             | Must contain | Failing means                                 |
| ------------------------- | ---------------------------------- | ------------ | --------------------------------------------- |
| `servicepro-web`          | `servicepro.sharkfins.xyz/`        | `id="root"`  | Worker, DNS or certificate                    |
| `servicepro-edge`         | `servicepro.sharkfins.xyz/version` | `"git_sha"`  | Users' API path (Worker proxy → Cloud Run)    |
| `servicepro-ready-direct` | `<run.app>/health/ready`           | `"status"`   | API down, DB unreachable, schema behind/dirty |

Edge failing while ready-direct passes points at the Worker/Cloudflare, not the
API. Each alert's documentation lists the first commands to run.

The script is idempotent (skips anything that exists, matched by display name)
and never deletes. To change a check, delete it in Monitoring → Uptime checks
and re-run. Uptime checks are free up to 1M executions per month per project;
three checks every 5 minutes from ~6 regions is ~160k.

**Run it (one-time, needs Monitoring Editor or Owner on the project):**

```bash
gcloud auth login
gcloud config set project service-pro-512609
ALERT_EMAIL=<your alert address> scripts/setup-monitoring.sh
```

Then Monitoring → Alerting → Edit notification channels → send a test to the
email channel. Better Stack (T165) can replace these probes later.
