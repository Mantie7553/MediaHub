#!/usr/bin/env bash
# Print the current quick tunnel address. It changes whenever the cloudflared-quick container
# restarts (reboot, docker restart), so run this again after one.
cd "$(dirname "$0")"
DOCKER="docker"
docker info >/dev/null 2>&1 || DOCKER="sudo docker"

for _ in $(seq 1 30); do
  url=$($DOCKER compose logs cloudflared-quick 2>/dev/null | grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' | tail -n 1)
  if [ -n "$url" ]; then
    echo "  Remote address (quick tunnel): $url"
    exit 0
  fi
  sleep 2
done
echo "  No quick tunnel address yet. Is the quicktunnel profile on? Check: $DOCKER compose logs cloudflared-quick" >&2
exit 1
