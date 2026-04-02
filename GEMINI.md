# Project Mandates (GEMINI.md)

This document contains foundational mandates and architectural principles that take absolute precedence over general workflows.

## Development Scope
- **Gemini's Role:** The purpose of Gemini in this project is to aid in its development.
- **AI-Free Functionality:** The actual application must NOT use AI or LLMs to function; it is a deterministic middle-layer for Ledger CLI.

## Architectural Source of Truth
- The **`.ledger` file** is the primary and only source of truth (database).
- All operations (CRUD) must maintain the integrity and format required by [Ledger CLI](https://ledger-cli.org/).
- The application acts as a middle-layer to facilitate data entry into this file.

## Core Directives
1. **Ledger Consistency:** Any code modifying the ledger must ensure transactions follow the plain-text format:
   ```ledger
   YYYY/MM/DD Description
       Account:Expense    $Amount
       Account:Asset
   ```
2. **Modular Inputs:** Each input method (Telegram, Excel Parser, API) must be decoupled from the core Ledger CRUD logic.
3. **No Database Redundancy:** Avoid creating parallel databases or caches that could get out of sync with the `.ledger` file.
4. **Validation:** Every entry must be validated against basic ledger syntax before writing to the file.

## Coding Standards
- **Language/Framework:** Go (Golang).
- **Tooling:** Use `go fmt` for formatting and `golangci-lint` for linting.
- **Error Handling:** Graceful failure is mandatory; never leave the `.ledger` file in a corrupted or half-written state.
- **Testing:** New features must include tests that verify the generated ledger entries are valid Ledger CLI transactions.
