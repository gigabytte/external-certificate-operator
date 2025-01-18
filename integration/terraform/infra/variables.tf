variable "tenant_id" { default = "bd6704ff-1437-477c-9ac9-c30d6f5133c5" }
variable "subscription_id" { default = "adb4e6c2-42ba-452f-b625-d3b29b972c6e" }
variable "brand" { default = "corp" }
variable "costcenter" { default = "00000" }
variable "project_name" {}
variable "project_number" { default = "001" }
variable "environment" {}
variable "primary_location" {}
variable "primary_location_code" {}
variable "resource_group_name" {}
variable "vnet_name" {}
variable "vnet_rg" {}
variable "aks_version" { default = "1.30.6" }
variable "role_assignment" {
  default = []
  type    = list(map(any))
}
variable "log_analytics_id" { default = "/subscriptions/c1db24d3-f1c5-46b0-8e75-69fc8a0ffd2e/resourceGroups/ctc-prod-loganalytics-workspace-cc-rg/providers/Microsoft.OperationalInsights/workspaces/ctc-prod-log-analytics-workspace" }
variable "firewall_ip" { default = "10.20.0.132" }
variable "ARM_CLIENT_ID" { default = "" }
variable "ARM_CLIENT_SECRET" { default = "" }
variable "LOCAL_DEV_USER_OBJECT_ID" {
  description = "The object id of the user that will be granted access to the keyvault for validation"
  type       = string
  default = ""
}
