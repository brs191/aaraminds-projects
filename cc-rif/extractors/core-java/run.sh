#!/usr/bin/env bash
set -euo pipefail
JAR="$(dirname "$0")/target/rif-extractor-1.0.0-SNAPSHOT-shaded.jar"
exec java -Xmx2g -jar "$JAR" "$@"
