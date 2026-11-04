#!/usr/bin/env bash
# Sentry cron monitor check-ins for GitHub Actions schedules (T252).
#
#   scripts/sentry-cron.sh start  <slug> <crontab> <margin-min> <max-runtime-min> <failures-before-issue>
#       -> prints a check-in id (empty when check-ins are off)
#   scripts/sentry-cron.sh finish <slug> <check-in-id> <ok|error>
#
# `start` sends an in_progress check-in carrying the monitor's config, so the
# monitor is created (upserted) on its first check-in and kept in sync with the
# schedule here — nothing to click in Sentry. `finish` closes that check-in.
# A run that never finishes is marked failed after max-runtime; a run that never
# starts (GitHub skipped or delayed the schedule past the margin) is "missed".
# Either one opens a Sentry issue after <failures-before-issue> in a row.
#
# Reads SENTRY_CRONS_DSN — the servicepro-api project's DSN
# (https://<public-key>@o<org>.ingest.<region>.sentry.io/<project-id>). When it
# is unset this is a silent no-op, so workflows keep working without it.
# Check-ins are best-effort: a Sentry outage never fails the job itself.
# Never prints the DSN.
set -uo pipefail

DSN="${SENTRY_CRONS_DSN:-}"
ENVIRONMENT="${SENTRY_ENVIRONMENT:-production}"
[ -n "$DSN" ] || exit 0

# https://KEY@HOST/PROJECT -> https://HOST/api/PROJECT/cron/<slug>/KEY/
rest="${DSN#*://}"
key="${rest%%@*}"
hostpath="${rest#*@}"
host="${hostpath%%/*}"
project="${hostpath##*/}"
if [ -z "$key" ] || [ "$key" = "$rest" ] || [ -z "$host" ] || [ -z "$project" ]; then
  echo "::warning::SENTRY_CRONS_DSN is not a DSN (https://<key>@<host>/<project>); skipping check-in" >&2
  exit 0
fi

endpoint() { printf 'https://%s/api/%s/cron/%s/%s/' "$host" "$project" "$1" "$key"; }

new_id() {
  if command -v uuidgen >/dev/null; then
    uuidgen | tr '[:upper:]' '[:lower:]'
  elif [ -r /proc/sys/kernel/random/uuid ]; then
    cat /proc/sys/kernel/random/uuid
  else
    python3 -c 'import uuid; print(uuid.uuid4())'
  fi
}

cmd="${1:-}"
case "$cmd" in
  start)
    [ $# -eq 6 ] || { echo "usage: $0 start <slug> <crontab> <margin-min> <max-runtime-min> <failures-before-issue>" >&2; exit 2; }
    slug="$2" schedule="$3" margin="$4" runtime="$5" threshold="$6"
    id=$(new_id)
    body=$(jq -nc --arg s "$schedule" --argjson m "$margin" --argjson r "$runtime" --argjson t "$threshold" '{
      status: "in_progress",
      monitor_config: {
        schedule: {type: "crontab", value: $s},
        checkin_margin: $m, max_runtime: $r,
        failure_issue_threshold: $t, recovery_threshold: 1,
        timezone: "UTC"
      }
    }')
    if curl -fsS -m 10 -o /dev/null -X POST -H 'Content-Type: application/json' \
         --data "$body" "$(endpoint "$slug")?check_in_id=${id}&environment=${ENVIRONMENT}" 2>/dev/null; then
      echo "$id"
    else
      echo "::warning::Sentry check-in (in_progress) failed for ${slug}" >&2
    fi
    ;;
  finish)
    [ $# -eq 4 ] || { echo "usage: $0 finish <slug> <check-in-id> <ok|error>" >&2; exit 2; }
    slug="$2" id="$3" status="$4"
    case "$status" in ok|error) ;; *) echo "status must be ok or error" >&2; exit 2 ;; esac
    q="status=${status}&environment=${ENVIRONMENT}"
    # No id (the start check-in failed): send a standalone result anyway.
    [ -n "$id" ] && q="check_in_id=${id}&${q}"
    curl -fsS -m 10 -o /dev/null "$(endpoint "$slug")?${q}" 2>/dev/null \
      || echo "::warning::Sentry check-in (${status}) failed for ${slug}" >&2
    ;;
  *)
    echo "usage: $0 start|finish ..." >&2
    exit 2
    ;;
esac
exit 0
