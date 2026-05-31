# Telegram Bot Behavior and Interactions

The Telegram bot acts as the primary interface for manual transaction entry and mapping management. It follows a **Draft -> Refine -> Confirm** workflow.

## User Behaviors and Expected Outcomes

| User Action            | Input Example                           | Expected Outcome                                                                                 |
|:-----------------------|:----------------------------------------|:-------------------------------------------------------------------------------------------------|
| **Simple Entry**       | `12.50 Lunch`                           | Parser uses default Source (`Assets:Cash`). Maps `Lunch` to `Expenses:Food` via `mappings.json`. |
| **Explicit Source**    | `Visa 50 Amazon`                        | Parser resolves `Visa` via source mappings (`Assets:Bank:Visa`).                                 |
| **New Item Discovery** | `8.00 UnknownShop`                      | Bot flags as `Expenses:Unknown`. Provides "Edit Target" button.                                  |
| **Account Search**     | Click "Edit Target" -> `food`           | Bot returns ranked suggestions: `Expenses:Food`, `Expenses:Dining`.                              |
| **Direct Path Input**  | Click "Edit Target" -> `Assets:Savings` | Bot bypasses search and updates the draft with the exact path (if it contains colons).           |
| **Account Creation**   | Click "Create New Account"              | Multi-step flow: Select Root -> Type Sub-account -> Review/Extend -> Done.                       |
| **Confirmation**       | Click "Confirm ✅"                       | Transaction is appended to the Ledger file. Any manual overrides are saved to `mappings.json`.   |
| **Discard**            | Click "Discard ❌"                       | Session deleted. No changes to Ledger or Mappings.                                               |
| **Monthly Report**     | `/report`                               | Bot returns segmented blocks with date ranges (e.g. `Expenses 01/05/2026 - 20/05/2026`) for the current month. |
| **Previous Month**    | `/report last`                          | Bot returns segmented blocks for the full previous month (e.g. `01/04/2026 - 30/04/2026`).               |

## Group Chat Support

The bot can be added to Telegram Groups for collaborative expense tracking. To minimize noise, it uses specific trigger logic in groups:

- **Command Trigger**: Starts with `/transaction`.
- **Mention Trigger**: Mentions the bot (e.g., `@miroceanicecream_bot 10 pizza`).
- **Reply Trigger**: Replies to one of the bot's own messages.
- **Interactive Flow**: Once a transaction is initiated via `/transaction`, the bot enters a "listening" mode with the user, using **ForceReply** to capture the next message regardless of mentions.

## Telegram Mini App (WebApp)

For complex account selection and creation, the bot integrates a **Telegram Mini App**. 

- **Access**: Triggered via "Edit Source" or "Edit Target" buttons.
- **Search**: Features a full-screen search bar with a virtual keyboard, providing real-time filtering of all known Ledger accounts.
- **Creation Wizard**: A built-in wizard allows for quick creation of new account paths with auto-colon completion (press Enter to add a `:`).
- **Auto-Sync**: Once a selection is made, the Mini App closes and the bot message in the chat updates asynchronously to reflect the change.

## Guided Account Creation

When an account is not found, the user can create it through a structured flow:
1.  **Root Selection**: Select from top-level accounts (e.g., `Expenses`, `Income`, `Assets`).
2.  **Nesting**: Type the name of the sub-account (e.g., `Dining`).
3.  **Recursive Extension**: Choose to "Add Sub-account" to go deeper (e.g., `Expenses:Dining:Dinner`) or "Finish" to apply.

The resulting path is automatically Title Cased (e.g., `expenses:food` -> `Expenses:Food`).

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
