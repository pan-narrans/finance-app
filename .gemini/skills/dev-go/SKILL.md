---
name: dev-go
description: Enforces Go coding standards and idiomatic implementation rules. Use when writing, refactoring, or optimizing Go source code and tests.
---

# Dev-Go

Ensure high-quality, idiomatic Go implementation by enforcing local coding standards and project mandates.

## Coding Standards

- **Tooling**: Use `go fmt` for formatting and `golangci-lint` for linting.
- **Naming Conventions**:
    - Always use **descriptive variable names**.
    - DO NOT use short abbreviations or single-letter variables (e.g., use `file` instead of `f`, `fileRepository` instead of `repo`).
    - **Exception**: Method receivers (reference to self) MAY use a single letter to follow standard Go conventions (e.g., `func (t *Transaction) ...`). This is the **only** exception.
    - Test functions must follow: `Test<Subject>_<Method>_Should<ExpectedBehavior>_When<Condition>`.
- **Error Handling**: Graceful failure is mandatory. Never leave the system in a corrupted or half-written state. Use structured domain errors where appropriate.
- **Testing**: New features MUST include unit tests. Ensure generated ledger entries are valid Ledger CLI transactions.
- **Interface Checks**: All port implementations (Adapters and Application Services) must include a compile-time satisfaction check:
    ```go
    var _ ports.PortName = (*ImplementationStruct)(nil)
    ```

## Function Structure

- **Single Return Point**: Prefer a single return point per function to improve readability and traceability.
- **Guard Clauses**: Early returns for validation or error checks are encouraged exceptions.
- **Performance Trade-offs**: If multiple returns provide significant performance benefits, propose the trade-off before implementing.

## Documentation

- Follow GoDoc standards.
- Use `/* ... */` block comments for GoDoc when they exceed a single line or describe complex logic.
- Document internal helpers (unexported) if they contain non-trivial logic.

## Local Development Environment

- **Secrets**: Use `[REPO_ROOT]/.local/env` for all local secrets (Telegram Token, Tunnel Name).
- **Persistent Tunnels**: Configure `CLOUDFLARE_TUNNEL_NAME` and `WEBAPP_BASE_URL` in `.local/env` to use a permanent subdomain.
- **Workflow**: Run `./dev.sh` from any worktree; it will automatically load local environment variables and start the tunnel and hot-reloader.
