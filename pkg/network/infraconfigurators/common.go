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

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

package infraconfigurators

import (
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	v1 "kubevirt.io/api/core/v1"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

type PodNetworkInfraConfigurator interface {
	DiscoverPodNetworkInterface(podIfaceName string) error
	PreparePodNetworkInterface() error
	GenerateNonRecoverableDomainIfaceSpec() *api.Interface
	// The method should return dhcp configuration that cannot be calculated in virt-launcher's phase2
	GenerateNonRecoverableDHCPConfig() *cache.DHCPConfig
}

func createAndBindTapToBridge(handler netdriver.NetworkHandler, deviceName string, bridgeIfaceName string, launcherPID int, mtu int, tapOwner string, vmi *v1.VirtualMachineInstance) error {
	err := handler.CreateTapDevice(deviceName, calculateNetworkQueues(vmi), launcherPID, mtu, tapOwner)
	if err != nil {
		return err
	}
	return handler.BindTapDeviceToBridge(deviceName, bridgeIfaceName)
}

func calculateNetworkQueues(vmi *v1.VirtualMachineInstance) uint32 {
	if isMultiqueue(vmi) {
		return converter.CalculateNetworkQueues(vmi)
	}
	return 0
}

func isMultiqueue(vmi *v1.VirtualMachineInstance) bool {
	return (vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue != nil) &&
		(*vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue)
}
