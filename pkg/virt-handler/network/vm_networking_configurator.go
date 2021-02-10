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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package network

import (
	"encoding/json"
	"fmt"
	"os"

	networkdriver "kubevirt.io/kubevirt/pkg/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type VMNetworkingConfiguration interface {
	cacheInterface() error
	discoverPodNetworkInterface() error
	exportVIF() error
	loadCachedInterface() error
	prepareVMNetworkingInterfaces() error
}

func createAndBindTapToBridge(deviceName string, bridgeIfaceName string, queueNumber uint32, launcherPID int, mtu int) error {
	err := networkdriver.Handler.CreateTapDevice(deviceName, queueNumber, launcherPID, mtu)
	if err != nil {
		return err
	}
	return networkdriver.Handler.BindTapDeviceToBridge(deviceName, bridgeIfaceName)
}

func generateTapDeviceName(podInterfaceName string) string {
	return "tap" + podInterfaceName[3:]
}

func loadCachedInterface(launcherPID int, ifaceName string) (*api.Interface, error) {
	var ifaceConfig api.Interface
	err := networkdriver.ReadFromVirtLauncherCachedFile(&ifaceConfig, fmt.Sprintf("%d", launcherPID), ifaceName)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &ifaceConfig, nil
}

func setCachedVIF(vif networkdriver.VIF, pid int, ifaceName string) error {
	buf, err := json.MarshalIndent(vif, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling vif object: %v", err)
	}

	launcherPID := "self"
	if pid != 0 {
		launcherPID = fmt.Sprintf("%d", pid)
	}
	return networkdriver.WriteVifFile(buf, launcherPID, ifaceName)
}
