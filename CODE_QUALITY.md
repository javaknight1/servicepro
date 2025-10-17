# Code Quality Tools Configuration

This document describes the code quality tools configured for this project.

## Overview

The project uses the following tools to maintain code quality:

- **Prettier**: Code formatting
- **ESLint**: JavaScript/TypeScript linting (Frontend)
- **golangci-lint**: Go linting (Backend)
- **Pre-commit hooks**: Automated validation before commits

## Quick Start

```bash
# Install all dependencies and set up hooks
make install-deps

# Or just set up pre-commit hooks
make setup-hooks
```

## Prettier Configuration

Prettier is configured at the root level (`.prettierrc`) with the following settings:

- Tab width: 2 spaces
- Single quotes
- Trailing commas: ES5
- Semicolons: enabled
- Line width: 80 characters

### Usage

```bash
# Format all files
make format

# Format specific directory
prettier --write "frontend/src/**/*.ts"

# Check formatting without changes
prettier --check .
```

## ESLint Configuration (Frontend)

Located at `frontend/.eslintrc.json` with:

- **Style Guide**: Airbnb
- **TypeScript Support**: Full type-aware linting
- **React Rules**: Including hooks validation
- **Prettier Integration**: No conflicts with formatting

### Rules Highlights

- React hooks validation (rules-of-hooks, exhaustive-deps)
- TypeScript strict checking
- Import/export best practices
- Unused variable detection

### Usage

```bash
# Run ESLint
make frontend-lint

# Run ESLint with auto-fix
make frontend-lint-fix

# Or directly
cd frontend && npx eslint . --ext .js,.jsx,.ts,.tsx --fix
```

### Required Dependencies

Add to `frontend/package.json`:

```json
{
  "devDependencies": {
    "@typescript-eslint/eslint-plugin": "^6.19.0",
    "@typescript-eslint/parser": "^6.19.0",
    "eslint": "^8.56.0",
    "eslint-config-airbnb": "^19.0.4",
    "eslint-config-airbnb-typescript": "^17.1.0",
    "eslint-config-prettier": "^9.1.0",
    "eslint-plugin-import": "^2.29.1",
    "eslint-plugin-jsx-a11y": "^6.8.0",
    "eslint-plugin-react": "^7.33.2",
    "eslint-plugin-react-hooks": "^4.6.0",
    "prettier": "^3.1.0",
    "typescript": "^5.3.3"
  }
}
```

## golangci-lint Configuration (Backend)

Located at `backend/.golangci.yml` with:

### Enabled Linters

- **Formatting**: gofmt, goimports, gci
- **Code Quality**: govet, staticcheck, unused, gosimple
- **Security**: gosec
- **Performance**: prealloc, bodyclose
- **Best Practices**: gocritic, revive, errcheck

### Usage

```bash
# Run golangci-lint
make backend-lint

# Format and fix issues
make backend-lint-fix

# Or directly
cd backend && golangci-lint run --config .golangci.yml
```

### Installation

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Or use Go
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Pre-commit Hooks

Configuration in `.pre-commit-config.yaml` includes:

### Automatic Checks (on every commit)

1. **File checks**: trailing whitespace, EOF, YAML/JSON validation
2. **Security**: detect private keys, check large files
3. **Prettier**: format code automatically
4. **ESLint**: lint frontend with auto-fix
5. **golangci-lint**: lint backend code
6. **Go formatting**: gofmt and goimports

### Manual Checks (optional)

- Frontend tests: `pre-commit run --hook-stage manual frontend-tests`
- Backend tests: `pre-commit run --hook-stage manual backend-tests`

### Setup

```bash
# Install pre-commit
pip install pre-commit
# or
brew install pre-commit

# Install hooks
./scripts/setup-pre-commit.sh
# or
make setup-hooks
```

### Usage

```bash
# Hooks run automatically on git commit

# Run manually on all files
pre-commit run --all-files

# Run with tests included
pre-commit run --all-files --hook-stage manual

# Update hook versions
pre-commit autoupdate

# Skip hooks temporarily (not recommended)
git commit --no-verify
```

## Makefile Commands

All commands are available through the Makefile:

```bash
# View all available commands
make help

# Setup
make install-deps        # Install all dependencies
make setup-hooks         # Setup pre-commit hooks

# Linting
make lint               # Run all linters
make lint-fix           # Run linters with auto-fix
make frontend-lint      # Lint frontend only
make backend-lint       # Lint backend only

# Formatting
make format             # Format all code
make frontend-format    # Format frontend only
make backend-format     # Format backend only

# Testing
make test              # Run all tests
make frontend-test     # Run frontend tests
make backend-test      # Run backend tests

# Building
make frontend-build    # Build frontend
make backend-build     # Build backend
```

## CI/CD Integration

Add these commands to your CI pipeline:

```yaml
# Example GitHub Actions
- name: Run linters
  run: make lint

- name: Check formatting
  run: prettier --check .

- name: Run tests
  run: make test
```

## IDE Integration

### VSCode

Install extensions:

- ESLint
- Prettier - Code formatter
- Go (with gopls)

Add to `.vscode/settings.json`:

```json
{
  "editor.formatOnSave": true,
  "editor.defaultFormatter": "esbenp.prettier-vscode",
  "editor.codeActionsOnSave": {
    "source.fixAll.eslint": true
  },
  "[go]": {
    "editor.defaultFormatter": "golang.go"
  },
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--config=backend/.golangci.yml"]
}
```

### IntelliJ/GoLand

1. Enable Prettier: Settings → Languages & Frameworks → JavaScript → Prettier
2. Enable ESLint: Settings → Languages & Frameworks → JavaScript → Code Quality Tools → ESLint
3. Enable golangci-lint: Settings → Go → Linter → golangci-lint

## Troubleshooting

### Pre-commit hooks not running

```bash
pre-commit install
```

### ESLint errors in IDE

```bash
cd frontend && npm install
```

### golangci-lint not found

```bash
brew install golangci-lint
# or
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### TypeScript errors in ESLint

Ensure `tsconfig.json` exists in frontend directory and paths are correct in `.eslintrc.json`.
