# GoMail API Documentation

## Overview

GoMail provides a REST API for email operations. All endpoints except health and metrics require bearer token authentication.

## Authentication

Include your bearer token in the Authorization header:

```http
Authorization: Bearer your-api-token-here
```

## Base URL

Default: `http://localhost:3000`

Configure with the `port` setting in `/etc/gomail.yaml`

## Endpoints

### POST /mail/inbound

Receives email from Postfix pipe transport. This endpoint is called automatically by Postfix when email arrives.

#### Request Headers
- `Authorization: Bearer <token>` (required)
- `Content-Type: application/json`
- `X-Request-ID: <uuid>` (optional, will be generated if not provided)

#### Request Body

```json
{
  "sender": "sender@example.org",
  "recipient": "recipient@yourdomain.com",
  "raw": "Full RFC822 email message including headers and body..."
}
```

#### Response

**Success (200 OK)**
```json
{
  "status": "success",
  "message": "Email processed successfully",
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Authentication Failed (401 Unauthorized)**
```json
{
  "error": "unauthorized",
  "message": "Invalid or missing bearer token"
}
```

**Rate Limited (429 Too Many Requests)**
```json
{
  "error": "rate_limited",
  "message": "Too many requests",
  "retry_after": 60
}
```

Headers included in rate limited response:
- `X-RateLimit-Limit: 60`
- `X-RateLimit-Remaining: 0`
- `X-RateLimit-Reset: 1640995200`

**Server Error (500 Internal Server Error)**
```json
{
  "error": "internal_error",
  "message": "An internal error occurred",
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### GET /health

Health check endpoint for monitoring. No authentication required.

#### Response

**Healthy (200 OK)**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "uptime": 3600
}
```

**Unhealthy (503 Service Unavailable)**
```json
{
  "status": "unhealthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "errors": ["database connection failed"]
}
```

### GET /metrics

Prometheus metrics endpoint. No authentication required.

#### Response

```text
# HELP gomail_emails_received_total Total number of emails received
# TYPE gomail_emails_received_total counter
gomail_emails_received_total{domain="example.com",status="success"} 1234

# HELP gomail_email_processing_seconds Email processing duration
# TYPE gomail_email_processing_seconds histogram
gomail_email_processing_seconds_bucket{domain="example.com",le="0.1"} 100
gomail_email_processing_seconds_bucket{domain="example.com",le="0.5"} 950
gomail_email_processing_seconds_bucket{domain="example.com",le="1"} 1200

# HELP gomail_spf_pass_total Total number of SPF pass results
# TYPE gomail_spf_pass_total counter
gomail_spf_pass_total 890

# HELP gomail_dkim_pass_total Total number of DKIM pass results  
# TYPE gomail_dkim_pass_total counter
gomail_dkim_pass_total 750
```

## Webhook Integration

GoMail forwards processed emails to your configured webhook endpoint.

### Webhook Configuration

Set your webhook URL in the configuration:

```yaml
api_endpoint: https://your-app.com/email-webhook
```

Or via command:

```bash
gomail config set api_endpoint https://your-app.com/email-webhook
```

### Webhook Payload

GoMail sends a POST request to your webhook with this JSON payload:

```json
{
  "sender": "from@example.org",
  "recipient": "to@yourdomain.com",
  "received_at": "2024-01-15T10:30:00Z",
  "raw": "From: from@example.org\r\nTo: to@yourdomain.com\r\nSubject: Test Email\r\n\r\nEmail body content...",
  "subject": "Test Email",
  "message_id": "<unique-id@example.org>",
  "from_header": "John Doe <from@example.org>",
  "to_header": "Jane Smith <to@yourdomain.com>",
  "authentication": {
    "spf": {
      "result": "pass",
      "client_ip": "192.0.2.1",
      "mail_from": "from@example.org",
      "helo_domain": "mail.example.org",
      "received_spf_header": "Pass (IP: 192.0.2.1 in SPF record)"
    },
    "dkim": {
      "result": "pass",
      "signatures": [
        {
          "domain": "example.org",
          "selector": "default",
          "algorithm": "rsa-sha256",
          "valid": true
        }
      ],
      "from_domain": "example.org",
      "signed_by": ["example.org"]
    },
    "dmarc": {
      "result": "pass",
      "policy": "quarantine",
      "from_header": "from@example.org",
      "return_path": "bounce@example.org",
      "spf_alignment": "strict",
      "dkim_alignment": "relaxed",
      "authentication_results": "example.com; spf=pass smtp.mailfrom=example.org; dkim=pass header.d=example.org; dmarc=pass header.from=example.org"
    }
  },
  "metadata": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "processing_time_ms": 125,
    "size_bytes": 4096,
    "attachments": 2,
    "spam_score": 0.1
  }
}
```

### Webhook Requirements

Your webhook endpoint should:

1. **Accept POST requests** with JSON content
2. **Respond quickly** (within 30 seconds)
3. **Return appropriate status codes**:
   - `200 OK`: Email accepted
   - `400 Bad Request`: Invalid email data
   - `401 Unauthorized`: Authentication failed
   - `429 Too Many Requests`: Rate limited
   - `500 Internal Server Error`: Temporary failure

4. **Handle retries**: GoMail will retry failed webhooks with exponential backoff

### Webhook Security

Secure your webhook endpoint:

1. **Verify bearer token** if you configure one
2. **Use HTTPS** for encrypted transmission
3. **Validate request origin** by IP if needed
4. **Implement rate limiting** on your end
5. **Log requests** for audit purposes

## Error Handling

### Error Response Format

All error responses follow this structure:

```json
{
  "error": "error_type",
  "message": "Human-readable error message",
  "details": {
    "field": "Additional context"
  },
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Error Types

| Error Type | HTTP Status | Description |
|------------|-------------|-------------|
| `validation_error` | 400 | Invalid request data |
| `unauthorized` | 401 | Missing or invalid authentication |
| `forbidden` | 403 | Insufficient permissions |
| `not_found` | 404 | Resource not found |
| `method_not_allowed` | 405 | HTTP method not supported |
| `conflict` | 409 | Resource conflict |
| `payload_too_large` | 413 | Request body too large |
| `rate_limited` | 429 | Too many requests |
| `internal_error` | 500 | Server error |
| `not_implemented` | 501 | Feature not implemented |
| `service_unavailable` | 503 | Service temporarily unavailable |
| `timeout` | 504 | Request timeout |

## Rate Limiting

Default limits:
- **60 requests per minute** per IP address
- **Burst of 10 requests** allowed
- Configurable via `rate_limit_per_minute` and `rate_limit_burst`

Rate limit headers included in all responses:
- `X-RateLimit-Limit`: Maximum requests per minute
- `X-RateLimit-Remaining`: Requests remaining
- `X-RateLimit-Reset`: Unix timestamp when limit resets

## Request Tracking

All requests are assigned a unique ID for tracking:

1. **Client-provided**: Include `X-Request-ID` header
2. **Auto-generated**: System generates UUID if not provided

The request ID is:
- Included in response headers as `X-Request-ID`
- Included in error responses
- Logged for debugging
- Passed to webhooks for correlation

## Timeouts

Default timeouts:
- **Read timeout**: 30 seconds
- **Write timeout**: 30 seconds
- **Handler timeout**: 60 seconds
- **Idle timeout**: 120 seconds

Configure in `/etc/gomail.yaml`:

```yaml
http_timeouts:
  read: 30s
  write: 30s
  handler: 60s
  idle: 120s
```

## Size Limits

- **Maximum email size**: 25MB (configurable)
- **Maximum JSON payload**: 26MB
- **Maximum header size**: 1MB

Configure via:

```yaml
max_message_size: 26214400  # bytes
```

## API Examples

### Using cURL

```bash
# Test health endpoint
curl http://localhost:3000/health

# Test with authentication
curl -H "Authorization: Bearer your-token" \
     http://localhost:3000/health

# Send test email (usually done by Postfix)
curl -X POST \
     -H "Authorization: Bearer your-token" \
     -H "Content-Type: application/json" \
     -d '{"sender":"test@example.com","recipient":"user@yourdomain.com","raw":"From: test@example.com\r\nTo: user@yourdomain.com\r\nSubject: Test\r\n\r\nTest message"}' \
     http://localhost:3000/mail/inbound
```

### Using Node.js

```javascript
const axios = require('axios');

// Configure client
const client = axios.create({
  baseURL: 'http://localhost:3000',
  headers: {
    'Authorization': 'Bearer your-token'
  }
});

// Health check
const health = await client.get('/health');
console.log('Health:', health.data);

// Handle webhook
app.post('/webhook', (req, res) => {
  const email = req.body;
  
  // Process email
  console.log('Email from:', email.sender);
  console.log('SPF result:', email.authentication.spf.result);
  
  // Store or process as needed
  
  res.status(200).json({ status: 'success' });
});
```

### Using Python

```python
import requests

# Configuration
base_url = 'http://localhost:3000'
headers = {
    'Authorization': 'Bearer your-token'
}

# Health check
response = requests.get(f'{base_url}/health')
print(f'Health: {response.json()}')

# Webhook handler (Flask example)
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route('/webhook', methods=['POST'])
def handle_webhook():
    email = request.json
    
    # Process email
    print(f"Email from: {email['sender']}")
    print(f"SPF: {email['authentication']['spf']['result']}")
    
    # Your processing logic here
    
    return jsonify({'status': 'success'}), 200
```

## Monitoring Integration

### Prometheus Queries

```promql
# Email receive rate (per minute)
rate(gomail_emails_received_total[1m])

# Average processing time
rate(gomail_email_processing_seconds_sum[5m]) / rate(gomail_email_processing_seconds_count[5m])

# Authentication failure rate
rate(gomail_spf_fail_total[5m]) + rate(gomail_dkim_fail_total[5m])

# API error rate
rate(gomail_http_requests_total{status=~"5.."}[5m])
```

### Grafana Dashboard

Import the provided dashboard JSON from the repository for:
- Email traffic visualization
- Authentication metrics
- API performance
- Error tracking
- System health

## Security Considerations

1. **Always use HTTPS in production** for API and webhooks
2. **Rotate bearer tokens regularly**
3. **Implement IP whitelisting** if possible
4. **Monitor for unusual patterns** in metrics
5. **Set up alerts** for authentication failures
6. **Use webhook signing** for additional security

## Versioning

The API follows semantic versioning. Version is included in:
- Health endpoint response
- Binary version (`gomail --version`)
- Metrics labels

## Support

- GitHub Issues: https://github.com/grumpyguvner/gomail/issues
- API Status: Check `/health` endpoint
- Metrics: Check `/metrics` endpoint