GO ?= go

.PHONY: build test vet run tidy clean

build:
	$(GO) build -o bin/server ./cmd/server

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

run:
	$(GO) run ./cmd/server

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin data
