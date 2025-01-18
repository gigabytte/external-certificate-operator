locals {
  namespaces = [
    {
      name = "foo-bar-ns"
      labels = {
        "ctc-ca-bundle/inject" = "enabled"
      }
    },
    {
      name = "cert-dist-op-ns"
      labels = {
        "ctc-ca-bundle/inject" = "enabled"
      }
    }
  ]
  service_accounts = [
    {
      name           = "foo-bar-sa"
      namespace_name = module.main.namespace_name["foo-bar-ns"]
      annotations = {
        "azure.workload.identity/client-id" = "ffd55ed6-18ef-4ef9-a6d2-4410061c19ef",
        "azure.workload.identity/tenant-id" = "bd6704ff-1437-477c-9ac9-c30d6f5133c5"
      }
    },
    {
      name           = "cert-dist-op-sa"
      namespace_name = module.main.namespace_name["cert-dist-op-ns"]
      annotations = {}
    }
  ]
}
