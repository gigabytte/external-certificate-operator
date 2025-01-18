locals {
  aks = {
    cdotest = {
      name_override = {
        cluster = "${var.brand}-${var.environment}-${var.project_number}-cdotest-${var.primary_location_code}-aks"
        managed_rg = "${var.brand}-${var.environment}-${var.project_number}-cdotest-${var.primary_location_code}-rg"
      }
      costcenter = var.costcenter
      subnet_id  =  data.azurerm_subnet.system-nodepool.id
      dns_prefix = "cdotest"

      network_profile = {
        network_plugin_mode = "overlay"
        network_policy      = "calico"
        outbound_type       = "userDefinedRouting"
        service_cidr        = "172.16.0.0/16"
        dns_service_ip      = "172.16.0.10"
        pod_cidr            = "10.244.0.0/16"
      }

      default_node_pool = {
        temporary_name_for_rotation  = "tempnodes"
        zones                        = [1, 2, 3]
        node_count                   = 2
        auto_scaling_enabled         = false // We want to manually control scale out behavior
        max_count                    = null
        min_count                    = null
        only_critical_addons_enabled = true
        orchestrator_version         = var.aks_version
      }
      additional_node_pools = {
        platform01 = {
          name                 = "platform01"
          node_count           = 2
          auto_scaling_enabled = false // We want to manually control scale out behavior
          max_count            = null
          min_count            = null
          zones                = [1, 2, 3]
          vm_size              = "Standard_D4ds_v5"
          node_os              = "Linux"
          subnet_id            = data.azurerm_subnet.system-nodepool.id
          node_labels = {
            "node.kubernetes.io/reserved-for" = "tools"
          }
          orchestrator_version = var.aks_version
        }
      }
      enable_prometheus          = true
      private_cluster_enabled    = true
      sku_tier                   = "Standard"
      microsoft_defender_enabled = true
      kubernetes_version         = var.aks_version
      automatic_upgrade_channel  = "none"
      keyvault = {
        enabled     = false
        resource_id = null
      }
      key_vault_secrets_provider = {}
      workload_identity_enabled = true
      oidc_issuer_enabled       = true
      identity_type             = "UserAssigned" # Control Plane MSI
      user_assigned_identity = {
        ids          = [module.main.user_msi_identity_id[lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-control-plane-msi")]]
        principal_id = module.main.user_msi_principal_id[lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-control-plane-msi")]
      }
      // Depends on roles asigned to MSI before deploying AKS
      kubelet_user_assigned_identity = {
        id           = module.main.user_msi_identity_id[lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-aks-sys-kubelet-msi")]
        client_id    = module.main.user_msi_client_id[lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-aks-sys-kubelet-msi")]
        principal_id = module.main.user_msi_principal_id[lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-aks-sys-kubelet-msi")]
      }
      storage_profile = {}
      log_analytics_workspace_id = var.log_analytics_id
      service_mesh_profile       = {}
      flux = {}
    }
  }
}
