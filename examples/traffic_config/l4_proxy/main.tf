terraform {
  required_providers {
    baffinbay = {
      source = "baffinbay/baffinbay"
    }
  }
}

provider "baffinbay" {}

resource "baffinbay_traffic_config" "l4_example" {
  type = "l4Proxy"
  name = "terraform-l4-example"
  
  frontend = {
    connection_type = "PLAINTEXT"
    port            = 8080
    ipv4            = "185.195.93.210"
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