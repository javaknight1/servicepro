#!/bin/bash
# scripts/release.sh - Automated release script with version detection
#
# Version bump rules (in priority order):
#   - major: commit             → MAJOR (always, even pre-1.0)
#   - minor: commit             → MINOR
#   - BREAKING CHANGE or feat!  → MAJOR (MINOR if pre-1.0)
#   - feat                      → MINOR
#   - fix or perf               → PATCH
#   - docs, chore, test, etc.   → NO RELEASE
#
# Usage:
#   ./scripts/release.sh           # Auto-detect version
#   ./scripts/release.sh --dry-run # Preview without making changes
#   ./scripts/release.sh --force   # Force release even with no release-worthy commits

set -e

# =============================================================================
# Configuration
# =============================================================================

REPO_URL="https://github.com/javaknight1/servicepro"
MAIN_BRANCH="master"

# =============================================================================
# Colors & Helpers
# =============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

info()    { echo -e "${BLUE}ℹ${NC} $1"; }
success() { echo -e "${GREEN}✓${NC} $1"; }
warn()    { echo -e "${YELLOW}⚠${NC} $1"; }
error()   { echo -e "${RED}✗${NC} $1"; exit 1; }
header()  { echo -e "\n${BOLD}${CYAN}$1${NC}"; }

# =============================================================================
# Parse Arguments
# =============================================================================

DRY_RUN=false
FORCE_RELEASE=false
SKIP_TESTS=false
SKIP_CONFIRM=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)    DRY_RUN=true ;;
        --force)      FORCE_RELEASE=true ;;
        --skip-tests) SKIP_TESTS=true ;;
        --yes|-y)     SKIP_CONFIRM=true ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --dry-run     Preview release without making changes"
            echo "  --force       Force release even without release-worthy commits"
            echo "  --skip-tests  Skip running tests and bundle size check"
            echo "  --yes, -y     Skip confirmation prompts"
            echo "  --help, -h    Show this help"
            echo ""
            echo "Version bump rules (in priority order):"
            echo "  MAJOR: 'major:' commit (always, even pre-1.0)"
            echo "  MINOR: 'minor:' commit"
            echo "  MAJOR: Any commit with '!' or 'BREAKING CHANGE' (MINOR if pre-1.0)"
            echo "  MINOR: Any 'feat' commit"
            echo "  PATCH: Any 'fix' or 'perf' commit"
            echo "  NONE:  Only docs, chore, test, refactor, ci, etc."
            exit 0
            ;;
        *) error "Unknown option: $1. Use --help for usage." ;;
    esac
    shift
done

# =============================================================================
# Preflight Checks
# =============================================================================

header "🔍 Preflight Checks"

# Check we're in repo root
if [[ ! -d ".git" ]]; then
    error "Not in a git repository root"
fi

# Check for required tools
if ! command -v git-cliff &> /dev/null; then
    error "git-cliff not found. Install with: brew install git-cliff"
fi

if ! command -v gh &> /dev/null; then
    error "gh not found. Install with: brew install gh"
fi

# Check branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "$MAIN_BRANCH" && "$CURRENT_BRANCH" != "master" ]]; then
    error "Must be on $MAIN_BRANCH branch (currently on $CURRENT_BRANCH)"
fi
success "On branch: $CURRENT_BRANCH"

# Check working directory
if [[ -n "$(git status --porcelain)" ]]; then
    warn "Working directory has uncommitted changes:"
    git status --short
    echo ""
    if [[ "$SKIP_CONFIRM" != true ]]; then
        read -p "Continue anyway? (y/N) " -n 1 -r
        echo
        [[ ! $REPLY =~ ^[Yy]$ ]] && error "Aborted"
    fi
else
    success "Working directory clean"
fi

# Fetch latest
info "Fetching latest from remote..."
git fetch --tags --quiet
success "Fetched latest tags"

# =============================================================================
# Get Current Version
# =============================================================================

header "📌 Version Detection"

# Try VERSION file first, then fall back to git tags
if [[ -f "VERSION" ]]; then
    CURRENT_VERSION=$(cat VERSION | tr -d '[:space:]')
    success "Current version (from VERSION file): $CURRENT_VERSION"
else
    CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
    success "Current version (from git tag): $CURRENT_VERSION"
fi

# Parse version components
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# Get last tag for commit analysis
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [[ -z "$LAST_TAG" ]]; then
    COMMIT_RANGE="HEAD"
    info "No previous tags found, analyzing all commits"
else
    COMMIT_RANGE="${LAST_TAG}..HEAD"
    info "Analyzing commits since $LAST_TAG"
fi

# =============================================================================
# Analyze Commits
# =============================================================================

header "📝 Commit Analysis"

# Get commits since last tag
COMMITS=$(git log $COMMIT_RANGE --pretty=format:"%s" 2>/dev/null || git log --pretty=format:"%s")

if [[ -z "$COMMITS" ]]; then
    error "No commits found since last release. Nothing to release."
fi

# Initialize counters
MAJOR_COUNT=0
MINOR_COUNT=0
BREAKING_COUNT=0
FEAT_COUNT=0
FIX_COUNT=0
PERF_COUNT=0
DOCS_COUNT=0
REFACTOR_COUNT=0
TEST_COUNT=0
CHORE_COUNT=0
CI_COUNT=0
OTHER_COUNT=0

# Collect commits by type for display
declare -a MAJOR_COMMITS=()
declare -a MINOR_COMMITS=()
declare -a BREAKING_COMMITS=()
declare -a FEAT_COMMITS=()
declare -a FIX_COMMITS=()
declare -a PERF_COMMITS=()
declare -a OTHER_COMMITS=()

# Regex patterns (stored in variables to avoid bash parsing issues with parentheses)
# These match conventional commit format: type(scope)?: message
# Using simpler patterns that match the prefix
MAJOR_PATTERN='^major[(:!]'
MINOR_PATTERN='^minor[(:!]'
BREAKING_PATTERN1='^[a-z]+!:'
BREAKING_PATTERN2='^[a-z]+\([^)]+\)!:'
FEAT_PATTERN='^feat[(:!]'
FIX_PATTERN='^fix[(:!]'
PERF_PATTERN='^perf[(:!]'
DOCS_PATTERN='^docs[(:!]'
REFACTOR_PATTERN='^refactor[(:!]'
TEST_PATTERN='^test[(:!]'
CHORE_PATTERN='^chore[(:!]'
CI_PATTERN='^ci[(:!]'
STYLE_BUILD_PATTERN='^(style|build)[(:!]'

while IFS= read -r commit; do
    [[ -z "$commit" ]] && continue

    # Check for explicit major/minor commits first (highest priority)
    if [[ "$commit" =~ $MAJOR_PATTERN ]]; then
        ((MAJOR_COUNT++))
        MAJOR_COMMITS+=("$commit")
    elif [[ "$commit" =~ $MINOR_PATTERN ]]; then
        ((MINOR_COUNT++))
        MINOR_COMMITS+=("$commit")
    # Check for breaking changes
    elif [[ "$commit" =~ $BREAKING_PATTERN1 ]] || [[ "$commit" =~ $BREAKING_PATTERN2 ]] || [[ "$commit" == *"BREAKING CHANGE"* ]]; then
        ((BREAKING_COUNT++))
        BREAKING_COMMITS+=("$commit")
    elif [[ "$commit" =~ $FEAT_PATTERN ]]; then
        ((FEAT_COUNT++))
        FEAT_COMMITS+=("$commit")
    elif [[ "$commit" =~ $FIX_PATTERN ]]; then
        ((FIX_COUNT++))
        FIX_COMMITS+=("$commit")
    elif [[ "$commit" =~ $PERF_PATTERN ]]; then
        ((PERF_COUNT++))
        PERF_COMMITS+=("$commit")
    elif [[ "$commit" =~ $DOCS_PATTERN ]]; then
        ((DOCS_COUNT++))
        OTHER_COMMITS+=("📚 $commit")
    elif [[ "$commit" =~ $REFACTOR_PATTERN ]]; then
        ((REFACTOR_COUNT++))
        OTHER_COMMITS+=("♻️  $commit")
    elif [[ "$commit" =~ $TEST_PATTERN ]]; then
        ((TEST_COUNT++))
        OTHER_COMMITS+=("✅ $commit")
    elif [[ "$commit" =~ $CHORE_PATTERN ]]; then
        ((CHORE_COUNT++))
        # Don't add chore to display
    elif [[ "$commit" =~ $CI_PATTERN ]]; then
        ((CI_COUNT++))
        # Don't add ci to display
    elif [[ "$commit" =~ $STYLE_BUILD_PATTERN ]]; then
        ((OTHER_COUNT++))
    else
        ((OTHER_COUNT++))
        OTHER_COMMITS+=("❓ $commit")
    fi
done <<< "$COMMITS"

# Calculate totals
RELEASE_WORTHY=$((MAJOR_COUNT + MINOR_COUNT + BREAKING_COUNT + FEAT_COUNT + FIX_COUNT + PERF_COUNT))
NON_RELEASE=$((DOCS_COUNT + REFACTOR_COUNT + TEST_COUNT + CHORE_COUNT + CI_COUNT + OTHER_COUNT))
TOTAL_COMMITS=$((RELEASE_WORTHY + NON_RELEASE))

# Display commit summary
echo "Found $TOTAL_COMMITS commits since last release:"
echo ""

# Release-worthy commits (highlighted)
if [[ $RELEASE_WORTHY -gt 0 ]]; then
    echo -e "${GREEN}Release-worthy commits: $RELEASE_WORTHY${NC}"
    [[ $MAJOR_COUNT -gt 0 ]]    && echo -e "  ${RED}🚨 Major:       $MAJOR_COUNT${NC}"
    [[ $MINOR_COUNT -gt 0 ]]    && echo -e "  ${MAGENTA}🎯 Minor:       $MINOR_COUNT${NC}"
    [[ $BREAKING_COUNT -gt 0 ]] && echo -e "  ${RED}💥 Breaking:    $BREAKING_COUNT${NC}"
    [[ $FEAT_COUNT -gt 0 ]]     && echo -e "  ${GREEN}✨ Features:    $FEAT_COUNT${NC}"
    [[ $FIX_COUNT -gt 0 ]]      && echo -e "  ${YELLOW}🐛 Fixes:       $FIX_COUNT${NC}"
    [[ $PERF_COUNT -gt 0 ]]     && echo -e "  ${CYAN}⚡ Performance: $PERF_COUNT${NC}"
    echo ""
fi

# Non-release commits (dimmed)
if [[ $NON_RELEASE -gt 0 ]]; then
    echo -e "${DIM}Other commits (won't trigger release): $NON_RELEASE${NC}"
    [[ $DOCS_COUNT -gt 0 ]]     && echo -e "${DIM}  📚 Docs:       $DOCS_COUNT${NC}"
    [[ $REFACTOR_COUNT -gt 0 ]] && echo -e "${DIM}  ♻️  Refactor:   $REFACTOR_COUNT${NC}"
    [[ $TEST_COUNT -gt 0 ]]     && echo -e "${DIM}  ✅ Tests:      $TEST_COUNT${NC}"
    [[ $CHORE_COUNT -gt 0 ]]    && echo -e "${DIM}  🔧 Chores:     $CHORE_COUNT${NC}"
    [[ $CI_COUNT -gt 0 ]]       && echo -e "${DIM}  👷 CI:         $CI_COUNT${NC}"
    [[ $OTHER_COUNT -gt 0 ]]    && echo -e "${DIM}  📦 Other:      $OTHER_COUNT${NC}"
    echo ""
fi

# Show release-worthy commit details
if [[ ${#MAJOR_COMMITS[@]} -gt 0 ]]; then
    echo -e "${RED}Major Changes:${NC}"
    for commit in "${MAJOR_COMMITS[@]}"; do
        echo "    🚨 $commit"
    done
    echo ""
fi

if [[ ${#MINOR_COMMITS[@]} -gt 0 ]]; then
    echo -e "${MAGENTA}Minor Changes:${NC}"
    for commit in "${MINOR_COMMITS[@]}"; do
        echo "    🎯 $commit"
    done
    echo ""
fi

if [[ ${#BREAKING_COMMITS[@]} -gt 0 ]]; then
    echo -e "${RED}Breaking Changes:${NC}"
    for commit in "${BREAKING_COMMITS[@]}"; do
        echo "    💥 $commit"
    done
    echo ""
fi

if [[ ${#FEAT_COMMITS[@]} -gt 0 ]]; then
    echo -e "${GREEN}Features:${NC}"
    for commit in "${FEAT_COMMITS[@]}"; do
        echo "    ✨ $commit"
    done
    echo ""
fi

if [[ ${#FIX_COMMITS[@]} -gt 0 ]]; then
    echo -e "${YELLOW}Fixes:${NC}"
    for commit in "${FIX_COMMITS[@]}"; do
        echo "    🐛 $commit"
    done
    echo ""
fi

if [[ ${#PERF_COMMITS[@]} -gt 0 ]]; then
    echo -e "${CYAN}Performance:${NC}"
    for commit in "${PERF_COMMITS[@]}"; do
        echo "    ⚡ $commit"
    done
    echo ""
fi

# =============================================================================
# Determine Version Bump
# =============================================================================

header "🎯 Version Calculation"

# Check if we have any release-worthy commits
if [[ $RELEASE_WORTHY -eq 0 ]]; then
    if [[ "$FORCE_RELEASE" == true ]]; then
        warn "No release-worthy commits, but --force specified"
        BUMP_TYPE="patch"
    else
        echo -e "${YELLOW}No release-worthy commits found.${NC}"
        echo ""
        echo "Only these commit types trigger releases:"
        echo "  • major                    → MAJOR"
        echo "  • minor                    → MINOR"
        echo "  • feat! / BREAKING CHANGE  → MAJOR"
        echo "  • feat                     → MINOR"
        echo "  • fix / perf               → PATCH"
        echo ""
        echo "Your commits are: docs, chores, tests, refactoring, ci, etc."
        echo "These will be included in the next release."
        echo ""
        info "Use --force to create a patch release anyway."
        exit 0
    fi
else
    # Determine bump type based on highest priority commit
    # Priority: major > minor > breaking > feat > fix/perf
    if [[ $MAJOR_COUNT -gt 0 ]]; then
        BUMP_TYPE="major"
        info "Found $MAJOR_COUNT major commit(s) → MAJOR bump"
    elif [[ $MINOR_COUNT -gt 0 ]]; then
        BUMP_TYPE="minor"
        info "Found $MINOR_COUNT minor commit(s) → MINOR bump"
    elif [[ $BREAKING_COUNT -gt 0 ]]; then
        BUMP_TYPE="major"
        info "Found $BREAKING_COUNT breaking change(s) → MAJOR bump"
    elif [[ $FEAT_COUNT -gt 0 ]]; then
        BUMP_TYPE="minor"
        info "Found $FEAT_COUNT feature(s) → MINOR bump"
    elif [[ $FIX_COUNT -gt 0 || $PERF_COUNT -gt 0 ]]; then
        BUMP_TYPE="patch"
        info "Found $((FIX_COUNT + PERF_COUNT)) fix(es)/perf → PATCH bump"
    fi
fi

# Calculate new version
case $BUMP_TYPE in
    major)
        if [[ $MAJOR -eq 0 && $MAJOR_COUNT -eq 0 ]]; then
            # Pre-1.0: breaking changes bump minor (but explicit major commits always bump major)
            NEW_VERSION="$MAJOR.$((MINOR + 1)).0"
            warn "Pre-1.0: Breaking changes bump MINOR (0.$MINOR.x → 0.$((MINOR + 1)).0)"
        else
            NEW_VERSION="$((MAJOR + 1)).0.0"
        fi
        ;;
    minor)
        NEW_VERSION="$MAJOR.$((MINOR + 1)).0"
        ;;
    patch)
        NEW_VERSION="$MAJOR.$MINOR.$((PATCH + 1))"
        ;;
esac

echo ""
echo -e "  Version: ${BOLD}$CURRENT_VERSION${NC} → ${BOLD}${GREEN}$NEW_VERSION${NC}"
echo ""

# =============================================================================
# Dry Run Exit
# =============================================================================

if [[ "$DRY_RUN" == true ]]; then
    header "🏃 Dry Run Complete"
    echo ""
    echo -e "Would create release ${GREEN}v$NEW_VERSION${NC} with:"
    echo ""
    echo "  🧪 Validations:"
    echo "      • Backend tests"
    echo "      • Frontend tests"
    echo "      • Bundle size check (warn >500KB, fail >750KB gzipped)"
    echo ""
    echo "  📋 Updated files:"
    echo "      • VERSION"
    echo "      • CHANGELOG.md"
    [[ -d "backend" ]] && echo "      • backend/internal/version/version.go"
    [[ -f "frontend/package.json" ]] && echo "      • frontend/package.json"
    echo ""
    echo "  📦 Git operations:"
    echo "      • Commit: chore(release): v$NEW_VERSION"
    echo "      • Tag: v$NEW_VERSION"
    echo "      • Push to origin/$CURRENT_BRANCH"
    echo ""
    echo "  🚀 GitHub Release created automatically"
    echo ""
    info "Run without --dry-run to execute."
    exit 0
fi

# =============================================================================
# Confirmation
# =============================================================================

if [[ "$SKIP_CONFIRM" != true ]]; then
    echo ""
    read -p "$(echo -e ${YELLOW}?${NC} Create release v$NEW_VERSION? [y/N] )" -n 1 -r
    echo
    [[ ! $REPLY =~ ^[Yy]$ ]] && error "Aborted"
fi

# =============================================================================
# Run Tests
# =============================================================================

if [[ "$SKIP_TESTS" != true ]]; then
    header "🧪 Running Tests"

    if [[ -f "backend/go.mod" ]]; then
        info "Running backend tests..."
        if (cd backend && go test ./... -short 2>/dev/null); then
            success "Backend tests passed"
        else
            error "Backend tests failed. Use --skip-tests to bypass."
        fi
    fi

    if [[ -f "frontend/package.json" ]]; then
        info "Running frontend tests..."
        if (cd frontend && npm test -- --watchAll=false --passWithNoTests 2>/dev/null); then
            success "Frontend tests passed"
        else
            warn "Frontend tests failed or skipped"
        fi
    fi
else
    info "Skipping tests (--skip-tests)"
fi

# =============================================================================
# Bundle Size Check
# =============================================================================

if [[ "$SKIP_TESTS" != true && -f "frontend/package.json" ]]; then
    header "📦 Bundle Size Check"

    info "Building frontend..."
    if (cd frontend && npm run build > /dev/null 2>&1); then
        success "Frontend build complete"

        info "Analyzing bundle size..."
        if (cd frontend && npm run analyze:bundle -- --ci); then
            success "Bundle size within limits"
        else
            error "Bundle size exceeds limits. Fix bundle size before releasing."
        fi
    else
        error "Frontend build failed. Fix build errors before releasing."
    fi
fi

# =============================================================================
# Generate Changelog
# =============================================================================

header "📋 Generating Changelog"

git-cliff --tag "v$NEW_VERSION" -o CHANGELOG.md
success "Generated CHANGELOG.md"

# Preview changelog
echo ""
echo -e "${DIM}─────────────────────────────────────────${NC}"
head -50 CHANGELOG.md | tail -45
echo -e "${DIM}─────────────────────────────────────────${NC}"

# =============================================================================
# Update Version Files
# =============================================================================

header "📝 Updating Version Files"

# Update VERSION file
echo "$NEW_VERSION" > VERSION
success "VERSION → $NEW_VERSION"

# Update backend version
if [[ -d "backend" ]]; then
    mkdir -p backend/internal/version
    cat > backend/internal/version/version.go << EOF
// Code generated by release.sh. DO NOT EDIT.
package version

var (
    Version   = "$NEW_VERSION"
    GitCommit = "$(git rev-parse --short HEAD)"
    BuildTime = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
)
EOF
    success "backend/internal/version/version.go → $NEW_VERSION"
fi

# Update frontend package.json
if [[ -f "frontend/package.json" ]]; then
    if command -v jq &> /dev/null; then
        jq ".version = \"$NEW_VERSION\"" frontend/package.json > tmp.json && mv tmp.json frontend/package.json
    else
        sed -i.bak "s/\"version\": \".*\"/\"version\": \"$NEW_VERSION\"/" frontend/package.json
        rm -f frontend/package.json.bak
    fi
    success "frontend/package.json → $NEW_VERSION"
fi

# =============================================================================
# Git Commit & Tag
# =============================================================================

header "📦 Creating Release"

# Stage files
git add VERSION CHANGELOG.md
git add backend/internal/version/version.go 2>/dev/null || true
git add frontend/package.json 2>/dev/null || true

# Create commit
git commit -m "chore(release): v$NEW_VERSION"
success "Created commit"

# Create tag with release notes
RELEASE_NOTES=$(git-cliff --tag "v$NEW_VERSION" --unreleased --strip all 2>/dev/null || echo "Release v$NEW_VERSION")
git tag -a "v$NEW_VERSION" -m "Release v$NEW_VERSION

$RELEASE_NOTES"
success "Created tag v$NEW_VERSION"

# =============================================================================
# Push
# =============================================================================

header "🚀 Pushing to Remote"

git push origin "$CURRENT_BRANCH"
success "Pushed commits"

git push origin "v$NEW_VERSION"
success "Pushed tag v$NEW_VERSION"

# =============================================================================
# Create GitHub Release
# =============================================================================

if command -v gh &> /dev/null; then
    header "📢 Creating GitHub Release"

    gh release create "v$NEW_VERSION" \
        --title "v$NEW_VERSION" \
        --notes "$RELEASE_NOTES" \
        2>/dev/null && success "GitHub Release created" || warn "GitHub Release will be created by CI"
fi

# =============================================================================
# Done
# =============================================================================

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  ✓ Released ${BOLD}v$NEW_VERSION${NC}${GREEN} successfully!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "  📋 Changelog   $REPO_URL/blob/main/CHANGELOG.md"
echo "  🏷️  Tag         v$NEW_VERSION"
echo "  🔗 Release     $REPO_URL/releases/tag/v$NEW_VERSION"
echo ""
