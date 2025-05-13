package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stacklok/toolhive/pkg/client"
	"github.com/stacklok/toolhive/pkg/config"
	"github.com/stacklok/toolhive/pkg/container"
	rt "github.com/stacklok/toolhive/pkg/container/runtime"
	"github.com/stacklok/toolhive/pkg/labels"
	"github.com/stacklok/toolhive/pkg/logger"
	"github.com/stacklok/toolhive/pkg/secrets"
	"github.com/stacklok/toolhive/pkg/transport"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage application configuration",
	Long:  "The config command provides subcommands to manage application configuration settings.",
}

var listRegisteredClientsCmd = &cobra.Command{
	Use:   "list-registered-clients",
	Short: "List all registered MCP clients",
	Long:  "List all clients that are registered for MCP server configuration.",
	RunE:  listRegisteredClientsCmdFunc,
}

var secretsProviderCmd = &cobra.Command{
	Use:   "secrets-provider [provider]",
	Short: "Set the secrets provider type",
	Long: `Set the secrets provider type for storing and retrieving secrets.
Valid providers are:
  - encrypted: Stores secrets in an encrypted file using AES-256-GCM`,
	Args: cobra.ExactArgs(1),
	RunE: secretsProviderCmdFunc,
}

var autoDiscoveryCmd = &cobra.Command{
	Use:   "auto-discovery [true|false]",
	Short: "Set whether to enable auto-discovery of MCP clients",
	Long: `Set whether to enable auto-discovery and configuration of MCP clients.
When enabled, ToolHive will automatically update client configuration files
with the URLs of running MCP servers.`,
	Args: cobra.ExactArgs(1),
	RunE: autoDiscoveryCmdFunc,
}

var registerClientCmd = &cobra.Command{
	Use:   "register-client [client]",
	Short: "Register a client for MCP server configuration",
	Long: `Register a client for MCP server configuration.
Valid clients are:
  - roo-code: Roo Code extension for VS Code
  - cursor: Cursor editor
  - claude-code: Claude Code CLI
  - vscode: Visual Studio Code
  - vscode-insider: Visual Studio Code Insiders edition`,
	Args: cobra.ExactArgs(1),
	RunE: registerClientCmdFunc,
}

var removeClientCmd = &cobra.Command{
	Use:   "remove-client [client]",
	Short: "Remove a client from MCP server configuration",
	Long: `Remove a client from MCP server configuration.
Valid clients are:
  - roo-code: Roo Code extension for VS Code
  - cursor: Cursor editor
  - claude-code: Claude Code CLI
  - vscode: Visual Studio Code
  - vscode-insider: Visual Studio Code Insiders edition`,
	Args: cobra.ExactArgs(1),
	RunE: removeClientCmdFunc,
}

func init() {
	// Add config command to root command
	rootCmd.AddCommand(configCmd)

	// Add subcommands to config command
	configCmd.AddCommand(secretsProviderCmd)
	configCmd.AddCommand(autoDiscoveryCmd)
	configCmd.AddCommand(registerClientCmd)
	configCmd.AddCommand(removeClientCmd)
	configCmd.AddCommand(listRegisteredClientsCmd)
}

func secretsProviderCmdFunc(_ *cobra.Command, args []string) error {
	provider := args[0]
	return SetSecretsProvider(secrets.ProviderType(provider))
}

func autoDiscoveryCmdFunc(cmd *cobra.Command, args []string) error {
	value := args[0]

	// Validate the boolean value
	var enabled bool
	switch value {
	case "true", "1", "yes":
		enabled = true
	case "false", "0", "no":
		enabled = false
	default:
		return fmt.Errorf("invalid boolean value: %s (valid values: true, false)", value)
	}

	// Update the auto-discovery setting
	err := config.UpdateConfig(func(c *config.Config) {
		c.Clients.AutoDiscovery = enabled
		// If auto-discovery is enabled, update all registered clients with currently running MCPs
		if enabled && len(c.Clients.RegisteredClients) > 0 {
			for _, clientName := range c.Clients.RegisteredClients {
				if err := addRunningMCPsToClient(cmd.Context(), clientName); err != nil {
					fmt.Printf("Warning: Failed to add running MCPs to client %s: %v\n", clientName, err)
				}
			}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	fmt.Printf("Auto-discovery of MCP clients %s\n", map[bool]string{true: "enabled", false: "disabled"}[enabled])

	return nil
}

func registerClientCmdFunc(cmd *cobra.Command, args []string) error {
	clientType := args[0]

	// Validate the client type
	switch clientType {
	case "roo-code", "cursor", "claude-code", "vscode-insider", "vscode":
		// Valid client type
	default:
		return fmt.Errorf("invalid client type: %s (valid types: roo-code, cursor, claude-code, vscode, vscode-insider)", clientType)
	}

	err := config.UpdateConfig(func(c *config.Config) {
		// Check if client is already registered and skip.
		for _, registeredClient := range c.Clients.RegisteredClients {
			if registeredClient == clientType {
				fmt.Printf("Client %s is already registered, skipping...\n", clientType)
				return
			}
		}

		// Add the client to the registered clients list
		c.Clients.RegisteredClients = append(c.Clients.RegisteredClients, clientType)
	})
	if err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	fmt.Printf("Successfully registered client: %s\n", clientType)

	// Add currently running MCPs to the newly registered client
	if err := addRunningMCPsToClient(cmd.Context(), clientType); err != nil {
		fmt.Printf("Warning: Failed to add running MCPs to client: %v\n", err)
	}

	return nil
}

func removeClientCmdFunc(_ *cobra.Command, args []string) error {
	clientType := args[0]

	// Validate the client type
	switch clientType {
	case "roo-code", "cursor", "claude-code", "vscode-insider", "vscode":
		// Valid client type
	default:
		return fmt.Errorf("invalid client type: %s (valid types: roo-code, cursor, claude-code, vscode, vscode-insider)", clientType)
	}

	err := config.UpdateConfig(func(c *config.Config) {
		// Find and remove the client from the registered clients list
		found := false
		for i, registeredClient := range c.Clients.RegisteredClients {
			if registeredClient == clientType {
				// Remove the client by appending the slice before and after the index
				c.Clients.RegisteredClients = append(c.Clients.RegisteredClients[:i], c.Clients.RegisteredClients[i+1:]...)
				found = true
				break
			}
		}
		if found {
			fmt.Printf("Client %s removed from registered clients.\n", clientType)
		} else {
			fmt.Printf("Client %s not found in registered clients.\n", clientType)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	fmt.Printf("Successfully removed client: %s\n", clientType)
	return nil
}

// addRunningMCPsToClient adds currently running MCP servers to the specified client's configuration
func addRunningMCPsToClient(ctx context.Context, clientName string) error {
	// Create container runtime
	runtime, err := container.NewFactory().Create(ctx)
	if err != nil {
		return fmt.Errorf("failed to create container runtime: %v", err)
	}

	// List containers
	containers, err := runtime.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %v", err)
	}

	// Filter containers to only show those managed by ToolHive and running
	var runningContainers []rt.ContainerInfo
	for _, c := range containers {
		if labels.IsToolHiveContainer(c.Labels) && c.State == "running" {
			runningContainers = append(runningContainers, c)
		}
	}

	if len(runningContainers) == 0 {
		// No running servers, nothing to do
		return nil
	}

	// Find the client configuration for the specified client
	clientConfigs, err := client.FindClientConfigs()
	if err != nil {
		return fmt.Errorf("failed to find client configurations: %w", err)
	}

	// If no configs found, nothing to do
	if len(clientConfigs) == 0 {
		return nil
	}

	// For each running container, add it to the client configuration
	for _, c := range runningContainers {
		// Get container name from labels
		name := labels.GetContainerName(c.Labels)
		if name == "" {
			name = c.Name // Fallback to container name
		}

		// Get tool type from labels
		toolType := labels.GetToolType(c.Labels)

		// Only include containers with tool type "mcp"
		if toolType != "mcp" {
			continue
		}

		// Get port from labels
		port, err := labels.GetPort(c.Labels)
		if err != nil {
			continue // Skip if we can't get the port
		}

		// Generate URL for the MCP server
		url := client.GenerateMCPServerURL(transport.LocalhostIPv4, port, name)

		// Update each configuration file
		for _, clientConfig := range clientConfigs {
			// Update the MCP server configuration with locking
			if err := client.Upsert(clientConfig, name, url); err != nil {
				logger.Warnf("Warning: Failed to update MCP server configuration in %s: %v", clientConfig.Path, err)
				continue
			}

			fmt.Printf("Added MCP server %s to client %s\n", name, clientName)
		}
	}

	return nil
}

func listRegisteredClientsCmdFunc(_ *cobra.Command, _ []string) error {
	// Get the current config
	cfg := config.GetConfig()

	// Check if there are any registered clients
	if len(cfg.Clients.RegisteredClients) == 0 {
		fmt.Println("No clients are currently registered.")
		return nil
	}

	// Print the list of registered clients
	fmt.Println("Registered clients:")
	for _, clientName := range cfg.Clients.RegisteredClients {
		fmt.Printf("  - %s\n", clientName)
	}

	return nil
}
