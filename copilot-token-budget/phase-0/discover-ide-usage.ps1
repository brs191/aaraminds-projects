<#
.SYNOPSIS
  discover-ide-usage.ps1 — Phase 0.5 data-source discovery for VS Code IDE Copilot usage (Windows).

  Native PowerShell port of discover-ide-usage.sh. READ-ONLY. ZERO-NETWORK.
  Makes no changes and contacts no servers. Enumerates where Copilot (CLI + VS Code IDE)
  writes local data and prints a REDACTED schema sample so the reader can be built against
  the real format.

  Redaction: long string values and email-like tokens are masked; JSON KEYS and NUMERIC
  values (token counts, credits) are preserved — those are what we need.

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File phase-0\discover-ide-usage.ps1 > ide-usage-report.txt
  # then paste ide-usage-report.txt back into the chat
#>

$ErrorActionPreference = 'SilentlyContinue'
$HomeDir  = $env:USERPROFILE
$AppData  = $env:APPDATA          # %USERPROFILE%\AppData\Roaming

function Section($t) { "`n========== $t ==========" }

# Mask emails and long string values; keep JSON keys + numbers.
function Redact([string]$s) {
  if ($null -eq $s) { return '' }
  $s = [regex]::Replace($s, '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}', '<EMAIL>')
  $s = [regex]::Replace($s, '("(apiKey|token|authorization|access_token|webhook|url|content|text|prompt|completion|message)"\s*:\s*")[^"]*"', '$1<REDACTED>"', 'IgnoreCase')
  $s = [regex]::Replace($s, '("[A-Za-z0-9_]+"\s*:\s*")[^"]{41,}"', '$1<STR>"')
  return $s
}

# Print schema of a JSONL file: union of top-level keys + redacted first/last record.
function Sample-Jsonl([string]$f) {
  $info = Get-Item $f
  $lines = @(Get-Content -LiteralPath $f -ErrorAction SilentlyContinue)
  "--- $f  ($($lines.Count) lines, $([math]::Round($info.Length/1KB,1)) KB)"
  $keys = New-Object System.Collections.Generic.HashSet[string]
  foreach ($ln in ($lines | Select-Object -First 50)) {
    if ([string]::IsNullOrWhiteSpace($ln)) { continue }
    try { ($ln | ConvertFrom-Json).PSObject.Properties.Name | ForEach-Object { [void]$keys.Add($_) } } catch {}
  }
  "  top-level keys (union, first 50 lines): " + (($keys | Sort-Object) -join ', ')
  $fields = 'totalNanoAiu|nanoAiu|tokens|inputTokens|outputTokens|promptTokens|completionTokens|cachedTokens|usage|model|premiumRequests|credits|cost'
  $hits = [regex]::Matches((($lines -join "`n")), "`"($fields)`"") | ForEach-Object { $_.Value } | Sort-Object -Unique
  "  billing/token fields seen: " + ($hits -join ' ')
  if ($lines.Count -gt 0) {
    "  first record (redacted):"; "    " + (Redact ($lines[0]).Substring(0, [math]::Min(1200, $lines[0].Length)))
    "  last record (redacted):";  "    " + (Redact ($lines[-1]).Substring(0, [math]::Min(1200, $lines[-1].Length)))
  }
}

Section "ENVIRONMENT"
"date: $((Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ'))"
"os: $([System.Environment]::OSVersion.VersionString)  ($env:PROCESSOR_ARCHITECTURE)"
"code CLI present: $([bool](Get-Command code -ErrorAction SilentlyContinue))"

Section "~/.copilot TREE (CLI + any shared streams)"
$copilot = Join-Path $HomeDir '.copilot'
if (Test-Path $copilot) {
  Get-ChildItem -Recurse -Directory $copilot -Depth 3 | ForEach-Object { $_.FullName.Replace($HomeDir,'~') }
  "--- file types/counts under ~/.copilot:"
  Get-ChildItem -Recurse -File $copilot | Group-Object Extension | ForEach-Object { "{0,5}  {1}" -f $_.Count, $_.Name }
  "--- newest 15 files:"
  Get-ChildItem -Recurse -File $copilot | Sort-Object LastWriteTime -Descending | Select-Object -First 15 |
    ForEach-Object { "{0}  {1}" -f $_.LastWriteTime.ToString('s'), $_.FullName.Replace($HomeDir,'~') }
} else { "~/.copilot NOT found" }

Section "~/.copilot/otel SAMPLES (ccusage reads these)"
Get-ChildItem (Join-Path $copilot 'otel\*.jsonl') -ErrorAction SilentlyContinue | ForEach-Object { Sample-Jsonl $_.FullName }

Section "~/.copilot OTHER *.jsonl / *.log SAMPLES"
Get-ChildItem -Recurse -File $copilot -Include *.jsonl,*.log -ErrorAction SilentlyContinue |
  Where-Object { $_.FullName -notmatch '\\otel\\' -and $_.FullName -notmatch '\\session-state\\' } |
  ForEach-Object { Sample-Jsonl $_.FullName }

# VS Code user-data roots on Windows: %APPDATA%\Code, Code - Insiders, VSCodium.
$CodeRoots = @('Code','Code - Insiders','VSCodium') | ForEach-Object { Join-Path $AppData $_ }

Section "VS CODE — Copilot extension storage (Windows)"
"Scanning user-data roots: " + (($CodeRoots | ForEach-Object { $_.Replace($HomeDir,'~') }) -join '  ')
foreach ($root in $CodeRoots) {
  if (-not (Test-Path $root)) { continue }
  $gs = Join-Path $root 'User\globalStorage'
  if (Test-Path $gs) {
    "--- $($root.Replace($HomeDir,'~'))\User\globalStorage entries matching copilot:"
    Get-ChildItem $gs -ErrorAction SilentlyContinue | Where-Object Name -match 'copilot' | ForEach-Object { $_.FullName.Replace($HomeDir,'~') }
    Get-ChildItem -Recurse -File $gs -ErrorAction SilentlyContinue | Where-Object FullName -match 'copilot' | Select-Object -First 40 | ForEach-Object { $_.FullName.Replace($HomeDir,'~') }
  }
  $ws = Join-Path $root 'User\workspaceStorage'
  if (Test-Path $ws) {
    "--- $($root.Replace($HomeDir,'~'))\User\workspaceStorage copilot files (first 20):"
    Get-ChildItem -Recurse -File $ws -ErrorAction SilentlyContinue | Where-Object FullName -match 'copilot' | Select-Object -First 20 | ForEach-Object { $_.FullName.Replace($HomeDir,'~') }
  }
}

Section "VS CODE — Copilot logs (diagnostic; check if any carry token/usage)"
foreach ($root in $CodeRoots) {
  $logs = Join-Path $root 'logs'
  if (-not (Test-Path $logs)) { continue }
  "--- $($root.Replace($HomeDir,'~')) newest copilot log files:"
  $logFiles = Get-ChildItem -Recurse -File $logs -ErrorAction SilentlyContinue | Where-Object FullName -match 'copilot' | Sort-Object LastWriteTime -Descending
  $logFiles | Select-Object -First 10 | ForEach-Object { "{0}  {1}" -f $_.LastWriteTime.ToString('s'), $_.FullName.Replace($HomeDir,'~') }
  if ($logFiles.Count -gt 0) {
    "--- token/usage/premium mentions in newest copilot log ($($logFiles[0].Name)):"
    (Select-String -Path $logFiles[0].FullName -Pattern '(token[s]?|usage|premium|quota|model|credit)' -AllMatches).Line |
      Select-Object -Unique | Select-Object -First 25
  }
}

Section "ANY OTHER likely usage DBs (state.vscdb / sqlite)"
foreach ($root in $CodeRoots) {
  Get-ChildItem -Recurse -File (Join-Path $root 'User') -Filter 'state.vscdb' -ErrorAction SilentlyContinue |
    ForEach-Object { $_.FullName.Replace($HomeDir,'~') }
}
"(If a copilot usage table lives in state.vscdb, note it — it's SQLite.)"

Section "DONE"
"Paste this whole report back. Nothing was modified; no network calls were made."
