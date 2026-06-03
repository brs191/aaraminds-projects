# Copilot routing-mode check (Windows / PowerShell) - Token Optimizer M0-lite pre-flight.
# Run on ONE developer machine to decide whether the localhost LiteLLM proxy can
# interpose on Copilot traffic, and whether the S=$100/dev/mo input still holds.
#
# Decision table (canonical: ..\planning\validate_with_copilot.md):
#   baseUrl set? | per-token Anthropic billing? | traffic goes to        | Mode | Action
#   No           | No                           | *.githubcopilot.com    | A    | STOP - proxy blind, S wrong
#   No           | Yes                          | *.githubcopilot.com    | B    | Swap cohort to Claude Code / Cursor
#   Yes          | Yes                          | api.anthropic.com      | C    | PROCEED - proxyable
#
# Usage:
#   1) Open VS Code and send ONE Copilot Chat message so a request is in flight.
#   2) Within ~20s run:  powershell -ExecutionPolicy Bypass -File .\check_copilot_mode.ps1

Write-Host "==== Copilot routing-mode check ====" -ForegroundColor Cyan

# --- Check 1 (DECISIVE): custom endpoint in VS Code settings -----------------
Write-Host "`n[1] Scanning VS Code settings for a custom endpoint / baseUrl..." -ForegroundColor Yellow
$settingsPaths = @(
  "$env:APPDATA\Code\User\settings.json",
  "$env:APPDATA\Code - Insiders\User\settings.json",
  (Join-Path (Get-Location) ".vscode\settings.json")
)
$patterns = 'baseUrl|endpoint|copilot\.advanced|anthropic|azure|byok|github\.copilot\.chat'
$foundCustom = $false
foreach ($p in $settingsPaths) {
  if (Test-Path $p) {
    Write-Host "  file: $p"
    $hits = Select-String -Path $p -Pattern $patterns -ErrorAction SilentlyContinue
    if ($hits) {
      $foundCustom = $true
      $hits | ForEach-Object { Write-Host ("    > " + $_.Line.Trim()) -ForegroundColor Green }
    } else {
      Write-Host "    (no endpoint/baseUrl/byok keys found)"
    }
  } else {
    Write-Host "  file: $p  (not present)"
  }
}
if (-not $foundCustom) {
  Write-Host "  RESULT: no custom baseUrl/endpoint -> consistent with Mode A or B (default model picker)." -ForegroundColor Magenta
} else {
  Write-Host "  RESULT: custom endpoint keys present -> possibly Mode C (confirm traffic + billing)." -ForegroundColor Magenta
}

# --- Check 3 (confirmatory): where does VS Code traffic actually go? ----------
Write-Host "`n[3] Observing VS Code outbound connections (trigger a Copilot request NOW)..." -ForegroundColor Yellow
$codeProcs = Get-Process Code -ErrorAction SilentlyContinue
if (-not $codeProcs) { $codeProcs = Get-Process "Code - Insiders" -ErrorAction SilentlyContinue }
if (-not $codeProcs) {
  Write-Host "  VS Code process not found - is it running?" -ForegroundColor Red
} else {
  $procIds = $codeProcs.Id
  $remoteHosts = @{}
  1..20 | ForEach-Object {
    $conns = Get-NetTCPConnection -State Established -ErrorAction SilentlyContinue |
             Where-Object { $procIds -contains $_.OwningProcess -and $_.RemoteAddress -notmatch '^(127\.|::1|0\.0\.0\.0)' }
    foreach ($c in $conns) {
      $name = try { [System.Net.Dns]::GetHostEntry($c.RemoteAddress).HostName } catch { $c.RemoteAddress }
      $remoteHosts[$name] = $true
    }
    Start-Sleep -Milliseconds 750
  }
  if ($remoteHosts.Count -eq 0) {
    Write-Host "  No external connections captured. Re-run WHILE a Copilot request is in flight." -ForegroundColor Red
  } else {
    Write-Host "  Remote hosts seen from VS Code:"
    $remoteHosts.Keys | Sort-Object | ForEach-Object { Write-Host "    $_" }
    $toAnthropic = $remoteHosts.Keys | Where-Object { $_ -match 'anthropic' }
    $toGithub    = $remoteHosts.Keys | Where-Object { $_ -match 'githubcopilot|copilot|github' }
    Write-Host ""
    if ($toAnthropic) {
      Write-Host "  -> Direct api.anthropic.com traffic: consistent with Mode C (PROXYABLE)." -ForegroundColor Green
    } elseif ($toGithub) {
      Write-Host "  -> Only GitHub/Copilot hosts: consistent with Mode A or B (NOT proxyable)." -ForegroundColor Red
    } else {
      Write-Host "  -> Only CDN/IP names matched; network check is confirmatory only - trust [1] + [2]." -ForegroundColor Yellow
    }
  }
}

# --- Check 2 (MANUAL): per-token Anthropic billing distinguishes A vs B -------
Write-Host "`n[2] MANUAL - confirm per-token Anthropic billing (distinguishes Mode A from B):" -ForegroundColor Yellow
Write-Host "    Open https://console.anthropic.com -> Settings -> Billing -> Usage (last 30 days)."
Write-Host "    Spend roughly `$100 x dev-count on AITO's key  -> per-token exists (Mode B or C)."
Write-Host "    No AITO Anthropic account / no usage           -> flat-rate only (Mode A: S=`$100 is wrong)."

Write-Host "`n==== Combine [1] baseUrl + [2] billing + [3] traffic -> Mode -> Action ====" -ForegroundColor Cyan
Write-Host "  No baseUrl + default picker is the Mode A signature: the build as scoped cannot intercept."
Write-Host "  See ..\planning\validate_with_copilot.md for the full decision table." -ForegroundColor Cyan
