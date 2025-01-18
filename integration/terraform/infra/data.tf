data "azurerm_subnet" "private-endpoints" {
  name                 = "corp-qa-corenetwork-cc-vnet-01-private-endpoints-snet"
  virtual_network_name = "corp-qa-corenetwork-cc-vnet-01"
  resource_group_name  = "corp-qa-001-corenetwork-cc-rg"
}

data "azurerm_subnet" "system-nodepool" {
  name                 = "corp-qa-corenetwork-cc-vnet-01-system-nodepool-snet"
  virtual_network_name = "corp-qa-corenetwork-cc-vnet-01"
  resource_group_name  = "corp-qa-001-corenetwork-cc-rg"
}
