resource "kubernetes_manifest" "trust_manager_bundle" {
  manifest = {
    apiVersion = "trust.cert-manager.io/v1alpha1"
    kind       = "Bundle"
    metadata = {
      name = "ctc-corpissuingca"  # The bundle name will also be used for the target

    }
    spec = {
      sources = [
        {
          useDefaultCAs = true
        },
        {
          secret = {
            name = "ca-secret"
            key  = "ca.crt"
          }
        }
      ]
      target = {
        secret = {
          key = "root-certs.pem"
        }
        namespaceSelector = {
          matchLabels = {
            "ctc-ca-bundle/inject" = "enabled" // All namespaces requiring the bundle should have this label
          }
        }
      }
    }
  }
}
