package dra

import (
	"fmt"
	"strconv"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
)

func getDRAUSBHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, []api.HostDevice, error) {
	if vmi.Status.DeviceStatus == nil {
		return nil, nil, fmt.Errorf("vmi has dra usb devices but no device status found")
	}

	var (
		hostDevices             []api.HostDevice
		hotpluggableHostDevices []api.HostDevice
	)

	hotpluggable := make(map[string]struct{})
	for _, resourceClaim := range vmi.Spec.ResourceClaims {
		if resourceClaim.Hotpluggable {
			hotpluggable[resourceClaim.Name] = struct{}{}
		}
	}

	for _, hdStatus := range vmi.Status.DeviceStatus.HostDeviceStatuses {
		hdStatus := hdStatus.DeepCopy()
		if hdStatus.DeviceResourceClaimStatus != nil && hdStatus.DeviceResourceClaimStatus.Attributes != nil {
			if usbAddress := hdStatus.DeviceResourceClaimStatus.Attributes.USBAddress; usbAddress != nil {

				// handle only hotpluggable devices in spec
				if _, ok := hotpluggable[hdStatus.Name]; ok && hdStatus.Hotplug != nil {
					hostDevice := newUSBHostDevice(usbAddress, hdStatus.Name, true)
					hotpluggableHostDevices = append(hotpluggableHostDevices, hostDevice)
					continue
				}

				// skip hotplug devices which are not in spec
				if hdStatus.Hotplug != nil {
					continue
				}

				hostDevice := newUSBHostDevice(usbAddress, hdStatus.Name, false)
				hostDevices = append(hostDevices, hostDevice)
			}
		}
	}

	return hostDevices, hotpluggableHostDevices, nil
}

func newUSBHostDevice(usbAddress *v1.USBAddress, name string, hotplug bool) api.HostDevice {
	var alias *api.Alias
	startupPolicy := "required"
	if hotplug {
		alias = api.NewUserDefinedAlias(DRAHotplugHostDeviceAliasPrefix + name)
		startupPolicy = "optional"
	} else {
		alias = api.NewUserDefinedAlias(DRAHostDeviceAliasPrefix + name)
	}
	return api.HostDevice{
		Type:  api.HostDeviceUSB,
		Mode:  "subsystem",
		Alias: alias,
		Source: api.HostDeviceSource{
			Address: &api.Address{
				Bus:    strconv.FormatInt(usbAddress.Bus, 10),
				Device: strconv.FormatInt(usbAddress.DeviceNumber, 10),
			},
			StartupPolicy: startupPolicy,
		},
	}
}

func GetDRAUSBHostDevicesToAttach(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec) ([]api.HostDevice, error) {
	_, hotpluggableHostDevices, err := getDRAUSBHostDevices(vmi)
	if err != nil {
		return nil, err
	}
	currentAttachedUSBHostDevices := FilterUSBHostDevicesByAlias(domainSpec.Devices.HostDevices, true)

	usbHostDevicesToAttach := hostdevice.DifferenceHostDevicesByAlias(hotpluggableHostDevices, currentAttachedUSBHostDevices)

	return usbHostDevicesToAttach, nil
}

func GetDRAUSBHostDevicesToDetach(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec) ([]api.HostDevice, error) {
	_, hotpluggableHostDevices, err := getDRAUSBHostDevices(vmi)
	if err != nil {
		return nil, err
	}
	currentAttachedUSBHostDevices := FilterUSBHostDevicesByAlias(domainSpec.Devices.HostDevices, true)

	usbHostDevicesToDetach := hostdevice.DifferenceHostDevicesByAlias(currentAttachedUSBHostDevices, hotpluggableHostDevices)

	return usbHostDevicesToDetach, nil
}

func FilterUSBHostDevicesByAlias(hostDevices []api.HostDevice, hotplug bool) []api.HostDevice {
	prefix := DRAHostDeviceAliasPrefix
	if hotplug {
		prefix = DRAHotplugHostDeviceAliasPrefix
	}

	var usbDevices []api.HostDevice
	for _, device := range hostdevice.FilterHostDevicesByAlias(hostDevices, prefix) {
		if device.Type == api.HostDeviceUSB {
			usbDevices = append(usbDevices, device)
		}
	}

	return usbDevices
}
