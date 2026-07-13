param(
    [Parameter(Mandatory = $true)]
    [string]$NodeHome,
    [Parameter(Mandatory = $true)]
    [ValidateSet("check-types", "compile", "test", "package:vsix")]
    [string]$Task
)

$ErrorActionPreference = "Stop"
$node = Join-Path $NodeHome "node.exe"
$npm = Join-Path $NodeHome "npm.cmd"
if (-not (Test-Path -LiteralPath $node) -or -not (Test-Path -LiteralPath $npm)) {
    throw "NodeHome must contain node.exe and npm.cmd: $NodeHome"
}
$env:PATH = "$NodeHome;$env:PATH"
& $npm run $Task
if ($LASTEXITCODE -ne 0) {
    throw "npm run $Task failed with exit code $LASTEXITCODE"
}

