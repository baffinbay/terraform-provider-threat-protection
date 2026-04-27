# Specification: Provider Scaffold & Authentication

## 1. Overview
This track focuses on initializing the Terraform provider project structure and implementing the core authentication mechanism to interface with the Baffin Bay Threat Protection API. This forms the foundation for all subsequent resource management capabilities.

## 2. Goals
- Initialize a Go module for the Terraform provider.
- Set up the basic project structure following HashiCorp's recommended layout.
- Implement the provider entry point using `terraform-plugin-framework`.
- Implement the client logic to authenticate with the Baffin Bay API.
- Verify authentication using a simple data source or connectivity test.

## 3. Requirements
- **Language:** Go 1.21+ (or latest stable).
- **Framework:** `hashicorp/terraform-plugin-framework`.
- **Configuration:**
  - The provider must accept an `api_key` (sensitive) for authentication.
  - Optionally accept an `api_url` to support different environments (defaulting to the production API).
- **Security:**
  - API keys must be marked as sensitive in the Terraform schema.
  - Support reading credentials from environment variables (e.g., `BAFFINBAY_API_KEY`).
- **Testing:**
  - Unit tests for the authentication client.
  - Acceptance tests for the provider configuration (using `terraform-plugin-testing`).

## 4. Out of Scope
- Implementation of specific resources (e.g., policies, sensors) is out of scope for this track.
- Advanced retry logic or complex error handling beyond basic connectivity checks.
