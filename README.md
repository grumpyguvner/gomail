# GoMail - API-Driven Mail Server

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Sprint%203%20Complete-yellow)](docs/production-status.md)

GoMail is a modern, high-performance mail server that combines Postfix SMTP reception with HTTP API forwarding. Built as a single Go binary, it provides a programmatic interface for email handling without the complexity of traditional mail servers.

## 🚀 Quick Start

```bash
# One-line installation
curl -sSL https://github.com/grumpyguvner/gomail/releases/latest/download/quickinstall.sh | sudo bash

# Or interactive setup
sudo gomail quickstart
```

## 📋 Key Features

- **Single Binary**: 15MB executable with no runtime dependencies
- **API-Driven**: All email operations through authenticated REST API
- **Email Authentication**: SPF, DKIM, and DMARC verification
- **Security Hardened**: TLS 1.2+, STARTTLS, rate limiting, connection security
- **Multi-Domain**: Handle multiple email domains from one instance
- **Monitoring**: Built-in Prometheus metrics and health endpoints

## 🏗️ Architecture

GoMail is **NOT** a traditional mail server for email clients. It's an API-driven system designed for programmatic email handling:

```
Inbound:  [Internet] → [Port 25/SMTP] → [GoMail] → [Your API Webhook]
Outbound: [Your App] → [GoMail API] → [Port 25/SMTP] → [Internet]
```

See [Architecture Documentation](docs/architecture.md) for details.

## 📚 Documentation

- [Installation Guide](docs/installation.md) - Detailed setup instructions
- [Configuration Reference](docs/configuration.md) - All configuration options
- [API Documentation](docs/api.md) - REST API endpoints and webhooks
- [Security Features](docs/security.md) - Authentication and protection mechanisms
- [Operations Guide](docs/operations.md) - Monitoring, backup, and maintenance
- [Production Status](docs/production-status.md) - Current readiness state

## 🛠️ Development

For development setup and contribution guidelines:
- [Development Guide](docs/development.md) - Building and testing
- [Contributing](docs/contributing.md) - How to contribute
- [Release Process](docs/release.md) - Release workflow

## 📊 Current Status

GoMail has completed Sprint 3 of the production readiness plan:
- ✅ Core functionality and testing infrastructure
- ✅ Error handling, monitoring, and metrics
- ✅ TLS/STARTTLS and email authentication (SPF/DKIM/DMARC)
- ⏳ Sprint 4: Operational excellence (pending)

## 📬 Use Cases

**Ideal for:**
- Transactional email systems
- Email-to-webhook services
- Support ticket systems
- SaaS application email handling
- Email automation via APIs

**Not suitable for:**
- Personal email hosting (no IMAP/POP3)
- Email clients like Outlook/Thunderbird
- Traditional user mailboxes

## 🔗 Links

- [GitHub Repository](https://github.com/grumpyguvner/gomail)
- [Issue Tracker](https://github.com/grumpyguvner/gomail/issues)
- [Changelog](docs/changelog.md)

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.