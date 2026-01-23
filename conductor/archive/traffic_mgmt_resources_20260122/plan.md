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
    - [ ] Sub-task: Write acceptance tests for IP allocation (RED).
    - [ ] Sub-task: Implement schema and CRUD logic in `internal/provider/ip_allocation_resource.go` (GREEN).
    - [ ] Sub-task: Implement `ImportState` logic.

## Phase 2: Core Traffic Configuration (L4 & DSR)
- [x] Task: Unified `baffinbay_traffic_config` Resource Scaffold
    - [x] Sub-task: Define the unified schema with type discriminator in `internal/provider/traffic_config_resource.go`.
    - [x] Sub-task: Implement basic validation and CRUD scaffolding.
- [ ] Task: Implement L4 Proxy and Routed DSR Logic
    - [ ] Sub-task: Write acceptance tests for L4 and DSR configurations (RED).
    - [ ] Sub-task: Implement flattened attributes and CRUD logic for L4/DSR types (GREEN).
    - [ ] Sub-task: Implement `ImportState` logic for the unified resource.

## Phase 3: Advanced Traffic Configuration (HTTP Proxy & WAF)
- [ ] Task: Implement HTTP Proxy Support
    - [ ] Sub-task: Write acceptance tests for HTTP Proxy with TLS/Plaintext frontends (RED).
    - [ ] Sub-task: Implement HTTP specific schema and CRUD logic (GREEN).
- [ ] Task: Implement WAF and Protocol Settings
    - [ ] Sub-task: Write acceptance tests for WAF enforcement and rule sets (RED).
    - [ ] Sub-task: Implement WAF attributes mapping and protocol settings (GREEN).

## Phase 4: Data Sources & Finalization
- [ ] Task: Implement Data Sources
    - [ ] Sub-task: Implement `baffinbay_traffic_configs` data source.
    - [ ] Sub-task: Implement `baffinbay_certificates` data source.
    - [ ] Sub-task: Implement `baffinbay_ip_sources` data source.
- [ ] Task: Documentation & Verification
    - [ ] Sub-task: Run `go generate` to update provider documentation using `tfplugindocs`.
    - [ ] Sub-task: Perform final end-to-end verification of all resources and data sources.
