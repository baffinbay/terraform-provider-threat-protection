# Development Workflow

## 1. Context Analysis
- **Goal:** Understand the task and existing code before acting.
- **Action:**
  - Read `conductor/product.md` and relevant files in `conductor/tracks/`.
  - Use `search_file_content` or `glob` to find relevant code.
  - Read the code to understand the current implementation.

## 2. Test-Driven Development (TDD) Protocol
- **Goal:** Ensure correctness and design quality through the RED/GREEN/REFACTOR cycle.
- **Protocol:**
  1.  **RED:** Write a failing test case that defines the desired behavior or reproduces a bug. Verify it fails.
  2.  **GREEN:** Write the minimal amount of code necessary to pass the test. Verify it passes.
  3.  **REFACTOR:** Improve the code structure, readability, and performance without changing behavior. Verify tests still pass.
- **Requirement:** All new features and bug fixes MUST demonstrate this cycle.

## 3. Implementation
- **Goal:** Execute the plan with precision.
- **Action:**
  - Follow the TDD protocol above.
  - Adhere to the `conductor/code_styleguides/` and `conductor/tech-stack.md`.
  - Write idiomatic code.

## 4. Verification
- **Goal:** Ensure quality and correctness.
- **Action:**
  - **Test Coverage:** Maintain >80% code test coverage.
  - Run project-specific build and test commands (e.g., `go test ./...`).
  - Run linting and type-checking commands.

## 5. Documentation & Commit
- **Goal:** Track progress and maintain history.
- **Action:**
  - **Commit Frequency:** Commit changes **After Each Task**.
  - **Commit Message:** Use Conventional Commits (e.g., `feat: add user authentication`, `fix: resolve timeout issue`). Provide a brief summary in the message body.
  - **Task Tracking:** Use Git Notes to record detailed task summaries and status updates.
    - Command: `git notes add -m "Task Summary: [Details]"`