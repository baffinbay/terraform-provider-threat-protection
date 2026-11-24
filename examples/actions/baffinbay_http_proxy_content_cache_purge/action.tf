# Purge on demand (Terraform 1.14 or later):
#   terraform apply -invoke=action.baffinbay_http_proxy_content_cache_purge.manual
action "baffinbay_http_proxy_content_cache_purge" "manual" {
  config {
    traffic_config_id = baffinbay_http_proxy.example.id
  }
}

# Purge automatically after every update of the proxy (Terraform 1.16 or later).
# A triggered action can't reference the resource that triggers it, because
# that is a dependency cycle. Use `caller`, which is the triggering resource.
action "baffinbay_http_proxy_content_cache_purge" "on_update" {
  config {
    traffic_config_id = caller.id
  }
}

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
      actions = [action.baffinbay_http_proxy_content_cache_purge.on_update]
    }
  }
}
