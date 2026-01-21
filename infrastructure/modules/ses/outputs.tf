output "domain_identity_arn" {
  description = "SES domain identity ARN"
  value       = var.domain != "" ? aws_ses_domain_identity.main[0].arn : null
}

output "configuration_set_name" {
  description = "SES configuration set name"
  value       = var.domain != "" ? aws_ses_configuration_set.main[0].name : null
}
