# Implementation Plan - OIDC Authentication

## Phase 1: API Client OIDC Support
- [x] Task: Implement OIDC Token Acquisition
    - [x] Sub-task: Define OIDC request and response structs in `internal/client`.
    - [x] Sub-task: Add `Authenticate` method to `Client` to perform the OIDC Client Credentials exchange.
    - [x] Sub-task: Write unit tests with a mock OIDC server to verify token acquisition logic (RED/GREEN).
- [x] Task: Update Request Authorization
    - [x] Sub-task: Refactor `NewRequest` to ensure it uses the OIDC `access_token` if available.
    - [x] Sub-task: Update unit tests for `NewRequest` to verify the `Bearer` token header (RED/GREEN).

## Phase 2: Provider Schema & Configuration
- [x] Task: Update Provider Schema
    - [x] Sub-task: Add `client_id` and `client_secret` attributes to the schema in `internal/provider/provider.go`.
    - [x] Sub-task: Mark `api_key` as deprecated or optional.
    - [x] Sub-task: Update `BaffinBayProviderModel` to include new OIDC fields.
- [x] Task: Implement OIDC Configuration Logic
    - [x] Sub-task: Update `Configure` method to prioritize OIDC credentials from config or environment variables.
    - [x] Sub-task: Add logic to trigger `client.Authenticate()` during provider configuration.
    - [x] Sub-task: Write unit tests for schema validation and configuration logic (RED/GREEN).

## Phase 3: Verification & Cleanup
- [x] Task: Acceptance Testing
    - [x] Sub-task: Update `TestAccProvider` in `internal/provider/provider_test.go` to use OIDC credentials and a mock auth server.
    - [x] Sub-task: Verify end-to-end authentication flow (RED/GREEN).
- [x] Task: Documentation Updates
    - [x] Sub-task: Update `README.md` with OIDC configuration instructions and new environment variables.
    - [x] Sub-task: Regenerate provider documentation using `tfplugindocs`.

## Phase 4: Finalization
- [x] Task: Conductor - User Manual Verification 'Phase 1: API Client OIDC Support' (Protocol in workflow.md)
- [x] Task: Conductor - User Manual Verification 'Phase 2: Provider Schema & Configuration' (Protocol in workflow.md)
- [x] Task: Conductor - User Manual Verification 'Phase 3: Verification & Cleanup' (Protocol in workflow.md)
