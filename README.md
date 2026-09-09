# Terraform Provider for Baffin Bay Threat Protection

This is the official Terraform provider for Baffin Bay Threat Protection, enabling you to manage your security configuration as code.

## Requirements

* [Terraform](https://www.terraform.io/downloads.html) >= 1.0
* [Go](https://golang.org/doc/install) >= 1.26.6

## Local Development

### Prerequisites

Ensure you have the following installed:
*   Go (1.26.6 or later)
*   Terraform (1.0 or later)
*   Make (optional, but recommended)

### Building the Provider

1.  Clone the repository:
    ```bash
    git clone https://github.com/baffinbay/terraform-provider-threat-protection
    cd terraform-provider-threat-protection
    ```

2.  Build the provider:
    ```bash
    go build -o terraform-provider-threat-protection
    ```

### API Specification

The provider is built against the Baffin Bay Traffic Management API. The latest OpenAPI specification can always be found at: [https://docs.baffinbay.com/openapi/traffic-mgmt.yml](https://docs.baffinbay.com/openapi/traffic-mgmt.yml)

### Developer Overrides

To test the provider locally without publishing it to the Terraform Registry, you can use the development overrides feature.

1.  Create or edit your `~/.terraformrc` (or `%APPDATA%\terraform.rc` on Windows) file.
2.  Add the following configuration, replacing `/path/to/repo` with the absolute path to your local repository build directory (where the binary resides):

    ```hcl
    provider_installation {

      dev_overrides {
          "baffinbay/threat-protection" = "/path/to/repo"
      }

      # For all other providers, install them directly from their origin provider
      # registries as normal. If you omit this, Terraform will _only_ use
      # the dev_overrides block, and so no other providers will be available.
      direct {}
    }
    ```

    **Note:** When using `dev_overrides`, `terraform init` will report a warning that it is using a local provider. This is expected.

### Adopting Existing Resources

Traffic configurations are managed with type-specific resources:

| API type | Terraform resource |
| --- | --- |
| `httpProxy` | `baffinbay_http_proxy` |
| `l4Proxy` | `baffinbay_l4_proxy` |
| `routedDsr` | `baffinbay_routed_dsr` |

The matching type-specific data sources are `baffinbay_http_proxy`, `baffinbay_l4_proxy`, and `baffinbay_routed_dsr`. The generic `baffinbay_traffic_config` and `baffinbay_traffic_configs` data sources have been removed.

When using Terraform config generation to adopt an existing traffic config, import it into the matching typed resource:

```hcl
import {
  to = baffinbay_http_proxy.example
  id = "00000000-0000-0000-0000-000000000000"
}
```

```bash
terraform plan -generate-config-out=generated.tf
```

Terraform may generate a resource containing:

```hcl
provider = threat-protection
```

Remove that generated line, or replace it with:

```hcl
provider = baffinbay
```

The `provider` meta-argument references the local provider name, not the registry source address. This provider is published as `baffinbay/threat-protection`, but customer configurations normally declare the local provider name as `baffinbay` so resources such as `baffinbay_http_proxy`, `baffinbay_l4_proxy`, and `baffinbay_routed_dsr` work without explicit provider references.

### Migrating from `baffinbay_traffic_config`

The generic `baffinbay_traffic_config` resource has been removed in favor of the type-specific resources. Before upgrading to this major version, move each existing state entry and update its configuration to the matching typed resource.

1.  Identify the traffic config type. You can inspect the current Terraform config, query the API, or use the generic data source in the previous provider version.
2.  Update the Terraform resource block:
    ```hcl
    # Before
    resource "baffinbay_traffic_config" "example" {
      type = "l4Proxy"
      # ...
    }

    # After
    resource "baffinbay_l4_proxy" "example" {
      # ...
    }
    ```
3.  Move the existing state address before applying:
    ```bash
    terraform state mv baffinbay_traffic_config.example baffinbay_l4_proxy.example
    ```

Use the corresponding target resource for each API type:

```bash
terraform state mv baffinbay_traffic_config.http baffinbay_http_proxy.http
terraform state mv baffinbay_traffic_config.l4 baffinbay_l4_proxy.l4
terraform state mv baffinbay_traffic_config.routed baffinbay_routed_dsr.routed
```

If the old state is not available, import the remote traffic config directly into the typed resource instead:

```bash
terraform import baffinbay_http_proxy.example 00000000-0000-0000-0000-000000000000
```

Run `terraform plan` after each migration and resolve any schema differences before applying changes.

### Environment Management

Set your OIDC credentials in the environment before running Terraform or `bb-tool`:

1.  Copy the example file and fill it in:
    ```bash
    cp .env.example .env
    ```
2.  Export the values (or use any dotenv loader of your choice):
    ```bash
    set -a && source .env && set +a
    ```

Required variables: `BAFFINBAY_CLIENT_ID`, `BAFFINBAY_CLIENT_SECRET`, `BAFFINBAY_TENANT_ID`.
Optional: `BAFFINBAY_API_URL`, `BAFFINBAY_OIDC_URL`, `BAFFINBAY_TOKEN_CACHE`.

### Token cache

Access tokens are minted via the OIDC `client_credentials` grant and persisted
to a local cache file so that subsequent invocations (and concurrent jobs in the
same CI matrix) reuse the same token until it expires.

*   Default path is under `os.UserConfigDir()` — typically
    `$XDG_CONFIG_HOME/baffinbay/token-cache-<hash>.json` on Linux,
    `~/Library/Application Support/baffinbay/...` on macOS, and
    `%AppData%\baffinbay\...` on Windows.
*   Override with the `BAFFINBAY_TOKEN_CACHE` environment variable.
*   The cache file is written atomically with mode `0600`; concurrent writers
    coordinate via `flock` so exactly one token is minted per expiry window.

### Testing

#### Unit Tests
Run unit tests to verify internal logic:
```bash
go test ./...
```

#### Acceptance Tests
Acceptance tests run against the actual Baffin Bay API. **Warning:** These tests may create real resources.

1.  Ensure your `.env` file is configured and exported.
2.  Set a non-production frontend IP for the HTTP proxy tests:
    ```bash
    export BAFFINBAY_ACC_FRONTEND_IPV4=<test-frontend-ip>
    ```
3.  Run tests with `TF_ACC=1`:
    ```bash
    TF_ACC=1 go test -v ./...
    ```

The full HTTP proxy acceptance test creates dummy certificates, a dummy CA
certificate, dummy custom pages, and a dummy IP list instead of referencing
existing production resources. By default it keeps the traffic config
`UNDEPLOYED`; set `BAFFINBAY_ACC_DEPLOYMENT_STATE=DEPLOYED` only when the test
environment is safe for deployment. You can also override
`BAFFINBAY_ACC_HOST`, `BAFFINBAY_ACC_BOT_HOST`, and
`BAFFINBAY_ACC_BACKEND_ADDRESS` for environment-specific routing.

## Documentation

The provider documentation is automatically generated using [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs). This ensures that the documentation is always in sync with the provider's schema.

*   The source templates are located in `templates/`.
*   The generated markdown is stored in `docs/`.
*   To update the documentation after changing the schema, run:
    ```bash
    go generate ./...
    ```
    or
    ```bash
    go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
    ```

As long as the code and schema descriptions are maintained, the documentation requires minimal manual intervention.

## Releasing

Releases are automated through `.github/workflows/release.yml`. To release a version, tag the release commit with a Semantic Version prefixed by `v` and push the tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The workflow runs the tests, verifies generated documentation, builds the supported platform archives, signs the checksum file, and creates the GitHub Release. The Terraform Registry release webhook then ingests the new `baffinbay/threat-protection` version automatically.

Published versions are immutable. If a release needs to be corrected, create a new version instead of moving the tag or replacing its artifacts.

## Troubleshooting Utility (bb-tool)

A CLI tool is included to help troubleshoot API connectivity and verify resource states independently of Terraform.

### Building bb-tool
```bash
go build -o bb-tool ./cmd/bb-tool
```

### Usage
Ensure your `.env` is set up correctly.

*   **Ping API:**
    ```bash
    ./bb-tool ping
    ```

*   **List Resources:**
    ```bash
    ./bb-tool ls --type traffic-config
    ./bb-tool ls --type cert
    ./bb-tool ls --type custom-page
    ```

*   **Delete Resource:**
    ```bash
    ./bb-tool rm --type traffic-config --id <uuid>
    ```

*   **Check OpenAPI Spec:**
    Validates the local API specification and implementation against the live OpenAPI definition.
    ```bash
    ./bb-tool spec-check
    ```
    You can also specify custom paths:
    ```bash
    ./bb-tool spec-check --remote-url https://example.com/spec.yml --local-path path/to/spec.yml
    ```
