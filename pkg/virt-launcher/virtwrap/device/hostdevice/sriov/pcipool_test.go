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

package sriov_test

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/sriov"
)

type networkData struct {
	Name         string
	ResourceEnv  envData
	DeviceEnv    envData
	resource     resourceData
	PCIAddresses []string
}

type resourceData struct {
	Name         string
	PCIAddresses []string
}

type envData struct {
	Name  string
	Value string
}

var _ = Describe("SRIOV PCI address pool", func() {
	It("fails to pop an address given no interfaces", func() {
		pool := sriov.NewPCIAddressPool([]v1.Interface{})
		expectPoolPopFailure(pool, "foo")
	})

	It("fails to pop an address given a non-SRIOV interface", func() {
		pool := sriov.NewPCIAddressPool([]v1.Interface{{Name: "foo"}})
		expectPoolPopFailure(pool, "foo")
	})

	It("fails to pop an address given a missing resource name env", func() {
		iface := newSRIOVInterface("foo")
		pool := sriov.NewPCIAddressPool([]v1.Interface{iface})

		expectPoolPopFailure(pool, "foo")
	})

	It("fails to pop an address given a missing device name env", func() {
		net := newNetworkData("net1", newResourceData("resource1", "0000:81:01.0"))
		env := []envData{net.ResourceEnv}
		withEnvironmentContext(env, func() {
			iface := newSRIOVInterface(net.Name)
			pool := sriov.NewPCIAddressPool([]v1.Interface{iface})

			expectPoolPopFailure(pool, net.Name)
		})
	})

	It("provides 1 address given 1xInterface, 1xResource, 1xPCI", func() {
		// The comma at the tail of the PCI address is intentional, validating it is ignored by the implementation.
		const pciAddress = "0000:81:01.0"
		net := newNetworkData("net1", newResourceData("resource1", pciAddress+","))
		env := []envData{net.ResourceEnv, net.DeviceEnv}
		withEnvironmentContext(env, func() {
			iface := newSRIOVInterface(net.Name)
			pool := sriov.NewPCIAddressPool([]v1.Interface{iface})

			Expect(pool.Pop(net.Name)).To(Equal(pciAddress))
			expectPoolPopFailure(pool, net.Name)
		})
	})

	It("provides 2 addresses given 1xInterface, 1xResource, 2xPCI", func() {
		net := newNetworkData("net1", newResourceData("resource1", "0000:81:01.0", "0000:81:01.1"))
		env := []envData{net.ResourceEnv, net.DeviceEnv}
		withEnvironmentContext(env, func() {
			iface := newSRIOVInterface(net.Name)
			pool := sriov.NewPCIAddressPool([]v1.Interface{iface})

			Expect(pool.Pop(net.Name)).To(Equal(net.PCIAddresses[0]))
			Expect(pool.Pop(net.Name)).To(Equal(net.PCIAddresses[1]))
			expectPoolPopFailure(pool, net.Name)
		})
	})

	It("provides 2 addresses given 2xInterface, 1xResource, 2xPCI", func() {
		resource := newResourceData("resource1", "0000:81:01.0", "0000:81:02.0")
		net1 := newNetworkData("net1", resource)
		net2 := newNetworkData("net2", resource)
		env := []envData{net1.ResourceEnv, net1.DeviceEnv, net2.ResourceEnv}
		withEnvironmentContext(env, func() {
			iface1 := newSRIOVInterface(net1.Name)
			iface2 := newSRIOVInterface(net2.Name)
			pool := sriov.NewPCIAddressPool([]v1.Interface{iface1, iface2})

			Expect(pool.Pop(net1.Name)).To(Equal(net1.PCIAddresses[0]))
			Expect(pool.Pop(net2.Name)).To(Equal(net1.PCIAddresses[1]))
			expectPoolPopFailure(pool, net1.Name)
			expectPoolPopFailure(pool, net2.Name)
		})
	})

	It("provides 2 addresses given 2xInterface, 2xResource, 2xPCI", func() {
		net1 := newNetworkData("net1", newResourceData("resource1", "0000:81:01.0"))
		net2 := newNetworkData("net2", newResourceData("resource2", "0000:81:02.0"))
		env := []envData{net1.ResourceEnv, net1.DeviceEnv, net2.ResourceEnv, net2.DeviceEnv}
		withEnvironmentContext(env, func() {
			iface1 := newSRIOVInterface(net1.Name)
			iface2 := newSRIOVInterface(net2.Name)
			pool := sriov.NewPCIAddressPool([]v1.Interface{iface1, iface2})

			Expect(pool.Pop(net1.Name)).To(Equal(net1.PCIAddresses[0]))
			Expect(pool.Pop(net2.Name)).To(Equal(net2.PCIAddresses[0]))
			expectPoolPopFailure(pool, net1.Name)
			expectPoolPopFailure(pool, net2.Name)
		})
	})

	It("provides 1 addresses given 2xInterface, 2xResource, 1xPCI", func() {
		net1 := newNetworkData("net1", newResourceData("resource1", "0000:81:01.0"))
		net2 := newNetworkData("net2", newResourceData("resource2", "0000:81:01.0"))
		env := []envData{net1.ResourceEnv, net1.DeviceEnv, net2.ResourceEnv, net2.DeviceEnv}
		withEnvironmentContext(env, func() {
			iface1 := newSRIOVInterface(net1.Name)
			iface2 := newSRIOVInterface(net2.Name)
			pool := sriov.NewPCIAddressPool([]v1.Interface{iface1, iface2})

			Expect(pool.Pop(net1.Name)).To(Equal(net1.PCIAddresses[0]))
			expectPoolPopFailure(pool, net2.Name)
			expectPoolPopFailure(pool, net1.Name)
		})
	})
})

func newResourceData(resourceName string, addresses ...string) resourceData {
	return resourceData{Name: resourceName, PCIAddresses: addresses}
}

func newNetworkData(netName string, resource resourceData) networkData {
	return networkData{
		Name: netName,
		ResourceEnv: envData{
			"KUBEVIRT_RESOURCE_NAME_" + netName,
			"intel.com/" + resource.Name + "_pool",
		},
		DeviceEnv: envData{
			"PCIDEVICE_INTEL_COM_" + strings.ToUpper(resource.Name) + "_POOL",
			strings.Join(resource.PCIAddresses, ","),
		},
		PCIAddresses: resource.PCIAddresses,
	}
}

func withEnvironmentContext(envDataList []envData, f func()) {
	for _, envVar := range envDataList {
		os.Setenv(envVar.Name, envVar.Value)
		defer os.Unsetenv(envVar.Name)
	}
	f()
}

func expectPoolPopFailure(pool *sriov.PCIAddressPool, networkName string) {
	address, err := pool.Pop(networkName)
	ExpectWithOffset(1, err).To(HaveOccurred())
	ExpectWithOffset(1, address).To(BeEmpty())
}
