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

## Technical Architecture
- **Language:** Go (Golang)
- **Primary Database:** Plain-text `.ledger` file (Ledger CLI format).
- **Core Logic:** CRUD operations on the ledger file.
- **Integrations:** Modular Hexagonal Architecture (Ports & Adapters).

## Documentation
The project uses `godoc` for documentation. You can view it in the terminal or browser:
- **Terminal:** `go doc -all`
- **Browser:** Run `godoc -http=:6060` and navigate to `http://localhost:6060/pkg/github.com/a-perez/finance-app/`

> **Note:** If `godoc` is not installed, run `go install golang.org/x/tools/cmd/godoc@latest`.
