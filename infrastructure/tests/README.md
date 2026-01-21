# Terraform Infrastructure Testing

This directory contains tests and validation procedures for the ServicePro infrastructure.

## Test Categories

1. **Static Analysis** - Terraform validation, formatting, security scanning
2. **Unit Tests** - Module-level testing with Terratest
3. **Integration Tests** - Full infrastructure deployment tests
4. **Compliance Tests** - Policy and compliance validation

## Prerequisites

```bash
# Install required tools
brew install terraform tflint tfsec checkov terratest

# Or using Go for Terratest
go install github.com/gruntwork-io/terratest/modules/terraform@latest
```

## Running Tests

### Quick Validation (CI/CD)

```bash
# From infrastructure/terraform directory
make validate
```

### Full Test Suite

```bash
make test
```

### Individual Tests

```bash
# Static analysis only
make lint

# Security scan only
make security-scan

# Integration tests (requires AWS credentials)
make integration-test
```

## Test Files

- `validate.sh` - Static validation script
- `terratest/` - Go-based infrastructure tests
- `policies/` - OPA/Sentinel policy files
- `fixtures/` - Test fixtures and mock data
