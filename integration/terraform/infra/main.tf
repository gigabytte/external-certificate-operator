module "main" {
  source              = "../modules"
  brand               = var.brand
  project_name        = var.project_name
  environment         = var.environment
  location            = var.primary_location
  location_code       = var.primary_location_code
  resource_group_name = var.resource_group_name

  # Authorization
  role_assignment         = local.role_assignment
  user_assigned_identites = local.user_assigned_identites

  # Federated Credentials
  fededated_credentials = local.fededated_credentials

  #AKS
  aks = local.aks

  # KeyVaults
  keyvaults = local.keyvaults

  #ACR
  acr = local.acr
}
