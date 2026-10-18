#!/usr/bin/env bash
# MediaHub installer for Ubuntu/Debian. Installs Docker if needed, creates .env with fresh
# secrets, starts the stack and waits for the first-run wiring. Safe to run again.
set -euo pipefail
cd "$(dirname "$0")"

say()  { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!!\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31mxx\033[0m %s\n' "$*" >&2; exit 1; }
ask()  { local prompt=$1 default=${2:-} reply; read -r -p "$prompt${default:+ [$default]}: " reply; echo "${reply:-$default}"; }
secret() { openssl rand -hex "${1:-32}"; }
# a dead third-party repository (e.g. an old Sonarr one) makes apt-get update fail, which shouldn't
# stop the install; a real problem with Docker's repository still fails at apt-get install
apt_update() { $SUDO apt-get update || warn "apt-get update reported errors (often an old third-party repository in /etc/apt/sources.list.d); continuing"; }
esc() { printf '%s' "$1" | sed -e 's/[\\|&]/\\&/g'; }  # for sed replacement text

SUDO=""
[ "$(id -u)" -ne 0 ] && SUDO="sudo"

# ---- Docker ----
if ! command -v docker >/dev/null 2>&1 || ! docker compose version >/dev/null 2>&1; then
  say "Docker Engine with the compose plugin is not installed"
  [ "$(ask 'Install it now from Docker'"'"'s apt repository? (y/n)' y)" = "y" ] || die "Docker is required"
  . /etc/os-release
  apt_update
  $SUDO apt-get install -y ca-certificates curl openssl
  $SUDO install -m 0755 -d /etc/apt/keyrings
  $SUDO curl -fsSL "https://download.docker.com/linux/${ID}/gpg" -o /etc/apt/keyrings/docker.asc
  $SUDO chmod a+r /etc/apt/keyrings/docker.asc
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/${ID} ${VERSION_CODENAME} stable" \
    | $SUDO tee /etc/apt/sources.list.d/docker.list >/dev/null
  apt_update
  $SUDO apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  $SUDO systemctl enable --now docker
  [ -n "$SUDO" ] && $SUDO usermod -aG docker "$USER" && warn "Added $USER to the docker group (takes effect at next login)"
fi
command -v openssl >/dev/null 2>&1 || $SUDO apt-get install -y openssl

DOCKER="docker"
docker info >/dev/null 2>&1 || DOCKER="$SUDO docker"

# ---- .env ----
FIRST_RUN=false
if [ ! -f .env ]; then
  FIRST_RUN=true
  say "Creating .env"
  media_host=$(ask "Media folder on this machine (holds TV Shows, Movies, Music...)" /mnt/nas/Media)
  smb=n
  if [ ! -d "$media_host" ]; then
    smb=$(ask "$media_host does not exist. Is your media on an SMB share Docker should mount? (y/n)" n)
  fi
  if [ "$smb" = "y" ]; then
    smb_host=$(ask "SMB server (name or IP)")
    smb_share=$(ask "Share name (may include a sub folder, e.g. media/Media)")
    smb_user=$(ask "SMB username")
    read -r -s -p "SMB password: " smb_pass; echo
    media_root=/media
  else
    mkdir -p "$media_host" 2>/dev/null || $SUDO mkdir -p "$media_host"
    [ -w "$media_host" ] || warn "$media_host is not writable by $(id -un); the containers run as uid $(id -u)"
    # same path inside the containers, so paths stored by Sonarr/Radarr match the host
    media_root=$media_host
  fi
  public_url=$(ask "Address people will open MediaHub at" "http://$(hostname -I 2>/dev/null | awk '{print $1}')")
  echo "Remote access: paste a Cloudflare named tunnel token, type 'quick' for a free random"
  echo "trycloudflare.com address (changes on restart), or leave blank for home network only."
  tunnel_token=$(ask "Tunnel token / quick / blank" "")
  tz=$(cat /etc/timezone 2>/dev/null || timedatectl show -p Timezone --value 2>/dev/null || echo Etc/UTC)

  compose_file=docker-compose.yml
  render_gid=""
  # NVIDIA first: its driver also creates renderD128, which VAAPI can't use
  if command -v nvidia-smi >/dev/null 2>&1 && $DOCKER info 2>/dev/null | grep -qi "runtimes:.*nvidia"; then
    say "NVIDIA GPU found, enabling NVENC transcoding"
    compose_file="$compose_file:docker-compose.nvidia.yml"
  elif [ -e /dev/dri/renderD128 ]; then
    say "GPU found at /dev/dri/renderD128, enabling VAAPI transcoding"
    compose_file="$compose_file:docker-compose.gpu.yml"
    render_gid=$(stat -c %g /dev/dri/renderD128)
  elif lspci 2>/dev/null | grep -qi "vga.*nvidia"; then
    warn "NVIDIA GPU found but the NVIDIA driver or nvidia-container-toolkit is missing, transcoding on the CPU (see README, Hardware transcoding)"
  fi
  [ "$smb" = "y" ] && compose_file="$compose_file:docker-compose.smb.yml"
  profiles=""
  if [ "$tunnel_token" = "quick" ]; then
    profiles=quicktunnel
    tunnel_token=""
  elif [ -n "$tunnel_token" ]; then
    profiles=tunnel
  fi

  sed \
    -e "s|^MEDIA_HOST_PATH=.*|MEDIA_HOST_PATH=$(esc "${media_host}")|" \
    -e "s|^MEDIA_ROOT=.*|MEDIA_ROOT=$(esc "${media_root}")|" \
    -e "s|^SMB_HOST=.*|SMB_HOST=$(esc "${smb_host:-}")|" \
    -e "s|^SMB_SHARE=.*|SMB_SHARE=$(esc "${smb_share:-}")|" \
    -e "s|^SMB_USERNAME=.*|SMB_USERNAME=$(esc "${smb_user:-}")|" \
    -e "s|^SMB_PASSWORD=.*|SMB_PASSWORD=$(esc "${smb_pass:-}")|" \
    -e "s|^PUBLIC_URL=.*|PUBLIC_URL=$(esc "${public_url}")|" \
    -e "s|^TUNNEL_TOKEN=.*|TUNNEL_TOKEN=$(esc "${tunnel_token}")|" \
    -e "s|^COMPOSE_FILE=.*|COMPOSE_FILE=${compose_file}|" \
    -e "s|^COMPOSE_PROFILES=.*|COMPOSE_PROFILES=${profiles}|" \
    -e "s|^TZ=.*|TZ=$(esc "${tz}")|" \
    -e "s|^PUID=.*|PUID=$(id -u)|" \
    -e "s|^PGID=.*|PGID=$(id -g)|" \
    -e "s|^RENDER_GID=.*|RENDER_GID=${render_gid}|" \
    -e "s|^DB_PASSWORD=.*|DB_PASSWORD=$(secret)|" \
    -e "s|^JWT_SECRET=.*|JWT_SECRET=$(secret)|" \
    -e "s|^MEDIA_SIGNING_SECRET=.*|MEDIA_SIGNING_SECRET=$(secret)|" \
    -e "s|^WEBHOOK_SECRET=.*|WEBHOOK_SECRET=$(secret)|" \
    -e "s|^ADMIN_PASSWORD=.*|ADMIN_PASSWORD=$(secret 12)|" \
    -e "s|^ARR_PASSWORD=.*|ARR_PASSWORD=$(secret 12)|" \
    -e "s|^SONARR_API_KEY=.*|SONARR_API_KEY=$(secret 16)|" \
    -e "s|^RADARR_API_KEY=.*|RADARR_API_KEY=$(secret 16)|" \
    -e "s|^PROWLARR_API_KEY=.*|PROWLARR_API_KEY=$(secret 16)|" \
    .env.example > .env
  chmod 600 .env
else
  say "Using existing .env"
fi

# ---- start ----
# a NAS share that isn't mounted would leave the containers looking at an empty folder
get_media() { grep -E '^MEDIA_HOST_PATH=' .env | cut -d= -f2-; }
case "$(grep -E '^COMPOSE_FILE=' .env | cut -d= -f2-)" in
  *docker-compose.smb.yml*) ;;  # Docker mounts the share itself
  *) deploy/check-media-mount.sh "$(get_media)" || exit 1 ;;
esac

say "Pulling images"
$DOCKER compose pull --ignore-pull-failures || true
# compose builds a MediaHub image locally only if it could not be pulled (e.g. before CI has published one)
$DOCKER compose up -d

say "Waiting for first-run setup (bootstrap)"
# start -a re-runs it if an earlier run already finished, or attaches if up just started it
if ! $DOCKER start -a "$($DOCKER compose ps -a -q bootstrap)" > .bootstrap.log 2>&1; then
  cat .bootstrap.log
  die "bootstrap failed. Fix the problem above, then run ./install.sh again"
fi
grep -E 'added|warning|note' .bootstrap.log || true

get() { grep -E "^$1=" .env | cut -d= -f2-; }
echo
say "MediaHub is running"
echo "  MediaHub     $(get PUBLIC_URL)  (or http://localhost:$(get HTTP_PORT))"
echo "  qBittorrent  http://localhost:8080    Sonarr http://localhost:8989"
echo "  Radarr       http://localhost:7878    Prowlarr http://localhost:9696"
if $FIRST_RUN; then
  echo
  echo "  MediaHub login:           admin@example.com / $(get ADMIN_PASSWORD)"
  echo "  *arr + qBittorrent login: admin / $(get ARR_PASSWORD)"
  echo "  (also stored in .env; change the MediaHub password after logging in)"
fi
case ",$(get COMPOSE_PROFILES)," in
  *,quicktunnel,*) echo; ./tunnel-url.sh ;;
esac
echo
echo "  Next: add indexers in Prowlarr. Updates install themselves after each push to main."
