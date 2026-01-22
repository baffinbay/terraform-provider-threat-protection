# Implementation Plan - Provider Scaffold & Auth

## Phase 1: Project Initialization
- [x] Task: Project Configuration & Hygiene
    - [x] Sub-task: Create `.gitignore` to exclude local environment files and the `conductor/` directory.
- [x] Task: Initialize Go Module and Project Structure
    - [x] Sub-task: Create `go.mod` and install dependencies (`hashicorp/terraform-plugin-framework`).
    - [x] Sub-task: Create standard directory structure (`internal/provider`, `internal/client`, `examples`).
    - [x] Sub-task: Create a dummy test to verify the test runner works (RED/GREEN).

## Phase 2: API Client Implementation
- [x] Task: Define Client Interface and Configuration
    - [x] Sub-task: Create `internal/client` package.
    - [x] Sub-task: Define a `Client` struct that holds the Host URL and HTTP Client.
    - [x] Sub-task: Write unit tests for client initialization (RED/GREEN).
- [x] Task: Implement Authentication Logic
    - [x] Sub-task: Add `SetAuth` method to the client to inject the API Key into headers.
    - [x] Sub-task: Write unit tests ensuring the `Authorization` header is set correctly (RED/GREEN).

## Phase 3: Provider Implementation
- [x] Task: Define Provider Schema
    - [x] Sub-task: Create `internal/provider/provider.go`.
    - [x] Sub-task: Define the provider schema with `api_key` (Sensitive) and `api_url` (Optional).
    - [x] Sub-task: Implement `Configure` method to instantiate the API client with the provided config.
- [x] Task: Connect Provider to Main
    - [x] Sub-task: Create `main.go` serving the provider.
    - [x] Sub-task: Write a basic acceptance test verifying the provider can initialize without errors (RED/GREEN).

## Phase 4: Verification
- [x] Task: End-to-End Authentication Test
    - [x] Sub-task: Create a "ping" or "validate" method in the client (even if it just hits a health endpoint or similar).
    - [x] Sub-task: Write an acceptance test that configures the provider with a mock/real key and asserts success.
