# GoMail Architecture

## System Overview

GoMail is an API-driven mail server that bridges SMTP email reception with modern HTTP webhooks. It's designed as a single Go binary that replaces complex traditional mail server stacks.

## Core Design Principles

1. **Single Binary Distribution**: Everything compiles into one ~15MB executable
2. **API-First Design**: All operations through REST API, no traditional mail client support
3. **Zero Runtime Dependencies**: No need for Node.js, Python, or other runtimes
4. **Modular Architecture**: Clean separation of concerns with internal packages
5. **Security by Default**: Authentication required, TLS enforced, rate limiting built-in

## System Architecture

### Email Flow

```
┌─────────────┐     SMTP:25      ┌──────────┐     Pipe      ┌──────────┐
│   Internet  │ ───────────────> │ Postfix  │ ────────────> │  GoMail  │
└─────────────┘                  └──────────┘               │   API    │
                                                            └────┬─────┘
                                                                 │ HTTP
                                                                 ↓
                                                          ┌──────────────┐
                                                          │ Your Webhook │
                                                          └──────────────┘
```

### Component Architecture

```
GoMail Binary
├── CLI Layer (Cobra)
│   ├── server command
│   ├── install command
│   ├── domain command
│   ├── config command
│   └── quickstart wizard
│
├── API Layer
│   ├── HTTP Server (net/http)
│   ├── Middleware Stack
│   │   ├── Recovery (panic protection)
│   │   ├── Request ID tracking
│   │   ├── Authentication
│   │   ├── Rate Limiting
│   │   ├── Timeout handling
│   │   └── Error handling
│   └── Endpoints
│       ├── /mail/inbound (POST)
│       ├── /health (GET)
│       └── /metrics (GET)
│
├── Mail Processing
│   ├── RFC822 Parser
│   ├── Authentication
│   │   ├── SPF Verification
│   │   ├── DKIM Verification
│   │   └── DMARC Enforcement
│   └── Storage Writer
│
├── Security Layer
│   ├── TLS/STARTTLS
│   ├── Connection Limiter
│   ├── IP Ban Management
│   └── Token Authentication
│
└── Storage Layer
    ├── JSON File Storage
    └── Connection Pooling
```

## Key Design Decisions

### Why API-Only (No Port 587)?

GoMail is designed for programmatic email handling, not end-user email clients:

- **No SMTP AUTH needed**: API uses bearer tokens instead
- **No user management**: No mailboxes or IMAP/POP3
- **Simplified security**: One authentication mechanism (API tokens)
- **Modern integration**: Webhooks instead of mail protocols

### Why Go?

- **Single binary deployment**: No runtime dependencies
- **Excellent concurrency**: Handles thousands of connections efficiently
- **Strong standard library**: HTTP, TLS, and networking built-in
- **Memory efficient**: Low resource usage compared to Node.js/Python
- **Fast compilation**: Quick development cycle

### Why Postfix Integration?

- **Battle-tested**: Decades of production use
- **Security**: Excellent security track record
- **Standards compliant**: Full SMTP implementation
- **Flexible**: Easy to integrate via pipe transport

## Package Structure

### `/internal/api`
HTTP server implementation with all API endpoints.

### `/internal/auth`
Email authentication protocols (SPF, DKIM, DMARC) using go-msgauth library.

### `/internal/config`
Configuration management with validation and schema enforcement.

### `/internal/mail`
Email parsing, processing, and data extraction.

### `/internal/metrics`
Prometheus metrics collection and exposure.

### `/internal/middleware`
HTTP middleware stack for cross-cutting concerns.

### `/internal/postfix`
Postfix installation, configuration, and integration.

### `/internal/security`
Connection security, rate limiting, and IP management.

### `/internal/storage`
Data persistence layer with connection pooling.

### `/internal/tls`
TLS configuration and STARTTLS implementation.

### `/internal/validation`
Input validation and sanitization.

## Security Architecture

### Defense in Depth

1. **Network Layer**
   - TLS 1.2+ enforcement
   - STARTTLS on port 25
   - Strong cipher suites only

2. **Application Layer**
   - Bearer token authentication
   - Rate limiting per IP
   - Connection limits
   - Request timeouts

3. **Email Layer**
   - SPF verification
   - DKIM signature validation
   - DMARC policy enforcement
   - Size limits

4. **System Layer**
   - Runs as unprivileged user
   - Systemd hardening
   - Panic recovery
   - Structured logging

## Data Flow

### Inbound Email Processing

1. Email arrives at Postfix on port 25
2. Postfix performs initial checks
3. Email piped to GoMail via pipe transport
4. GoMail parses RFC822 message
5. SPF/DKIM/DMARC verification performed
6. Email data extracted and structured
7. JSON payload created with metadata
8. Webhook called with retry logic
9. Email stored to disk as JSON

### API Request Flow

1. Request arrives at API endpoint
2. Request ID generated for tracking
3. Authentication middleware validates token
4. Rate limiter checks request limits
5. Timeout middleware sets deadline
6. Handler processes request
7. Response sent with appropriate headers
8. Metrics updated

## Scalability Considerations

### Current Capabilities
- Handles 1000+ emails/minute (target)
- Connection pooling for efficiency
- Concurrent request processing
- Async webhook delivery

### Future Scaling Options
- Horizontal scaling with load balancer
- Message queue integration (Redis/RabbitMQ)
- Database storage option (PostgreSQL)
- Kubernetes deployment

## Monitoring & Observability

### Metrics (Prometheus)
- Request rates and latencies
- Email processing counts
- Authentication results
- Error rates by type
- Connection pool status

### Logging (Structured)
- Request tracking with IDs
- Error details with stack traces
- Authentication decisions
- Performance measurements

### Health Checks
- `/health` endpoint for liveness
- Postfix queue monitoring
- Storage availability checks

## Configuration Strategy

Three-layer configuration with precedence:

1. **Command-line flags** (highest priority)
2. **Environment variables** (MAIL_ prefix)
3. **YAML configuration file** (base configuration)

This allows for:
- Easy containerization (env vars)
- Persistent settings (YAML)
- Quick overrides (flags)

## Integration Points

### Webhook Payload

Standardized JSON structure containing:
- Sender/recipient information
- Full RFC822 message
- Extracted metadata
- Authentication results
- Processing timestamps

### API Authentication

Simple bearer token in Authorization header:
```
Authorization: Bearer <token>
```

## Deployment Architecture

### Systemd Service
- Automatic restart on failure
- Resource limits
- Security hardening
- Log integration with journald

### File System Layout
```
/usr/local/bin/gomail         # Binary
/etc/gomail.yaml              # Configuration
/opt/mailserver/data/         # Email storage
/var/log/gomail/              # Logs (if file logging)
```

## Future Architecture Considerations

### Planned Enhancements
- Kubernetes operators
- Multi-region support
- S3 storage backend
- Event streaming (Kafka)

### Maintaining Simplicity
Despite future enhancements, the core principle remains:
- Single binary by default
- Optional complexity through plugins/modules
- API-first for all features