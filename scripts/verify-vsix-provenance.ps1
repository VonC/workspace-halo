param(
    [string]$Vsix,
    [switch]$Independent,
    [switch]$RequireClean,
    [switch]$RequireVersionTag
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$env:GOTELEMETRY = "off"
$expectedPath = Join-Path $root "dist\build-provenance.json"

if ([string]::IsNullOrWhiteSpace($Vsix)) {
    if (-not (Test-Path -LiteralPath $expectedPath)) {
        throw "Build provenance not found: $expectedPath. Run build.bat first or pass a VSIX path."
    }
    $selection = Get-Content -Raw $expectedPath | ConvertFrom-Json
    $Vsix = $selection.artifact
}

$candidatePath = if ([System.IO.Path]::IsPathRooted($Vsix)) {
    $Vsix
}
else {
    Join-Path $root $Vsix
}
$vsixPath = (Resolve-Path -LiteralPath $candidatePath).Path
$expected = if ($Independent) {
    $null
}
else {
    Get-Content -Raw $expectedPath | ConvertFrom-Json
}

Add-Type -AssemblyName System.IO.Compression.FileSystem
$archive = [System.IO.Compression.ZipFile]::OpenRead($vsixPath)
$tempHost = [System.IO.Path]::GetTempFileName()
$tagName = $null
$tagTarget = $null
$tagType = $null

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

    if ($actual.commit -notmatch "^[0-9a-f]{40}$") {
        throw "Packaged provenance commit is not a full SHA-1: '$($actual.commit)'"
    }
    if ($actual.shortCommit -ne $actual.commit.Substring(0, 7)) {
        throw "Packaged short commit '$($actual.shortCommit)' does not match '$($actual.commit)'"
    }
    if ($actual.nativeHostVcsStamped -ne $true) {
        throw "Packaged provenance does not declare a VCS-stamped native host"
    }
    if ($RequireClean -and [bool]$actual.dirty) {
        throw "A version-tag audit requires a clean VSIX, but packaged provenance reports dirty=True"
    }
    $dirtySuffix = if ([bool]$actual.dirty) { "-dirty" } else { "" }
    $calculatedArtifact = "workspace-halo-$($actual.version)-$($actual.shortCommit)$dirtySuffix-win32-x64.vsix"
    if ($actual.artifact -ne $calculatedArtifact) {
        throw "Packaged artifact '$($actual.artifact)' != calculated artifact '$calculatedArtifact'"
    }
    if ((Split-Path -Leaf $vsixPath) -ne $actual.artifact) {
        throw "VSIX filename does not match packaged artifact '$($actual.artifact)'"
    }

    if ($null -ne $expected) {
        foreach ($field in @("schemaVersion", "version", "commit", "shortCommit", "commitTime", "dirty", "nativeHostVcsStamped", "artifact")) {
            if ($actual.$field -ne $expected.$field) {
                throw "Packaged provenance mismatch for '$field': '$($actual.$field)' != '$($expected.$field)'"
            }
        }
    }
    if ($package.version -ne $actual.version) {
        throw "Package version '$($package.version)' != provenance version '$($actual.version)'"
    }
    $identity = $manifest.PackageManifest.Metadata.Identity
    if ($identity.Version -ne $actual.version) {
        throw "VSIX version '$($identity.Version)' != provenance version '$($actual.version)'"
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
    if ($revision -ne $actual.commit) {
        throw "Native host revision '$revision' != provenance commit '$($actual.commit)'"
    }
    if ($modified -ne [bool]$actual.dirty) {
        throw "Native host dirty state '$modified' != provenance dirty state '$($actual.dirty)'"
    }

    if ($RequireVersionTag) {
        $tagName = [string]$package.version
        $tagRef = "refs/tags/$tagName"
        $tagTargetOutput = @(& git -C $root rev-list -n 1 $tagRef 2>$null)
        if ($LASTEXITCODE -ne 0 -or $tagTargetOutput.Count -eq 0) {
            throw "Local version tag '$tagName' does not exist"
        }
        $tagTarget = $tagTargetOutput[0].Trim()
        if ($tagTarget -ne $actual.commit) {
            throw "Local version tag '$tagName' targets '$tagTarget', not packaged commit '$($actual.commit)'"
        }
        $tagType = ((& git -C $root cat-file -t $tagRef) | Select-Object -First 1).Trim()
        if ($LASTEXITCODE -ne 0) {
            throw "Cannot read local version tag '$tagName'"
        }
    }
}
finally {
    $archive.Dispose()
    Remove-Item -LiteralPath $tempHost -Force -ErrorAction SilentlyContinue
}

$hash = (Get-FileHash $vsixPath -Algorithm SHA256).Hash
Write-Output "Verified VSIX provenance: version=$($actual.version) commit=$($actual.commit) dirty=$($actual.dirty)"
if ($RequireVersionTag) {
    Write-Output "Version tag: $tagName -> $tagTarget ($tagType)"
}
Write-Output "SHA256: $hash"
