.PHONY: build clean test install

# Build variables
BINARY_NAME=easyprofiler-mcp
GO=go
GOFLAGS=-v

# Build the binary
build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME)

# Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BINARY_NAME).exe

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BINARY_NAME)

# Build for macOS
build-mac:
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BINARY_NAME)

# Build for all platforms
build-all: build-windows build-linux build-mac

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	$(GO) clean

# Download dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Run tests
test:
	$(GO) test -v ./...

# Install the binary
install:
	$(GO) install

# Run the server (for testing)
run: build
	./$(BINARY_NAME)

# Format code
fmt:
	$(GO) fmt ./...

# Run linter
lint:
	golangci-lint run

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build for current platform"
	@echo "  build-windows - Build for Windows (amd64)"
	@echo "  build-linux   - Build for Linux (amd64)"
	@echo "  build-mac     - Build for macOS (amd64)"
	@echo "  build-all     - Build for all platforms"
	@echo "  clean         - Remove build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  test          - Run tests"
	@echo "  install       - Install binary"
	@echo "  run           - Build and run server"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
