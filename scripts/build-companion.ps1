$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$env:GOCACHE = Join-Path $root ".gocache"
$env:GOTELEMETRY = "off"
$output = Join-Path $root "bin\workspace-halo-companion.exe"
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $output) | Out-Null
Push-Location (Join-Path $root "companion")
try {
    go build -buildvcs=false -trimpath -o $output .
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE"
    }
}
finally {
    Pop-Location
}
