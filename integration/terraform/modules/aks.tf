module "aks" {
  source  = ""
  version = "4.0.1"

  for_each                            = var.aks
  brand                               = lookup(each.value, "brand", var.brand)
  project_name                        = lookup(each.value, "project_name", var.project_name)
  costcenter                          = lookup(each.value, "costcenter", var.costcenter)
  environment                         = lookup(each.value, "environment", var.environment)
  location                            = lookup(each.value, "location", var.location)
  resource_group_name                 = lookup(each.value, "resource_group_name", var.resource_group_name)
  name_override                       = lookup(each.value, "name_override", null)
  subnet_id                           = each.value["subnet_id"]
  dns_prefix                          = each.value["dns_prefix"]
  network_profile                     = lookup(each.value, "network_profile", null)
  public_ip                           = lookup(each.value, "public_ip", null)
  default_node_pool                   = lookup(each.value, "default_node_pool", null)
  additional_node_pools               = lookup(each.value, "additional_node_pools", {})
  ingress_application_gateway         = lookup(each.value, "ingress_application_gateway", {})
  kubernetes_version                  = lookup(each.value, "kubernetes_version", null)
  private_cluster_enabled             = lookup(each.value, "private_cluster_enabled", false)
  private_cluster_public_fqdn_enabled = lookup(each.value, "private_cluster_public_fqdn_enabled", true)
  sku_tier                            = lookup(each.value, "sku_tier", "Free")
  auto_scaler_profile                 = lookup(each.value, "auto_scaler_profile", null)
  automatic_upgrade_channel           = lookup(each.value, "automatic_upgrade_channel", "none")
  identity_type                       = lookup(each.value, "identity_type", "SystemAssigned")
  microsoft_defender_enabled          = lookup(each.value, "microsoft_defender_enabled", false)
  key_vault_secrets_provider          = lookup(each.value, "key_vault_secrets_provider", {})
  workload_identity_enabled           = lookup(each.value, "workload_identity_enabled", false)
  oidc_issuer_enabled                 = lookup(each.value, "oidc_issuer_enabled", false)
  user_assigned_identity              = lookup(each.value, "user_assigned_identity", null)
  kubelet_user_assigned_identity      = lookup(each.value, "kubelet_user_assigned_identity", null)
  keyvault                            = lookup(each.value, "keyvault", {})
  storage_profile                     = lookup(each.value, "storage_profile", {})
  log_analytics_workspace_id          = lookup(each.value, "log_analytics_workspace_id", null)
  oms_agent_enabled                   = lookup(each.value, "oms_agent_enabled", false)
  maintenance_window                  = lookup(each.value, "maintenance_window", {})
  service_mesh_profile                = lookup(each.value, "service_mesh_profile", {})
  flux                                = lookup(each.value, "flux", {})
  enable_prometheus                   = lookup(each.value, "enable_prometheus", false)
  tags            = { deployed_for = "Operator Integration Testing - caas-certificate-distribution-operator ${trimspace(file("../../../VERSION"))}" }
}
