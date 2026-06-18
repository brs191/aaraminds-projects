#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ARTIFACT_DIR="${REPO_ROOT}/dist"
VSIX_PATH=""
OUTPUT_DIR="${REPO_ROOT}/distr/v1.0.0"
VERSION=""

usage() {
  cat <<'EOF'
Usage:
  ./scripts/package_macos_oob.sh --version v1.0.0 [--artifact-dir dist] [--vsix path/to.vsix] [--output-dir distr/v1.0.0]

Behavior:
  - Creates a single macOS bundle zip for Intel + Apple Silicon.
  - Copies the bootstrap installer, the VS Code .vsix, and the macOS binaries.
  - The output zip can be handed to a team member who then runs install.sh.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --artifact-dir)
      [[ $# -ge 2 ]] || { echo "Error: --artifact-dir requires a directory" >&2; exit 1; }
      ARTIFACT_DIR="$2"
      shift 2
      ;;
    --vsix)
      [[ $# -ge 2 ]] || { echo "Error: --vsix requires a file path" >&2; exit 1; }
      VSIX_PATH="$2"
      shift 2
      ;;
    --output-dir)
      [[ $# -ge 2 ]] || { echo "Error: --output-dir requires a directory" >&2; exit 1; }
      OUTPUT_DIR="$2"
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

require_cmd tar
require_cmd zip

hash_cmd=()
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
  vsix_candidates=("${REPO_ROOT}"/dist-vsix/*.vsix "${REPO_ROOT}"/extension/*.vsix)
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

mkdir -p "$OUTPUT_DIR"
staging_dir="$(mktemp -d "${TMPDIR:-/tmp}/copilot-token-budget-macos-oob.XXXXXX")"
bundle_root="${staging_dir}/copilot-token-budget-macos-${VERSION}"
mkdir -p "$bundle_root/binaries/darwin_amd64" "$bundle_root/binaries/darwin_arm64" "$bundle_root/extension"

cp "${SCRIPT_DIR}/install_macos_oob.sh" "${bundle_root}/install.sh"
chmod +x "${bundle_root}/install.sh"

cat > "${bundle_root}/launch-caveman-demo.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="${SCRIPT_DIR}/examples/token-optimization-demo"

if command -v code >/dev/null 2>&1; then
  exec code "$TARGET"
fi

exec open -a "Visual Studio Code" "$TARGET"
EOF
chmod +x "${bundle_root}/launch-caveman-demo.sh"

if [[ -d "${REPO_ROOT}/examples/token-optimization-demo" ]]; then
  mkdir -p "${bundle_root}/examples/token-optimization-demo"
  cp -R "${REPO_ROOT}/examples/token-optimization-demo/." "${bundle_root}/examples/token-optimization-demo/"
fi

cat > "${bundle_root}/manifest.json" <<EOF
{
  "name": "copilot-token-budget",
  "version": "${VERSION}",
  "platforms": ["darwin_amd64", "darwin_arm64"],
  "binaries": ["copilot-analyze", "copilot-dashboard", "copilot-statusline", "copilot-alert", "copilot-budget-mcp"]
}
EOF

copy_from_archive() {
  local binary="$1"
  local arch="$2"
  local archive="${ARTIFACT_DIR}/${binary}_${VERSION}_${arch}.tar.gz"
  local target_dir="${bundle_root}/binaries/${arch}"
  local tmpdir
  local found

  if [[ -f "$archive" ]]; then
    tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/ctb-archive.XXXXXX")"
    tar -xzf "$archive" -C "$tmpdir"
    found="$(find "$tmpdir" -type f -name "$binary" | head -n 1 || true)"
    if [[ -z "$found" ]]; then
      echo "Error: could not find ${binary} inside ${archive}" >&2
      exit 1
    fi
    cp "$found" "${target_dir}/${binary}"
    chmod +x "${target_dir}/${binary}"
    rm -rf "$tmpdir"
    return 0
  fi

  found="$(find "$ARTIFACT_DIR" -type f -name "$binary" | grep "${arch}" | head -n 1 || true)"
  if [[ -z "$found" ]]; then
    echo "Error: missing artifact for ${binary} (${arch}) in ${ARTIFACT_DIR}" >&2
    exit 1
  fi
  cp "$found" "${target_dir}/${binary}"
  chmod +x "${target_dir}/${binary}"
}

binaries=(
  copilot-analyze
  copilot-dashboard
  copilot-statusline
  copilot-alert
  copilot-budget-mcp
)

for arch in darwin_amd64 darwin_arm64; do
  for binary in "${binaries[@]}"; do
    copy_from_archive "$binary" "$arch"
  done
done

cp "$VSIX_PATH" "${bundle_root}/extension/"

bundle_zip="${OUTPUT_DIR}/copilot-token-budget-macos-${VERSION}.zip"
(cd "$staging_dir" && zip -qr "$bundle_zip" "copilot-token-budget-macos-${VERSION}")

"${hash_cmd[@]}" "$bundle_zip" > "${bundle_zip}.sha256"

echo "$bundle_zip"
echo "${bundle_zip}.sha256"

rm -rf "$staging_dir"
