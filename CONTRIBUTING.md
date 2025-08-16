# Contributing to GoMail

Thank you for your interest in contributing to GoMail! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

Please be respectful and constructive in all interactions. We welcome contributors of all experience levels.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git
- Make
- Linux development environment

### Setup

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/gomail.git
   cd gomail
   ```
3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/grumpyguvner/gomail.git
   ```
4. Install dependencies:
   ```bash
   make deps
   ```

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Make Your Changes

Follow these guidelines:
- Write clear, self-documenting code
- Follow Go best practices and idioms
- Keep functions small and focused
- Add comments for complex logic
- Update tests for new functionality

### 3. Format and Lint Your Code

```bash
# Format your code
make fmt

# Run the linter
make lint

# Fix any issues before proceeding
```

### 4. Run Tests

```bash
# Run all tests
make test

# Run all CI checks (recommended)
make check
```

### 5. Commit Your Changes

Write clear commit messages following this format:
```
type: Short description (50 chars or less)

Longer explanation if needed (wrap at 72 chars)

- Bullet points for multiple changes
- Another change

Fixes #123
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Formatting, missing semicolons, etc.
- `refactor`: Code restructuring
- `test`: Adding tests
- `chore`: Maintenance tasks

### 6. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub with:
- Clear title describing the change
- Description of what and why
- Reference to any related issues
- Screenshots if UI changes

## Testing

### Running Tests

```bash
# Run all tests with coverage
make test

# Quick test (no race detection)
go test ./...

# Test specific package
go test ./internal/api/...

# Verbose output
go test -v ./...
```

### Writing Tests

- Place tests in `*_test.go` files
- Use table-driven tests where appropriate
- Mock external dependencies
- Aim for good coverage of critical paths

Example test:
```go
func TestParseEmail(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *EmailData
        wantErr bool
    }{
        {
            name:  "valid email",
            input: "From: test@example.com...",
            want:  &EmailData{...},
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseEmail(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseEmail() error = %v, wantErr %v", err, tt.wantErr)
            }
            // Compare got and want...
        })
    }
}
```

## Code Style

### Go Guidelines

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting (automated via `make fmt`)
- Keep line length reasonable (80-120 chars)
- Use meaningful variable names
- Group imports (stdlib, external, internal)

### Error Handling

```go
// Good
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Bad
if err := doSomething(); err != nil {
    return err // Lost context
}
```

### Logging

```go
// Use structured logging where possible
log.Printf("Email received: from=%s to=%s size=%d", from, to, size)

// For errors
log.Printf("Failed to parse email: %v", err)
```

## Making a Release

If you're a maintainer:

1. Update CHANGELOG.md
2. Run release preparation:
   ```bash
   make release VERSION=v1.0.2
   ```
3. Create and push tag:
   ```bash
   make release-tag VERSION=v1.0.2
   git push origin v1.0.2
   ```

## Project Structure

```
gomail/
├── cmd/mailserver/     # Main application entry point
├── internal/           # Private application code
│   ├── api/           # HTTP API server
│   ├── commands/      # CLI commands
│   ├── config/        # Configuration management
│   ├── mail/          # Email parsing and processing
│   ├── postfix/       # Postfix integration
│   └── storage/       # Data storage
├── scripts/           # Installation and utility scripts
├── .github/workflows/ # GitHub Actions CI/CD
└── docs/             # Additional documentation
```

## Common Tasks

### Adding a New CLI Command

1. Create a new file in `internal/commands/`
2. Implement the command using Cobra
3. Register it in `cmd/mailserver/main.go`
4. Add tests
5. Update documentation

### Adding an API Endpoint

1. Add handler in `internal/api/server.go`
2. Add routing in `NewServer()`
3. Implement business logic
4. Add tests
5. Update API documentation in README

### Debugging

```bash
# Run with verbose output
./gomail server -v 3

# Check Postfix logs
journalctl -u postfix -f

# Check GoMail logs
journalctl -u gomail -f

# Test email delivery
make test
```

## Getting Help

- Check existing issues on GitHub
- Read the documentation in README.md
- Ask questions in issues (tag as "question")
- Review existing code for examples

## Checklist Before Submitting PR

- [ ] Code follows project style guidelines
- [ ] All tests pass (`make check`)
- [ ] Documentation updated if needed
- [ ] Commit messages are clear
- [ ] PR description explains the change
- [ ] No sensitive information in code
- [ ] Signed commits (if required)

## Recognition

Contributors will be recognized in:
- GitHub contributors page
- Release notes for significant contributions
- README acknowledgments for major features

Thank you for contributing to GoMail!