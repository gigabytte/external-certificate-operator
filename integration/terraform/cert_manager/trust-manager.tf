resource "helm_release" "trust-manager" {
  name       = "trust-manager"
  namespace  = module.main.namespace_name["trust-manager"]
  repository = "https://charts.jetstack.io"
  chart      = "trust-manager"
  version    = "v0.14.0"

  create_namespace = false

  set {
    name  = "secretTargets.enabled"
    value = "true"
  }

  set_list {
    name  = "secretTargets.authorizedSecrets"
    value = ["ctc-corpissuingca"]
  }

  depends_on = [ helm_release.cert_manager ]

}
