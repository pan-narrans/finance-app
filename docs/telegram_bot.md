# Telegram Bot Behavior and Interactions

The Telegram bot acts as the primary interface for manual transaction entry and mapping management. It follows a **Draft -> Refine -> Confirm** workflow.

## User Behaviors and Expected Outcomes

| User Action | Input Example | Expected Outcome |
| :--- | :--- | :--- |
| **Simple Entry** | `12.50 Lunch` | Parser uses default Source (`Assets:Cash`). Maps `Lunch` to `Expenses:Food` via `mappings.json`. |
| **Explicit Source** | `Visa 50 Amazon` | Parser resolves `Visa` via source mappings (`Assets:Bank:Visa`). |
| **New Item Discovery**| `8.00 UnknownShop` | Bot flags as `Expenses:Unknown`. Provides "Edit Target" button. |
| **Account Search** | Click "Edit Target" -> `food` | Bot returns ranked suggestions: `Expenses:Food`, `Expenses:Dining`. |
| **Direct Path Input** | Click "Edit Target" -> `Assets:Savings` | Bot bypasses search and updates the draft with the exact path (if it contains colons). |
| **Account Creation** | Click "Create New Account" | Multi-step flow: Select Root (`Expenses`) -> Type Sub-account (`Gifts`) -> Review. |
| **Confirmation** | Click "Confirm ✅" | Transaction is appended to the Ledger file. Any manual overrides are saved to `mappings.json`. |
| **Discard** | Click "Discard ❌" | Session deleted. No changes to Ledger or Mappings. |

## Learning Mechanism (Mapping Persistence)

The bot "learns" from user corrections to reduce future friction.

1.  **Target Learning**: If you override an "Unknown" or incorrect expense account, the system maps the **UPPERCASE** description to the selected account.
    - *Example*: `Coffee` -> Selected `Expenses:Drinks`. Next time `coffee 5` will auto-map to `Expenses:Drinks`.
2.  **Source Learning**: If you override a source keyword, the system maps the **lowercase** keyword to the selected account.
    - *Example*: `Personal 10 Lunch` -> Selected `Assets:Checking:Personal`. Next time `personal 20 ...` will auto-map to that account.

## Security and Authorization

- **Whitelisting**: Only User IDs defined in the environment/config can interact with the bot.
- **Unauthorized Access**: The bot ignores messages from unauthorized IDs and logs the attempt. No data is exposed or modified.

## Session Management

- Sessions are transient and stored in memory.
- If a user sends a new transaction while a previous draft is active, the old draft is **overwritten**.
- Buttons on old messages will trigger a "Session expired" response if the session has been cleared or replaced.
