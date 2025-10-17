# Contributing to ServicePro

Thank you for your interest in contributing to ServicePro! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and collaborative environment for all contributors.

## Getting Started

1. Fork the repository
2. Clone your fork locally
3. Set up the development environment following the README.md
4. Create a new branch for your feature or bugfix

## Development Workflow

### Branch Naming

Use descriptive branch names with prefixes:

- `feature/` - New features (e.g., `feature/user-authentication`)
- `fix/` - Bug fixes (e.g., `fix/login-error`)
- `refactor/` - Code refactoring (e.g., `refactor/api-handlers`)
- `docs/` - Documentation updates (e.g., `docs/api-documentation`)
- `test/` - Test additions/updates (e.g., `test/user-service`)

### Commit Messages

Follow the Conventional Commits specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `ci`: CI/CD changes

Examples:

```
feat(auth): add JWT authentication middleware

fix(api): resolve null pointer error in user handler

docs(readme): update setup instructions

ci(argocd): update staging deployment configuration
```

### Code Style

**Backend (Go):**

- Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Write meaningful comments for exported functions
- Keep functions small and focused

**Frontend (React/TypeScript):**

- Follow the [Airbnb JavaScript Style Guide](https://github.com/airbnb/javascript)
- Use TypeScript for type safety
- Use functional components with hooks
- Keep components small and reusable
- Write meaningful prop types and interfaces

### Testing

**Backend:**

- Write unit tests for all business logic
- Aim for >80% code coverage
- Use table-driven tests where appropriate
- Mock external dependencies

```bash
go test ./... -v
go test ./... -cover
```

**Frontend:**

- Write unit tests for utilities and hooks
- Write component tests for UI components
- Write integration tests for critical flows

```bash
npm test
npm run test:coverage
```

## Pull Request Process

1. **Update your branch** with the latest from master:

   ```bash
   git checkout master
   git pull origin master
   git checkout your-branch
   git rebase master
   ```

2. **Ensure all tests pass**:

   ```bash
   # Backend
   cd backend && go test ./...

   # Frontend
   cd frontend && npm test
   ```

3. **Run linters**:

   ```bash
   # Backend
   golangci-lint run
   go vet ./...

   # Frontend
   npm run lint
   npm run type-check
   ```

4. **Build Docker images** (if applicable):

   ```bash
   # Backend
   cd backend && docker build -t servicepro-backend:test .

   # Frontend
   cd frontend && docker build -t servicepro-frontend:test .
   ```

5. **Create a pull request**:

   - Provide a clear title and description
   - Reference any related issues
   - Include screenshots for UI changes
   - Ensure CI/CD checks pass

6. **Code review**:

   - Address reviewer feedback
   - Keep discussions professional and constructive
   - Push updates to the same branch

7. **Merge**:
   - Squash commits if requested
   - Wait for approval from maintainers
   - Maintainers will merge your PR

## Infrastructure Changes

### Terraform

When making infrastructure changes:

1. Work in the appropriate environment directory
2. Run `terraform fmt` to format code
3. Run `terraform validate` to validate configuration
4. Run `terraform plan` and review changes
5. Include plan output in PR description
6. Never commit `.tfstate` files

```bash
cd infrastructure/terraform/environments/staging
terraform fmt
terraform validate
terraform plan
```

### Helm Charts

When updating Helm charts:

1. Test charts locally using `helm template`
2. Validate against Kubernetes using `helm lint`
3. Update Chart.yaml version following semantic versioning
4. Document changes in chart README

```bash
cd backend/helm
helm lint .
helm template . --values ../../gitops/values/backend/staging.yaml
```

### ArgoCD Applications

When modifying ArgoCD applications:

1. Validate YAML syntax
2. Test against a non-production environment first
3. Ensure sync policies are appropriate
4. Document environment-specific configurations

## GitOps Workflow

### Deployment Flow

**Staging:**

- Auto-deploys on any push to `master` branch
- ArgoCD syncs automatically
- Used for development testing

**PreProd:**

- Deploys on tags matching `preprod-*` pattern
- Auto-sync enabled
- Used for QA and integration testing

**Production:**

- Deploys on tags matching `v*` pattern (e.g., `v1.0.0`)
- **Manual sync required** for safety
- Used for production workloads

### Creating Releases

**For PreProd:**

```bash
git tag -a preprod-2024.01.15 -m "PreProd release 2024.01.15"
git push origin preprod-2024.01.15
```

**For Production:**

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## Database Migrations

When adding database changes:

1. Create a new migration file
2. Write both up and down migrations
3. Test the migration locally
4. Document breaking changes
5. Coordinate with team for production deployment

## API Changes

When modifying the API:

1. Update OpenAPI/Swagger documentation
2. Maintain backward compatibility when possible
3. Version breaking changes
4. Update API client code
5. Update relevant Helm chart configurations

## Documentation

- Update README.md for setup changes
- Update API documentation for endpoint changes
- Add inline comments for complex logic
- Create/update docs/ files for significant features
- Update Helm chart READMEs for configuration changes

## Reporting Issues

When reporting bugs:

1. Use the issue template
2. Provide a clear description
3. Include steps to reproduce
4. Add relevant logs or screenshots
5. Specify your environment (OS, Go version, Node version, K8s version, etc.)

## Feature Requests

When requesting features:

1. Use the feature request template
2. Describe the problem it solves
3. Provide use cases
4. Suggest implementation approaches if possible

## Security

If you discover a security vulnerability:

1. **DO NOT** create a public issue
2. Email security@servicepro.com
3. Include detailed information about the vulnerability
4. Wait for response before disclosing publicly

## Questions?

- Create a discussion in GitHub Discussions
- Check existing issues and documentation
- Contact the maintainers

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (MIT License).

---

Thank you for contributing to ServicePro!
