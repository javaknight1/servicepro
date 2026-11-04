#!/usr/bin/env bash
# Create (or re-assert) ServicePro's Google Cloud cost guardrails (T213).
# Documentation that executes — see docs/ops/cost-guardrails.md.
#
#   1. A monthly budget on the project's billing account, scoped to this
#      project, emailing at 50%, 90% and 100% of actual spend and at 100% of
#      forecast spend. Billing admins/users get the emails by default; with
#      ALERT_EMAIL set, the Monitoring email channel created by
#      scripts/setup-monitoring.sh gets them too.
#   2. The Artifact Registry cleanup policy (keep the 20 newest images, delete
#      anything older than 90 days), so every release's image doesn't pile up
#      storage forever.
#
# A budget only ALERTS — it never stops spend. The hard ceilings are Cloud Run
# --max-instances and the DB pool, pinned in scripts/deploy-cloudrun.sh.
#
# Idempotent: an existing budget with the same display name is left alone
# (change it with `gcloud billing budgets update`, see the doc); the cleanup
# policy is re-applied as-is. Never deletes a budget.
#
# Usage:
#   scripts/setup-cost-guardrails.sh                         # $10/month budget
#   BUDGET_AMOUNT=25 ALERT_EMAIL=you@example.com scripts/setup-cost-guardrails.sh
# Needs: gcloud logged in as a Billing Account Administrator (budgets) with
# Artifact Registry admin on the project (Owner works).
set -euo pipefail

PROJECT="${GCP_PROJECT_ID:-service-pro-512609}"
REGION="${GCP_REGION:-us-east5}"
AR_REPO="${AR_REPOSITORY:-servicepro}"
BUDGET_AMOUNT="${BUDGET_AMOUNT:-10}"          # USD per calendar month
BUDGET_NAME="${BUDGET_NAME:-servicepro-monthly}"
ALERT_EMAIL="${ALERT_EMAIL:-}"

BILLING=$(gcloud billing projects describe "$PROJECT" --format='value(billingAccountName)')
BILLING="${BILLING#billingAccounts/}"
: "${BILLING:?project ${PROJECT} has no billing account linked}"

echo "project: ${PROJECT}   billing account: ${BILLING}"
echo "budget:  ${BUDGET_NAME} = ${BUDGET_AMOUNT} USD/month"
echo

gcloud services enable billingbudgets.googleapis.com --project="$PROJECT"

# ─── 1. budget ──────────────────────────────────────────────────────
existing=$(gcloud billing budgets list --billing-account="$BILLING" \
  --format='value(name,displayName)' 2>/dev/null | awk -F'\t' -v n="$BUDGET_NAME" '$2 == n { print $1; exit }')
if [ -n "$existing" ]; then
  echo "ok  budget exists: ${BUDGET_NAME} (${existing})"
  echo "    change the amount: gcloud billing budgets update ${existing} --budget-amount=<N>USD"
else
  channel_flags=()
  if [ -n "$ALERT_EMAIL" ]; then
    channel=$(gcloud beta monitoring channels list --project="$PROJECT" \
      --filter="type='email' AND labels.email_address='${ALERT_EMAIL}'" \
      --format="value(name)" 2>/dev/null | head -1)
    if [ -n "$channel" ]; then
      channel_flags=(--notifications-rule-monitoring-notification-channels="$channel")
      echo "    also notifying Monitoring channel ${channel}"
    else
      echo "    no Monitoring email channel for ${ALERT_EMAIL} yet (run scripts/setup-monitoring.sh); billing admins still get emails"
    fi
  fi
  gcloud billing budgets create --billing-account="$BILLING" \
    --display-name="$BUDGET_NAME" \
    --budget-amount="${BUDGET_AMOUNT}USD" \
    --calendar-period=month \
    --filter-projects="projects/${PROJECT}" \
    --threshold-rule=percent=0.5 \
    --threshold-rule=percent=0.9 \
    --threshold-rule=percent=1.0 \
    --threshold-rule=percent=1.0,basis=forecasted-spend \
    ${channel_flags[@]+"${channel_flags[@]}"} >/dev/null
  echo "ok  created budget: ${BUDGET_NAME}"
fi

# ─── 2. Artifact Registry cleanup policy ────────────────────────────
policy=$(mktemp)
trap 'rm -f "$policy"' EXIT
cat > "$policy" <<'JSON'
[
  {"name": "keep-recent", "action": {"type": "Keep"}, "mostRecentVersions": {"keepCount": 20}},
  {"name": "delete-old", "action": {"type": "Delete"}, "condition": {"olderThan": "90d"}}
]
JSON
gcloud artifacts repositories set-cleanup-policies "$AR_REPO" \
  --project="$PROJECT" --location="$REGION" --policy="$policy" --no-dry-run >/dev/null
echo "ok  Artifact Registry ${AR_REPO}: keep 20 newest, delete older than 90 days"

echo
echo "Done. Budgets: https://console.cloud.google.com/billing/${BILLING}/budgets"
