# =============================================================================
# ServicePro Infrastructure - Resource Group Module Outputs
# =============================================================================

output "name" {
  description = "Resource group name"
  value       = aws_resourcegroups_group.main.name
}

output "arn" {
  description = "Resource group ARN"
  value       = aws_resourcegroups_group.main.arn
}
