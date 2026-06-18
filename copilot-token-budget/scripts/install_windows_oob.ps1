param(
  [string]$Bundle,
  [string]$BinDir = "$env:USERPROFILE\bin",
  [string]$CodeCmd = "code",
  [switch]$NoPathUpdate,
  [switch]$SkipExtension
)

$ErrorActionPreference = 'Stop'

function Write-ErrorAndExit([string]$Message) {
  Write-Error $Message
  exit 1
}

function Resolve-BundleRoot {
  if ($Bundle -and $Bundle.Trim() -ne '') {
    return (Resolve-Path $Bundle).Path
  }

  $scriptRoot = $PSScriptRoot
  if (Test-Path (Join-Path $scriptRoot 'manifest.json') -PathType Leaf -and
      Test-Path (Join-Path $scriptRoot 'binaries') -PathType Container) {
    return $scriptRoot
  }

  $parent = Split-Path $scriptRoot -Parent
  if (Test-Path (Join-Path $parent 'manifest.json') -PathType Leaf -and
      Test-Path (Join-Path $parent 'binaries') -PathType Container) {
    return $parent
  }

  Write-ErrorAndExit 'Bundle directory not found. Pass -Bundle C:\path\to\copilot-token-budget-windows-<version>.'
}

function Install-File([string]$Src, [string]$Dst) {
  $dstDir = Split-Path $Dst -Parent
  New-Item -ItemType Directory -Force -Path $dstDir | Out-Null
  Copy-Item -Force $Src $Dst
}

function Update-UserPath([string]$PathDir) {
  if ($NoPathUpdate) {
    return
  }

  $current = [Environment]::GetEnvironmentVariable('Path', 'User')
  if ([string]::IsNullOrWhiteSpace($current)) {
    [Environment]::SetEnvironmentVariable('Path', $PathDir, 'User')
    return
  }

  $segments = $current -split ';' | Where-Object { $_ -ne '' }
  if ($segments -notcontains $PathDir) {
    [Environment]::SetEnvironmentVariable('Path', ($current.TrimEnd(';') + ';' + $PathDir), 'User')
  }
}

$bundleRoot = Resolve-BundleRoot
$manifestPath = Join-Path $bundleRoot 'manifest.json'
$binaryRoot = Join-Path $bundleRoot 'binaries'
if (-not (Test-Path $manifestPath -PathType Leaf) -or -not (Test-Path $binaryRoot -PathType Container)) {
  Write-ErrorAndExit "Invalid bundle layout at: $bundleRoot"
}

$archDir = Join-Path $binaryRoot 'windows_amd64'
if (-not (Test-Path $archDir -PathType Container)) {
  Write-ErrorAndExit "Missing architecture bundle: $archDir"
}

$bins = @(
  'copilot-analyze.exe',
  'copilot-dashboard.exe',
  'copilot-statusline.exe',
  'copilot-alert.exe',
  'copilot-budget-mcp.exe'
)

New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

foreach ($bin in $bins) {
  $src = Join-Path $archDir $bin
  if (-not (Test-Path $src -PathType Leaf)) {
    Write-ErrorAndExit "Missing binary in bundle: $src"
  }
  Install-File $src (Join-Path $BinDir $bin)
}

if (-not $SkipExtension) {
  $vsix = Get-ChildItem -Path (Join-Path $bundleRoot 'extension') -Filter *.vsix -File | Select-Object -First 1
  if (-not $vsix) {
    Write-ErrorAndExit "Expected a VSIX in $bundleRoot\extension"
  }

  if (-not (Get-Command $CodeCmd -ErrorAction SilentlyContinue)) {
    Write-ErrorAndExit "VS Code CLI not found: $CodeCmd"
  }

  & $CodeCmd --install-extension $vsix.FullName --force
  if ($LASTEXITCODE -ne 0) {
    Write-ErrorAndExit "VSIX install failed"
  }
}

Update-UserPath $BinDir
$env:Path = "$BinDir;$env:Path"

Write-Host "Installed Windows bundle from: $bundleRoot"
Write-Host "Binaries: $BinDir"
Write-Host "Next: open VS Code and run 'Copilot Budget: Show Dashboard'"
Write-Host "Caveman demo (optional): run .\launch-caveman-demo.ps1 to open examples\token-optimization-demo"
