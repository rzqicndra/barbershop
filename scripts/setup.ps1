$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$target = Join-Path $projectRoot ".env"
if (Test-Path $target) {
    Write-Host ".env sudah tersedia. Nilai lama dipertahankan."
    exit 0
}
function New-RandomHex([int] $length) {
    $bytes = New-Object byte[] $length
    $generator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try { $generator.GetBytes($bytes) } finally { $generator.Dispose() }
    return ([BitConverter]::ToString($bytes)).Replace("-", "").ToLowerInvariant()
}
$content = @(
    "# Rahasia lokal. Jangan commit atau kirim file ini."
    "DB_PASSWORD=$(New-RandomHex 24)"
    "APP_DB_PASSWORD=$(New-RandomHex 24)"
    "TOKEN_KEY=$(New-RandomHex 32)"
    "PUBLIC_ORIGIN=http://localhost:8080"
    "COOKIE_SECURE=false"
    "BIND_ADDRESS=127.0.0.1"
    "APP_PORT=8080"
    ""
) -join "`n"
$stream = [System.IO.File]::Open($target, [System.IO.FileMode]::CreateNew)
try {
    $bytes = ([System.Text.UTF8Encoding]::new($false)).GetBytes($content)
    $stream.Write($bytes, 0, $bytes.Length)
} finally { $stream.Dispose() }
Write-Host ".env berhasil dibuat. Lanjutkan panduan README.md."
