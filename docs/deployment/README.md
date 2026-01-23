# Deployment

> **Coming Soon**: Deployment documentation will be added once hosting infrastructure is finalized.

## Current Status

The hosting and deployment infrastructure is under evaluation. Options being considered include:

- Cloud providers (AWS, GCP, Azure)
- Container orchestration
- Managed services vs self-hosted

## Local Development

For local development, see [Getting Started](../getting-started/quick-start.md).

## CI/CD

GitHub Actions are used for continuous integration:

- Run tests on push
- Lint and format checks
- Build validation

See `.github/workflows/` for current CI configuration.
