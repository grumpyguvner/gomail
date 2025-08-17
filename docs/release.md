# GoMail Release Process

## Release Types

### Patch Release (x.x.PATCH)
- Bug fixes only
- No new features
- No breaking changes
- Can be released immediately for critical fixes

### Minor Release (x.MINOR.x)
- New features
- Backward compatible changes
- Performance improvements
- Documentation updates

### Major Release (MAJOR.x.x)
- Breaking changes
- Major architectural changes
- Incompatible API changes
- Significant feature additions

## Pre-Release Checklist

### Code Quality
- [ ] All tests passing (`make test`)
- [ ] Test coverage meets target (85%)
- [ ] Linter passing (`make lint`)
- [ ] Security scan clean (`make security-audit`)
- [ ] No vulnerable dependencies (`nancy go.sum`)

### Documentation
- [ ] CHANGELOG.md updated
- [ ] Documentation reflects changes
- [ ] API documentation current
- [ ] Migration guide (if breaking changes)

### Testing
- [ ] Manual testing completed
- [ ] Integration tests passing
- [ ] Load testing (for minor/major releases)
- [ ] Backward compatibility verified

## Release Process

### 1. Prepare Release Branch

```bash
# Create release branch
git checkout -b release/v1.2.3

# Update version in code if needed
# Update CHANGELOG.md with release date
# Commit changes
git add .
git commit -m "chore: Prepare release v1.2.3"
```

### 2. Run Release Build

```bash
# Full release with all checks
make release VERSION=v1.2.3

# This will:
# - Run all tests
# - Run linter
# - Build for all platforms
# - Create release archives
# - Generate checksums
```

### 3. Quick Release (Optional)

For patch releases when confident:

```bash
# Skip tests for faster release
make release-quick VERSION=v1.2.3
```

### 4. Tag Release

```bash
# Create annotated tag
make release-tag VERSION=v1.2.3

# Or manually
git tag -a v1.2.3 -m "Release v1.2.3"

# Push tag
git push origin v1.2.3
```

### 5. Create GitHub Release

```bash
# Using GitHub CLI
gh release create v1.2.3 \
  --title "Release v1.2.3" \
  --notes-file RELEASE_NOTES.md \
  build/release/*.tar.gz \
  build/release/*.zip \
  build/release/checksums.txt
```

Or manually:
1. Go to GitHub releases page
2. Click "Draft a new release"
3. Select the tag
4. Add release title and notes
5. Upload release artifacts
6. Publish release

### 6. Update Installation Script

```bash
# Update quickinstall.sh if needed
# Test installation script with new release
curl -sSL https://github.com/grumpyguvner/gomail/releases/download/v1.2.3/quickinstall.sh | bash
```

## Release Artifacts

### Binary Naming Convention

```
gomail-<os>-<arch>
```

Examples:
- `gomail-linux-amd64`
- `gomail-linux-arm64`
- `gomail-darwin-amd64`
- `gomail-darwin-arm64`
- `gomail-windows-amd64.exe`

### Archive Structure

```
gomail-v1.2.3-linux-amd64.tar.gz
├── gomail                  # Binary
├── LICENSE                 # License file
├── README.md              # Basic readme
├── example.mailserver.yaml # Example config
└── quickinstall.sh        # Installation script
```

### Checksums

Generate SHA256 checksums:

```bash
# Automated in Makefile
make checksums

# Manual
sha256sum build/release/* > checksums.txt
```

## Version Injection

Version is injected at build time:

```go
// cmd/mailserver/version.go
var (
    Version   = "dev"
    GitCommit = "unknown"
    BuildDate = "unknown"
)
```

Build with version:
```bash
go build -ldflags "-X main.Version=v1.2.3 -X main.GitCommit=$(git rev-parse HEAD) -X main.BuildDate=$(date -u +%Y-%m-%d)"
```

## Release Notes Template

```markdown
# Release v1.2.3

## 🎉 Highlights
- Brief summary of major changes
- Key features or fixes

## ✨ Features
- Feature 1 (#123)
- Feature 2 (#124)

## 🐛 Bug Fixes
- Fix 1 (#125)
- Fix 2 (#126)

## 📚 Documentation
- Documentation updates

## 🔧 Maintenance
- Dependency updates
- Internal improvements

## ⚠️ Breaking Changes (if any)
- Change 1
- Migration instructions

## 📦 Installation

### Quick Install
\`\`\`bash
curl -sSL https://github.com/grumpyguvner/gomail/releases/download/v1.2.3/quickinstall.sh | sudo bash
\`\`\`

### Manual Download
- [Linux AMD64](https://github.com/grumpyguvner/gomail/releases/download/v1.2.3/gomail-linux-amd64)
- [Linux ARM64](https://github.com/grumpyguvner/gomail/releases/download/v1.2.3/gomail-linux-arm64)
- [macOS AMD64](https://github.com/grumpyguvner/gomail/releases/download/v1.2.3/gomail-darwin-amd64)
- [macOS ARM64](https://github.com/grumpyguvner/gomail/releases/download/v1.2.3/gomail-darwin-arm64)

## 👥 Contributors
- @contributor1
- @contributor2

## 📊 Stats
- X commits since last release
- Y files changed
- Z tests added

**Full Changelog**: https://github.com/grumpyguvner/gomail/compare/v1.2.2...v1.2.3
```

## Automation

### GitHub Actions Release

`.github/workflows/release.yml`:
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build Release
        run: make release VERSION=${{ github.ref_name }}
      
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            build/release/*.tar.gz
            build/release/*.zip
            build/release/checksums.txt
          draft: false
          prerelease: false
```

## Post-Release Tasks

### Immediate
- [ ] Verify release artifacts downloadable
- [ ] Test installation script
- [ ] Update documentation site
- [ ] Announce release (if significant)

### Within 24 Hours
- [ ] Monitor for issues
- [ ] Respond to user feedback
- [ ] Update roadmap if needed

### Within 1 Week
- [ ] Gather metrics on adoption
- [ ] Plan next release
- [ ] Address any reported issues

## Rollback Procedure

If a release has critical issues:

1. **Mark as Pre-release** on GitHub
2. **Document issues** in release notes
3. **Prepare patch release** immediately
4. **Communicate** with users

```bash
# Revert tag locally
git tag -d v1.2.3
git push origin :refs/tags/v1.2.3

# Create new patch release
make release VERSION=v1.2.4
```

## Security Releases

For security fixes:

1. **Don't disclose** vulnerability details initially
2. **Prepare fix** in private
3. **Release immediately** when ready
4. **Announce** with CVE if applicable
5. **Update** security advisory

## Release Communication

### Channels
- GitHub Release Notes
- GitHub Discussions announcement
- Update README.md badge
- Email major users (if applicable)

### Messaging Template
```
GoMail v1.2.3 Released!

Key improvements:
- Feature X for better performance
- Fixed issue Y affecting Z users
- Enhanced security with feature W

Upgrade: https://github.com/grumpyguvner/gomail/releases/v1.2.3
```

## Makefile Release Targets

```bash
# Full release process
make release VERSION=v1.2.3

# Individual steps
make release-prep VERSION=v1.2.3   # Build artifacts
make release-tag VERSION=v1.2.3    # Create git tag
make checksums                     # Generate checksums

# Platform-specific builds
make build-linux VERSION=v1.2.3
make build-darwin VERSION=v1.2.3
make build-windows VERSION=v1.2.3
```

## Support

- Release issues: Create GitHub issue
- Release automation: Check GitHub Actions
- Release verification: See post-release checklist