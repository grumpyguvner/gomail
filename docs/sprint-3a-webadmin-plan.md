# Sprint 3a: GoMail Web Administration UI

## 🎯 Objective
Create a web-based administration interface for GoMail using a Backend-for-Frontend (BFF) pattern with JAMstack frontend and comprehensive domain health monitoring.

## 🏗️ Architecture Overview

### BFF Server (Go)
- **Location**: `/cmd/webadmin/` 
- **Port**: 443 (HTTPS by default, configurable)
- **SSL**: Use existing GoMail SSL certificates
- **Purpose**: API aggregation, authentication, static asset serving, domain health checks
- **Deployment**: systemd service

### Frontend (JAMstack)
- **Technology**: Vanilla HTML/JS + TailwindCSS (local build)
- **Location**: `/webadmin/` directory
- **Architecture**: Single Page Application (SPA) with proper URL paths (no hash routing)
- **CSS**: TailwindCSS compiled locally, no CDN dependencies

### Real-time Updates
- **Technology**: Server-Sent Events (SSE)
- **Use Cases**: New email notifications, domain health status, live metrics

## 📋 Implementation Plan

### Phase 1: BFF Server Foundation (Week 1)
1. **Create BFF server structure**
   - New Go module: `cmd/webadmin/main.go`
   - HTTPS configuration with existing GoMail certificates
   - Configuration management extending GoMail config
   - HTTP server with proper SPA routing

2. **Domain Health Monitoring System** ⭐ CORE FEATURE
   - DNS record validation (A, MX, TXT records)
   - PTR record (reverse DNS) checking
   - SPF record validation and syntax checking
   - DKIM record discovery and validation
   - DMARC policy checking and compliance
   - SSL certificate validation and expiry monitoring
   - Deliverability testing (blacklist checking, reputation scoring)

3. **API Gateway Layer**
   - Proxy endpoints to GoMail API (port 3000)
   - Authentication middleware (Bearer token)
   - Background health checking with caching

### Phase 2: Core Admin APIs (Week 1-2)
1. **Email Management**
   - `GET /api/emails` - List stored emails with pagination
   - `GET /api/emails/{id}` - Get email details
   - `DELETE /api/emails/{id}` - Delete email
   - `GET /api/emails/{id}/raw` - Download raw email

2. **Domain Management**
   - `GET /api/domains` - List configured domains
   - `POST /api/domains` - Add new domain
   - `PUT /api/domains/{domain}` - Update domain configuration
   - `DELETE /api/domains/{domain}` - Remove domain

3. **Domain Health Monitoring** ⭐ NEW CORE FEATURE
   - `GET /api/domains/{domain}/health` - Comprehensive health check
   - `POST /api/domains/{domain}/health/refresh` - Force health check
   - `GET /api/domains/{domain}/dns` - DNS record analysis
   - `GET /api/domains/{domain}/spf` - SPF record validation
   - `GET /api/domains/{domain}/dkim` - DKIM record checking
   - `GET /api/domains/{domain}/dmarc` - DMARC policy analysis
   - `GET /api/domains/{domain}/ssl` - SSL certificate status
   - `GET /api/domains/{domain}/deliverability` - Blacklist/reputation check

4. **Email Routing Configuration**
   - `GET /api/routing/rules` - List all routing rules
   - `POST /api/routing/rules` - Create routing rule
   - `PUT /api/routing/rules/{id}` - Update routing rule
   - `DELETE /api/routing/rules/{id}` - Delete routing rule

5. **Real-time Updates**
   - `GET /api/events` - SSE endpoint for live updates
   - `GET /api/health` - Combined system health
   - `GET /api/metrics` - Aggregated metrics

### Phase 3: Frontend Application (Week 2)
1. **TailwindCSS Setup**
   - Local TailwindCSS installation and build process
   - Custom configuration for GoMail branding
   - Component-based CSS architecture
   - Dark/light theme support with Tailwind

2. **Core UI Framework**
   - Responsive grid layouts with Tailwind
   - Modern component library
   - SPA router with proper URL paths
   - Loading states and error handling

3. **Dashboard Views**
   - `/` - System overview dashboard
   - `/emails` - Email list with search/filter  
   - `/emails/{id}` - Email detail viewer
   - `/domains` - Domain management interface
   - `/domains/{domain}` - Domain-specific configuration
   - `/domains/{domain}/health` - Domain health dashboard ⭐ NEW
   - `/routing` - Email routing rules management
   - `/settings` - System configuration

4. **Domain Health Interface** ⭐ CORE FEATURE
   - Real-time health status indicators
   - DNS record visualization
   - SPF/DKIM/DMARC status cards
   - SSL certificate monitoring
   - Deliverability score dashboard
   - Historical health trends
   - Issue resolution guidance

### Phase 4: Advanced Features (Week 3)
1. **Advanced Domain Health**
   - Automated health check scheduling
   - Alert thresholds and notifications
   - Health trend analysis and reporting
   - Issue remediation suggestions
   - Bulk domain health checking

2. **Enhanced Email Management**
   - Full-text search across email content
   - Domain-based filtering and analytics
   - Bulk email operations
   - Export capabilities

3. **Real-time Monitoring**
   - Live health status updates via SSE
   - Critical alert notifications
   - System performance monitoring

## 🔧 Technical Specifications

### BFF Server Structure
```
cmd/webadmin/
├── main.go
├── config/
│   └── config.go
├── handlers/
│   ├── api.go
│   ├── domains.go
│   ├── health.go        ⭐ NEW - Domain health checks
│   ├── routing.go
│   ├── sse.go
│   ├── auth.go
│   └── static.go
├── middleware/
│   ├── auth.go
│   ├── cors.go
│   └── logging.go
├── health/              ⭐ NEW - Health checking engine
│   ├── dns.go
│   ├── spf.go
│   ├── dkim.go
│   ├── dmarc.go
│   ├── ssl.go
│   ├── deliverability.go
│   └── scheduler.go
└── proxy/
    └── gomail.go
```

### Frontend Structure with TailwindCSS
```
webadmin/
├── index.html
├── package.json         ⭐ NEW - For Tailwind build
├── tailwind.config.js   ⭐ NEW - Tailwind configuration
├── assets/
│   ├── css/
│   │   ├── input.css    ⭐ NEW - Tailwind input
│   │   └── output.css   ⭐ NEW - Compiled Tailwind
│   ├── js/
│   │   ├── app.js
│   │   ├── api.js
│   │   ├── router.js
│   │   ├── sse.js
│   │   └── components/
│   │       ├── domain-manager.js
│   │       ├── health-dashboard.js  ⭐ NEW
│   │       └── routing-rules.js
│   └── images/
└── views/
    ├── dashboard.html
    ├── emails.html
    ├── domains.html
    ├── domain-health.html  ⭐ NEW
    ├── routing.html
    └── settings.html
```

### Domain Health Check Structure
```go
type DomainHealth struct {
    Domain        string                 `json:"domain"`
    LastChecked   time.Time             `json:"last_checked"`
    OverallScore  int                   `json:"overall_score"` // 0-100
    DNS           DNSHealth             `json:"dns"`
    SPF           SPFHealth             `json:"spf"`
    DKIM          DKIMHealth            `json:"dkim"`
    DMARC         DMARCHealth           `json:"dmarc"`
    SSL           SSLHealth             `json:"ssl"`
    Deliverability DeliverabilityHealth `json:"deliverability"`
}

type DNSHealth struct {
    Status      string   `json:"status"` // "healthy", "warning", "error"
    ARecords    []string `json:"a_records"`
    MXRecords   []string `json:"mx_records"`
    PTRRecord   string   `json:"ptr_record"`
    Issues      []string `json:"issues"`
}

type SPFHealth struct {
    Status    string   `json:"status"`
    Record    string   `json:"record"`
    Valid     bool     `json:"valid"`
    Issues    []string `json:"issues"`
    Includes  []string `json:"includes"`
}

type DKIMHealth struct {
    Status    string             `json:"status"`
    Selectors []DKIMSelector     `json:"selectors"`
    Issues    []string           `json:"issues"`
}

type DKIMSelector struct {
    Selector string `json:"selector"`
    Record   string `json:"record"`
    Valid    bool   `json:"valid"`
    KeyType  string `json:"key_type"`
}

type DMARCHealth struct {
    Status   string   `json:"status"`
    Record   string   `json:"record"`
    Policy   string   `json:"policy"`
    Percent  int      `json:"percent"`
    Valid    bool     `json:"valid"`
    Issues   []string `json:"issues"`
}

type SSLHealth struct {
    Status     string    `json:"status"`
    Valid      bool      `json:"valid"`
    Expiry     time.Time `json:"expiry"`
    DaysLeft   int       `json:"days_left"`
    Issuer     string    `json:"issuer"`
    Issues     []string  `json:"issues"`
}

type DeliverabilityHealth struct {
    Status       string   `json:"status"`
    Score        int      `json:"score"` // 0-100
    Blacklisted  bool     `json:"blacklisted"`
    Blacklists   []string `json:"blacklists"`
    Reputation   string   `json:"reputation"`
    Issues       []string `json:"issues"`
}
```

### TailwindCSS Configuration
```js
// tailwind.config.js
module.exports = {
  content: ["./webadmin/**/*.{html,js}"],
  theme: {
    extend: {
      colors: {
        'gomail': {
          50: '#f0f9ff',
          500: '#3b82f6',
          900: '#1e3a8a'
        }
      }
    },
  },
  plugins: [],
}
```

### Configuration Extensions
```yaml
# Add to existing GoMail config
webadmin:
  enabled: true
  port: 443
  ssl_cert: "/etc/mailserver/ssl/cert.pem"  # Reuse GoMail certs
  ssl_key: "/etc/mailserver/ssl/key.pem"
  static_dir: "/opt/gomail/webadmin"
  health_check_interval: "1h"
  
domains:
  # Domain-specific email handling rules
  example.com:
    action: "store"           # store, forward, discard, bounce
    forward_to: []           # for forward action
    bounce_message: ""       # for bounce action
    health_checks: true      # enable health monitoring
```

## 🔐 Security Considerations
- HTTPS by default with existing certificates
- Rate limiting on health check endpoints
- Domain validation before health checks
- Secure DNS resolution
- SSL certificate validation
- CSRF protection
- Content Security Policy
- Input sanitization for routing rules

## 📊 Success Criteria
- [ ] HTTPS BFF server with TailwindCSS frontend
- [ ] Comprehensive domain health monitoring ⭐ CORE
- [ ] Real-time health status updates via SSE
- [ ] DNS, SPF, DKIM, DMARC, SSL validation
- [ ] Deliverability and reputation monitoring
- [ ] Domain management interface
- [ ] Email routing rules configuration
- [ ] SPA with proper URL routing (no hash)
- [ ] Mobile-responsive Tailwind design
- [ ] Performance: <200ms response times
- [ ] Test coverage: 80%+

## 🚀 Deployment Process
1. Build TailwindCSS assets
2. Build webadmin binary
3. Install systemd service
4. Configure HTTPS with existing certificates
5. Deploy static assets
6. Configure health check scheduling
7. Health check and monitoring

## 📝 Implementation Notes
- Focus on domain health monitoring as the primary value proposition
- Ensure all health checks are non-blocking and cacheable
- Implement proper error handling for external DNS/SSL checks
- Use background workers for scheduled health checks
- Provide actionable remediation guidance for health issues

**Key Focus**: Domain health monitoring as the primary administrative feature for email deliverability management.

**Estimated timeline**: 3 weeks for full implementation.