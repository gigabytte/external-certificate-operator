locals {
  keyvaults = {
    akv = {
      subnet_id                = data.azurerm_subnet.private-endpoints.id
      name_override            = "${var.brand}${var.environment}${var.project_number}${var.primary_location_code}-kv"
      purge_protection_enabled = false
      network_acls = {
        bypass         = "AzureServices"
        default_action = "Deny"
      }
    }
  }
}
