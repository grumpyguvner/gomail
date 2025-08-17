package digitalocean

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Droplet represents a DigitalOcean droplet
type Droplet struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Networks Networks `json:"networks"`
}

// Networks contains droplet network information
type Networks struct {
	V4 []NetworkV4 `json:"v4"`
	V6 []NetworkV6 `json:"v6"`
}

// NetworkV4 represents IPv4 network information
type NetworkV4 struct {
	IPAddress string `json:"ip_address"`
	Netmask   string `json:"netmask"`
	Gateway   string `json:"gateway"`
	Type      string `json:"type"`
}

// NetworkV6 represents IPv6 network information
type NetworkV6 struct {
	IPAddress string `json:"ip_address"`
	Netmask   int    `json:"netmask"`
	Gateway   string `json:"gateway"`
	Type      string `json:"type"`
}

// GetCurrentDroplet finds the current droplet by matching local IP address
func (c *Client) GetCurrentDroplet() (*Droplet, error) {
	// Get local IP address
	localIP, err := getLocalPublicIP()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IP: %w", err)
	}

	// Get all droplets
	droplets, err := c.ListDroplets()
	if err != nil {
		return nil, fmt.Errorf("failed to list droplets: %w", err)
	}

	// Find droplet with matching IP
	for _, droplet := range droplets {
		for _, network := range droplet.Networks.V4 {
			if network.Type == "public" && network.IPAddress == localIP {
				return &droplet, nil
			}
		}
	}

	return nil, fmt.Errorf("could not find droplet with IP %s", localIP)
}

// ListDroplets retrieves all droplets in the account
func (c *Client) ListDroplets() ([]Droplet, error) {
	resp, err := c.doRequest("GET", "/droplets", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list droplets: %w", err)
	}

	var result struct {
		Droplets []Droplet `json:"droplets"`
		Links    struct {
			Pages struct {
				Next string `json:"next"`
			} `json:"pages"`
		} `json:"links"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Droplets, nil
}

// RenameDroplet changes the name of a droplet using the actions API
func (c *Client) RenameDroplet(dropletID int, newName string) error {
	body := map[string]string{
		"type": "rename",
		"name": newName,
	}

	path := fmt.Sprintf("/droplets/%d/actions", dropletID)
	_, err := c.doRequest("POST", path, body)
	if err != nil {
		return fmt.Errorf("failed to rename droplet: %w", err)
	}

	return nil
}

// SetupPTRRecord ensures the droplet is named correctly for PTR record
func (c *Client) SetupPTRRecord(mailHostname string) error {
	// Get current droplet
	droplet, err := c.GetCurrentDroplet()
	if err != nil {
		return fmt.Errorf("failed to get current droplet: %w", err)
	}

	// Check if droplet name already matches
	if droplet.Name == mailHostname {
		// Already configured correctly
		return nil
	}

	// Rename droplet to match mail hostname
	// This automatically configures the PTR record in DigitalOcean
	if err := c.RenameDroplet(droplet.ID, mailHostname); err != nil {
		return fmt.Errorf("failed to rename droplet: %w", err)
	}

	// Update local hostname to match
	if err := updateLocalHostname(mailHostname); err != nil {
		// Don't fail the whole operation, but warn
		fmt.Printf("Warning: failed to update local hostname: %v\n", err)
	}

	return nil
}

// updateLocalHostname updates the local system hostname and /etc/hosts
func updateLocalHostname(hostname string) error {
	// Update the running hostname
	if err := exec.Command("hostnamectl", "set-hostname", hostname).Run(); err != nil {
		// Try the older method if hostnamectl isn't available
		if err := exec.Command("hostname", hostname).Run(); err != nil {
			return fmt.Errorf("failed to set hostname: %w", err)
		}
	}

	// Update /etc/hostname
	if err := os.WriteFile("/etc/hostname", []byte(hostname+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write /etc/hostname: %w", err)
	}

	// Update /etc/hosts
	if err := updateEtcHosts(hostname); err != nil {
		return fmt.Errorf("failed to update /etc/hosts: %w", err)
	}

	return nil
}

// updateEtcHosts updates the /etc/hosts file with the new hostname
func updateEtcHosts(hostname string) error {
	// Read current /etc/hosts
	content, err := os.ReadFile("/etc/hosts")
	if err != nil {
		return fmt.Errorf("failed to read /etc/hosts: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	updated := false

	// Get the short hostname (without domain)
	shortHostname := hostname
	if idx := strings.Index(hostname, "."); idx > 0 {
		shortHostname = hostname[:idx]
	}

	for _, line := range lines {
		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			newLines = append(newLines, line)
			continue
		}

		// Check if this is a localhost line that needs updating
		if strings.HasPrefix(trimmed, "127.0.1.1") || strings.HasPrefix(trimmed, "127.0.0.1") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				ip := fields[0]
				if ip == "127.0.1.1" {
					// Update the 127.0.1.1 line with new hostname
					newLines = append(newLines, fmt.Sprintf("127.0.1.1\t%s %s", hostname, shortHostname))
					updated = true
				} else if ip == "127.0.0.1" {
					// Keep localhost line as is, but ensure it has localhost
					if !strings.Contains(line, "localhost") {
						newLines = append(newLines, "127.0.0.1\tlocalhost")
					} else {
						newLines = append(newLines, line)
					}
				} else {
					newLines = append(newLines, line)
				}
			} else {
				newLines = append(newLines, line)
			}
		} else {
			newLines = append(newLines, line)
		}
	}

	// If we didn't find a 127.0.1.1 line, add one
	if !updated {
		// Find where to insert (after 127.0.0.1 line)
		for i, line := range newLines {
			if strings.Contains(line, "127.0.0.1") {
				// Insert after this line
				newLines = append(newLines[:i+1], append([]string{fmt.Sprintf("127.0.1.1\t%s %s", hostname, shortHostname)}, newLines[i+1:]...)...)
				break
			}
		}
	}

	// Write back to /etc/hosts
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile("/etc/hosts", []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write /etc/hosts: %w", err)
	}

	return nil
}

// GetDropletPublicIP returns the public IP of the droplet
func (c *Client) GetDropletPublicIP(dropletID int) (string, error) {
	path := fmt.Sprintf("/droplets/%d", dropletID)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get droplet: %w", err)
	}

	var result struct {
		Droplet Droplet `json:"droplet"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Find public IPv4
	for _, network := range result.Droplet.Networks.V4 {
		if network.Type == "public" {
			return network.IPAddress, nil
		}
	}

	return "", fmt.Errorf("no public IP found for droplet")
}

// getLocalPublicIP gets the local server's public IP address
func getLocalPublicIP() (string, error) {
	// Get all network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Skip if not IPv4
			if ip == nil || ip.To4() == nil {
				continue
			}

			// Skip private addresses
			if ip.IsLoopback() || ip.IsPrivate() {
				continue
			}

			// This should be our public IP
			return ip.String(), nil
		}
	}

	// If we can't find it from interfaces, try a different approach
	// Get the IP that would be used to connect to an external address
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("failed to determine public IP: %w", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip := localAddr.IP.String()

	// If this is a private IP, we might be behind NAT
	// In that case, we'll need to get the IP from metadata service
	if strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "172.") {
		// Try DigitalOcean metadata service
		return getIPFromMetadata()
	}

	return ip, nil
}

// getIPFromMetadata gets the public IP from DigitalOcean metadata service
func getIPFromMetadata() (string, error) {
	// DigitalOcean metadata service
	// This is only available from within a DO droplet
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://169.254.169.254/metadata/v1/interfaces/public/0/ipv4/address")
	if err != nil {
		return "", fmt.Errorf("failed to get IP from metadata service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read metadata response: %w", err)
	}

	return strings.TrimSpace(string(body)), nil
}
