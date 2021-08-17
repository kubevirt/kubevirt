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

package legacy_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/legacy"
)

var _ = Describe("GPU/vGPU HostDevice", func() {
	It("creates no device given no resources", func() {
		Expect(legacy.CreateGPUHostDevices()).To(BeEmpty())
		Expect(legacy.CreateVGPUHostDevices()).To(BeEmpty())
	})

	It("fails to create (GPU) device given invalid addresses", func() {
		pool := &addressPoolStub{addresses: []string{"bad:address"}}
		_, err := legacy.CreateGPUHostDevicesFromPool(pool)
		Expect(err).To(HaveOccurred())
	})

	It("creates two GPU devices", func() {
		const pci0 = "0000:19:90.0"
		const pci1 = "0000:19:90.1"

		pool := &addressPoolStub{addresses: []string{pci0, pci1}}
		devices, err := legacy.CreateGPUHostDevicesFromPool(pool)

		expectHostDevice1 := createExpectedPCIHostDevice("0x0000", "0x19", "0x90", "0x0")
		expectHostDevice2 := createExpectedPCIHostDevice("0x0000", "0x19", "0x90", "0x1")
		Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
	})

	It("creates two vGPU devices", func() {
		const uuid0 = "aa618089-8b16-4d01-a136-25a0f3c73123"
		const uuid1 = "aa618089-8b16-4d01-a136-25a0f3c73124"

		pool := &addressPoolStub{addresses: []string{uuid0, uuid1}}
		devices, err := legacy.CreateVGPUHostDevicesFromPool(pool)

		expectHostDevice1 := createExpectedMdevHostDevice(uuid0)
		expectHostDevice2 := createExpectedMdevHostDevice(uuid1)
		Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
	})
})

type addressPoolStub struct {
	addresses []string
}

func (p *addressPoolStub) Len() int {
	fmt.Printf("len: %v", p.addresses)
	return len(p.addresses)
}

func (p *addressPoolStub) Pop() (string, error) {
	if p.Len() > 0 {
		fmt.Printf("pop: %v", p.addresses)
		addr := p.addresses[0]
		p.addresses = p.addresses[1:]
		return addr, nil
	}
	return "", fmt.Errorf("pool empty")
}

func createExpectedPCIHostDevice(domain, bus, slot, function string) api.HostDevice {
	hostPCIAddress := api.Address{Type: "pci", Domain: domain, Bus: bus, Slot: slot, Function: function}
	return api.HostDevice{
		Source:  api.HostDeviceSource{Address: &hostPCIAddress},
		Type:    "pci",
		Managed: "yes",
	}
}

func createExpectedMdevHostDevice(uuid string) api.HostDevice {
	return api.HostDevice{
		Source: api.HostDeviceSource{
			Address: &api.Address{
				UUID: uuid,
			},
		},
		Type:  "mdev",
		Mode:  "subsystem",
		Model: "vfio-pci",
	}
}
