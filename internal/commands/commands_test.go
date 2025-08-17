package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigCommand(t *testing.T) {
	cmd := NewConfigCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "config", cmd.Use)
	assert.Equal(t, "Manage configuration", cmd.Short)

	// Check subcommands exist
	subcommands := []string{"show", "set", "generate"}
	for _, subcmd := range subcommands {
		found := false
		for _, c := range cmd.Commands() {
			if c.Name() == subcmd {
				found = true
				break
			}
		}
		assert.True(t, found, "Subcommand %s not found", subcmd)
	}
}

func TestNewDNSCommand(t *testing.T) {
	cmd := NewDNSCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "dns", cmd.Use)
	assert.Equal(t, "Manage DNS records", cmd.Short)

	// Check subcommands exist
	subcommands := []string{"setup", "check"}
	for _, subcmd := range subcommands {
		found := false
		for _, c := range cmd.Commands() {
			if c.Name() == subcmd {
				found = true
				break
			}
		}
		assert.True(t, found, "Subcommand %s not found", subcmd)
	}
}

func TestNewDomainCommand(t *testing.T) {
	cmd := NewDomainCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "domain", cmd.Use)
	assert.Equal(t, "Manage email domains", cmd.Short)

	// Check subcommands exist
	subcommands := []string{"add", "remove", "list", "test"}
	for _, subcmd := range subcommands {
		found := false
		for _, c := range cmd.Commands() {
			if c.Name() == subcmd {
				found = true
				break
			}
		}
		assert.True(t, found, "Subcommand %s not found", subcmd)
	}
}

func TestNewSSLCommand(t *testing.T) {
	cmd := NewSSLCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "ssl", cmd.Use)
	assert.Equal(t, "Manage SSL certificates", cmd.Short)

	// Check subcommands exist
	subcommands := []string{"setup", "renew", "status"}
	for _, subcmd := range subcommands {
		found := false
		for _, c := range cmd.Commands() {
			if c.Name() == subcmd {
				found = true
				break
			}
		}
		assert.True(t, found, "Subcommand %s not found", subcmd)
	}
}

func TestNewTestCommand(t *testing.T) {
	cmd := NewTestCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "test", cmd.Use)
	assert.Equal(t, "Test the mail server configuration", cmd.Short)
}

func TestNewInstallCommand(t *testing.T) {
	cmd := NewInstallCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "install", cmd.Use)
	assert.Equal(t, "Install and configure the mail server", cmd.Short)
}

func TestNewServerCommand(t *testing.T) {
	cmd := NewServerCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "server", cmd.Use)
	assert.Equal(t, "Run the mail API server", cmd.Short)
}

func TestConfigShowCommand(t *testing.T) {
	cmd := newConfigShowCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "show", cmd.Use)
	assert.Equal(t, "Show current configuration", cmd.Short)

	// Test execution with default config
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Running this will attempt to load config which should succeed with defaults
	err := cmd.Execute()
	// We expect this to succeed as it will use defaults
	assert.NoError(t, err)
}

func TestConfigSetCommand(t *testing.T) {
	cmd := newConfigSetCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "set [key] [value]", cmd.Use)
	assert.Equal(t, "Set a configuration value", cmd.Short)

	// Test args validation
	args := cmd.Args
	require.NotNil(t, args)

	// Test with no args
	err := args(cmd, []string{})
	assert.Error(t, err)

	// Test with one arg
	err = args(cmd, []string{"key"})
	assert.Error(t, err)

	// Test with two args (valid)
	err = args(cmd, []string{"key", "value"})
	assert.NoError(t, err)
}

func TestConfigGenerateCommand(t *testing.T) {
	cmd := newConfigGenerateCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "generate", cmd.Use)
	assert.Equal(t, "Generate a new configuration file", cmd.Short)
}

func TestDNSSetupCommand(t *testing.T) {
	cmd := newDNSSetupCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "setup [domain]", cmd.Use)
	assert.Equal(t, "Alias for 'dns create' command", cmd.Short)
}

func TestDNSCheckCommand(t *testing.T) {
	cmd := newDNSCheckCommand()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "check")
	assert.Contains(t, cmd.Short, "Check DNS")
}

func TestDomainAddCommand(t *testing.T) {
	cmd := newDomainAddCommand()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "add")
	assert.Contains(t, cmd.Short, "Add")
}

func TestDomainRemoveCommand(t *testing.T) {
	cmd := newDomainRemoveCommand()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "remove")
	assert.Contains(t, cmd.Short, "Remove")
}

func TestDomainListCommand(t *testing.T) {
	cmd := newDomainListCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List all configured domains", cmd.Short)
}

func TestDomainTestCommand(t *testing.T) {
	cmd := newDomainTestCommand()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "test")
	assert.Contains(t, cmd.Short, "Test")
}

func TestSSLSetupCommand(t *testing.T) {
	cmd := newSSLSetupCommand()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "setup")
	assert.Contains(t, cmd.Short, "Setup")
}

func TestSSLRenewCommand(t *testing.T) {
	cmd := newSSLRenewCommand()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "renew")
	assert.Contains(t, cmd.Short, "Renew")
}

func TestSSLStatusCommand(t *testing.T) {
	cmd := newSSLStatusCommand()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "status")
	assert.Contains(t, cmd.Short, "status")
}

func TestCommandStructure(t *testing.T) {
	// Test that all commands can be created without panic
	commands := []func() *cobra.Command{
		NewConfigCommand,
		NewDNSCommand,
		NewDomainCommand,
		NewSSLCommand,
		NewTestCommand,
		NewInstallCommand,
		NewServerCommand,
	}

	for _, cmdFunc := range commands {
		assert.NotPanics(t, func() {
			cmd := cmdFunc()
			assert.NotNil(t, cmd)
		})
	}
}
