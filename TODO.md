# Project Roadmap (TODO.md)

This file tracks the upcoming tasks and development milestones for the Finance App.

## Phase 1: Preparation & Documentation
- [x] Create project `README.md`
- [x] Create `GEMINI.md` with architectural mandates
- [x] Initialize development environment (Go/Golang)
- [x] Create agents and skills for gemini to accelerate development

## Phase 1.5: Configuration & Infrastructure
- [x] Implement environment variable support (`.env`).
- [x] Configure `LEDGER_ROOT` for dynamic file paths.

## Phase 2: Core Module - Ledger CRUD
- [X] Implement `Create` (Add transaction to `.ledger`).
- [X] Implement `Read` (Fetch transactions from `.ledger`).
- [X] Implement `Update` (Safely modify existing entries).
- [X] Implement `Delete`.
- [X] Add unit tests with valid/invalid ledger entry samples.

## Phase 3: Input Integrations
- [ ] **Excel/CSV Parser:**
    - [X] CLI tool for manual import.
    - [ ] Hotfolder watcher for automated import (`LEDGER_ROOT/imports`).
    - [ ] Persistent logger for imports
    - [X] Bank identification via filename (e.g., `openbank.xlsx` -> Openbank parser).
    - [X] Analyze bank-specific CSV/XLSX formats:
      - [X] Openbank
      - [X] ImaginBank
    - [X] Create mapping logic to transform rows into ledger transactions.
- [X] Devise a way to univocally add a code to transactions. Transactions added via file/bot/aggregator should not duplicate but update.
- [ ] **Telegram Bot:**
    - [ ] Set up bot API integration.
    - [ ] Implement command parser for simple expense entry (e.g., `/expense 15 Lunch`).
- [ ] **Bank Aggregator API:**
  - [ ] Research possible API providers (e.g., Salt Edge, Nordigen, or local alternatives).

## Phase 4: Refinement
- [ ] Ensure full compatibility with Paisa GUI.
- [ ] Add logging and error reporting.
- [ ] Optimize the CRUD performance for large ledger files.
