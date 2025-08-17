package ssl

import (
	"fmt"
	"os"
	"os/exec"
)

// SetupAutoRenewal sets up automatic certificate renewal via systemd timer
func SetupAutoRenewal() error {
	// Create systemd service for renewal
	serviceContent := `[Unit]
Description=Renew GoMail SSL Certificate
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/mailserver ssl renew
User=root
StandardOutput=journal
StandardError=journal
`

	servicePath := "/etc/systemd/system/mailserver-ssl-renew.service"
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to create renewal service: %w", err)
	}

	// Create systemd timer for daily checks
	timerContent := `[Unit]
Description=Daily renewal check for GoMail SSL Certificate
Requires=mailserver-ssl-renew.service

[Timer]
OnCalendar=daily
RandomizedDelaySec=1h
Persistent=true

[Install]
WantedBy=timers.target
`

	timerPath := "/etc/systemd/system/mailserver-ssl-renew.timer"
	if err := os.WriteFile(timerPath, []byte(timerContent), 0644); err != nil {
		return fmt.Errorf("failed to create renewal timer: %w", err)
	}

	// Reload systemd daemon
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable and start the timer
	if err := exec.Command("systemctl", "enable", "mailserver-ssl-renew.timer").Run(); err != nil {
		return fmt.Errorf("failed to enable renewal timer: %w", err)
	}

	if err := exec.Command("systemctl", "start", "mailserver-ssl-renew.timer").Run(); err != nil {
		return fmt.Errorf("failed to start renewal timer: %w", err)
	}

	return nil
}

// CheckRenewalTimer checks the status of the auto-renewal timer
func CheckRenewalTimer() (string, error) {
	output, err := exec.Command("systemctl", "status", "mailserver-ssl-renew.timer").Output()
	if err != nil {
		// Check if timer exists
		if _, err := os.Stat("/etc/systemd/system/mailserver-ssl-renew.timer"); os.IsNotExist(err) {
			return "Timer not installed", nil
		}
		return "", fmt.Errorf("failed to check timer status: %w", err)
	}
	return string(output), nil
}
