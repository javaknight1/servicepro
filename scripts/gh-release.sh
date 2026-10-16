#!/usr/bin/env bash
# Publish the GitHub Release (the changelog) for the pushed tag. Called by
# .github/workflows/release-deploy.yml.
#
# Commits-as-changelog: there is no CHANGELOG.md to maintain (it is frozen at
# v0.7.0). We push straight to master, so GitHub's --generate-notes (which lists
# merged PRs) would be nearly empty. Build the notes from the commit subjects in
# this tag's range instead. Requires GH_TOKEN, plus a full-history checkout with
# tags (fetch-depth: 0) so `git describe` can find the previous tag.
#
# The changelog is user-facing, so housekeeping commits are filtered out by their
# type prefix: deps/chore/docs/ci/build/test/style/refactor/perf/release, plus bare
# TODO: notes. feat:/fix: (and unprefixed, and legacy minor:/major:) commits
# survive. Prefix your commits accordingly — see RELEASING.md.
#
# `deps` is filtered because Dependabot titles its PRs that way ("deps(npm): bump
# ..."). A dependency change worth announcing should be committed as fix: or feat:
# on its own merits (a patched CVE is a fix).
#
# Reverts are filtered only when what they revert was itself housekeeping —
# `Revert "chore: ..."` is still chore. Reverting a feat: or fix: IS user-facing.
#
# Local preview (prints the notes, publishes nothing):
#   GITHUB_REF_NAME=v0.8.0 scripts/gh-release.sh --dry-run
set -euo pipefail

DRY_RUN=false
if [ "${1:-}" = "--dry-run" ]; then DRY_RUN=true; fi

: "${GITHUB_REF_NAME:?GITHUB_REF_NAME (the tag, e.g. v0.8.0) must be set}"
SERVER_URL="${GITHUB_SERVER_URL:-https://github.com}"
REPOSITORY="${GITHUB_REPOSITORY:-javaknight1/servicepro}"

prev=$(git describe --tags --abbrev=0 "${GITHUB_REF_NAME}^" 2>/dev/null || true)
if [ -n "$prev" ]; then
  range="$prev..${GITHUB_REF_NAME}"
  footer="**Full changelog**: ${SERVER_URL}/${REPOSITORY}/compare/${prev}...${GITHUB_REF_NAME}"
else
  range="${GITHUB_REF_NAME}"
  footer=""
fi

# grep returns non-zero when it filters everything out; tolerate that under pipefail.
HOUSEKEEPING='(deps|chore|docs|ci|build|test|style|refactor|perf|release)(\([^)]*\))?!?:'
changes=$(git log --no-merges --pretty='- %s (%h)' "$range" \
  | grep -vE "^- ${HOUSEKEEPING}" \
  | grep -vE "^- Revert \"${HOUSEKEEPING}" \
  | grep -vE '^- TODO:' || true)

notes_file="${RUNNER_TEMP:-${TMPDIR:-/tmp}}/release-notes.md"
{
  echo "## What changed"
  echo
  if [ -n "$changes" ]; then
    echo "$changes"
  else
    echo "_No user-facing changes in this release._"
  fi
  echo
  echo "$footer"
} > "$notes_file"

if $DRY_RUN; then
  cat "$notes_file"
  exit 0
fi

gh release create "$GITHUB_REF_NAME" --title "$GITHUB_REF_NAME" --notes-file "$notes_file" --verify-tag
