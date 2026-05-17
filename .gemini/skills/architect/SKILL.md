---
name: architect
description: Enforces Hexagonal Architecture and system design principles. Use when planning new modules, refactoring layers, or updating technical documentation (docs/architecture.md).
---

# Architect

Ensure system integrity by enforcing Hexagonal Architecture (Ports and Adapters) and the inward-pointing Dependency Rule.

## Architectural Source of Truth

- The **`.ledger` file** is the primary and only source of truth (database).
- All operations (CRUD) must maintain the integrity and format required by [Ledger CLI](https://ledger-cli.org/).
- The application acts as a middle-layer to facilitate data entry into this file.

## Core Directives: Hexagonal Architecture

1.  **Strict Isolation**: Use Hexagonal Architecture to decouple core business logic from external dependencies (Ledger CLI, Telegram, Excel).
2.  **Dependency Rule**: Dependencies must always point inwards. The `Domain` and `App` layers must not depend on `Adapters`.
3.  **Package Structure**:
    - `internal/domain`: Pure entities and business rules (no external imports).
    - `internal/app`: Use cases and Ports (Interfaces).
    - `internal/adapters/primary`: Driving adapters (Telegram, CLI).
    - `internal/adapters/secondary`: Driven adapters (Ledger file I/O, Excel parsers).
4.  **Ledger Consistency**: Any code modifying the ledger must ensure transactions follow the plain-text format:
    ```ledger
    YYYY/MM/DD Description
        Account:Expense    $Amount
        Account:Asset
    ```
5.  **Modular Inputs**: Each input method must be implemented as a Primary Adapter, decoupled from the core CRUD logic.
6.  **No Database Redundancy**: Avoid creating parallel databases or caches that could get out of sync with the `.ledger` file.
7.  **Validation**: Every entry must be validated against basic ledger syntax before writing to the file.

## Documentation Responsibility

- Maintain `docs/architecture.md` and keep the Mermaid diagrams in sync with the actual code structure.
