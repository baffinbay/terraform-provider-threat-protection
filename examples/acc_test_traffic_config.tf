terraform {
  required_providers {
    baffinbay = {
      source = "baffinbay/baffinbay"
    }
  }
}

provider "baffinbay" {
}

resource "baffinbay_traffic_config" "test" {
  type = "l4Proxy"
  name = "TF-Acceptance-Test-L4"

  frontend = {
    connection_type = "PLAINTEXT"
    port            = 8888
    # Using a dummy IP to bypass the "ALLOCATE" validation for now if needed,
    # though usually 'null' or omission handles auto-allocation if supported.
    # However, since the API rejected "ALLOCATE", let's try a valid IPv4.
    ipv4            = "1.1.1.1" 
    hosts = [
      {
        host = "tf-test.baffinbay.com"
      }
    ]
  }

  backend = {
    hosts = [
      {
        address = "1.2.3.4"
        port    = 80
      }
    ]
    delivery_method = "ROUND_ROBIN"
    server_name     = "origin.example.com"
  }
}

output "traffic_config_id" {
  value = baffinbay_traffic_config.test.id
}
