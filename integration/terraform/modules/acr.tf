module "acr" {
  source  = ""
  version = "4.0.0"

  for_each = var.acr

  name_override           = lookup(each.value, "name_override", null)
  environment             = lookup(each.value, "environment", var.environment)
  costcenter              = lookup(each.value, "costcenter", var.costcenter)
  project_name            = lookup(each.value, "project_name", var.project_name)
  location                = lookup(each.value, "location", var.location)
  resource_group_name     = lookup(each.value, "resource_group_name", var.resource_group_name)
  subnet_id               = lookup(each.value, "subnet_id", null)
  georeplications         = lookup(each.value, "georeplications", [])
  zone_redundancy_enabled = lookup(each.value, "zone_redundancy_enabled", false)
  admin_enabled           = lookup(each.value, "admin_enabled", null)
  tags            = { deployed_for = "Operator Integration Testing - caas-certificate-distribution-operator ${trimspace(file("../../../VERSION"))}" }
}
