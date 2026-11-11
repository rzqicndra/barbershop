$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$backupDirectory = Join-Path $projectRoot "backups"
New-Item -ItemType Directory -Force $backupDirectory | Out-Null
$filename = "gilir-$(Get-Date -Format 'yyyyMMdd-HHmmss').dump"
Push-Location $projectRoot
try {
    docker compose exec -T db pg_dump -U gilir -d gilir -Fc -f /tmp/gilir-backup.dump
    if ($LASTEXITCODE -ne 0) { throw "pg_dump gagal." }
    docker compose cp db:/tmp/gilir-backup.dump (Join-Path $backupDirectory $filename)
    if ($LASTEXITCODE -ne 0) { throw "Salinan backup gagal." }
    docker compose exec -T db rm /tmp/gilir-backup.dump
    Write-Host "Backup tersimpan: backups/$filename"
} finally { Pop-Location }
