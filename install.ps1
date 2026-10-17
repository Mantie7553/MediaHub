# MediaHub installer for Windows 10/11. Installs Docker Desktop if needed, creates .env with fresh
# secrets, starts the stack and waits for the first-run wiring. Safe to run again.
# Run from the repo folder:  powershell -ExecutionPolicy Bypass -File .\install.ps1
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

function Say($msg)  { Write-Host "==> $msg" -ForegroundColor Cyan }
function Warn($msg) { Write-Host "!!  $msg" -ForegroundColor Yellow }
function Die($msg)  { Write-Host "xx  $msg" -ForegroundColor Red; exit 1 }
function Ask($prompt, $default = '') {
    $suffix = if ($default) { " [$default]" } else { '' }
    $reply = Read-Host "$prompt$suffix"
    if ([string]::IsNullOrWhiteSpace($reply)) { return $default }
    return $reply.Trim()
}
function New-Secret([int]$bytes = 32) {
    $buf = New-Object byte[] $bytes
    [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($buf)
    return -join ($buf | ForEach-Object { $_.ToString('x2') })
}
function Test-DockerEngine {
    & docker info *> $null
    return $LASTEXITCODE -eq 0
}

# ---- Docker Desktop ----
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Say 'Docker Desktop is not installed'
    if ((Ask 'Install it now with winget? (y/n)' 'y') -ne 'y') { Die 'Docker Desktop is required' }
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
        Die 'winget is missing. Install "App Installer" from the Microsoft Store, or Docker Desktop from docker.com, then run this again'
    }
    & wsl --status *> $null
    if ($LASTEXITCODE -ne 0) {
        Say 'Installing WSL 2 (Docker Desktop runs on it). Windows will ask for admin rights'
        Start-Process wsl -ArgumentList '--install', '--no-distribution' -Verb RunAs -Wait
    }
    & winget install -e --id Docker.DockerDesktop --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -ne 0) { Die 'Docker Desktop install failed' }
    Warn 'Docker Desktop is installed. Restart Windows, open Docker Desktop once to accept its terms, then run .\install.ps1 again'
    exit 0
}

if (-not (Test-DockerEngine)) {
    $exe = Join-Path $env:ProgramFiles 'Docker\Docker\Docker Desktop.exe'
    if (-not (Test-Path $exe)) { Die 'Docker is installed but its engine is not running. Start Docker Desktop and run this again' }
    Say 'Starting Docker Desktop'
    Start-Process $exe
    $deadline = (Get-Date).AddMinutes(4)
    while (-not (Test-DockerEngine)) {
        if ((Get-Date) -gt $deadline) { Die 'Docker Desktop did not start. Open it, finish any first-run prompts, then run this again' }
        Start-Sleep -Seconds 5
    }
}

# start Docker Desktop at sign-in so MediaHub comes back after a reboot
foreach ($name in 'settings-store.json', 'settings.json') {
    $path = Join-Path $env:APPDATA "Docker\$name"
    if (Test-Path $path) {
        $settings = Get-Content $path -Raw | ConvertFrom-Json
        $key = if ($name -eq 'settings-store.json') { 'AutoStart' } else { 'autoStart' }
        if ($settings.$key -ne $true) {
            $settings | Add-Member -NotePropertyName $key -NotePropertyValue $true -Force
            [IO.File]::WriteAllText($path, ($settings | ConvertTo-Json -Depth 20))
            Say 'Docker Desktop will now start when you sign in'
        }
        break
    }
}

# ---- .env ----
$firstRun = $false
if (-not (Test-Path .env)) {
    $firstRun = $true
    Say 'Creating .env'
    $values = @{
        MEDIA_ROOT = '/media'; PUID = '1000'; PGID = '1000'; TZ = 'Etc/UTC'
        COMPOSE_FILE = 'docker-compose.yml'; COMPOSE_PATH_SEPARATOR = ':'
        DB_PASSWORD = New-Secret; JWT_SECRET = New-Secret; MEDIA_SIGNING_SECRET = New-Secret
        WEBHOOK_SECRET = New-Secret; ADMIN_PASSWORD = New-Secret 12; ARR_PASSWORD = New-Secret 12
        SONARR_API_KEY = New-Secret 16; RADARR_API_KEY = New-Secret 16; PROWLARR_API_KEY = New-Secret 16
    }

    $media = Ask 'Media folder on this PC (holds TV Shows, Movies, Music...), or a share like \\nas\media' 'D:\Media'
    if ($media -match '^\\\\([^\\]+)\\(.+)$') {
        # Docker Desktop can't see Windows network drives, so Docker mounts the share itself
        $values.SMB_HOST = $Matches[1]
        $values.SMB_SHARE = $Matches[2] -replace '\\', '/'
        $values.SMB_USERNAME = Ask 'SMB username'
        $cred = Read-Host 'SMB password' -AsSecureString
        $values.SMB_PASSWORD = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($cred))
        $values.COMPOSE_FILE = 'docker-compose.yml:docker-compose.smb.yml'
        $values.MEDIA_HOST_PATH = './media'
    } else {
        if ($media -match '^[A-Za-z]:$') { $media += '\' }
        if (-not (Test-Path $media)) { New-Item -ItemType Directory -Path $media | Out-Null }
        $values.MEDIA_HOST_PATH = (Resolve-Path $media).Path -replace '\\', '/'
    }

    $ip = (Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue |
           Where-Object { $_.PrefixOrigin -in 'Dhcp', 'Manual' } | Select-Object -First 1).IPAddress
    $values.PUBLIC_URL = Ask 'Address people will open MediaHub at' $(if ($ip) { "http://$ip" } else { 'http://localhost' })
    Write-Host "Remote access: paste a Cloudflare named tunnel token, type 'quick' for a free random"
    Write-Host 'trycloudflare.com address (changes on restart), or leave blank for home network only.'
    $tunnel = Ask 'Tunnel token / quick / blank'
    if ($tunnel -eq 'quick') { $values.COMPOSE_PROFILES = 'quicktunnel' }
    elseif ($tunnel) { $values.TUNNEL_TOKEN = $tunnel; $values.COMPOSE_PROFILES = 'tunnel' }
    # Windows PowerShell can't convert Windows time zone ids to the IANA names Linux uses
    $values.TZ = Ask 'Time zone (IANA name, e.g. America/New_York)' 'Etc/UTC'

    $lines = foreach ($line in Get-Content .env.example) {
        if ($line -match '^([A-Z_]+)=' -and $values.ContainsKey($Matches[1])) { "$($Matches[1])=$($values[$Matches[1]])" }
        else { $line }
    }
    # compose rejects a BOM, so no Set-Content/Out-File here
    [IO.File]::WriteAllText((Join-Path $PWD '.env'), (($lines -join "`n") + "`n"), (New-Object Text.UTF8Encoding $false))
} else {
    Say 'Using existing .env'
}

# ---- start ----
Say 'Pulling images'
& docker compose pull --ignore-pull-failures
# compose builds a MediaHub image locally only if it could not be pulled (e.g. before CI has published one)
& docker compose up -d
if ($LASTEXITCODE -ne 0) { Die 'docker compose up failed' }

Say 'Waiting for first-run setup (bootstrap)'
# start -a re-runs it if an earlier run already finished, or attaches if up just started it
$log = & docker start -a (& docker compose ps -a -q bootstrap)
if ($LASTEXITCODE -ne 0) {
    $log
    Die 'bootstrap failed. Fix the problem above, then run .\install.ps1 again'
}
$log | Select-String -Pattern 'added|warning|note'

$envValues = @{}
Get-Content .env | Where-Object { $_ -match '^([A-Z_]+)=(.*)$' } | ForEach-Object { $envValues[$Matches[1]] = $Matches[2] }
$port = if ($envValues.HTTP_PORT) { $envValues.HTTP_PORT } else { '80' }
Write-Host ''
Say 'MediaHub is running'
Write-Host "  MediaHub     $($envValues.PUBLIC_URL)  (or http://localhost:$port)"
Write-Host '  qBittorrent  http://localhost:8080    Sonarr   http://localhost:8989'
Write-Host '  Radarr       http://localhost:7878    Prowlarr http://localhost:9696'
if ($firstRun) {
    Write-Host ''
    Write-Host "  MediaHub login:           admin@example.com / $($envValues.ADMIN_PASSWORD)"
    Write-Host "  *arr + qBittorrent login: admin / $($envValues.ARR_PASSWORD)"
    Write-Host '  (also stored in .env; change the MediaHub password after logging in)'
}
if ((",$($envValues.COMPOSE_PROFILES),") -like '*,quicktunnel,*') {
    Write-Host ''
    & (Join-Path $PSScriptRoot 'tunnel-url.ps1')
}
Write-Host ''
Write-Host '  Next: add indexers in Prowlarr. Updates install themselves after each push to main.'
