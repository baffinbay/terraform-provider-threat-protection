# Implementation Plan - Production Readiness

## Phase 1: Authentication & Connection Verification
- [x] Task: Audit Environment Setup
    - [x] Sub-task: Verify Go installation and pathing.
    - [x] Sub-task: Confirm OIDC credentials in `.env` are valid via Ping test.
- [ ] Task: Live API Readiness Check
    - [ ] Sub-task: Implement a diagnostic tool or test to verify real API connectivity using the provider's `SetAuth` and `Ping`.

## Phase 2: Resource & Data Source Audit
- [x] Task: Test Suite Stabilization
    - [x] Sub-task: Fix `account_id` requirement in all existing acceptance tests.
    - [x] Sub-task: Align `Certificate` resource test paths with actual API.
- [ ] Task: Real Resource Verification
    - [ ] Sub-task: Run data source tests against Live API to verify Read logic.
    - [ ] Sub-task: Run limited resource tests (Create/Delete) against Live API to verify Write logic.
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
- [ ] Task: Final Build & Linting
    - [ ] Sub-task: Run `golangci-lint` and fix high-priority warnings.
