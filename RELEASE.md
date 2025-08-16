# Release Process

This document describes the release process for GoMail.

## Prerequisites

- Ensure you have push access to the repository
- Ensure all tests are passing on main branch
- Ensure CHANGELOG.md is updated with release notes

## Release Steps

### 1. Prepare Release Locally

Choose one of the following options:

#### Option A: Full Release (Recommended)
```bash
# Run all checks and build release artifacts
make release VERSION=v1.0.1

# This will:
# - Check code formatting
# - Run golangci-lint
# - Run full test suite with race detection
# - Build binaries for all platforms
# - Generate checksums
# - Copy installation scripts
# - Create release artifacts in build/release/
```

#### Option B: Quick Release (When confident)
```bash
# Skip tests for faster release
make release-quick VERSION=v1.0.1

# This will:
# - Check code formatting
# - Run golangci-lint
# - Build a test binary
# - Build all platform binaries
# - Generate checksums
# - Create release artifacts in build/release/
```

### 2. Review Release Artifacts

```bash
# Check the generated files
ls -la build/release/

# Test a binary locally
./build/release/gomail-linux-amd64 --version
```

### 3. Create and Push Tag

```bash
# Create annotated tag
make release-tag VERSION=v1.0.1

# Push the tag (this triggers GitHub Actions release workflow)
git push origin v1.0.1
```

### 4. Monitor GitHub Actions

The release workflow will automatically:
- Run tests
- Build binaries with version injection
- Create GitHub release with all artifacts
- Upload binaries, checksums, and installation scripts

```bash
# Monitor the release workflow
gh run watch
```

### 5. Verify Release

```bash
# Check the release was created
gh release view v1.0.1

# Test the installation script
curl -sSL https://github.com/grumpyguvner/gomail/releases/latest/download/quickinstall.sh | sudo bash
```

## Quick Release Commands

For experienced maintainers:

```bash
# Full release process with all checks
make release VERSION=v1.0.2 && git push origin v1.0.2

# Quick release (skip tests)
make release-quick VERSION=v1.0.2 && git push origin v1.0.2

# Just test the build
make release-build VERSION=v1.0.2

# Just prepare artifacts without checks
make release-prep VERSION=v1.0.2
```

## Version Numbering

We follow semantic versioning (semver):
- MAJOR version (v2.0.0) - Incompatible API changes
- MINOR version (v1.1.0) - New functionality, backwards compatible
- PATCH version (v1.0.1) - Bug fixes, backwards compatible

## Troubleshooting

### Release workflow fails

1. Check GitHub Actions logs:
   ```bash
   gh run list --workflow=release.yml
   gh run view <run-id>
   ```

2. Run checks locally:
   ```bash
   make check
   ```

### Binary doesn't show correct version

Ensure you're using the VERSION parameter:
```bash
make release-build VERSION=v1.0.1
./build/release/gomail-linux-amd64 --version  # Should show v1.0.1
```

### Tag already exists

Delete the local tag and try again:
```bash
git tag -d v1.0.1
git push origin :refs/tags/v1.0.1  # Delete remote tag (be careful!)
```

## Emergency Release

If you need to create a release quickly:

```bash
# Skip local checks (not recommended)
git tag v1.0.3 && git push origin v1.0.3
```

The GitHub Actions workflow will still run all checks.