#!/bin/bash
# Don't exit on error - we want to keep the droplet for debugging
set +e

# GoMail Droplet Test Script
# This script creates a test droplet, tests installation, and keeps it for testing

# Configuration
DO_TOKEN="${DO_TOKEN:-}"  # Must be set as environment variable
TEST_DOMAIN="${TEST_DOMAIN:-}"  # Must be set as environment variable
REGION="${REGION:-lon1}"
SIZE="${SIZE:-s-1vcpu-1gb-intel}"
IMAGE="${IMAGE:-centos-stream-9-x64}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_rsa}"
RELEASE_VERSION="${RELEASE_VERSION:-latest}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Cleanup function
cleanup() {
    if [ ! -z "$DROPLET_ID" ]; then
        log_info "Cleaning up droplet $DROPLET_ID..."
        curl -X DELETE "https://api.digitalocean.com/v2/droplets/$DROPLET_ID" \
            -H "Authorization: Bearer $DO_TOKEN" \
            -H "Content-Type: application/json" 2>/dev/null
        log_info "Droplet deleted"
    fi
    
    # Clean up DNS records (except NS and SOA which are required)
    if [ ! -z "$TEST_DOMAIN" ]; then
        log_info "Cleaning up DNS records for $TEST_DOMAIN..."
        # Get all DNS records except NS and SOA
        RECORDS=$(curl -s -X GET "https://api.digitalocean.com/v2/domains/$TEST_DOMAIN/records" \
            -H "Authorization: Bearer $DO_TOKEN" | jq -r '.domain_records[] | select(.type != "NS" and .type != "SOA") | "\(.id)"')
        
        for RECORD_ID in $RECORDS; do
            curl -s -X DELETE "https://api.digitalocean.com/v2/domains/$TEST_DOMAIN/records/$RECORD_ID" \
                -H "Authorization: Bearer $DO_TOKEN" 2>/dev/null
        done
        log_info "DNS records cleaned (NS and SOA preserved)"
    fi
}

# Set trap for cleanup on exit (only if not keeping droplet)
KEEP_DROPLET=${KEEP_DROPLET:-true}
if [ "$KEEP_DROPLET" != "true" ]; then
    trap cleanup EXIT
fi

# Main test process
main() {
    # Check for required variables
    if [ -z "$DO_TOKEN" ]; then
        log_error "DO_TOKEN environment variable is required"
        log_error "Usage: DO_TOKEN=your_token TEST_DOMAIN=yourdomain.com ./test-droplet.sh"
        exit 1
    fi
    
    if [ -z "$TEST_DOMAIN" ]; then
        log_error "TEST_DOMAIN environment variable is required"
        log_error "Usage: DO_TOKEN=your_token TEST_DOMAIN=yourdomain.com ./test-droplet.sh"
        exit 1
    fi
    
    log_info "Starting GoMail installation test"
    log_info "Domain: $TEST_DOMAIN"
    log_info "Region: $REGION"
    log_info "Size: $SIZE"
    log_info "Image: $IMAGE"
    
    # Step 1: Get SSH keys from account
    log_info "Getting SSH keys from DigitalOcean..."
    SSH_KEY_IDS=$(curl -s -X GET "https://api.digitalocean.com/v2/account/keys" \
        -H "Authorization: Bearer $DO_TOKEN" | jq -r '[.ssh_keys[].id] | @json')
    
    if [ "$SSH_KEY_IDS" == "[]" ] || [ -z "$SSH_KEY_IDS" ]; then
        log_warn "No SSH keys found in DigitalOcean account, droplet will use password auth"
        SSH_KEY_IDS="[]"
    else
        log_info "Found SSH keys: $SSH_KEY_IDS"
    fi
    
    # Step 2: Create droplet
    log_info "Creating test droplet..."
    DROPLET_NAME="gomail-test-$(date +%s)"
    
    RESPONSE=$(curl -s -X POST "https://api.digitalocean.com/v2/droplets" \
        -H "Authorization: Bearer $DO_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"name\": \"$DROPLET_NAME\",
            \"region\": \"$REGION\",
            \"size\": \"$SIZE\",
            \"image\": \"$IMAGE\",
            \"ssh_keys\": $SSH_KEY_IDS,
            \"backups\": false,
            \"ipv6\": false,
            \"monitoring\": false,
            \"tags\": [\"gomail-test\"]
        }")
    
    DROPLET_ID=$(echo "$RESPONSE" | jq -r '.droplet.id')
    
    if [ "$DROPLET_ID" == "null" ] || [ -z "$DROPLET_ID" ]; then
        log_error "Failed to create droplet"
        echo "$RESPONSE" | jq .
        exit 1
    fi
    
    log_info "Droplet created with ID: $DROPLET_ID"
    
    # Step 3: Wait for droplet to be ready
    log_info "Waiting for droplet to be ready..."
    ATTEMPTS=0
    MAX_ATTEMPTS=60
    
    while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
        STATUS=$(curl -s -X GET "https://api.digitalocean.com/v2/droplets/$DROPLET_ID" \
            -H "Authorization: Bearer $DO_TOKEN" | jq -r '.droplet.status')
        
        if [ "$STATUS" == "active" ]; then
            # Get the public IP (v4[1] is public, v4[0] is private)
            DROPLET_IP=$(curl -s -X GET "https://api.digitalocean.com/v2/droplets/$DROPLET_ID" \
                -H "Authorization: Bearer $DO_TOKEN" | jq -r '.droplet.networks.v4[] | select(.type=="public") | .ip_address')
            
            if [ ! -z "$DROPLET_IP" ] && [ "$DROPLET_IP" != "null" ]; then
                log_info "Droplet is active with IP: $DROPLET_IP"
                break
            fi
        fi
        
        sleep 5
        ATTEMPTS=$((ATTEMPTS + 1))
        echo -n "."
    done
    echo
    
    if [ $ATTEMPTS -eq $MAX_ATTEMPTS ]; then
        log_error "Timeout waiting for droplet to be ready"
        exit 1
    fi
    
    # Step 4: Wait for SSH to be ready
    log_info "Waiting for SSH to be ready..."
    ATTEMPTS=0
    MAX_ATTEMPTS=30
    
    while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
        if ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o UserKnownHostsFile=/dev/null \
            -i "$SSH_KEY" "root@$DROPLET_IP" "echo 'SSH ready'" 2>/dev/null; then
            log_info "SSH is ready"
            break
        fi
        sleep 5
        ATTEMPTS=$((ATTEMPTS + 1))
        echo -n "."
    done
    echo
    
    if [ $ATTEMPTS -eq $MAX_ATTEMPTS ]; then
        log_error "Timeout waiting for SSH"
        exit 1
    fi
    
    # Step 5: Install wget (needed for CentOS)
    log_info "Installing wget on droplet..."
    ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
        -i "$SSH_KEY" "root@$DROPLET_IP" "dnf install -y wget" || true
    
    # Step 6: Run quickinstall script
    log_info "Running GoMail quickinstall script..."
    
    # Create a wrapper script to capture output and exit code
    ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
        -i "$SSH_KEY" "root@$DROPLET_IP" << EOF
#!/bin/bash
set -e

# Download and run quickinstall
if [ "$RELEASE_VERSION" == "latest" ]; then
    SCRIPT_URL="https://github.com/grumpyguvner/gomail/releases/latest/download/quickinstall.sh"
else
    SCRIPT_URL="https://github.com/grumpyguvner/gomail/releases/download/$RELEASE_VERSION/quickinstall.sh"
fi

echo "Downloading quickinstall from: \$SCRIPT_URL"
curl -sSL "\$SCRIPT_URL" -o /tmp/quickinstall.sh
chmod +x /tmp/quickinstall.sh

# Run the installation
echo "Running installation for domain: $TEST_DOMAIN"
/tmp/quickinstall.sh "$TEST_DOMAIN" --token "$DO_TOKEN" 2>&1 | tee /tmp/install.log

# Check if installation succeeded
if [ \${PIPESTATUS[0]} -ne 0 ]; then
    echo "Installation failed. Last 50 lines of log:"
    tail -50 /tmp/install.log
    exit 1
fi

echo "Installation completed successfully"
EOF
    
    if [ $? -ne 0 ]; then
        log_error "Installation failed"
        
        # Get more debug info
        log_info "Fetching debug information..."
        ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
            -i "$SSH_KEY" "root@$DROPLET_IP" << 'EOF' || true
echo "=== System Info ==="
cat /etc/os-release
echo
echo "=== GoMail Binary Check ==="
ls -la /usr/local/bin/gomail* || echo "Binaries not found"
echo
echo "=== Config File ==="
cat /etc/gomail.yaml 2>/dev/null || echo "Config not found"
echo
echo "=== Service Status ==="
systemctl status gomail --no-pager 2>/dev/null || echo "Service not found"
echo
echo "=== Install Log ==="
cat /tmp/install.log 2>/dev/null | tail -100
echo
echo "=== Manual Install Test ==="
/usr/local/bin/gomail --version 2>&1 || echo "Binary execution failed"
echo
echo "=== Running install command manually ==="
/usr/local/bin/gomail install --config /etc/gomail.yaml 2>&1 || echo "Install command failed"
EOF
        exit 1
    fi
    
    # Step 7: Test services
    log_info "Testing GoMail services..."
    
    ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
        -i "$SSH_KEY" "root@$DROPLET_IP" << 'EOF'
#!/bin/bash
set -e

echo "=== Service Status ==="
systemctl status gomail --no-pager
systemctl status gomail-webadmin --no-pager 2>/dev/null || echo "WebAdmin service not found"

echo
echo "=== API Health Check ==="
curl -s http://localhost:3000/health | jq . || echo "API health check failed"

echo
echo "=== Testing API with Bearer Token ==="
TOKEN=$(grep bearer_token /etc/gomail.yaml | awk '{print $2}' | tr -d '"')
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:3000/domains | jq . || echo "API test failed"

echo
echo "=== DNS Records Created ==="
echo "Checking DNS records for $TEST_DOMAIN..."
# The gomail tool should have created these
/usr/local/bin/gomail dns show $TEST_DOMAIN 2>/dev/null || echo "DNS show failed"

echo
echo "=== Mail Test ==="
echo "Testing mail configuration..."
/usr/local/bin/gomail test 2>/dev/null || echo "Mail test failed"
EOF
    
    # Step 8: Test web admin
    log_info "Testing Web Admin interface..."
    
    # Get the bearer token
    BEARER_TOKEN=$(ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
        -i "$SSH_KEY" "root@$DROPLET_IP" \
        "grep bearer_token /etc/gomail.yaml | awk '{print \$2}' | tr -d '\"'")
    
    log_info "Bearer token: $BEARER_TOKEN"
    
    # Test HTTPS access (will fail cert validation, that's ok)
    curl -k -s "https://$DROPLET_IP/" > /dev/null && \
        log_info "WebAdmin HTTPS is accessible" || \
        log_warn "WebAdmin HTTPS not accessible (certificate issue?)"
    
    # Summary
    log_info "========================================="
    log_info "Installation test completed successfully!"
    log_info "========================================="
    log_info "Droplet IP: $DROPLET_IP"
    log_info "Domain: $TEST_DOMAIN"
    log_info "API URL: http://$DROPLET_IP:3000"
    log_info "WebAdmin URL: https://$TEST_DOMAIN/"
    log_info "Bearer Token: $BEARER_TOKEN"
    log_info "========================================="
    
    # Always keep the droplet for manual testing
    log_info "========================================="
    log_info "Test environment ready for manual testing"
    log_info "========================================="
    log_info "SSH Access: ssh root@$DROPLET_IP"
    log_info "Domain: $TEST_DOMAIN"
    log_info "Mail Hostname: mail.$TEST_DOMAIN"
    log_info "API: http://$DROPLET_IP:3000"
    log_info "WebAdmin: https://mail.$TEST_DOMAIN/ (or https://$DROPLET_IP/)"
    log_info "Bearer Token: $BEARER_TOKEN"
    log_info ""
    log_info "Useful commands:"
    log_info "  systemctl status gomail"
    log_info "  systemctl status gomail-webadmin"
    log_info "  systemctl status postfix"
    log_info "  journalctl -u gomail -f"
    log_info "  gomail test"
    log_info ""
    log_info "To delete the droplet when done:"
    log_info "  DO_TOKEN=$DO_TOKEN curl -X DELETE 'https://api.digitalocean.com/v2/droplets/$DROPLET_ID' \\"
    log_info "    -H \"Authorization: Bearer \$DO_TOKEN\""
    log_info ""
    
    # Save access details to file
    cat > test-droplet-access.txt << EOF
Test Droplet Access Details
============================
Created: $(date)
Droplet ID: $DROPLET_ID
IP Address: $DROPLET_IP
Domain: $TEST_DOMAIN
Mail Hostname: mail.$TEST_DOMAIN
SSH: ssh root@$DROPLET_IP
API: http://$DROPLET_IP:3000
WebAdmin: https://mail.$TEST_DOMAIN/ (or https://$DROPLET_IP/)
Bearer Token: $BEARER_TOKEN

Delete command:
DO_TOKEN=$DO_TOKEN curl -X DELETE 'https://api.digitalocean.com/v2/droplets/$DROPLET_ID' -H "Authorization: Bearer \$DO_TOKEN"
EOF
    
    log_info "Access details saved to: test-droplet-access.txt"
    trap - EXIT  # Remove cleanup trap to keep droplet
}

# Function to wait for DNS TTL
wait_for_ttl() {
    log_info "Waiting 30 seconds for DNS TTL to expire..."
    for i in {30..1}; do
        echo -ne "\r${YELLOW}[WAIT]${NC} $i seconds remaining...   "
        sleep 1
    done
    echo -e "\r${GREEN}[INFO]${NC} TTL wait complete            "
}

# Run main function
main "$@"

# If we're running multiple tests, wait for TTL between them
if [ "$1" == "--loop" ]; then
    wait_for_ttl
fi