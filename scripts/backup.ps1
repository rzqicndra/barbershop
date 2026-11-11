$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$backupDirectory = Join-Path $projectRoot "backups"
New-Item -ItemType Directory -Force $backupDirectory | Out-Null
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$filename = "gilir-$stamp.dump"
Push-Location $projectRoot
try {
    docker compose exec -T db pg_dump -U gilir -d gilir -Fc -f /tmp/gilir-backup.dump
    if ($LASTEXITCODE -ne 0) { throw "pg_dump gagal." }
    docker compose cp db:/tmp/gilir-backup.dump (Join-Path $backupDirectory $filename)
    if ($LASTEXITCODE -ne 0) { throw "Salinan backup gagal." }
    docker compose exec -T db rm /tmp/gilir-backup.dump
    $photos = Join-Path $backupDirectory "gilir-$stamp-uploads"
    docker compose cp app:/app/uploads $photos
    if ($LASTEXITCODE -ne 0) { throw "Backup database berhasil, tetapi salinan foto gagal. Pastikan container app berjalan." }
    Write-Host "Backup tersimpan: backups/$filename dan backups/gilir-$stamp-uploads"
} finally { Pop-Location }
