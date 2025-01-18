locals {
  vnet_id = "/subscriptions/${var.subscription_id}/resourceGroups/${var.vnet_rg}/providers/Microsoft.Network/virtualNetworks/${var.vnet_name}"
  acr = {
    platform_qa_acr = {
      name_override                 = "${var.brand}${var.environment}${var.project_number}${var.primary_location_code}acr"
      location                      = var.primary_location
      admin_enabled                 = true
      public_network_access_enabled = false
      subnet_id                     = data.azurerm_subnet.private-endpoints.id
      georeplications               = []
      zone_redundancy_enabled       = true
    }
  }
}
