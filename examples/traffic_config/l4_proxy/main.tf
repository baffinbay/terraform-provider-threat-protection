terraform {
  required_providers {
    baffinbay = {
      source = "baffinbay/threat-protection"
    }
  }
}

provider "baffinbay" {
  # Configuration is read from BAFFINBAY_CLIENT_ID, BAFFINBAY_CLIENT_SECRET,
  # and BAFFINBAY_TENANT_ID environment variables by default.
}

resource "baffinbay_l4_proxy" "l4_example" {
  name = "terraform-l4-example"

  frontend = {
    connection_type = "PLAINTEXT"
    port            = 8080
    ipv4            = "203.0.113.2"
  }

  backend = {
    hosts = [
      {
        address = "example.com"
        port    = 80
      },
    ]
    delivery_method = "ROUND_ROBIN"
    server_name     = "example.com"
  }
}
