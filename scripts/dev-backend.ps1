$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$envPath = Join-Path $projectRoot ".env"
if (-not (Test-Path $envPath)) { throw "Jalankan node scripts/setup.mjs dari folder proyek terlebih dahulu." }
$config = @{}
foreach ($line in Get-Content $envPath) {
    if ($line -match '^([A-Z_]+)=(.*)$') { $config[$matches[1]] = $matches[2] }
}
$env:DATABASE_URL = "postgres://gilir_app:$($config['APP_DB_PASSWORD'])@localhost:5432/gilir?sslmode=disable"
$env:PUBLIC_ORIGIN = "http://localhost:5173"
$env:EXTRA_ORIGINS = "http://localhost:8080"
$env:TOKEN_KEY = $config['TOKEN_KEY']
$env:COOKIE_SECURE = "false"
$env:ADDR = ":8080"
$env:WEB_DIR = Join-Path $projectRoot "frontend/dist"
Push-Location (Join-Path $projectRoot "backend")
try {
    go run ./cmd/gilir serve
    if ($LASTEXITCODE -ne 0) { throw "Backend berhenti dengan error." }
} finally { Pop-Location }
