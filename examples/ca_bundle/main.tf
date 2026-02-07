terraform {
  required_providers {
    baffinbay = {
      source = "baffinbay/baffinbay"
    }
  }
}

provider "baffinbay" {}

resource "baffinbay_ca_bundle" "test" {
  name        = "terraform-test-ca"
  certificate = file("${path.module}/../certificate/cert.pem")
}
