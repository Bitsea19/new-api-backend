<#
.SYNOPSIS
    Package the backend source for upload to a server that builds its own image.

.DESCRIPTION
    The server runs `docker compose up -d --build`, which compiles the Go binary
    inside a container. So the archive has to contain the full source tree, not
    just a binary: go.mod, go.sum, the local relaykit/tokenkit modules, the
    Dockerfile and the source of every package.

    The dashboard is a separate repository and is deliberately not included.

.PARAMETER OutputDir
    Where to write the archive. Defaults to the parent of this repository, so
    the archive lands next to the repository rather than inside it.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts\package-for-server.ps1
#>
[CmdletBinding()]
param(
    [string]$OutputDir
)

$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
if (-not $OutputDir) {
    $OutputDir = Split-Path -Parent $repoRoot
}
$archive = Join-Path $OutputDir 'new-api-backend-src.tar.gz'

Push-Location $repoRoot
try {
    # Every tracked file, with its current content from the working tree, so
    # uncommitted edits are included. Untracked files are skipped.
    $files = & git ls-files
    if ($LASTEXITCODE -ne 0 -or -not $files) {
        throw 'git ls-files failed; is this a git repository?'
    }

    # Stage the files under a `new-api-backend/` prefix, so that extracting the
    # archive on the server creates /opt/new-api-backend instead of scattering
    # files over the target directory. Windows tar cannot rewrite path prefixes
    # on the fly, so the copy happens first.
    $staging = Join-Path ([System.IO.Path]::GetTempPath()) ("new-api-pkg-" + [guid]::NewGuid().ToString('N'))
    $stagedRoot = Join-Path $staging 'new-api-backend'
    New-Item -ItemType Directory -Path $stagedRoot -Force | Out-Null

    try {
        foreach ($file in $files) {
            $src = Join-Path $repoRoot $file
            if (-not (Test-Path -LiteralPath $src -PathType Leaf)) {
                # A tracked path that no longer exists on disk (a removed file
                # that was not staged yet) has nothing to package.
                continue
            }
            $dst = Join-Path $stagedRoot $file
            $dstDir = Split-Path -Parent $dst
            if (-not (Test-Path -LiteralPath $dstDir)) {
                New-Item -ItemType Directory -Path $dstDir -Force | Out-Null
            }
            Copy-Item -LiteralPath $src -Destination $dst -Force
        }

        # tar ships with Windows 10+, so no extra tooling is needed.
        & tar -czf $archive -C $staging new-api-backend
        if ($LASTEXITCODE -ne 0) {
            throw "tar failed with exit code $LASTEXITCODE"
        }
    }
    finally {
        Remove-Item -LiteralPath $staging -Recurse -Force -ErrorAction SilentlyContinue
    }
}
finally {
    Pop-Location
}

Write-Host "Wrote $archive"
Write-Host ''
Write-Host 'Upload it to the server, then:'
Write-Host '  tar -xzf new-api-backend-src.tar.gz -C /opt'
Write-Host '  cd /opt/new-api-backend'
Write-Host '  docker compose up -d --build'
