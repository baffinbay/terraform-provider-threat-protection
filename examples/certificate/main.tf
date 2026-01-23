terraform {
  required_providers {
    baffinbay = {
      source = "baffinbay/baffinbay"
    }
  }
}

provider "baffinbay" {
  # Configuration read from environment variables
}

resource "baffinbay_certificate" "test_pem" {
  type = "pem" # API expects "importPemCertificate", mapped by provider
  
  certificate = file("${path.module}/cert.pem")
  key         = file("${path.module}/key.pem")
}
