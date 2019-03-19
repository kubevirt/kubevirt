/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2019 Red Hat, Inc.
 *
 */

package tests

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

const (
	DefaultConfigFile string = "tests/default-config.json"
)

var ConfigFile = ""
var Config *KubeVirtTestsConfiguration

func init() {
	flag.StringVar(&ConfigFile, "config", "", "Path to a JSON formatted file from which the test suite will load its configuration. The path may be absolute or relative; relative paths start at the current working directory.")
}

// KubeVirtTestsConfiguration contains the configuration for KubeVirt tests
type KubeVirtTestsConfiguration struct {
	// StorageClass to use to create local PVCs
	StorageClassLocal string `json:"storageClassLocal"`
	// StorageClass to use to create host-path PVCs
	StorageClassHostPath string `json:"storageClassHostPath"`
	// StorageClass to use to create block-volume PVCs
	StorageClassBlockVolume string `json:"storageClassBlockVolume"`
	// StorageClass to use to create rhel PVCs
	StorageClassRhel string `json:"storageClassRhel"`
	// StorageClass to use to create windows PVCs
	StorageClassWindows string `json:"storageClassWindows"`
}

// Returns a new KubeVirtTestsConfiguration with default values
func NewKubeVirtTestsConfiguration() *KubeVirtTestsConfiguration {
	config := &KubeVirtTestsConfiguration{}

	err := loadConfigFromFile(DefaultConfigFile, config)

	if err != nil {
		panic(fmt.Sprintf("Couldn't load default test suite configuration: %s\n", err))
	}

	return config
}

func loadConfig() *KubeVirtTestsConfiguration {
	config := NewKubeVirtTestsConfiguration()

	if ConfigFile != "" {
		err := loadConfigFromFile(ConfigFile, config)

		if err != nil {
			panic(fmt.Sprintf("Couldn't load test suite configuration file: %s\n", err))
		}
	}

	return config
}

func loadConfigFromFile(file string, config *KubeVirtTestsConfiguration) error {
	// open configuration file
	jsonFile, err := os.Open(file)
	if err != nil {
		return err
	}

	defer jsonFile.Close()

	// read the configuration file as a byte array
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// convert the byte array as a KubeVirtTestsConfiguration struct
	err = json.Unmarshal(byteValue, config)

	return err
}
