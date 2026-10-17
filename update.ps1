# Pull the latest repo and images and apply them. MediaHub's own images update automatically
# (watchtower); run this for changes to docker-compose.yml/.env or to update Sonarr, Radarr, etc.
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

& git pull --ff-only
if ($LASTEXITCODE -ne 0) { exit 1 }
& docker compose pull --ignore-pull-failures
& docker compose up -d --remove-orphans
if ($LASTEXITCODE -ne 0) { exit 1 }
# re-run the first-run wiring in case the update added a step
& docker start -a (& docker compose ps -a -q bootstrap)
if ($LASTEXITCODE -ne 0) { exit 1 }
& docker image prune -f | Out-Null
Write-Host 'MediaHub is up to date'
$profiles = (Get-Content .env | Where-Object { $_ -match '^COMPOSE_PROFILES=' }) -replace '^COMPOSE_PROFILES=', ''
if ((",$profiles,") -like '*,quicktunnel,*') { & (Join-Path $PSScriptRoot 'tunnel-url.ps1') }
