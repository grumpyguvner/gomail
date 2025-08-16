# GoMail Production Readiness - Sprint 2 Day 4 Continuation

## Current Status: Sprint 2 Day 3 Complete (Aug 16, 2025)

**Project:** GoMail - Modern mail server in Go  
**Repository:** https://github.com/grumpyguvner/gomail  
**Working Directory:** /root/postfix

## ✅ Sprint 2 Completed (Days 1-3):

### Day 1-2: Rate Limiting & Config Validation ✓
- Token bucket rate limiting (60 req/min, configurable)
- X-RateLimit headers in responses
- JSON schema validation for configuration
- New `validate` command with --show-schema option
- 94.0% test coverage for middleware
- 89.3% test coverage for config

### Day 3: Graceful Shutdown ✓
- 30-second timeout for connection draining
- Active request tracking with atomic counters
- Reject new requests during shutdown (503 response)
- Progress logging every 5 seconds
- Force shutdown after timeout with warnings
- Comprehensive test suite (78.5% coverage for api package)

## 📊 Current Metrics:
- **Test Coverage:** 48.2% (Target: 50% by end of Sprint 2)
- **Completed Features:** 3/6 (50%)
- **Lines of Code:** ~3,500 (Go code only)
- **Test Files:** 15 with 85+ test cases
- **CI Status:** ✅ All checks passing

## 🎯 Day 4 Task: Prometheus Metrics Integration

### Requirements:
1. **HTTP Metrics:**
   - Request duration histogram (by method, endpoint, status)
   - Active connections gauge
   - Request rate counter
   - Response size histogram

2. **Email Metrics:**
   - Emails processed counter (by status)
   - Email size histogram
   - Processing duration histogram
   - Storage operations counter

3. **System Metrics:**
   - Go runtime metrics (goroutines, memory, GC)
   - Rate limit metrics (allowed/denied)
   - Shutdown metrics (graceful vs forced)

4. **Configuration:**
   - Enable/disable metrics endpoint
   - Custom metrics port (default: 9090)
   - Metrics path (default: /metrics)

## 📝 Validation Commands for Current Implementation:

```bash
# 1. Verify Sprint 2 implementations are working
cd /root/postfix

# Check git status and recent commits
git status
git log --oneline -5

# 2. Test rate limiting functionality
go run cmd/mailserver/main.go server --config /dev/null &
SERVER_PID=$!
sleep 2

# Send 15 rapid requests to test rate limiting
for i in {1..15}; do
  curl -s -X POST http://localhost:3000/mail/inbound \
    -H "Authorization: Bearer test-token" \
    -w "Request $i: %{http_code} - " \
    -H "X-Real-IP: 192.168.1.$i" \
    2>/dev/null
  curl -s http://localhost:3000/metrics | jq -r '.active_requests'
done

kill $SERVER_PID 2>/dev/null

# 3. Test config validation
echo '{"port": -1, "bearer_token": "weak"}' > /tmp/bad-config.json
go run cmd/mailserver/main.go validate --config /tmp/bad-config.json

# Show schema
go run cmd/mailserver/main.go validate --show-schema | jq '.properties | keys'

# 4. Test graceful shutdown
timeout 10 go run cmd/mailserver/main.go server --config /dev/null &
SERVER_PID=$!
sleep 2

# Check metrics during operation
curl -s http://localhost:3000/metrics | jq '.'

# Send SIGTERM and observe graceful shutdown
kill -TERM $SERVER_PID
wait $SERVER_PID

# 5. Run test suite and check coverage
go test -v ./... -run "Test.*Shutdown" | grep -E "(PASS|FAIL)"
go test -cover ./... | grep -E "coverage:|ok"
go test -coverprofile=coverage.out ./... 2>/dev/null
go tool cover -func=coverage.out | tail -1

# 6. Verify no race conditions
go test -race ./internal/api -run TestServer_GracefulShutdown
```

## 🚀 Day 4 Implementation Plan:

### 1. Add Prometheus Dependencies
```bash
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
go get github.com/prometheus/client_golang/prometheus/collectors
```

### 2. Create Metrics Package
Create `/root/postfix/internal/metrics/metrics.go`:
```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/collectors"
)

var (
    // HTTP Metrics
    HTTPRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "HTTP request latencies in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint", "status"},
    )
    
    HTTPActiveRequests = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "http_active_requests",
            Help: "Number of active HTTP requests",
        },
    )
    
    // Email Metrics
    EmailsProcessed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "emails_processed_total",
            Help: "Total number of emails processed",
        },
        []string{"status"},
    )
    
    EmailSize = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name: "email_size_bytes",
            Help: "Size of processed emails in bytes",
            Buckets: prometheus.ExponentialBuckets(1024, 2, 15), // 1KB to 16MB
        },
    )
    
    // Rate Limit Metrics
    RateLimitHits = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rate_limit_hits_total",
            Help: "Number of rate limit hits",
        },
        []string{"action"}, // "allowed" or "denied"
    )
)

func Init() {
    // Register all metrics
    prometheus.MustRegister(HTTPRequestDuration)
    prometheus.MustRegister(HTTPActiveRequests)
    prometheus.MustRegister(EmailsProcessed)
    prometheus.MustRegister(EmailSize)
    prometheus.MustRegister(RateLimitHits)
    
    // Register Go runtime metrics
    prometheus.MustRegister(collectors.NewGoCollector())
    prometheus.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
}
```

### 3. Add Prometheus Middleware
Update `/root/postfix/internal/middleware/prometheus.go`:
```go
package middleware

import (
    "net/http"
    "strconv"
    "time"
    
    "github.com/grumpyguvner/gomail/internal/metrics"
)

func PrometheusMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Track active requests
        metrics.HTTPActiveRequests.Inc()
        defer metrics.HTTPActiveRequests.Dec()
        
        // Wrap response writer to capture status
        wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
        
        // Process request
        next.ServeHTTP(wrapped, r)
        
        // Record metrics
        duration := time.Since(start).Seconds()
        status := strconv.Itoa(wrapped.statusCode)
        
        metrics.HTTPRequestDuration.WithLabelValues(
            r.Method,
            r.URL.Path,
            status,
        ).Observe(duration)
    })
}
```

### 4. Update Server Configuration
Add to `/root/postfix/internal/config/config.go`:
```go
type Config struct {
    // ... existing fields ...
    
    // Metrics configuration
    MetricsEnabled bool   `json:"metrics_enabled" yaml:"metrics_enabled"`
    MetricsPort    int    `json:"metrics_port" yaml:"metrics_port"`
    MetricsPath    string `json:"metrics_path" yaml:"metrics_path"`
}
```

### 5. Start Metrics Server
Update `/root/postfix/internal/commands/server.go` to start metrics server:
```go
// Start metrics server if enabled
if cfg.MetricsEnabled {
    go func() {
        mux := http.NewServeMux()
        mux.Handle(cfg.MetricsPath, promhttp.Handler())
        
        metricsAddr := fmt.Sprintf(":%d", cfg.MetricsPort)
        logging.Get().Infof("Starting metrics server on %s%s", metricsAddr, cfg.MetricsPath)
        
        if err := http.ListenAndServe(metricsAddr, mux); err != nil {
            logging.Get().Errorf("Metrics server error: %v", err)
        }
    }()
}
```

### 6. Integration Points
- Update rate limiter to record metrics
- Update email handler to record processing metrics
- Update shutdown handler to record shutdown type
- Add metrics to existing middleware chain

### 7. Testing Requirements
- Unit tests for metrics package
- Integration tests for Prometheus endpoint
- Verify metrics accuracy under load
- Test metrics server lifecycle

## 📋 Acceptance Criteria:
- [ ] Prometheus metrics endpoint at :9090/metrics
- [ ] All HTTP requests tracked with duration and status
- [ ] Email processing metrics recorded
- [ ] Rate limit metrics (allowed vs denied)
- [ ] Go runtime metrics exposed
- [ ] Configurable via YAML/environment
- [ ] No performance degradation (<1% overhead)
- [ ] Tests maintain >48% coverage
- [ ] Documentation updated

## 🎪 Quick Test After Implementation:
```bash
# Start server with metrics
cat > /tmp/metrics-config.yaml << EOF
port: 3000
bearer_token: test-token
metrics_enabled: true
metrics_port: 9090
metrics_path: /metrics
EOF

go run cmd/mailserver/main.go server --config /tmp/metrics-config.yaml &
SERVER_PID=$!
sleep 2

# Generate some traffic
for i in {1..10}; do
  curl -X POST http://localhost:3000/mail/inbound \
    -H "Authorization: Bearer test-token" \
    -d '{"from":"test@example.com","to":"dest@example.com"}' \
    2>/dev/null
done

# Check metrics
curl -s http://localhost:9090/metrics | grep -E "^(http_|email_|rate_limit_)"

# Check specific metrics
curl -s http://localhost:9090/metrics | grep "http_request_duration_seconds"
curl -s http://localhost:9090/metrics | grep "emails_processed_total"
curl -s http://localhost:9090/metrics | grep "go_goroutines"

kill $SERVER_PID
```

## 🔍 Common Issues & Solutions:

1. **Metric Registration Conflicts**
   - Use `prometheus.Register()` instead of `MustRegister()` in tests
   - Clear registry between tests with `prometheus.DefaultRegisterer = prometheus.NewRegistry()`

2. **High Cardinality**
   - Avoid using user IDs or IPs as labels
   - Limit endpoint labels to major routes only

3. **Memory Leaks**
   - Ensure histograms have reasonable bucket counts
   - Monitor prometheus_sd_discovered_targets metric

## 📈 Expected Outcomes:
- Comprehensive observability into system behavior
- Ability to create Grafana dashboards
- Performance baseline metrics
- Rate limit effectiveness monitoring
- Resource usage tracking

## 🚨 Remember:
1. **NO AI references in commits** - Keep commits professional
2. **Run `make check` before commits** - Ensure CI passes
3. **Update CHANGELOG.md** - Track progress with checkboxes
4. **Maintain test coverage** - Don't let it drop below 48%
5. **Follow Go idioms** - Use standard Prometheus patterns

## Next Session Focus:
After completing Prometheus metrics (Day 4), the next tasks are:
- Day 5: Error type categorization
- Day 6-7: Connection pooling

Good luck with Day 4 implementation! The goal is to have full observability by end of day.