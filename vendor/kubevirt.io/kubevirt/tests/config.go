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
	"io/ioutil"
	"os"

	"kubevirt.io/kubevirt/tests/flags"
)

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
	// Flag if true the storageclasses are managed, false otherwise
	ManageStorageClasses bool `json:"manageStorageClasses"`
}

func loadConfig() (*KubeVirtTestsConfiguration, error) {
	// open configuration file
	jsonFile, err := os.Open(flags.ConfigFile)
	if err != nil {
		return nil, err
	}

	defer jsonFile.Close()

	// read the configuration file as a byte array
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// convert the byte array to a KubeVirtTestsConfiguration struct
	config := &KubeVirtTestsConfiguration{}
	err = json.Unmarshal(byteValue, config)

	return config, err
}
