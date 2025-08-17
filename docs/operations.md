# GoMail Operations Guide

## Service Management

### Starting and Stopping

```bash
# Start service
sudo systemctl start gomail

# Stop service
sudo systemctl stop gomail

# Restart service
sudo systemctl restart gomail

# Reload configuration (graceful)
sudo systemctl reload gomail

# Check status
systemctl status gomail
```

### Enable/Disable Auto-start

```bash
# Enable auto-start on boot
sudo systemctl enable gomail

# Disable auto-start
sudo systemctl disable gomail
```

## Monitoring

### Health Checks

```bash
# Basic health check
curl http://localhost:3000/health

# Detailed health check
curl -s http://localhost:3000/health | jq .

# Continuous monitoring
watch -n 5 'curl -s http://localhost:3000/health | jq .'
```

### Metrics

Access Prometheus metrics:

```bash
# View all metrics
curl http://localhost:9090/metrics

# Key metrics to monitor
curl -s http://localhost:9090/metrics | grep -E "gomail_emails_received|gomail_email_processing_seconds|gomail_spf_"
```

### Logs

```bash
# View recent logs
journalctl -u gomail -n 100

# Follow logs in real-time
journalctl -u gomail -f

# View logs for specific time range
journalctl -u gomail --since "2024-01-15" --until "2024-01-16"

# Filter by log level
journalctl -u gomail -p err  # Errors only
journalctl -u gomail -p warning  # Warnings and above

# Export logs
journalctl -u gomail > gomail.log
```

## Performance Monitoring

### Email Queue

```bash
# Check Postfix queue
postqueue -p

# Queue summary
postqueue -p | tail -n 1

# Flush queue (force delivery)
postqueue -f

# Delete specific message
postsuper -d MESSAGE_ID

# Delete all messages
postsuper -d ALL
```

### System Resources

```bash
# CPU and memory usage
systemctl status gomail | grep -E "Memory|CPU"

# Detailed resource usage
ps aux | grep gomail

# Connection count
ss -tan | grep :3000 | wc -l
ss -tan | grep :25 | wc -l

# Disk usage
du -sh /opt/mailserver/data
df -h /opt/mailserver
```

### Performance Metrics

Key metrics to track:

```promql
# Email processing rate
rate(gomail_emails_received_total[5m])

# Average processing time
rate(gomail_email_processing_seconds_sum[5m]) / rate(gomail_email_processing_seconds_count[5m])

# Error rate
rate(gomail_emails_received_total{status="error"}[5m])

# API latency (p95)
histogram_quantile(0.95, rate(gomail_http_request_duration_seconds_bucket[5m]))
```

## Backup and Recovery

### Backup Strategy

```bash
#!/bin/bash
# backup.sh - Daily backup script

BACKUP_DIR="/backup/gomail"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_PATH="${BACKUP_DIR}/${DATE}"

# Create backup directory
mkdir -p "${BACKUP_PATH}"

# Backup configuration
cp /etc/gomail.yaml "${BACKUP_PATH}/gomail.yaml"

# Backup data
tar -czf "${BACKUP_PATH}/data.tar.gz" /opt/mailserver/data

# Backup TLS certificates
tar -czf "${BACKUP_PATH}/certs.tar.gz" /etc/gomail/certs

# Backup Postfix config
postconf -n > "${BACKUP_PATH}/postfix.conf"

# Rotate old backups (keep 30 days)
find "${BACKUP_DIR}" -type d -mtime +30 -exec rm -rf {} \;

echo "Backup completed: ${BACKUP_PATH}"
```

### Recovery Procedure

```bash
#!/bin/bash
# restore.sh - Restore from backup

BACKUP_PATH=$1

if [ -z "$BACKUP_PATH" ]; then
    echo "Usage: $0 <backup_path>"
    exit 1
fi

# Stop service
systemctl stop gomail

# Restore configuration
cp "${BACKUP_PATH}/gomail.yaml" /etc/gomail.yaml

# Restore data
tar -xzf "${BACKUP_PATH}/data.tar.gz" -C /

# Restore certificates
tar -xzf "${BACKUP_PATH}/certs.tar.gz" -C /

# Start service
systemctl start gomail

echo "Restore completed from: ${BACKUP_PATH}"
```

### Automated Backups

Add to crontab:

```bash
# Daily backup at 2 AM
0 2 * * * /usr/local/bin/backup-gomail.sh

# Weekly full backup on Sunday
0 3 * * 0 /usr/local/bin/backup-gomail-full.sh
```

## Maintenance Tasks

### Log Rotation

Configure logrotate (`/etc/logrotate.d/gomail`):

```
/var/log/gomail/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0640 gomail gomail
    postrotate
        systemctl reload gomail
    endscript
}
```

### Data Cleanup

```bash
# Clean old email files (30+ days)
find /opt/mailserver/data -type f -mtime +30 -delete

# Clean empty directories
find /opt/mailserver/data -type d -empty -delete

# Automated cleanup (add to cron)
0 4 * * * find /opt/mailserver/data -type f -mtime +30 -delete
```

### Certificate Renewal

```bash
# Check certificate expiry
openssl x509 -in /etc/gomail/certs/cert.pem -noout -enddate

# Renew Let's Encrypt certificate
gomail ssl renew

# Automated renewal (add to cron)
0 2 * * 1 /usr/local/bin/gomail ssl renew && systemctl reload gomail
```

## Troubleshooting

### Common Issues

#### Service Won't Start

```bash
# Check for errors
journalctl -u gomail -n 50 --no-pager

# Validate configuration
gomail config validate

# Check port conflicts
ss -tlnp | grep -E ":3000|:25"

# Check file permissions
ls -la /etc/gomail.yaml
ls -la /opt/mailserver/data
```

#### Emails Not Processing

```bash
# Check Postfix queue
postqueue -p

# Check API connectivity
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:3000/health

# Check webhook endpoint
curl -X POST -H "Content-Type: application/json" YOUR_WEBHOOK_URL

# Review error logs
journalctl -u gomail -p err -n 100
```

#### High Memory Usage

```bash
# Check current usage
ps aux | grep gomail

# Review goroutine count
curl -s http://localhost:9090/metrics | grep go_goroutines

# Check for memory leaks
curl -s http://localhost:9090/metrics | grep go_memstats

# Restart if necessary
systemctl restart gomail
```

#### Performance Issues

```bash
# Check processing times
curl -s http://localhost:9090/metrics | grep gomail_email_processing_seconds

# Monitor rate limiting
curl -s http://localhost:9090/metrics | grep rate_limit

# Check connection pool
curl -s http://localhost:9090/metrics | grep connection_pool

# Analyze slow queries
journalctl -u gomail | grep "slow_request"
```

## Scaling

### Vertical Scaling

Adjust resources:

```yaml
# Increase connection pool
connection_pool_size: 50

# Increase rate limits
rate_limit_per_minute: 300

# Adjust timeouts for better hardware
http_timeouts:
  read: 10s
  write: 10s
```

### Horizontal Scaling (Future)

Load balancer configuration:

```nginx
upstream gomail_backends {
    server gomail1.internal:3000;
    server gomail2.internal:3000;
    server gomail3.internal:3000;
}

server {
    listen 443 ssl;
    
    location / {
        proxy_pass http://gomail_backends;
        proxy_set_header Authorization $http_authorization;
    }
}
```

## Alerting

### Prometheus Alerts

```yaml
# alerts.yml
groups:
  - name: gomail
    rules:
      - alert: HighErrorRate
        expr: rate(gomail_emails_received_total{status="error"}[5m]) > 0.1
        for: 5m
        annotations:
          summary: "High email error rate"
          
      - alert: SlowProcessing
        expr: rate(gomail_email_processing_seconds_sum[5m]) / rate(gomail_email_processing_seconds_count[5m]) > 5
        for: 10m
        annotations:
          summary: "Slow email processing"
          
      - alert: ServiceDown
        expr: up{job="gomail"} == 0
        for: 1m
        annotations:
          summary: "GoMail service is down"
```

### Health Check Monitoring

External monitoring setup:

```bash
# UptimeRobot / Pingdom configuration
URL: https://mail.example.com/health
Method: GET
Expected Status: 200
Check Frequency: 5 minutes
```

## Disaster Recovery

### Failure Scenarios

#### Complete Server Failure

1. Provision new server
2. Run installation script
3. Restore from backup
4. Update DNS if IP changed
5. Verify service

#### Data Corruption

1. Stop service
2. Identify corruption extent
3. Restore from last good backup
4. Replay webhook deliveries if needed

#### Security Breach

1. Rotate all tokens immediately
2. Review audit logs
3. Ban suspicious IPs
4. Update and patch
5. Security audit

### RTO and RPO Targets

- **Recovery Time Objective (RTO)**: 1 hour
- **Recovery Point Objective (RPO)**: 24 hours
- **Backup Retention**: 30 days
- **Test Frequency**: Quarterly

## Capacity Planning

### Storage Requirements

```bash
# Calculate daily email volume
DAILY_EMAILS=1000
AVG_SIZE_KB=50
DAILY_STORAGE_MB=$((DAILY_EMAILS * AVG_SIZE_KB / 1024))

# Project monthly storage
MONTHLY_STORAGE_GB=$((DAILY_STORAGE_MB * 30 / 1024))

echo "Daily: ${DAILY_STORAGE_MB}MB"
echo "Monthly: ${MONTHLY_STORAGE_GB}GB"
```

### Resource Planning

| Emails/Day | CPU Cores | RAM | Storage/Month |
|------------|-----------|-----|---------------|
| < 1,000    | 1         | 512MB | 2GB |
| < 10,000   | 2         | 1GB | 20GB |
| < 100,000  | 4         | 2GB | 200GB |
| < 1,000,000| 8         | 4GB | 2TB |

## Maintenance Windows

### Planned Maintenance

```bash
# Announce maintenance
echo "Maintenance scheduled for $(date -d 'next Sunday 02:00')"

# During maintenance window
systemctl stop gomail
# Perform updates
systemctl start gomail

# Verify service
curl http://localhost:3000/health
```

### Zero-Downtime Updates

```bash
# For future cluster deployments
# Rolling update pattern
for server in gomail1 gomail2 gomail3; do
    ssh $server "systemctl stop gomail"
    ssh $server "update-gomail.sh"
    ssh $server "systemctl start gomail"
    sleep 30
done
```

## Support

- Logs: `journalctl -u gomail -f`
- Metrics: `http://localhost:9090/metrics`
- Health: `http://localhost:3000/health`
- GitHub: https://github.com/grumpyguvner/gomail/issues