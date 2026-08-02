$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$packagePath = Join-Path $root "package.json"
$outputPath = Join-Path $root "dist\build-provenance.json"
$package = Get-Content -Raw $packagePath | ConvertFrom-Json

function Invoke-Git {
    param([Parameter(Mandatory = $true)][string[]]$Arguments)

    $output = & git -C $root @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "git $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
    return $output
}

$commit = (Invoke-Git @("rev-parse", "HEAD")).Trim()
$shortCommit = (Invoke-Git @("rev-parse", "--short=7", "HEAD")).Trim()
$commitTime = (Invoke-Git @("show", "-s", "--format=%cI", "HEAD")).Trim()
$status = @(Invoke-Git @("status", "--porcelain", "--untracked-files=normal"))
$dirty = $status.Count -gt 0
$dirtySuffix = if ($dirty) { "-dirty" } else { "" }
$artifact = "workspace-halo-$($package.version)-$shortCommit$dirtySuffix-win32-x64.vsix"

$provenance = [ordered]@{
    schemaVersion = 1
    version = $package.version
    commit = $commit
    shortCommit = $shortCommit
    commitTime = $commitTime
    dirty = $dirty
    nativeHostVcsStamped = $true
    artifact = $artifact
}

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $outputPath) | Out-Null
$json = $provenance | ConvertTo-Json
[System.IO.File]::WriteAllText(
    $outputPath,
    $json + [Environment]::NewLine,
    [System.Text.UTF8Encoding]::new($false)
)

Write-Output "Build provenance: version=$($package.version) commit=$shortCommit dirty=$dirty"
Write-Output "Build artifact: $artifact"
