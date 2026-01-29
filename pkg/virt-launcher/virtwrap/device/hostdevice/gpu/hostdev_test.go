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

package gpu_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/gpu"
)

var _ = Describe("GPU HostDevice", func() {
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		vmi = &v1.VirtualMachineInstance{}
	})

	It("creates no device given no GPU/s", func() {
		Expect(gpu.CreateHostDevices(vmi.Spec.Domain.Devices.GPUs)).To(BeEmpty())
	})

	It("fails to create devices given no resource", func() {
		vmi.Spec.Domain.Devices.GPUs = []v1.GPU{{DeviceName: gpuResource0, Name: gpuName0}}
		_, err := gpu.CreateHostDevices(vmi.Spec.Domain.Devices.GPUs)
		Expect(err).To(HaveOccurred())
	})

	It("fails to create device given two devices but only one address", func() {
		vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
			{DeviceName: gpuResource0, Name: gpuName0},
			{DeviceName: gpuResource0, Name: gpuName1},
		}
		pciPool := newAddressPoolStub()
		pciPool.AddResource(gpuResource0, gpuPCIAddress0)
		mdevPool := newAddressPoolStub()
		mdevPool.AddResource(gpuResource1, gpuPCIAddress1)

		_, err := gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool)
		Expect(err).To(HaveOccurred())
	})

	It("creates two devices, PCI and MDEV", func() {
		vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
			{DeviceName: gpuResource0, Name: gpuName0},
			{DeviceName: gpuResource1, Name: gpuName1},
		}
		pciPool := newAddressPoolStub()
		pciPool.AddResource(gpuResource0, gpuPCIAddress0)
		mdevPool := newAddressPoolStub()
		mdevPool.AddResource(gpuResource1, gpuMDEVAddress1)

		hostPCIAddress := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
		expectHostDevice0 := api.HostDevice{
			Alias:   api.NewUserDefinedAlias(gpu.AliasPrefix + gpuName0),
			Source:  api.HostDeviceSource{Address: &hostPCIAddress},
			Type:    api.HostDevicePCI,
			Managed: "no",
		}

		hostMDEVAddress := api.Address{UUID: gpuMDEVAddress1}
		expectHostDevice1 := api.HostDevice{
			Alias:   api.NewUserDefinedAlias(gpu.AliasPrefix + gpuName1),
			Source:  api.HostDeviceSource{Address: &hostMDEVAddress},
			Type:    api.HostDeviceMDev,
			Mode:    "subsystem",
			Model:   "vfio-pci",
			Display: "on",
			RamFB:   "on",
		}

		Expect(gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool)).
			To(Equal([]api.HostDevice{expectHostDevice0, expectHostDevice1}))
	})
	It("creates MDEV with display option turned off", func() {
		_false := false
		vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
			{
				DeviceName: gpuResource1,
				Name:       gpuName1,
				VirtualGPUOptions: &v1.VGPUOptions{
					Display: &v1.VGPUDisplayOptions{
						Enabled: &_false,
					},
				},
			},
		}
		pciPool := newAddressPoolStub()
		pciPool.AddResource(gpuResource0, gpuPCIAddress0)
		mdevPool := newAddressPoolStub()
		mdevPool.AddResource(gpuResource1, gpuMDEVAddress1)

		hostMDEVAddress := api.Address{UUID: gpuMDEVAddress1}
		expectHostDevice1 := api.HostDevice{
			Alias:  api.NewUserDefinedAlias(gpu.AliasPrefix + gpuName1),
			Source: api.HostDeviceSource{Address: &hostMDEVAddress},
			Type:   api.HostDeviceMDev,
			Mode:   "subsystem",
			Model:  "vfio-pci",
		}

		Expect(gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool)).
			To(Equal([]api.HostDevice{expectHostDevice1}))
	})
	It("creates MDEV with display ramFB option turned off", func() {
		_false := false
		vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
			{
				DeviceName: gpuResource1,
				Name:       gpuName1,
				VirtualGPUOptions: &v1.VGPUOptions{
					Display: &v1.VGPUDisplayOptions{
						RamFB: &v1.FeatureState{
							Enabled: &_false,
						},
					},
				},
			},
		}
		pciPool := newAddressPoolStub()
		pciPool.AddResource(gpuResource0, gpuPCIAddress0)
		mdevPool := newAddressPoolStub()
		mdevPool.AddResource(gpuResource1, gpuMDEVAddress1)

		hostMDEVAddress := api.Address{UUID: gpuMDEVAddress1}
		expectHostDevice1 := api.HostDevice{
			Alias:   api.NewUserDefinedAlias(gpu.AliasPrefix + gpuName1),
			Source:  api.HostDeviceSource{Address: &hostMDEVAddress},
			Type:    api.HostDeviceMDev,
			Mode:    "subsystem",
			Model:   "vfio-pci",
			Display: "on",
		}

		Expect(gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool)).
			To(Equal([]api.HostDevice{expectHostDevice1}))
	})
	It("creates MDEV with enabled display and ramfb by default", func() {
		vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
			{
				DeviceName: gpuResource1,
				Name:       gpuName1,
				VirtualGPUOptions: &v1.VGPUOptions{
					Display: &v1.VGPUDisplayOptions{},
				},
			},
		}
		pciPool := newAddressPoolStub()
		pciPool.AddResource(gpuResource0, gpuPCIAddress0)
		mdevPool := newAddressPoolStub()
		mdevPool.AddResource(gpuResource1, gpuMDEVAddress1)

		hostMDEVAddress := api.Address{UUID: gpuMDEVAddress1}
		expectHostDevice1 := api.HostDevice{
			Alias:   api.NewUserDefinedAlias(gpu.AliasPrefix + gpuName1),
			Source:  api.HostDeviceSource{Address: &hostMDEVAddress},
			Type:    api.HostDeviceMDev,
			Mode:    "subsystem",
			Model:   "vfio-pci",
			Display: "on",
			RamFB:   "on",
		}

		Expect(gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool)).
			To(Equal([]api.HostDevice{expectHostDevice1}))
	})
})

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

func (p *stubAddressPool) PopAll(resource string) ([]string, error) {
	var addresses []string
	for {
		addr, err := p.Pop(resource)
		if err != nil {
			break
		}
		addresses = append(addresses, addr)
	}
	return addresses, nil
}

var _ = Describe("GPU IOMMU Companion Devices", func() {
	const (
		gpuPCIAddress2 = "0000:81:01.2"
		gpuPCIAddress3 = "0000:81:01.3"
	)

	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		vmi = &v1.VirtualMachineInstance{}
	})

	It("creates IOMMU companion devices for remaining addresses in pool", func() {
		// GPU with one requested device but two PCI addresses in pool (GPU + audio controller)
		vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
			{DeviceName: gpuResource0, Name: gpuName0},
		}
		pciPool := newAddressPoolStub()
		// First address is for the GPU, second is for the IOMMU companion (e.g., audio controller)
		pciPool.AddResource(gpuResource0, gpuPCIAddress0, gpuPCIAddress1)
		mdevPool := newAddressPoolStub()

		hostDevices, err := gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool)
		Expect(err).NotTo(HaveOccurred())

		// Should have 2 devices: 1 primary GPU + 1 IOMMU companion
		Expect(hostDevices).To(HaveLen(2))

		// First device should be the primary GPU
		Expect(hostDevices[0].Alias.GetName()).To(Equal(gpu.AliasPrefix + gpuName0))

		// Second device should be the IOMMU companion
		Expect(hostDevices[1].Alias.GetName()).To(ContainSubstring("iommu-companion"))
		Expect(hostDevices[1].Type).To(Equal(api.HostDevicePCI))
		Expect(hostDevices[1].Managed).To(Equal("no"))
	})

	It("creates multiple IOMMU companion devices when pool has many addresses", func() {
		vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
			{DeviceName: gpuResource0, Name: gpuName0},
		}
		pciPool := newAddressPoolStub()
		// GPU with 3 additional IOMMU group members
		pciPool.AddResource(gpuResource0, gpuPCIAddress0, gpuPCIAddress1, gpuPCIAddress2, gpuPCIAddress3)
		mdevPool := newAddressPoolStub()

		hostDevices, err := gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool)
		Expect(err).NotTo(HaveOccurred())

		// Should have 4 devices: 1 primary GPU + 3 IOMMU companions
		Expect(hostDevices).To(HaveLen(4))

		// First device should be the primary GPU
		Expect(hostDevices[0].Alias.GetName()).To(Equal(gpu.AliasPrefix + gpuName0))

		// Remaining devices should be IOMMU companions
		for i := 1; i < len(hostDevices); i++ {
			Expect(hostDevices[i].Alias.GetName()).To(ContainSubstring("iommu-companion"))
		}
	})

	It("creates no IOMMU companion devices when pool is exactly exhausted", func() {
		vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
			{DeviceName: gpuResource0, Name: gpuName0},
		}
		pciPool := newAddressPoolStub()
		// Exactly one address for one GPU
		pciPool.AddResource(gpuResource0, gpuPCIAddress0)
		mdevPool := newAddressPoolStub()

		hostDevices, err := gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool)
		Expect(err).NotTo(HaveOccurred())

		// Should have 1 device: just the primary GPU
		Expect(hostDevices).To(HaveLen(1))
		Expect(hostDevices[0].Alias.GetName()).To(Equal(gpu.AliasPrefix + gpuName0))
	})

	It("handles IOMMU companion devices with multiple GPUs from same resource", func() {
		vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
			{DeviceName: gpuResource0, Name: gpuName0},
			{DeviceName: gpuResource0, Name: gpuName1},
		}
		pciPool := newAddressPoolStub()
		// 2 primary GPUs + 2 IOMMU companions
		pciPool.AddResource(gpuResource0, gpuPCIAddress0, gpuPCIAddress1, gpuPCIAddress2, gpuPCIAddress3)
		mdevPool := newAddressPoolStub()

		hostDevices, err := gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool)
		Expect(err).NotTo(HaveOccurred())

		// Should have 4 devices: 2 primary GPUs + 2 IOMMU companions
		Expect(hostDevices).To(HaveLen(4))

		// First two devices should be the primary GPUs
		Expect(hostDevices[0].Alias.GetName()).To(Equal(gpu.AliasPrefix + gpuName0))
		Expect(hostDevices[1].Alias.GetName()).To(Equal(gpu.AliasPrefix + gpuName1))

		// Last two should be IOMMU companions
		Expect(hostDevices[2].Alias.GetName()).To(ContainSubstring("iommu-companion"))
		Expect(hostDevices[3].Alias.GetName()).To(ContainSubstring("iommu-companion"))
	})

	It("does not include MDEV addresses in IOMMU companion devices", func() {
		vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
			{DeviceName: gpuResource0, Name: gpuName0},
			{DeviceName: gpuResource1, Name: gpuName1},
		}
		pciPool := newAddressPoolStub()
		pciPool.AddResource(gpuResource0, gpuPCIAddress0, gpuPCIAddress1) // GPU + companion
		mdevPool := newAddressPoolStub()
		mdevPool.AddResource(gpuResource1, gpuMDEVAddress1) // Just MDEV, no companion

		hostDevices, err := gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool)
		Expect(err).NotTo(HaveOccurred())

		// Should have 3 devices: 1 PCI GPU + 1 IOMMU companion + 1 MDEV
		Expect(hostDevices).To(HaveLen(3))

		// IOMMU companion should only come from PCI pool
		companionCount := 0
		for _, device := range hostDevices {
			if device.Alias != nil && device.Type == api.HostDevicePCI {
				if device.Alias.GetName() != gpu.AliasPrefix+gpuName0 {
					companionCount++
					Expect(device.Alias.GetName()).To(ContainSubstring("iommu-companion"))
				}
			}
		}
		Expect(companionCount).To(Equal(1))
	})
})
