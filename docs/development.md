# GoMail Development Guide

## Development Environment Setup

### Prerequisites

- Go 1.21 or higher
- Git
- Make
- golangci-lint (for linting)
- Linux development environment

### Initial Setup

```bash
# Clone repository
git clone https://github.com/grumpyguvner/gomail.git
cd gomail

# Install dependencies
make deps

# Install development tools
make install-tools

# Verify setup
make check
```

## Building

### Local Development Build

```bash
# Quick build for development
make build

# Run the binary
./build/gomail --version
./build/gomail server --config example.mailserver.yaml
```

### Production Build

```bash
# Build with version injection and optimizations
make release VERSION=v1.0.3

# Output will be in build/release/
ls -la build/release/
```

### Cross-Platform Builds

```bash
# Build for all platforms
make build-all

# Build for specific platforms
make build-linux    # Linux amd64 and arm64
make build-darwin   # macOS amd64 and arm64
make build-windows  # Windows amd64
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
make test-race

# Run specific package tests
go test ./internal/api/...
go test ./internal/auth/...

# Run integration tests
INTEGRATION_TEST=true make test

# Verbose output
go test -v ./...
```

### Writing Tests

Example test structure:

```go
package api_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestEmailProcessing(t *testing.T) {
    // Setup
    server := setupTestServer(t)
    defer server.Close()
    
    // Test cases
    t.Run("valid email", func(t *testing.T) {
        email := createTestEmail()
        resp, err := server.ProcessEmail(email)
        
        require.NoError(t, err)
        assert.Equal(t, 200, resp.StatusCode)
    })
    
    t.Run("invalid email", func(t *testing.T) {
        email := createInvalidEmail()
        resp, err := server.ProcessEmail(email)
        
        require.NoError(t, err)
        assert.Equal(t, 400, resp.StatusCode)
    })
}
```

### Test Coverage

```bash
# Generate coverage report
make test-coverage

# View coverage in browser
go tool cover -html=coverage.out

# Check coverage percentage
go tool cover -func=coverage.out | grep total
```

Current coverage targets:
- Minimum: 60%
- Target: 85%
- Per-package minimum: 70%

## Code Quality

### Linting

```bash
# Run linter
make lint

# Run specific linters
golangci-lint run --enable gofmt
golangci-lint run --enable govet
golangci-lint run --enable ineffassign

# Auto-fix issues
golangci-lint run --fix
```

### Formatting

```bash
# Format code
make fmt

# Check formatting
make fmt-check

# Format specific files
go fmt ./internal/api/server.go
```

### Static Analysis

```bash
# Security analysis
make security-audit

# Run gosec
gosec ./...

# Check dependencies
nancy go.sum

# Vulnerability scanning
govulncheck ./...
```

## Development Workflow

### Feature Development

1. **Create feature branch**
   ```bash
   git checkout -b feature/new-feature
   ```

2. **Write tests first (TDD)**
   ```bash
   # Create test file
   touch internal/feature/feature_test.go
   
   # Write failing test
   # Implement feature
   # Make test pass
   ```

3. **Implement feature**
   ```bash
   # Write code
   # Run tests frequently
   make test
   ```

4. **Check quality**
   ```bash
   make check  # Runs all checks
   ```

5. **Commit changes**
   ```bash
   git add .
   git commit -m "feat: Add new feature"
   ```

6. **Push and create PR**
   ```bash
   git push origin feature/new-feature
   # Create pull request on GitHub
   ```

### Bug Fixes

1. **Reproduce the bug**
   ```bash
   # Write a failing test that reproduces the issue
   ```

2. **Fix the bug**
   ```bash
   # Implement fix
   # Ensure test passes
   ```

3. **Verify fix**
   ```bash
   make test
   make check
   ```

4. **Commit with descriptive message**
   ```bash
   git commit -m "fix: Correct DKIM signature validation"
   ```

## Project Structure

```
gomail/
├── cmd/
│   └── mailserver/       # CLI entry point
│       ├── main.go      # Application entry
│       └── *.go         # CLI commands
├── internal/            # Private packages
│   ├── api/            # HTTP API server
│   ├── auth/           # Email authentication
│   ├── commands/       # CLI command implementations
│   ├── config/         # Configuration management
│   ├── errors/         # Error types and handling
│   ├── logging/        # Structured logging
│   ├── mail/           # Email processing
│   ├── metrics/        # Prometheus metrics
│   ├── middleware/     # HTTP middleware
│   ├── postfix/        # Postfix integration
│   ├── security/       # Security features
│   ├── storage/        # Data persistence
│   ├── tls/            # TLS/STARTTLS
│   └── validation/     # Input validation
├── scripts/            # Installation scripts
├── docs/              # Documentation
├── test/              # Integration tests
├── Makefile           # Build automation
├── go.mod             # Go dependencies
└── go.sum             # Dependency checksums
```

## Debugging

### Debug Build

```bash
# Build with debug symbols
go build -gcflags="all=-N -l" -o build/gomail-debug ./cmd/mailserver

# Run with delve debugger
dlv exec ./build/gomail-debug -- server --config example.mailserver.yaml
```

### Debug Logging

```bash
# Enable debug logging
MAIL_LOG_LEVEL=debug ./build/gomail server

# Or in config
log_level: debug
debug_mode: true
```

### Performance Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Run pprof web interface
go tool pprof -http=:8080 cpu.prof
```

## Makefile Targets

```bash
# Core targets
make build          # Build binary
make test           # Run tests
make lint           # Run linter
make fmt            # Format code
make check          # Run all checks

# Development targets
make run            # Run server locally
make dev            # Development mode with hot reload
make clean          # Clean build artifacts

# Release targets
make release        # Full release build
make release-quick  # Quick release (skip some checks)
make release-prep   # Prepare release artifacts

# Utility targets
make deps           # Download dependencies
make install-tools  # Install dev tools
make update-deps    # Update dependencies
```

## Configuration for Development

### Example Development Config

```yaml
# dev.yaml
mode: development
port: 3001  # Different port to avoid conflicts
log_level: debug
debug_mode: true

# Relaxed limits for testing
rate_limit_per_minute: 1000
max_message_size: 52428800  # 50MB

# Local storage
data_dir: ./tmp/data

# Disable TLS for local dev
tls_enabled: false

# Test webhook endpoint
api_endpoint: http://localhost:8080/webhook
```

Run with development config:
```bash
./build/gomail server --config dev.yaml
```

## Git Workflow

### Commit Message Format

Follow conventional commits:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Code style (formatting)
- `refactor`: Code refactoring
- `test`: Testing
- `chore`: Maintenance

Examples:
```bash
git commit -m "feat(api): Add webhook retry logic"
git commit -m "fix(auth): Correct SPF record parsing"
git commit -m "docs: Update API documentation"
git commit -m "test(mail): Add DKIM verification tests"
```

### Branch Naming

- `feature/description` - New features
- `fix/description` - Bug fixes
- `docs/description` - Documentation
- `refactor/description` - Refactoring
- `test/description` - Testing improvements

## Dependencies

### Adding Dependencies

```bash
# Add a new dependency
go get github.com/some/package

# Update go.mod and go.sum
go mod tidy

# Verify
go mod verify
```

### Updating Dependencies

```bash
# Update all dependencies
make update-deps

# Update specific dependency
go get -u github.com/some/package

# Check for available updates
go list -u -m all
```

## IDE Setup

### VS Code

`.vscode/settings.json`:
```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"],
  "go.testFlags": ["-v", "-race"],
  "go.buildTags": "integration",
  "go.testTimeout": "30s"
}
```

### GoLand/IntelliJ

1. Set Go SDK to 1.21+
2. Enable Go Modules
3. Configure golangci-lint as external tool
4. Set up run configurations for different commands

## Troubleshooting Development Issues

### Common Issues

| Issue | Solution |
|-------|----------|
| `make: command not found` | Install make: `apt-get install make` |
| Linter not found | Run `make install-tools` |
| Tests failing | Check Go version, run `go mod download` |
| Port already in use | Change port in config or kill process |
| Permission denied | Check file permissions, may need sudo |

### Getting Help

- Check existing issues on GitHub
- Review test files for usage examples
- Check `make help` for available commands
- Ask in GitHub Discussions

## Contributing

See [Contributing Guide](contributing.md) for:
- Code of conduct
- Pull request process
- Code review guidelines
- Release process