# Mappings and Scoring Logic

A core feature of the Finance App is its ability to deterministically map messy input (bank statements, manual chat messages) into clean, structured Ledger transactions. This is handled by the `MappingService` in the Domain layer.

## Keyword Resolution

The system uses two main mapping types defined in `mappings.json`:

1.  **Account Mappings**: Maps description keywords or source keywords to specific ledger accounts (Assets, Liabilities, Expenses, Income).
    - *Logic*: Iterates through keywords sorted by length (descending). The first keyword found as a substring in the input wins.
    - *Example*: `"AMAZON MARKETPLACE"` is checked before `"AMAZON"`.
    - **Collision Warning**: Since sources and targets share the same mapping pool, avoid very short keywords (e.g., "a", "c") that might appear accidentally in descriptions. Use descriptive keywords like "cash" or "visa" to ensure deterministic resolution.

2.  **Description Cleaning**: Strips technical "junk" from bank descriptions.
    - *Example*: `"TARJETA Apple Pay: Mercadona"` -> `"Mercadona"`.

## Dynamic Account Discovery

To improve suggestion accuracy without manual configuration, the system automatically discovers existing accounts directly from your `.ledger` file.

1.  **Extraction**: The system executes the `ledger accounts` command via the Ledger CLI. This command automatically handles `include` directives and traverses your entire ledger structure.
2.  **Synchronization**: Discovered accounts are synchronized with the `MappingService` during application startup and whenever the ledger file is modified.
3.  **Limitations**:
    - Only accounts with at least one transaction are discovered by the CLI. Declared but unused accounts (using the `account` keyword) are not listed.
    - If the ledger file has syntax errors, the discovery process will fail, and a warning will be logged.
4.  **Scoring Integration**: These accounts are included in the search scoring algorithm, ensuring that your existing ledger structure is always available for autocomplete suggestions even without explicit mappings in `mappings.json`.

## Search Scoring Algorithm

When an account cannot be resolved automatically, the system provides ranked suggestions. The score for each account is calculated based on the following rules:

| Condition | Points |
| :--- | :--- |
| **All Tokens Match** | Mandatory for non-zero score |
| **Token Length** | +Length of each matching token |
| **Exact Substring** | +100 points (full query matches exactly) |
| **Prefix Match** | +50 points (account starts with full query) |
| **Account Name Match** | Base score |
| **Mapping Key Match** | Base score - 1 (penalized to favor direct names) |

### Example
Query: `"Exp Food"`
- `Expenses:Food` -> High score (Prefix + Substring)
- `Expenses:Personal:Food` -> Medium score (Substring)
- `Income:Salary` -> 0 score (Missing tokens)

## Deduplication (Hashing)

To prevent duplicate entries when importing files multiple times, the system generates a stable `ID` in the metadata:

1.  **Bank Imports**: Uses the MD5 hash of the "Balance" field (if available) or unique row data.
2.  **Manual Bot Entry**: Uses a hash of the timestamp during draft creation to ensure each chat message creates a unique transaction intent.

The `Transaction.GenerateCode()` method then creates a 16-character SHA-256 prefix based on the Date, Description, and Postings to serve as the stable identifier in the ledger file.

## Learning Mechanism (Persistence)

The system can "learn" from user overrides to improve future auto-mapping accuracy. This logic resides in `domain.MappingData.Learn`.

1.  **Target Learning**: When a user overrides the expense/income account, the system maps the **UPPERCASE** description to the selected account.
2.  **Source Learning**: When a user overrides a source keyword (e.g., "cash"), the system maps the **UPPERCASE** keyword to the selected asset/liability account.

Learned mappings are persisted back to `mappings.json` via the `ConfigurationUseCase`.
