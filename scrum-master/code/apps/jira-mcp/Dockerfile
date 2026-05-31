# Build
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/jira-mcp .

# Run
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/jira-mcp /jira-mcp
EXPOSE 8080
ENTRYPOINT ["/jira-mcp"]
