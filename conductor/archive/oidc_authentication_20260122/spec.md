# Specification: OIDC Authentication Implementation

## 1. Overview
This track implements OIDC Client Credentials authentication as the primary authentication method for the Baffin Bay Terraform Provider, replacing the initial API Key implementation. This aligns the provider with the official Baffin Bay API authentication standards.

## 2. Goals
- Update the API Client to support obtaining a JWT access token via OIDC Client Credentials flow.
- Update the Provider Schema to accept `client_id` and `client_secret` instead of (or in addition to) `api_key`.
- Implement logic to authenticate once per Terraform execution and reuse the token.

## 3. Requirements

### 3.1 API Client Updates (`internal/client`)
- **Token Acquisition:** Implement a method to call `POST https://m2m-auth.baffinbay.com/oauth/token`.
- **Request Payload:**
  - `client_id`: (Required)
  - `client_secret`: (Required)
  - `grant_type`: "client_credentials"
  - `audience`: "https://portal.baffinbay.com"
  - `Content-Type`: "application/json"
- **Token Storage:** Store the `access_token` returned from the OIDC server.
- **Request Authorization:** Automatically attach the `Authorization: Bearer <access_token>` header to all subsequent API requests.

### 3.2 Provider Schema Updates (`internal/provider`)
- **New Attributes:**
  - `client_id`: (String, Optional/Required) Supported via environment variable `BAFFINBAY_CLIENT_ID`.
  - `client_secret`: (String, Optional/Required, Sensitive) Supported via environment variable `BAFFINBAY_CLIENT_SECRET`.
- **Legacy Support:** Mark `api_key` as deprecated or optional, ensuring a smooth transition.
- **Validation:** Ensure either OIDC credentials or an API Key (if still supported) are provided.

### 3.3 Authentication Lifecycle
- The provider will perform the OIDC exchange during the `Configure` phase of the Terraform lifecycle.
- The resulting token will be cached in the client instance for the duration of the Terraform plan/apply operation.

## 4. Acceptance Criteria
- Unit tests verify the OIDC token request payload and response handling.
- Acceptance tests verify that the provider can successfully authenticate using `client_id` and `client_secret`.
- The provider correctly errors if credentials are missing or invalid.
- sensitive data (`client_secret`) is not leaked in logs or state.

## 5. Out of Scope
- Token refreshing during long-running operations (tokens are valid for 24h).
- Advanced OIDC flows (e.g., Authorization Code).
- Persistent token caching between different Terraform runs.
