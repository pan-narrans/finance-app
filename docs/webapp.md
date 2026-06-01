# Telegram Mini App (TMA) Technical Documentation

The Account Search Mini App is a React-based frontend integrated into the Telegram bot to provide a native search and selection experience.

## Architecture

The Mini App operates as a client-side single-page application (SPA) served by the Go backend's primary Telegram adapter.

### Component Diagram
```
Telegram Client <-> Bot (Long Polling)
      ^                |
      | WebApp API     |
      v                v
Mini App (React) <-> Go HTTP API (Internal)
```

## Implementation Details

### Frontend
- **Framework**: React 18 + TypeScript.
- **Build Tool**: Vite.
- **Location**: `internal/adapters/primary/telegram/webapp/`.
- **Theme Matching**: Uses Telegram CSS variables (`--tg-theme-*`) to match the user's client theme (dark/light).
- **Communication**: Uses `@twa-dev/sdk` for Telegram bridge functionality (Back button, Main button, Haptic feedback).

### Backend (Go)
- **HTTP Server**: A lightweight `net/http` server runs concurrently with the Telegram poller.
- **Endpoints**:
  - `GET /`: Serves static assets from `webapp/dist`.
  - `GET /api/accounts`: Returns a JSON list of all known accounts and root categories.
  - `POST /api/select`: Receives the selected account and `initData`.
- **Security**: The `/api/select` endpoint validates the `initData` HMAC-SHA256 signature using the `TELEGRAM_TOKEN` to ensure requests originate from an authorized Telegram session.

## UX Features

### Asynchronous Sync
When an account is selected in the Mini App:
1. The app POSTs the selection to `/api/select`.
2. The Go backend updates the user's `UserSession` state.
3. The backend calls `bot.Edit()` to update the transaction draft message in the Telegram chat.
4. The Mini App calls `WebApp.close()`.

This creates a seamless flow where the chat message is already updated by the time the user returns to the conversation.

### Account Creation Wizard
The Mini App features a multi-step creation wizard:
1. **Search Fallback**: If no results match, a "Create New Account" button appears.
2. **Root Selection**: User picks a top-level Ledger account.
3. **Hierarchy Construction**: User types the sub-account. Pressing **Enter** automatically appends a colon (`:`), allowing for rapid construction of deep hierarchies.
4. **Validation**: The backend automatically Title Cases the entire path before saving.

## Environment Configuration
- `WEBAPP_BASE_URL`: The public HTTPS URL where the WebApp is accessible (required for Telegram buttons).
- `HTTP_PORT`: The local port the server listens on (default: `8080`).

## Deployment
The Mini App is automatically built and packaged during the Docker build process. For detailed instructions on setting up a reverse proxy (e.g., Nginx, Caddy) for your domain, see the [Deployment Guide](deployment.md).
