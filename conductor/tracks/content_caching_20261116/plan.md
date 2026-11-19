# Implementation Plan - HTTP Proxy Content Caching (PDEV-4527)

## Context
Epic PDEV-4277 ("Content Caching (CDN)") added two things to the **public** traffic-mgmt API:

1. `attributes.cachePolicy.strategy` on HTTP proxies. It can be sent on POST, PUT and PATCH and is always returned on GET.
   - Values: `DISABLED` (default) and `BACKEND_RESPONSE_HEADERS`.
2. `POST /api/v2/traffic-mgmt/traffic-configs/{id}/content-cache/purge`
   - No body. Optional `If-Match` revision header.
   - `202` returns the HTTP proxy with `relationships.activeRollout` and a new revision.
   - `400` when the proxy is not deployed or not ready for updates. `400` when the config is not an HTTP proxy. `404` when it is unknown. `412` on a revision mismatch.

Source of truth: the internal API documentation (`cachePolicy` added in PDEV-4588). The local `CachePolicy` schema and purge path in `api-spec/traffic-mgmt-2.yml` are copied from it.

### Why this is urgent
The provider updates proxies with **PUT**, which replaces the whole proxy: every optional block that is left out (rate limiting, custom pages, data protection, `cachePolicy`, ...) is reset to its default. Provider versions without `cache_policy` never send it, so every `terraform apply` on a proxy turns off caching that was enabled in the portal. Users on those versions keep hitting this until they upgrade.

## Scope
- In scope: `cache_policy` on `baffinbay_http_proxy` and its data source, plus a purge **action**.
- Out of scope: settings that are not part of the public API. Terraform only covers the public API.

## Decisions
- Schema: a `cache_policy` single nested attribute with `strategy` inside it, mirroring `bot_protection`. It leaves room for future cache settings.
- Allowed values: `DISABLED` and `BACKEND_RESPONSE_HEADERS`, the two documented values, so the plan always matches what the API returns.
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
    - Note (2026-11-16): `cachePolicy` was missing from the API documentation and was added by PDEV-4588. The local `CachePolicy` schema and purge path are copies of it. The `ip-lists` client warning and the older traffic-config paths missing from the local spec are out of scope (separate PR).

## Phase 2: `baffinbay_http_proxy.cache_policy`
- [x] Task: Add the schema and model (`internal/provider/http_proxy_resource.go`)
    - [x] RED: `TestHTTPProxyCachePolicySchema`, plus `TestHTTPProxyCachePolicyRejectsUnknownStrategy`, which rejects undocumented and lowercase values.
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
- [x] Task: Scaffold the action (`internal/provider/http_proxy_content_cache_purge_action.go`)
    - [x] RED: `TestProviderActionsIncludeContentCachePurge`, `TestContentCachePurgeActionSchema` (only `traffic_config_id`, required and validated), and `TestContentCachePurgeActionRejectsInvalidID`, which runs through Terraform.
    - [x] GREEN: the action implements `action.ActionWithConfigure` through a new shared `configuredAction` in `configure.go`. The provider implements `ProviderWithActions`, sets `resp.ActionData`, and lists the action in `Actions()`.
- [x] Task: Invoke logic
    - [x] RED: these tests call `Invoke` directly against a mock server (`newPurgeServer`):
        - `TestContentCachePurgeInvokeWaitsForRollout`: one purge, polling `IN_PROGRESS` then `COMPLETED`, with progress messages.
        - `...WithoutRollout`: no polling when there's no active rollout.
        - `...Errors`: a `FAILED` rollout, a 400 (not deployed or wrong type, with the API detail kept in the message), a 404, and a 500.
        - `...WithoutConfiguredClient` and `ConfigureRejectsWrongProviderData`.
    - [x] GREEN: `PurgeHTTPProxyContentCache`, then `WaitForTrafficConfigRollout`, with `resp.SendProgress` before and after.
    - Note: the backend answers 400 both when the proxy isn't deployed and when the ID isn't an HTTP proxy, so a single message covers both. There's no type pre-check, to avoid an extra GET.
- [ ] Task: Optional `If-Match` support. Not exposed in v1; revisit only if requested.
- [x] Task: End-to-end test through Terraform
    - [x] `newHTTPProxyStateServer` now serves the purge endpoint (`purgeStatus`, `purgeCount`) and a `purge-rollout` (`purgeRolloutReads`).
    - [x] `TestContentCachePurgeActionTriggerLifecycle`: `action_trigger { events = [after_update] }`. No purge on create; exactly one purge on update, with the rollout polled.
    - [x] `TestContentCachePurgeActionTriggerError`: a 400 from purge fails the update step with the "deployed HTTP proxy that is ready for updates" message.
    - [x] Mutation check: a no-op `Invoke` makes both tests fail.
    - Note: plugin-testing v1.16 has no step mode for `terraform apply -invoke`; that path is covered by the live check in Phase 5.
- [x] Task: Pin the Terraform CLI in CI
    - [x] `.github/workflows/build.yml` now runs `hashicorp/setup-terraform` with the same SHA pin as `release.yml` (v4.0.1), `terraform_version: v1.17.2` and `terraform_wrapper: false`, before the unit tests. actionlint and zizmor (pedantic) report nothing.
- Verification: `go test ./...`, `go vet`, `gofmt` and `golangci-lint` are clean. Coverage is 78.8% for `internal/provider` (78.4% after Phase 2) and 21.9% for `internal/client`.

## Phase 4: Docs & Examples
- [x] Task: Update `examples/resources/baffinbay_http_proxy/resource.tf` with a `cache_policy` block.
- [x] Task: Add `examples/actions/baffinbay_http_proxy_content_cache_purge/action.tf`, showing a manual run with `-invoke` and an `action_trigger` on `after_update`. Note that Terraform 1.14 or later is required.
    - Both examples pass `terraform fmt -check` and `terraform validate` against a locally built provider (dev override, Terraform 1.14.9).
- [x] Task: Run `go generate ./...` (tfplugindocs v0.25) and check `docs/resources/http_proxy.md`, `docs/data-sources/http_proxy.md` and the new `docs/actions/http_proxy_content_cache_purge.md`.
- [x] Task: Outside this repo, document `cachePolicy` in the API documentation. Done in PDEV-4588; `api-spec/traffic-mgmt-2.yml` matches the merged version.
- [ ] Task: Outside this repo, add `cachePolicy` and the purge endpoint to the public docs portal API reference (separate task and PR).

## Review follow-up: unknown values
- [x] Task: Keep an unknown `cache_policy.strategy` unknown in the plan
    - Before: a strategy computed from another resource was replaced with `DISABLED`, and Terraform rejected the plan ("planned value does not match config value").
    - [x] RED: `TestHTTPProxyCachePolicyUnknownStrategy` (terraform_data, create and update), plus an "unknown strategy stays unknown" case in `TestHTTPProxyCachePolicyPlanMerge`.
    - [x] GREEN: only a null strategy gets the default.
- [ ] Follow-up **PDEV-4598** (related to this ticket): the same problem affects every nested block of `baffinbay_http_proxy`.
    - About 30 places default unknown values during planning.
    - A whole block set to an unknown value fails to decode in `ValidateConfig` and `ModifyPlan`, because blocks are stored as Go pointers.
    - `TestHTTPProxyCachePolicyUnknownObjectIsNotYetSupported` documents the current error for `cache_policy`.

## CI follow-up: action reference cycle on Terraform 1.16 and later
- [x] CI (Terraform 1.17.2) failed both `action_trigger` tests with `Error: action reference cycle involving action.baffinbay_http_proxy_content_cache_purge.purge and baffinbay_http_proxy.test`.
    - Terraform 1.16 and later reject an action whose config references the resource that triggers it. The 1.14.9 spike allowed it.
    - The replacement is the `caller` symbol, added in Terraform 1.16 (hashicorp/terraform#38668). It holds the triggering resource, so the config uses `traffic_config_id = caller.id`.
- [x] The trigger tests use `caller.id` and skip on Terraform below 1.16 (`actionCallerVersionChecks`). The `ExpectError` regex now tolerates Terraform 1.17's line wrapping.
- [x] The action example has a `manual` action (a direct reference, for `-invoke`, Terraform 1.14 or later) and an `on_update` action (`caller.id`, Terraform 1.16 or later). The action docs are regenerated. The live test's purge step uses `caller.id` and checks the Terraform version.
- [x] Verified: `go test ./...` passes with Terraform 1.17.2, matching CI, and with Terraform 1.14.9, where the trigger tests skip. The example passes `terraform validate` on 1.17.2.

## Phase 5: Verification
- [x] Task: Run `go test ./...`, `golangci-lint run` and `govulncheck ./...`.
    - All are clean, along with `go vet`, `gofmt`, `go mod verify`, actionlint, and a `go generate` check showing no docs drift.
    - govulncheck: 0 reachable vulnerabilities. Like master, it reports 2 vulnerabilities in required modules that the code doesn't call.
    - Coverage: `internal/provider` is 78.8% (master 78.2%) and `internal/client` is 21.9% (master 20.1%). Both went up, but `internal/provider` was already below the 80% target on master.
- [ ] Task: Live check against a test environment. **Pending: needs test-environment credentials.**
    - The check is automated as `TestAccHTTPProxyContentCachingLive` (`internal/provider/http_proxy_content_caching_acc_test.go`). Its HCL passes `terraform validate`, and it skips cleanly without credentials. The steps:
        - Create a proxy with `BACKEND_RESPONSE_HEADERS`.
        - Apply without the block; the policy is kept.
        - Change the policy through the API, as the portal would; the plan shows drift.
        - Apply restores the policy.
        - With `BAFFINBAY_ACC_DEPLOYMENT_STATE=DEPLOYED`, an update fires the purge through `action_trigger` and waits for the rollout.
    - Run it:
      ```
      set -a && source .env && set +a
      TF_ACC=1 BAFFINBAY_ACC_FRONTEND_IPV4=<test-ip> BAFFINBAY_ACC_DEPLOYMENT_STATE=DEPLOYED \
        go test ./internal/provider -run TestAccHTTPProxyContentCachingLive -v
      ```
    - Still manual: `terraform apply -invoke=action.baffinbay_http_proxy_content_cache_purge.<name>`, because plugin-testing has no `-invoke` step. Use `examples/actions/baffinbay_http_proxy_content_cache_purge/action.tf`.
- [x] Task: Changelog. The repo has no CHANGELOG file and GoReleaser's changelog is disabled (`changelog.disable: true`), so the release notes go in the PR description and GitHub release instead:
    - **New:** `cache_policy` on `baffinbay_http_proxy` and the `baffinbay_http_proxy` data source. The `strategy` is `DISABLED` (the default) or `BACKEND_RESPONSE_HEADERS`.
    - **New:** the `baffinbay_http_proxy_content_cache_purge` action (Terraform 1.14 or later), which purges the content cache of a deployed HTTP Proxy and waits for the rollout.
    - **Fix:** applying a `baffinbay_http_proxy` no longer resets content caching that was enabled outside Terraform. An omitted `cache_policy` keeps the current value.
    - **Upgrade note:** older provider versions don't know about `cache_policy`, so every apply with them turns off caching enabled in the portal. Upgrade before enabling content caching on proxies managed by Terraform.
    - **CI:** unit tests run on Terraform 1.17.2.
