provider "kubernetes" {
  host                   = data.azurerm_kubernetes_cluster.cdotest.kube_config.0.host
  cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.cdotest.kube_config.0.cluster_ca_certificate)
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "kubelogin"
    args = [
      "get-token",
      "--login",
      "azurecli",
      "--server-id",
      "6dae42f8-4368-4678-94ff-3960e28e3630"
    ]
  }
}

provider "helm" {
  kubernetes {
    host                   = data.azurerm_kubernetes_cluster.cdotest.kube_config.0.host
    cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.cdotest.kube_config.0.cluster_ca_certificate)
    exec {
        api_version = "client.authentication.k8s.io/v1beta1"
        command     = "kubelogin"
        args = [
        "get-token",
        "--login",
        "azurecli",
        "--server-id",
        "6dae42f8-4368-4678-94ff-3960e28e3630"
        ]
    }
  }
}

#AKS Cluster Data sources
data "azurerm_kubernetes_cluster" "cdotest" {
  name                = "cdotest-qa-001-cdotest-cc-aks"
  resource_group_name = var.resource_group_name
}

provider "azurerm" {
  subscription_id = var.subscription_id
  tenant_id       = var.tenant_id
  features {}
}
