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
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/gpu"
)

type envData struct {
	Name  string
	Value string
}

const (
	gpuName0 = "gpu_name0"
	gpuName1 = "gpu_name1"

	gpuResource0    = "vendor.com/gpu_name0"
	gpuResource1    = "vendor.com/gpu_name1"
	envGPUResource0 = "VENDOR_COM_GPU_NAME0"
	envGPUResource1 = "VENDOR_COM_GPU_NAME1"

	gpuPCIAddress0 = "0000:81:01.0"
	gpuPCIAddress1 = "0000:81:01.1"

	gpuMDEVAddress0 = "123456789-0"
	gpuMDEVAddress1 = "123456789-1"
)

var _ = Describe("GPU Address Pool", func() {
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		vmi = &v1.VirtualMachineInstance{}
	})

	DescribeTable("creates an empty pool when no GPUs are specified",
		func(newPool func([]v1.GPU) *hostdevice.AddressPool) {
			pool := newPool(vmi.Spec.Domain.Devices.GPUs)
			expectPoolPopFailure(pool, gpuResource0)
		},
		Entry("PCI", gpu.NewPCIAddressPool),
		Entry("MDEV", gpu.NewMDEVAddressPool),
	)

	DescribeTable("creates an empty pool when no resources are specified",
		func(newPool func([]v1.GPU) *hostdevice.AddressPool) {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{{DeviceName: gpuResource0, Name: gpuName0}}
			pool := newPool(vmi.Spec.Domain.Devices.GPUs)
			expectPoolPopFailure(pool, gpuResource0)
		},
		Entry("PCI", gpu.NewPCIAddressPool),
		Entry("MDEV", gpu.NewMDEVAddressPool),
	)

	DescribeTable("succeeds to pop 2 addresses from same resource",
		func(newPool func([]v1.GPU) *hostdevice.AddressPool, prefix, address0, address1 string) {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{{DeviceName: gpuResource0, Name: gpuName0}}
			env := []envData{newResourceEnv(prefix, envGPUResource0, address0, address1)}
			withEnvironmentContext(env, func() {
				pool := newPool(vmi.Spec.Domain.Devices.GPUs)
				Expect(pool.Pop(gpuResource0)).To(Equal(address0))
				Expect(pool.Pop(gpuResource0)).To(Equal(address1))
			})
		},
		Entry("PCI", gpu.NewPCIAddressPool, v1.PCIResourcePrefix, gpuPCIAddress0, gpuPCIAddress1),
		Entry("MDEV", gpu.NewMDEVAddressPool, v1.MDevResourcePrefix, gpuMDEVAddress0, gpuMDEVAddress1),
	)

	DescribeTable("succeeds to pop 2 addresses from two resources",
		func(newPool func([]v1.GPU) *hostdevice.AddressPool, prefix, address0, address1 string) {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{DeviceName: gpuResource0, Name: gpuName0},
				{DeviceName: gpuResource1, Name: gpuName1},
			}
			env := []envData{
				newResourceEnv(prefix, envGPUResource0, address0),
				newResourceEnv(prefix, envGPUResource1, address1),
			}
			withEnvironmentContext(env, func() {
				pool := newPool(vmi.Spec.Domain.Devices.GPUs)
				Expect(pool.Pop(gpuResource0)).To(Equal(address0))
				Expect(pool.Pop(gpuResource1)).To(Equal(address1))
			})
		},
		Entry("PCI", gpu.NewPCIAddressPool, v1.PCIResourcePrefix, gpuPCIAddress0, gpuPCIAddress1),
		Entry("MDEV", gpu.NewMDEVAddressPool, v1.MDevResourcePrefix, gpuMDEVAddress0, gpuMDEVAddress1),
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
