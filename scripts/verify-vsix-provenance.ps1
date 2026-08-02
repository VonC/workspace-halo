param(
    [Parameter(Mandatory = $true)]
    [string]$Vsix
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$env:GOTELEMETRY = "off"
$vsixPath = (Resolve-Path (Join-Path $root $Vsix)).Path
$expectedPath = Join-Path $root "dist\build-provenance.json"
$expected = Get-Content -Raw $expectedPath | ConvertFrom-Json

if ((Split-Path -Leaf $vsixPath) -ne $expected.artifact) {
    throw "VSIX filename does not match provenance artifact '$($expected.artifact)'"
}

Add-Type -AssemblyName System.IO.Compression.FileSystem
$archive = [System.IO.Compression.ZipFile]::OpenRead($vsixPath)
$tempHost = [System.IO.Path]::GetTempFileName()

function Read-ZipText {
    param(
        [Parameter(Mandatory = $true)]$Archive,
        [Parameter(Mandatory = $true)][string]$EntryName
    )

    $entry = $Archive.GetEntry($EntryName)
    if ($null -eq $entry) {
        throw "VSIX entry is missing: $EntryName"
    }
    $reader = [System.IO.StreamReader]::new($entry.Open())
    try {
        return $reader.ReadToEnd()
    }
    finally {
        $reader.Dispose()
    }
}

try {
    $package = (Read-ZipText $archive "extension/package.json") | ConvertFrom-Json
    [xml]$manifest = Read-ZipText $archive "extension.vsixmanifest"
    $actual = (Read-ZipText $archive "extension/dist/build-provenance.json") | ConvertFrom-Json

    foreach ($field in @("schemaVersion", "version", "commit", "shortCommit", "commitTime", "dirty", "nativeHostVcsStamped", "artifact")) {
        if ($actual.$field -ne $expected.$field) {
            throw "Packaged provenance mismatch for '$field': '$($actual.$field)' != '$($expected.$field)'"
        }
    }
    if ($package.version -ne $expected.version) {
        throw "Package version '$($package.version)' != provenance version '$($expected.version)'"
    }
    $identity = $manifest.PackageManifest.Metadata.Identity
    if ($identity.Version -ne $expected.version) {
        throw "VSIX version '$($identity.Version)' != provenance version '$($expected.version)'"
    }
    if ($identity.TargetPlatform -ne "win32-x64") {
        throw "VSIX target '$($identity.TargetPlatform)' != 'win32-x64'"
    }

    $hostEntry = $archive.GetEntry("extension/bin/win32-x64/workspace-halo-host.exe")
    if ($null -eq $hostEntry) {
        throw "VSIX native host is missing"
    }
    $hostInput = $hostEntry.Open()
    $hostOutput = [System.IO.File]::OpenWrite($tempHost)
    try {
        $hostInput.CopyTo($hostOutput)
    }
    finally {
        $hostOutput.Dispose()
        $hostInput.Dispose()
    }

    $buildInfo = @(& go version -m $tempHost)
    if ($LASTEXITCODE -ne 0) {
        throw "go version -m failed with exit code $LASTEXITCODE"
    }
    $revisionLine = $buildInfo | Where-Object { $_ -match "vcs\.revision=" } | Select-Object -First 1
    $modifiedLine = $buildInfo | Where-Object { $_ -match "vcs\.modified=" } | Select-Object -First 1
    if ($null -eq $revisionLine -or $null -eq $modifiedLine) {
        throw "Native host does not contain Go VCS metadata"
    }
    $revision = ($revisionLine -split "=", 2)[1].Trim()
    $modified = ($modifiedLine -split "=", 2)[1].Trim().ToLowerInvariant() -eq "true"
    if ($revision -ne $expected.commit) {
        throw "Native host revision '$revision' != provenance commit '$($expected.commit)'"
    }
    if ($modified -ne [bool]$expected.dirty) {
        throw "Native host dirty state '$modified' != provenance dirty state '$($expected.dirty)'"
    }
}
finally {
    $archive.Dispose()
    Remove-Item -LiteralPath $tempHost -Force -ErrorAction SilentlyContinue
}

$hash = (Get-FileHash $vsixPath -Algorithm SHA256).Hash
Write-Output "Verified VSIX provenance: version=$($expected.version) commit=$($expected.shortCommit) dirty=$($expected.dirty)"
Write-Output "SHA256: $hash"
