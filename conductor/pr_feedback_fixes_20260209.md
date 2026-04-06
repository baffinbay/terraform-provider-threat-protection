# Implementation Plan - PR Feedback Fixes

## Background & Motivation
Feedback from a recent Pull Request highlighted two main issues:
1. Examples and tests currently use actively assigned IP addresses (e.g., `185.195.93.xxx`) instead of standard example/documentation IPs.
2. Several enum attributes lack plan-time validation, which means invalid values are only caught during `terraform apply`.

## Scope & Impact
This plan covers updating the IP addresses in specific examples, tests, and the API spec to use RFC 5737 and RFC 3849 standard documentation ranges (`203.0.113.0/24` and `2001:db8::/32`). It also includes adding `stringvalidator.OneOf()` from `terraform-plugin-framework-validators` to the schemas of `baffinbay_certificate` and `baffinbay_traffic_config` to catch invalid enums at `terraform plan` time.

## Proposed Solution
### Phase 1: Example IP Replacements
Update the following files to replace the actively assigned IP addresses with example documentation addresses:
- `examples/traffic_config/http_proxy/main.tf` (`185.195.93.212` -> `203.0.113.1`)
- `examples/traffic_config/l4_proxy/main.tf` (`185.195.93.210` -> `203.0.113.2`)
- `internal/provider/traffic_config_live_test.go` (`185.195.93.211` -> `203.0.113.3`)
- `api-spec/traffic-mgmt.yml` (`185.195.95.0/24` -> `203.0.113.0/24`, `2a0a:56c4:8001:3::/64` -> `2001:db8::/64`)

### Phase 2: Plan-time Validation (Enums)
Import `github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator` and `github.com/hashicorp/terraform-plugin-framework/schema/validator`.
Add `Validators: []validator.String{stringvalidator.OneOf(...)}` to the following attributes:

- **`baffinbay_certificate`** in `internal/provider/certificate_resource.go`:
  - `type` ("pem", "lets_encrypt")

- **`baffinbay_traffic_config`** in `internal/provider/traffic_config_resource.go`:
  - `type` ("l4Proxy", "routedDsr", "httpProxy")
  - `deployment_state` ("DEPLOYED", "UNDEPLOYED")
  - `frontend.connection_type` ("SECURE", "PLAINTEXT")
  - `frontend.hosts[].tls_config` ("ADVANCED", "INTERMEDIATE")
  - `frontend.client_certificate_verification.mode` ("DISABLED", "VERIFY_AND_REJECT")
  - `protocol_settings.version` ("HTTP1.1", "HTTP2.0")
  - `backend.delivery_method` ("ROUND_ROBIN", "LEAST_CONNECTIONS")
  - `backend.tls_settings.verify_certificate.mode` ("DISABLED", "SYSTEM_TRUSTSTORE", "CUSTOM_TRUSTSTORE")

## Alternatives Considered
- **Waiting until `apply`:** Leaving the validation to the API. However, failing fast at `plan` time is a much better UX for Terraform users.
- **Using a generic validator:** Custom validation logic could be written, but `stringvalidator.OneOf()` is the standard framework-provided way to validate string enums.

## Verification
- Run `go test ./...` and ensure all tests continue to pass.
- Run a manual `terraform plan` with an invalid enum value to verify that the plan fails as expected.
