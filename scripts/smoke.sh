#!/usr/bin/env bash
# Post-deploy smoke test for production (T251).
#
# Hits the live URLs end to end and checks what unit tests can't: real DNS and
# certificate, the servicepro-web Worker (static assets, security headers, the
# /api proxy with its edge secret) and the Cloud Run API behind it.
#
# Runs at the end of every release (release-deploy.yml, `smoke` job); run it by
# hand any time with `make smoke`. Read-only: it only sends GETs, never logs in
# and never writes data.
#
# Checks (each retried with backoff until SMOKE_TIMEOUT runs out):
#   1. SITE/version          200; git_sha / app_version match EXPECTED_* (if set)
#   2. SITE/                 200 HTML app shell, Content-Security-Policy and
#                            Strict-Transport-Security headers present
#   3. SITE/login            200 HTML (SPA fallback for deep links)
#   4. SITE/api/v1/users/me  401 without a session: the Worker reached the API,
#                            the edge secret was accepted (a mismatch is 403)
#                            and auth middleware runs
#   5. API_URL/health/ready  200 (direct run.app; only when API_URL is set)
#   6. API_URL/api/v1/users/me  403 — edge enforcement blocks bypassing the
#                            Worker (only when API_URL is set; WARNING only)
#
# Env:
#   SITE_URL          default https://servicepro.sharkfins.xyz
#   API_URL           direct Cloud Run URL (optional; checks 5–6)
#   EXPECTED_SHA      commit that must be serving (optional)
#   EXPECTED_VERSION  app_version that must be serving, without the v (optional)
#   PREVIOUS_TAG      e.g. v0.9.0 — names the revision in the rollback hint
#   SMOKE_TIMEOUT     seconds to keep retrying each check (default 120)
#
# Exit 0 = all checks passed. Exit 1 = at least one failed; the rollback
# commands from docs/ops/deploy.md are printed.
#
# The check_* functions are called indirectly through run().
# shellcheck disable=SC2329
set -uo pipefail

SITE_URL="${SITE_URL:-https://servicepro.sharkfins.xyz}"
SITE_URL="${SITE_URL%/}"
API_URL="${API_URL:-}"
API_URL="${API_URL%/}"
EXPECTED_SHA="${EXPECTED_SHA:-}"
EXPECTED_VERSION="${EXPECTED_VERSION:-}"
EXPECTED_VERSION="${EXPECTED_VERSION#v}"
PREVIOUS_TAG="${PREVIOUS_TAG:-}"
SMOKE_TIMEOUT="${SMOKE_TIMEOUT:-120}"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

failures=0
warnings=0

# fetch URL — GET into $WORK/body and $WORK/headers; prints the HTTP status
# (000 when the request itself failed).
fetch() {
  curl -sS -m 20 -o "$WORK/body" -D "$WORK/headers" -w '%{http_code}' \
    -H 'User-Agent: servicepro-smoke' "$1" 2>"$WORK/curl.err" || true
}

header_present() { # NAME — case-insensitive response header lookup
  grep -qi "^$1:" "$WORK/headers"
}

# Each check_* function returns 0 on pass, or prints why and returns 1.

check_version() {
  local code sha ver
  code=$(fetch "$SITE_URL/version")
  [ "$code" = 200 ] || { echo "HTTP $code"; return 1; }
  sha=$(jq -r '.git_sha // ""' "$WORK/body" 2>/dev/null)
  ver=$(jq -r '.app_version // ""' "$WORK/body" 2>/dev/null)
  if [ -n "$EXPECTED_SHA" ] && [ "$sha" != "$EXPECTED_SHA" ]; then
    echo "git_sha=${sha:-<none>}, want $EXPECTED_SHA"; return 1
  fi
  if [ -n "$EXPECTED_VERSION" ] && [ "$ver" != "$EXPECTED_VERSION" ]; then
    echo "app_version=${ver:-<none>}, want $EXPECTED_VERSION"; return 1
  fi
  echo "app_version=$ver git_sha=$sha"
}

check_shell() {
  local code
  code=$(fetch "$SITE_URL/")
  [ "$code" = 200 ] || { echo "HTTP $code"; return 1; }
  grep -q 'id="root"' "$WORK/body" || { echo "body is not the SPA shell (no id=\"root\")"; return 1; }
  header_present 'content-security-policy' || { echo "Content-Security-Policy header missing (frontend/public/_headers)"; return 1; }
  header_present 'strict-transport-security' || { echo "Strict-Transport-Security header missing"; return 1; }
  echo "app shell + CSP + HSTS"
}

check_login() {
  local code
  code=$(fetch "$SITE_URL/login")
  [ "$code" = 200 ] || { echo "HTTP $code"; return 1; }
  grep -q 'id="root"' "$WORK/body" || { echo "deep link did not fall back to the SPA shell"; return 1; }
  echo "SPA fallback ok"
}

check_api_via_worker() {
  local code
  code=$(fetch "$SITE_URL/api/v1/users/me")
  case "$code" in
    401) echo "401 without a session (proxy + edge secret + auth ok)" ;;
    403) echo "HTTP 403: the API rejected the Worker — EDGE_PROXY_SECRET differs between the Worker and Secret Manager"; return 1 ;;
    200) echo "HTTP 200 for an anonymous /users/me — auth middleware is not running"; return 1 ;;
    *) echo "HTTP $code (want 401)"; return 1 ;;
  esac
}

check_ready_direct() {
  local code status
  code=$(fetch "$API_URL/health/ready")
  status=$(jq -r '.status // ""' "$WORK/body" 2>/dev/null)
  [ "$code" = 200 ] || { echo "HTTP $code status=${status:-?} (503 = DB unreachable or schema behind/dirty)"; return 1; }
  echo "status=$status"
}

check_edge_enforced() {
  local code
  code=$(fetch "$API_URL/api/v1/users/me")
  [ "$code" = 403 ] || { echo "HTTP $code (want 403): the run.app URL accepts /api/* without the Worker's edge secret"; return 1; }
  echo "direct /api/* blocked"
}

# run NAME FUNCTION [warn] — retry FUNCTION with backoff until it passes or
# SMOKE_TIMEOUT runs out. With "warn", a failure is a warning, not a failure.
run() {
  local name="$1" fn="$2" mode="${3:-fail}" deadline delay=5 out
  deadline=$(( $(date +%s) + SMOKE_TIMEOUT ))
  while :; do
    if out=$("$fn" 2>&1); then
      echo "PASS  $name — $out"
      [ -n "${GITHUB_STEP_SUMMARY:-}" ] && echo "| $name | pass | ${out} |" >> "$GITHUB_STEP_SUMMARY"
      return 0
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
      [ -s "$WORK/curl.err" ] && out="$out ($(tr -d '\n' < "$WORK/curl.err"))"
      if [ "$mode" = warn ]; then
        echo "WARN  $name — $out"
        [ -n "${GITHUB_ACTIONS:-}" ] && echo "::warning title=smoke: ${name}::${out}"
        [ -n "${GITHUB_STEP_SUMMARY:-}" ] && echo "| $name | **warning** | ${out} |" >> "$GITHUB_STEP_SUMMARY"
        warnings=$((warnings + 1))
      else
        echo "FAIL  $name — $out"
        [ -n "${GITHUB_ACTIONS:-}" ] && echo "::error title=smoke: ${name}::${out}"
        [ -n "${GITHUB_STEP_SUMMARY:-}" ] && echo "| $name | **FAIL** | ${out} |" >> "$GITHUB_STEP_SUMMARY"
        failures=$((failures + 1))
      fi
      return 1
    fi
    echo "      $name — $out; retrying in ${delay}s"
    sleep "$delay"
    delay=$(( delay * 2 > 30 ? 30 : delay * 2 ))
  done
}

command -v jq >/dev/null || { echo "smoke: jq is required" >&2; exit 2; }

echo "smoke: site=$SITE_URL api=${API_URL:-<not set>} sha=${EXPECTED_SHA:-any} version=${EXPECTED_VERSION:-any}"
if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  { echo "### Smoke test"; echo; echo "| check | result | detail |"; echo "|---|---|---|"; } >> "$GITHUB_STEP_SUMMARY"
fi

run "version via Worker" check_version
run "app shell + headers" check_shell
run "deep link /login" check_login
run "API via Worker (anonymous 401)" check_api_via_worker
if [ -n "$API_URL" ]; then
  run "API ready (direct)" check_ready_direct
  run "edge enforcement (direct /api 403)" check_edge_enforced warn
fi

if [ "$failures" -eq 0 ]; then
  echo "smoke: all checks passed (${warnings} warning(s))"
  exit 0
fi

prev_rev="servicepro-api-<previous-tag-with-dashes>"
if [ -n "$PREVIOUS_TAG" ]; then
  prev_rev="servicepro-api-$(echo "$PREVIOUS_TAG" | tr '[:upper:]' '[:lower:]' | tr -c 'a-z0-9' '-' | sed 's/-*$//')"
fi
rollback=$(cat <<EOF
Rollback (docs/ops/deploy.md) — decide within ~30 minutes of the release:

  # API: flip traffic back to the previous Cloud Run revision
  gcloud run revisions list --service=servicepro-api --region=us-east5
  gcloud run services update-traffic servicepro-api --region=us-east5 \\
    --to-revisions=${prev_rev}=100

  # Web: roll the servicepro-web Worker back to its previous version
  cd frontend && npx wrangler deployments list && npx wrangler rollback

Migrations are NOT rolled back (they are additive); never run migrate down on production.
EOF
)
echo
echo "smoke: ${failures} check(s) FAILED"
echo
echo "$rollback"
if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  { echo; echo "**${failures} smoke check(s) failed.**"; echo; echo '```'; echo "$rollback"; echo '```'; } >> "$GITHUB_STEP_SUMMARY"
fi
exit 1
