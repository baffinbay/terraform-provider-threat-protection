# Implementation Plan - Provider Scaffold & Auth

## Phase 1: Project Initialization
- [x] Task: Project Configuration & Hygiene
    - [x] Sub-task: Create `.gitignore` to exclude local environment files and the `conductor/` directory.
- [x] Task: Initialize Go Module and Project Structure
    - [x] Sub-task: Create `go.mod` and install dependencies (`hashicorp/terraform-plugin-framework`).
    - [x] Sub-task: Create standard directory structure (`internal/provider`, `internal/client`, `examples`).
    - [x] Sub-task: Create a dummy test to verify the test runner works (RED/GREEN).

## Phase 2: API Client Implementation
- [ ] Task: Define Client Interface and Configuration
    - [ ] Sub-task: Create `internal/client` package.
    - [ ] Sub-task: Define a `Client` struct that holds the Host URL and HTTP Client.
    - [ ] Sub-task: Write unit tests for client initialization (RED/GREEN).
- [ ] Task: Implement Authentication Logic
    - [ ] Sub-task: Add `SetAuth` method to the client to inject the API Key into headers.
    - [ ] Sub-task: Write unit tests ensuring the `Authorization` header is set correctly (RED/GREEN).

## Phase 3: Provider Implementation
- [ ] Task: Define Provider Schema
    - [ ] Sub-task: Create `internal/provider/provider.go`.
    - [ ] Sub-task: Define the provider schema with `api_key` (Sensitive) and `api_url` (Optional).
    - [ ] Sub-task: Implement `Configure` method to instantiate the API client with the provided config.
- [ ] Task: Connect Provider to Main
    - [ ] Sub-task: Create `main.go` serving the provider.
    - [ ] Sub-task: Write a basic acceptance test verifying the provider can initialize without errors (RED/GREEN).

## Phase 4: Verification
- [ ] Task: End-to-End Authentication Test
    - [ ] Sub-task: Create a "ping" or "validate" method in the client (even if it just hits a health endpoint or similar).
    - [ ] Sub-task: Write an acceptance test that configures the provider with a mock/real key and asserts success.
