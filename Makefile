.PHONY: build install clean test

BINARY := diary-cli
BUILD_DIR := ./cmd/diary-cli

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

build:
	go build -ldflags="-s -w -X github.com/soli0222/diary-cli/internal/cli.Version=$(VERSION)" -o $(BINARY) $(BUILD_DIR)

install:
	go install $(BUILD_DIR)

clean:
	rm -f $(BINARY)

test:
	go test ./...
