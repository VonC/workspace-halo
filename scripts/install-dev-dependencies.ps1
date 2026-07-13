param(
    [Parameter(Mandatory = $true)]
    [string]$NodeHome
)

$ErrorActionPreference = "Stop"
$node = Join-Path $NodeHome "node.exe"
$npm = Join-Path $NodeHome "npm.cmd"
if (-not (Test-Path -LiteralPath $node) -or -not (Test-Path -LiteralPath $npm)) {
    throw "NodeHome must contain node.exe and npm.cmd: $NodeHome"
}
$env:PATH = "$NodeHome;$env:PATH"
& $npm install
if ($LASTEXITCODE -ne 0) {
    throw "npm install failed with exit code $LASTEXITCODE"
}
