# GoMail Installation Guide

## Requirements

### System Requirements
- **OS**: Linux (CentOS 9, Ubuntu 20.04+, Debian 11+)
- **Architecture**: amd64 or arm64
- **Memory**: 512MB minimum, 1GB recommended
- **Disk**: 1GB for application, additional for email storage
- **Ports**: 25 (SMTP), 3000 (API), 9090 (metrics)
- **Privileges**: Root access for installation

### Prerequisites
- systemd-based Linux distribution
- Internet connectivity for downloading
- Domain name with DNS control
- (Optional) DigitalOcean API token for DNS automation

## Installation Methods

### Method 1: One-Line Installation (Recommended)

The quickest way to install GoMail:

```bash
# Basic installation - will prompt for configuration
curl -sSL https://github.com/grumpyguvner/gomail/releases/latest/download/quickinstall.sh | sudo bash

# With domain specified
curl -sSL https://github.com/grumpyguvner/gomail/releases/latest/download/quickinstall.sh | sudo bash -s example.com

# With domain and DigitalOcean token for automatic DNS
curl -sSL https://github.com/grumpyguvner/gomail/releases/latest/download/quickinstall.sh | sudo bash -s example.com --token YOUR_DO_TOKEN
```

#### What the Installer Does
1. Downloads the correct binary for your architecture
2. Installs to `/usr/local/bin/gomail`
3. Creates configuration file at `/etc/gomail.yaml`
4. Installs and configures Postfix
5. Sets up systemd service
6. Configures firewall rules (if applicable)
7. Starts the GoMail service
8. Optionally configures DigitalOcean DNS

### Method 2: Interactive Installation

For more control over the installation process:

```bash
# Download the latest release
wget https://github.com/grumpyguvner/gomail/releases/latest/download/gomail-linux-amd64
chmod +x gomail-linux-amd64
sudo mv gomail-linux-amd64 /usr/local/bin/gomail

# Run interactive setup
sudo gomail quickstart
```

The quickstart wizard will:
- Detect existing installations
- Prompt for configuration values
- Offer DigitalOcean DNS setup
- Configure all components
- Start the service

### Method 3: Manual Installation

For complete control over the installation:

```bash
# 1. Download and install binary
wget https://github.com/grumpyguvner/gomail/releases/latest/download/gomail-linux-amd64
chmod +x gomail-linux-amd64
sudo mv gomail-linux-amd64 /usr/local/bin/gomail

# 2. Create configuration
sudo tee /etc/gomail.yaml << EOF
port: 3000
bearer_token: $(openssl rand -base64 32)
primary_domain: example.com
mail_hostname: mail.example.com
api_endpoint: http://localhost:3000/mail/inbound
data_dir: /opt/mailserver/data
EOF

# 3. Install system components
sudo gomail install

# 4. Configure domain
sudo gomail domain add example.com

# 5. Start service
sudo systemctl start gomail
sudo systemctl enable gomail
```

## Post-Installation Setup

### 1. Verify Installation

```bash
# Check service status
systemctl status gomail

# Test API health
curl http://localhost:3000/health

# Check Postfix integration
postqueue -p

# View logs
journalctl -u gomail -f
```

### 2. Configure DNS Records

Add these DNS records for your domain:

```
Type  Name    Value                   Priority
MX    @       mail.example.com        10
A     mail    YOUR_SERVER_IP          -
TXT   @       "v=spf1 ip4:YOUR_IP ~all"  -
```

For DKIM (if enabled):
```
TXT   default._domainkey   "v=DKIM1; k=rsa; p=YOUR_PUBLIC_KEY"
```

### 3. Configure Firewall

```bash
# For firewalld (CentOS/RHEL)
sudo firewall-cmd --permanent --add-service=smtp
sudo firewall-cmd --permanent --add-port=3000/tcp
sudo firewall-cmd --reload

# For ufw (Ubuntu/Debian)
sudo ufw allow 25/tcp
sudo ufw allow 3000/tcp
sudo ufw reload
```

### 4. Configure Your Application

Update your application to handle webhooks:

```javascript
// Example webhook handler
app.post('/email-webhook', (req, res) => {
  const email = req.body;
  console.log('Received email:', {
    from: email.sender,
    to: email.recipient,
    subject: email.subject
  });
  res.status(200).send('OK');
});
```

Configure GoMail to send to your webhook:

```bash
gomail config set api_endpoint https://your-app.com/email-webhook
```

## Configuration

### Using Environment Variables

```bash
export MAIL_BEARER_TOKEN=your-secure-token
export MAIL_PORT=3000
export MAIL_PRIMARY_DOMAIN=example.com
```

### Using Configuration File

Edit `/etc/gomail.yaml`:

```yaml
# API Configuration
port: 3000
bearer_token: your-secure-token-here

# Domain Configuration
primary_domain: example.com
mail_hostname: mail.example.com

# Webhook Configuration  
api_endpoint: https://your-app.com/webhook

# Storage
data_dir: /opt/mailserver/data

# Security
rate_limit_per_minute: 60
rate_limit_burst: 10

# TLS Configuration
tls_cert_file: /etc/gomail/certs/cert.pem
tls_key_file: /etc/gomail/certs/key.pem

# Authentication
spf_enabled: true
dkim_enabled: true
dmarc_enabled: true
dmarc_enforcement: relaxed
```

## Upgrading

### Automatic Upgrade

```bash
# Stop service
sudo systemctl stop gomail

# Download and run installer
curl -sSL https://github.com/grumpyguvner/gomail/releases/latest/download/quickinstall.sh | sudo bash

# Service will be restarted automatically
```

### Manual Upgrade

```bash
# Stop service
sudo systemctl stop gomail

# Backup configuration
sudo cp /etc/gomail.yaml /etc/gomail.yaml.backup

# Download new version
wget https://github.com/grumpyguvner/gomail/releases/latest/download/gomail-linux-amd64
chmod +x gomail-linux-amd64
sudo mv gomail-linux-amd64 /usr/local/bin/gomail

# Start service
sudo systemctl start gomail
```

## Uninstallation

To completely remove GoMail:

```bash
# Stop and disable service
sudo systemctl stop gomail
sudo systemctl disable gomail

# Remove service file
sudo rm /etc/systemd/system/gomail.service

# Remove binary
sudo rm /usr/local/bin/gomail

# Remove configuration
sudo rm /etc/gomail.yaml

# Remove data (careful!)
sudo rm -rf /opt/mailserver

# Note: Postfix configuration is preserved
```

## Troubleshooting

### Service Won't Start

```bash
# Check for errors
journalctl -u gomail -n 50

# Validate configuration
gomail config validate

# Check port availability
sudo ss -tlnp | grep :3000
sudo ss -tlnp | grep :25
```

### Emails Not Receiving

```bash
# Check Postfix queue
postqueue -p

# Test SMTP connection
telnet localhost 25

# Check DNS records
dig MX example.com
dig A mail.example.com
```

### API Not Responding

```bash
# Check bearer token
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:3000/health

# Check rate limiting
# You may be rate limited if making too many requests

# Check logs for errors
journalctl -u gomail -f
```

### Common Issues

| Issue | Solution |
|-------|----------|
| Port 25 blocked by ISP | Contact hosting provider for unblock |
| Permission denied | Ensure running with sudo for installation |
| Service fails to start | Check config with `gomail config validate` |
| DNS not resolving | Wait 24-48 hours for propagation |
| Emails queued | Verify API endpoint is correct |

## Getting Help

- GitHub Issues: https://github.com/grumpyguvner/gomail/issues
- Documentation: Check other files in `/docs/`
- Logs: `journalctl -u gomail -f`
- Config validation: `gomail config validate`