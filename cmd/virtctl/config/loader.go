package config

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "virtctl"), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "config.yaml"), nil
}

func SaveConfig(cfg *Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	// Ensure the config directory exists
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Convert struct → YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	// Write config file with secure permissions
	return os.WriteFile(path, data, 0600)
}

func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config at %s: %w", path, err)
	}

	return cfg, nil
}
