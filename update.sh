#!/usr/bin/env bash
# Pull the latest repo and images and apply them. MediaHub's own images update automatically
# (watchtower); run this for changes to docker-compose.yml/.env or to update Sonarr, Radarr, etc.
set -euo pipefail
cd "$(dirname "$0")"
DOCKER="docker"
docker info >/dev/null 2>&1 || DOCKER="sudo docker"

git pull --ff-only
$DOCKER compose pull --ignore-pull-failures || true
$DOCKER compose up -d --remove-orphans
# re-run the first-run wiring in case the update added a step
$DOCKER start -a "$($DOCKER compose ps -a -q bootstrap)" || exit 1
$DOCKER image prune -f >/dev/null
echo "MediaHub is up to date"
case ",$(grep -E '^COMPOSE_PROFILES=' .env | cut -d= -f2-)," in
  *,quicktunnel,*) ./tunnel-url.sh ;;
esac
