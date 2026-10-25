# Deploy + rollback runbook

> **Live since v0.9.0 (2026-10-24).** Every `v*` tag deploys: Cloud Run
> `servicepro-api` in **us-east5** (GCP project `service-pro-512609`), Worker
> `servicepro-web` on <https://servicepro.sharkfins.xyz>, database Neon
> Postgres 18 (AWS us-east-2). One-time setup: [`cloud-run-setup.md`](./cloud-run-setup.md).

ServicePro ships from a **`v*` tag**, never from a push to `master`. See
[`RELEASING.md`](../../RELEASING.md) for the process. A tag runs
`.github/workflows/release-deploy.yml`, which ships two artifacts:

- **API** — `scripts/deploy-cloudrun.sh` builds `backend/Dockerfile` in CI
  (build args `GIT_SHA`, `APP_VERSION`), pushes
  `${REGION}-docker.pkg.dev/${PROJECT}/servicepro/api:<sha>` (and `:vX.Y.Z`),
  deploys a new Cloud Run revision of `servicepro-api` with `--no-traffic`, then
  shifts 100% to it. The job polls `vars.API_URL/version` until `git_sha`
  matches the tagged commit.
- **Web** — `frontend/` builds with `VITE_API_URL=/api` (same-origin), source maps go to
  Sentry and are stripped, then `wrangler deploy` ships the `servicepro-web`
  Worker (static assets + `/api/*` proxy). The job then polls
  `https://servicepro.sharkfins.xyz/version` through the proxy.

Every release is a new immutable artifact on both sides, so rollback is always
"point at a known-good prior artifact".

---

## When to roll back

Within ~30 minutes of a release, roll back without hesitation if:

- the `api` job fails at traffic shift or verification (the prior revision is
  still serving, but **this release's migration has already applied**);
- 5xx rate spikes in Sentry / Cloud Run metrics;
- a clearly version-shaped bug ("worked yesterday") is reported.

Don't roll back for cosmetic issues — a rollback also removes the release's
good changes.

## API rollback (Cloud Run)

Cloud Run keeps every revision addressable; rollback is a traffic flip — no
rebuild, no migration replay.

```bash
export SERVICE=servicepro-api REGION=us-east5

gcloud run revisions list --service="$SERVICE" --region="$REGION"
gcloud run services describe "$SERVICE" --region="$REGION" --format='value(status.traffic)'

# Revision names carry the tag with dots -> dashes (v0.8.0 -> servicepro-api-v0-8-0;
# a re-run attempt adds -r2, -r3 ...).
gcloud run services update-traffic "$SERVICE" --region="$REGION" \
  --to-revisions=servicepro-api-v0-7-1=100
```

Gradual: `--to-revisions=servicepro-api-v0-7-1=90,servicepro-api-v0-8-0=10`,
watch Sentry, then decide.

### Migration caveat

A traffic rollback does **not** roll back the schema, and `migrate` runs
before the deploy. That's why migrations must be additive (expand/contract):
the previous revision has to keep working against the newer schema. If a
release shipped a destructive change in one step, roll traffic back anyway to
stop the bleeding, then fix forward with a new migration + release. Never run
`migrate down` against production.

## Web rollback (Cloudflare Workers)

Workers keeps prior versions of `servicepro-web`:

```bash
cd frontend
npx wrangler deployments list        # recent deployments + version ids
npx wrangler rollback                # back to the previous version
npx wrangler rollback <version-id>   # or a specific one
```

Dashboard path: Workers & Pages → `servicepro-web` → Deployments → roll back.
Check `https://servicepro.sharkfins.xyz/` in a fresh incognito window.

If the API was rolled back too, roll the web back to the matching release so
the bundle doesn't call endpoints that no longer exist.

## Forward-fix vs rollback

Roll back first if the release is < 30 minutes old or users see errors.
Forward-fix (commit, `make bump`, tag) when the damage is bounded, the fix is
small, or a destructive schema change would make rollback unsafe.

## How the pipeline catches a silent failure

1. **`--no-traffic` deploy, then an explicit traffic shift.** A revision that
   can't boot (missing secret, unreachable DB/Redis, panic) fails its startup
   probe while 100% of traffic is still on the prior revision.
2. **`/version` polling.** `/version` reports `GIT_SHA` baked into the image, so
   the job only goes green once the new code is actually serving — first on the
   run.app URL, then through the Worker on the public domain.

Triage a failure at either step:

```bash
gcloud run revisions list --service=servicepro-api --region=us-east5
gcloud run services logs read servicepro-api --region=us-east5 --limit=100
```

## Releases with hosting unconfigured

While `vars.GCP_PROJECT_ID` is unset, the `guard` job sets `deploy=false`,
`migrate`/`api`/`web` show as **skipped**, and `release` still publishes the
GitHub Release notes. That is expected, not a failure.
