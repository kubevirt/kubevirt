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

package hostdevice_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
)

type createHostDevices func([]hostdevice.HostDeviceMetaData, hostdevice.AddressPooler) ([]api.HostDevice, error)

const (
	aliasPrefix = "test_prefix"

	resourceName0 = "test_resource0"
	resourceName1 = "test_resource1"

	devName0 = "test_device0"
	devName1 = "test_device1"
)

var _ = Describe("HostDevice", func() {
	createMDEVWithoutDisplay := func(hostDevicesMetaData []hostdevice.HostDeviceMetaData, pool hostdevice.AddressPooler) ([]api.HostDevice, error) {
		return hostdevice.CreateMDEVHostDevices(hostDevicesMetaData, pool, false)
	}

	It("creates no device given no devices-metadata", func() {
		Expect(hostdevice.CreatePCIHostDevices(nil, newAddressPoolStub())).To(BeEmpty())
		Expect(hostdevice.CreateMDEVHostDevices(nil, newAddressPoolStub(), false)).To(BeEmpty())
	})

	var pool *stubAddressPool

	BeforeEach(func() {
		pool = newAddressPoolStub()
	})

	DescribeTable("fails to create device given no available addresses in pool",
		func(createHostDevices createHostDevices) {
			hostDevicesMetaData := []hostdevice.HostDeviceMetaData{{}}
			_, err := createHostDevices(hostDevicesMetaData, pool)

			Expect(err).To(HaveOccurred())
		},
		Entry("PCI", hostdevice.CreatePCIHostDevices),
		Entry("MDEV", createMDEVWithoutDisplay),
	)

	It("fails to create a device given bad host PCI address", func() {
		pool.AddResource(resource0, "0bad0pci0address0")
		hostDevicesMetaData := []hostdevice.HostDeviceMetaData{{ResourceName: resource0}}
		_, err := hostdevice.CreatePCIHostDevices(hostDevicesMetaData, pool)

		Expect(err).To(HaveOccurred())
	})

	DescribeTable("fails to create a device when hook returns error",
		func(createHostDevices createHostDevices) {
			pool.AddResource(resource0, "0000:81:01.0")
			hostDevicesMetaData := []hostdevice.HostDeviceMetaData{{
				ResourceName: resource0,
				DecorateHook: func(hostDevice *api.HostDevice) error { return fmt.Errorf("failed hook") },
			}}

			_, err := createHostDevices(hostDevicesMetaData, pool)

			Expect(err).To(HaveOccurred())
		},
		Entry("PCI", hostdevice.CreatePCIHostDevices),
		Entry("MDEV", createMDEVWithoutDisplay),
	)

	DescribeTable("fails to create a device given two devices but only one address",
		func(createHostDevices createHostDevices) {
			pool.AddResource(resource0, "0000:81:01.0")
			hostDevicesMetaData := []hostdevice.HostDeviceMetaData{{ResourceName: resource0}, {ResourceName: resource0}}

			_, err := createHostDevices(hostDevicesMetaData, pool)

			Expect(err).To(HaveOccurred())
		},
		Entry("PCI", hostdevice.CreatePCIHostDevices),
		Entry("MDEV", createMDEVWithoutDisplay),
	)

	Context("PCI", func() {
		const pciAddress0 = "0000:81:01.0"
		hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
		expectHostDevice1 := api.HostDevice{
			Alias:   newAlias(devName0),
			Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
			Type:    api.HostDevicePCI,
			Managed: "no",
		}
		const pciAddress1 = "0000:81:01.1"
		hostPCIAddress2 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x1"}
		expectHostDevice2 := api.HostDevice{
			Alias:   newAlias(devName1),
			Source:  api.HostDeviceSource{Address: &hostPCIAddress2},
			Type:    api.HostDevicePCI,
			Managed: "no",
		}

		It("creates 2 PCI devices that share the same resource", func() {
			hostDevicesMetaData := []hostdevice.HostDeviceMetaData{
				{AliasPrefix: aliasPrefix, Name: devName0, ResourceName: resourceName0},
				{AliasPrefix: aliasPrefix, Name: devName1, ResourceName: resourceName0},
			}
			pool.AddResource(resourceName0, pciAddress0, pciAddress1)

			hostDevices, err := hostdevice.CreatePCIHostDevices(hostDevicesMetaData, pool)

			Expect(hostDevices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
		})

		It("creates 2 PCI devices that are connected to different resources", func() {
			hostDevicesMetaData := []hostdevice.HostDeviceMetaData{
				{AliasPrefix: aliasPrefix, Name: devName0, ResourceName: resourceName0},
				{AliasPrefix: aliasPrefix, Name: devName1, ResourceName: resourceName1},
			}
			pool.AddResource(resourceName0, pciAddress0)
			pool.AddResource(resourceName1, pciAddress1)

			hostDevices, err := hostdevice.CreatePCIHostDevices(hostDevicesMetaData, pool)

			Expect(hostDevices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
		})
	})

	Context("MDEV", func() {
		const uuid0 = "0123456789-0"
		hostMDEVAddress0 := api.Address{UUID: uuid0}
		const uuid1 = "0123456789-1"
		hostMDEVAddress1 := api.Address{UUID: uuid1}
		var expectHostDevice1 api.HostDevice
		var expectHostDevice2 api.HostDevice
		BeforeEach(func() {
			expectHostDevice1 = api.HostDevice{
				Alias:  newAlias(devName0),
				Source: api.HostDeviceSource{Address: &hostMDEVAddress0},
				Type:   api.HostDeviceMDev,
				Mode:   "subsystem",
				Model:  "vfio-pci",
			}
			expectHostDevice2 = api.HostDevice{
				Alias:  newAlias(devName1),
				Source: api.HostDeviceSource{Address: &hostMDEVAddress1},
				Type:   api.HostDeviceMDev,
				Mode:   "subsystem",
				Model:  "vfio-pci",
			}
		})

		It("creates 2 MDEV devices that share the same resource", func() {
			hostDevicesMetaData := []hostdevice.HostDeviceMetaData{
				{AliasPrefix: aliasPrefix, Name: devName0, ResourceName: resourceName0},
				{AliasPrefix: aliasPrefix, Name: devName1, ResourceName: resourceName0},
			}
			pool.AddResource(resourceName0, uuid0, uuid1)

			hostDevices, err := hostdevice.CreateMDEVHostDevices(hostDevicesMetaData, pool, false)

			Expect(hostDevices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
		})

		It("makes sure that a vGPU MDEV device will turn display and ramfb on", func() {
			hostDevicesMetaData := []hostdevice.HostDeviceMetaData{
				{AliasPrefix: aliasPrefix, Name: devName0, ResourceName: resourceName0},
			}
			pool.AddResource(resourceName0, uuid0, uuid1)

			hostDevices, err := hostdevice.CreateMDEVHostDevices(hostDevicesMetaData, pool, true)
			expectHostDevice1.Display = "on"
			expectHostDevice1.RamFB = "on"

			Expect(hostDevices, err).To(Equal([]api.HostDevice{expectHostDevice1}))
		})

		It("makes sure that only one ramfb is configured with 2 vGPU devices", func() {
			hostDevicesMetaData := []hostdevice.HostDeviceMetaData{
				{AliasPrefix: aliasPrefix, Name: devName0, ResourceName: resourceName0},
				{AliasPrefix: aliasPrefix, Name: devName1, ResourceName: resourceName0},
			}
			pool.AddResource(resourceName0, uuid0, uuid1)

			hostDevices, err := hostdevice.CreateMDEVHostDevices(hostDevicesMetaData, pool, true)
			expectHostDevice1.Display = "on"
			expectHostDevice1.RamFB = "on"

			Expect(hostDevices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
		})

		It("makes sure that only one ramfb is configured with 2 vGPU devices and an explicit VirtualGPUOptions setting", func() {
			hostDevicesMetaData := []hostdevice.HostDeviceMetaData{
				{AliasPrefix: aliasPrefix, Name: devName0, ResourceName: resourceName0},
				{
					AliasPrefix:  aliasPrefix,
					Name:         devName1,
					ResourceName: resourceName0,
					VirtualGPUOptions: &v1.VGPUOptions{
						Display: &v1.VGPUDisplayOptions{
							Enabled: pointer.P(true),
							RamFB: &v1.FeatureState{
								Enabled: pointer.P(true),
							},
						},
					},
				},
			}
			pool.AddResource(resourceName0, uuid0, uuid1)

			hostDevices, err := hostdevice.CreateMDEVHostDevices(hostDevicesMetaData, pool, true)
			expectHostDevice2.Display = "on"
			expectHostDevice2.RamFB = "on"

			Expect(hostDevices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
		})

		It("makes sute that explicitly setting VirtualGPUOptions can override the default display and ramfb setting", func() {
			_false := false
			hostDevicesMetaData := []hostdevice.HostDeviceMetaData{
				{
					AliasPrefix:  aliasPrefix,
					Name:         devName0,
					ResourceName: resourceName0,
					VirtualGPUOptions: &v1.VGPUOptions{
						Display: &v1.VGPUDisplayOptions{
							Enabled: &_false,
						},
					},
				},
			}
			pool.AddResource(resourceName0, uuid0, uuid1)

			hostDevices, err := hostdevice.CreateMDEVHostDevices(hostDevicesMetaData, pool, true)

			Expect(hostDevices, err).To(Equal([]api.HostDevice{expectHostDevice1}))
		})

		It("creates 2 PCI devices that are connected to different resources", func() {
			hostDevicesMetaData := []hostdevice.HostDeviceMetaData{
				{AliasPrefix: aliasPrefix, Name: devName0, ResourceName: resourceName0},
				{AliasPrefix: aliasPrefix, Name: devName1, ResourceName: resourceName1},
			}
			pool.AddResource(resourceName0, uuid0)
			pool.AddResource(resourceName1, uuid1)

			hostDevices, err := hostdevice.CreateMDEVHostDevices(hostDevicesMetaData, pool, false)

			Expect(hostDevices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
		})
	})
})

func newAlias(netName string) *api.Alias {
	return api.NewUserDefinedAlias(aliasPrefix + netName)
}

type stubAddressPool struct {
	addresses map[string][]string
}

func newAddressPoolStub() *stubAddressPool {
	return &stubAddressPool{addresses: make(map[string][]string)}
}

func (p *stubAddressPool) AddResource(resource string, addresses ...string) {
	p.addresses[resource] = addresses
}

func (p *stubAddressPool) Pop(resource string) (string, error) {
	addresses, exists := p.addresses[resource]
	if !exists {
		return "", fmt.Errorf("no resource: %s", resource)
	}
	if len(addresses) == 0 {
		return "", fmt.Errorf("pool is empty")
	}

	address := addresses[0]
	p.addresses[resource] = addresses[1:]

	return address, nil
}
