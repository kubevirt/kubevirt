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
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
	drautil "kubevirt.io/kubevirt/pkg/dra"
	"kubevirt.io/kubevirt/pkg/network/deviceinfo"
	"kubevirt.io/kubevirt/pkg/network/downwardapi"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
)

func CreateHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	SRIOVInterfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		if iface.SRIOV == nil {
			return false
		}
		ifaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, iface.Name)
		return ifaceStatus != nil && vmispec.ContainsInfoSource(ifaceStatus.InfoSource, vmispec.InfoSourceMultusStatus)
	})
	if len(SRIOVInterfaces) == 0 {
		return []api.HostDevice{}, nil
	}
	netStatusPath := path.Join(downwardapi.MountPath, downwardapi.NetworkInfoVolumePath)
	pciAddressPoolWithNetworkStatus, err := newPCIAddressPoolWithNetworkStatusFromFile(netStatusPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create SR-IOV hostdevices: %v", err)
	}
	if pciAddressPoolWithNetworkStatus.Len() == 0 {
		log.Log.Object(vmi).Warningf("found no SR-IOV networks to PCI-Address mapping.")
		return nil, fmt.Errorf("found no SR-IOV networks to PCI-Address mapping")
	}

	return CreateHostDevicesFromIfacesAndPool(SRIOVInterfaces, pciAddressPoolWithNetworkStatus)
}

// newPCIAddressPoolWithNetworkStatusFromFile polls the given file path until populated, then uses it to create the
// PCI-Address Pool.
// possible return values are:
// - file populated - return PCI-Address Pool using the data in file.
// - file empty post-polling (timeout) - return err to fail SyncVMI.
// - other error reading file (i.e. file not exist) - return no error but PCIAddressWithNetworkStatusPool.Len() will return 0.
func newPCIAddressPoolWithNetworkStatusFromFile(path string) (*PCIAddressWithNetworkStatusPool, error) {
	const failedCreatePciPoolFmt = "failed to create PCI address pool with network status from file: %w"

	networkDeviceInfoBytes, err := readFileUntilNotEmpty(path)
	if err != nil {
		if isFileEmptyAfterTimeout(err, networkDeviceInfoBytes) {
			return nil, fmt.Errorf(failedCreatePciPoolFmt, err)
		}
		return nil, nil
	}

	pciPool, err := NewPCIAddressPoolWithNetworkStatus(networkDeviceInfoBytes)
	if err != nil {
		return nil, fmt.Errorf(failedCreatePciPoolFmt, err)
	}
	return pciPool, nil
}

func readFileUntilNotEmpty(networkPCIMapPath string) ([]byte, error) {
	var networkPCIMapBytes []byte
	err := virtwait.PollImmediately(100*time.Millisecond, time.Second, func(_ context.Context) (bool, error) {
		var err error
		networkPCIMapBytes, err = os.ReadFile(networkPCIMapPath)
		return len(networkPCIMapBytes) > 0, err
	})
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("%w: file is not populated with network-info", err)
	}
	return networkPCIMapBytes, err
}

func isFileEmptyAfterTimeout(err error, data []byte) bool {
	return errors.Is(err, context.DeadlineExceeded) && len(data) == 0
}

func CreateHostDevicesFromIfacesAndPool(ifaces []v1.Interface, pool hostdevice.AddressPooler) ([]api.HostDevice, error) {
	hostDevicesMetaData := createHostDevicesMetadata(ifaces)
	return hostdevice.CreatePCIHostDevices(hostDevicesMetaData, pool)
}

func createHostDevicesMetadata(ifaces []v1.Interface) []hostdevice.HostDeviceMetaData {
	var hostDevicesMetaData []hostdevice.HostDeviceMetaData
	for _, iface := range ifaces {
		hostDevicesMetaData = append(hostDevicesMetaData, hostdevice.HostDeviceMetaData{
			AliasPrefix:  deviceinfo.SRIOVAliasPrefix,
			Name:         iface.Name,
			ResourceName: iface.Name,
			DecorateHook: newDecorateHook(iface),
		})
	}
	return hostDevicesMetaData
}

func newDecorateHook(iface v1.Interface) func(hostDevice *api.HostDevice) error {
	return func(hostDevice *api.HostDevice) error {
		if guestPCIAddress := iface.PciAddress; guestPCIAddress != "" {
			addr, err := device.NewPciAddressField(guestPCIAddress)
			if err != nil {
				return fmt.Errorf("failed to interpret the guest PCI address: %v", err)
			}
			hostDevice.Address = addr
		}

		if iface.BootOrder != nil {
			hostDevice.BootOrder = &api.BootOrder{Order: *iface.BootOrder}
		}
		return nil
	}
}

// CreateDRAHostDevices creates SR-IOV host devices for networks that use DRA (Dynamic Resource Allocation).
// Unlike traditional SR-IOV which gets PCI addresses from multus network status, DRA-based SR-IOV
// gets PCI addresses from the VMI device status populated by the DRA controller.
func CreateDRAHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	// Find interfaces with SRIOV binding that reference networks with resourceClaim
	sriovDRAInterfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		if iface.SRIOV == nil {
			return false
		}
		network := vmispec.LookupNetworkByName(vmi.Spec.Networks, iface.Name)
		return network != nil && drautil.IsNetworkDRA(*network)
	})

	if len(sriovDRAInterfaces) == 0 {
		return []api.HostDevice{}, nil
	}

	if vmi.Status.DeviceStatus == nil {
		return nil, fmt.Errorf("vmi has DRA SR-IOV interfaces but no device status found")
	}

	var hostDevices []api.HostDevice
	for _, iface := range sriovDRAInterfaces {
		// Find the corresponding device status by interface/network name
		var deviceStatus *v1.DeviceStatusInfo
		for i := range vmi.Status.DeviceStatus.HostDeviceStatuses {
			if vmi.Status.DeviceStatus.HostDeviceStatuses[i].Name == iface.Name {
				deviceStatus = &vmi.Status.DeviceStatus.HostDeviceStatuses[i]
				break
			}
		}

		if deviceStatus == nil {
			return nil, fmt.Errorf("no device status found for SR-IOV DRA interface %s", iface.Name)
		}

		if deviceStatus.DeviceResourceClaimStatus == nil ||
			deviceStatus.DeviceResourceClaimStatus.Attributes == nil ||
			deviceStatus.DeviceResourceClaimStatus.Attributes.PCIAddress == nil {
			return nil, fmt.Errorf("no PCI address found in device status for SR-IOV DRA interface %s", iface.Name)
		}

		hostAddr, err := device.NewPciAddressField(*deviceStatus.DeviceResourceClaimStatus.Attributes.PCIAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to create PCI address for SR-IOV DRA interface %s: %v", iface.Name, err)
		}

		hostDevice := api.HostDevice{
			Alias:   api.NewUserDefinedAlias(deviceinfo.SRIOVAliasPrefix + iface.Name),
			Source:  api.HostDeviceSource{Address: hostAddr},
			Type:    api.HostDevicePCI,
			Managed: "no",
		}

		// Add guest PCI address if specified
		if iface.PciAddress != "" {
			addr, err := device.NewPciAddressField(iface.PciAddress)
			if err != nil {
				return nil, fmt.Errorf("failed to interpret the guest PCI address for interface %s: %v", iface.Name, err)
			}
			hostDevice.Address = addr
		}

		// Add boot order if specified
		if iface.BootOrder != nil {
			hostDevice.BootOrder = &api.BootOrder{Order: *iface.BootOrder}
		}

		hostDevices = append(hostDevices, hostDevice)
	}

	return hostDevices, nil
}

func SafelyDetachHostDevices(domainSpec *api.DomainSpec, eventDetach hostdevice.EventRegistrar, dom hostdevice.DeviceDetacher, timeout time.Duration) error {
	sriovDevices := hostdevice.FilterHostDevicesByAlias(domainSpec.Devices.HostDevices, deviceinfo.SRIOVAliasPrefix)
	return hostdevice.SafelyDetachHostDevices(sriovDevices, eventDetach, dom, timeout)
}

func GetHostDevicesToAttach(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec) ([]api.HostDevice, error) {
	sriovDevices, err := CreateHostDevices(vmi)
	if err != nil {
		return nil, err
	}

	// Add DRA-based SR-IOV devices
	sriovDRADevices, err := CreateDRAHostDevices(vmi)
	if err != nil {
		return nil, err
	}
	sriovDevices = append(sriovDevices, sriovDRADevices...)

	currentAttachedSRIOVHostDevices := hostdevice.FilterHostDevicesByAlias(domainSpec.Devices.HostDevices, deviceinfo.SRIOVAliasPrefix)

	sriovHostDevicesToAttach := hostdevice.DifferenceHostDevicesByAlias(sriovDevices, currentAttachedSRIOVHostDevices)

	return sriovHostDevicesToAttach, nil
}
