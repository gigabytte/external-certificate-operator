module "authorization" {
  source  = ""
  version = "1.1.0"

  role_definition        = var.role_definition
  role_assignment        = var.role_assignment
  user_assigned_identity = var.user_assigned_identites
  fededated_credentials  = var.fededated_credentials

}
