# Track content_caching_20261116 Context

- [Implementation Plan](./plan.md)
- [Metadata](./metadata.json)

## Goals
Support HTTP proxy content caching (Jira PDEV-4527, epic PDEV-4277). Add `cache_policy` to `baffinbay_http_proxy` and its data source, add a `baffinbay_http_proxy_content_cache_purge` Terraform action, and stop applies from resetting a cache policy that was enabled in the portal.
