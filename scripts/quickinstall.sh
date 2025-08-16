#!/bin/bash
set -e

# GoMail Quick Install Script
# This script automatically installs and configures GoMail with sensible defaults

echo "╔══════════════════════════════════════╗"
echo "║     GoMail Quick Install Script      ║"
echo "╚══════════════════════════════════════╝"
echo

# Parse command line arguments
PRIMARY_DOMAIN=""
DO_TOKEN=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --token|-t)
      DO_TOKEN="$2"
      shift 2
      ;;
    *)
      PRIMARY_DOMAIN="$1"
      shift
      ;;
  esac
done

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

BINARY="gomail-${OS}-${ARCH}"
RELEASE_URL="https://github.com/grumpyguvner/gomail/releases/latest/download/${BINARY}"

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use sudo)" 
   exit 1
fi

# Check if this is a fresh install or reinstall
CONFIG_FILE="/etc/gomail.yaml"
IS_FRESH_INSTALL=true
if [ -f "$CONFIG_FILE" ]; then
  IS_FRESH_INSTALL=false
  echo "📋 Existing configuration detected at $CONFIG_FILE"
  # Read existing values
  if [ -z "$PRIMARY_DOMAIN" ]; then
    PRIMARY_DOMAIN=$(grep "primary_domain:" "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')
  fi
  if [ -z "$DO_TOKEN" ]; then
    DO_TOKEN=$(grep "do_api_token:" "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')
  fi
  BEARER_TOKEN=$(grep "bearer_token:" "$CONFIG_FILE" | awk '{print $2}' | tr -d '"')
else
  echo "🆕 Fresh installation detected"
fi

# For fresh installs, prompt for missing values
if [ "$IS_FRESH_INSTALL" = true ]; then
  # Prompt for domain if not provided
  if [ -z "$PRIMARY_DOMAIN" ]; then
    read -p "Enter your primary domain (e.g., example.com): " PRIMARY_DOMAIN
    if [ -z "$PRIMARY_DOMAIN" ]; then
      HOSTNAME=$(hostname -f)
      PRIMARY_DOMAIN=$HOSTNAME
      echo "Using hostname as domain: $PRIMARY_DOMAIN"
    fi
  fi
  
  # Prompt for DigitalOcean token if not provided
  if [ -z "$DO_TOKEN" ]; then
    echo
    echo "📌 DigitalOcean API token enables automatic DNS configuration"
    read -p "Enter your DigitalOcean API token (or press Enter to skip): " DO_TOKEN
  fi
  
  # Generate new bearer token for fresh install
  BEARER_TOKEN=$(openssl rand -base64 32 | tr -d '\n')
fi

# Step 1: Download and install binary
echo
echo "📦 Installing GoMail..."
wget -q -O /tmp/gomail "$RELEASE_URL" || { echo "Failed to download GoMail"; exit 1; }
chmod +x /tmp/gomail
mv /tmp/gomail /usr/local/bin/gomail
echo "✅ GoMail binary installed"

# Step 2: Generate or update configuration
echo "🔧 Configuring GoMail..."

# Write configuration
if [ -n "$DO_TOKEN" ]; then
  cat > $CONFIG_FILE << EOF
port: 3000
mode: simple
data_dir: /opt/gomail/data
bearer_token: ${BEARER_TOKEN}
mail_hostname: mail.${PRIMARY_DOMAIN}
primary_domain: ${PRIMARY_DOMAIN}
api_endpoint: http://localhost:3000/mail/inbound
do_api_token: ${DO_TOKEN}
EOF
else
  cat > $CONFIG_FILE << EOF
port: 3000
mode: simple
data_dir: /opt/gomail/data
bearer_token: ${BEARER_TOKEN}
mail_hostname: mail.${PRIMARY_DOMAIN}
primary_domain: ${PRIMARY_DOMAIN}
api_endpoint: http://localhost:3000/mail/inbound
EOF
fi

if [ "$IS_FRESH_INSTALL" = true ]; then
  echo "✅ Configuration created"
else
  echo "✅ Configuration updated"
fi

# Step 3: Run installation
echo "🚀 Installing mail server components..."
gomail install --config $CONFIG_FILE >/dev/null 2>&1 || { echo "Installation failed"; exit 1; }
echo "✅ Mail server components installed"

# Step 4: Add primary domain
echo "🌐 Configuring domain ${PRIMARY_DOMAIN}..."
gomail domain add ${PRIMARY_DOMAIN} --config $CONFIG_FILE >/dev/null 2>&1
echo "✅ Domain configured"

# Step 5: Configure DNS if DO token provided
if [ -n "$DO_TOKEN" ]; then
  echo "🔧 Configuring DigitalOcean DNS records..."
  gomail dns create ${PRIMARY_DOMAIN} --config $CONFIG_FILE >/dev/null 2>&1 && echo "✅ DNS records created" || echo "⚠️  DNS configuration failed - configure manually"
fi

# Step 5: Create systemd service
echo "⚙️  Setting up systemd service..."
cat > /etc/systemd/system/gomail.service << 'EOF'
[Unit]
Description=GoMail Server
After=network.target postfix.service

[Service]
Type=simple
User=gomail
Group=gomail
Environment="MAIL_CONFIG=/etc/gomail.yaml"
ExecStart=/usr/local/bin/gomail server --config /etc/gomail.yaml
Restart=on-failure
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/gomail/data

[Install]
WantedBy=multi-user.target
EOF

# Create service user if it doesn't exist
if ! id -u gomail >/dev/null 2>&1; then
  useradd -r -s /sbin/nologin -d /opt/gomail gomail
fi

# Create data directory
mkdir -p /opt/gomail/data
chown -R gomail:gomail /opt/gomail

systemctl daemon-reload
systemctl enable gomail >/dev/null 2>&1
systemctl start gomail
echo "✅ Service started"

# Step 6: Test the installation
echo "🧪 Testing installation..."
sleep 2
if curl -s http://localhost:3000/health | grep -q "healthy"; then
  echo "✅ Health check passed"
else
  echo "⚠️  Health check failed - check logs with: journalctl -u gomail -n 50"
fi

# Final output
echo
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                  🎉 Installation Complete! 🎉                 ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo
echo "📋 Configuration:"
echo "   • Config file: $CONFIG_FILE"
if [ "$IS_FRESH_INSTALL" = true ]; then
  echo "   • Bearer token: ${BEARER_TOKEN}"
fi
echo "   • Primary domain: ${PRIMARY_DOMAIN}"
echo "   • API endpoint: http://localhost:3000/mail/inbound"
if [ -n "$DO_TOKEN" ]; then
  echo "   • DigitalOcean: Configured"
fi
echo
echo "📝 Next steps:"
if [ -z "$DO_TOKEN" ]; then
  echo "   1. Configure DNS records:"
  echo "      gomail dns show ${PRIMARY_DOMAIN}"
else
  echo "   1. Verify DNS records:"
  echo "      gomail dns show ${PRIMARY_DOMAIN}"
fi
echo "   2. Test email delivery:"
echo "      gomail test"
echo "   3. View logs:"
echo "      journalctl -u gomail -f"
echo
if [ "$IS_FRESH_INSTALL" = true ]; then
  echo "🔐 IMPORTANT: Save your bearer token securely!"
  echo "   It's required for API authentication."
fi
echo