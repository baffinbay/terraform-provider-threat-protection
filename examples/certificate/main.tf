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

resource "baffinbay_certificate" "test_pem" {
  type = "pem" # API expects "importPemCertificate", mapped by provider

  certificate = file("${path.module}/cert.pem")
  key         = file("${path.module}/key.pem")
}
