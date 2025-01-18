module "k8s-core" {
  source  = ""
  version = "0.1.0"

  namespace = var.namespace
  service-account = var.service_accounts
}
