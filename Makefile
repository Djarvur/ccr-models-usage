.PHONY: build test lint ci clean
GO ?= go
BIN_DIR ?= bin
BINARY ?= $(BIN_DIR)/ccr-models-usage

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BINARY) ./cmd/ccr-models-usage

test:
	$(GO) test ./...

lint:
	golangci-lint run ./...

ci: lint test build

clean:
	rm -rf $(BIN_DIR)
