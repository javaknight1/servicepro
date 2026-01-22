# =============================================================================
# ServicePro Infrastructure - Resource Group Module
# =============================================================================
# Creates an AWS Resource Group that automatically includes all resources
# tagged with the project's common tags.
# =============================================================================

resource "aws_resourcegroups_group" "main" {
  name        = "${var.name_prefix}-resources"
  description = "All ${var.project_name} ${var.environment} resources managed by Terraform"

  resource_query {
    query = jsonencode({
      ResourceTypeFilters = ["AWS::AllSupported"]
      TagFilters = [
        {
          Key    = "Project"
          Values = [var.project_name]
        },
        {
          Key    = "Environment"
          Values = [var.environment]
        },
        {
          Key    = "ManagedBy"
          Values = ["terraform"]
        }
      ]
    })
  }

  tags = var.tags
}
