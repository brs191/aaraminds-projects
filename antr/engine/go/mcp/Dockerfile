# Build stage — compile the MCP server binary
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache module downloads separately from source
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /mcp-server ./mcp/...

# Runtime stage — minimal distroless image (no shell, no package manager)
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /mcp-server /mcp-server

# MCP_HTTP_PORT activates Streamable HTTP transport (Teams bot).
# Unset = stdio transport (Copilot CLI).
# Container Apps ingress routes HTTPS:443 → HTTP:8080.
EXPOSE 8080

ENTRYPOINT ["/mcp-server"]
