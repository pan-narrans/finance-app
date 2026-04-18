---
name: ledger-format
description: Ensures all generated or modified transactions follow the Ledger CLI plain-text format. Use when creating, updating, or validating entries in a .ledger file.
---

# Ledger Format

Ensure all transactions follow these precise specifications for the Ledger CLI format to maintain compatibility with `ledger-cli` and `Paisa`.

## 1. Basic Transaction Structure

A standard transaction consists of a header and at least two indented postings that sum to zero.

```ledger
YYYY/MM/DD [Status] [(Code)] Payee
    Account:Expense                 $Amount
    Account:Asset                  -$Amount
```

### Header Components
- **Date:** `YYYY/MM/DD` (recommended) or `YYYY-MM-DD`.
- **Status (Optional):** `*` for cleared, `!` for pending.
- **Code (Optional):** Reference number in parentheses, e.g., `(123)`.
- **Payee:** Description of the transaction.

### Postings Components
- **Indentation:** Must be indented by at least four spaces.
- **Account Name:** Hierarchical names separated by colons (e.g., `Assets:Bank:Checking`).
- **Amount:** Separated from the account name by **at least two spaces**.
- **Commodity:** Symbol (e.g., `$`, `€`) or name (e.g., `USD`, `AAPL`).

## 2. Formatting Rules

- **Automatic Balancing:** One posting can omit the amount; Ledger calculates it automatically.
- **Comments:** Use `;` for comments. Can be on their own line or at the end of a posting.
- **Multi-line Payee:** Use `;` on the next line if the description needs to be longer.
- **Prices/Conversions:** Use `@` for per-unit price or `@@` for total price.
  ```ledger
  2024/04/02 Stock Purchase
      Assets:Brokerage             10 AAPL @ $150.00
      Assets:Checking             -$1500.00
  ```

## 3. Best Practices for This Project

- **Hierarchy:** Always use full account paths (e.g., `Expenses:Personal:Dining`).
- **Precision:** Ensure at least two spaces exist between account names and amounts.
- **Validation:** Before final write, ensure the transaction balances to zero (unless using virtual postings).
- **Metadata:** Use tags for extra context: `; :Category: Food:`

## 4. Verification
Verify generated format using `ledger-tools` commands (e.g., `ledger stats`) before final commit.
