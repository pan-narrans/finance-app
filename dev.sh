#!/bin/bash

# Load local environment variables if they exist
if [ -f "../.local/env" ]; then
    export $(grep -v '^#' ../.local/env | xargs)
fi

echo "Starting Cloudflare tunnel..."
rm -f tunnel.log tunnel.pid

if [ ! -z "$CLOUDFLARE_TUNNEL_NAME" ]; then
    echo "Using persistent tunnel: $CLOUDFLARE_TUNNEL_NAME"
    # Persistent tunnels usually require a config file or --url flag
    # Assuming standard setup routing to localhost:8080
    cloudflared tunnel run --url http://localhost:8080 "$CLOUDFLARE_TUNNEL_NAME" > tunnel.log 2>&1 &
else
    echo "Using ephemeral tunnel..."
    cloudflared tunnel --url http://localhost:8080 > tunnel.log 2>&1 &
fi

echo $! > tunnel.pid

echo "Waiting for tunnel URL..."
while true; do
    if [ ! -z "$CLOUDFLARE_TUNNEL_NAME" ] && [ ! -z "$WEBAPP_BASE_URL" ]; then
        TUNNEL_URL=$WEBAPP_BASE_URL
        break
    fi

    if grep -q "trycloudflare.com" tunnel.log 2>/dev/null; then
        TUNNEL_URL=$(grep -o 'https://[-a-z0-9.]*trycloudflare.com' tunnel.log | head -n 1)
        if [ ! -z "$TUNNEL_URL" ]; then
            break
        fi
    fi
    
    # Check for common Cloudflare error if tunnel fails to start
    if grep -q "error" tunnel.log 2>/dev/null; then
        echo "Error starting tunnel. Check tunnel.log"
        kill $(cat tunnel.pid)
        exit 1
    fi
    sleep 1
done

echo "Tunnel URL: $TUNNEL_URL"

# Update .env only if it exists and we have a new URL
if [ -f ".env" ]; then
    echo "Updating .env..."
    sed -i.bak "s|^WEBAPP_BASE_URL=.*|WEBAPP_BASE_URL=$TUNNEL_URL|" .env && rm .env.bak
fi

echo ""
echo "--- WEBAPP CONFIG ---"
echo "URL: $TUNNEL_URL"
echo "----------------------"
echo ""

# Ensure cleanup on exit
trap 'kill $(cat tunnel.pid) 2>/dev/null; rm tunnel.pid tunnel.log' EXIT

~/go/bin/air
