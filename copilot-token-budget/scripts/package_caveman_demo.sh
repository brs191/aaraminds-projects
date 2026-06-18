#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUTPUT_DIR="${REPO_ROOT}/distr/v1.0.0"
VERSION="v1.0.0"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/package_caveman_demo.sh [--output-dir distr/v1.0.0]

Behavior:
  - Builds a lean Caveman companion ZIP.
  - Includes a launcher script and the token-optimization demo workspace.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output-dir)
      [[ $# -ge 2 ]] || { echo "Error: --output-dir requires a directory" >&2; exit 1; }
      OUTPUT_DIR="$2"
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

if [[ "$OUTPUT_DIR" != /* ]]; then
  OUTPUT_DIR="${REPO_ROOT}/${OUTPUT_DIR}"
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required command not found: $1" >&2
    exit 1
  fi
}

require_cmd zip
require_cmd shasum

mkdir -p "$OUTPUT_DIR"
staging_dir="$(mktemp -d "${TMPDIR:-/tmp}/copilot-token-budget-caveman.XXXXXX")"
bundle_root="${staging_dir}/caveman-demo"

mkdir -p "${bundle_root}/examples/token-optimization-demo"
cp -R "${REPO_ROOT}/examples/token-optimization-demo/." "${bundle_root}/examples/token-optimization-demo/"

cat > "${bundle_root}/README.md" <<'EOF'
# Caveman Demo

Unzip this folder and run `./launch-caveman-demo.sh`.

The workspace is under `examples/token-optimization-demo/`.
EOF

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

bundle_zip="${OUTPUT_DIR}/caveman-demo.zip"
(cd "$staging_dir" && zip -qr "$bundle_zip" "caveman-demo")
shasum -a 256 "$bundle_zip" > "${bundle_zip}.sha256"

echo "$bundle_zip"
echo "${bundle_zip}.sha256"

rm -rf "$staging_dir"
