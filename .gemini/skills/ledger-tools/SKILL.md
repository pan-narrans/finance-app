---
name: ledger-tools
description: Provides information on the built-in commands and utilities of Ledger CLI. Use this to avoid reinventing features that Ledger already provides (like balance checking, transaction printing, and validation).
---

# Ledger Tools

This skill guides you on how to leverage the existing `ledger-cli` commands to perform common financial operations and data validation.

## 1. Core Reporting Commands

Use these commands to extract data from your `.ledger` file without writing custom parsers for everything.

- **`ledger balance` (or `bal`)**: Shows current balances of all accounts.
  - *Example:* `ledger -f file.ledger bal ^Assets`
- **`ledger register` (or `reg`)**: Lists all transactions and a running total.
  - *Example:* `ledger -f file.ledger reg Expenses`
- **`ledger print`**: Outputs transactions in a standardized format. Useful for cleaning up a file or reformatting entries.
- **`ledger accounts`**: Lists every account name used in the file. Useful for auto-completion or validation logic.
- **`ledger payees`**: Lists all payee names used in the file.
- **`ledger stats`**: Provides a summary of the ledger (number of transactions, date range, etc.).

## 2. Validation & Formatting Flags

Use these flags to ensure the integrity of the data being read or written.

- **`-f <path>`**: Specifies the ledger file. **Required** if not using the default `LEDGER_FILE` environment variable.
- **`--check` / `--strict`**: Forces Ledger to validate account names and ensure the file is structurally sound.
- **`-S date`**: Sorts transactions by date (useful before printing back to the file).
- **`--output <file>`**: Directs the output to a specific file.

## 3. Advanced Querying

- **Period Filtering:** Use `--period "this month"` or `--begin "2024/01/01" --end "2024/01/31"`.
- **Payee Search:** Use `@PayeeName` to filter by a specific payee.
- **Account Regex:** Use `^Expenses:Food` to match specific account hierarchies.

## 4. Integration Strategy for This Project

- **Avoid manual parsing:** When possible, use `ledger print` or `ledger -f file.ledger csv` to get data in a machine-readable format rather than regex-ing the raw text file.
- **Validation:** Always run `ledger -f file.ledger stats` after writing a new transaction to ensure no syntax errors were introduced.
- **Account Verification:** Before adding a new transaction, use `ledger accounts` to check if the destination account already exists or if it's a new one.
