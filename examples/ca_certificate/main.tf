terraform {
  required_providers {
    baffinbay = {
      source = "baffinbay/baffinbay"
    }
  }
}

provider "baffinbay" {}

resource "baffinbay_ca_certificate" "test" {
  certificate = file("${path.module}/../certificate/cert.pem")
}
