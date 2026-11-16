# Requires Terraform 1.14 or later.
action "baffinbay_http_proxy_content_cache_purge" "example" {
  config {
    traffic_config_id = baffinbay_http_proxy.example.id
  }
}

# Run the purge on demand:
#   terraform apply -invoke=action.baffinbay_http_proxy_content_cache_purge.example

# Or purge automatically every time the proxy is updated.
resource "baffinbay_http_proxy" "example" {
  name             = "terraform-http-example"
  deployment_state = "DEPLOYED"

  frontend = {
    connection_type = "PLAINTEXT"
    ipv4            = "203.0.113.1"
    port            = 80
    hosts           = [{ host = "www.example.com" }]
  }

  backend = {
    hosts = [{ address = "origin.example.com", port = 80 }]
  }

  protocol_settings = {
    version = "HTTP1.1"
  }

  cache_policy = {
    strategy = "BACKEND_RESPONSE_HEADERS"
  }

  lifecycle {
    action_trigger {
      events  = [after_update]
      actions = [action.baffinbay_http_proxy_content_cache_purge.example]
    }
  }
}
