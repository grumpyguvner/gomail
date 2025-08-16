# GoMail - Modern Mail Server in Go

A high-performance mail server solution that combines Postfix SMTP with HTTP API forwarding, written entirely in Go. GoMail provides a single binary that handles everything from installation to email processing.

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Pre--Production-orange)](PRODUCTION-READINESS-PLAN.md)
[![Test Coverage](https://img.shields.io/badge/Coverage-42.1%25-yellow)](CHANGELOG.md)

> ⚠️ **IMPORTANT:** GoMail is currently undergoing production readiness improvements and is NOT yet suitable for production use. Sprint 1 of 6 completed. See [Production Readiness Plan](PRODUCTION-READINESS-PLAN.md) for our 6-week roadmap to production.

## Features

- 🚀 **Single 15MB Binary** - Everything in one executable, no dependencies
- 📧 **Complete Mail Server** - Full Postfix SMTP server with TLS support
- 🔄 **API Forwarding** - HTTP webhook for received emails with JSON payloads
- 🔐 **Authentication Metadata** - SPF/DKIM/DMARC data extraction
- 🌐 **Multi-Domain Support** - Handle multiple email domains
- 🔧 **Zero Configuration** - Works out of the box with sensible defaults
- 🏗️ **Idempotent Installation** - Safe to run multiple times
- 🛡️ **Input Validation** - Email validation with configurable rules
- 🔍 **Request Tracking** - Unique request IDs for debugging
- 💪 **Panic Recovery** - Automatic recovery from unexpected errors

## Quick Start

### One-Line Installation

```bash
# Basic installation (will prompt for domain and optional DigitalOcean token)
curl -sSL https://github.com/grumpyguvner/gomail/releases/latest/download/quickinstall.sh | sudo bash

# With domain specified
curl -sSL https://github.com/grumpyguvner/gomail/releases/latest/download/quickinstall.sh | sudo bash -s example.com

# With domain and DigitalOcean token for automatic DNS setup
curl -sSL https://github.com/grumpyguvner/gomail/releases/latest/download/quickinstall.sh | sudo bash -s example.com --token YOUR_DO_TOKEN
```

That's it! GoMail is now installed and running. The installer:
- ✅ Downloads the correct binary for your system
- ✅ Generates secure configuration automatically
- ✅ Installs and configures Postfix
- ✅ Sets up your domain
- ✅ Configures DigitalOcean DNS (if token provided)
- ✅ Starts the service
- ✅ Detects fresh install vs reinstall

### Manual Installation

```bash
# Download the latest release
wget https://github.com/grumpyguvner/gomail/releases/latest/download/gomail-linux-amd64
chmod +x gomail-linux-amd64
sudo mv gomail-linux-amd64 /usr/local/bin/gomail

# Interactive setup (prompts for domain and optional DigitalOcean token)
sudo gomail quickstart

# Or with parameters
sudo gomail quickstart example.com --token YOUR_DO_TOKEN
```

## Architecture

GoMail provides a unified CLI with multiple commands:

```
gomail
├── server      # Run API server
├── install     # Install system components
├── domain      # Manage email domains
├── dns         # Configure DNS records
├── ssl         # Manage SSL certificates
├── test        # Test configuration
└── config      # Manage configuration
```

### Email Flow

```
[Email] → [Postfix:25] → [pipe transport] → [gomail API:3000] → [JSON storage]
```

## Configuration

GoMail can be configured through:
- YAML configuration file (`mailserver.yaml`)
- Environment variables (prefix: `MAIL_`)
- Command-line flags

### Example Configuration

```yaml
port: 3000
mode: simple
data_dir: /opt/mailserver/data
bearer_token: your-secure-token-here

mail_hostname: mail.example.com
primary_domain: example.com
api_endpoint: http://localhost:3000/mail/inbound
```

### Environment Variables

```bash
export MAIL_BEARER_TOKEN=your-secure-token
export MAIL_PORT=3000
export MAIL_PRIMARY_DOMAIN=example.com
```

## API Documentation

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/mail/inbound` | POST | Receive email from Postfix |
| `/health` | GET | Health check |
| `/metrics` | GET | Server metrics |

### Webhook Payload

```json
{
  "sender": "from@example.org",
  "recipient": "to@yourdomain.com",
  "received_at": "2024-01-15T10:30:00Z",
  "raw": "Complete RFC-822 email message...",
  "subject": "Email subject",
  "message_id": "<unique-id@example.org>",
  "authentication": {
    "spf": {
      "client_ip": "192.168.1.100",
      "mail_from": "sender@example.org"
    },
    "dkim": {
      "signatures": ["..."],
      "from_domain": "example.org"
    },
    "dmarc": {
      "from_header": "from@example.org",
      "return_path": "bounce@example.org"
    }
  }
}
```

## CLI Commands

### Domain Management

```bash
# Add a domain
sudo gomail domain add example.com

# List domains
gomail domain list

# Remove a domain
sudo gomail domain remove example.com

# Test domain configuration
gomail domain test example.com
```

### System Management

```bash
# Run installation
sudo gomail install

# Test the system
gomail test

# Check configuration
gomail config show

# Set configuration value
gomail config set bearer_token new-token-value
```

## Building from Source

### Requirements

- Go 1.21 or higher
- Linux (CentOS 9, Ubuntu 20.04+, Debian 11+)
- Root access for installation

### Build Instructions

```bash
# Clone the repository
git clone https://github.com/grumpyguvner/gomail.git
cd gomail

# Download dependencies
make deps

# Run all checks (recommended before pushing)
make check

# Build the binary
make build

# Install to system
sudo make install
```

### Development Workflow

```bash
# Before making changes
make deps                # Download dependencies

# During development
make build              # Build the binary
make run                # Run the server locally
make test               # Run tests with coverage
make fmt                # Format code
make lint               # Run linter

# Before committing
make check              # Run all CI checks locally
make pre-push           # Quick validation before pushing

# Build for multiple platforms
make build-all          # Build for all platforms
make build-linux        # Build for Linux (amd64 + arm64)
make build-darwin       # Build for macOS (amd64 + arm64)

# Clean up
make clean              # Remove build artifacts
```

### Release Process

```bash
# Full release with all checks
make release VERSION=v1.0.2

# Quick release (skip tests)
make release-quick VERSION=v1.0.2

# Just build release artifacts
make release-prep VERSION=v1.0.2

# Create and push tag
make release-tag VERSION=v1.0.2
git push origin v1.0.2
```

## Deployment

### Systemd Service

GoMail automatically installs as a systemd service:

```bash
# Start the service
sudo systemctl start gomail

# Enable on boot
sudo systemctl enable gomail

# Check status
systemctl status gomail

# View logs
journalctl -u gomail -f
```

### Docker (Coming Soon)

```bash
docker run -d \
  -p 25:25 \
  -p 3000:3000 \
  -v /opt/mailserver/data:/data \
  -e MAIL_BEARER_TOKEN=your-token \
  grumpyguvner/gomail
```

## Storage

Emails are stored as JSON files organized by date:

```
/opt/mailserver/data/
├── inbox/
│   └── 2024/
│       └── 01/
│           └── 15/
│               ├── msg_1705321800_a1b2c3.json
│               └── msg_1705321860_d4e5f6.json
└── processed/
```

## Security

- ✅ Bearer token authentication for API
- ✅ Not configured as open relay
- ✅ TLS encryption support
- ✅ Runs as unprivileged user
- ✅ systemd hardening
- ✅ No external dependencies

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| Port 25 blocked | Contact your hosting provider |
| API not receiving | Check bearer token configuration |
| Emails queued | Verify API service is running |
| Permission denied | Ensure proper file permissions |

### Debug Commands

```bash
# Check Postfix queue
postqueue -p

# Test email delivery
swaks --to test@yourdomain.com --server localhost

# Check API health
curl http://localhost:3000/health

# View stored emails
ls -la /opt/mailserver/data/inbox/
```

## Production Readiness Status

**Current Status:** Pre-production (actively working toward production readiness)

- 📋 [Track sprint progress](CHANGELOG.md) - See checkboxes for completed items
- 📊 [View audit findings](PRODUCTION-AUDIT-REPORT.md) - Detailed gap analysis
- 🗺️ [Implementation roadmap](PRODUCTION-READINESS-PLAN.md) - 6-week plan with all details

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for detailed instructions.

**Priority Areas for Contributors:**
1. Writing tests (target: 85% coverage)
2. Implementing security features from the roadmap
3. Documentation improvements
4. Performance optimization

### Quick Start for Contributors

```bash
# Fork and clone the repository
git clone https://github.com/YOUR_USERNAME/gomail.git
cd gomail

# Install dependencies and run checks
make deps
make check

# Make your changes and test
make test
make pre-push

# Submit your pull request
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with Go and love ❤️
- Powered by Postfix for reliable SMTP handling
- Inspired by the need for a simple, modern mail server

## Support

- 📧 Email: support@example.com
- 💬 Discord: [Join our community](https://discord.gg/example)
- 🐛 Issues: [GitHub Issues](https://github.com/grumpyguvner/gomail/issues)

---

**GoMail** - Making email servers simple again.