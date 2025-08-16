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
	$(GOTEST) -v ./...

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
	@echo "  make test          - Run tests"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make deps          - Download dependencies"
	@echo "  make system-install - Full system installation"
	@echo "  make config        - Generate configuration file"