# Changelog

All notable changes to GoMail will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - Production Readiness Sprint

### 🚧 Sprint 1: Critical Security & Testing Foundation (2025-08-16 to 2025-08-26)

#### Added
- [x] Test infrastructure with testify framework
- [x] Unit tests for API server (target: 40% coverage - achieved 42.1%)
- [x] Integration test suite
- [x] Panic recovery middleware
- [x] Input validation framework
- [x] Structured logging with zap
- [x] Security audit tooling (gosec, nancy, trivy)
- [x] Request ID tracking
- [x] PRODUCTION-AUDIT-REPORT.md documenting current state
- [x] PRODUCTION-READINESS-PLAN.md with 6-week roadmap

#### Changed
- [x] Replaced log.Printf with structured logging
- [x] Updated CLAUDE.md with production readiness status
- [x] Updated README.md with current development status

#### Security
- [x] Added panic recovery to prevent crashes
- [x] Implemented input validation and sanitization
- [x] Added security scanning to CI pipeline

### 🚧 Sprint 2: Core Functionality Hardening (2025-08-16 to 2025-08-26)

#### Progress
- **Test Coverage:** 48.2% (Target: 50%)
- **Completed:** 5/6 features (83%)

#### Completed
- [x] Rate limiting middleware (60 req/min default)
  - Token bucket algorithm implementation
  - Configurable via rate_limit_per_minute and rate_limit_burst
  - X-RateLimit headers in responses
  - 100% test coverage for rate limiter
- [x] Configuration schema validation
  - JSON schema for configuration structure
  - Comprehensive field validation (ports, domains, paths, tokens)
  - New `validate` command to check configuration
  - Clear error messages for invalid configs
  - 89.3% test coverage for config package
- [x] Graceful shutdown with 30s timeout
  - Active request tracking and monitoring
  - Reject new requests during shutdown
  - Log progress every 5 seconds
  - Force shutdown after timeout
  - Comprehensive test coverage
- [x] Prometheus metrics integration
  - HTTP request metrics (duration, count, active, response size)
  - Email processing metrics (count by status, size, duration)
  - Storage operation metrics (read/write operations)
  - Rate limiting metrics (allowed vs denied)
  - Shutdown metrics (graceful vs forced)
  - Go runtime metrics (goroutines, memory, GC)
  - Configurable metrics endpoint (default :9090/metrics)
  - Comprehensive test coverage for metrics package
- [x] Error type categorization (Day 5)
  - Comprehensive error types package with 10 error categories
  - Structured error responses in JSON format
  - Error metrics tracking by type and handler
  - Proper HTTP status codes for each error type
  - Error handler middleware for centralized error handling
  - 100% test coverage for error package
- [ ] Connection pooling & Request timeout handling (Day 6-7)

### 🔜 Sprint 3: SMTP Security & Standards (2025-09-02 to 2025-09-12)

#### Planned
- [ ] TLS 1.2+ support with strong ciphers
- [ ] STARTTLS on port 25
- [ ] Port 587 submission service
- [ ] DKIM signing implementation
- [ ] SPF validation
- [ ] DMARC policy enforcement
- [ ] Connection limiting per IP
- [ ] Banned IP list management

### 🔜 Sprint 4: Operational Excellence (2025-09-12 to 2025-09-26)

#### Planned
- [ ] Backup and recovery scripts
- [ ] Prometheus/Grafana monitoring stack
- [ ] Load testing suite (1000 emails/min target)
- [ ] Operational runbooks
- [ ] Disaster recovery procedures
- [ ] AppArmor security profiles
- [ ] Advanced systemd hardening
- [ ] Kubernetes deployment manifests

---

## [1.0.0] - 2025-08-15

### Added
- Initial Go implementation replacing Node.js/bash version
- Single binary distribution (~15MB)
- Postfix SMTP integration on port 25
- HTTP API server on port 3000
- Bearer token authentication
- JSON file storage for emails
- Interactive quickstart wizard
- One-line installation script
- Domain management commands
- Basic health and metrics endpoints
- Systemd service configuration
- DigitalOcean DNS automation support

### Known Issues
- No test coverage (0%)
- No panic recovery mechanisms
- No TLS/STARTTLS support
- No rate limiting
- Basic logging only (no structure/levels)
- No port 587 submission support
- Plain text bearer token storage
- Limited input validation
- No DKIM/SPF/DMARC validation

---

## [0.9.0] - 2025-08-01 (Legacy Node.js)

### Final Node.js/Bash Version
- Node.js mail API server
- Bash installation scripts
- Separate postfix configuration scripts
- Multiple script dependencies

### Deprecated
- Entire Node.js/bash implementation replaced by Go

---

## Migration Notes

### From Node.js (0.9.0) to Go (1.0.0)
1. Backup existing configuration and data
2. Stop Node.js mail-api service
3. Install GoMail using quickinstall.sh
4. Configuration is mostly compatible (minor adjustments needed)
5. Data format remains the same (JSON files)

### Breaking Changes
- Binary name changed from `mail-api` to `gomail`
- Service name changed from `mail-api` to `gomail`
- Configuration file moved from `/etc/mail-api/` to `/etc/gomail.yaml`
- Some environment variable names changed (see documentation)

---

## Version History

- **1.0.0** (2025-08-15): Initial Go release
- **0.9.0** (2025-08-01): Final Node.js version
- **0.x.x**: Legacy Node.js development versions

---

## Links

- [Production Audit Report](PRODUCTION-AUDIT-REPORT.md)
- [Production Readiness Plan](PRODUCTION-READINESS-PLAN.md)
- [Contributing Guidelines](CONTRIBUTING.md)
- [GitHub Repository](https://github.com/grumpyguvner/gomail)
- [Issue Tracker](https://github.com/grumpyguvner/gomail/issues)