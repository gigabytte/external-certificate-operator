resource "kubernetes_manifest" "self_signed_issuer" {
  manifest = {
    "apiVersion" = "cert-manager.io/v1"
    "kind"       = "Issuer"
    "metadata" = {
      "name"      = "self-signed-issuer"
      "namespace" = local.namespace_names["cert-manager"]
    }
    "spec" = {
      "selfSigned" = {}
    }
  }
}

resource "kubernetes_manifest" "ca_certificate" {
  manifest = {
    "apiVersion" = "cert-manager.io/v1"
    "kind"       = "Certificate"
    "metadata" = {
      "name"      = "ca-secret"
      "namespace" = local.namespace_names["cert-manager"]
    }
    "spec" = {
      "isCA"       = true
      "commonName" = "ca-secret"
      "subject" = {
        "organizations" = ["ACME Inc."]
        "organizationalUnits" = ["Widgets"]
      }
      "secretName" = "ca-secret"
      "privateKey" = {
        "algorithm" = "ECDSA"
        "size"      = 256
      }
      "issuerRef" = {
        "name"  = kubernetes_manifest.self_signed_issuer.manifest.metadata.name
        "kind"  = "Issuer"
        "group" = "cert-manager.io"
      }
    }
  }
}

resource "kubernetes_manifest" "ca_issuer" {
  manifest = {
    "apiVersion" = "cert-manager.io/v1"
    "kind"       = "ClusterIssuer"
    "metadata" = {
      "name"      = "ca-issuer"
    }
    "spec" = {
      "ca" = {
        "secretName" = kubernetes_manifest.ca_certificate.manifest.spec.secretName
      }
    }
  }
}
