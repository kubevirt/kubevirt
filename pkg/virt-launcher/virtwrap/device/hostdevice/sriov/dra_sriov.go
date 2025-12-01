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
 * Copyright The KubeVirt Authors.
 *
 */

package sriov

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/deviceinfo"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

const (
	DRASRIOVResultsPath = "/etc/dra-sriov/cni-results.json"
)

// CreateDRASRIOVHostDevices reads PCI addresses from DRA SR-IOV results file and creates host devices
func CreateDRASRIOVHostDevices() ([]api.HostDevice, error) {
	return CreateDRASRIOVHostDevicesFromPath(DRASRIOVResultsPath)
}

// CreateDRASRIOVHostDevicesFromPath reads PCI addresses from specified path and creates host devices
// The file format is: {"<claimName>/<requestName>": ["<pciAddress>", ...], ...}
func CreateDRASRIOVHostDevicesFromPath(path string) ([]api.HostDevice, error) {
	// TODO once we fix the file, need to filter absent (and compare to whatever we are doing already)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Log.Infof("DRA SR-IOV results file not found at %s - not using DRA SR-IOV", path)
			return nil, fmt.Errorf("DRA SR-IOV results file not found at %s", path)
		}
		return nil, fmt.Errorf("failed to read DRA SR-IOV results file: %v", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("DRA SR-IOV results file exists but is empty at %s", path)
	}

	// Parse the file format: {"<claimName>/<requestName>": ["<pciAddress>", ...], ...}
	var results map[string][]string
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("failed to parse DRA SR-IOV results: %v", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("DRA SR-IOV results file has no entries")
	}

	var hostDevices []api.HostDevice
	deviceIndex := 0

	for claimKey, pciAddresses := range results {
		for _, pciAddress := range pciAddresses {
			if pciAddress == "" {
				continue
			}

			hostAddr, err := device.NewPciAddressField(pciAddress)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PCI address %s for %s: %v", pciAddress, claimKey, err)
			}

			// Use the same alias prefix as regular SR-IOV devices
			hostDevice := api.HostDevice{
				Alias:   api.NewUserDefinedAlias(deviceinfo.SRIOVAliasPrefix + fmt.Sprintf("dra-net%d", deviceIndex)),
				Source:  api.HostDeviceSource{Address: hostAddr},
				Type:    api.HostDevicePCI,
				Managed: "no",
			}

			log.Log.Infof("Created DRA SR-IOV host device with PCI address %s for %s", pciAddress, claimKey)
			hostDevices = append(hostDevices, hostDevice)
			deviceIndex++
		}
	}

	if len(hostDevices) == 0 {
		return nil, fmt.Errorf("DRA SR-IOV results file has no valid PCI addresses")
	}

	return hostDevices, nil
}
