# Implementation Plan - Traffic Management Resources

## Phase 1: Supporting Resources & IP Management
- [x] Task: Certificate Management Resource (`baffinbay_certificate`)
    - [x] Sub-task: Write acceptance tests for PEM import and Let's Encrypt config (RED).
    - [x] Sub-task: Implement schema and CRUD logic in `internal/provider/certificate_resource.go` (GREEN).
    - [x] Sub-task: Implement `ImportState` logic for UUID-based import.
- [x] Task: CA Bundle Resource (`baffinbay_ca_bundle`)
    - [x] Sub-task: Write acceptance tests for CA bundle creation (RED).
    - [x] Sub-task: Implement schema and CRUD logic in `internal/provider/ca_bundle_resource.go` (GREEN).
    - [x] Sub-task: Implement `ImportState` logic.
- [x] Task: Custom Page Resource (`baffinbay_custom_page`)
    - [x] Sub-task: Write acceptance tests for HTML upload (RED).
    - [x] Sub-task: Implement schema and CRUD logic in `internal/provider/custom_page_resource.go` (GREEN).
    - [x] Sub-task: Implement `ImportState` logic.
- [ ] Task: IP Allocation Resource (`baffinbay_ip_allocation`)
    - [ ] Sub-task: Deferred: API endpoints for creation not found in spec.

## Phase 2: Core Traffic Configuration (L4 & DSR)
- [x] Task: Unified `baffinbay_traffic_config` Resource Scaffold
    - [x] Sub-task: Define the unified schema with type discriminator in `internal/provider/traffic_config_resource.go`.
    - [x] Sub-task: Implement basic validation and CRUD scaffolding.
- [ ] Task: Implement L4 Proxy and Routed DSR Logic
    - [x] Sub-task: Write acceptance tests for L4 and DSR configurations (RED).
    - [x] Sub-task: Implement flattened attributes and CRUD logic for L4/DSR types (GREEN).
    - [x] Sub-task: Implement `ImportState` logic for the unified resource.
    - [x] Sub-task: Connect `Create` and `Delete` to Client methods (Real API).

## Phase 3: Refactoring (Client Cleanup)
- [x] Task: Split Client Package
    - [x] Sub-task: Move authentication logic to `internal/client/auth.go`.
    - [x] Sub-task: Move custom page logic to `internal/client/custom_page.go`.
    - [x] Sub-task: Move certificate and CA bundle logic to `internal/client/certificate.go`.
    - [x] Sub-task: Move traffic config logic to `internal/client/traffic_config.go`.
    - [x] Sub-task: Ensure all tests pass after refactoring.

## Phase 4: Advanced Traffic Configuration (HTTP Proxy & WAF)
- [ ] Task: Implement HTTP Proxy Support
    - [ ] Sub-task: Write acceptance tests for HTTP Proxy with TLS/Plaintext frontends (RED).
    - [ ] Sub-task: Implement HTTP specific schema and CRUD logic (GREEN).
- [ ] Task: Implement WAF and Protocol Settings
    - [ ] Sub-task: Write acceptance tests for WAF enforcement and rule sets (RED).
    - [ ] Sub-task: Implement WAF attributes mapping and protocol settings (GREEN).

## Phase 5: Data Sources & Finalization
- [ ] Task: Implement Data Sources
    - [ ] Sub-task: Implement `baffinbay_traffic_configs` data source.
    - [ ] Sub-task: Implement `baffinbay_certificates` data source.
    - [ ] Sub-task: Implement `baffinbay_ca_bundles` data source.
    - [ ] Sub-task: Implement `baffinbay_custom_pages` data source.
    - [ ] Sub-task: Implement `baffinbay_ip_sources` data source.
- [ ] Task: Documentation & Verification
