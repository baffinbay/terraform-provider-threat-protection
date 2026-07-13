resource "baffinbay_http_proxy" "example" {
  name = "terraform-http-example"

  frontend = {
    connection_type = "PLAINTEXT"
    ipv4            = "203.0.113.1"
    port            = 80
    hosts = [
      {
        host = "www.example.com"
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
  }

  protocol_settings = {
    version = "HTTP1.1"
  }
}
