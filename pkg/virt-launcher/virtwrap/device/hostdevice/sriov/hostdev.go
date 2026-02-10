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
	"kubevirt.io/kubevirt/pkg/network/deviceinfo"
	"kubevirt.io/kubevirt/pkg/network/downwardapi"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
)

func CreateHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	var hostDevices []api.HostDevice

	// Handle Multus-based SR-IOV
	multusHostDevs, err := createMultusSRIOVHostDevices(vmi)
	if err != nil {
		return nil, fmt.Errorf("failed to create Multus SR-IOV host devices: %v", err)
	}
	hostDevices = append(hostDevices, multusHostDevs...)

	// Handle DRA-based SR-IOV
	draHostDevs, err := createDRASRIOVHostDevices(vmi)
	if err != nil {
		return nil, fmt.Errorf("failed to create DRA SR-IOV host devices: %v", err)
	}
	hostDevices = append(hostDevices, draHostDevs...)

	return hostDevices, nil
}

// createMultusSRIOVHostDevices creates SR-IOV host devices for Multus-based networks
func createMultusSRIOVHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	// Filter Multus-based SR-IOV interfaces
	multusInterfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		if iface.SRIOV == nil {
			return false
		}
		network := findNetworkByName(vmi, iface.Name)
		if network == nil || network.Multus == nil {
			return false
		}
		// Only process Multus interfaces that have InfoSourceMultusStatus
		ifaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, iface.Name)
		return ifaceStatus != nil && vmispec.ContainsInfoSource(ifaceStatus.InfoSource, vmispec.InfoSourceMultusStatus)
	})

	if len(multusInterfaces) == 0 {
		return []api.HostDevice{}, nil
	}

	netStatusPath := path.Join(downwardapi.MountPath, downwardapi.NetworkInfoVolumePath)
	pciAddressPoolWithNetworkStatus, err := newPCIAddressPoolWithNetworkStatusFromFile(netStatusPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Multus SR-IOV hostdevices: %v", err)
	}
	if pciAddressPoolWithNetworkStatus.Len() == 0 {
		log.Log.Object(vmi).Warningf("found no Multus SR-IOV networks to PCI-Address mapping.")
		return nil, fmt.Errorf("found no Multus SR-IOV networks to PCI-Address mapping")
	}

	return CreateHostDevicesFromIfacesAndPool(multusInterfaces, pciAddressPoolWithNetworkStatus)
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

func SafelyDetachHostDevices(domainSpec *api.DomainSpec, eventDetach hostdevice.EventRegistrar, dom hostdevice.DeviceDetacher, timeout time.Duration) error {
	sriovDevices := hostdevice.FilterHostDevicesByAlias(domainSpec.Devices.HostDevices, deviceinfo.SRIOVAliasPrefix)
	return hostdevice.SafelyDetachHostDevices(sriovDevices, eventDetach, dom, timeout)
}

func GetHostDevicesToAttach(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec) ([]api.HostDevice, error) {
	sriovDevices, err := CreateHostDevices(vmi)
	if err != nil {
		return nil, err
	}
	currentAttachedSRIOVHostDevices := hostdevice.FilterHostDevicesByAlias(domainSpec.Devices.HostDevices, deviceinfo.SRIOVAliasPrefix)

	sriovHostDevicesToAttach := hostdevice.DifferenceHostDevicesByAlias(sriovDevices, currentAttachedSRIOVHostDevices)

	return sriovHostDevicesToAttach, nil
}

// createDRASRIOVHostDevices creates host devices for DRA-based SR-IOV networks
func createDRASRIOVHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	// Filter DRA-based SR-IOV interfaces
	draInterfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		if iface.SRIOV == nil {
			return false
		}
		network := findNetworkByName(vmi, iface.Name)
		return network != nil && network.ResourceClaim != nil
	})

	if len(draInterfaces) == 0 {
		return []api.HostDevice{}, nil
	}

	var hostDevices []api.HostDevice

	for _, iface := range draInterfaces {
		network := findNetworkByName(vmi, iface.Name)
		if network == nil || network.ResourceClaim == nil {
			continue
		}

		// Find device status entry by network name
		deviceStatus := findDeviceStatusByName(vmi, network.Name)
		if deviceStatus == nil {
			return nil, fmt.Errorf("device status not found for DRA network %s", network.Name)
		}

		if deviceStatus.DeviceResourceClaimStatus == nil {
			return nil, fmt.Errorf("device resource claim status not populated for network %s", network.Name)
		}

		if deviceStatus.DeviceResourceClaimStatus.Attributes == nil {
			return nil, fmt.Errorf("device attributes not populated for network %s", network.Name)
		}

		// Extract PCI address
		pciAddress := deviceStatus.DeviceResourceClaimStatus.Attributes.PCIAddress
		if pciAddress == nil || *pciAddress == "" {
			return nil, fmt.Errorf("PCI address not found for DRA network %s", network.Name)
		}

		// Parse PCI address (format: 0000:05:00.1)
		address, err := parsePCIAddress(*pciAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid PCI address %s for network %s: %v", *pciAddress, network.Name, err)
		}

		// Create hostdev with DRA-specific alias
		hostDev := api.HostDevice{
			Alias: api.NewUserDefinedAlias(deviceinfo.DraSRIOVAliasPrefix + iface.Name),
			Source: api.HostDeviceSource{
				Address: &api.Address{
					Type:     "pci",
					Domain:   address.Domain,
					Bus:      address.Bus,
					Slot:     address.Slot,
					Function: address.Function,
				},
			},
			Type:    "pci",
			Managed: "no",
		}

		// Apply additional decorations (boot order, guest PCI address)
		decorateHook := newDecorateHook(iface)
		if err := decorateHook(&hostDev); err != nil {
			return nil, fmt.Errorf("failed to decorate DRA SR-IOV host device for %s: %v", iface.Name, err)
		}

		hostDevices = append(hostDevices, hostDev)
	}

	return hostDevices, nil
}

// Helper functions

func findNetworkByName(vmi *v1.VirtualMachineInstance, name string) *v1.Network {
	for i := range vmi.Spec.Networks {
		if vmi.Spec.Networks[i].Name == name {
			return &vmi.Spec.Networks[i]
		}
	}
	return nil
}

func findDeviceStatusByName(vmi *v1.VirtualMachineInstance, name string) *v1.DeviceStatusInfo {
	if vmi.Status.DeviceStatus == nil {
		return nil
	}

	for i := range vmi.Status.DeviceStatus.HostDeviceStatuses {
		if vmi.Status.DeviceStatus.HostDeviceStatuses[i].Name == name {
			return &vmi.Status.DeviceStatus.HostDeviceStatuses[i]
		}
	}
	return nil
}

// PCIAddress represents parsed PCI address components
type PCIAddress struct {
	Domain   string
	Bus      string
	Slot     string
	Function string
}

// parsePCIAddress parses PCI address string (format: 0000:05:00.1)
func parsePCIAddress(addr string) (*PCIAddress, error) {
	// Split by colon
	parts := []string{}
	current := ""
	for _, char := range addr {
		if char == ':' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	parts = append(parts, current)

	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid PCI address format: %s (expected format: 0000:05:00.1)", addr)
	}

	// Split last part by dot
	slotFunc := []string{}
	current = ""
	for _, char := range parts[2] {
		if char == '.' {
			slotFunc = append(slotFunc, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	slotFunc = append(slotFunc, current)

	if len(slotFunc) != 2 {
		return nil, fmt.Errorf("invalid PCI address format: %s (expected slot.function)", addr)
	}

	return &PCIAddress{
		Domain:   "0x" + parts[0],
		Bus:      "0x" + parts[1],
		Slot:     "0x" + slotFunc[0],
		Function: "0x" + slotFunc[1],
	}, nil
}
