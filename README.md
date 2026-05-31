# Finance App

A toolset designed to reduce friction in personal finance management, acting as a bridge to [Ledger CLI](https://ledger-cli.org/).

## Project Context
This project aims to automate and simplify the process of adding expenses to a `.ledger` file, which serves as the primary database for all financial records. It is intended to complement [Paisa](https://paisa.fyi/), a GUI for Ledger CLI.

## Core Objective
Reduce friction when recording daily expenses by providing multiple input channels that automatically format data for the ledger.

## Usage

The application primarily operates through a **Telegram Bot** that interfaces with your Ledger file.

### Telegram Bot Setup

To use the bot in **Telegram Groups**, you must configure the following:

1.  **Group Privacy:** In [@BotFather](https://t.me/botfather), select your bot and go to `Bot Settings` > `Group Privacy` > **Turn OFF**.
    - This allows the bot to read messages in the group to detect commands, mentions, and replies.
2.  **Permissions:** Ensure the bot has "Read Messages" permission in the group settings.

### 1. Manual Entry (Chat)
Send a message to the bot in the following format:
`[source] <amount> <description/target>`

- **Format:** `cash 10.50 coffee` or `12.50 dinner`
- **Source (Optional):** Keywords like `cash`, `visa`, or names map to specific asset accounts. If omitted, the default asset account is used.
- **Amount:** Supports both `.` and `,` as decimal separators.
- **Description:** Anything after the amount is used as the payee/description.

**The Flow:**
1. Send the command.
2. The bot parses the text and shows a **Draft Transaction**.
3. Use **Inline Buttons** to:
   - ✅ **Confirm:** Save directly to the `.ledger` file.
   - ✏️ **Edit Target/Source:** Search for a specific account if the suggestion is wrong.
   - ❌ **Discard:** Cancel the entry.

### 2. Bank Statement Import
Upload a supported bank export file (CSV/XLS) directly to the chat.

- **Auto-Detection:** The bot identifies the bank (e.g., OpenBank, ImaginBank) by the filename.
- **Deduplication:** Uses stable MD5 hashing of transaction data to prevent duplicate entries in your ledger.
- **Summary:** After processing, the bot returns a summary of added, updated, and failed rows.

### 3. Financial Reports
Get segmented overviews of your spending and income with automatic date ranges.

- **Commands:**
  - `/report`: Summary for **this month** (01/MM/YYYY - Today).
  - `/report last`: Summary for **last month** (01/MM/YYYY - 31/MM/YYYY).
- **Output:** Hierarchical balance reports, segmented into separate blocks (e.g., **Expenses 01/05/2026 - 31/05/2026**) based on your `RootAccounts` configuration.
- **Source:** Direct execution of `ledger balance` with period and account filters.

### 4. Configuration & Mappings
The bot's behavior is driven by two JSON files in the `config/` directory:

- **`config.json`**: Global settings like `default_currency`, `ledger_alignment`, and default fallback accounts.
- **`mappings.json`**:
  - `accounts`: Maps keywords in descriptions or manual entry sources to ledger accounts (e.g., `"MERCADONA": "Expenses:Groceries"`, `"alex": "Assets:Cash:Alex"`).
  - `prefixes`: Junk text to strip from bank descriptions (e.g., `"TARJETA Apple Pay:"`).
  - `cards`: Maps card numbers found in descriptions to owners for metadata tracking.

## Architecture & Design Principles

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

## Technical Documentation
For deep dives into the system's design and logic, see the following documents:
- [Architecture](docs/architecture.md): Hexagonal design, layers, and data flow.
- [Mappings & Scoring](docs/mappings.md): Details on keyword resolution and search ranking algorithms.

> **Note:** If `godoc` is not installed, run `go install golang.org/x/tools/cmd/godoc@latest`.
