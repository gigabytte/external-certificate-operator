module "main" {
  source              = "../modules"

  brand               = ""
  project_name        = ""
  environment         = ""
  location            = ""
  location_code       = ""
  resource_group_name = ""

  # Namespace
    namespace = local.namespaces
    service_accounts = local.service_accounts

}
