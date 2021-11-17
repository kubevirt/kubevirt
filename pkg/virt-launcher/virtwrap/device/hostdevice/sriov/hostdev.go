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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package sriov

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
)

const (
	AliasPrefix = "sriov-"

	MaxConcurrentHotPlugDevicesEvents = 32

	affectLiveAndConfigLibvirtFlags = libvirt.DOMAIN_DEVICE_MODIFY_LIVE | libvirt.DOMAIN_DEVICE_MODIFY_CONFIG
)

func CreateHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	SRIOVInterfaces := filterVMISRIOVInterfaces(vmi)
	return CreateHostDevicesFromIfacesAndPool(SRIOVInterfaces, NewPCIAddressPool(SRIOVInterfaces))
}

func CreateHostDevicesFromIfacesAndPool(ifaces []v1.Interface, pool hostdevice.AddressPooler) ([]api.HostDevice, error) {
	hostDevicesMetaData := createHostDevicesMetadata(ifaces)
	return hostdevice.CreatePCIHostDevices(hostDevicesMetaData, pool)
}

func createHostDevicesMetadata(ifaces []v1.Interface) []hostdevice.HostDeviceMetaData {
	var hostDevicesMetaData []hostdevice.HostDeviceMetaData
	for _, iface := range ifaces {
		hostDevicesMetaData = append(hostDevicesMetaData, hostdevice.HostDeviceMetaData{
			AliasPrefix:  AliasPrefix,
			Name:         iface.Name,
			ResourceName: iface.Name,
			DecorateHook: func(hostDevice *api.HostDevice) error {
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
			},
		})
	}
	return hostDevicesMetaData
}

type deviceDetacher interface {
	DetachDeviceFlags(xml string, flags libvirt.DomainDeviceModifyFlags) error
}

type eventRegistrar interface {
	Register() error
	Deregister() error
	EventChannel() <-chan interface{}
}

func SafelyDetachHostDevices(domainSpec *api.DomainSpec, eventDetach eventRegistrar, dom deviceDetacher, timeout time.Duration) error {
	sriovDevices := FilterHostDevices(domainSpec)
	if len(sriovDevices) == 0 {
		log.Log.Info("No SR-IOV host-devices to detach.")
		return nil
	}

	if err := eventDetach.Register(); err != nil {
		return fmt.Errorf("failed to detach host-devices: %v", err)
	}
	defer func() {
		if err := eventDetach.Deregister(); err != nil {
			log.Log.Reason(err).Errorf("failed to detach host-devices: %v", err)
		}
	}()

	if err := detachHostDevices(dom, sriovDevices); err != nil {
		return err
	}

	return waitHostDevicesToDetach(eventDetach, sriovDevices, timeout)
}

func FilterHostDevices(domainSpec *api.DomainSpec) []api.HostDevice {
	var hostDevices []api.HostDevice

	for _, hostDevice := range domainSpec.Devices.HostDevices {
		if hostDevice.Alias != nil && strings.HasPrefix(hostDevice.Alias.GetName(), AliasPrefix) {
			hostDevices = append(hostDevices, hostDevice)
		}
	}
	return hostDevices
}

func detachHostDevices(dom deviceDetacher, hostDevices []api.HostDevice) error {
	for _, hostDev := range hostDevices {
		devXML, err := xml.Marshal(hostDev)
		if err != nil {
			return fmt.Errorf("failed to encode (xml) hostdev %v, err: %v", hostDev, err)
		}
		err = dom.DetachDeviceFlags(string(devXML), affectLiveAndConfigLibvirtFlags)
		if err != nil {
			return fmt.Errorf("failed to detach hostdev %s, err: %v", devXML, err)
		}
		log.Log.Infof("Successfully hot-unplug hostdev: %s (%v)", hostDev.Alias.GetName(), hostDev.Source.Address)
	}
	return nil
}

func waitHostDevicesToDetach(eventDetach eventRegistrar, hostDevices []api.HostDevice, timeout time.Duration) error {
	var detachedHostDevices []string
	var desiredDetachCount = len(hostDevices)

	for {
		select {
		case deviceAlias := <-eventDetach.EventChannel():
			if dev := deviceLookup(hostDevices, deviceAlias.(string)); dev != nil {
				detachedHostDevices = append(detachedHostDevices, dev.Alias.GetName())
			}
			if desiredDetachCount == len(detachedHostDevices) {
				return nil
			}
		case <-time.After(timeout):

			return fmt.Errorf(
				"failed to wait for host-devices detach, timeout reached: %v/%v",
				detachedHostDevices, hostDevicesNames(hostDevices))
		}
	}
}

func hostDevicesNames(hostDevices []api.HostDevice) []string {
	var names []string
	for _, dev := range hostDevices {
		names = append(names, dev.Alias.GetName())
	}
	return names
}

func deviceLookup(hostDevices []api.HostDevice, deviceAlias string) *api.HostDevice {
	deviceAlias = strings.TrimPrefix(deviceAlias, api.UserAliasPrefix)
	for _, dev := range hostDevices {
		if dev.Alias.GetName() == deviceAlias {
			return &dev
		}
	}
	return nil
}

type deviceAttacher interface {
	AttachDeviceFlags(xmlData string, flags libvirt.DomainDeviceModifyFlags) error
}

func AttachHostDevices(dom deviceAttacher, hostDevices []api.HostDevice) error {
	var errs []error
	for _, hostDev := range hostDevices {
		if err := attachHostDevice(dom, hostDev); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return buildAttachHostDevicesErrorMessage(errs)
	}

	return nil
}

func attachHostDevice(dom deviceAttacher, hostDev api.HostDevice) error {
	devXML, err := xml.Marshal(hostDev)
	if err != nil {
		return fmt.Errorf("failed to encode (xml) host-device %v, err: %v", hostDev, err)
	}
	err = dom.AttachDeviceFlags(string(devXML), affectLiveAndConfigLibvirtFlags)
	if err != nil {
		return fmt.Errorf("failed to attach host-device %s, err: %v", devXML, err)
	}
	log.Log.Infof("Successfully hot-plug host-device: %s (%v)", hostDev.Alias.GetName(), hostDev.Source.Address)

	return nil
}

func buildAttachHostDevicesErrorMessage(errors []error) error {
	errorMessageBuilder := strings.Builder{}
	for _, err := range errors {
		errorMessageBuilder.WriteString(err.Error() + "\n")
	}
	return fmt.Errorf(errorMessageBuilder.String())
}

func GetHostDevicesToAttach(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec) ([]api.HostDevice, error) {
	sriovDevices, err := CreateHostDevices(vmi)
	if err != nil {
		return nil, err
	}
	currentAttachedSRIOVHostDevices := FilterHostDevices(domainSpec)

	sriovHostDevicesToAttach := DifferenceHostDevicesByAlias(sriovDevices, currentAttachedSRIOVHostDevices)

	return sriovHostDevicesToAttach, nil
}

// DifferenceHostDevicesByAlias given two slices of host-devices, according to Alias.Name,
// it returns a slice with host-devices that exists on the first slice and not exists on the second.
func DifferenceHostDevicesByAlias(desiredHostDevices, actualHostDevices []api.HostDevice) []api.HostDevice {
	actualHostDevicesByAlias := make(map[string]struct{}, len(actualHostDevices))
	for _, hostDev := range actualHostDevices {
		actualHostDevicesByAlias[hostDev.Alias.GetName()] = struct{}{}
	}

	var filteredSlice []api.HostDevice
	for _, desiredHostDevice := range desiredHostDevices {
		if _, exists := actualHostDevicesByAlias[desiredHostDevice.Alias.GetName()]; !exists {
			filteredSlice = append(filteredSlice, desiredHostDevice)
		}
	}

	return filteredSlice
}
