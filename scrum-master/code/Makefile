.PHONY: up down logs test test-go test-py build-go tidy fmt clean

# Run the whole P0 slice end-to-end (postgres + jira-mcp + teams-adapter + orchestrator).
up:
	docker compose up --build

down:
	docker compose down -v

logs:
	docker compose logs -f orchestrator

# Tests
test: test-go test-py

test-go:
	cd apps/jira-mcp && go test ./...
	cd apps/teams-adapter && go test ./...

test-py:
	cd apps/orchestrator && PYTHONPATH=. python -m pytest -q || PYTHONPATH=. python -m pytest tests -q

# Compile the Go services (sanity).
build-go:
	cd apps/jira-mcp && go build ./...
	cd apps/teams-adapter && go build ./...

tidy:
	cd apps/jira-mcp && go mod tidy
	cd apps/teams-adapter && go mod tidy

fmt:
	cd apps/jira-mcp && go fmt ./...
	cd apps/teams-adapter && go fmt ./...

clean:
	docker compose down -v --rmi local 2>/dev/null || true
