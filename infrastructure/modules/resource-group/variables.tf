# =============================================================================
# ServicePro Infrastructure - Resource Group Module Variables
# =============================================================================

variable "name_prefix" {
  description = "Prefix for resource names"
  type        = string
}

variable "project_name" {
  description = "Name of the project (used in tag filter)"
  type        = string
}

variable "environment" {
  description = "Deployment environment (used in tag filter)"
  type        = string
}

variable "tags" {
  description = "Tags to apply to the resource group itself"
  type        = map(string)
  default     = {}
}
