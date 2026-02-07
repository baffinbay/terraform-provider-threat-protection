# Implementation Plan - Production Readiness

## Phase 1: Authentication & Connection Verification
- [x] Task: Audit Environment Setup
    - [x] Sub-task: Verify Go installation and pathing.
    - [x] Sub-task: Confirm OIDC credentials in `.env` are valid via Ping test.
- [x] Task: Live API Readiness Check
    - [x] Sub-task: Implement a diagnostic tool or test to verify real API connectivity using the provider's `SetAuth` and `Ping`.

## Phase 2: Resource & Data Source Audit
- [x] Task: Test Suite Stabilization
    - [x] Sub-task: Fix `account_id` requirement in all existing acceptance tests.
    - [x] Sub-task: Align `Certificate` resource test paths with actual API.
- [x] Task: Schema & Spec Alignment
    - [x] Sub-task: Compare `internal/provider` schemas with `api-spec/traffic-mgmt.yml`.
    - [x] Sub-task: Refactor `TrafficConfigResource` schema to support complex HTTP Proxy settings (HSTS, Protocol Versions).
    - [x] Sub-task: Refactor `WafModel` to support full compliance and exclusion configurations.
    - [x] Sub-task: Implement missing attributes or correct misaligned ones (e.g., protocol settings mapping).
- [x] Task: Implement Update Logic
    - [x] Sub-task: Add `UpdateTrafficConfig` method to the client.
    - [x] Sub-task: Implement `Update` method in `TrafficConfigResource` with full attribute mapping.
- [x] Task: Documentation Automation
    - [x] Sub-task: Verify `tfplugindocs` generation works for all resources.
- [x] Task: Client & Test Architecture Refactor (TDD Readiness)
    - [x] Sub-task: Implement `ForceRefresh` and `SkipTimer` logic in `Authenticate` to support unit testing.
    - [x] Sub-task: Isolation: Ensure tests use `t.TempDir()` for `.env` and `.token_time` to prevent state leakage.
    - [x] Sub-task: Interface Audit: Decouple HTTP client and file system access for better mocking.
    - [x] Sub-task: Fix current failing unit tests (`TestAuthenticate`, `TestProviderConfigure_OIDC`).
- [ ] Task: Final Build & Linting
    - [x] Task: Run `golangci-lint` and fix high-priority warnings.
- [x] Task: API Troubleshooting Utility
    - [x] Sub-task: Implement a small tool to troubleshoot the API and manually manage resources during the testing phase.
- [~] Task: Real Resource Verification
    - [x] Sub-task: Fix "IP Bug" (frontend.ip mapping) in Traffic Configuration resource.
    - [x] Sub-task: Fix ProtocolSettings validation error in Traffic Configuration schema.
    - [ ] Sub-task: Run data source tests against Live API to verify Read logic.
    - [ ] Sub-task: Create, Update, and Destroy **Traffic Configuration** resource and verify manually in Portal.
    - [ ] Sub-task: Create, Update, and Destroy **Certificate** resource and verify manually in Portal.
    - [ ] Sub-task: Create, Update, and Destroy **CA Bundle** resource and verify manually in Portal.
    - [ ] Sub-task: Create, Update, and Destroy **Custom Page** resource and verify manually in Portal.
