#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUTPUT_DIR="${REPO_ROOT}/distr/v1.0.0"
VERSION=""
VSIX_PATH=""
DATE_UTC="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
COMMIT="unknown"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/package_windows_oob.sh --version v1.0.0 [--output-dir distr/v1.0.0] [--vsix path/to.vsix]

Behavior:
  - Cross-compiles the Windows/amd64 binaries from source.
  - Copies the Windows PowerShell installer, VS Code .vsix, and the Windows binaries.
  - Produces a single Windows bundle zip that can be unpacked and installed on Windows.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output-dir)
      [[ $# -ge 2 ]] || { echo "Error: --output-dir requires a directory" >&2; exit 1; }
      OUTPUT_DIR="$2"
      shift 2
      ;;
    --vsix)
      [[ $# -ge 2 ]] || { echo "Error: --vsix requires a file path" >&2; exit 1; }
      VSIX_PATH="$2"
      shift 2
      ;;
    --version)
      [[ $# -ge 2 ]] || { echo "Error: --version requires a value" >&2; exit 1; }
      VERSION="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Error: unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

[[ -n "$VERSION" ]] || { echo "Error: --version is required" >&2; exit 1; }

if [[ "$OUTPUT_DIR" != /* ]]; then
  OUTPUT_DIR="${REPO_ROOT}/${OUTPUT_DIR}"
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required command not found: $1" >&2
    exit 1
  fi
}

require_cmd go
require_cmd zip

if command -v sha256sum >/dev/null 2>&1; then
  hash_cmd=(sha256sum)
elif command -v shasum >/dev/null 2>&1; then
  hash_cmd=(shasum -a 256)
else
  echo "Error: required command not found: sha256sum or shasum" >&2
  exit 1
fi

if [[ -z "$VSIX_PATH" ]]; then
  shopt -s nullglob
  vsix_candidates=("${REPO_ROOT}"/extension/*.vsix "${REPO_ROOT}"/dist-vsix/*.vsix)
  shopt -u nullglob
  if [[ ${#vsix_candidates[@]} -lt 1 ]]; then
    echo "Error: no VSIX found. Pass --vsix /path/to/copilot-token-budget-<version>.vsix." >&2
    exit 1
  fi
  VSIX_PATH="${vsix_candidates[0]}"
fi

if [[ ! -f "$VSIX_PATH" ]]; then
  echo "Error: VSIX not found: $VSIX_PATH" >&2
  exit 1
fi

build_windows_binary() {
  local module_dir="$1"
  local main_pkg="$2"
  local binary_name="$3"
  local out_dir="$4"
  local ldflags="$5"

  mkdir -p "$out_dir"
  (
    cd "${REPO_ROOT}/${module_dir}"
    GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$ldflags" -o "${out_dir}/${binary_name}.exe" "$main_pkg"
  )
}

ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE_UTC}"
staging_dir="$(mktemp -d "${TMPDIR:-/tmp}/copilot-token-budget-windows-oob.XXXXXX")"
bundle_root="${staging_dir}/copilot-token-budget-windows-${VERSION}"
mkdir -p "$bundle_root/binaries/windows_amd64" "$bundle_root/extension"

cp "${SCRIPT_DIR}/install_windows_oob.ps1" "${bundle_root}/install.ps1"

cat > "${bundle_root}/launch-caveman-demo.ps1" <<'EOF'
param(
  [string]$CodeCmd = "code"
)

$ErrorActionPreference = 'Stop'
$target = Join-Path $PSScriptRoot 'examples\token-optimization-demo'
$code = Get-Command $CodeCmd -ErrorAction SilentlyContinue
if ($code) {
  & $CodeCmd $target
  exit $LASTEXITCODE
}

Start-Process -FilePath 'explorer.exe' -ArgumentList $target | Out-Null
EOF

if [[ -d "${REPO_ROOT}/examples/token-optimization-demo" ]]; then
  mkdir -p "${bundle_root}/examples/token-optimization-demo"
  cp -R "${REPO_ROOT}/examples/token-optimization-demo/." "${bundle_root}/examples/token-optimization-demo/"
fi

build_windows_binary core ./cmd/analyze copilot-analyze "${bundle_root}/binaries/windows_amd64" "$ldflags"
build_windows_binary core ./cmd/dashboard copilot-dashboard "${bundle_root}/binaries/windows_amd64" "$ldflags"
build_windows_binary core ./cmd/statusline copilot-statusline "${bundle_root}/binaries/windows_amd64" "$ldflags"
build_windows_binary alerting ./cmd/alert copilot-alert "${bundle_root}/binaries/windows_amd64" "$ldflags"
build_windows_binary mcp ./cmd/mcp-server copilot-budget-mcp "${bundle_root}/binaries/windows_amd64" "$ldflags"

cat > "${bundle_root}/manifest.json" <<EOF
{
  "name": "copilot-token-budget",
  "version": "${VERSION}",
  "platforms": ["windows_amd64"],
  "binaries": ["copilot-analyze.exe", "copilot-dashboard.exe", "copilot-statusline.exe", "copilot-alert.exe", "copilot-budget-mcp.exe"]
}
EOF

cp "$VSIX_PATH" "${bundle_root}/extension/"

bundle_zip="${OUTPUT_DIR}/copilot-token-budget-windows-${VERSION}.zip"
mkdir -p "$OUTPUT_DIR"
(cd "$staging_dir" && zip -qr "$bundle_zip" "copilot-token-budget-windows-${VERSION}")
"${hash_cmd[@]}" "$bundle_zip" > "${bundle_zip}.sha256"

rm -rf "${OUTPUT_DIR}/copilot-token-budget-windows-${VERSION}"
unzip -q "$bundle_zip" -d "$OUTPUT_DIR"

echo "$bundle_zip"
echo "${bundle_zip}.sha256"

rm -rf "$staging_dir"
