#!/usr/bin/env bash
# Adds secrets introduced after an install to an existing .env, leaving every other line alone.
# Run from the repo root by install.sh and update.sh
set -euo pipefail

add_secret() { # name, random bytes
  grep -qE "^$1=.+" .env && return
  # an empty "NAME=" line from an older .env.example is replaced, not duplicated
  sed -i "/^$1=/d" .env
  [ -z "$(tail -c1 .env)" ] || echo >> .env
  echo "$1=$(openssl rand -hex "$2")" >> .env
  echo "Added $1 to .env"
}

add_secret SUWAYOMI_PASSWORD 12
