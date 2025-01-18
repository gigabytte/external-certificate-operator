resource "helm_release" "cert_manager" {
  name       = "cert-manager"
  namespace  = module.main.namespace_name["cert-manager"]
  repository = "https://charts.jetstack.io"
  chart      = "cert-manager"
  version    = "v1.16.2"

  create_namespace = true

  set {
    name  = "installCRDs"
    value = "true"
  }

  set {
    name = "featureGates"
    value = "AdditionalCertificateOutputFormats=true"
  }

}
