locals {
  role_assignment = [
    { // Kubelet MSI needs pull access to ACR
      name            = "${var.brand}-${var.environment}-${var.project_number}-cdotest-aks-sys-msi-kubelet-pull",
      scope           = module.main.acr_id["platform_qa_acr"],
      definition_name = "AcrPull",
      principal_id    = module.main.user_msi_principal_id[lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-aks-sys-kubelet-msi")]
    },
    { // Control Plane MSI needs MI operator role on the RG
      name            = "${var.brand}-${var.environment}-${var.project_number}-control-plane-msi",
      scope           = module.main.user_msi_identity_id[lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-aks-sys-kubelet-msi")],
      definition_name = "Managed Identity Operator",
      principal_id    = module.main.user_msi_principal_id[lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-control-plane-msi")]
    },
    { // Foo bar MSI needs MI operator role on the RG
      name            = "keyvault-secrets-officer-${var.brand}-${var.environment}-${var.project_number}-foo-bar-msi",
      scope           = module.main.keyvault_id["akv"],
      definition_name = "Key Vault Administrator",
      principal_id    = module.main.user_msi_principal_id[lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-foo-bar-msi")]
    },
    { // Local Dev User needs access to Key Vault
      name            = "keyvault-secrets-officer-${var.brand}-${var.environment}-${var.project_number}-local-dev",
      scope           = module.main.keyvault_id["akv"],
      definition_name = "Key Vault Administrator",
      principal_id    = var.LOCAL_DEV_USER_OBJECT_ID
    },
  ]
  user_assigned_identites = [
    { // Kubelet ID for cdotest
      resource_group_name = var.resource_group_name
      location            = var.primary_location
      name                = lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-aks-sys-kubelet-msi")
    },
    // Control Plane Id for cdotest
    {
      resource_group_name = var.resource_group_name
      location            = var.primary_location
      name                = lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-control-plane-msi")
    },
    // Foo bar Id for cdotest
    {
      resource_group_name = var.resource_group_name
      location            = var.primary_location
      name                = lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-foo-bar-msi")
    },
  ]
}
