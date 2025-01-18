module "keyvault" {
  source  = ""
  version = "4.6.7"

  for_each                 = var.keyvaults
  brand                    = lookup(each.value, "brand", var.brand)
  environment              = lookup(each.value, "environment", var.environment)
  location                 = lookup(each.value, "location", var.location)
  name_override            = lookup(each.value, "name_override", null)
  project_name             = lookup(each.value, "project_name", var.project_name)
  resource_group_name      = lookup(each.value, "resource_group_name", var.resource_group_name)
  subnet_id                = lookup(each.value, "subnet_id", null)
  bypass                   = try(each.value.network_acls["bypass"], "None")
  default_action           = try(each.value.network_acls["default_action"], "Deny")
  disable_private_endpoint = lookup(each.value, "disable_private_endpoint", false)
  custom_policy            = lookup(each.value, "custom_policy", [])
  purge_protection_enabled = lookup(each.value, "purge_protection_enabled", false)
  diagnostics              = lookup(each.value, "diagnostics", null)
  tags            = { deployed_for = "Operator Integration Testing - caas-certificate-distribution-operator ${trimspace(file("../../../VERSION"))}" }

}
