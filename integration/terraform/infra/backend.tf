terraform {
  backend "remote" {
    hostname     = "terraform.corp.ad.ctc"
    organization = "azure"

    workspaces {
      name = "certificate-distribution-operator-integration-test"
    }
  }
}
