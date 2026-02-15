.PHONY: build install clean test

BINARY := diary-cli
BUILD_DIR := ./cmd/diary-cli

build:
	go build -o $(BINARY) $(BUILD_DIR)

install:
	go install $(BUILD_DIR)

clean:
	rm -f $(BINARY)

test:
	go test ./...
