$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = (Resolve-Path (Join-Path $scriptDir "..")).Path

function Has-Command {
  param([Parameter(Mandatory = $true)][string]$Name)
  return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

function Add-PathIfMissing {
  param([Parameter(Mandatory = $true)][string]$Candidate)
  if (-not (Test-Path $Candidate)) {
    return
  }

  $current = $env:Path -split ";"
  if ($current -notcontains $Candidate) {
    $env:Path = "$env:Path;$Candidate"
  }
}

function Install-WithWinget {
  param([Parameter(Mandatory = $true)][string]$Id)
  winget install -e --id $Id --accept-package-agreements --accept-source-agreements
}

function Install-WithChoco {
  param([Parameter(Mandatory = $true)][string]$Package)
  choco install -y $Package
}

Write-Host "[setup] Checking prerequisites"

if (-not (Has-Command "go")) {
  if (Has-Command "winget") {
    Write-Host "[setup] Installing Go with winget"
    Install-WithWinget -Id "GoLang.Go"
  }
  elseif (Has-Command "choco") {
    Write-Host "[setup] Installing Go with Chocolatey"
    Install-WithChoco -Package "golang"
  }
  else {
    throw "Go is missing and no supported package manager (winget/choco) was found."
  }
}

if (-not (Has-Command "node")) {
  if (Has-Command "winget") {
    Write-Host "[setup] Installing Node.js LTS with winget"
    Install-WithWinget -Id "OpenJS.NodeJS.LTS"
  }
  elseif (Has-Command "choco") {
    Write-Host "[setup] Installing Node.js LTS with Chocolatey"
    Install-WithChoco -Package "nodejs-lts"
  }
  else {
    throw "Node.js is missing and no supported package manager (winget/choco) was found."
  }
}

# New installs may not refresh PATH in current shell.
Add-PathIfMissing -Candidate "$env:ProgramFiles\Go\bin"
Add-PathIfMissing -Candidate "$env:ProgramFiles\nodejs"

if (-not (Has-Command "go")) { throw "Go is still unavailable in PATH." }
if (-not (Has-Command "npm")) { throw "npm is still unavailable in PATH." }

Write-Host "[setup] Building web UI"
Push-Location (Join-Path $repoRoot "internal\webui")
npm ci --no-audit --no-fund
npm run build
Pop-Location

Write-Host "[setup] Compiling Go binaries"
$binDir = Join-Path $repoRoot "bin"
New-Item -ItemType Directory -Force -Path $binDir | Out-Null

Push-Location $repoRoot
go mod download
go build -o ".\bin\baobun-client.exe" .\cmd\client
go build -o ".\bin\baobun-maker.exe" .\cmd\maker
Pop-Location

Write-Host "[setup] Done."
Write-Host "[setup] Run: .\bin\baobun-client.exe"
