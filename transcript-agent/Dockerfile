# syntax=docker/dockerfile:1
# Multi-stage build: React UI (node) -> Go backend -> distroless runtime.
# The runtime image serves the built UI (WEB_DIST) and the API from one
# binary, with migrations baked in for STORAGE=postgres.
# NOTE: untested in this environment (no docker available) — see README.md.

# --- stage 1: build the React UI -------------------------------------------
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# --- stage 2: build the Go backend ------------------------------------------
FROM golang:1.26-alpine AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server
# Empty, nonroot-owned data dir for the runtime image (distroless has no RUN).
RUN mkdir -p /out/data

# --- stage 3: runtime ---------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=backend /out/server /app/server
COPY --from=web /src/web/dist /app/web/dist
COPY backend/migrations /app/migrations
COPY --from=backend --chown=nonroot:nonroot /out/data /data
ENV WEB_DIST=/app/web/dist \
    MIGRATIONS_DIR=/app/migrations \
    DATA_DIR=/data \
    PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/server"]
