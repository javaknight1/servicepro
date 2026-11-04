#!/usr/bin/env bash
# Create (or re-assert) ServicePro's Google Cloud uptime checks and their email
# alerts (T251). Documentation that executes — see docs/ops/monitoring.md.
#
# Why an outside probe: Sentry and PostHog report from inside the app, so an
# outage looks like a quiet night. The Worker keeps serving the SPA while Cloud
# Run is down. Only an outside prober tells the difference.
#
# Probes (GET every 5 min from every probe region; an alert fires only when
# MORE THAN ONE region fails, so one flaky region doesn't page):
#   servicepro-web           SITE_HOST/          → Worker, DNS, certificate
#   servicepro-edge          SITE_HOST/version   → the path users' API calls
#                            take: Worker proxy → Cloud Run
#   servicepro-ready-direct  <run.app>/health/ready → API up but DB unreachable
#                            or schema behind/dirty (503); direct, so a Worker
#                            outage doesn't mask it
# Edge failing while ready-direct passes = the Worker/Cloudflare, not the API.
#
# Idempotent: re-running skips anything that already exists (matched by display
# name). It never deletes. To change a probe, delete it in the console
# (Monitoring → Uptime checks) and re-run.
#
# Usage:
#   ALERT_EMAIL=you@example.com scripts/setup-monitoring.sh
# Needs: gcloud logged in as a user with Monitoring Editor on the project
# (Owner works). Env overrides: GCP_PROJECT_ID, GCP_REGION, SERVICE_NAME,
# SITE_HOST, RUN_HOST.
set -euo pipefail

PROJECT="${GCP_PROJECT_ID:-service-pro-512609}"
REGION="${GCP_REGION:-us-east5}"
SERVICE="${SERVICE_NAME:-servicepro-api}"
SITE_HOST="${SITE_HOST:-servicepro.sharkfins.xyz}"
: "${ALERT_EMAIL:?set ALERT_EMAIL to the address that should receive outage alerts}"

if [ -z "${RUN_HOST:-}" ]; then
  RUN_HOST=$(gcloud run services describe "$SERVICE" --project="$PROJECT" \
    --region="$REGION" --format='value(status.url)' | sed 's#^https://##; s#/$##')
fi
: "${RUN_HOST:?could not resolve the Cloud Run URL; set RUN_HOST}"

echo "project: ${PROJECT}"
echo "site:    ${SITE_HOST}   api (direct): ${RUN_HOST}"
echo "email:   ${ALERT_EMAIL}"
echo

gcloud services enable monitoring.googleapis.com --project="$PROJECT"

# ─── 1. notification channel ────────────────────────────────────────
CHANNEL=$(gcloud beta monitoring channels list --project="$PROJECT" \
  --filter="type='email' AND labels.email_address='${ALERT_EMAIL}'" \
  --format="value(name)" 2>/dev/null | head -1)
if [ -n "$CHANNEL" ]; then
  echo "ok  email channel exists: ${CHANNEL}"
else
  CHANNEL=$(gcloud beta monitoring channels create --project="$PROJECT" \
    --display-name="ServicePro alerts (email)" --type=email \
    --channel-labels="email_address=${ALERT_EMAIL}" --format="value(name)")
  echo "ok  created email channel: ${CHANNEL}"
  echo "    Send yourself a test: Monitoring → Alerting → Edit notification channels."
fi

# ensure_policy NAME FILE — create an alert policy from FILE unless one with
# that display name exists.
ensure_policy() {
  local name="$1" file="$2" existing
  existing=$(gcloud beta monitoring policies list --project="$PROJECT" \
    --filter="displayName='${name}'" --format="value(name)" 2>/dev/null | head -1)
  if [ -n "$existing" ]; then
    echo "ok  alert policy exists: ${name}"
  else
    gcloud beta monitoring policies create --project="$PROJECT" --policy-from-file="$file" >/dev/null
    echo "ok  created alert policy: ${name}"
  fi
  rm -f "$file"
}

# uptime_check_id NAME — the check's id (last path segment), or empty.
uptime_check_id() {
  gcloud monitoring uptime list-configs --project="$PROJECT" \
    --filter="displayName='$1'" --format="value(name)" 2>/dev/null | head -1 | sed 's#.*/##'
}

# ensure_probe CHECK_NAME HOST PATH MATCH POLICY_NAME RUNBOOK
ensure_probe() {
  local check_name="$1" host="$2" path="$3" match="$4" policy_name="$5" runbook="$6"
  local check_id policy_file match_json

  check_id=$(uptime_check_id "$check_name")
  if [ -n "$check_id" ]; then
    echo "ok  uptime check exists: ${check_name}"
  else
    gcloud monitoring uptime create "$check_name" --project="$PROJECT" \
      --resource-type=uptime-url --resource-labels="host=${host},project_id=${PROJECT}" \
      --protocol=https --port=443 --path="$path" --request-method=get \
      --period=5 --timeout=10 \
      --matcher-type=contains-string --matcher-content="$match" >/dev/null
    check_id=$(uptime_check_id "$check_name")
    echo "ok  created uptime check: ${check_name}"
  fi
  [ -n "$check_id" ] || { echo "error: no id for uptime check ${check_name}" >&2; exit 1; }

  policy_file=$(mktemp)
  match_json=$(printf '%s' "$match" | jq -Rs . | sed 's/^"//; s/"$//')
  cat > "$policy_file" <<JSON
{
  "displayName": "${policy_name}",
  "documentation": {
    "content": "https://${host}${path} stopped returning a 2xx body containing ${match_json} from more than one probe region.\\n\\n${runbook}\\n\\nRunbooks: docs/ops/monitoring.md, docs/ops/deploy.md (rollback)",
    "mimeType": "text/markdown"
  },
  "combiner": "OR",
  "conditions": [{
    "displayName": "${host}${path} failing in >1 region",
    "conditionThreshold": {
      "filter": "resource.type = \\"uptime_url\\" AND metric.type = \\"monitoring.googleapis.com/uptime_check/check_passed\\" AND metric.label.check_id = \\"${check_id}\\"",
      "aggregations": [{
        "alignmentPeriod": "300s", "perSeriesAligner": "ALIGN_NEXT_OLDER",
        "crossSeriesReducer": "REDUCE_COUNT_FALSE", "groupByFields": ["resource.label.host"]
      }],
      "comparison": "COMPARISON_GT", "thresholdValue": 1, "duration": "60s", "trigger": { "count": 1 }
    }
  }],
  "notificationChannels": ["${CHANNEL}"],
  "alertStrategy": { "autoClose": "1800s" }
}
JSON
  ensure_policy "$policy_name" "$policy_file"
}

# ─── 2. uptime checks + alert policies ──────────────────────────────
ensure_probe "servicepro-web" "$SITE_HOST" "/" 'id="root"' \
  "ServicePro website is down" \
  "First checks:\\n1. curl -sSI https://${SITE_HOST}/\\n2. Cloudflare → Workers & Pages → servicepro-web → Deployments (roll back: cd frontend && npx wrangler rollback)\\n3. DNS + certificate for ${SITE_HOST}"

ensure_probe "servicepro-edge" "$SITE_HOST" "/version" '"git_sha"' \
  "ServicePro API unreachable via ${SITE_HOST}" \
  "Users' API calls are failing. First checks:\\n1. Is servicepro-ready-direct failing too? Then it's the API: curl -sS https://${RUN_HOST}/health/ready\\n2. Direct OK but this failing: the Worker proxy (frontend/worker/proxy.ts) — Cloudflare → servicepro-web → Logs; check API_ORIGIN in frontend/wrangler.jsonc"

ensure_probe "servicepro-ready-direct" "$RUN_HOST" "/health/ready" '"status"' \
  "ServicePro API is down or not ready" \
  "/health/ready returns 503 when the DB is unreachable or the schema is behind/dirty. First checks:\\n1. curl -sS https://${RUN_HOST}/health/ready (no answer = container down)\\n2. gcloud run revisions list --service=${SERVICE} --region=${REGION}; gcloud run services logs read ${SERVICE} --region=${REGION} --limit=100\\n3. DB: Neon console (compute suspended? connection limit?)\\n4. Schema behind/dirty: did the release migrate job run? RELEASING.md"

echo
echo "Done. Uptime checks: https://console.cloud.google.com/monitoring/uptime?project=${PROJECT}"
echo "      Alert policies: https://console.cloud.google.com/monitoring/alerting/policies?project=${PROJECT}"
