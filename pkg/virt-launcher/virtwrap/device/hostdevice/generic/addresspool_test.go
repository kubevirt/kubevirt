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
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/generic"
)

type envData struct {
	Name  string
	Value string
}

const (
	hostdevName0 = "hostdev_name0"
	hostdevName1 = "hostdev_name1"

	hostdevResource0    = "vendor.com/hostdev_name0"
	hostdevResource1    = "vendor.com/hostdev_name1"
	envHostDevResource0 = "VENDOR_COM_HOSTDEV_NAME0"
	envHostDevResource1 = "VENDOR_COM_HOSTDEV_NAME1"

	hostdevPCIAddress0 = "0000:81:01.0"
	hostdevPCIAddress1 = "0000:81:01.1"

	hostdevMDEVAddress0 = "123456789-0"
	hostdevMDEVAddress1 = "123456789-1"
)

var _ = Describe("Generic Address Pool", func() {
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		vmi = &v1.VirtualMachineInstance{}
	})

	DescribeTable("creates an empty pool when no HostDevices are specified",
		func(newPool func([]v1.HostDevice) *hostdevice.AddressPool) {
			pool := newPool(vmi.Spec.Domain.Devices.HostDevices)
			expectPoolPopFailure(pool, hostdevResource0)
		},
		Entry("PCI", generic.NewPCIAddressPool),
		Entry("MDEV", generic.NewMDEVAddressPool),
	)

	DescribeTable("creates an empty pool when no resources are specified",
		func(newPool func([]v1.HostDevice) *hostdevice.AddressPool) {
			vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{{DeviceName: hostdevResource0, Name: hostdevName0}}
			pool := newPool(vmi.Spec.Domain.Devices.HostDevices)
			expectPoolPopFailure(pool, hostdevResource0)
		},
		Entry("PCI", generic.NewPCIAddressPool),
		Entry("MDEV", generic.NewMDEVAddressPool),
	)

	DescribeTable("succeeds to pop 2 addresses from same resource",
		func(newPool func([]v1.HostDevice) *hostdevice.AddressPool, prefix, address0, address1 string) {
			vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{{DeviceName: hostdevResource0, Name: hostdevName0}}
			env := []envData{newResourceEnv(prefix, envHostDevResource0, address0, address1)}
			withEnvironmentContext(env, func() {
				pool := newPool(vmi.Spec.Domain.Devices.HostDevices)
				Expect(pool.Pop(hostdevResource0)).To(Equal(address0))
				Expect(pool.Pop(hostdevResource0)).To(Equal(address1))
			})
		},
		Entry("PCI", generic.NewPCIAddressPool, v1.PCIResourcePrefix, hostdevPCIAddress0, hostdevPCIAddress1),
		Entry("MDEV", generic.NewMDEVAddressPool, v1.MDevResourcePrefix, hostdevMDEVAddress0, hostdevMDEVAddress1),
	)

	DescribeTable("succeeds to pop 2 addresses from two resources",
		func(newPool func([]v1.HostDevice) *hostdevice.AddressPool, prefix, address0, address1 string) {
			vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
				{DeviceName: hostdevResource0, Name: hostdevName0},
				{DeviceName: hostdevResource1, Name: hostdevName1},
			}
			env := []envData{
				newResourceEnv(prefix, envHostDevResource0, address0),
				newResourceEnv(prefix, envHostDevResource1, address1),
			}
			withEnvironmentContext(env, func() {
				pool := newPool(vmi.Spec.Domain.Devices.HostDevices)
				Expect(pool.Pop(hostdevResource0)).To(Equal(address0))
				Expect(pool.Pop(hostdevResource1)).To(Equal(address1))
			})
		},
		Entry("PCI", generic.NewPCIAddressPool, v1.PCIResourcePrefix, hostdevPCIAddress0, hostdevPCIAddress1),
		Entry("MDEV", generic.NewMDEVAddressPool, v1.MDevResourcePrefix, hostdevMDEVAddress0, hostdevMDEVAddress1),
	)
})

func newResourceEnv(prefix, resourceName string, addresses ...string) envData {
	resourceName = strings.ToUpper(resourceName)
	return envData{
		Name:  strings.Join([]string{prefix, resourceName}, "_"),
		Value: strings.Join(addresses, ","),
	}
}

func withEnvironmentContext(envDataList []envData, f func()) {
	for _, envVar := range envDataList {
		if os.Setenv(envVar.Name, envVar.Value) == nil {
			defer os.Unsetenv(envVar.Name)
		}
	}
	f()
}

func expectPoolPopFailure(pool *hostdevice.AddressPool, resource string) {
	address, err := pool.Pop(resource)
	ExpectWithOffset(1, err).To(HaveOccurred())
	ExpectWithOffset(1, address).To(BeEmpty())
}
