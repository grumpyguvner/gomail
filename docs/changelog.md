# Changelog

All notable changes to GoMail are documented here. Format based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### In Progress
- Sprint 4: Operational Excellence
  - Load testing suite
  - Monitoring dashboards
  - Backup procedures
  - Kubernetes support

## [1.4.0] - 2025-08-17

### Added - Sprint 3a: Web Administration UI
- Web-based administration interface (BFF architecture)
- HTTPS server on port 443 with existing SSL certificates
- Comprehensive domain health monitoring system
  - DNS record validation (A, MX, TXT, PTR)
  - SPF record validation and syntax checking
  - DKIM record discovery and validation
  - DMARC policy checking and compliance
  - SSL certificate validation and expiry monitoring
  - Deliverability testing (blacklist checking, reputation scoring)
- Bearer token authentication for web interface
- Custom optimized CSS (no external dependencies)
- Real-time updates via Server-Sent Events (SSE)
- Domain management interface
- Email routing rules configuration
- Systemd service integration (gomail-webadmin)
- Installation integration with main GoMail installer

### Changed
- Makefile updated to build webadmin binary
- Installation process enhanced with webadmin deployment
- Added /cmd/webadmin/ directory for BFF server

### Technical
- Pure Go implementation without npm dependencies
- Embedded static files for single binary distribution
- Parallel health checks with caching
- SPA routing without hash URLs

## [1.3.0] - 2025-08-17

### Added - Sprint 3: SMTP Security & Standards
- TLS 1.2+ enforcement with strong cipher suites
- STARTTLS support on port 25 for opportunistic encryption
- SPF validation using go-msgauth library
- DKIM verification and optional signing for outgoing mail
- DMARC policy enforcement with configurable levels
- Connection limiting per IP address
- IP ban management with automatic violation detection
- Connection throttling with token bucket rate limiting
- Comprehensive authentication metrics (SPF/DKIM/DMARC pass/fail rates)
- TLS connection metrics (versions, cipher suites, handshake duration)

### Changed
- Architecture clarified as API-driven (no port 587 submission service)
- Documentation reorganized into /docs directory
- Removed references to email client support (IMAP/POP3)

### Security
- Enforced minimum TLS 1.2 for all encrypted connections
- Added strong cipher suite preferences
- Implemented email authentication to prevent spoofing
- Added connection security to prevent abuse

## [1.2.0] - 2025-08-16

### Added - Sprint 2: Core Functionality Hardening
- Rate limiting middleware (60 req/min default with token bucket)
- Configuration schema validation with JSON schema
- Graceful shutdown with 30-second timeout
- Prometheus metrics integration
- Error type categorization (10 distinct error types)
- Connection pooling for storage operations
- Request timeout handling with context propagation
- Comprehensive HTTP timeout configuration
- X-RateLimit headers in API responses

### Changed
- Test coverage increased to 56.4%
- Error responses now use structured JSON format
- Metrics endpoint moved to port 9090 by default

### Fixed
- Memory leaks in connection handling
- Race conditions in concurrent operations
- Timeout handling in middleware stack

## [1.1.0] - 2025-08-16

### Added - Sprint 1: Critical Security & Testing Foundation
- Test infrastructure with testify framework
- Unit tests achieving 42.1% coverage
- Integration test suite
- Panic recovery middleware
- Input validation framework
- Structured logging with zap
- Security audit tooling (gosec, nancy, trivy)
- Request ID tracking for debugging
- Environment variable configuration support

### Changed
- Replaced all log.Printf with structured logging
- Improved error messages throughout
- Configuration now validates on startup

### Security
- Added panic recovery to prevent server crashes
- Implemented comprehensive input validation
- Added security scanning to CI pipeline

## [1.0.0] - 2025-08-15

### Added - Initial Go Release
- Complete rewrite from Node.js to Go
- Single binary distribution (~15MB)
- Postfix SMTP integration on port 25
- HTTP API server on port 3000
- Bearer token authentication
- JSON file storage for emails
- Interactive quickstart wizard
- One-line installation script
- Domain management CLI commands
- Health and metrics endpoints
- Systemd service configuration
- DigitalOcean DNS automation

### Changed
- Entire codebase migrated from Node.js/bash to Go
- Simplified deployment to single binary
- Improved performance and resource usage

### Known Issues
- No TLS/STARTTLS support (added in v1.3.0)
- No email authentication (added in v1.3.0)
- Limited test coverage (improved in v1.1.0+)

## [0.9.0] - 2025-08-01 - Final Node.js Version

### Deprecated
- Node.js implementation deprecated in favor of Go
- Bash installation scripts replaced
- Multiple script dependencies consolidated

---

## Migration Guide

### From 0.9.0 (Node.js) to 1.x (Go)

1. **Backup existing data**
   ```bash
   cp -r /etc/mail-api /tmp/mail-api-backup
   cp -r /opt/mailserver/data /tmp/mailserver-backup
   ```

2. **Stop old service**
   ```bash
   systemctl stop mail-api
   systemctl disable mail-api
   ```

3. **Install GoMail**
   ```bash
   curl -sSL https://github.com/grumpyguvner/gomail/releases/latest/download/quickinstall.sh | sudo bash
   ```

4. **Migrate configuration**
   - Token: Copy `apiToken` to `bearer_token`
   - Domain: Copy `domain` to `primary_domain`
   - Port: Copy `apiPort` to `port`

5. **Verify installation**
   ```bash
   systemctl status gomail
   curl http://localhost:3000/health
   ```

### Breaking Changes by Version

#### v1.0.0
- Binary name: `mail-api` → `gomail`
- Service name: `mail-api` → `gomail`
- Config location: `/etc/mail-api/` → `/etc/gomail.yaml`
- Environment variables: Different prefix (`MAIL_`)

#### v1.3.0
- Port 587 removed (API-only architecture)
- Authentication now through REST API only
- No SMTP AUTH support

---

## Version Summary

| Version | Date | Major Changes |
|---------|------|---------------|
| 1.3.0 | 2025-08-17 | Security & Authentication (Sprint 3) |
| 1.2.0 | 2025-08-16 | Core Hardening (Sprint 2) |
| 1.1.0 | 2025-08-16 | Testing & Security Foundation (Sprint 1) |
| 1.0.0 | 2025-08-15 | Initial Go release |
| 0.9.0 | 2025-08-01 | Final Node.js version |

---

[Unreleased]: https://github.com/grumpyguvner/gomail/compare/v1.3.0...HEAD
[1.3.0]: https://github.com/grumpyguvner/gomail/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/grumpyguvner/gomail/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/grumpyguvner/gomail/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/grumpyguvner/gomail/compare/v0.9.0...v1.0.0
[0.9.0]: https://github.com/grumpyguvner/gomail/releases/tag/v0.9.0