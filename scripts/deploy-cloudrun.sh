#!/usr/bin/env bash
# Build the API image, push it to Artifact Registry, and deploy it to Cloud Run.
# Called by .github/workflows/release-deploy.yml after migrations have applied.
#
# NOT LIVE YET: the GCP project does not exist. release-deploy.yml skips the job
# that calls this script until the repo variable GCP_PROJECT_ID is set. One-time
# setup: docs/ops/cloud-run-setup.md.
#
# We build in CI (not Cloud Build) because ci.yml's docker-build job already
# builds this exact backend/Dockerfile on every PR — a release ships the image
# we've been testing all along, and it keeps us off Cloud Build's meter.
#
# The image is tagged twice: with the immutable commit SHA (what Cloud Run
# deploys — a moving tag would make "which code is live" unanswerable) and with
# the release tag for human legibility in the registry UI.
#
# Deploy is `--no-traffic` first, then an explicit traffic migration. The
# revision boots and passes its startup probe before it gets traffic; if it
# crash-loops, traffic stays on the prior revision and this script exits non-zero.
#
# Requires (set by the workflow):
#   GCP_PROJECT_ID   target project
#   GCP_REGION       e.g. us-east5 (production)
#   AR_REPOSITORY    Artifact Registry repo name, e.g. servicepro
#   SERVICE_NAME     Cloud Run service name, e.g. servicepro-api
#   GIT_SHA          commit being released (immutable image tag)
#   RELEASE_TAG      e.g. v0.8.0 (human-facing tag + Cloud Run revision suffix)
# Assumes `gcloud` is already authenticated (google-github-actions/auth).
set -euo pipefail

: "${GCP_PROJECT_ID:?}" "${GCP_REGION:?}" "${AR_REPOSITORY:?}"
: "${SERVICE_NAME:?}" "${GIT_SHA:?}" "${RELEASE_TAG:?}"

cd "$(git rev-parse --show-toplevel)"

REGISTRY="${GCP_REGION}-docker.pkg.dev/${GCP_PROJECT_ID}/${AR_REPOSITORY}"
IMAGE_SHA="${REGISTRY}/api:${GIT_SHA}"
IMAGE_TAG="${REGISTRY}/api:${RELEASE_TAG}"

# Cloud Run revision suffixes allow only lowercase alphanumerics and dashes —
# `v0.8.0` becomes `v0-8-0`. A re-run of a failed release must not reuse the
# first attempt's name (the revision exists and can't be deleted while newest),
# so attempt 2+ gets `-r<attempt>`.
REVISION_SUFFIX=$(echo "$RELEASE_TAG" | tr '[:upper:]' '[:lower:]' | tr -c 'a-z0-9' '-' | sed 's/-*$//')
if [ "${GITHUB_RUN_ATTEMPT:-1}" -gt 1 ]; then
  REVISION_SUFFIX="${REVISION_SUFFIX}-r${GITHUB_RUN_ATTEMPT}"
fi

echo "--- Building ${IMAGE_SHA}"
docker build \
  --file backend/Dockerfile \
  --build-arg "GIT_SHA=${GIT_SHA}" \
  --build-arg "APP_VERSION=${RELEASE_TAG#v}" \
  --tag "$IMAGE_SHA" \
  --tag "$IMAGE_TAG" \
  backend

echo "--- Pushing to Artifact Registry"
docker push "$IMAGE_SHA"
docker push "$IMAGE_TAG"

# has_enabled_version NAME -> exit 0 if the secret has an ENABLED VERSION.
#
# Tests for a version, not the secret: a secret with zero versions passes
# `gcloud secrets describe`, but mounting NAME:latest from it fails the revision
# at startup. A gcloud FAILURE (usually the deploy SA lacking
# roles/secretmanager.viewer) aborts the deploy instead of reading as "none", so
# a permission problem can't silently ship a revision with a feature switched off.
has_enabled_version() {
  local out
  if ! out=$(gcloud secrets versions list "$1" \
               --project="$GCP_PROJECT_ID" \
               --filter="state=enabled" --limit=1 \
               --format="value(name)" 2>&1); then
    if grep -q 'NOT_FOUND' <<<"$out"; then
      return 1
    fi
    echo "error: cannot list versions of secret $1 — the deploy service account needs" >&2
    echo "       roles/secretmanager.viewer on it (metadata only, no payload access):" >&2
    echo "       $out" >&2
    exit 1
  fi
  [ -n "$(printf '%s\n' "$out" | grep -E '^[0-9]+$' || true)" ]
}

# Optional secrets: mounted only when they have an enabled version, so a project
# that hasn't set one up yet still deploys. --update-secrets (not --set-secrets)
# adds these and leaves the required references (DATABASE_URL, JWT_SECRET) that
# were attached once at service creation alone.
#
#   printf '%s' '<value>' | gcloud secrets versions add <NAME> --data-file=-
#
#   SENTRY_DSN             error reporting (off without it)
#   RESEND_API_KEY         transactional email
#   STRIPE_SECRET_KEY      payments
#   STRIPE_WEBHOOK_SECRET  Stripe webhook signature verification
#   POSTHOG_API_KEY        product analytics
#   REDIS_URL              Upstash (rediss://...) once Redis is provisioned
#   TEXTBELT_API_KEY       SMS
#   INTERNAL_JOBS_SECRET   X-Internal-Jobs-Token for /internal/jobs/* (scheduled-jobs.yml)
#   EDGE_PROXY_SECRET      shared with the servicepro-web Worker for client-IP forwarding
OPTIONAL_SECRETS=(
  SENTRY_DSN
  RESEND_API_KEY
  STRIPE_SECRET_KEY
  STRIPE_WEBHOOK_SECRET
  POSTHOG_API_KEY
  REDIS_URL
  TEXTBELT_API_KEY
  INTERNAL_JOBS_SECRET
  EDGE_PROXY_SECRET
)
SECRET_FLAGS=()
for s in "${OPTIONAL_SECRETS[@]}"; do
  if has_enabled_version "$s"; then
    SECRET_FLAGS+=("--update-secrets=${s}=${s}:latest")
    echo "--- ${s}: attached"
  else
    echo "--- note: ${s} has no enabled version; not attached"
  fi
done

# Scaling is pinned HERE, not left to whatever was last set in the console.
# --max-instances x DB_MAX_OPEN_CONNS must stay under the Postgres connection
# limit (docs/ops/cloud-run-setup.md). --update-env-vars MERGES (it never drops
# the other vars or secret refs): production must never migrate at startup (the
# release `migrate` job owns that) or run in-process tickers (scheduled-jobs.yml
# drives them through /internal/jobs/*), because Cloud Run throttles CPU between
# requests and scales to zero.
echo "--- Deploying revision (no traffic)"
gcloud run deploy "$SERVICE_NAME" \
  --project="$GCP_PROJECT_ID" \
  --region="$GCP_REGION" \
  --image="$IMAGE_SHA" \
  --revision-suffix="$REVISION_SUFFIX" \
  --concurrency=40 \
  --max-instances=4 \
  --cpu=1 \
  --memory=512Mi \
  --timeout=60 \
  --update-env-vars="RUN_MIGRATIONS_ON_STARTUP=false,RUN_BACKGROUND_WORKERS=false" \
  ${SECRET_FLAGS[@]+"${SECRET_FLAGS[@]}"} \
  --no-traffic \
  --quiet

# Address the revision by suffix (not --to-latest) so a concurrent deploy can't
# make us promote someone else's revision.
echo "--- Migrating traffic to ${SERVICE_NAME}-${REVISION_SUFFIX}"
gcloud run services update-traffic "$SERVICE_NAME" \
  --project="$GCP_PROJECT_ID" \
  --region="$GCP_REGION" \
  --to-revisions="${SERVICE_NAME}-${REVISION_SUFFIX}=100" \
  --quiet

SERVICE_URL=$(gcloud run services describe "$SERVICE_NAME" \
  --project="$GCP_PROJECT_ID" --region="$GCP_REGION" \
  --format='value(status.url)')
echo "deployed: ${SERVICE_URL} (revision ${SERVICE_NAME}-${REVISION_SUFFIX})"
