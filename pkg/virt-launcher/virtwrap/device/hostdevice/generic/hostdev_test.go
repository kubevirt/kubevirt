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

package generic_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/generic"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/vfio"
)

var _ = Describe("Generic HostDevice", func() {
	var vmi *v1.VirtualMachineInstance
	var vfioSpec vfio.VFIOSpec

	DescribeTableSubtree("creation", func(viaIOMMUFD bool) {
		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{}

			mockVFIOSpec := vfio.NewMockVFIOSpec(gomock.NewController(GinkgoT()))
			mockVFIOSpec.EXPECT().IsPCIAssignableViaIOMMUFD(hostdevPCIAddress0).Return(viaIOMMUFD).AnyTimes()
			mockVFIOSpec.EXPECT().IsPCIAssignableViaIOMMUFD(gomock.Any()).Times(0)
			mockVFIOSpec.EXPECT().IsMDevAssignableViaIOMMUFD(hostdevMDEVAddress1).Return(viaIOMMUFD).AnyTimes()
			mockVFIOSpec.EXPECT().IsMDevAssignableViaIOMMUFD(gomock.Any()).Times(0)
			vfioSpec = mockVFIOSpec
		})

		It("creates no device given no generic host-devices/s", func() {
			Expect(generic.CreateHostDevices(vmi.Spec.Domain.Devices.HostDevices, vfioSpec)).To(BeEmpty())
		})

		It("fails to create devices given no resource", func() {
			vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{{DeviceName: hostdevResource0, Name: hostdevName0}}
			_, err := generic.CreateHostDevices(vmi.Spec.Domain.Devices.HostDevices, vfioSpec)
			Expect(err).To(HaveOccurred())
		})

		It("fails to create device given two devices but only one address", func() {
			vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
				{DeviceName: hostdevResource0, Name: hostdevName0},
				{DeviceName: hostdevResource0, Name: hostdevName1},
			}
			pciPool := newAddressPoolStub()
			pciPool.AddResource(hostdevResource0, hostdevPCIAddress0)
			mdevPool := newAddressPoolStub()
			mdevPool.AddResource(hostdevResource1, hostdevPCIAddress1)
			usbPool := newAddressPoolStub()

			_, err := generic.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.HostDevices, pciPool, mdevPool, usbPool, vfioSpec)
			Expect(err).To(HaveOccurred())
		})

		It("creates two devices, PCI and MDEV", func() {
			vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
				{DeviceName: hostdevResource0, Name: hostdevName0},
				{DeviceName: hostdevResource1, Name: hostdevName1},
			}
			pciPool := newAddressPoolStub()
			pciPool.AddResource(hostdevResource0, hostdevPCIAddress0)
			mdevPool := newAddressPoolStub()
			mdevPool.AddResource(hostdevResource1, hostdevMDEVAddress1)
			usbPool := newAddressPoolStub()

			hostPCIAddress := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			expectHostDevice0 := api.HostDevice{
				Alias:   api.NewUserDefinedAlias(generic.AliasPrefix + hostdevName0),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress},
				Type:    api.HostDevicePCI,
				Managed: "no",
			}
			if viaIOMMUFD {
				expectHostDevice0.Driver = &api.HostDeviceDriver{IOMMUFD: "yes"}
			}

			hostMDEVAddress := api.Address{UUID: hostdevMDEVAddress1}
			expectHostDevice1 := api.HostDevice{
				Alias:  api.NewUserDefinedAlias(generic.AliasPrefix + hostdevName1),
				Source: api.HostDeviceSource{Address: &hostMDEVAddress},
				Type:   api.HostDeviceMDev,
				Mode:   "subsystem",
				Model:  "vfio-pci",
			}
			if viaIOMMUFD {
				expectHostDevice1.Driver = &api.HostDeviceDriver{IOMMUFD: "yes"}
			}

			Expect(generic.CreateHostDevicesFromPools(vmi.Spec.Domain.Devices.HostDevices, pciPool, mdevPool, usbPool, vfioSpec)).
				To(Equal([]api.HostDevice{expectHostDevice0, expectHostDevice1}))
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
