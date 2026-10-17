#!/bin/sh
set -e

# YouTube changes often enough that an old yt-dlp stops working, so update it on every start.
# A failed update (offline, rate limited) keeps the version baked into the image.
if [ "${YTDLP_AUTO_UPDATE:-true}" = "true" ]; then
  "$YTDLP_PATH" -U >/dev/null 2>&1 || echo "yt-dlp update failed, using the bundled version"
fi

# transcodes and yt-dlp temp files go on the media disk, not the container layer
if [ -n "$MEDIA_ROOT" ]; then
  mkdir -p "$MEDIA_ROOT/temp"
  export TMPDIR="$MEDIA_ROOT/temp"
fi

# run as the same user as the *arr containers so everything on the media disk has one owner.
# --keep-groups keeps group_add groups such as render for /dev/dri
PUID="${PUID:-1000}"
PGID="${PGID:-1000}"
chown "$PUID:$PGID" /data
[ -n "$MEDIA_ROOT" ] && chown "$PUID:$PGID" "$MEDIA_ROOT/temp" 2>/dev/null || true
exec setpriv --reuid="$PUID" --regid="$PGID" --keep-groups "$@"
