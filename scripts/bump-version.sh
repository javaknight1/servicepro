#!/usr/bin/env bash
# Bump the root VERSION file for the next release.
#
# Auto-detects the bump from Conventional-Commits subjects since the last vX.Y.Z tag:
#   pre-1.0 (0.x, our case):  breaking `!`/BREAKING CHANGE, feat, minor, major -> MINOR; fix -> PATCH
#   post-1.0 (SemVer):        breaking or major -> MAJOR; feat or minor -> MINOR; fix -> PATCH
#   deps only:                PATCH (dependency bumps still ship something)
#   nothing user-facing:      PATCH, with a note
# In 0.x, MAJOR (1.0.0) is never automatic — leaving beta is a deliberate decision (--set).
#
# Legacy prefixes: before v0.8.0 this repo used `minor:` and `major:` instead of
# `feat:` / `feat!:`. They are still understood here so a range that contains
# them bumps correctly; the commit-msg hook rejects them for new commits.
#
# Override the computed version with --set X.Y.Z. The script REFUSES anything that isn't a
# strictly-higher valid semver than the current one.
#
# It edits VERSION ONLY — it does not commit or tag, because pushing a tag triggers the
# production release (RELEASING.md) and must stay an explicit step. It prints those commands.
#
# VERSION is the single source of truth for both the API (baked into the image as
# APP_VERSION) and the web bundle (VITE_APP_VERSION). frontend/package.json's "version"
# is NOT read by the release pipeline.
#
# Usage:
#   scripts/bump-version.sh              # auto-detect + apply
#   scripts/bump-version.sh --set 0.9.0  # force a specific version
#   scripts/bump-version.sh --dry-run    # show the decision, change nothing
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

DRY_RUN=false
OVERRIDE=""
while [ $# -gt 0 ]; do
  case "$1" in
    --set) OVERRIDE="${2:-}"; shift 2 ;;
    --set=*) OVERRIDE="${1#*=}"; shift ;;
    --dry-run) DRY_RUN=true; shift ;;
    -h|--help) grep '^#' "$0" | sed 's/^#\{1,\} \{0,1\}//'; exit 0 ;;
    *) echo "error: unknown argument '$1'" >&2; exit 2 ;;
  esac
done

is_semver() { [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; }
# strictly_greater A B -> exit 0 when A > B (numeric per-field compare)
strictly_greater() {
  local -a a b; IFS=. read -ra a <<<"$1"; IFS=. read -ra b <<<"$2"
  for i in 0 1 2; do
    (( a[i] > b[i] )) && return 0
    (( a[i] < b[i] )) && return 1
  done
  return 1  # equal
}

[ -f VERSION ] || { echo "error: VERSION file not found at repo root" >&2; exit 1; }
CURRENT=$(tr -d '[:space:]' < VERSION)
is_semver "$CURRENT" || { echo "error: VERSION '$CURRENT' is not X.Y.Z" >&2; exit 1; }
IFS=. read -r MA MI PA <<<"$CURRENT"

# Only release tags (vX.Y.Z). `|| true`: no tag yet is a normal state, and
# grep's exit 1 would otherwise abort under pipefail.
LAST_TAG=$(git tag --list 'v*' --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1 || true)
if [ -n "$LAST_TAG" ]; then RANGE="${LAST_TAG}..HEAD"; else RANGE="HEAD"; fi
COUNT=$(git rev-list --count "$RANGE")
BODIES=$(git log --no-merges --pretty='%s%n%b' "$RANGE" 2>/dev/null || true)

has() { grep -qE "$1" <<<"$BODIES"; }
breaking=false; major=false; feat=false; fix=false; deps=false
# Explicit ifs, not `has X && flag=true`: a failing AND-list aborts under set -e.
if has '^[a-z]+(\([^)]*\))?!:';     then breaking=true; fi
if has 'BREAKING CHANGE';           then breaking=true; fi
if has '^major(\([^)]*\))?!?:';     then major=true;    fi  # legacy prefix
if has '^(feat|minor)(\([^)]*\))?!?:'; then feat=true;  fi  # minor: is legacy for feat:
if has '^fix(\([^)]*\))?!?:';       then fix=true;      fi
# deps is a PATCH signal, never more: the script can't tell a lockfile patch
# from a framework migration. Say so with --set when a bump warrants more.
if has '^deps(\([^)]*\))?!?:';      then deps=true;     fi

if [ -n "$OVERRIDE" ]; then
  NEXT="$OVERRIDE"
  REASON="--set"
  is_semver "$NEXT" || { echo "error: --set '$NEXT' is not X.Y.Z" >&2; exit 1; }
elif [ "$MA" -eq 0 ]; then
  # Pre-1.0: breaking changes (and legacy major:) ride a MINOR bump.
  if $breaking || $major; then
    NEXT="0.$((MI + 1)).0"; REASON="breaking change (0.x -> minor)"
  elif $feat; then
    NEXT="0.$((MI + 1)).0"; REASON="feat"
  elif $fix; then
    NEXT="0.${MI}.$((PA + 1))"; REASON="fix"
  elif $deps; then
    NEXT="0.${MI}.$((PA + 1))"; REASON="deps only — patch (use --set if a bump warrants more)"
  else
    NEXT="0.${MI}.$((PA + 1))"; REASON="no user-facing commits — defaulting to patch"
  fi
else
  if $breaking || $major; then
    NEXT="$((MA + 1)).0.0"; REASON="breaking change"
  elif $feat; then
    NEXT="${MA}.$((MI + 1)).0"; REASON="feat"
  elif $fix; then
    NEXT="${MA}.${MI}.$((PA + 1))"; REASON="fix"
  elif $deps; then
    NEXT="${MA}.${MI}.$((PA + 1))"; REASON="deps only — patch (use --set if a bump warrants more)"
  else
    NEXT="${MA}.${MI}.$((PA + 1))"; REASON="no user-facing commits — defaulting to patch"
  fi
fi

strictly_greater "$NEXT" "$CURRENT" || {
  echo "error: refusing to bump $CURRENT -> $NEXT (must be strictly higher)" >&2; exit 1; }

echo "current:  $CURRENT"
echo "next:     $NEXT   ($REASON)"
echo "commits:  $COUNT since ${LAST_TAG:-<no release tag yet>}"

if $DRY_RUN; then
  echo
  echo "(--dry-run: VERSION not modified)"
  exit 0
fi

printf '%s\n' "$NEXT" > VERSION

cat <<EOF

VERSION updated. Next:

  git add VERSION && git commit -m "release: v${NEXT}"
  git push origin master
  git tag v${NEXT} && git push origin v${NEXT}    # <- this starts the release pipeline

EOF
