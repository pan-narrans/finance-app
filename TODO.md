# Project Roadmap (TODO.md)

This file tracks the upcoming tasks and development milestones for the Finance App.

## Phase 1: Preparation & Documentation
- [x] Create project `README.md`
- [x] Create `GEMINI.md` with architectural mandates
- [x] Initialize development environment (Go/Golang)
- [x] Create agents and skills for gemini to accelerate development

## Phase 2: Core Module - Ledger CRUD
- [ ] Research and design the core Ledger parser/writer logic.
- [ ] Implement `Create` (Add transaction to `.ledger`).
- [ ] Implement `Read` (Fetch transactions from `.ledger`).
- [ ] Implement `Update/Delete` (Safely modify existing entries).
- [ ] Add unit tests with valid/invalid ledger entry samples.

## Phase 3: Input Integrations
- [ ] **Telegram Bot:**
  - [ ] Set up bot API integration.
  - [ ] Implement command parser for simple expense entry (e.g., `/expense 15 Lunch`).
- [ ] **Excel Parser:**
  - [ ] Analyze bank-specific CSV/Excel formats.
  - [ ] Create mapping logic to transform rows into ledger transactions.
- [ ] **Bank Aggregator API:**
  - [ ] Research possible API providers (e.g., Salt Edge, Nordigen, or local alternatives).

## Phase 4: Refinement
- [ ] Ensure full compatibility with Paisa GUI.
- [ ] Add logging and error reporting.
- [ ] Optimize the CRUD performance for large ledger files.
