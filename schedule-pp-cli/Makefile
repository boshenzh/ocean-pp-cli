.PHONY: build test lint install clean

build:
	go build -o bin/schedule-pp-cli ./cmd/schedule-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/schedule-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/schedule-pp-mcp ./cmd/schedule-pp-mcp

install-mcp:
	go install ./cmd/schedule-pp-mcp

build-all: build build-mcp
