terraform {
  required_providers {
    baffinbay = {
      source = "baffinbay/baffinbay"
    }
  }
}

provider "baffinbay" {}

resource "baffinbay_certificate" "http_cert" {
  type        = "pem"
  certificate = file("${path.module}/../../certificate/cert.pem")
  key         = file("${path.module}/../../certificate/key.pem")
}

resource "baffinbay_traffic_config" "http_example" {
  type = "httpProxy"
  name = "terraform-http-example"
  
  frontend = {
    connection_type = "PLAINTEXT"
    port            = 80
    ipv4            = "203.0.113.1" # Example IP address
    hosts = [
      {
        host           = "tf-test.example.com"
        certificate_id = baffinbay_certificate.http_cert.id
      }
    ]
  }
  
  backend = {
    hosts = [
      {
        address = "origin.example.com"
        port    = 80
      }
    ]
    delivery_method = "ROUND_ROBIN"
    server_name     = "origin.example.com"
  }

  protocol_settings = {
    version = "HTTP1.1"
  }
}