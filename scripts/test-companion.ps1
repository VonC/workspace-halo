$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$env:GOCACHE = Join-Path $root ".gocache"
$env:GOTELEMETRY = "off"
Push-Location (Join-Path $root "companion")
try {
    go test -buildvcs=false ./...
    if ($LASTEXITCODE -ne 0) {
        throw "go test failed with exit code $LASTEXITCODE"
    }
}
finally {
    Pop-Location
}
