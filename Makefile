.PHONY: build test test-race lint fmt verify

build:
	go build -trimpath -o bin/voice-asset-mcp ./cmd/voice-asset-mcp

test:
	go test -cover ./...

test-race:
	go test -race -cover ./...

lint:
	go vet ./...

fmt:
	gofmt -w .

verify: lint test build
