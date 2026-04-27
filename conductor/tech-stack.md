# Technology Stack

## Core Development
- **Language:** Go (Golang) - The industry standard for Terraform provider development, offering strong concurrency support and efficient execution.
- **Framework:** Terraform Plugin Framework (`terraform-plugin-framework`) - The modern, official framework from HashiCorp for building robust and feature-rich Terraform providers.

## Integration & Communication
- **API Interface:** Baffin Bay REST API - The provider will interface with the Baffin Bay Threat Protection API (https://docs.baffinbay.com/docs/products/api-docs/).
- **Communication:** Standard Go `net/http` library (or a specialized SDK if available) for secure and reliable API requests.

## Testing & Quality
- **Unit Testing:** Go's built-in `testing` package for validating internal logic.
- **Acceptance Testing:** Terraform's `helper/resource` and `helper/schema` (or framework equivalents) for end-to-end testing against actual Baffin Bay API endpoints.
