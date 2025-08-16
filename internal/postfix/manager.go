package postfix

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/grumpyguvner/gomail/internal/config"
)

type DomainManager struct {
	config *config.Config
}

func NewDomainManager(cfg *config.Config) *DomainManager {
	return &DomainManager{config: cfg}
}

func (m *DomainManager) AddDomain(domain string) error {
	// Get current domains
	domains, err := m.ListDomains()
	if err != nil {
		return fmt.Errorf("failed to get current domains: %w", err)
	}

	// Check if domain already exists
	for _, d := range domains {
		if d == domain {
			return fmt.Errorf("domain %s already configured", domain)
		}
	}

	// Add domain to domains.list
	file, err := os.OpenFile(m.config.PostfixDomainsList, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open domains.list: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(domain + "\n"); err != nil {
		return fmt.Errorf("failed to write domain: %w", err)
	}

	// Update Postfix virtual_mailbox_domains
	domains = append(domains, domain)
	domainsStr := strings.Join(domains, " ")
	
	cmd := exec.Command("postconf", "-e", fmt.Sprintf("virtual_mailbox_domains=%s", domainsStr))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update Postfix configuration: %w", err)
	}

	return nil
}

func (m *DomainManager) RemoveDomain(domain string) error {
	// Get current domains
	domains, err := m.ListDomains()
	if err != nil {
		return fmt.Errorf("failed to get current domains: %w", err)
	}

	// Filter out the domain to remove
	newDomains := []string{}
	found := false
	for _, d := range domains {
		if d != domain {
			newDomains = append(newDomains, d)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("domain %s not found", domain)
	}

	// Rewrite domains.list
	content := strings.Join(newDomains, "\n")
	if len(newDomains) > 0 {
		content += "\n"
	}
	
	if err := os.WriteFile(m.config.PostfixDomainsList, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to update domains.list: %w", err)
	}

	// Update Postfix virtual_mailbox_domains
	domainsStr := strings.Join(newDomains, " ")
	if domainsStr == "" {
		domainsStr = "localhost" // Postfix needs at least one domain
	}
	
	cmd := exec.Command("postconf", "-e", fmt.Sprintf("virtual_mailbox_domains=%s", domainsStr))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update Postfix configuration: %w", err)
	}

	return nil
}

func (m *DomainManager) ListDomains() ([]string, error) {
	domains := []string{}

	// Read from domains.list if it exists
	if _, err := os.Stat(m.config.PostfixDomainsList); err == nil {
		file, err := os.Open(m.config.PostfixDomainsList)
		if err != nil {
			return nil, fmt.Errorf("failed to open domains.list: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			domain := strings.TrimSpace(scanner.Text())
			if domain != "" && !strings.HasPrefix(domain, "#") {
				domains = append(domains, domain)
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read domains.list: %w", err)
		}
	} else {
		// Fallback to reading from Postfix configuration
		cmd := exec.Command("postconf", "virtual_mailbox_domains")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get Postfix configuration: %w", err)
		}

		// Parse output
		line := strings.TrimSpace(string(output))
		if strings.HasPrefix(line, "virtual_mailbox_domains = ") {
			domainsStr := strings.TrimPrefix(line, "virtual_mailbox_domains = ")
			if domainsStr != "" && domainsStr != "localhost" {
				domains = append(domains, strings.Fields(domainsStr)...)
			}
		}
	}

	return domains, nil
}

func (m *DomainManager) ReloadPostfix() error {
	cmd := exec.Command("systemctl", "reload", "postfix")
	return cmd.Run()
}

func (m *DomainManager) IsPostfixRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "postfix")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "active"
}

func (m *DomainManager) GetPostfixStatus() (string, error) {
	cmd := exec.Command("systemctl", "status", "postfix", "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		// Command returns non-zero if service is not running, but we still want the output
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(exitErr.Stderr), nil
		}
		return "", fmt.Errorf("failed to get Postfix status: %w", err)
	}
	return string(output), nil
}