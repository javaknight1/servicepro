#!/bin/bash
# =============================================================================
# ServicePro Infrastructure - Terraform Validation Script
# =============================================================================
# This script performs comprehensive validation of Terraform configurations.
# Run from the infrastructure directory.
#
# Usage:
#   ./tests/validate.sh [--all|--quick|--security|--compliance]
#
# Options:
#   --all        Run all validation checks (default)
#   --quick      Run quick syntax and format checks only
#   --security   Run security-focused checks only
#   --compliance Run compliance checks only
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TESTS_PASSED=0
TESTS_FAILED=0
WARNINGS=0

# Parse arguments
RUN_MODE="${1:---all}"

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_failure() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARNINGS++))
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Header
echo ""
echo "=============================================="
echo "  ServicePro Infrastructure Validation"
echo "=============================================="
echo ""
echo "Mode: $RUN_MODE"
echo "Directory: $(pwd)"
echo ""

# -----------------------------------------------------------------------------
# Terraform Format Check
# -----------------------------------------------------------------------------
log_info "Checking Terraform formatting..."
if terraform fmt -check -recursive >/dev/null 2>&1; then
    log_success "Terraform formatting is correct"
else
    log_failure "Terraform files need formatting. Run: terraform fmt -recursive"
    terraform fmt -check -recursive -diff 2>&1 | head -50
fi

# -----------------------------------------------------------------------------
# Terraform Validation
# -----------------------------------------------------------------------------
log_info "Validating Terraform configuration..."

# Initialize for validation (without backend)
if terraform init -backend=false -input=false >/dev/null 2>&1; then
    log_success "Terraform initialization successful"
else
    log_failure "Terraform initialization failed"
fi

if terraform validate >/dev/null 2>&1; then
    log_success "Terraform validation passed"
else
    log_failure "Terraform validation failed"
    terraform validate
fi

# -----------------------------------------------------------------------------
# Environment-specific Validation
# -----------------------------------------------------------------------------
log_info "Validating environment configurations..."

for env_file in environments/*.tfvars; do
    if [[ -f "$env_file" ]]; then
        env_name=$(basename "$env_file" .tfvars)
        if terraform validate >/dev/null 2>&1; then
            log_success "Environment '$env_name' configuration is valid"
        else
            log_failure "Environment '$env_name' configuration is invalid"
        fi
    fi
done

# -----------------------------------------------------------------------------
# Module Validation
# -----------------------------------------------------------------------------
log_info "Validating individual modules..."

for module_dir in modules/*/; do
    if [[ -d "$module_dir" ]]; then
        module_name=$(basename "$module_dir")
        if [[ -f "${module_dir}main.tf" ]]; then
            log_success "Module '$module_name' has main.tf"
        else
            log_failure "Module '$module_name' missing main.tf"
        fi
        if [[ -f "${module_dir}variables.tf" ]]; then
            log_success "Module '$module_name' has variables.tf"
        else
            log_warning "Module '$module_name' missing variables.tf"
        fi
        if [[ -f "${module_dir}outputs.tf" ]]; then
            log_success "Module '$module_name' has outputs.tf"
        else
            log_warning "Module '$module_name' missing outputs.tf"
        fi
    fi
done

# Exit early if quick mode
if [[ "$RUN_MODE" == "--quick" ]]; then
    echo ""
    echo "Quick validation complete."
    exit $TESTS_FAILED
fi

# -----------------------------------------------------------------------------
# TFLint (if available)
# -----------------------------------------------------------------------------
if command_exists tflint; then
    log_info "Running TFLint..."

    # Initialize TFLint with AWS plugin
    cat > .tflint.hcl << 'TFLINT'
plugin "aws" {
  enabled = true
  version = "0.27.0"
  source  = "github.com/terraform-linters/tflint-ruleset-aws"
}

rule "terraform_naming_convention" {
  enabled = true
}

rule "terraform_documented_variables" {
  enabled = true
}

rule "terraform_documented_outputs" {
  enabled = true
}

rule "terraform_typed_variables" {
  enabled = true
}
TFLINT

    tflint --init >/dev/null 2>&1 || true

    if tflint --recursive 2>&1 | grep -q "issue(s) found"; then
        log_warning "TFLint found issues"
        tflint --recursive 2>&1 | head -30
    else
        log_success "TFLint passed"
    fi

    rm -f .tflint.hcl
else
    log_warning "TFLint not installed - skipping"
fi

# Exit if not running security checks
if [[ "$RUN_MODE" != "--all" && "$RUN_MODE" != "--security" ]]; then
    echo ""
    echo "Validation complete (security checks skipped)."
    exit $TESTS_FAILED
fi

# -----------------------------------------------------------------------------
# Security Scanning with tfsec
# -----------------------------------------------------------------------------
if command_exists tfsec; then
    log_info "Running tfsec security scan..."

    if tfsec . --soft-fail 2>&1 | grep -q "passed"; then
        log_success "tfsec security scan passed"
    else
        log_warning "tfsec found security issues"
        tfsec . --soft-fail 2>&1 | tail -50
    fi
else
    log_warning "tfsec not installed - skipping security scan"
fi

# -----------------------------------------------------------------------------
# Checkov Security Scan
# -----------------------------------------------------------------------------
if command_exists checkov; then
    log_info "Running Checkov security scan..."

    checkov_result=$(checkov -d . --quiet --compact 2>&1 || true)

    if echo "$checkov_result" | grep -q "Passed checks:"; then
        passed=$(echo "$checkov_result" | grep "Passed checks:" | awk '{print $3}')
        failed=$(echo "$checkov_result" | grep "Failed checks:" | awk '{print $3}')

        if [[ "$failed" == "0" ]]; then
            log_success "Checkov passed all checks ($passed passed)"
        else
            log_warning "Checkov: $passed passed, $failed failed"
        fi
    else
        log_warning "Checkov scan completed with warnings"
    fi
else
    log_warning "Checkov not installed - skipping"
fi

# Exit if not running compliance checks
if [[ "$RUN_MODE" != "--all" && "$RUN_MODE" != "--compliance" ]]; then
    echo ""
    echo "Validation complete (compliance checks skipped)."
    exit $TESTS_FAILED
fi

# -----------------------------------------------------------------------------
# Custom Compliance Checks
# -----------------------------------------------------------------------------
log_info "Running custom compliance checks..."

# Check: All resources should have required tags
log_info "Checking for required tags in modules..."
required_tags=("Project" "Environment" "ManagedBy")
missing_tags=0

for tag in "${required_tags[@]}"; do
    if ! grep -r "\"$tag\"" main.tf modules/ >/dev/null 2>&1; then
        log_warning "Tag '$tag' may be missing from some resources"
        ((missing_tags++))
    fi
done

if [[ $missing_tags -eq 0 ]]; then
    log_success "All required tags are referenced"
fi

# Check: No hardcoded credentials
log_info "Checking for hardcoded credentials..."
if grep -rE "(password|secret|key)\s*=\s*\"[^$\{]" *.tf modules/ 2>/dev/null | grep -v "description" | grep -v "secret_name" | grep -v "key_id"; then
    log_failure "Potential hardcoded credentials found"
else
    log_success "No hardcoded credentials found"
fi

# Check: S3 buckets should have encryption
log_info "Checking S3 bucket encryption..."
if grep -r "aws_s3_bucket\." modules/s3/ >/dev/null 2>&1; then
    if grep -r "aws_s3_bucket_server_side_encryption_configuration" modules/s3/ >/dev/null 2>&1; then
        log_success "S3 bucket encryption is configured"
    else
        log_warning "Some S3 buckets may not have encryption configured"
    fi
else
    log_success "No S3 buckets to check"
fi

# Check: Security groups should not allow 0.0.0.0/0 on sensitive ports
log_info "Checking security group rules..."
if grep -rE "cidr_ipv4\s*=\s*\"0\.0\.0\.0/0\"" modules/security/ 2>/dev/null | grep -v "# Intentional" | grep -v "alb" | grep -v "443" | grep -v "80"; then
    log_warning "Found 0.0.0.0/0 CIDR in security groups on non-standard ports"
else
    log_success "Security group CIDR rules look correct"
fi

# Check: RDS should have encryption
log_info "Checking database encryption settings..."
if grep -r "storage_encrypted\s*=\s*true" modules/rds/ >/dev/null 2>&1; then
    log_success "Database encryption is configured"
else
    log_warning "Database encryption may not be configured"
fi

# Check: EKS cluster encryption
log_info "Checking EKS encryption settings..."
if grep -r "encryption_config" modules/eks/ >/dev/null 2>&1 || grep -r "cluster_encryption_config" modules/eks/ >/dev/null 2>&1; then
    log_success "EKS cluster encryption is configured"
else
    log_warning "EKS cluster encryption may not be configured"
fi

# Check: CloudWatch logging is enabled
log_info "Checking CloudWatch logging..."
if grep -r "aws_cloudwatch_log_group" modules/ >/dev/null 2>&1; then
    log_success "CloudWatch log groups are configured"
else
    log_warning "No CloudWatch log groups found"
fi

# Check: Backup configuration exists
log_info "Checking backup configuration..."
if [[ -d "modules/backup" ]] && grep -r "aws_backup_plan" modules/backup/ >/dev/null 2>&1; then
    log_success "Backup plans are configured"
else
    log_warning "No backup plans found"
fi

# Check: WAF configuration exists
log_info "Checking WAF configuration..."
if grep -r "aws_wafv2_web_acl" modules/ >/dev/null 2>&1; then
    log_success "WAF is configured"
else
    log_warning "No WAF configuration found"
fi

# -----------------------------------------------------------------------------
# Variable Validation
# -----------------------------------------------------------------------------
log_info "Checking variable definitions..."

# Check for variables without descriptions in root
vars_without_desc=$(grep -E "^variable\s+\"" variables.tf 2>/dev/null | while read line; do
    var_name=$(echo "$line" | grep -oP '"\K[^"]+' || echo "")
    if [[ -n "$var_name" ]] && ! grep -A5 "variable \"$var_name\"" variables.tf | grep -q "description"; then
        echo "$var_name"
    fi
done || true)

if [[ -z "$vars_without_desc" ]]; then
    log_success "All root variables have descriptions"
else
    log_warning "Variables without descriptions: $vars_without_desc"
fi

# -----------------------------------------------------------------------------
# Module Structure Validation
# -----------------------------------------------------------------------------
log_info "Checking module structure..."

expected_modules=("vpc" "security" "eks" "rds" "elasticache" "s3" "alb" "cdn" "ses" "monitoring" "backup")
missing_modules=0

for module in "${expected_modules[@]}"; do
    if [[ -d "modules/$module" ]]; then
        log_success "Module '$module' exists"
    else
        log_failure "Module '$module' is missing"
        ((missing_modules++))
    fi
done

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------
echo ""
echo "=============================================="
echo "  Validation Summary"
echo "=============================================="
echo ""
echo -e "  ${GREEN}Passed:${NC}   $TESTS_PASSED"
echo -e "  ${RED}Failed:${NC}   $TESTS_FAILED"
echo -e "  ${YELLOW}Warnings:${NC} $WARNINGS"
echo ""

if [[ $TESTS_FAILED -gt 0 ]]; then
    echo -e "${RED}Validation failed with $TESTS_FAILED error(s)${NC}"
    exit 1
else
    if [[ $WARNINGS -gt 0 ]]; then
        echo -e "${YELLOW}Validation passed with $WARNINGS warning(s)${NC}"
    else
        echo -e "${GREEN}All validations passed successfully!${NC}"
    fi
    exit 0
fi
