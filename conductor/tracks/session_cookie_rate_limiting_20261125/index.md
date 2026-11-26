# Track session_cookie_rate_limiting_20261125 Context

- [Implementation Plan](./plan.md)
- [Metadata](./metadata.json)

## Goals
Support session cookie rate limiting (Jira PDEV-4575, idea MDP-203). Add `strategy`, `overall`, `per_path` and `ip_rate_limit_multiplier` to `baffinbay_http_proxy.rate_limiting` and its data source, following the corrected API documentation (PDEV-4651). Keep `by_source_ip`/`by_source_ip_and_url` working as deprecated aliases.
