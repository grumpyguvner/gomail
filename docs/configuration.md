# GoMail Configuration Reference

## Configuration Overview

GoMail uses a three-layer configuration system with the following precedence (highest to lowest):

1. **Command-line flags** - Override all other settings
2. **Environment variables** - Override file settings
3. **YAML configuration file** - Base configuration

## Configuration File

Default location: `/etc/gomail.yaml`

### Complete Configuration Example

```yaml
# API Server Configuration
port: 3000                        # API server port
mode: production                   # Mode: development or production

# Authentication
bearer_token: "your-secure-token-here"  # API authentication token

# Domain Configuration
primary_domain: example.com       # Primary email domain
mail_hostname: mail.example.com   # Mail server hostname
additional_domains:                # Additional domains to handle
  - example.org
  - example.net

# Webhook Configuration
api_endpoint: https://your-app.com/webhook  # Webhook URL for emails
webhook_timeout: 30s               # Webhook request timeout
webhook_retries: 3                 # Number of retry attempts
webhook_retry_delay: 5s            # Delay between retries

# Storage Configuration
data_dir: /opt/mailserver/data     # Email storage directory
storage_mode: json                 # Storage mode: json or database
connection_pool_size: 10           # Storage connection pool size
max_storage_size: 10GB            # Maximum storage size

# TLS/SSL Configuration
tls_enabled: true                  # Enable TLS
tls_cert_file: /etc/gomail/certs/cert.pem  # TLS certificate
tls_key_file: /etc/gomail/certs/key.pem    # TLS private key
tls_min_version: "1.2"            # Minimum TLS version
tls_cipher_suites:                # Allowed cipher suites
  - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
  - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
starttls_enabled: true            # Enable STARTTLS on port 25

# Email Authentication
spf_enabled: true                 # Enable SPF checking
spf_enforcement: strict           # SPF enforcement: none, soft, strict

dkim_enabled: true                # Enable DKIM verification
dkim_selector: default            # DKIM selector
dkim_private_key_path: /etc/gomail/dkim/private.key  # DKIM signing key
dkim_sign_outbound: false         # Sign outgoing emails

dmarc_enabled: true               # Enable DMARC enforcement
dmarc_enforcement: relaxed        # DMARC: none, relaxed, strict
dmarc_reporting: true             # Enable DMARC aggregate reports

# Security Configuration
rate_limit_per_minute: 60         # Requests per minute per IP
rate_limit_burst: 10              # Burst allowance
max_connections_per_ip: 10        # Max concurrent connections per IP
max_total_connections: 1000       # Max total connections
ban_duration: 1h                  # IP ban duration
ban_threshold: 5                  # Violations before ban
max_message_size: 26214400        # Max email size (bytes)

# HTTP Timeouts
http_timeouts:
  read: 30s                       # Read timeout
  write: 30s                      # Write timeout
  handler: 60s                    # Handler timeout
  idle: 120s                      # Idle timeout
  shutdown: 30s                   # Graceful shutdown timeout

# Postfix Integration
postfix_queue_directory: /var/spool/postfix  # Postfix queue location
postfix_config_directory: /etc/postfix       # Postfix config location

# Logging Configuration
log_level: info                   # Log level: debug, info, warn, error
log_file: /var/log/gomail/gomail.log  # Log file path (empty for stdout)
log_format: json                  # Log format: json or text
log_rotation_size: 100MB          # Rotate after size
log_rotation_age: 7d              # Rotate after age
log_retention: 30d                # Keep logs for duration

# Metrics Configuration
metrics_enabled: true              # Enable Prometheus metrics
metrics_port: 9090                # Metrics server port
metrics_path: /metrics            # Metrics endpoint path

# Operational Settings
health_check_interval: 30s        # Health check frequency
cleanup_interval: 1h              # Old file cleanup frequency
cleanup_age: 30d                  # Delete files older than

# DigitalOcean Integration (optional)
digitalocean_token: ""            # DO API token for DNS management
digitalocean_domain_id: ""        # DO domain ID

# Advanced Settings
debug_mode: false                 # Enable debug mode
panic_recovery: true              # Enable panic recovery
request_id_header: X-Request-ID   # Request ID header name
cors_enabled: false               # Enable CORS
cors_origins:                     # Allowed CORS origins
  - https://app.example.com
```

## Environment Variables

All configuration options can be set via environment variables with the `MAIL_` prefix:

```bash
# Core settings
export MAIL_PORT=3000
export MAIL_BEARER_TOKEN="your-secure-token"
export MAIL_PRIMARY_DOMAIN="example.com"
export MAIL_API_ENDPOINT="https://your-app.com/webhook"

# Security
export MAIL_RATE_LIMIT_PER_MINUTE=60
export MAIL_MAX_MESSAGE_SIZE=26214400

# TLS
export MAIL_TLS_ENABLED=true
export MAIL_TLS_CERT_FILE="/etc/gomail/certs/cert.pem"
export MAIL_TLS_KEY_FILE="/etc/gomail/certs/key.pem"

# Authentication
export MAIL_SPF_ENABLED=true
export MAIL_DKIM_ENABLED=true
export MAIL_DMARC_ENABLED=true

# Logging
export MAIL_LOG_LEVEL=info
export MAIL_LOG_FILE="/var/log/gomail/gomail.log"

# Metrics
export MAIL_METRICS_ENABLED=true
export MAIL_METRICS_PORT=9090
```

### Environment Variable Naming

Convert YAML keys to environment variables:
1. Prefix with `MAIL_`
2. Convert to uppercase
3. Replace dots and hyphens with underscores

Examples:
- `bearer_token` → `MAIL_BEARER_TOKEN`
- `http_timeouts.read` → `MAIL_HTTP_TIMEOUTS_READ`
- `tls-cert-file` → `MAIL_TLS_CERT_FILE`

## Command-Line Flags

Override configuration for specific commands:

```bash
# Override port
gomail server --port 8080

# Override bearer token
gomail server --bearer-token "temporary-token"

# Override log level
gomail server --log-level debug

# Multiple overrides
gomail server \
  --port 8080 \
  --bearer-token "temp-token" \
  --log-level debug \
  --metrics-port 9091
```

## Configuration Commands

### View Current Configuration

```bash
# Show all configuration
gomail config show

# Show specific value
gomail config get bearer_token
gomail config get primary_domain
```

### Update Configuration

```bash
# Set a value
gomail config set bearer_token "new-token"
gomail config set primary_domain "example.com"
gomail config set rate_limit_per_minute 120

# Set nested values
gomail config set http_timeouts.read 45s
```

### Validate Configuration

```bash
# Validate configuration file
gomail config validate

# Test with specific file
gomail config validate --config /path/to/config.yaml
```

## Configuration Profiles

### Development Profile

```yaml
# development.yaml
mode: development
port: 3000
log_level: debug
debug_mode: true
rate_limit_per_minute: 1000  # Relaxed for testing
tls_enabled: false            # No TLS for local dev
```

Load with:
```bash
gomail server --config development.yaml
```

### Production Profile

```yaml
# production.yaml
mode: production
port: 3000
log_level: info
debug_mode: false
tls_enabled: true
rate_limit_per_minute: 60
panic_recovery: true
```

### Testing Profile

```yaml
# testing.yaml
mode: testing
port: 3001  # Different port for tests
data_dir: /tmp/gomail-test
log_level: error  # Minimal logging
rate_limit_per_minute: 10000  # No rate limiting
```

## Security Best Practices

### Bearer Token

1. **Generate strong tokens**:
   ```bash
   openssl rand -base64 32
   ```

2. **Store securely**:
   - Use environment variables in production
   - Never commit tokens to git
   - Rotate tokens regularly

3. **Token rotation**:
   ```bash
   # Generate new token
   NEW_TOKEN=$(openssl rand -base64 32)
   
   # Update configuration
   gomail config set bearer_token "$NEW_TOKEN"
   
   # Restart service
   systemctl restart gomail
   ```

### TLS Configuration

1. **Use strong ciphers**:
   ```yaml
   tls_cipher_suites:
     - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
     - TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
   ```

2. **Enforce minimum TLS version**:
   ```yaml
   tls_min_version: "1.2"  # or "1.3" for maximum security
   ```

3. **Auto-renewal with Let's Encrypt**:
   ```bash
   gomail ssl generate --domain example.com --email admin@example.com
   ```

## Performance Tuning

### Connection Pool

```yaml
# For high-volume sites
connection_pool_size: 50
max_total_connections: 5000

# For low-volume sites
connection_pool_size: 5
max_total_connections: 100
```

### Rate Limiting

```yaml
# Strict (public API)
rate_limit_per_minute: 60
rate_limit_burst: 10

# Moderate (trusted partners)
rate_limit_per_minute: 300
rate_limit_burst: 50

# Relaxed (internal only)
rate_limit_per_minute: 1000
rate_limit_burst: 100
```

### Timeouts

```yaml
# Fast network (local/same datacenter)
http_timeouts:
  read: 10s
  write: 10s
  handler: 30s

# Slow network (cross-region)
http_timeouts:
  read: 60s
  write: 60s
  handler: 120s
```

## Monitoring Configuration

### Metrics Collection

```yaml
# Full metrics
metrics_enabled: true
metrics_port: 9090
metrics_path: /metrics

# Minimal metrics (production)
metrics_enabled: true
metrics_port: 9090
# Use firewall to restrict access
```

### Logging

```yaml
# Development
log_level: debug
log_format: text
log_file: ""  # stdout

# Production
log_level: info
log_format: json
log_file: /var/log/gomail/gomail.log
log_rotation_size: 100MB
log_retention: 30d
```

## Migration from Legacy Config

If migrating from the Node.js version:

```javascript
// Old Node.js config
{
  "apiPort": 3000,
  "apiToken": "token",
  "domain": "example.com"
}
```

Converts to:

```yaml
# New GoMail config
port: 3000
bearer_token: "token"
primary_domain: "example.com"
```

## Troubleshooting Configuration

### Common Issues

| Issue | Solution |
|-------|----------|
| Config not loading | Check file permissions and YAML syntax |
| Environment vars ignored | Ensure MAIL_ prefix is used |
| Changes not taking effect | Restart service after changes |
| Validation errors | Run `gomail config validate` |

### Debug Configuration Loading

```bash
# Enable debug logging
MAIL_LOG_LEVEL=debug gomail server

# This will show:
# - Config file location
# - Environment variables loaded
# - Final configuration values
```

## Configuration Schema

The configuration is validated against a JSON schema. View the schema:

```bash
gomail config schema
```

This shows all valid configuration keys and their types.

## Support

- Configuration issues: Run `gomail config validate`
- Documentation: See other files in `/docs/`
- GitHub Issues: https://github.com/grumpyguvner/gomail/issues