// Federated Credential to integrate Flux source controller with Azure AD.
locals {
  fededated_credentials = [
    {
      name                = "foo-bar-integration-testing-akv"
      resource_group_name = var.resource_group_name
      audience            = ["api://AzureADTokenExchange"]
      issuer              = module.main.aks_oidc_issuer_url["cdotest"]
      parent_id           = module.main.user_msi_identity_id[lower("${var.brand}-${var.environment}-${var.project_number}-cdotest-foo-bar-msi")]
      subject             = "system:serviceaccount:foo-bar-ns:foo-bar-sa"
    },
  ]
}
