# Contributing to GoMail

Thank you for your interest in contributing to GoMail! This guide will help you get started.

## Code of Conduct

By participating in this project, you agree to:
- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive criticism
- Accept feedback gracefully

## How to Contribute

### Reporting Bugs

1. **Check existing issues** first to avoid duplicates
2. **Use the bug report template** when creating issues
3. **Include relevant information**:
   - GoMail version (`gomail --version`)
   - Operating system and version
   - Steps to reproduce
   - Expected vs actual behavior
   - Error messages and logs

### Suggesting Features

1. **Check the roadmap** in `/docs/production-status.md`
2. **Open a feature request** with:
   - Use case description
   - Proposed solution
   - Alternative solutions considered
   - Impact on existing features

### Submitting Code

#### First Time Contributors

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/gomail.git
   cd gomail
   ```

3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/grumpyguvner/gomail.git
   ```

4. Create a branch:
   ```bash
   git checkout -b feature/your-feature
   ```

#### Development Process

1. **Write tests first** (TDD approach)
2. **Implement your changes**
3. **Ensure all tests pass**:
   ```bash
   make test
   ```

4. **Check code quality**:
   ```bash
   make check
   ```

5. **Update documentation** if needed

#### Pull Request Process

1. **Update your branch**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Push your changes**:
   ```bash
   git push origin feature/your-feature
   ```

3. **Create Pull Request** with:
   - Clear title and description
   - Reference to related issues
   - List of changes made
   - Screenshots if UI changes

4. **Address review feedback**:
   - Make requested changes
   - Push updates to the same branch
   - Respond to all comments

## Development Guidelines

### Code Style

- Follow Go idioms and best practices
- Use `gofmt` for formatting
- Follow existing patterns in the codebase
- Keep functions small and focused
- Add comments for complex logic

### Testing Requirements

- **Minimum 70% coverage** for new code
- Write unit tests for all functions
- Include edge cases and error conditions
- Use table-driven tests where appropriate
- Mock external dependencies

Example test:
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "TEST", false},
        {"empty input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Feature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Feature() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("Feature() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Documentation

- Update relevant docs in `/docs/`
- Add godoc comments to exported functions
- Include examples in complex features
- Update CHANGELOG.md for user-facing changes

### Commit Guidelines

Use conventional commits format:

```
type(scope): brief description

Longer explanation if needed.

Fixes #123
```

Examples:
- `feat(api): Add batch email processing`
- `fix(auth): Correct DMARC alignment check`
- `docs: Update installation guide`
- `test(storage): Add connection pool tests`

## Priority Areas

Current areas needing contributions:

### High Priority
1. **Test Coverage**: Increase to 85% target
2. **Load Testing**: Implement comprehensive load tests
3. **Documentation**: API examples and tutorials
4. **Security**: Additional security features

### Good First Issues
- Adding unit tests to low-coverage packages
- Documentation improvements
- Code cleanup and refactoring
- Simple bug fixes

Look for issues labeled `good first issue` or `help wanted`.

## Review Process

### What We Look For

1. **Code Quality**
   - Clean, readable code
   - Proper error handling
   - No code duplication
   - Performance considerations

2. **Testing**
   - Comprehensive test coverage
   - Tests pass locally and in CI
   - Edge cases covered

3. **Documentation**
   - Code is self-documenting
   - Complex logic explained
   - User docs updated if needed

4. **Security**
   - No hardcoded secrets
   - Input validation
   - Safe concurrency patterns

### Review Timeline

- Initial review: Within 48 hours
- Follow-up reviews: Within 24 hours
- Merge decision: After 2 approvals

## Release Process

### Version Numbering

We follow semantic versioning (MAJOR.MINOR.PATCH):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes

### Release Cycle

- **Patch releases**: As needed for critical fixes
- **Minor releases**: Every 2-4 weeks
- **Major releases**: When breaking changes required

## Getting Help

### Resources

- [Development Guide](development.md)
- [Architecture Documentation](architecture.md)
- [API Documentation](api.md)
- GitHub Discussions for questions

### Communication Channels

- **GitHub Issues**: Bug reports and features
- **GitHub Discussions**: Questions and discussions
- **Pull Requests**: Code contributions

## Recognition

Contributors are recognized in:
- CHANGELOG.md for their contributions
- GitHub contributors page
- Release notes

## Legal

By contributing, you agree that your contributions will be licensed under the MIT License.

## Thank You!

Your contributions make GoMail better for everyone. We appreciate your time and effort!