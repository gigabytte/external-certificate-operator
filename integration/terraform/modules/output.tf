output "vault_id" {
  value = { for k, v in module.keyvault : k => v.vault_id }
}

output "aks_cluster_id" {
  value = { for k, v in module.aks : k => v.aks_cluster_id }
}

output "aks_host" {
  value = { for k, v in module.aks : k => v.host }
}

output "aks_client_certificate" {
  value = { for k, v in module.aks : k => v.client_certificate }
}

output "aks_cluster_ca_certificate" {
  value = { for k, v in module.aks : k => v.cluster_ca_certificate }
}

output "aks_client_key" {
  value = { for k, v in module.aks : k => v.client_key }
}

output "aks_oidc_issuer_url" {
  value = { for k, v in module.aks : k => v.oidc_issuer_url }
}

output "user_msi_principal_id" {
  value = module.authorization.user_assigned_identity_principal_id
}

output "user_msi_client_id" {
  value = module.authorization.user_assigned_identity_client_id
}

output "user_msi_identity_id" {
  value = module.authorization.user_assigned_identity_id
}
output "federated_identity_credentials_id" {
  value = module.authorization.federated_identity_credentials_id
}
output "acr_id" {
  value = { for k, v in module.acr : k => v.acr_id }
}

output "keyvault_id" {
  value = { for k, v in module.keyvault : k => v.vault_id }
}

output "key_vault_secrets_provider_identity_object_id" {
  value = { for k, v in module.aks : k => v.key_vault_secrets_provider_identity_object_id }
}

output "namespace_name" {
  value       = module.k8s-core.namespace_name

}
