#!/usr/bin/env bash
# Build + run the Phase-1 resolving extractor against a built credit-routing-service
# checkout. Needs JDK 17 + Maven (the research sandbox has only a JDK 11 JRE, so this
# is the "finish in a proper env" path). The classpath is harvested OFFLINE from the
# fat jar's BOOT-INF/lib — no mvn/.m2 needed for the *target's* deps.
set -euo pipefail

REPO="${1:?usage: run.sh <path-to-credit-routing-service@44b6b86> [workdir]}"
WORK="${2:-./_work}"
HERE="$(cd "$(dirname "$0")" && pwd)"
SHA="$(git -C "$REPO" rev-parse --short HEAD)"
mkdir -p "$WORK/lib"

# 1) harvest the resolved dependency set (194 jars) from the Spring Boot fat jar
JAR="$(ls "$REPO"/target/*-SNAPSHOT.jar | head -1)"
( cd "$WORK/lib" && unzip -j -o "$JAR" 'BOOT-INF/lib/*.jar' >/dev/null )

# 2) classpath = compiled classes (Lombok + JAXB bytecode) + dep jars
CP="$REPO/target/classes"
for j in "$WORK"/lib/*.jar; do CP="$CP:$j"; done

# 3) source roots = hand-written + generated JAXB
GEN="$(ls -d "$REPO"/target/generated-sources/*/src/main/java 2>/dev/null | paste -sd, -)"

# 4) build the extractor (JDK 17 + Maven required here)
mvn -q -f "$HERE/pom.xml" -DskipTests package

# 5) extract -> graph.json, then assert the provenance gate
java -jar "$HERE/target/extractor.jar" \
  --repo "$(basename "$REPO")" --sha "$SHA" --basepkg com.att.creditcheck \
  --src "$REPO/src/main/java" --gen "$GEN" --classpath "$CP" \
  --out "$WORK/graph.json"

python3 "$HERE/../eval/provenance_check.py" "$WORK/graph.json"
echo "next: load with ../loader/load_age.py (regenerate against \$WORK/graph.json) then run the AGE benchmark (gate G7)"
