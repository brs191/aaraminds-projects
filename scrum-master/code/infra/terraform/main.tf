# STUB — Terraform skeleton for the AaraMinds fixed stack (Azure-primary, AzureRM
# RBAC mode). Fill in during P0/P1. This compiles structurally but provisions
# nothing until resources are added.
#
# Target topology (per scrum-master/design/Architecture.md):
#   - Azure Container Apps: jira-mcp, teams-adapter, orchestrator
#   - Azure Database for PostgreSQL (Flexible Server)
#   - Azure Key Vault (OAuth client secret, Teams webhook) via managed identity

terraform {
  required_version = ">= 1.7"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.110"
    }
  }
  # backend "azurerm" {}  # configure remote state before first apply
}

provider "azurerm" {
  features {}
  # Auth via GitHub Actions OIDC (see ../../.github/workflows/ci.yml). Do not commit creds.
}

resource "azurerm_resource_group" "this" {
  name     = "rg-${var.project}-${var.environment}"
  location = var.location
  tags     = var.tags
}

# TODO(P0): azurerm_postgresql_flexible_server
# TODO(P0): azurerm_key_vault + access via managed identity
# TODO(P1): azurerm_container_app_environment + azurerm_container_app x3
