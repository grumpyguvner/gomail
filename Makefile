.PHONY: build install clean test run dev

# Build variables
BINARY_NAME=gomail
BUILD_DIR=build
INSTALL_PATH=/usr/local/bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build the binary
build:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v ./cmd/mailserver

# Install the binary to system
install: build
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/
	sudo chmod 755 $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✓ Installed to $(INSTALL_PATH)/$(BINARY_NAME)"

# Run the server locally (development)
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) server

# Development mode with auto-reload (requires air)
dev:
	air -c .air.toml

# Run tests
test:
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run linter (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run --timeout=5m ./...

# Format code
fmt:
	$(GOCMD) fmt ./...
	@echo "✓ Code formatted"

# Check formatting
fmt-check:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files need formatting:"; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "✓ All files are properly formatted"

# Run all checks (what CI runs)
check: fmt-check lint test build
	@echo "✓ All checks passed!"

# Quick check before pushing (faster than full check)
pre-push: fmt-check lint build
	$(GOTEST) -short ./...
	@echo "✓ Pre-push checks passed!"

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build for multiple platforms
build-all: build-linux build-darwin

build-linux:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 -v ./cmd/mailserver

build-darwin:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 -v ./cmd/mailserver

# Full system installation
system-install: build
	@echo "Installing mail server system-wide..."
	sudo ./$(BUILD_DIR)/$(BINARY_NAME) install
	@echo "✓ System installation complete"

# Generate configuration
config:
	./$(BUILD_DIR)/$(BINARY_NAME) config generate

# Help target
help:
	@echo "Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make install       - Install binary to $(INSTALL_PATH)"
	@echo "  make run           - Run the server locally"
	@echo "  make test          - Run tests with race detection and coverage"
	@echo "  make lint          - Run golangci-lint"
	@echo "  make fmt           - Format code"
	@echo "  make fmt-check     - Check code formatting"
	@echo "  make check         - Run all CI checks (fmt, lint, test, build)"
	@echo "  make pre-push      - Quick checks before pushing"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make deps          - Download dependencies"
	@echo "  make system-install - Full system installation"
	@echo "  make config        - Generate configuration file"