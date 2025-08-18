.PHONY: build install clean test run dev help

# Default target
all: help

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
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-webadmin -v ./cmd/webadmin

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
	@if ! which golangci-lint > /dev/null 2>&1; then \
		if [ -f $(HOME)/go/bin/golangci-lint ]; then \
			echo "Using golangci-lint from ~/go/bin"; \
		else \
			echo "Installing golangci-lint..."; \
			go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		fi \
	fi
	@if which golangci-lint > /dev/null 2>&1; then \
		golangci-lint run --timeout=5m ./...; \
	else \
		$(HOME)/go/bin/golangci-lint run --timeout=5m ./...; \
	fi

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

# Security scanning with gosec
sec-gosec:
	@if ! which gosec > /dev/null 2>&1; then \
		if [ -f $(HOME)/go/bin/gosec ]; then \
			echo "Using gosec from ~/go/bin"; \
		else \
			echo "Installing gosec..."; \
			go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		fi \
	fi
	@echo "Running gosec security scan..."
	@if which gosec > /dev/null 2>&1; then \
		gosec -fmt json -out gosec-report.json ./... || true; \
		gosec ./...; \
	else \
		$(HOME)/go/bin/gosec -fmt json -out gosec-report.json ./... || true; \
		$(HOME)/go/bin/gosec ./...; \
	fi

# Dependency vulnerability scanning with nancy
sec-nancy:
	@if ! which nancy > /dev/null 2>&1; then \
		if [ -f $(HOME)/go/bin/nancy ]; then \
			echo "Using nancy from ~/go/bin"; \
		else \
			echo "Installing nancy..."; \
			go install github.com/sonatype-nexus-community/nancy@latest; \
		fi \
	fi
	@echo "Running nancy dependency scan..."
	@go list -json -deps ./... | $(HOME)/go/bin/nancy sleuth

# Container/binary scanning with trivy
sec-trivy:
	@if ! which trivy > /dev/null 2>&1; then \
		if [ -f $(HOME)/go/bin/trivy ]; then \
			echo "Using trivy from ~/go/bin"; \
		else \
			echo "Installing trivy..."; \
			curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b $(HOME)/go/bin; \
		fi \
	fi
	@echo "Running trivy filesystem scan..."
	@$(HOME)/go/bin/trivy fs --scanners vuln,misconfig,secret .

# Run all security scans
security: sec-gosec sec-nancy sec-trivy
	@echo "✓ All security scans completed!"

# Run all checks (what CI runs)
check: fmt-check lint test security build
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
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-webadmin-linux-amd64 -v ./cmd/webadmin
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 -v ./cmd/mailserver
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-webadmin-linux-arm64 -v ./cmd/webadmin

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

# Release preparation (with full checks)
release-prep: check
	@echo "Preparing release..."
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release-prep VERSION=v1.0.1"; \
		exit 1; \
	fi

# Release preparation (without security checks - for fixing critical issues)
release-prep-nosec: fmt-check lint test build
	@echo "Preparing release (skipping security checks)..."
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release-prep-nosec VERSION=v1.0.1"; \
		exit 1; \
	fi
	@echo "Building release binaries for version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)/release
	@echo "  • Building gomail linux-amd64..."
	@GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 ./cmd/mailserver
	@echo "  • Building gomail linux-arm64..."
	@GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-arm64 ./cmd/mailserver
	@echo "  • Building gomail darwin-amd64..."
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 ./cmd/mailserver
	@echo "  • Building gomail darwin-arm64..."
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 ./cmd/mailserver
	@echo "  • Building webadmin linux-amd64..."
	@GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-webadmin-linux-amd64 ./cmd/webadmin
	@echo "  • Building webadmin linux-arm64..."
	@GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-webadmin-linux-arm64 ./cmd/webadmin
	@echo "  • Building webadmin darwin-amd64..."
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-webadmin-darwin-amd64 ./cmd/webadmin
	@echo "  • Building webadmin darwin-arm64..."
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-webadmin-darwin-arm64 ./cmd/webadmin
	@echo "  • Creating checksums..."
	@cd $(BUILD_DIR)/release && sha256sum $(BINARY_NAME)-* > checksums.txt
	@echo "  • Copying installation scripts..."
	@cp scripts/quickinstall.sh $(BUILD_DIR)/release/
	@cp example.mailserver.yaml $(BUILD_DIR)/release/
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
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/release/$(BINARY_NAME)-webadmin-linux-amd64 ./cmd/webadmin
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

# Quick release (skip full test suite)
release-quick: fmt-check lint build
	@echo "Quick release check passed"
	@$(MAKE) release-prep VERSION=$(VERSION)

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

# Release without security checks (for critical fixes)
release-nosec: release-prep-nosec
	@echo "════════════════════════════════════════════════════════════"
	@echo "Release $(VERSION) prepared successfully (security checks skipped)!"
	@echo "════════════════════════════════════════════════════════════"
	@echo "⚠️  WARNING: Security checks were skipped for this release"
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
	@echo "  make check         - Run all CI checks (fmt, lint, test, security, build)"
	@echo "  make pre-push      - Quick checks before pushing"
	@echo ""
	@echo "Security:"
	@echo "  make security      - Run all security scans"
	@echo "  make sec-gosec     - Run gosec static analysis"
	@echo "  make sec-nancy     - Run nancy dependency scan"
	@echo "  make sec-trivy     - Run trivy vulnerability scan"
	@echo ""
	@echo "Release:"
	@echo "  make release VERSION=v1.0.1       - Full release (all checks + build)"
	@echo "  make release-quick VERSION=v1.0.1 - Quick release (skip tests)"
	@echo "  make release-prep VERSION=v1.0.1  - Prepare release artifacts only"
	@echo "  make release-build VERSION=v1.0.1 - Test single platform build"
	@echo "  make release-tag VERSION=v1.0.1   - Create release tag"
	@echo ""
	@echo "Installation:"
	@echo "  make system-install - Full system installation"
	@echo "  make config        - Generate configuration file"