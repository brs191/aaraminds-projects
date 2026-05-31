# Build
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/teams-adapter .

# Run
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/teams-adapter /teams-adapter
EXPOSE 8090
ENTRYPOINT ["/teams-adapter"]
