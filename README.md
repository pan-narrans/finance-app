# Finance App

A toolset designed to reduce friction in personal finance management, acting as a bridge to [Ledger CLI](https://ledger-cli.org/).

## Project Context
This project aims to automate and simplify the process of adding expenses to a `.ledger` file, which serves as the primary database for all financial records. It is intended to complement [Paisa](https://paisa.fyi/), a GUI for Ledger CLI.

## Core Objective
Reduce friction when recording daily expenses by providing multiple input channels that automatically format data for the ledger.

## Planned Features / Ideas
- **Ledger CRUD:** A core module to interact with the `.ledger` file programmatically.
- **Telegram Bot:** For quick, on-the-go expense entry via chat.
- **Excel File Parser:** Custom parsers for bank-specific statement formats.
- **Bank Aggregator API:** Potential integration with banking APIs for automated syncing.

## Setup

1. **Environment Variables:**
   Copy the `.env.example` file to `.env` and fill in the required values:
   ```bash
   cp .env.example .env
   ```

2. **Configuration Properties:**
   - `LEDGER_ROOT`: The directory where your `.ledger` files are located (Default: `.`).
   - `CONFIG_ROOT`: The directory for application configuration (Default: `./config`).
   - `TELEGRAM_TOKEN`: The API token for your Telegram Bot.
   - `TELEGRAM_USER_IDS`: A comma-separated list of authorized Telegram User IDs.

## Architecture & Design Principles

The application is built following **Hexagonal Architecture (Ports and Adapters)** to ensure strict isolation between core business logic and external dependencies.

### Core Mandates
- **Single Source of Truth:** The `.ledger` file is the primary database. No redundant databases or caches are allowed.
- **AI-Free Core:** The application is a deterministic middle-layer; it does not use AI or LLMs for its core functionality.
- **Ledger Integrity:** All operations must maintain the strict formatting required by [Ledger CLI](https://ledger-cli.org/).
- **Validation:** Every entry is validated against ledger syntax before persistence.

### Hexagonal Architecture
1. **Strict Isolation:** Business logic is decoupled from external systems (Ledger CLI, Telegram, Excel).
2. **Dependency Rule:** Dependencies always point inwards. `Domain` and `App` layers never depend on `Adapters`.

### Package Structure
- `internal/domain`: Pure entities and business rules. No external imports.
- `internal/app`: Use cases and Ports (Interfaces).
- `internal/adapters/primary`: Driving adapters (e.g., Telegram Bot, CLI, API).
- `internal/adapters/secondary`: Driven adapters (e.g., Ledger File I/O).

### Development Standards
- **Language:** Go (Golang)
- **Tooling:** Uses `go fmt` for formatting and `golangci-lint` for linting.
- **Testing:** Mandatory unit tests for all features, ensuring generated entries are valid Ledger transactions.
- **Error Handling:** Graceful failure is required to prevent `.ledger` file corruption.

## Documentation
The project uses `godoc` for documentation. You can view it in the terminal or browser:
- **Terminal:** `go doc -all`
- **Browser:** Run `godoc -http=:6060` and navigate to `http://localhost:6060/pkg/github.com/a-perez/finance-app/`

> **Note:** If `godoc` is not installed, run `go install golang.org/x/tools/cmd/godoc@latest`.
