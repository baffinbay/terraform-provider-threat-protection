# Product Guidelines

## Voice and Tone
- **Technical and Precise:** Documentation and provider messages must be accurate, using industry-standard terminology. Focus on clarity and technical correctness to build trust with infrastructure and security professionals.

## Naming Conventions
- **Standard Terraform Naming:** All resources and attributes must use `snake_case`. Resource names should be prefixed with `baffinbay_` (e.g., `baffinbay_policy`). This ensures the provider feels native to the Terraform ecosystem and follows HashiCorp's best practices.

## Error Handling
- **Informative and Actionable:** Error messages must provide clear context on the failure and, where possible, offer guidance on how to resolve the issue. This is critical for security configurations where ambiguity can lead to vulnerabilities.

## Security and Sensitive Data
- **Strict Secrecy:** Sensitive credentials (like API keys) must be handled with care. The provider should prioritize obtaining these from environment variables or secure provider configurations. Documentation must explicitly discourage hardcoding secrets and provide clear examples of secure management.
