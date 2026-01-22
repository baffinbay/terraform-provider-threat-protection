# Implementation Plan - Project Documentation & Automation

## Phase 1: Developer Environment & Hygiene
- [x] Task: Environment Configuration Setup
    - [x] Sub-task: Create `.env.example` with placeholders for `BAFFINBAY_API_KEY` and `BAFFINBAY_API_URL`.
    - [x] Sub-task: Ensure `.env` is added to `.gitignore`.
- [x] Task: Update README.md (Setup & Overrides)
    - [x] Sub-task: Write "Prerequisites" section (Go, Terraform, etc.).
    - [x] Sub-task: Write "Local Development" section with build instructions.
    - [x] Sub-task: Add detailed "Developer Overrides" guide for `~/.terraformrc`.
    - [x] Sub-task: Document "Testing" procedures (Unit vs Acceptance with `TF_ACC=1`).

## Phase 2: Documentation Automation Setup
- [ ] Task: Install and Configure tfplugindocs
    - [ ] Sub-task: Add `tfplugindocs` dependency to `go.mod` (or installation instructions).
    - [ ] Sub-task: Create a `Makefile` or update `main.go` with `//go:generate` tags for documentation.
- [ ] Task: Scaffold Documentation Templates
    - [ ] Sub-task: Create `templates/index.md.tmpl` for provider-level documentation.
    - [ ] Sub-task: Ensure directory structure (`docs/`, `templates/`) exists.

## Phase 3: Generation and Verification
- [ ] Task: Generate Initial Documentation
    - [ ] Sub-task: Run `tfplugindocs` to generate `docs/index.md` and `docs/data-sources/ping.md`.
    - [ ] Sub-task: Verify the generated output matches Terraform Registry style.
- [ ] Task: Final Documentation Review
    - [ ] Sub-task: Review all generated files for clarity, broken links, and formatting.

## Phase 4: Finalization
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Developer Environment & Hygiene' (Protocol in workflow.md)
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Documentation Automation Setup' (Protocol in workflow.md)
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Generation and Verification' (Protocol in workflow.md)
