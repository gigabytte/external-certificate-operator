resource "helm_release" "cert_dist_op_crd" {
  name       = "caas-certificate-distribution-operator-crd"
  namespace  = module.main.namespace_name["cert-dist-op-ns"]
  chart      = "../../../charts/crds"
  create_namespace = false
  values = [
    "${file("./values/cert-dist-op-crds.yaml")}"
  ]

  depends_on = [ kubernetes_manifest.trust_manager_bundle ]
}

resource "helm_release" "cert_dist_op" {
  name       = "caas-certificate-distribution-operator"
  namespace  = module.main.namespace_name["cert-dist-op-ns"]
  chart      = "../../../charts/operator"
  create_namespace = false
  values = [
    "${file("./values/cert-dist-op.yaml")}"
  ]

  depends_on = [ helm_release.cert_dist_op_crd ]
}
