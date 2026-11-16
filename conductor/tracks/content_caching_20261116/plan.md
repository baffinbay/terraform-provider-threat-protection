# Implementation Plan - HTTP Proxy Content Caching (PDEV-4527)

## Context
Epic PDEV-4277 ("Content Caching (CDN)") added two things to the **public** traffic-mgmt API:

1. `attributes.cachePolicy.strategy` on HTTP proxies. It can be sent on POST, PUT and PATCH and is always returned on GET.
   - Values: `DISABLED` (default) and `BACKEND_RESPONSE_HEADERS`. The backend also accepts the aliases `RESPONSE_HEADERS` and `ENABLED`, which it turns into `BACKEND_RESPONSE_HEADERS`.
2. `POST /api/v2/traffic-mgmt/traffic-configs/{id}/content-cache/purge`
   - No body. Optional `If-Match` revision header.
   - `202` returns the HTTP proxy with `relationships.activeRollout` and a new revision.
   - `400` when the proxy is not deployed or not ready for updates. `400` when the config is not an HTTP proxy. `404` when it is unknown. `412` on a revision mismatch.

Source of truth: `api-docs/traffic-mgmt.yml`. `cachePolicy` is documented by PDEV-4588 (api-docs MR !245, matching `traffic-management` `origin/main` `8c30702d`); the purge endpoint is on master. The local `CachePolicy` schema and purge path are copied verbatim from the PDEV-4588 branch. Re-sync if review changes the MR. The published `docs.baffinbay.com` spec lags behind and is not used.

### Why this is urgent
The provider updates proxies with **PUT**. The backend's PUT path (`HttpProxySpecDto`) turns a missing `cachePolicy` into `DISABLED` and overwrites the stored policy with it. Today every `terraform apply` on a proxy silently turns off caching that was enabled in the portal.

## Scope
- In scope: `cache_policy` on `baffinbay_http_proxy` and its data source, plus a purge **action**.
- Out of scope: `CacheSizeGB`, `CachePurgeRequestedAt` and anything else that only exists in the management API. Terraform only covers the public API.

## Decisions
- Schema: a `cache_policy` single nested attribute with `strategy` inside it, mirroring `bot_protection`. It leaves room for future cache settings.
- Allowed values: `DISABLED` and `BACKEND_RESPONSE_HEADERS` only. The aliases are left out so the plan always matches what the API returns.
- When `cache_policy` is omitted, follow the existing `mergeHTTPProxyOptionalState` pattern:
  - On create, use the default `DISABLED` (matches the portal).
  - On update, keep the value from prior state, which comes from the API. Terraform therefore never wipes a policy set in the portal unless the user configures one.
- The request must always include `cachePolicy`, because PUT treats a missing value as `DISABLED`. If neither the plan nor prior state has a value (for example, an old state used with `-refresh=false`), fetch the current proxy before the PUT and send its value.
- Purge is a Terraform action named `baffinbay_http_proxy_content_cache_purge`, an unlinked action. It needs Terraform 1.14 or later. The framework is v1.19 and plugin-go v0.31, so both support actions.
- The action waits for the purge rollout to finish, using `WaitForTrafficConfigRollout`. Purge changes the revision, but the resource does not store the revision in state, so this causes no drift.

## Phase 1: Client
- [x] Task: Add the cache policy to the wire model (`internal/client/http_proxy.go`)
    - [x] RED: add a test that a request with a policy produces JSON containing `"cachePolicy":{"strategy":"BACKEND_RESPONSE_HEADERS"}`, and that a response with `cachePolicy` decodes it.
    - [x] GREEN: add the type `HTTPProxyCachePolicy{Strategy string \`json:"strategy"\`}` and the field `CachePolicy *HTTPProxyCachePolicy \`json:"cachePolicy,omitempty"\`` on `HTTPProxyAttributes`.
- [x] Task: Add the purge client method
    - [x] RED: test that `PurgeHTTPProxyContentCache(ctx, id)` sends a POST with no body to `/api/v2/traffic-mgmt/traffic-configs/{id}/content-cache/purge`, accepts `202` and returns `ActiveRolloutID()`. Test that `400`, `404` and `412` come back as errors carrying the API problem detail.
    - [x] GREEN: implemented as a dedicated no-body POST that accepts only `202` (`mutateTrafficConfig` always sends a JSON body).
- [x] Task: Add the purge path to `api-spec/traffic-mgmt-2.yml` and `cachePolicy` to the `HttpProxy`, `CreateHttpProxy` and `PatchHttpProxy` schemas. Run `bb-tool spec-check`.
    - Note (2026-11-16): `api-docs` is the source of truth, not the published `docs.baffinbay.com` spec. `cachePolicy` was missing from `api-docs` and is added by PDEV-4588 (MR !245). The local `CachePolicy` schema and purge path are now verbatim copies from that branch. The `ip-lists` client warning and the older traffic-config paths missing from the local spec are out of scope (separate PR).

## Phase 2: `baffinbay_http_proxy.cache_policy`
- [x] Task: Add the schema and model (`internal/provider/http_proxy_resource.go`)
    - [x] RED: `TestHTTPProxyCachePolicySchema`, plus `TestHTTPProxyCachePolicyRejectsUnknownStrategy`, which rejects the aliases `ENABLED` and `RESPONSE_HEADERS` and lowercase values.
    - [x] GREEN: `HTTPProxyCachePolicyModel`, `httpProxyCachePolicyAttribute()` (Optional + Computed; `strategy` Optional + Computed with default `DISABLED`, allowing only `DISABLED` and `BACKEND_RESPONSE_HEADERS`), and `defaultHTTPProxyCachePolicy()`.
- [x] Task: Extend plan merging (`mergeHTTPProxyOptionalState`)
    - [x] RED: `TestHTTPProxyCachePolicyPlanMerge` covers these cases:
        - Create without the block, and with an empty block, gives `DISABLED`.
        - Update without the block keeps the prior value.
        - An explicit value wins.
        - Update with neither a config nor a prior value stays unresolved.
    - [x] GREEN: unlike the other optional blocks, an update never falls back to the default here.
- [x] Task: Map in both directions (`internal/provider/http_proxy_mapping.go`)
    - [x] RED: `TestHTTPProxyCachePolicyRequestMapping` covers the create and update payloads. `TestHTTPProxyCachePolicyResponseMapping` covers reading the policy back, and keeping the prior value when the response lacks `cachePolicy`.
    - [x] GREEN: `mapHTTPProxyCachePolicyToAPI` and response mapping in `mapHTTPProxyResponseToModel`.
- [x] Task: Keep the portal-set policy when state has no value
    - [x] RED: `TestHTTPProxyCachePolicyResolvedFromRemoteWhenStateHasNone` and `...ResolvesToDisabledWhenRemoteOmitsIt`.
    - [x] GREEN: `resolveHTTPProxyCachePolicy` is called from `ModifyPlan` for updates, so the plan shows the real value, and again in `Update` as a safety net.
- [x] Task: Drift and lifecycle tests using `newHTTPProxyStateServer`
    - [x] The mock server now treats a missing `cachePolicy` on POST/PUT as `DISABLED`, like the backend, and has `setRemoteCachePolicy` to simulate portal changes.
    - [x] `TestHTTPProxyCachePolicyLifecycle` steps through:
        - create, which sends `DISABLED`;
        - enabling caching;
        - import with `ImportStateVerify`;
        - a portal change, which shows drift;
        - an apply that restores the policy;
        - disabling caching again.
    - [x] `TestHTTPProxyCachePolicyPortalValueSurvivesUnmanagedApply` is the regression test. Caching enabled in the portal gives an empty plan when the config doesn't manage it, and a later unrelated apply sends `BACKEND_RESPONSE_HEADERS`.
    - [x] The full fixture config sets `BACKEND_RESPONSE_HEADERS`. `TestHTTPProxyResourceImportThenUpdatePreservesRemoteFields` now also checks that it survives an update that omits the block.
    - [x] Mutation check: removing `cachePolicy` from the payload, or resetting it to `DISABLED` on update, makes the tests fail.
- [x] Task: Data source
    - [x] `TestSingularDataSourcesReadByID` checks that `cache_policy.strategy` is filled in on `data.baffinbay_http_proxy`.
- Verification: `go test ./...`, `go vet` and `golangci-lint` are clean. Coverage went from 78.2% to 78.4% for `internal/provider` and from 20.1% to 21.9% for `internal/client`; master was already below the 80% target.

## Phase 3: `baffinbay_http_proxy_content_cache_purge` action
- [ ] Task: Scaffold the action (`internal/provider/http_proxy_content_cache_purge_action.go`)
    - [ ] RED: test that the provider lists the action through `ProviderWithActions.Actions`, and test the schema: `traffic_config_id` is required and must be a UUID.
    - [ ] GREEN: implement `action.Action` and `action.ActionWithConfigure` using the shared `configure.go` client wiring. Register it in `provider.go`.
- [ ] Task: Invoke logic
    - [ ] RED: unit tests that call `Invoke` against a mock server:
        - The happy path polls the rollout until it is `COMPLETED`.
        - A `FAILED` rollout produces an error diagnostic.
        - A `400` (not deployed) produces a clear message: "content cache can only be purged for a deployed HTTP proxy that is ready for updates".
        - `404`.
        - A non-HTTP-proxy config.
    - [ ] GREEN: call `PurgeHTTPProxyContentCache`, then `WaitForTrafficConfigRollout`. Send progress messages through `resp.SendProgress` (purge requested, then rollout completed).
- [ ] Task: Optional `If-Match` support. Not exposed in v1; revisit only if requested.
- [ ] Task: End-to-end test through Terraform (checked 2026-11-16, see note below)
    - [ ] Add a `tfresource.UnitTest` lifecycle test using `newHTTPProxyStateServer`. The config has an `action "baffinbay_http_proxy_content_cache_purge"` block and an `action_trigger { events = [after_update] }` on the proxy.
        - Step 1 (create): assert no purge request.
        - Step 2 (update): assert exactly one POST to `.../content-cache/purge` with the right ID, and that the rollout was polled. Use `PostApplyFunc` or server counters.
    - [ ] Add an error test: the mock server returns `400` for purge, and the update step uses `ExpectError` matching the message.
    - Note: plugin-testing v1.16 has no step mode for `terraform apply -invoke`. But test steps run a real `terraform apply`, so actions fired by `action_trigger` do run. A throwaway spike (framework v1.19, plugin-testing v1.16, Terraform 1.14.9) showed no invoke on create, exactly one invoke with the resolved ID on update, and that an `Invoke` error diagnostic fails the step with the expected message. The `-invoke` CLI path is covered by the live check in Phase 5.
- [ ] Task: Pin the Terraform CLI in CI
    - [ ] `.github/workflows/build.yml` runs `go test ./...` without installing Terraform, so the tests use whatever binary the runner has. Action tests need Terraform 1.14 or later. Add `hashicorp/setup-terraform` with `terraform_version: v1.17.2` (matching `release.yml`) and `terraform_wrapper: false` before the unit tests.

## Phase 4: Docs & Examples
- [ ] Task: Update `examples/resources/baffinbay_http_proxy/resource.tf` with a `cache_policy` block.
- [ ] Task: Add `examples/actions/baffinbay_http_proxy_content_cache_purge/action.tf`, showing a manual run with `-invoke` and an `action_trigger` on `after_update`. Note that Terraform 1.14 or later is required.
- [ ] Task: Run `go generate ./...` (tfplugindocs v0.25, which supports actions) and check `docs/resources/http_proxy.md`, `docs/data-sources/http_proxy.md` and `docs/actions/http_proxy_content_cache_purge.md`.
- [ ] Task: Outside this repo, add the `cachePolicy` schema to the HTTP proxy schemas in `api-docs/traffic-mgmt.yml`.

## Phase 5: Verification
- [ ] Task: Run `go test ./...`, `golangci-lint run` and `govulncheck ./...`. Coverage must stay above 80% for the touched packages.
- [ ] Task: Live check against QA/preprod:
    - Create a proxy with `BACKEND_RESPONSE_HEADERS`.
    - Toggle the policy in the portal and confirm the plan shows drift.
    - Apply without a `cache_policy` block and confirm the portal value is kept.
    - Run `terraform apply -invoke=action.baffinbay_http_proxy_content_cache_purge.x` and confirm the rollout completes.
- [ ] Task: Add a changelog entry covering the new attribute, the new action, and the fix for portal-enabled caching being reset on apply.
