# Install the gtt CLI (Windows, PowerShell):
#   irm https://raw.githubusercontent.com/GTT-Community/gtt-cli/main/install.ps1 | iex
#
# Options (environment variables):
#   $env:GTT_VERSION = "1.2.3"        a specific release (default: the latest)
#   $env:GTT_INSTALL_DIR = "C:\path"  where to put gtt.exe (default: %LOCALAPPDATA%\Programs\gtt)
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$Repo = "GTT-Community/gtt-cli"
$Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    default { throw "gtt install: unsupported architecture $($env:PROCESSOR_ARCHITECTURE)" }
}
if ($env:GTT_VERSION) {
    $Base = "https://github.com/$Repo/releases/download/v$($env:GTT_VERSION.TrimStart('v'))"
} else {
    $Base = "https://github.com/$Repo/releases/latest/download"
}
$Archive = "gtt_windows_$Arch.zip"
$Dir = if ($env:GTT_INSTALL_DIR) { $env:GTT_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\gtt" }

$Tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("gtt-install-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $Tmp | Out-Null
try {
    Write-Host "Downloading $Archive ..."
    Invoke-WebRequest "$Base/$Archive" -OutFile (Join-Path $Tmp $Archive) -UseBasicParsing
    Invoke-WebRequest "$Base/checksums.txt" -OutFile (Join-Path $Tmp "checksums.txt") -UseBasicParsing

    $Line = Get-Content (Join-Path $Tmp "checksums.txt") | Where-Object { $_ -match " $([regex]::Escape($Archive))$" }
    if (-not $Line) { throw "gtt install: $Archive is not listed in checksums.txt" }
    $Expected = ($Line -split '\s+')[0].ToLower()
    $Actual = (Get-FileHash (Join-Path $Tmp $Archive) -Algorithm SHA256).Hash.ToLower()
    if ($Expected -ne $Actual) { throw "gtt install: checksum mismatch for $Archive" }

    Expand-Archive (Join-Path $Tmp $Archive) -DestinationPath (Join-Path $Tmp "x") -Force
    New-Item -ItemType Directory -Path $Dir -Force | Out-Null
    Copy-Item (Join-Path $Tmp "x\gtt.exe") (Join-Path $Dir "gtt.exe") -Force
} finally {
    Remove-Item $Tmp -Recurse -Force -ErrorAction SilentlyContinue
}

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not (($UserPath -split ';') -contains $Dir)) {
    [Environment]::SetEnvironmentVariable("Path", (($UserPath.TrimEnd(';'), $Dir) -join ';').TrimStart(';'), "User")
    $env:Path = "$env:Path;$Dir"
    Write-Host "Added $Dir to your user PATH (open a new terminal to use it everywhere)."
}
& (Join-Path $Dir "gtt.exe") version
Write-Host "GTT Bootstrap also needs Git for Windows (bash) and Python 3."
