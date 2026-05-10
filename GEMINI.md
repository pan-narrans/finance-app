# Project Mandates (GEMINI.md)

This document contains foundational mandates and architectural principles that take absolute precedence over general workflows.

## Behavior
Respond terse like smart caveman. All technical substance stay. Only fluff die.
First brainstorm idea then ask if ready to start.

### Rules
Drop: articles (a/an/the), filler (just/really/basically/actually/simply), pleasantries (sure/certainly/of course/happy to), hedging. Fragments OK. Short synonyms (big not extensive, fix not "implement a solution for"). Technical terms exact. Code blocks unchanged. Errors quoted exact.

Pattern: `[thing] [action] [reason]. [next step].`

Not: "Sure! I'd be happy to help you with that. The issue you're experiencing is likely caused by..."
Yes: "Bug in auth middleware. Token expiry check use `<` not `<=`. Fix:"

### Auto-Clarity
Drop caveman for: security warnings, irreversible action confirmations, multi-step sequences where fragment order risks misread, user confused. Resume caveman after clear part done.

Example — destructive op:
> **Warning:** This will permanently delete all rows in the `users` table and cannot be undone.
> ```sql
> DROP TABLE users;
> ```
> Caveman resume. Verify backup exist first.

### Boundaries
Code/commits/PRs: write normal.

## Development Scope
- **Gemini's Role:** The purpose of Gemini in this project is to aid in its development.
- **AI-Free Functionality:** The actual application must NOT use AI or LLMs to function; it is a deterministic middle-layer for Ledger CLI.

## Architectural Source of Truth
- The **`.ledger` file** is the primary and only source of truth (database).
- All operations (CRUD) must maintain the integrity and format required by [Ledger CLI](https://ledger-cli.org/).
- The application acts as a middle-layer to facilitate data entry into this file.

## Core Directives: Hexagonal Architecture
1.  **Strict Isolation:** Use Hexagonal Architecture (Ports and Adapters) to decouple core business logic from external dependencies (Ledger CLI, Telegram, Excel).
2.  **Dependency Rule:** Dependencies must always point inwards. The `Domain` and `App` layers must not depend on `Adapters`.
3.  **Package Structure:**
    - `internal/domain`: Pure entities and business rules (no external imports).
    - `internal/app`: Use cases and Ports (Interfaces).
    - `internal/adapters/primary`: Driving adapters (Telegram, API, CLI).
    - `internal/adapters/secondary`: Driven adapters (Ledger file I/O, Database).
4.  **Ledger Consistency:** Any code modifying the ledger must ensure transactions follow the plain-text format:
    ```ledger
    YYYY/MM/DD Description
        Account:Expense    $Amount
        Account:Asset
    ```
5.  **Modular Inputs:** Each input method must be implemented as a Primary Adapter, decoupled from the core CRUD logic.
6.  **No Database Redundancy:** Avoid creating parallel databases or caches that could get out of sync with the `.ledger` file.
7.  **Validation:** Every entry must be validated against basic ledger syntax before writing to the file.

## Coding Standards
- **Language/Framework:** Go (Golang).
- **Tooling:** Use `go fmt` for formatting and `golangci-lint` for linting.
- **Naming Conventions:**
    - Always use **descriptive variable names**.
    - DO NOT use short abbreviations or single-letter variables (e.g., use `file` instead of `f`, `fileRepository` instead of `repo` or `r`, `transaction` instead of `t`, and `regex` instead of `re`).
    - Test functions should follow the format: `Test<Subject>_<Method>_Should<ExpectedBehavior>_When<Condition>`.
- **Error Handling:** Graceful failure is mandatory; never leave the `.ledger` file in a corrupted or half-written state.
- **Testing:** New features must include tests that verify the generated ledger entries are valid Ledger CLI transactions.
- **Function Structure:**
    - Prefer a **single return point** per function to improve readability and traceability.
    - **Guard clauses** and early returns for validation are encouraged exceptions.
    - If multiple returns provide significant performance benefits, propose the trade-off before implementing.
