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

resource "baffinbay_traffic_config" "l4_example" {
  type          = "l4Proxy"
  name          = "terraform-l4-example"
  frontend_port = 8080
  # frontend_ipv4/ipv6 are optional, can be omitted to auto-allocate or specified if known
  
  backend = {
    hosts = [
      {
        address = "example.com"
        port    = 80
      }
    ]
    delivery_method = "ROUND_ROBIN"
    server_name     = "example.com"
  }
}
