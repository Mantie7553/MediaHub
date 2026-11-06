# Adds secrets introduced after an install to an existing .env, leaving every other line alone.
# Run from the repo root by install.ps1 and update.ps1
$ErrorActionPreference = 'Stop'

function New-Secret([int]$bytes) {
    $buf = New-Object byte[] $bytes
    [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($buf)
    return -join ($buf | ForEach-Object { $_.ToString('x2') })
}

$path = Join-Path $PWD '.env'
$lines = @(Get-Content $path)
$changed = $false
foreach ($s in @(@{ Name = 'SUWAYOMI_PASSWORD'; Bytes = 12 })) {
    if ($lines | Where-Object { $_ -match "^$($s.Name)=.+" }) { continue }
    # an empty "NAME=" line from an older .env.example is replaced, not duplicated
    $lines = @($lines | Where-Object { $_ -notmatch "^$($s.Name)=" }) + "$($s.Name)=$(New-Secret $s.Bytes)"
    Write-Host "Added $($s.Name) to .env"
    $changed = $true
}
if ($changed) {
    # compose rejects a BOM, so no Set-Content/Out-File here
    [IO.File]::WriteAllText($path, (($lines -join "`n") + "`n"), (New-Object Text.UTF8Encoding $false))
}
