.PHONY: build test lint install clean

build:
	go build -o bin/webprofile-pp-cli ./cmd/webprofile-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/webprofile-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/webprofile-pp-mcp ./cmd/webprofile-pp-mcp

install-mcp:
	go install ./cmd/webprofile-pp-mcp

build-all: build build-mcp
