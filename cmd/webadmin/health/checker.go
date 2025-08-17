package health

import (
	"sync"
	"time"

	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
)

type Checker struct {
	logger *logging.Logger
	cache  map[string]*CachedResult
	mutex  sync.RWMutex
}

type CachedResult struct {
	Health    *DomainHealth
	Timestamp time.Time
	TTL       time.Duration
}

type DomainHealth struct {
	Domain         string               `json:"domain"`
	LastChecked    time.Time            `json:"last_checked"`
	OverallScore   int                  `json:"overall_score"` // 0-100
	DNS            DNSHealth            `json:"dns"`
	SPF            SPFHealth            `json:"spf"`
	DKIM           DKIMHealth           `json:"dkim"`
	DMARC          DMARCHealth          `json:"dmarc"`
	SSL            SSLHealth            `json:"ssl"`
	Deliverability DeliverabilityHealth `json:"deliverability"`
}

type DNSHealth struct {
	Status    string   `json:"status"` // "healthy", "warning", "error"
	ARecords  []string `json:"a_records"`
	MXRecords []string `json:"mx_records"`
	PTRRecord string   `json:"ptr_record"`
	Issues    []string `json:"issues"`
	Score     int      `json:"score"` // 0-100
}

type SPFHealth struct {
	Status   string   `json:"status"`
	Record   string   `json:"record"`
	Valid    bool     `json:"valid"`
	Issues   []string `json:"issues"`
	Includes []string `json:"includes"`
	Score    int      `json:"score"` // 0-100
}

type DKIMHealth struct {
	Status    string         `json:"status"`
	Selectors []DKIMSelector `json:"selectors"`
	Issues    []string       `json:"issues"`
	Score     int            `json:"score"` // 0-100
}

type DKIMSelector struct {
	Selector string `json:"selector"`
	Record   string `json:"record"`
	Valid    bool   `json:"valid"`
	KeyType  string `json:"key_type"`
	KeySize  int    `json:"key_size"`
}

type DMARCHealth struct {
	Status  string   `json:"status"`
	Record  string   `json:"record"`
	Policy  string   `json:"policy"`
	Percent int      `json:"percent"`
	Valid   bool     `json:"valid"`
	Issues  []string `json:"issues"`
	Score   int      `json:"score"` // 0-100
}

type SSLHealth struct {
	Status   string    `json:"status"`
	Valid    bool      `json:"valid"`
	Expiry   time.Time `json:"expiry"`
	DaysLeft int       `json:"days_left"`
	Issuer   string    `json:"issuer"`
	Issues   []string  `json:"issues"`
	Score    int       `json:"score"` // 0-100
}

type DeliverabilityHealth struct {
	Status      string   `json:"status"`
	Score       int      `json:"score"` // 0-100
	Blacklisted bool     `json:"blacklisted"`
	Blacklists  []string `json:"blacklists"`
	Reputation  string   `json:"reputation"`
	Issues      []string `json:"issues"`
}

func NewChecker(logger *logging.Logger) *Checker {
	return &Checker{
		logger: logger,
		cache:  make(map[string]*CachedResult),
	}
}

func (c *Checker) CheckDomain(domain string) (*DomainHealth, error) {
	// Check cache first
	c.mutex.RLock()
	cached, exists := c.cache[domain]
	c.mutex.RUnlock()

	if exists && time.Since(cached.Timestamp) < cached.TTL {
		c.logger.Debug("Returning cached health result", "domain", domain)
		return cached.Health, nil
	}

	// Perform fresh health check
	return c.RefreshDomain(domain)
}

func (c *Checker) RefreshDomain(domain string) (*DomainHealth, error) {
	c.logger.Info("Performing health check", "domain", domain)

	health := &DomainHealth{
		Domain:      domain,
		LastChecked: time.Now(),
	}

	// Run all health checks in parallel
	var wg sync.WaitGroup
	wg.Add(5)

	// DNS Check
	go func() {
		defer wg.Done()
		health.DNS = c.checkDNS(domain)
	}()

	// SPF Check
	go func() {
		defer wg.Done()
		health.SPF = c.checkSPF(domain)
	}()

	// DKIM Check
	go func() {
		defer wg.Done()
		health.DKIM = c.checkDKIM(domain)
	}()

	// DMARC Check
	go func() {
		defer wg.Done()
		health.DMARC = c.checkDMARC(domain)
	}()

	// SSL Check
	go func() {
		defer wg.Done()
		health.SSL = c.checkSSL(domain)
	}()

	wg.Wait()

	// Deliverability check (after others complete)
	health.Deliverability = c.checkDeliverability(domain)

	// Calculate overall score
	health.OverallScore = c.calculateOverallScore(health)

	// Cache result
	c.mutex.Lock()
	c.cache[domain] = &CachedResult{
		Health:    health,
		Timestamp: time.Now(),
		TTL:       15 * time.Minute, // Cache for 15 minutes
	}
	c.mutex.Unlock()

	c.logger.Info("Health check completed",
		"domain", domain,
		"score", health.OverallScore,
		"dns_status", health.DNS.Status,
		"spf_status", health.SPF.Status,
		"dkim_status", health.DKIM.Status,
		"dmarc_status", health.DMARC.Status,
		"ssl_status", health.SSL.Status,
	)

	return health, nil
}

func (c *Checker) calculateOverallScore(health *DomainHealth) int {
	// Weighted scoring:
	// DNS: 20%
	// SPF: 20%
	// DKIM: 20%
	// DMARC: 15%
	// SSL: 15%
	// Deliverability: 10%

	score := float64(health.DNS.Score)*0.20 +
		float64(health.SPF.Score)*0.20 +
		float64(health.DKIM.Score)*0.20 +
		float64(health.DMARC.Score)*0.15 +
		float64(health.SSL.Score)*0.15 +
		float64(health.Deliverability.Score)*0.10

	return int(score)
}

func (c *Checker) checkDNS(domain string) DNSHealth {
	checker := NewDNSChecker(c.logger)
	return checker.Check(domain)
}

func (c *Checker) checkSPF(domain string) SPFHealth {
	checker := NewSPFChecker(c.logger)
	return checker.Check(domain)
}

func (c *Checker) checkDKIM(domain string) DKIMHealth {
	checker := NewDKIMChecker(c.logger)
	return checker.Check(domain)
}

func (c *Checker) checkDMARC(domain string) DMARCHealth {
	checker := NewDMARCChecker(c.logger)
	return checker.Check(domain)
}

func (c *Checker) checkSSL(domain string) SSLHealth {
	checker := NewSSLChecker(c.logger)
	return checker.Check(domain)
}

func (c *Checker) checkDeliverability(domain string) DeliverabilityHealth {
	checker := NewDeliverabilityChecker(c.logger)
	return checker.Check(domain)
}
