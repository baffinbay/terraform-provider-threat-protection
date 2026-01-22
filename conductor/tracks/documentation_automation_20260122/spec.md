# Specification: Project Documentation & Developer Onboarding

## 1. Overview
This track aims to create and maintain high-quality documentation for both developers and end-users of the Baffin Bay Terraform Provider. It ensures that the project is easy to set up for local development and establishes an automated documentation pipeline using `tfplugindocs`.

## 2. Goals
- Provide comprehensive local development instructions in `README.md`.
- Implement automated documentation generation using `tfplugindocs` to produce standard Terraform Registry documentation.
- Standardize the handling of secrets and local configuration for developers.

## 3. Requirements

### 3.1 README.md (Developer Onboarding)
The `README.md` must be updated to include:
- **Prerequisites:** List required tools (Go 1.21+, Terraform 1.0+, `tfplugindocs`, etc.).
- **Installation:** Instructions for building the provider binary.
- **Developer Overrides:** Step-by-step guide for setting up `~/.terraformrc` (or `terraform.rc`) to use the local provider build.
- **Environment Management:** Instructions on using a `.env` file for `BAFFINBAY_API_KEY` and `BAFFINBAY_API_URL` to avoid leaking secrets.
- **Testing:** Commands and environment requirements for running unit and acceptance tests.

### 3.2 Automated Documentation (User Reference)
- **Tooling:** Integrate `github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs` into the project.
- **Structure:** Generate standard documentation in the `docs/` directory (e.g., `docs/index.md`, `docs/data-sources/ping.md`).
- **Templates:** Create necessary templates if default ones are insufficient.
- **Workflow:** Add a makefile target or script (e.g., `go generate`) to easily regenerate docs.

## 4. Acceptance Criteria
- `README.md` exists and contains all specified developer setup sections.
- `tfplugindocs` is installed and configured.
- Running `go generate` (or the configured command) successfully produces `docs/` content.
- `docs/` contains documentation for the provider configuration and the existing `baffinbay_ping` data source.
- A template or example `.env.example` file is provided to guide developers.

## 5. Out of Scope
- Implementation of new resources or data sources.
- Publishing to the Terraform Registry (this track prepares the docs for it).
