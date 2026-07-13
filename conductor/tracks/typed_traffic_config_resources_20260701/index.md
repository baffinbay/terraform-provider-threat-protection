# Track typed_traffic_config_resources_20260701 Context

- [Implementation Plan](./plan.md)
- [Metadata](./metadata.json)

## Goals
Split the unified `baffinbay_traffic_config` resource into type-specific Terraform resources for HTTP Proxy, L4 Proxy, and Routed DSR traffic configurations while preserving compatibility for existing users.
