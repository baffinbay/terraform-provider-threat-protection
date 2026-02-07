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

To avoid committing sensitive information, use a `.env` file for your API credentials.

1.  Copy the example file:
    ```bash
    cp .env.example .env
    ```
2.  Edit `.env` and set your credentials. You can use either:
    *   **OIDC (Recommended):** `BAFFINBAY_CLIENT_ID` and `BAFFINBAY_CLIENT_SECRET`
    *   **API Key (Legacy):** `BAFFINBAY_API_KEY`
3.  Source the file before running Terraform commands:
    ```bash
    export $(cat .env | xargs)
    ```

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