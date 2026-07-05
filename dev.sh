#!/bin/bash

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Load local environment variables from root .local/env if it exists
if [ -f "$PROJECT_ROOT/.local/env" ]; then
    echo "Loading config from $PROJECT_ROOT/.local/env..."
    while IFS= read -r line || [ -n "$line" ]; do
        if [[ ! "$line" =~ ^# ]] && [[ "$line" == *"="* ]]; then
            export "$line"
        fi
    done < "$PROJECT_ROOT/.local/env"
fi

# Also load .env from current directory if it exists
if [ -f "$SCRIPT_DIR/.env" ]; then
    echo "Loading config from $SCRIPT_DIR/.env..."
    while IFS= read -r line || [ -n "$line" ]; do
        if [[ ! "$line" =~ ^# ]] && [[ "$line" == *"="* ]]; then
            export "$line"
        fi
    done < "$SCRIPT_DIR/.env"
fi

echo "Starting Cloudflare tunnel..."

# Cleanup existing tunnel processes on this port if any
# Using a more targeted kill if possible, but pkill is reliable for dev
pkill -f "cloudflared tunnel" || true
rm -f "$SCRIPT_DIR/tunnel.log" "$SCRIPT_DIR/tunnel.pid"

if [ ! -z "$CLOUDFLARE_TUNNEL_NAME" ]; then
    echo "Using persistent tunnel: $CLOUDFLARE_TUNNEL_NAME"
    cloudflared tunnel run --url http://localhost:8080 "$CLOUDFLARE_TUNNEL_NAME" > "$SCRIPT_DIR/tunnel.log" 2>&1 &
else
    echo "Using ephemeral tunnel (no CLOUDFLARE_TUNNEL_NAME found)..."
    cloudflared tunnel --url http://localhost:8080 > "$SCRIPT_DIR/tunnel.log" 2>&1 &
fi

echo $! > "$SCRIPT_DIR/tunnel.pid"

echo "Waiting for tunnel URL..."
TUNNEL_URL=""
while true; do
    # Priority 1: If we have a persistent tunnel name AND a pre-configured BASE_URL, use it.
    if [ ! -z "$CLOUDFLARE_TUNNEL_NAME" ] && [ ! -z "$WEBAPP_BASE_URL" ]; then
        TUNNEL_URL=$WEBAPP_BASE_URL
        echo "Detected persistent configuration. Using URL: $TUNNEL_URL"
        break
    fi

    # Priority 2: Scrape the log for a new ephemeral URL (only if not in persistent mode or if URL is missing)
    if grep -q "trycloudflare.com" "$SCRIPT_DIR/tunnel.log" 2>/dev/null; then
        SCRAPED_URL=$(grep -o 'https://[-a-z0-9.]*trycloudflare.com' "$SCRIPT_DIR/tunnel.log" | head -n 1)
        if [ ! -z "$SCRAPED_URL" ]; then
            TUNNEL_URL=$SCRAPED_URL
            echo "Scraped new ephemeral URL: $TUNNEL_URL"
            break
        fi
    fi
    
    # Check for common Cloudflare error if tunnel fails to start
    if grep -q "error" "$SCRIPT_DIR/tunnel.log" 2>/dev/null; then
        echo "Error starting tunnel. Check tunnel.log"
        kill $(cat "$SCRIPT_DIR/tunnel.pid")
        exit 1
    fi
    sleep 1
done

echo "Final WebApp URL: $TUNNEL_URL"

# Update .env only if it exists
if [ -f "$SCRIPT_DIR/.env" ]; then
    echo "Updating $SCRIPT_DIR/.env..."
    sed -i.bak "s|^WEBAPP_BASE_URL=.*|WEBAPP_BASE_URL=$TUNNEL_URL|" "$SCRIPT_DIR/.env" && rm "$SCRIPT_DIR/.env.bak"
fi

echo ""
echo "--- WEBAPP CONFIG ---"
echo "URL: $TUNNEL_URL"
echo "----------------------"
echo ""

# Ensure cleanup on exit
trap 'kill $(cat "$SCRIPT_DIR/tunnel.pid") 2>/dev/null; rm "$SCRIPT_DIR/tunnel.pid" "$SCRIPT_DIR/tunnel.log"' EXIT

~/go/bin/air
