# Makefile for GoMail Web Admin

# Variables
TAILWIND_VERSION := v3.4.0
TAILWIND_CLI := ./bin/tailwindcss
TAILWIND_URL_LINUX := https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-linux-x64
TAILWIND_URL_MACOS := https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-macos-arm64

# Detect OS
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Linux)
	TAILWIND_URL := $(TAILWIND_URL_LINUX)
else ifeq ($(UNAME_S),Darwin)
	ifeq ($(UNAME_M),arm64)
		TAILWIND_URL := $(TAILWIND_URL_MACOS)
	else
		TAILWIND_URL := https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-macos-x64
	endif
else
	$(error Unsupported operating system: $(UNAME_S))
endif

.PHONY: all build css css-watch clean install-tailwind

all: install-tailwind css build

# Install Tailwind CSS standalone CLI
install-tailwind:
	@echo "Installing Tailwind CSS standalone CLI..."
	@mkdir -p bin
	@if [ ! -f $(TAILWIND_CLI) ]; then \
		echo "Downloading Tailwind CSS $(TAILWIND_VERSION)..."; \
		curl -sL $(TAILWIND_URL) -o $(TAILWIND_CLI); \
		chmod +x $(TAILWIND_CLI); \
		echo "Tailwind CSS installed successfully"; \
	else \
		echo "Tailwind CSS already installed"; \
	fi

# Build optimized CSS (scans HTML/JS and includes only used styles)
css: install-tailwind
	@echo "Building optimized Tailwind CSS..."
	$(TAILWIND_CLI) -i ../../webadmin/assets/css/input.css \
		-o ../../webadmin/assets/css/output.css \
		-c ../../webadmin/tailwind.config.js \
		--minify
	@echo "CSS build complete. Size: $$(du -h ../../webadmin/assets/css/output.css | cut -f1)"

# Watch mode for development
css-watch: install-tailwind
	@echo "Starting Tailwind CSS in watch mode..."
	$(TAILWIND_CLI) -i ../../webadmin/assets/css/input.css \
		-o ../../webadmin/assets/css/output.css \
		-c ../../webadmin/tailwind.config.js \
		--watch

# Build the Go binary with embedded static files
build: css
	@echo "Building webadmin binary..."
	go build -o ../../build/webadmin .
	@echo "Build complete"

# Run the webadmin server
run: build
	../../build/webadmin

# Development mode - run with live CSS rebuilding
dev:
	@echo "Starting development mode..."
	@make css-watch &
	@go run .

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(TAILWIND_CLI)
	@rm -f ../../webadmin/assets/css/output.css
	@rm -f ../../build/webadmin
	@echo "Clean complete"

# Check CSS size and show stats
css-stats: css
	@echo "CSS Statistics:"
	@echo "Input CSS size:  $$(du -h ../../webadmin/assets/css/input.css | cut -f1)"
	@echo "Output CSS size: $$(du -h ../../webadmin/assets/css/output.css | cut -f1)"
	@echo "Compression:     $$(echo "scale=2; 100 - ($$(stat -f%z ../../webadmin/assets/css/output.css 2>/dev/null || stat -c%s ../../webadmin/assets/css/output.css) * 100 / $$(stat -f%z ../../webadmin/assets/css/input.css 2>/dev/null || stat -c%s ../../webadmin/assets/css/input.css))" | bc)%"

help:
	@echo "Available targets:"
	@echo "  make              - Install dependencies, build CSS, and build binary"
	@echo "  make css          - Build optimized Tailwind CSS"
	@echo "  make css-watch    - Watch and rebuild CSS on changes"
	@echo "  make build        - Build the webadmin binary"
	@echo "  make run          - Build and run the webadmin server"
	@echo "  make dev          - Run in development mode with CSS watching"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make css-stats    - Show CSS size statistics"
	@echo "  make help         - Show this help message"