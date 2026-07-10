GO ?= go

.PHONY: build test postgres-test vet run tidy clean

build:
	$(GO) build -o bin/server ./cmd/server

test:
	$(GO) test ./...

postgres-test:
	$(GO) test ./internal/app -run TestPostgresBackedWorkflowIntegration -count=1

vet:
	$(GO) vet ./...

run:
	$(GO) run ./cmd/server

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin data
