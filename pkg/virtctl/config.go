package virtctl

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type Config struct {
	VNCCommand string `json:"vncCommand"`
	SSHCommand string `json:"sshCommand"`
	DefaultNS  string `json:"defaultNS"`
}

var configFilePath = os.Getenv("HOME") + "/.virtctl"

func readConfigFile() (Config, error) {
	config := Config{}
	configFile, err := os.Open(configFilePath)
	if err != nil {
		return config, err
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func writeConfigFile(config Config) error {
	configFile, err := os.Create(configFilePath)
	if err != nil {
		return err
	}
	defer configFile.Close()

	jsonEncoder := json.NewEncoder(configFile)
	err = jsonEncoder.Encode(config)
	if err != nil {
		return err
	}

	return nil
}

func configCmd() error {
	config, err := readConfigFile()
	if err != nil {
		// Handle error
	}

	fmt.Println("Please enter your preferences:")
	fmt.Print("VNC command (e.g. 'vncviewer'): ")
	fmt.Scanln(&config.VNCCommand)
	fmt.Print("SSH command (e.g. 'ssh'): ")
	fmt.Scanln(&config.SSHCommand)
	fmt.Print("Default namespace (e.g. 'default'): ")
	fmt.Scanln(&config.DefaultNS)

	err = writeConfigFile(config)
	if err != nil {
		// Handle error
	}

	fmt.Println("Preferences saved!")

	return nil
}

func init() {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Set default commands and namespaces for virtctl",
		Long:  "Set default commands and namespaces for virtctl",
		RunE: func(cmd *cobra.Command, args []string) error {
			return configCmd()
		},
	}

	virtctlCmd.AddCommand(configCmd)
}

var virtctlCmd = &cobra.Command{
	Use:   "virtctl",
	Short: "A command line tool for interacting with virtual machines",
	Long:  "virtctl is a command line tool that provides a convenient way to interact with virtual machines running on a Kubernetes cluster.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle virtctl command
		return nil
	},
}
