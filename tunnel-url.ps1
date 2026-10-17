# Print the current quick tunnel address. It changes whenever the cloudflared-quick container
# restarts (reboot, docker restart), so run this again after one.
Set-Location $PSScriptRoot
for ($i = 0; $i -lt 30; $i++) {
    $logs = & docker compose logs cloudflared-quick 2>$null
    $found = [regex]::Matches(($logs -join "`n"), 'https://[a-z0-9-]+\.trycloudflare\.com')
    if ($found.Count -gt 0) {
        Write-Host "  Remote address (quick tunnel): $($found[$found.Count - 1].Value)"
        exit 0
    }
    Start-Sleep -Seconds 2
}
Write-Host '  No quick tunnel address yet. Is the quicktunnel profile on? Check: docker compose logs cloudflared-quick'
exit 1
