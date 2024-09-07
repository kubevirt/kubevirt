package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"kubevirt.io/client-go/log"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
    log.InitializeLogging("config")

    cmd := &cobra.Command{
        Use:   "config SUBCOMMAND",
        Short: "Manage kubevirt configuration",
        Long: `The 'config' command allows you to manage configuration options for kubevirt, such as specifying paths for SSH binaries or VNC clients. 
This command supports setting, retrieving, and resetting configurations, which will be stored in a local configuration file.

    Usage Examples:

    # Set the path to the ssh binary
    virtctl config set ssh /usr/bin/ssh

    # Get the configured ssh path
    virtctl config get ssh

    # Reset all configurations
    virtctl config reset
        `,
    }

    // Add subcommands here
    cmd.AddCommand(newSetCommand())
    cmd.AddCommand(newGetCommand())
    cmd.AddCommand(newResetCommand())

    return cmd
}


func newSetCommand() *cobra.Command {
    var key, value string
    cmd := &cobra.Command{
        Use:   "set <key> <value>",
        Short: "Set a configuration option",
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            key = args[0]
            value = args[1]
            err := setConfigOption(key, value)
            if err != nil {
                return err
            }
            fmt.Printf("Configuration %s set to %s\n", key, value)
            return nil
        },
    }
    return cmd
}

func newGetCommand() *cobra.Command {
    var key string
    cmd := &cobra.Command{
        Use:   "get <key>",
        Short: "Get a configuration option",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            key = args[0]
            value, err := getConfigOption(key)
            if err != nil {
                return err
            }
            fmt.Printf("Configuration %s: %s\n", key, value)
            return nil
        },
    }
    return cmd
}

func newResetCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "reset",
        Short: "Reset all configuration options to default",
        RunE: func(cmd *cobra.Command, args []string) error {
            err := resetConfig()
            if err != nil {
                return err
            }
            fmt.Println("Configuration reset to defaults")
            return nil
        },
    }
    return cmd
}

func setConfigOption(key, value string) error {
    configFilePath := filepath.Join(os.Getenv("HOME"), ".virtctl", "config")
    config := loadConfig(configFilePath)
    config[key] = value
    return saveConfig(configFilePath, config)
}

func getConfigOption(key string) (string, error) {
    configFilePath := filepath.Join(os.Getenv("HOME"), ".virtctl", "config")
    config := loadConfig(configFilePath)
    if value, exists := config[key]; exists {
        return value, nil
    }
    return "", fmt.Errorf("configuration for %s not found", key)
}

func resetConfig() error {
    configFilePath := filepath.Join(os.Getenv("HOME"), ".virtctl", "config")
    return os.Remove(configFilePath)
}

func loadConfig(filePath string) map[string]string {
    config := make(map[string]string)
    file, err := os.Open(filePath)
    if err != nil {
        return config
    }
    defer file.Close()
    // Load config (assuming JSON format for simplicity)
    json.NewDecoder(file).Decode(&config)
    return config
}

func saveConfig(filePath string, config map[string]string) error {
    os.MkdirAll(filepath.Dir(filePath), 0755)
    file, err := os.Create(filePath)
    if err != nil {
        return err
    }
    defer file.Close()
    return json.NewEncoder(file).Encode(config)
}


