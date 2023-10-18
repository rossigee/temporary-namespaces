BINARY_NAME := temporary-deployments
BINARY_NAME_ARM64 := temporary-deployments-arm64

build:
	@echo "Building binary for amd64..."
	@GOARCH=amd64 go build -o $(BINARY_NAME) .

build-arm64:
	@echo "Building binary for arm64..."
	@GOARCH=arm64 go build -o $(BINARY_NAME_ARM64) .

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME_ARM64)

all: build build-arm64
	@echo "Build completed for amd64 and arm64."

.PHONY: build build-arm64 clean all
