# Contributing to ServicePro

## Getting Started

1. Clone the repository.
2. Install the toolchain: Go matching `backend/go.mod`, Node.js 24
   (`frontend/.nvmrc`), Docker, and optionally `golangci-lint`
   (`make install-golangci-lint`).
3. `cp backend/.env.example backend/.env`, then `make setup` (dependencies, git
   hooks, database containers, migrations, seed data).
4. Run `make help` to see every task. There is one Makefile, at the repo root —
   don't add Makefiles in subdirectories.

ServicePro is live at <https://servicepro.sharkfins.xyz> (since v0.9.0,
2026-10-24); production deploys only from `v*` tags (see
[RELEASING.md](RELEASING.md) and [docs/deployment](docs/deployment/README.md)).
Develop and test locally.

## Git hooks

```bash
make install-hooks
```

Symlinks the versioned hooks in `scripts/git-hooks/`:

- **pre-commit** — on staged backend Go: `golangci-lint`, unit tests, build,
  and an OpenAPI drift check when handlers/models/`cmd/main.go` change; on
  staged frontend TS: `eslint` and `tsc --noEmit`.
- **commit-msg** — enforces the commit prefix contract below.

`git commit --no-verify` skips them in an emergency; CI runs the same checks.

## Commit messages

Subjects are the changelog: `scripts/gh-release.sh` builds GitHub Release notes
from them, and `scripts/bump-version.sh` picks the next version from them.

```
<type>[(scope)][!]: <subject>
```

| Type                                                                           | Use for                                                                     | In release notes |
| ------------------------------------------------------------------------------ | --------------------------------------------------------------------------- | ---------------- |
| `feat`                                                                         | a user-visible feature (`feat!:` = breaking)                                | yes              |
| `fix`                                                                          | a user-visible bug fix                                                      | yes              |
| `deps`                                                                         | dependency bumps (Dependabot uses `deps(go)`, `deps(npm)`, `deps(actions)`) | no               |
| `chore`, `docs`, `ci`, `build`, `test`, `style`, `refactor`, `perf`, `release` | everything else                                                             | no               |

Examples:

```
feat(jobs): add recurring job templates
fix(invoices): round tax per line, not per invoice
ci: cache Go modules in the lint job
```

The legacy `minor:` / `major:` prefixes are retired — use `feat:` / `feat!:`.
Full details: [RELEASING.md](RELEASING.md).

## Code style

**Backend (Go)** — `gofmt`/`goimports` (`make backend-lint-fix`),
`golangci-lint` (`make backend-lint`), GORM for data access, UUID primary keys,
contextual log prefixes. See `CLAUDE.md` for project conventions.

**Frontend (React/TypeScript)** — strict TypeScript, never `any`
(`unknown` + `getErrorMessage()` for errors), Zustand for state, the axios
services in `frontend/src/services/`. ESLint with zero warnings
(`make frontend-lint`); Prettier (`make format`).

## Testing

```bash
make test               # backend unit tests (-race) + frontend vitest
make test-integration   # backend integration suite against the compose stack
make test-e2e           # Cypress (app must be running)
make coverage
```

Frontend integration tests for critical workflows live in
`frontend/tests/integration/` — update them when you change those flows.

## Before you push

```bash
make ci-local           # lint, unit tests, builds, dead code, bundle size, OpenAPI drift
```

CI (`.github/workflows/ci.yml`) additionally runs govulncheck, `npm audit`, and
a Docker build of `backend/Dockerfile` with a Trivy scan.

## Database changes

Migrations are numbered golang-migrate pairs in `backend/migrations/`:

```bash
make migrate-create NAME=add_customer_tags   # creates 00000N_add_customer_tags.{up,down}.sql
make migrate
```

Once production exists, migrations are **additive only** (expand/contract for
renames and drops) — see RELEASING.md. Dev-only data goes in
`backend/seeds/dev_seed.sql`.

## API changes

Update the swag annotations on the handler, then:

```bash
make docs
git add backend/docs frontend/src/types/api.generated.ts
```

CI's `openapi-drift` job fails if the committed spec or TypeScript types are
stale.

## Environment variables

Add new backend variables to `backend/.env.example` (documented, placeholder
secrets), `backend/.env`, and `backend/config/config.go`. If production needs
it, also add it to `docs/ops/cloud-run-setup.md` (env var or Secret Manager
secret) and, for secrets, the optional list in `scripts/deploy-cloudrun.sh`.

## Releases

Maintainers cut releases from `master` with `make bump` and a `v*` tag — see
[RELEASING.md](RELEASING.md).

## Security

If you discover a security vulnerability, do **not** open a public issue —
email the maintainer with details and wait for a response before disclosing.
