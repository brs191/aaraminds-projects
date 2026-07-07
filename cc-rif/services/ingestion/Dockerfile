# =============================================================================
# Ingestion Service — multi-stage Dockerfile
#
# Stage layout:
#   go-builder   — compiles the Go binary (CGO_ENABLED=0 for static linking)
#   jar-stage    — holds the extractor JAR (sourced from Maven target/ output)
#   final        — eclipse-temurin:17-jre-alpine
#
# Why eclipse-temurin:17-jre-alpine instead of distroless/base-debian12?
#
#   The extractor runs as a subprocess via `java -jar`; distroless/base does
#   not include a JVM. distroless/java17-debian12 is Debian-based (~120 MB
#   overhead vs Alpine's ~50 MB JRE). We use the Alpine JRE image to minimise
#   the final image size while still having `java` in PATH.
#
#   Phase 2 migration: when extraction moves to a dedicated Extract Service
#   container, the Ingestion Service final stage reverts to
#   gcr.io/distroless/base-debian12 (no JVM needed). That is a
#   non-breaking change to this Dockerfile only.
#
# Build context: the build must be invoked from repository root so that
#   extractors/core-java/target/rif-extractor-1.0.0-SNAPSHOT-shaded.jar
# is reachable from the build context root.
# =============================================================================

# ── Stage 1: build Go binary ──────────────────────────────────────────────────
FROM golang:1.25-alpine AS go-builder

WORKDIR /build

# Copy module files first for layer-cached dependency download.
COPY services/ingestion/go.mod services/ingestion/go.sum ./services/ingestion/
COPY libs/graphstore/go.mod libs/graphstore/go.sum ./libs/graphstore/
COPY libs/phase5/go.mod libs/phase5/go.sum ./libs/phase5/

# Download dependencies.
RUN cd services/ingestion && go mod download

# Copy the full source for both modules.
COPY services/ingestion/ ./services/ingestion/
COPY libs/graphstore/ ./libs/graphstore/
COPY libs/phase5/ ./libs/phase5/

# Build a statically-linked binary (no CGO) for maximum portability.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -C services/ingestion -ldflags="-s -w" -o /app/ingestion-service ./...

# ── Stage 2: extractor JAR ────────────────────────────────────────────────────
FROM eclipse-temurin:17-jre-alpine AS jar-stage

# The shaded (uber) JAR produced by `mvn package -DskipTests`.
# Build this first: cd extractors/core-java && mvn package -DskipTests
COPY extractors/core-java/target/rif-extractor-1.0.0-SNAPSHOT-shaded.jar /app/rif-extractor.jar

# ── Stage 3: final image ──────────────────────────────────────────────────────
# eclipse-temurin:17-jre-alpine provides:
#   - A JRE so the extractor subprocess can run
#   - Alpine minimal footprint (~200 MB total image)
#   - No shell in production (CMD uses exec form)
FROM eclipse-temurin:17-jre-alpine

RUN apk add --no-cache git ca-certificates

# Ensure clone scratch space can be created inside the container even without
# the ephemeral volume mount (local dev / unit test runs).
RUN mkdir -p /tmp/rif-clones && chmod 1777 /tmp/rif-clones

COPY --from=go-builder /app/ingestion-service /app/ingestion-service
COPY --from=jar-stage  /app/rif-extractor.jar /app/rif-extractor.jar

# Default environment — all overridden by the Container App environment stanza.
ENV EXTRACTOR_JAR_PATH=/app/rif-extractor.jar \
    CLONE_DIR=/mnt/ephemeral/rif-clones \
    PORT=8080 \
    LOG_LEVEL=info

EXPOSE 8080

ENTRYPOINT ["/app/ingestion-service"]
