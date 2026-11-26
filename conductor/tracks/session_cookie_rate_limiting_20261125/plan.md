# Implementation Plan - Session Cookie Rate Limiting (PDEV-4575)

## Context
MDP-203 added session cookie rate limiting to the global rate limiting of HTTP proxies. The backend, portal and nginx template were released on 2026-10-08. Terraform support is the remaining part.

The public API (`traffic-management` main @ 6ce8e4d) uses this shape for `attributes.rateLimiting`:

```json
{
  "strategy": "BY_IP" | "BY_SESSION_COOKIE",
  "overall":  { "enforcement": "BLOCK|DISABLED", "rate": { "value": 1-10000, "unit": "r/s|r/m" }, "burst": 1-10000 },
  "perPath":  { "...": "same as overall" },
  "ipRateLimitMultiplier": 1-100000,
  "exclusions": ["..."]
}
```

- `ipRateLimitMultiplier` is required for `BY_SESSION_COOKIE` and is not returned for `BY_IP`.
- Deprecated shape: `bySrcIp`/`bySrcIpAndUrl`. A request without `strategy` is read as `BY_IP` with them as `overall`/`perPath`. Responses always include them. With `BY_SESSION_COOKIE` they hold the disabled IP-based defaults (`DISABLED`, 10 r/s, burst 10).
- Traffic rule `setRateLimit` is unchanged. It still only has `bySrcIp`/`bySrcIpAndUrl`.

Source of truth: the API documentation, corrected in PDEV-4651 (api-docs MR !249). The docs had `IP_BASED` instead of `BY_IP`, and listed session cookie fields on traffic rules that the backend does not implement.

## Decisions
- Follow the corrected docs. `rate_limiting` gets `strategy`, `overall`, `per_path` and `ip_rate_limit_multiplier`.
- The provider is published (v1.x), so `by_source_ip`/`by_source_ip_and_url` stay as deprecated aliases instead of being removed. This mirrors the API.
    - A block that only sets the aliases is planned as `BY_IP` with them as `overall`/`per_path`. Existing configurations keep working without changes.
    - Setting the aliases together with any of the new attributes is a validation error.
    - The aliases are computed from the new attributes, so the plan matches what the API returns. They mirror `overall`/`per_path` for `BY_IP` and hold the disabled defaults for `BY_SESSION_COOKIE`.
- Defaults match the portal (`web.portal` `trafficConfigsDefaults`): `BY_IP`, `BLOCK`, 10 r/s, burst 100, multiplier 100.
- `ip_rate_limit_multiplier` set with `BY_IP` (or with no strategy) is a validation error, because the API ignores it and the plan would never converge.
- Requests send only the new shape. Responses without `strategy` (the deprecated shape only) are read as `BY_IP`.
- Omitting `rate_limiting` keeps the current settings on update, like the other optional blocks.
- Known limitation: if `strategy` is unknown at plan time, the deprecated fields are planned as for `BY_IP`.

## Phase 1: Spec and client
- [x] Sync `RateLimit`, `PatchRateLimit`, `RateLimitRule`, `RateLimiting`, `PatchRateLimitRule` and `PatchRateLimiting` into `api-spec/traffic-mgmt-2.yml` from api-docs MR !249. Point global `rateLimiting` at `RateLimiting`/`PatchRateLimiting`.
- [x] `client.HTTPProxyRateLimiting`: add `Strategy`, `Overall`, `PerPath` and `IPRateLimitMultiplier`, and make the deprecated fields optional pointers.

## Phase 2: `baffinbay_http_proxy.rate_limiting`
- [x] RED: `internal/provider/http_proxy_rate_limiting_test.go`
    - `TestHTTPProxyRateLimitingSchema`: new attributes are present, aliases are deprecated, and traffic rule `rate_limit` is unchanged.
    - `TestHTTPProxyRateLimitingPlanMerge`: defaults, session cookie defaults, explicit values, BY_IP mirroring, alias-only config, update without the block, and prior state from before `strategy` existed.
    - `TestHTTPProxyRateLimitingRequestMapping` / `ResponseMapping`: wire format in both directions, including deprecated-only responses.
    - `TestHTTPProxyRateLimitingValidation`: unknown strategy (`IP_BASED`), multiplier with `BY_IP`, multiplier range, and aliases mixed with the new attributes.
    - `TestHTTPProxyRateLimitingLifecycle`: create with session cookie, switch to BY_IP through the aliases, then remove the block. The mock server now normalizes rate limiting like the backend.
- [x] GREEN: schema, `defaultHTTPProxyRateLimitingFields`, validation and mapping.
- [x] Update the existing default and full-payload assertions to the new wire shape.

## Phase 3: Docs and verification
- [x] Example in `examples/resources/baffinbay_http_proxy/resource.tf`. Regenerate docs with `go generate ./...`.
- [x] `TestAccHTTPProxyRateLimitingLive`: aliases, session cookie, update without the block, default multiplier, and back to BY_IP.
- [x] `go test ./...`, `golangci-lint run` and gofmt pass.
- [ ] Run `TestAccHTTPProxyRateLimitingLive` against a live tenant.
