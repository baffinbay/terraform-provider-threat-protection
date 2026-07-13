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

resource "baffinbay_certificate" "http_cert" {
  type        = "pem"
  certificate = file("${path.module}/../../certificate/cert.pem")
  key         = file("${path.module}/../../certificate/key.pem")
}

resource "baffinbay_http_proxy" "http_example" {
  name = "terraform-http-example"

  frontend = {
    connection_type = "SECURE"
    port            = 443
    ipv4            = "203.0.113.1" # Example IP address
    redirect_http   = true
    hosts = [
      {
        host           = "tf-test.example.com"
        certificate_id = baffinbay_certificate.http_cert.id
        tls_config     = "INTERMEDIATE"
      }
    ]
    hsts = {
      enabled            = true
      max_age            = 31536000
      include_subdomains = true
      preload            = true
    }
  }

  backend = {
    hosts = [
      {
        address = "origin.example.com"
        port    = 443
      }
    ]
    delivery_method = "LEAST_CONNECTIONS"
    server_name     = "origin.example.com"
  }

  protocol_settings = {
    version = "HTTP1.1"
  }
}
