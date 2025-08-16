#!/bin/bash
set -e

# GoMail Quick Install Script
# This script automatically installs and configures GoMail with sensible defaults

echo "╔══════════════════════════════════════╗"
echo "║     GoMail Quick Install Script      ║"
echo "╚══════════════════════════════════════╝"
echo

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

# Step 1: Download and install binary
echo "📦 Installing GoMail..."
wget -q -O /tmp/gomail "$RELEASE_URL" || { echo "Failed to download GoMail"; exit 1; }
chmod +x /tmp/gomail
mv /tmp/gomail /usr/local/bin/gomail
echo "✅ GoMail binary installed"

# Step 2: Generate default configuration with secure token
echo "🔧 Generating configuration..."
BEARER_TOKEN=$(openssl rand -base64 32 | tr -d '\n')
HOSTNAME=$(hostname -f)
PRIMARY_DOMAIN=${1:-$HOSTNAME}

cat > /etc/gomail.yaml << EOF
port: 3000
mode: simple
data_dir: /opt/gomail/data
bearer_token: ${BEARER_TOKEN}
mail_hostname: mail.${PRIMARY_DOMAIN}
primary_domain: ${PRIMARY_DOMAIN}
api_endpoint: http://localhost:3000/mail/inbound
EOF

echo "✅ Configuration generated"

# Step 3: Run installation
echo "🚀 Installing mail server components..."
gomail install --config /etc/gomail.yaml >/dev/null 2>&1 || { echo "Installation failed"; exit 1; }
echo "✅ Mail server components installed"

# Step 4: Add primary domain
echo "🌐 Adding domain ${PRIMARY_DOMAIN}..."
gomail domain add ${PRIMARY_DOMAIN} --config /etc/gomail.yaml >/dev/null 2>&1
echo "✅ Domain configured"

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
echo "   • Config file: /etc/gomail.yaml"
echo "   • Bearer token: ${BEARER_TOKEN}"
echo "   • Primary domain: ${PRIMARY_DOMAIN}"
echo "   • API endpoint: http://localhost:3000/mail/inbound"
echo
echo "📝 Next steps:"
echo "   1. Configure DNS records:"
echo "      gomail dns show ${PRIMARY_DOMAIN}"
echo "   2. Test email delivery:"
echo "      gomail test"
echo "   3. View logs:"
echo "      journalctl -u gomail -f"
echo
echo "🔐 IMPORTANT: Save your bearer token securely!"
echo "   It's required for API authentication."
echo