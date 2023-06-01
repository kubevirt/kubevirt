package tests

import (
	"flag"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

const (
	ConfigFileFlag = "config-file"
)

var (
	configFileName string
	config         *TestConfig
)

type QuickStartTestItem struct {
	Name        string `yaml:"name,omitempty"`
	DisplayName string `yaml:"displayName,omitempty"`
}

type QuickStartTestConfig struct {
	TestItems []QuickStartTestItem `yaml:"testItems,omitempty"`
}

type DashboardTestItem struct {
	Name      string `yaml:"name,omitempty"`
	Namespace string `yaml:"namespace,omitempty"`
	// keys expected in the configmap
	Keys []string `yaml:"keys,omitempty"`
}

type DashboardTestConfig struct {
	TestItems []DashboardTestItem `yaml:"testItems,omitempty"`
}

type TestConfig struct {
	QuickStart QuickStartTestConfig `yaml:"quickStart,omitempty"`
	Dashboard  DashboardTestConfig  `yaml:"dashboard,omitempty"`
}

func init() {
	flag.StringVar(&configFileName, ConfigFileFlag, "", "File contains test configuration")
}

func GetConfig() *TestConfig {
	once := sync.Once{}
	once.Do(func() {
		config = loadConfig(configFileName)
	})

	return config
}

func loadConfig(fileName string) *TestConfig {
	config := TestConfig{}

	if fileName != "" {
		file, err := os.Open(fileName)
		if err != nil {
			panic(err)
		}
		dec := yaml.NewDecoder(file)
		err = dec.Decode(&config)
		if err != nil {
			panic(err)
		}
	}

	return &config
}
