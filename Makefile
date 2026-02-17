.PHONY: build clean test help install

# Build output
BINARY_NAME := gust
BINARY_PATH := ./$(BINARY_NAME)
BIN_DIR := ./bin

help:
	@echo "gust - Zephyr RTOS TUI development"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build the binary at ./$(BINARY_NAME)"
	@echo "  install     Build and install to \$$GOBIN (default: \$$GOPATH/bin)"
	@echo "  test        Run all tests"
	@echo "  clean       Remove binary and build artifacts"
	@echo "  release     Build release binaries with goreleaser"

build:
	CGO_ENABLED=1 go build -o $(BINARY_PATH) ./cmd/gust

install: build
	go install ./cmd/gust

test:
	go test ./...

clean:
	rm -f $(BINARY_PATH)
	go clean

release:
	goreleaser release --snapshot --clean
