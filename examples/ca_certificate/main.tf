terraform {
  required_providers {
    baffinbay = {
      source = "baffinbay/baffinbay"
    }
  }
}

provider "baffinbay" {
  # Configuration is read from BAFFINBAY_CLIENT_ID, BAFFINBAY_CLIENT_SECRET,
  # and BAFFINBAY_TENANT_ID environment variables by default.
}

resource "baffinbay_ca_certificate" "test" {
  certificate = file("${path.module}/../certificate/cert.pem")
}
