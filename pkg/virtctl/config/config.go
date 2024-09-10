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

func NewConfigCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	log.InitializeLogging("config")

	cmd := &cobra.Command{
		Use:   "config <action> [key] [value]",
		Short: "Manage kubevirt configuration",
		Long: `The 'config' command allows you to manage configuration options for kubevirt, such as specifying paths for SSH binaries or VNC clients.
Supported actions are:
- set <key> <value>: Set a configuration option
- get <key>: Retrieve the value of a configuration option

Usage Examples:
# Set the path to the ssh binary
virtctl config set ssh /usr/bin/ssh
# Get the configured ssh path
virtctl config get ssh
		`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			action := args[0]

			switch action {
			case "set":
				if len(args) != 3 {
					return fmt.Errorf("invalid number of arguments for 'set'. Usage: config set <key> <value>")
				}
				key, value := args[1], args[2]
				return setConfigOption(key, value)

			case "get":
				if len(args) != 2 {
					return fmt.Errorf("invalid number of arguments for 'get'. Usage: config get <key>")
				}
				key := args[1]
				value, err := getConfigOption(key)
				if err != nil {
					return err
				}
				fmt.Printf("Configuration %s: %s\n", key, value)
				return nil

			default:
				return fmt.Errorf("invalid action: %s. Supported actions: set, get", action)
			}
		},
	}

	return cmd
}

func setConfigOption(key, value string) error {
	configFilePath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config := loadConfig(configFilePath)
	config[key] = value
	err := saveConfig(configFilePath, config)
	if err != nil {
		return err
	}
	fmt.Printf("Configuration %s set to %s\n", key, value)
	return nil
}

func getConfigOption(key string) (string, error) {
	configFilePath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config := loadConfig(configFilePath)
	if value, exists := config[key]; exists {
		return value, nil
	}
	return "", fmt.Errorf("configuration for %s not found", key)
}

func loadConfig(filePath string) map[string]string {
	config := make(map[string]string)
	file, err := os.Open(filePath)
	if err != nil {
		return config
	}
	defer file.Close()
	json.NewDecoder(file).Decode(&config)
	return config
}

func saveConfig(filePath string, config map[string]string) error {
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(config)
}
