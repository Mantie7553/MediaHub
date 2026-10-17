#!/usr/bin/env bash
# Usage: deploy/check-media-mount.sh <media path>
#
# If the media path lives on a filesystem listed in /etc/fstab (e.g. a NAS share on /mnt/nas),
# make sure it is actually mounted before the stack starts. Otherwise Docker mounts the empty
# mount-point folder instead, bootstrap creates empty media folders in it, and the library looks
# wiped. For network filesystems it also makes docker.service wait for the mount at boot.
set -euo pipefail

path=${1:?media path required}
FSTAB=${FSTAB:-/etc/fstab}
SUDO=""
[ "$(id -u)" -ne 0 ] && SUDO="sudo"

# the fstab entry with the longest mount point that contains the media path
best="" type=""
while read -r _ mp fstype _; do
  [ "$mp" = "/" ] && continue
  case "${path%/}/" in
    "${mp%/}"/*) if [ ${#mp} -gt ${#best} ]; then best=$mp; type=$fstype; fi ;;
  esac
done < <(grep -vE '^[[:space:]]*(#|$)' "$FSTAB" 2>/dev/null || true)

[ -z "$best" ] && exit 0

if ! mountpoint -q "$best"; then
  echo "!! $best ($type) is in $FSTAB but not mounted, trying to mount it"
  if ! $SUDO mount "$best" || ! mountpoint -q "$best"; then
    echo "xx Could not mount $best. Not starting MediaHub: it would see an empty folder instead of your media." >&2
    echo "   Fix the mount (sudo mount -v $best shows why), then run this script again." >&2
    exit 1
  fi
fi
echo "==> Media folder $path is on $best ($type)"

case "$type" in
  cifs|smb3|nfs|nfs4|sshfs|fuse.sshfs)
    dropin=/etc/systemd/system/docker.service.d/wait-for-media.conf
    want=$(printf '[Unit]\nRequiresMountsFor=%s\n' "$best")
    if [ -d /run/systemd/system ] && [ "$(cat "$dropin" 2>/dev/null)" != "$want" ]; then
      $SUDO mkdir -p "$(dirname "$dropin")"
      printf '%s\n' "$want" | $SUDO tee "$dropin" >/dev/null
      $SUDO systemctl daemon-reload
      echo "==> Docker will now wait for $best to mount at boot"
    fi
    ;;
esac
