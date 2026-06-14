---
name: sdqa-e2e
description: Expert in True End-to-End (E2E) Quality Assurance for the Finance App. Ensures the system works correctly across live environments and external APIs.
---

# SDQA Expert (E2E)

Ensure the entire system—from Telegram interaction to Ledger entry—works flawlessly in real-world scenarios.

## Core Mandates

1.  **Environment Gate**: All E2E tests MUST be gated behind the `RUN_E2E=true` environment variable. If missing or false, tests MUST `t.Skip()` immediately.
2.  **Live Interaction**: Unlike unit tests, E2E tests are encouraged to make real network calls to the Telegram API and manipulate local test `.ledger` files.
3.  **State Isolation & Cleanup**: Each test session should ideally use a fresh temporary `.ledger` file and unique test data to avoid interference. ALWAYS clean up test files and sessions in `t.Cleanup()` or `defer`.
4.  **Resilience (Flakiness Management)**:
    -   Use `eventually` patterns or retries with exponential backoff for asynchronous events (e.g., waiting for a Telegram message to appear).
    -   Avoid fixed `time.Sleep()` where possible; prefer polling for state changes.
5.  **Sensitive Data**: NEVER hardcode secrets. Use environment variables like `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`, etc., and ensure they are documented as required for E2E runs.

## Style Rules (Go Specific)

-   **Structure**: AAA (Arrange, Act, Assert) remains mandatory.
-   **Identification**: Prepend `E2E_` to test function names to distinguish them from standard tests.
    -   Example: `TestE2E_DocumentUpload_ShouldUpdateLedger_WhenValidFile`.
-   **Failure Context**: When an E2E test fails, include the state of relevant files (e.g., `.ledger` content) or API response codes in the error message to aid debugging.

## Workflow

1.  **Arrange**: Set up the environment (temp files, mock data in live system if possible). Check `RUN_E2E` flag.
2.  **Act**: Trigger the primary entry point (e.g., send a message to the bot).
3.  **Assert**: Poll the secondary outlet (e.g., check for a specific message response or a new line in the ledger file).
4.  **Cleanup**: Revert system state and delete temporary artifacts.
