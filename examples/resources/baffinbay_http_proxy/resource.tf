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

  # Cache responses as the backend's response headers allow.
  # Omit the block to keep the current policy (DISABLED on create).
  cache_policy = {
    strategy = "BACKEND_RESPONSE_HEADERS"
  }

  # Rate limit per session cookie instead of per source IP. Requests without a
  # recognized session cookie get the limit multiplied by ip_rate_limit_multiplier.
  # Omit the block to keep the current settings (BY_IP with defaults on create).
  rate_limiting = {
    strategy                 = "BY_SESSION_COOKIE"
    overall                  = { enforcement = "BLOCK", rate = { value = 10, unit = "r/s" }, burst = 100 }
    per_path                 = { enforcement = "BLOCK", rate = { value = 10, unit = "r/s" }, burst = 100 }
    ip_rate_limit_multiplier = 100
  }
}
