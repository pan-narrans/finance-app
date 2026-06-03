# Deployment & Reverse Proxy Guide

This document outlines the steps to deploy the Finance App and configure a reverse proxy to handle HTTPS and custom domains.

## Docker Deployment

The application is fully containerized. The `Dockerfile` uses a multi-stage build to package both the Go backend and the React Mini App.

### 1. Build the Image
```bash
docker build -t finance-app .
```

### 2. Run the Container
```bash
docker run -d \
  --name finance-app \
  -p 8080:8080 \
  -e TELEGRAM_TOKEN=your_token \
  -e TELEGRAM_USER_IDS=your_ids \
  -e WEBAPP_BASE_URL=https://foo.com \
  -v /path/to/ledger:/app/ledger \
  -v /path/to/config:/app/config \
  finance-app
```

---

## Reverse Proxy Setup (foo.com)

Telegram Mini Apps **require HTTPS**. You must use a reverse proxy to handle SSL termination.

### 1. Nginx Configuration
Add the following to your Nginx site configuration:

```nginx
server {
    server_name foo.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    listen 443 ssl; # managed by Certbot
    ssl_certificate /etc/letsencrypt/live/foo.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/foo.com/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;
}
```

### 2. Caddy Configuration (Easier)
If using Caddy, SSL is automatic:

```caddy
foo.com {
    reverse_proxy localhost:8080
}
```

### 3. Telegram Bot Setup
Once your domain is live and pointing to your server:
1. Update `.env` or Docker environment variables:
   - `WEBAPP_BASE_URL=https://foo.com`
2. Message **@BotFather** on Telegram:
   - Send `/setwebapp`.
   - Select your bot.
   - Send the URL: `https://foo.com`

---

## Troubleshooting

- **Blank Page**: Ensure `internal/adapters/primary/telegram/webapp/dist` exists in the container and the HTTP server logs show `[HTTP] GET /` requests.
- **Unauthorized API**: Verify that `TELEGRAM_TOKEN` is identical in the environment and what was used to generate the `initData` (handled automatically by the bot).
- **SSL Errors**: Mini Apps will not load on domains with invalid or self-signed certificates.
