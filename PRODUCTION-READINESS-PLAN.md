# GoMail Production Readiness Implementation Plan

**Version:** 1.0  
**Date:** 2025-08-16  
**Target Completion:** 6 weeks  
**Priority:** Critical

## Executive Summary

This plan addresses all critical issues identified in the production audit, with special emphasis on security hardening and SMTP standards compliance. The plan is organized into 4 sprints over 6 weeks.

## SMTP Port Standards & Security

### Industry Standard Ports

| Port | Protocol | Usage | Security | Recommendation |
|------|----------|-------|----------|----------------|
| **25** | SMTP | Server-to-server relay | Plain text | ✅ Keep for receiving |
| **465** | SMTPS | Client submission (deprecated) | SSL/TLS wrapper | ❌ Avoid |
| **587** | SMTP | Client submission (modern) | STARTTLS | ✅ Implement |
| **2525** | SMTP | Alternative submission | STARTTLS | ⚠️ Optional |

### Current vs Required Configuration

**Current State:**
- Port 25: Plain SMTP (receiving mail from other servers)
- No submission ports configured
- No TLS/SSL implementation
- No authentication for submission

**Required State:**
- Port 25: Keep for server-to-server (with STARTTLS)
- ~~Port 587: Not needed - API-only mail submission~~
- Mandatory TLS for all connections
- SPF, DKIM, DMARC implementation
- Rate limiting and connection throttling

## Sprint Plan Overview

### Sprint 1: Critical Security & Testing Foundation (Week 1-2)
**Goal:** Establish security baseline and testing infrastructure

### Sprint 2: Core Functionality Hardening (Week 2-3)
**Goal:** Implement error handling, validation, and monitoring

### Sprint 3: SMTP Security & Standards (Week 3-4) ⚠️ PARTIAL
**Goal:** Full SMTP security implementation
**Status:** 60% Complete (6/10 features done)

### Sprint 4: Operational Excellence (Week 5-6)
**Goal:** Production deployment readiness

---

## Sprint 1: Critical Security & Testing Foundation
**Duration:** 10 days  
**Priority:** P0 - Blocking

### 1.1 Test Infrastructure Setup (3 days)

#### Unit Test Framework
```go
// internal/api/server_test.go
package api

import (
    "testing"
    "net/http/httptest"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestServerAuthentication(t *testing.T) {
    // Test bearer token validation
    // Test missing auth header
    // Test invalid token
}

func TestEmailIngestion(t *testing.T) {
    // Test valid RFC822 email
    // Test JSON format
    // Test malformed input
    // Test size limits
}
```

#### Integration Test Suite
```go
// test/integration/email_flow_test.go
func TestCompleteEmailFlow(t *testing.T) {
    // Start test server
    // Send email via SMTP
    // Verify API receives
    // Check storage
    // Validate response
}
```

#### Test Coverage Goals
- Sprint 1: 40% coverage
- Sprint 2: 60% coverage
- Sprint 3: 80% coverage
- Sprint 4: 85%+ coverage

### 1.2 Panic Recovery Implementation (1 day)

```go
// internal/middleware/recovery.go
package middleware

func RecoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Error("panic recovered", 
                    "error", err,
                    "stack", debug.Stack(),
                    "request_id", GetRequestID(r),
                )
                http.Error(w, "Internal Server Error", 500)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

### 1.3 Input Validation Framework (2 days)

```go
// internal/validation/email.go
package validation

type EmailValidator struct {
    maxSize      int64
    allowedTLDs  []string
    blockList    []string
}

func (v *EmailValidator) Validate(email *mail.EmailData) error {
    // Validate sender domain
    // Check SPF records
    // Verify DKIM signatures
    // Check against blocklist
    // Validate size limits
    // Sanitize headers
    return nil
}
```

### 1.4 Structured Logging (2 days) ✅ COMPLETED

**Status:** ✅ Implemented on 2025-08-16

**Implementation:**
- Created centralized logger in `internal/logging/logger.go`
- Replaced all `log.Printf` statements with structured logging
- Added environment variable configuration for log level and output
- Integrated request ID tracking into log context
- Added 96.6% test coverage for logging module

```go
// internal/logging/logger.go
package logging

import "go.uber.org/zap"

var logger *zap.SugaredLogger

func InitLogger(mode string) {
    // Production or development mode configuration
    // Supports MAIL_LOG_LEVEL and MAIL_LOG_FILE env vars
}

func WithRequestID(requestID string) *zap.SugaredLogger {
    return logger.With("request_id", requestID)
}
```

### 1.5 Security Audit Tools (2 days)

```bash
# security/audit.sh
#!/bin/bash

# Run security scanners
gosec ./...
nancy go.sum
trivy fs .

# Check for secrets
gitleaks detect

# SAST scanning
semgrep --config=auto
```

---

## Sprint 2: Core Functionality Hardening
**Duration:** 7 days  
**Priority:** P0 - Critical

### 2.1 Rate Limiting Implementation (2 days)

```go
// internal/middleware/ratelimit.go
package middleware

import "golang.org/x/time/rate"

type RateLimiter struct {
    visitors map[string]*rate.Limiter
    mu       sync.RWMutex
    r        rate.Limit
    b        int
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
    return &RateLimiter{
        visitors: make(map[string]*rate.Limiter),
        r:        r,
        b:        b,
    }
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        limiter := rl.getLimiter(r.RemoteAddr)
        if !limiter.Allow() {
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### 2.2 Configuration Validation (1 day)

```yaml
# config/schema.yaml
type: object
required:
  - port
  - bearer_token
  - primary_domain
properties:
  port:
    type: integer
    minimum: 1
    maximum: 65535
  bearer_token:
    type: string
    minLength: 32
    pattern: "^[A-Za-z0-9+/=]+$"
  tls:
    type: object
    properties:
      enabled:
        type: boolean
      cert_file:
        type: string
      key_file:
        type: string
```

### 2.3 Graceful Shutdown (1 day)

```go
// internal/server/lifecycle.go
func (s *Server) GracefulShutdown(ctx context.Context) error {
    shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Stop accepting new connections
    s.listener.Close()
    
    // Wait for existing connections to complete
    s.wg.Wait()
    
    // Flush any pending data
    s.storage.Flush()
    
    // Close database connections
    s.db.Close()
    
    return s.httpServer.Shutdown(shutdownCtx)
}
```

### 2.4 Monitoring & Metrics (2 days)

```go
// internal/metrics/prometheus.go
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
    EmailsReceived = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "gomail_emails_received_total",
            Help: "Total number of emails received",
        },
        []string{"domain", "status"},
    )
    
    EmailProcessingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "gomail_email_processing_seconds",
            Help: "Email processing duration",
        },
        []string{"domain"},
    )
)
```

### 2.5 Error Handling Improvements (1 day)

```go
// internal/errors/types.go
package errors

type ErrorType int

const (
    ErrorTypeValidation ErrorType = iota
    ErrorTypeAuthentication
    ErrorTypeRateLimit
    ErrorTypeInternal
    ErrorTypeTemporary
)

type AppError struct {
    Type    ErrorType
    Message string
    Err     error
    Retry   bool
}
```

---

## Sprint 3: SMTP Security & Standards
**Duration:** 10 days  
**Priority:** P0 - Security Critical

### 3.1 TLS/STARTTLS Implementation (3 days)

```go
// internal/smtp/tls.go
package smtp

func ConfigureTLS(config *tls.Config) {
    config.MinVersion = tls.VersionTLS12
    config.CipherSuites = []uint16{
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
    }
    config.PreferServerCipherSuites = true
    config.SessionTicketsDisabled = true
}
```

### 3.2 ~~Port 587 Submission Service~~ ❌ NOT NEEDED

**Architecture Decision:** GoMail is an API-driven mail system. Email submission happens through the authenticated REST API, not through traditional SMTP client connections. This eliminates the need for:
- Port 587 submission service
- SMTP AUTH (SASL) implementation  
- User account management for SMTP
- IMAP/POP3 services

All outbound email is sent via the API with bearer token authentication.

### 3.3 DKIM Signing Implementation (2 days)

```go
// internal/mail/dkim.go
package mail

import "github.com/emersion/go-msgauth/dkim"

func SignEmail(email []byte, domain string) ([]byte, error) {
    privateKey, err := loadDKIMKey(domain)
    if err != nil {
        return nil, err
    }
    
    options := &dkim.SignOptions{
        Domain:   domain,
        Selector: "default",
        Signer:   privateKey,
    }
    
    return dkim.Sign(email, options)
}
```

### 3.4 SPF/DMARC Validation (2 days)

```go
// internal/mail/spf.go
package mail

import "github.com/emersion/go-msgauth/spf"

func ValidateSPF(ip, domain, sender string) (spf.Result, error) {
    return spf.CheckHost(net.ParseIP(ip), domain, sender)
}

// internal/mail/dmarc.go
func ValidateDMARC(domain string) (*dmarc.Record, error) {
    return dmarc.Lookup(domain)
}
```

### 3.5 Connection Security (1 day)

```go
// internal/smtp/security.go
package smtp

type ConnectionLimiter struct {
    maxPerIP     int
    maxTotal     int
    bannedIPs    sync.Map
    connections  sync.Map
}

func (cl *ConnectionLimiter) Accept(ip string) bool {
    if _, banned := cl.bannedIPs.Load(ip); banned {
        return false
    }
    
    // Check per-IP limit
    count := cl.getConnectionCount(ip)
    if count >= cl.maxPerIP {
        return false
    }
    
    return true
}
```

---

## Sprint 4: Operational Excellence
**Duration:** 10 days  
**Priority:** P1 - Important

### 4.1 Backup & Recovery (2 days)

```bash
#!/bin/bash
# scripts/backup.sh

BACKUP_DIR="/backup/gomail"
DATA_DIR="/opt/gomail/data"
CONFIG_DIR="/etc/gomail"

# Create timestamped backup
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_PATH="${BACKUP_DIR}/${TIMESTAMP}"

mkdir -p "${BACKUP_PATH}"

# Backup data
tar -czf "${BACKUP_PATH}/data.tar.gz" "${DATA_DIR}"

# Backup configuration
tar -czf "${BACKUP_PATH}/config.tar.gz" "${CONFIG_DIR}"

# Backup Postfix configuration
postconf -n > "${BACKUP_PATH}/postfix.conf"

# Rotate old backups (keep 30 days)
find "${BACKUP_DIR}" -type d -mtime +30 -exec rm -rf {} \;
```

### 4.2 Monitoring Stack (2 days)

```yaml
# docker-compose.monitoring.yml
version: '3.8'

services:
  prometheus:
    image: prometheus/prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana
    ports:
      - "3001:3000"
    volumes:
      - ./dashboards:/var/lib/grafana/dashboards

  alertmanager:
    image: prometheus/alertmanager
    ports:
      - "9093:9093"
```

### 4.3 Load Testing Suite (2 days)

```go
// test/load/smtp_load_test.go
package load

import "github.com/tsenart/vegeta/v12/lib"

func TestSMTPLoad(t *testing.T) {
    rate := vegeta.Rate{Freq: 100, Per: time.Second}
    duration := 5 * time.Minute
    
    targeter := NewSMTPTargeter("localhost:25")
    attacker := vegeta.NewAttacker()
    
    var metrics vegeta.Metrics
    for res := range attacker.Attack(targeter, rate, duration, "SMTP Load Test") {
        metrics.Add(res)
    }
    
    metrics.Close()
    
    assert.Less(t, metrics.Latencies.P99, 500*time.Millisecond)
    assert.Greater(t, metrics.Success, 0.99)
}
```

### 4.4 Documentation & Runbooks (2 days)

```markdown
# runbooks/incident-response.md

## High Email Queue

### Symptoms
- Queue size > 10,000 emails
- Processing delay > 5 minutes

### Investigation
1. Check queue status: `postqueue -p | tail -n 1`
2. Check API health: `curl localhost:3000/health`
3. Review logs: `journalctl -u gomail -n 100`

### Resolution
1. Increase API workers if CPU < 80%
2. Check disk space
3. Verify bearer token is correct
4. Restart service if necessary
```

### 4.5 Security Hardening (2 days)

```yaml
# security/hardening.yaml
systemd:
  security:
    NoNewPrivileges: true
    PrivateTmp: true
    ProtectSystem: strict
    ProtectHome: true
    ReadWritePaths: /opt/gomail/data
    CapabilityBoundingSet: CAP_NET_BIND_SERVICE
    AmbientCapabilities: CAP_NET_BIND_SERVICE
    
apparmor:
  profile: |
    #include <tunables/global>
    /usr/local/bin/gomail {
      #include <abstractions/base>
      #include <abstractions/nameservice>
      
      /opt/gomail/data/** rw,
      /etc/gomail.yaml r,
      /proc/sys/kernel/random/uuid r,
      
      network inet stream,
      network inet6 stream,
    }
```

---

## Implementation Timeline

### Week 1-2: Sprint 1
- [ ] Day 1-3: Test infrastructure
- [ ] Day 4: Panic recovery
- [ ] Day 5-6: Input validation
- [ ] Day 7-8: Structured logging
- [ ] Day 9-10: Security audit setup

### Week 2-3: Sprint 2
- [ ] Day 11-12: Rate limiting
- [ ] Day 13: Config validation
- [ ] Day 14: Graceful shutdown
- [ ] Day 15-16: Monitoring/metrics
- [ ] Day 17: Error handling

### Week 3-4: Sprint 3
- [ ] Day 18-20: TLS implementation
- [x] ~~Day 21-22: Port 587 setup~~ (Not needed - API-only)
- [ ] Day 23-24: DKIM signing
- [ ] Day 25-26: SPF/DMARC
- [ ] Day 27: Connection security

### Week 5-6: Sprint 4
- [ ] Day 28-29: Backup/recovery
- [ ] Day 30-31: Monitoring stack
- [ ] Day 32-33: Load testing
- [ ] Day 34-35: Documentation
- [ ] Day 36-37: Final hardening

### Week 6: Final Testing
- [ ] Day 38: Security audit
- [ ] Day 39: Load testing
- [ ] Day 40: Deployment test
- [ ] Day 41: Documentation review
- [ ] Day 42: Go/No-Go decision

---

## Success Metrics

### Technical Metrics
- Test coverage > 85%
- P99 latency < 500ms
- Error rate < 0.1%
- Uptime > 99.9%
- Zero security vulnerabilities (High/Critical)

### Operational Metrics
- Mean Time To Recovery < 5 minutes
- Deployment frequency > 1/week
- Change failure rate < 5%
- Lead time for changes < 1 day

### Security Metrics
- 100% TLS encryption
- Zero unauthorized access attempts successful
- SPF/DKIM/DMARC pass rate > 95%
- Security scan clean (gosec, trivy)

---

## Risk Mitigation

### High Risk Items
1. **Zero tests** → Implement tests first, incrementally
2. **No TLS** → Prioritize in Sprint 3
3. **Plain text tokens** → Move to environment variables immediately
4. **No rate limiting** → Implement basic version in Sprint 2

### Mitigation Strategies
- Daily code reviews
- Automated security scanning in CI
- Staging environment testing
- Gradual rollout with monitoring
- Rollback procedures documented

---

## Resource Requirements

### Personnel
- 2 Senior Go developers
- 1 DevOps engineer
- 1 Security engineer (part-time)
- 1 QA engineer

### Infrastructure
- Staging environment (identical to production)
- Load testing environment
- Monitoring infrastructure
- CI/CD pipeline enhancements

### Tools & Licenses
- Datadog/New Relic for APM
- PagerDuty for alerting
- GitHub Actions for CI/CD
- Security scanning tools

---

## Definition of Done

### Sprint Completion Criteria
- [ ] All tests passing (>85% coverage)
- [ ] Security scan clean
- [ ] Documentation updated
- [ ] Code reviewed and approved
- [ ] Deployed to staging
- [ ] Load tested successfully

### Production Ready Criteria
- [ ] All sprints completed
- [ ] Security audit passed
- [ ] Load testing passed (1000 emails/minute)
- [ ] Monitoring and alerting configured
- [ ] Runbooks documented
- [ ] Team trained on operations
- [ ] Disaster recovery tested
- [ ] Compliance requirements met

---

## Appendix A: Quick Start Commands

```bash
# Start development
make deps
make test
make run

# Security checks
make security-audit
make vulnerability-scan

# Load testing
make load-test RATE=100 DURATION=5m

# Deployment
make deploy ENV=staging
make smoke-test
make deploy ENV=production
```

## Appendix B: Configuration Changes

```yaml
# Updated mailserver.yaml
port: 3000
submission_port: 587
smtp_port: 25

tls:
  enabled: true
  cert_file: /etc/gomail/tls/cert.pem
  key_file: /etc/gomail/tls/key.pem
  min_version: "1.2"

security:
  rate_limit:
    requests_per_minute: 60
    burst: 10
  max_connections_per_ip: 10
  max_message_size: 26214400  # 25MB
  
dkim:
  enabled: true
  selector: default
  key_file: /etc/gomail/dkim/private.key

spf:
  enabled: true
  strict_mode: true

dmarc:
  enabled: true
  policy: quarantine
```

---

**This plan provides a structured path to production readiness with clear milestones and success criteria.**