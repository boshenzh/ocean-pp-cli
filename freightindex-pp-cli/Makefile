.PHONY: build test lint install clean

build:
	go build -o bin/freightindex-pp-cli ./cmd/freightindex-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/freightindex-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/freightindex-pp-mcp ./cmd/freightindex-pp-mcp

install-mcp:
	go install ./cmd/freightindex-pp-mcp

build-all: build build-mcp
