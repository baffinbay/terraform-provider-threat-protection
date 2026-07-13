# Implementation Plan - Typed Traffic Config Resources

## Context
The current `baffinbay_traffic_config` resource models three API types behind a required `type` discriminator: `httpProxy`, `l4Proxy`, and `routedDsr`. Their valid configuration differs enough that a single schema exposes attributes that are meaningless for some types, such as HTTP-only WAF and protocol settings beside Routed DSR-only `prefix` and `announced` fields.

This track introduces type-specific resources with narrow schemas and keeps the unified resource available for compatibility.

## Proposed Resources
- `baffinbay_http_proxy`
    - API type: `httpProxy`
    - Owns HTTP proxy attributes such as `frontend`, `backend`, `protocol_settings`, `waf`, and `rate_limiting`.
- `baffinbay_l4_proxy`
    - API type: `l4Proxy`
    - Owns L4 attributes such as `frontend`, `backend`, and L4 protocol defaults.
- `baffinbay_routed_dsr`
    - API type: `routedDsr`
    - Owns Routed DSR attributes shown in `api-spec/traffic-mgmt.yml`: `name`, `prefix`, `announced`, and `deployment.state`.

## Compatibility Position
- Keep `baffinbay_traffic_config` registered in the provider for existing configurations.
- Mark `baffinbay_traffic_config` deprecated as soon as the typed resources are complete and documented.
- Do not remove the unified resource in this track.
- Preserve the existing `baffinbay_traffic_configs` data source unless a later design needs type-specific data sources.

## Decisions
- Use the shorter resource names: `baffinbay_http_proxy`, `baffinbay_l4_proxy`, and `baffinbay_routed_dsr`.
- Deprecate `baffinbay_traffic_config` immediately when the typed resources ship.
- Implement `resource.ResourceWithMoveState` for the typed resources unless implementation risk proves too high during development. This lets users migrate state with Terraform `moved` blocks when changing resource type, instead of destroying/recreating or manually removing and importing state.
- Based on `api-spec/traffic-mgmt.yml`, Routed DSR supports `name`, `prefix`, `announced`, and `deployment.state`. `CreateRoutedDsr` requires `name`, `prefix`, and `deployment`; `announced` is optional with API default `false`.
- Keep the Terraform attribute name `deployment_state` for typed resources and map it to API `deployment.state`; the API object only contributes a state value for the supported traffic config operations.

## Phase 1: Schema Design
- [x] Task: Define the exact user-facing resource names.
    - [x] Sub-task: Use shorter resource names: `baffinbay_http_proxy`, `baffinbay_l4_proxy`, and `baffinbay_routed_dsr`.
    - [x] Sub-task: Confirm naming with product terminology and Terraform provider naming guidelines.
- [x] Task: Map the current unified schema into type-specific schemas.
    - [x] Sub-task: Identify common attributes shared across all three resources, such as `id`, `name`, and `deployment_state`.
    - [x] Sub-task: Identify HTTP Proxy-only attributes and nested blocks.
    - [x] Sub-task: Identify L4 Proxy-only attributes and nested blocks.
    - [x] Sub-task: Identify L4 access-control attributes required by the API: `protocols`, `proxyProtocol`, and `ipBasedAccessControl`; omit deprecated `geoFencing` and `allowedSources`.
    - [x] Sub-task: Identify Routed DSR-only attributes from the API spec: `prefix`, `announced`, and `deployment_state` mapped to API `deployment.state`.
- [x] Task: Decide whether nested object names should stay compatible with the unified resource.
    - [x] Sub-task: Prefer preserving existing block names where they remain semantically correct.
    - [x] Sub-task: Rename only when the current names hide type-specific behavior.

## Phase 2: Internal Refactor
- [ ] Task: Split reusable traffic config logic out of `traffic_config_resource.go`.
    - [ ] Sub-task: Extract shared models and schema builders for common frontend, backend, TLS, deployment, and polling behavior.
    - [ ] Sub-task: Extract request/response mapping helpers by API type.
    - [ ] Sub-task: Reconcile typed HTTP Proxy defaults with portal behavior as described in the HTTP Proxy Contract Reconciliation section; omit deprecated allowed sources and geo fencing.
- [ ] Task: Add typed resource implementations.
    - [ ] Sub-task: Implement metadata names for each typed resource.
    - [ ] Sub-task: Set the API `type` internally instead of exposing a Terraform `type` attribute.
    - [ ] Sub-task: Reuse the existing client create, read, update, delete, and polling methods.
    - [ ] Sub-task: Register the new resources in `BaffinBayProvider.Resources`.
    - [x] Sub-task: Implement and register the first typed resource, `baffinbay_routed_dsr`.
    - [x] Sub-task: Implement and register the second typed resource, `baffinbay_l4_proxy`.
    - [x] Sub-task: Implement complete L4 request/response mapping and defaults for protocols, proxy protocol, and IP based access control.
    - [x] Sub-task: Implement and register the third typed resource, `baffinbay_http_proxy`.

## Phase 3: Import and Migration
- [x] Task: Implement import support for each typed resource.
    - [x] Sub-task: Import by traffic config UUID, matching the existing unified resource import behavior.
    - [x] Sub-task: After import/read, verify the remote API type matches the target Terraform resource type.
    - [x] Sub-task: Return a clear diagnostic if a user imports an HTTP Proxy into the L4 or Routed DSR resource.
    - [x] Sub-task: Implement import and wrong-type read/import diagnostics for `baffinbay_routed_dsr`.
    - [x] Sub-task: Implement import and wrong-type read/import diagnostics for `baffinbay_l4_proxy`.
    - [x] Sub-task: Implement import and wrong-type read/import diagnostics for `baffinbay_http_proxy`.
- [ ] Task: Evaluate Terraform state migration support.
    - [ ] Sub-task: Implement `resource.ResourceWithMoveState` so users can use `moved` blocks from `baffinbay_traffic_config` to typed resources.
    - [ ] Sub-task: Document an import-based migration path as a fallback for users who do not use `moved` blocks.
    - [ ] Sub-task: Add tests for whichever migration path is chosen.
- [ ] Task: Document the compatibility contract.
    - [ ] Sub-task: Explain that the old resource remains supported but deprecated.
    - [ ] Sub-task: Explain that changing from unified to typed resources should not require recreating remote traffic configs when state migration or import is done correctly.

## Phase 4: Validation and Tests
- [ ] Task: Add schema tests for all typed resources.
    - [ ] Sub-task: Assert typed resources do not expose irrelevant attributes from other traffic config types.
    - [ ] Sub-task: Assert typed resources do not expose a user-settable `type`.
    - [x] Sub-task: Assert `baffinbay_l4_proxy` does not expose `type`, HTTP-only fields, or Routed DSR-only fields.
    - [x] Sub-task: Assert `baffinbay_l4_proxy` exposes and maps L4 access-control defaults and configured rule values.
    - [x] Sub-task: Assert `baffinbay_http_proxy` does not expose `type`, L4-only fields, Routed DSR-only fields, or deprecated access-control fields.
- [ ] Task: Add CRUD tests for each typed resource.
    - [x] Sub-task: Cover HTTP Proxy create, update, read, import, and delete.
    - [x] Sub-task: Cover L4 Proxy create, update, read, import, and delete.
    - [x] Sub-task: Cover Routed DSR create, update, read, import, and delete.
- [ ] Task: Add negative import tests.
    - [ ] Sub-task: Importing the wrong remote type into a typed resource must fail with a useful error.
    - [x] Sub-task: Cover wrong-type import for `baffinbay_l4_proxy`.
    - [x] Sub-task: Cover wrong-type import for `baffinbay_routed_dsr`.
    - [x] Sub-task: Cover wrong-type read and import for `baffinbay_http_proxy`.
- [ ] Task: Preserve existing tests for `baffinbay_traffic_config`.
    - [x] Sub-task: Keep the current tests passing while the compatibility resource remains registered.

## Phase 5: Documentation and Examples
- [ ] Task: Add examples for each typed resource.
    - [x] Sub-task: Move or copy the current HTTP Proxy example to the new HTTP resource.
    - [ ] Sub-task: Move or copy the current L4 Proxy example to the new L4 resource.
    - [ ] Sub-task: Add a Routed DSR example if there is enough API confidence to make it useful.
- [x] Task: Regenerate provider documentation with `tfplugindocs`.
    - [x] Sub-task: Ensure generated docs describe only valid attributes for each typed resource.
    - [x] Sub-task: Add a deprecation note to `baffinbay_traffic_config` documentation immediately when typed resources ship.
- [ ] Task: Add migration documentation.
    - [ ] Sub-task: Include examples for move-state migration if implemented.
    - [x] Sub-task: Include examples for import-based migration as the fallback.

## HTTP Proxy Contract Reconciliation

### Current Status

- `baffinbay_http_proxy` is implemented and registered with a dedicated client contract, Terraform model, create/update mapping, response mapping, and cross-field validation.
- The implementation follows the portal and traffic-management sources listed below where the checked-in OpenAPI document differs.
- Future HTTP fields must remain in the dedicated HTTP model and must not be added through the unified `mapModelToRequest` compatibility path.
- Keep `baffinbay_traffic_config`, `baffinbay_l4_proxy`, and `baffinbay_routed_dsr` registered and compatible while this work is performed.
- The unrelated modified `/Users/jamil/bbn/web.portal/Dockerfile` belongs to the user and must not be changed.

### Authoritative Sources

Use these implementations before the checked-in OpenAPI document when behavior differs:

- Portal request mapping:
  `/Users/jamil/bbn/web.portal/src/api/traffic-mgmt/traffic-configs/utils/map-to-api-utils.ts`
- Portal response/default mapping:
  `/Users/jamil/bbn/web.portal/src/api/traffic-mgmt/traffic-configs/utils/map-from-api-utils.ts`
- Portal HTTP form validation:
  `/Users/jamil/bbn/web.portal/src/zod-schemas/traffic-mgmt/traffic-configs/http-proxy-schemas.ts`
- Portal API DTO validation:
  `/Users/jamil/bbn/web.portal/src/api/traffic-mgmt/traffic-configs/zod-schemas/traffic-configs-schemas.ts`
- Backend deserialized wire contract:
  `/Users/jamil/bbn/traffic-management/App/Controllers/Api/TrafficConfig/Dto/HttpProxy/HttpProxyDeserialized/Type.fs`
- Backend create validation and defaults:
  `/Users/jamil/bbn/traffic-management/App/Controllers/Api/TrafficConfig/Dto/HttpProxy/HttpProxyDeserialized/ToHttpProxyCreateRequestDto.fs`
  `/Users/jamil/bbn/traffic-management/App/Controllers/Api/TrafficConfig/Dto/HttpProxy/HttpProxyCreateRequestDto.fs`
- Backend patch validation:
  `/Users/jamil/bbn/traffic-management/App/Controllers/Api/TrafficConfig/Dto/HttpProxy/HttpProxyDeserialized/ToHttpProxyPatchRequestDto.fs`
  `/Users/jamil/bbn/traffic-management/App/Controllers/Api/TrafficConfig/Dto/HttpProxy/HttpProxyPatchRequestDto.fs`
- Backend response mapping:
  `/Users/jamil/bbn/traffic-management/App/Controllers/Api/TrafficConfig/Dto/HttpProxy/HttpProxyResponseDto.fs`

### Settled Contract Findings

- HTTP Proxy uses API version `0.1.0`.
- Certificate trust-store arrays use API `caCertificateIds`, not `caBundleIds`.
- Frontend certificate verification supports `DISABLED` and `VERIFY_AND_REJECT` only.
- Frontend and backend certificate verification do not accept `verifyCrl`.
- HTTP/1.1 protocol settings contain `httpVersion` and required `enableWebsockets`.
- HTTP/2 protocol settings contain only `httpVersion` and require a secure frontend.
- Portal request mapping does not send `multiplexing`; remove it from the typed Terraform schema.
- Backend `serverName` is optional.
- HTTP backend delivery methods are `ROUND_ROBIN`, `LEAST_CONNECTIONS`, and `IP_HASH`.
- Secure frontend hosts require `host`, `certificateId`, and `tlsConfig`.
- A secure frontend requires `redirectHttp`, complete HSTS settings, and at least one certificate-backed host.
- A plaintext frontend requires at least one allowed host and must not send secure-only fields.
- Both frontend modes require at least one of `ipv4` or `ipv6`.
- HTTP IP access control includes IP ranges, IP lists, geo locations, ASNs, and known services. IP-range and known-service rules also support bot-protection bypass metadata.
- `geoFencing` and `allowedSources` are backend compatibility inputs only. Do not expose or send them from the typed resource.
- Portal HTTP updates use `PUT`, but traffic-management processes the attributes as a patch. Typed create and update payload construction must therefore be separate.

### Defaulting Decision

Target portal-created behavior rather than relying on backend omission defaults. The two differ in material ways:

- Portal defaults WAF enforcement to `LOG`; backend omission defaults it to `DISABLED`.
- Portal defaults bot protection to `AUTO` with `JS`; backend omission defaults to `AUTO` with `HTTP`.
- Portal defaults both global rate-limit dimensions to `BLOCK`, `10 r/s`, burst `100`.
- Portal defaults backend delivery to `LEAST_CONNECTIONS` in a new form.
- Portal defaults connection reuse to enabled.
- Portal defaults secure redirect to enabled and HSTS max age to 168 days when creating form state.

Defaults exposed in Terraform must be represented in schema/state. Hidden fields that are deferred to later phases must not be reset during update.

### Phase HTTP-0: Make the Core Resource Safe

- [x] Revert the provisional request wire name from `caBundleIds` to `caCertificateIds`.
- [x] Add dedicated HTTP request and response structs in `internal/client`; do not reuse request structs whose zero values leak fields from L4 or the unified resource.
- [x] Add dedicated HTTP Terraform models and mapping helpers; stop routing typed HTTP through `TrafficConfigResourceModel` and `mapModelToRequest`.
- [x] Split mapping into create and update functions.
    - [x] Create sends all API-required core fields and intentional portal-compatible defaults.
    - [x] Update sends only fields managed by the typed schema so deferred/unmanaged remote fields are preserved by backend patch semantics.
- [x] Keep API type `httpProxy` and version `0.1.0` internal.
- [x] Correct the core schema.
    - [x] Remove `protocol_settings.multiplexing`.
    - [x] Remove frontend and backend `verify_crl`.
    - [x] Remove frontend verification mode `VERIFY_AND_FORWARD`.
    - [x] Make `backend.server_name` optional.
    - [x] Add backend delivery method `IP_HASH`.
    - [x] Keep `frontend`, `backend`, and `protocol_settings` required.
- [x] Add resource-level conditional validation.
    - [x] Require at least one of frontend IPv4 or IPv6.
    - [x] Require HTTP/2 to use a secure frontend.
    - [x] Require secure hosts to include certificate ID and TLS config.
    - [x] Reject certificate, HSTS, redirect, and client-verification fields for plaintext frontends.
    - [x] Require redirect and complete HSTS values for secure frontends.
    - [x] Reject secure redirect on port 80.
    - [x] Require at least one CA certificate ID for `VERIFY_AND_REJECT` and `CUSTOM_TRUSTSTORE`.
- [x] Preserve import and wrong-type diagnostics.
- [x] Add an import-then-update test proving deferred remote HTTP fields are not reset.
- [x] Do not live-test the resource until this phase passes.

### Phase HTTP-1: Stable Portal Controls

- [x] Keep API-managed `gatewayPath` out of the Terraform schema and typed HTTP request mapping.
- [x] Add `connection_reuse_enabled`, defaulting to `true`.
- [x] Add `bot_protection`.
    - [x] `strategy`: `ALWAYS_ON | AUTO | DISABLED`.
    - [x] `challenge_type`: `HTTP | JS`.
    - [x] Portal-compatible defaults: `AUTO` and `JS`.
- [x] Add `data_protection.log_redaction.headers` and `.cookies`.
- [x] Add `custom_pages` entries with `id` and portal custom-page type validation.
- [x] Add create, read, update, import, and default-stability tests for each control.

### Phase HTTP-2: Full Rate Limiting and HTTP Access Control

- [x] Replace the provisional single-field `rate_limiting.enforcement` model with the real structure.
    - [x] `by_source_ip`: enforcement, rate value/unit, and burst.
    - [x] `by_source_ip_and_url`: enforcement, rate value/unit, and burst.
    - [x] `exclusions`: CIDR list.
    - [x] Validate enforcement as `BLOCK | DISABLED`, unit as `r/s | r/m`, and numeric ranges as 1 through 10000.
- [x] Replace the reused L4 access-control model with an HTTP-specific model.
    - [x] Keep default policy, IP ranges, IP lists, geo locations, and ASNs.
    - [x] Add known-service rules.
    - [x] Add `bypass_bot_protection` to IP-range and known-service rules.
    - [x] Preserve per-rule policy and API list shape.
- [x] Add exact request and response tests for defaults and configured values.

### Phase HTTP-3: Modern WAF Model

- [x] Replace legacy `core_rule_set_id` request mapping with `core_rule_set.version`.
- [x] Keep response compatibility with legacy `coreRuleSetId` only if traffic-management can still return it.
- [x] Model source exclusions as explicit enabled state plus sources.
- [x] Model HTTP compliance global configuration and resource configs.
- [x] Replace legacy path exclusion `type/value/description` with `match`, `disable_all`, and rule IDs.
- [x] Add cookie exclusions.
- [x] Add `matched_data_enabled`.
- [x] Add staged WAF configuration, including its path/cookie exclusions and mode-specific settings.
- [x] Validate WAF method lists include `GET` and `POST`, paranoia level is 1 through 4, required rule IDs are UUIDs, and CRS selectors use a valid three-part version or trailing-wildcard form.
- [x] Add portal-payload fixture tests for WAF mapping in both directions.

### Phase HTTP-4: Traffic Rules

- [x] Model traffic-rule names, matching conditions, host selectors, and paths.
- [x] Model actions for backends, headers, host header/SNI, redirects, rate limits, and maximum body size.
- [x] Preserve API null-versus-empty behavior for optional actions.
- [x] Add uniqueness and conditional validation matching the portal form.
- [x] Add round-trip tests using payloads captured from portal mapping.

### Phase HTTP-5: State, Compatibility, and Documentation

- [x] Decide whether any released provider version contains the provisional HTTP schema.
    - Not applicable: the provisional typed HTTP schema was never committed or released, so no state upgrader is required.
    - [x] Development state created with the provisional schema may need removal and re-import after schema changes.
- [x] Verify `baffinbay_traffic_config` behavior remains unchanged except for wire-level fixes that are valid for existing users.
- [x] Add an HTTP Proxy example only after HTTP-0 through the selected parity phases are stable.
- [x] Regenerate provider documentation.
- [x] Re-run `gofmt`, `go test ./internal/provider`, and `go test ./...`.

### Contract Test Strategy

- [x] Add exact JSON create fixtures generated from portal `mapTrafficConfigRequestToApi` behavior.
- [x] Add exact JSON update fixtures and assert omitted fields remain omitted.
- [x] Add response fixtures generated from traffic-management `HttpProxyResponseDto` behavior.
- [x] Cover plaintext HTTP/1.1, secure HTTP/1.1, and secure HTTP/2.
- [x] Cover explicit `false` values so JSON omission cannot change semantics.
- [x] Cover system and custom backend trust stores and frontend client-certificate verification.
- [x] Cover create defaults separately from imported remote values.
- [x] Cover import followed by an unrelated update without resetting WAF, rate limiting, bot protection, access control, custom pages, data protection, or traffic rules.

### Post-Implementation Status

1. HTTP-0 through HTTP-5 are complete.
2. The provisional typed HTTP schema was not released, so no schema upgrader was added.
3. Development state written by the provisional schema may need to be removed and imported again by traffic config UUID.
4. Future work in this track remains limited to non-HTTP items that are still unchecked above, including move-state support and the remaining typed-resource examples.

## Acceptance Criteria
- The provider exposes typed resources for HTTP Proxy, L4 Proxy, and Routed DSR traffic configurations.
- Typed resources set the API `type` internally and do not require a Terraform `type` argument.
- Typed schemas expose only fields valid for their traffic config type.
- Existing `baffinbay_traffic_config` configurations continue to work.
- Imports reject remote traffic configs with the wrong API type.
- Documentation and examples exist for all typed resources.
- Unit and acceptance tests cover schema, CRUD, import, and compatibility behavior.

## Open Questions
- No blocking HTTP Proxy questions remain. Move-state support and the remaining non-HTTP documentation tasks are still open for the broader track.
