variable "tenant_id" { default = "bd6704ff-1437-477c-9ac9-c30d6f5133c5" }
variable "subscription_id" { default = "" }
variable "brand" { default = "corp" }
variable "costcenter" { default = "00000" }
variable "project_name" {}
variable "project_number" { default = "001" }
variable "environment" {}
variable "location" {}
variable "location_code" {}
variable "resource_group_name" {}
variable "role_definition" { default = [] }
variable "role_assignment" {
  default = []
  type    = list(map(any))
}
variable "fededated_credentials" { default = [] }
variable "user_assigned_identity" { default = [] }
variable "aks" { default = {} }
variable "acr" { default = {} }
variable "user_assigned_identites" { default = [] }
variable "keyvaults" { default = {} }
variable "namespace" { default = [] }
variable "service_accounts" { default = [] }
