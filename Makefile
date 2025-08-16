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
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 -v ./cmd/mailserver

build-darwin:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 -v ./cmd/mailserver
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 -v ./cmd/mailserver

# Full system installation
system-install: build
	@echo "Installing mail server system-wide..."
	sudo ./$(BUILD_DIR)/$(BINARY_NAME) install
	@echo "✓ System installation complete"

# Generate configuration
config:
	./$(BUILD_DIR)/$(BINARY_NAME) config generate

# Release preparation
release-prep: check
	@echo "Preparing release..."
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release-prep VERSION=v1.0.1"; \
		exit 1; \
	fi
	@echo "Building release binaries for version $(VERSION)..."
	mkdir -p $(BUILD_DIR)/release
	# Build all platforms with version injection
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 ./cmd/mailserver
	GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-arm64 ./cmd/mailserver
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 ./cmd/mailserver
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 ./cmd/mailserver
	# Create checksums
	cd $(BUILD_DIR)/release && sha256sum $(BINARY_NAME)-* > checksums.txt
	# Copy installation scripts
	cp scripts/quickinstall.sh $(BUILD_DIR)/release/
	cp example.mailserver.yaml $(BUILD_DIR)/release/
	@echo "✓ Release artifacts created in $(BUILD_DIR)/release/"
	@echo "✓ Ready to tag and push: git tag $(VERSION) && git push origin $(VERSION)"

# Test release build without full checks
release-build:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release-build VERSION=v1.0.1"; \
		exit 1; \
	fi
	mkdir -p $(BUILD_DIR)/release
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 ./cmd/mailserver
	@echo "✓ Test build successful for $(VERSION)"

# Create and push a release tag
release-tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release-tag VERSION=v1.0.1"; \
		exit 1; \
	fi
	@echo "Creating release tag $(VERSION)..."
	git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "✓ Tag created. Push with: git push origin $(VERSION)"

# Full release process (local validation)
release: release-prep
	@echo "════════════════════════════════════════════════════════════"
	@echo "Release $(VERSION) prepared successfully!"
	@echo "════════════════════════════════════════════════════════════"
	@echo "Next steps:"
	@echo "  1. Review the release artifacts in $(BUILD_DIR)/release/"
	@echo "  2. Create and push tag: make release-tag VERSION=$(VERSION)"
	@echo "  3. Push tag: git push origin $(VERSION)"
	@echo "════════════════════════════════════════════════════════════"

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Development:"
	@echo "  make build         - Build the binary"
	@echo "  make install       - Install binary to $(INSTALL_PATH)"
	@echo "  make run           - Run the server locally"
	@echo "  make test          - Run tests with race detection and coverage"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make deps          - Download dependencies"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint          - Run golangci-lint"
	@echo "  make fmt           - Format code"
	@echo "  make fmt-check     - Check code formatting"
	@echo "  make check         - Run all CI checks (fmt, lint, test, build)"
	@echo "  make pre-push      - Quick checks before pushing"
	@echo ""
	@echo "Release:"
	@echo "  make release VERSION=v1.0.1  - Full release preparation with checks"
	@echo "  make release-prep VERSION=v1.0.1  - Prepare release artifacts"
	@echo "  make release-build VERSION=v1.0.1 - Test release build"
	@echo "  make release-tag VERSION=v1.0.1   - Create release tag"
	@echo ""
	@echo "Installation:"
	@echo "  make system-install - Full system installation"
	@echo "  make config        - Generate configuration file"