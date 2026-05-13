# Terraform Provider for Baffin Bay Threat Protection

This is the official Terraform provider for Baffin Bay Threat Protection, enabling you to manage your security configuration as code.

## Requirements

* [Terraform](https://www.terraform.io/downloads.html) >= 1.0
* [Go](https://golang.org/doc/install) >= 1.21

## Local Development

### Prerequisites

Ensure you have the following installed:
*   Go (1.21 or later)
*   Terraform (1.0 or later)
*   Make (optional, but recommended)

### Building the Provider

1.  Clone the repository:
    ```bash
    git clone https://github.com/baffinbay/terraform-provider-baffinbay
    cd terraform-provider-baffinbay
    ```

2.  Build the provider:
    ```bash
    go build -o terraform-provider-baffinbay
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
          "baffinbay/baffinbay" = "/path/to/repo"
      }

      # For all other providers, install them directly from their origin provider
      # registries as normal. If you omit this, Terraform will _only_ use
      # the dev_overrides block, and so no other providers will be available.
      direct {}
    }
    ```

    **Note:** When using `dev_overrides`, `terraform init` will report a warning that it is using a local provider. This is expected.

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

Required variables: `BAFFINBAY_CLIENT_ID`, `BAFFINBAY_CLIENT_SECRET`.
Optional: `BAFFINBAY_ACCOUNT_ID`, `BAFFINBAY_API_URL`, `BAFFINBAY_OIDC_URL`, `BAFFINBAY_TOKEN_CACHE`.

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
2.  Run tests with `TF_ACC=1`:
    ```bash
    TF_ACC=1 go test -v ./...
    ```

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

## Troubleshooting Utility (bb-tool)

A CLI tool is included to help troubleshoot API connectivity and verify resource states independently of Terraform.

### Building bb-tool
```bash
go build -o bb-tool ./cmd/bb-tool/main.go
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