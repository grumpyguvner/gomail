# GoMail Security Documentation

## Security Overview

GoMail implements defense-in-depth security with multiple layers of protection:

1. **Network Security**: TLS encryption and STARTTLS
2. **Authentication**: Bearer tokens and email authentication protocols
3. **Rate Limiting**: Protection against abuse and DoS
4. **Input Validation**: Comprehensive validation and sanitization
5. **System Hardening**: Systemd security and process isolation

## TLS/SSL Configuration

### TLS Support

GoMail enforces modern TLS standards:

- **Minimum Version**: TLS 1.2 (TLS 1.3 recommended)
- **Strong Ciphers Only**: ECDHE, AES-GCM, ChaCha20-Poly1305
- **Forward Secrecy**: Ephemeral key exchange
- **STARTTLS**: Opportunistic encryption on port 25

### Configuration

```yaml
# TLS Settings
tls_enabled: true
tls_cert_file: /etc/gomail/certs/cert.pem
tls_key_file: /etc/gomail/certs/key.pem
tls_min_version: "1.2"
starttls_enabled: true

# Strong cipher suites only
tls_cipher_suites:
  - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
  - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
  - TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
  - TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
  - TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
  - TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
```

### Let's Encrypt Integration

GoMail supports automatic SSL certificate acquisition and renewal using Let's Encrypt.

#### Automatic Certificate Acquisition

Certificates are automatically obtained during installation when a DigitalOcean API token is configured:

```bash
# Installation automatically attempts Let's Encrypt certificate
# if DO_API_TOKEN is configured
gomail install
```

#### DNS-01 Challenge Support

GoMail uses DNS-01 challenges via DigitalOcean's DNS API for certificate validation:
- **No port 80 required**: Works behind firewalls and NAT
- **Wildcard support**: Can obtain wildcard certificates
- **Automatic renewal**: Certificates renew automatically before expiration
- **Secure validation**: Uses DNS TXT records for domain ownership proof

#### Manual Certificate Management

```bash
# Obtain certificate manually
gomail ssl setup --email admin@example.com --agree-tos

# Check certificate status
gomail ssl status

# Renew certificate (if needed)
gomail ssl renew

# Force renewal
gomail ssl renew --force
```

#### Automatic Renewal

Add to crontab for automatic renewal:
```bash
# Check daily at 2 AM for certificate renewal
0 2 * * * /usr/local/bin/gomail ssl renew && systemctl reload postfix
```

#### Fallback to Self-Signed

If Let's Encrypt certificate acquisition fails (e.g., no DO API token), the system automatically:
1. Generates a self-signed certificate
2. Configures services to use the self-signed certificate
3. Logs a warning about the fallback

#### Requirements for Let's Encrypt

- **Domain**: Valid domain name pointing to server
- **DigitalOcean API Token**: Required for DNS-01 challenge
- **Email**: Valid email for Let's Encrypt notifications

Configure in environment or config file:
```yaml
do_api_token: "your-digitalocean-api-token"
mail_hostname: "mail.example.com"
```

## Email Authentication

### SPF (Sender Policy Framework)

Validates sender IP against domain's authorized senders:

```yaml
spf_enabled: true
spf_enforcement: strict  # none, soft, strict
```

SPF enforcement levels:
- **none**: Check but don't enforce
- **soft**: Mark failures but accept
- **strict**: Reject SPF failures

### DKIM (DomainKeys Identified Mail)

Verifies email signatures:

```yaml
dkim_enabled: true
dkim_selector: default
dkim_private_key_path: /etc/gomail/dkim/private.key
```

Generate DKIM keys:
```bash
# Generate 2048-bit RSA key
openssl genrsa -out /etc/gomail/dkim/private.key 2048
openssl rsa -in /etc/gomail/dkim/private.key -pubout -out /etc/gomail/dkim/public.key

# Add public key to DNS
gomail dkim show-record
```

### DMARC (Domain-based Message Authentication)

Policy enforcement combining SPF and DKIM:

```yaml
dmarc_enabled: true
dmarc_enforcement: relaxed  # none, relaxed, strict
dmarc_reporting: true
```

Enforcement levels:
- **none**: Monitor only, no enforcement
- **relaxed**: Flexible alignment checking
- **strict**: Exact domain matching required

## API Security

### Bearer Token Authentication

All API endpoints (except health/metrics) require authentication:

```http
Authorization: Bearer your-secure-token-here
```

Token best practices:
1. **Generate strong tokens**: 32+ bytes of randomness
2. **Rotate regularly**: Monthly or after incidents
3. **Store securely**: Environment variables or secrets manager
4. **Never log tokens**: Ensure tokens aren't in logs

Generate secure token:
```bash
openssl rand -base64 32
```

### Rate Limiting

Protection against abuse:

```yaml
rate_limit_per_minute: 60    # Requests per minute
rate_limit_burst: 10          # Burst allowance
```

Rate limiting features:
- Per-IP tracking
- Token bucket algorithm
- Configurable limits
- Headers indicate limit status

## Connection Security

### Connection Limiting

Prevent resource exhaustion:

```yaml
max_connections_per_ip: 10     # Per IP limit
max_total_connections: 1000    # Global limit
ban_threshold: 5                # Violations before ban
ban_duration: 1h                # Ban duration
```

### IP Ban Management

Automatic and manual IP management:

```bash
# View banned IPs
gomail security banned-ips

# Manually ban IP
gomail security ban 192.0.2.1

# Unban IP
gomail security unban 192.0.2.1
```

## Input Validation

### Email Validation

Comprehensive validation includes:
- Size limits (default 25MB)
- RFC822 compliance
- Header sanitization
- Attachment scanning
- Content filtering

```yaml
max_message_size: 26214400  # 25MB in bytes
validation:
  strict_headers: true
  reject_invalid_domains: true
  check_spf: true
  check_dkim: true
```

### API Input Validation

All API inputs are validated:
- JSON schema validation
- Type checking
- Range validation
- SQL injection prevention
- XSS protection

## System Security

### Process Isolation

GoMail runs with minimal privileges:

```ini
# systemd security settings
[Service]
User=gomail
Group=gomail
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/mailserver/data
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE
```

### File Permissions

Secure file permissions:

```bash
# Binary
-rwxr-xr-x root root /usr/local/bin/gomail

# Configuration (contains token)
-rw------- gomail gomail /etc/gomail.yaml

# TLS certificates
-rw-r--r-- gomail gomail /etc/gomail/certs/cert.pem
-rw------- gomail gomail /etc/gomail/certs/key.pem

# Data directory
drwx------ gomail gomail /opt/mailserver/data
```

### AppArmor Profile (Future)

```
#include <tunables/global>

/usr/local/bin/gomail {
  #include <abstractions/base>
  #include <abstractions/nameservice>
  
  # Read access
  /etc/gomail.yaml r,
  /etc/gomail/** r,
  /proc/sys/kernel/random/uuid r,
  
  # Write access
  /opt/mailserver/data/** rw,
  /var/log/gomail/** w,
  
  # Network access
  network inet stream,
  network inet6 stream,
  
  # Capabilities
  capability net_bind_service,
}
```

## Security Headers

HTTP security headers are automatically added:

```http
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'none'
```

## Logging and Auditing

### Security Event Logging

Security events are logged with context:

```json
{
  "level": "warn",
  "time": "2024-01-15T10:30:00Z",
  "event": "authentication_failed",
  "ip": "192.0.2.1",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "details": {
    "reason": "invalid_token",
    "endpoint": "/mail/inbound"
  }
}
```

### Audit Trail

Important events tracked:
- Authentication attempts
- Rate limit violations
- IP bans
- Configuration changes
- SPF/DKIM/DMARC failures
- TLS handshake failures

## Incident Response

### Detecting Attacks

Monitor these metrics:
```promql
# High authentication failure rate
rate(gomail_auth_failures_total[5m]) > 10

# Unusual traffic patterns
rate(gomail_emails_received_total[1m]) > 100

# SPF/DKIM failures spike
rate(gomail_spf_fail_total[5m]) > 20
```

### Response Procedures

1. **DDoS Attack**
   ```bash
   # Increase rate limits temporarily
   gomail config set rate_limit_per_minute 10
   
   # Ban offending IPs
   gomail security ban 192.0.2.1
   
   # Enable stricter validation
   gomail config set spf_enforcement strict
   ```

2. **Authentication Breach**
   ```bash
   # Rotate bearer token immediately
   NEW_TOKEN=$(openssl rand -base64 32)
   gomail config set bearer_token "$NEW_TOKEN"
   systemctl restart gomail
   
   # Review logs for unauthorized access
   journalctl -u gomail | grep "authentication"
   ```

3. **Spam Attack**
   ```bash
   # Enable strict DMARC
   gomail config set dmarc_enforcement strict
   
   # Reduce message size limit
   gomail config set max_message_size 5242880
   
   # Increase rate limiting
   gomail config set rate_limit_per_minute 30
   ```

## Security Checklist

### Pre-Production

- [ ] Generate strong bearer token (32+ bytes)
- [ ] Configure TLS with valid certificates
- [ ] Enable SPF/DKIM/DMARC checking
- [ ] Set appropriate rate limits
- [ ] Configure connection limits
- [ ] Enable structured logging
- [ ] Set up monitoring alerts
- [ ] Test backup/recovery procedures
- [ ] Document incident response plan

### Operational

- [ ] Rotate bearer tokens monthly
- [ ] Monitor authentication failures
- [ ] Review banned IP list weekly
- [ ] Update TLS certificates before expiry
- [ ] Audit log files regularly
- [ ] Test incident response quarterly
- [ ] Keep GoMail updated
- [ ] Monitor security advisories

## Vulnerability Reporting

Report security vulnerabilities to:
- Email: security@example.com (use PGP if available)
- GitHub Security Advisories (private)

Please include:
1. Description of vulnerability
2. Steps to reproduce
3. Potential impact
4. Suggested fix (if any)

## Compliance

GoMail can be configured to meet various compliance requirements:

### GDPR
- Data minimization through configurable retention
- Audit logging for access tracking
- Encryption in transit (TLS)

### HIPAA
- Encryption requirements met with TLS 1.2+
- Audit trails maintained
- Access controls via bearer tokens

### PCI DSS
- Strong cryptography (TLS 1.2+)
- Access control (authentication required)
- Monitoring and logging capabilities

## Security Tools Integration

### Security Scanning

```bash
# Static analysis
gosec ./...

# Dependency scanning
nancy go.sum

# Container scanning (if using Docker)
trivy image gomail:latest

# Secret scanning
gitleaks detect
```

### Monitoring Integration

Integrate with security tools:
- **SIEM**: Forward logs to Splunk/ELK
- **IDS/IPS**: Monitor network traffic
- **WAF**: Protect API endpoints
- **DLP**: Scan email content

## Updates and Patches

Stay secure with updates:

```bash
# Check for updates
gomail version --check-update

# Update to latest
curl -sSL https://github.com/grumpyguvner/gomail/releases/latest/download/quickinstall.sh | sudo bash

# Subscribe to security advisories
# Watch the GitHub repository for security updates
```

## Additional Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [GitHub Security Advisories](https://github.com/grumpyguvner/gomail/security)