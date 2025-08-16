package commands

import (
	"fmt"
	"os/exec"

	"github.com/grumpyguvner/gomail/internal/config"
	"github.com/spf13/cobra"
)

func NewTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test the mail server configuration",
		Long:  `Run a comprehensive test of the mail server setup including Postfix, API, and email delivery.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Running mail server tests...")
			fmt.Println()

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Test 1: Check Postfix status
			fmt.Println("1. Checking Postfix status...")
			if err := testPostfixStatus(); err != nil {
				fmt.Printf("   ✗ Postfix check failed: %v\n", err)
			} else {
				fmt.Println("   ✓ Postfix is running")
			}

			// Test 2: Check API service
			fmt.Println("\n2. Checking API service...")
			if err := testAPIService(cfg); err != nil {
				fmt.Printf("   ✗ API service check failed: %v\n", err)
			} else {
				fmt.Println("   ✓ API service is responding")
			}

			// Test 3: Check port 25
			fmt.Println("\n3. Checking SMTP port (25)...")
			if err := testPort25(); err != nil {
				fmt.Printf("   ✗ Port 25 check failed: %v\n", err)
			} else {
				fmt.Println("   ✓ Port 25 is open")
			}

			// Test 4: Send test email
			fmt.Println("\n4. Sending test email...")
			if err := sendTestEmail(cfg); err != nil {
				fmt.Printf("   ✗ Test email failed: %v\n", err)
			} else {
				fmt.Println("   ✓ Test email sent successfully")
			}

			fmt.Println("\nTest complete!")
			return nil
		},
	}
}

func testPostfixStatus() error {
	cmd := exec.Command("systemctl", "is-active", "postfix")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Postfix is not running")
	}
	return nil
}

func testAPIService(cfg *config.Config) error {
	// Check if service is running
	cmd := exec.Command("systemctl", "is-active", "mailserver")
	if err := cmd.Run(); err != nil {
		// Try to check if it's running directly
		url := fmt.Sprintf("http://localhost:%d/health", cfg.Port)
		cmd = exec.Command("curl", "-s", "-f", url)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("API service is not responding")
		}
	}
	return nil
}

func testPort25() error {
	cmd := exec.Command("nc", "-zv", "localhost", "25")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("port 25 is not accessible")
	}
	return nil
}

func sendTestEmail(cfg *config.Config) error {
	// Use swaks if available, otherwise use sendmail
	testAddr := fmt.Sprintf("test@%s", cfg.PrimaryDomain)
	
	// Try swaks first
	cmd := exec.Command("which", "swaks")
	if err := cmd.Run(); err == nil {
		cmd = exec.Command("swaks", "--to", testAddr, "--server", "localhost", "--silent", "--quit-after", "RCPT")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to send test email with swaks: %w", err)
		}
	} else {
		// Fallback to echo | sendmail
		cmd = exec.Command("sh", "-c", fmt.Sprintf("echo 'Test email' | sendmail -f test@example.com %s", testAddr))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to send test email: %w", err)
		}
	}
	
	return nil
}