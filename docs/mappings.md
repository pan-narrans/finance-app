# Mappings and Scoring Logic

A core feature of the Finance App is its ability to deterministically map messy input (bank statements, manual chat messages) into clean, structured Ledger transactions. This is handled by the `MappingService` in the Domain layer.

## Keyword Resolution

The system uses three main mapping types defined in `mappings.json`:

1.  **Account Mappings**: Maps description keywords to expense/income accounts.
    - *Logic*: Iterates through keywords sorted by length (descending). The first keyword found as a substring in the input wins.
    - *Example*: `"AMAZON MARKETPLACE"` is checked before `"AMAZON"`.

2.  **Source Mappings**: Maps manual entry prefixes to asset/income accounts.
    - *Example*: `"cash 10 coffee"` -> `cash` maps to `Assets:Cash`.

3.  **Description Cleaning**: Strips technical "junk" from bank descriptions.
    - *Example*: `"TARJETA Apple Pay: Mercadona"` -> `"Mercadona"`.

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
