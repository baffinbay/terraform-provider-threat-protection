# Specification: Traffic Management Resources & Data Sources

## 1. Overview
This track implements the core Traffic Management resources and data sources for the Baffin Bay Terraform Provider, based on the `api-spec/traffic-mgmt.yml` OpenAPI specification. This includes support for HTTP Proxies, L4 Proxies, and Routed DSR configurations, along with certificate and security management.

## 2. Goals
- Implement a unified resource for Traffic Configurations.
- Support core supporting resources (Certificates, CA Bundles, Custom Pages).
- Provide data sources for retrieving existing configurations.
- Ensure full support for Terraform Import for all implemented resources.

## 3. Requirements

### 3.1 Resource: `baffinbay_traffic_config`
- **Unified Logic:** Use a single resource with a `type` attribute (enum: `httpProxy`, `l4Proxy`, `routedDsr`).
- **Schema Design:**
  - **Flattened Attributes:** Nested structures like `frontend`, `backend`, and `waf` will be flattened where practical (e.g., `frontend_port`, `backend_hosts`) to improve the HCL authoring experience.
  - **Type-Specific Validation:** Logic to ensure only valid attributes are set for each `type`.
- **Import Support:** Importable via UUID (e.g., `terraform import baffinbay_traffic_config.example <uuid>`).

### 3.2 Supporting Resources
- **`baffinbay_certificate`:** Manage imported PEM or Let's Encrypt certificates.
- **`baffinbay_ca_bundle`:** Manage CA bundles for backend TLS verification.
- **`baffinbay_custom_page`:** Upload and manage HTML custom pages.
- **`baffinbay_ip_allocation`:** Manage IP allocations for tenants/sub-tenants.

### 3.3 Data Sources
- **`baffinbay_traffic_configs`:** List all traffic configurations.
- **`baffinbay_certificates`:** List all available certificates.
- **`baffinbay_ip_sources`:** Retrieve TPC IP sources.

### 3.4 Import Strategy
- All resources must support `ImportState`.
- Primary import ID will be the resource UUID.
- Tenant context is assumed to be provided by the provider-level configuration.

## 4. Acceptance Criteria
- Resources can be created, updated, and deleted successfully.
- `terraform import` works for all resources using their UUID.
- Documentation for all new resources/data sources is automatically generated via `tfplugindocs`.
- Unit and acceptance tests cover the main CRUD flows for each resource type.

## 5. Out of Scope
- Implementation of advanced WAF rule custom logic (beyond basic schema mapping).
- Integration with external secret managers (e.g., Vault) for certificates (handled via standard Terraform attributes).
