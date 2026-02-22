# Product Guidelines

## Voice and Tone
- **Technical and Precise:** Documentation and provider messages must be accurate, using industry-standard terminology. Focus on clarity and technical correctness to build trust with infrastructure and security professionals.

## Naming Conventions
- **Standard Terraform Naming:** All resources and attributes must use `snake_case`. Resource names should be prefixed with `baffinbay_` (e.g., `baffinbay_policy`). This ensures the provider feels native to the Terraform ecosystem and follows HashiCorp's best practices.

## Error Handling
- **Informative and Actionable:** Error messages must provide clear context on the failure and, where possible, offer guidance on how to resolve the issue. This is critical for security configurations where ambiguity can lead to vulnerabilities.

## Infrastructure Management Patterns
- **Shadow Utility:** For complex API integrations, implement a companion CLI tool (e.g., `bb-tool`) that shares the client library. This provides a direct path for troubleshooting and API exploration independent of the Terraform lifecycle.
- **Async Handling:** Treat asynchronous status codes (e.g., `202 Accepted`) as successful task initiations for Create/Update operations.

## Pull Request Hygiene
- **Verbose Descriptions:** PR descriptions for phase completions must be detailed and categorized. They must capture:
    - Implemented Resources and Data Sources.
    - Build and CI/CD enhancements.
    - Troubleshooting utilities developed.
    - Testing results (Unit, Acceptance, and Live verification).
    - Technical fixes and architectural improvements.
