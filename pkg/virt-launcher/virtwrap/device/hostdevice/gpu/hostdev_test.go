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
	"go.uber.org/mock/gomock"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/gpu"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/vfio"
)

var _ = Describe("GPU HostDevice", func() {
	var vmi *v1.VirtualMachineInstance
	var vfioSpec vfio.VFIOSpec

	DescribeTableSubtree("creation", func(viaIOMMUFD bool) {
		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{}

			mockVFIOSpec := vfio.NewMockVFIOSpec(gomock.NewController(GinkgoT()))
			mockVFIOSpec.EXPECT().IsPCIAssignableViaIOMMUFD(gpuPCIAddress0).Return(viaIOMMUFD).AnyTimes()
			mockVFIOSpec.EXPECT().IsPCIAssignableViaIOMMUFD(gomock.Any()).Times(0)
			mockVFIOSpec.EXPECT().IsMDevAssignableViaIOMMUFD(gpuMDEVAddress1).Return(viaIOMMUFD).AnyTimes()
			mockVFIOSpec.EXPECT().IsMDevAssignableViaIOMMUFD(gomock.Any()).Times(0)
			vfioSpec = mockVFIOSpec
		})

		It("creates no device given no GPU/s", func() {
			Expect(gpu.CreateHostDevices(vmi.Spec.Domain.Devices.GPUs, vfioSpec)).To(BeEmpty())
		})

		It("fails to create devices given no resource", func() {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{{DeviceName: gpuResource0, Name: gpuName0}}
			_, err := gpu.CreateHostDevices(vmi.Spec.Domain.Devices.GPUs, vfioSpec)
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

			_, err := gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool, vfioSpec)
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
			if viaIOMMUFD {
				expectHostDevice0.Driver = &api.HostDeviceDriver{IOMMUFD: "yes"}
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
			if viaIOMMUFD {
				expectHostDevice1.Driver = &api.HostDeviceDriver{IOMMUFD: "yes"}
			}

			Expect(gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool, vfioSpec)).
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
			if viaIOMMUFD {
				expectHostDevice1.Driver = &api.HostDeviceDriver{IOMMUFD: "yes"}
			}

			Expect(gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool, vfioSpec)).
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
			if viaIOMMUFD {
				expectHostDevice1.Driver = &api.HostDeviceDriver{IOMMUFD: "yes"}
			}

			Expect(gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool, vfioSpec)).
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
			if viaIOMMUFD {
				expectHostDevice1.Driver = &api.HostDeviceDriver{IOMMUFD: "yes"}
			}

			Expect(gpu.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.GPUs, pciPool, mdevPool, vfioSpec)).
				To(Equal([]api.HostDevice{expectHostDevice1}))
		})
	},
		Entry("via IOMMUFD", true),
		Entry("via VFIO legacy", false),
	)
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
