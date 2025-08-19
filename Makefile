BINARY_NAME := temporary-namespaces
BINARY_NAME_ARM64 := temporary-namespaces-arm64
GOLANGCI_LINT_VERSION := v2.4.0

build:
	@echo "Building binary for amd64..."
	@GOARCH=amd64 go build -o $(BINARY_NAME) .

build-arm64:
	@echo "Building binary for arm64..."
	@GOARCH=arm64 go build -o $(BINARY_NAME_ARM64) .

test:
	@echo "Running tests..."
	@go test -v ./...

lint:
	@echo "Running linter with golangci-lint $(GOLANGCI_LINT_VERSION)..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	fi
	@golangci-lint run

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME_ARM64)

all: build build-arm64
	@echo "Build completed for amd64 and arm64."

.PHONY: build build-arm64 test lint clean all
