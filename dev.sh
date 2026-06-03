#!/bin/bash
echo "Starting Cloudflare tunnel..."
rm -f tunnel.log tunnel.pid
# Use unbuffered output for cloudflared
cloudflared tunnel --url http://localhost:8080 > tunnel.log 2>&1 &
echo $! > tunnel.pid

echo "Waiting for tunnel URL..."
while true; do
    if grep -q "trycloudflare.com" tunnel.log 2>/dev/null; then
        TUNNEL_URL=$(grep -o 'https://[-a-z0-9.]*trycloudflare.com' tunnel.log | head -n 1)
        if [ ! -z "$TUNNEL_URL" ]; then
            break
        fi
    fi
    sleep 1
done

echo "Tunnel URL: $TUNNEL_URL"
echo "Updating .env..."
# Cross-platform sed compatibility
sed -i.bak "s|^WEBAPP_BASE_URL=.*|WEBAPP_BASE_URL=$TUNNEL_URL|" .env && rm .env.bak

echo ""
echo "--- BOTFATHER CONFIG ---"
echo "Use this URL for your WebApp:"
echo "$TUNNEL_URL"
echo "------------------------"
echo ""

# Ensure cleanup on exit
trap 'kill $(cat tunnel.pid) 2>/dev/null; rm tunnel.pid tunnel.log' EXIT

~/go/bin/air
