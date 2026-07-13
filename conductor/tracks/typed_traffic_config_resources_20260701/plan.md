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
- [ ] Task: Map the current unified schema into type-specific schemas.
    - [ ] Sub-task: Identify common attributes shared across all three resources, such as `id`, `name`, and `deployment_state`.
    - [ ] Sub-task: Identify HTTP Proxy-only attributes and nested blocks.
    - [x] Sub-task: Identify L4 Proxy-only attributes and nested blocks.
    - [x] Sub-task: Identify L4 access-control attributes required by the API: `protocols`, `proxyProtocol`, `geoFencing`, `allowedSources`, and `ipBasedAccessControl`.
    - [x] Sub-task: Identify Routed DSR-only attributes from the API spec: `prefix`, `announced`, and `deployment_state` mapped to API `deployment.state`.
- [ ] Task: Decide whether nested object names should stay compatible with the unified resource.
    - [ ] Sub-task: Prefer preserving existing block names where they remain semantically correct.
    - [ ] Sub-task: Rename only when the current names hide type-specific behavior.

## Phase 2: Internal Refactor
- [ ] Task: Split reusable traffic config logic out of `traffic_config_resource.go`.
    - [ ] Sub-task: Extract shared models and schema builders for common frontend, backend, TLS, deployment, and polling behavior.
    - [ ] Sub-task: Extract request/response mapping helpers by API type.
    - [ ] Sub-task: Keep defaulting behavior explicit, especially HTTP Proxy defaults for WAF, rate limiting, bot protection, allowed sources, and related API fields.
- [ ] Task: Add typed resource implementations.
    - [ ] Sub-task: Implement metadata names for each typed resource.
    - [ ] Sub-task: Set the API `type` internally instead of exposing a Terraform `type` attribute.
    - [ ] Sub-task: Reuse the existing client create, read, update, delete, and polling methods.
    - [ ] Sub-task: Register the new resources in `BaffinBayProvider.Resources`.
    - [x] Sub-task: Implement and register the first typed resource, `baffinbay_routed_dsr`.
    - [x] Sub-task: Implement and register the second typed resource, `baffinbay_l4_proxy`.
    - [x] Sub-task: Implement complete L4 request/response mapping and defaults for protocols, proxy protocol, geo fencing, allowed sources, and IP based access control.

## Phase 3: Import and Migration
- [ ] Task: Implement import support for each typed resource.
    - [ ] Sub-task: Import by traffic config UUID, matching the existing unified resource import behavior.
    - [ ] Sub-task: After import/read, verify the remote API type matches the target Terraform resource type.
    - [ ] Sub-task: Return a clear diagnostic if a user imports an HTTP Proxy into the L4 or Routed DSR resource.
    - [x] Sub-task: Implement import and wrong-type read/import diagnostics for `baffinbay_routed_dsr`.
    - [x] Sub-task: Implement import and wrong-type read/import diagnostics for `baffinbay_l4_proxy`.
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
- [ ] Task: Add CRUD tests for each typed resource.
    - [ ] Sub-task: Cover HTTP Proxy create, update, read, import, and delete.
    - [x] Sub-task: Cover L4 Proxy create, update, read, import, and delete.
    - [x] Sub-task: Cover Routed DSR create, update, read, import, and delete.
- [ ] Task: Add negative import tests.
    - [ ] Sub-task: Importing the wrong remote type into a typed resource must fail with a useful error.
    - [x] Sub-task: Cover wrong-type import for `baffinbay_l4_proxy`.
    - [x] Sub-task: Cover wrong-type import for `baffinbay_routed_dsr`.
- [ ] Task: Preserve existing tests for `baffinbay_traffic_config`.
    - [ ] Sub-task: Keep the current tests passing while the compatibility resource remains registered.

## Phase 5: Documentation and Examples
- [ ] Task: Add examples for each typed resource.
    - [ ] Sub-task: Move or copy the current HTTP Proxy example to the new HTTP resource.
    - [ ] Sub-task: Move or copy the current L4 Proxy example to the new L4 resource.
    - [ ] Sub-task: Add a Routed DSR example if there is enough API confidence to make it useful.
- [ ] Task: Regenerate provider documentation with `tfplugindocs`.
    - [ ] Sub-task: Ensure generated docs describe only valid attributes for each typed resource.
    - [ ] Sub-task: Add a deprecation note to `baffinbay_traffic_config` documentation immediately when typed resources ship.
- [ ] Task: Add migration documentation.
    - [ ] Sub-task: Include examples for move-state migration if implemented.
    - [ ] Sub-task: Include examples for import-based migration as the fallback.

## Acceptance Criteria
- The provider exposes typed resources for HTTP Proxy, L4 Proxy, and Routed DSR traffic configurations.
- Typed resources set the API `type` internally and do not require a Terraform `type` argument.
- Typed schemas expose only fields valid for their traffic config type.
- Existing `baffinbay_traffic_config` configurations continue to work.
- Imports reject remote traffic configs with the wrong API type.
- Documentation and examples exist for all typed resources.
- Unit and acceptance tests cover schema, CRUD, import, and compatibility behavior.

## Open Questions
- None.
