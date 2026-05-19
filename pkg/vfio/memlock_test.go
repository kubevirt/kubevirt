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

package vfio_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/vfio"
)

func TestVFIO(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/vfio")
}

func vmiWithMemoryAndDevices(memory string, devices v1.Devices) *v1.VirtualMachineInstance {
	return &v1.VirtualMachineInstance{
		Spec: v1.VirtualMachineInstanceSpec{
			Domain: v1.DomainSpec{
				Resources: v1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceMemory: resource.MustParse(memory),
					},
				},
				Devices: devices,
			},
		},
	}
}

var _ = Describe("CountDevices", func() {
	It("should return 0 for no devices", func() {
		vmi := vmiWithMemoryAndDevices("1Gi", v1.Devices{})
		Expect(vfio.CountDevices(vmi)).To(Equal(0))
	})

	It("should count GPUs", func() {
		vmi := vmiWithMemoryAndDevices("1Gi", v1.Devices{
			GPUs: []v1.GPU{{Name: "gpu0"}, {Name: "gpu1"}},
		})
		Expect(vfio.CountDevices(vmi)).To(Equal(2))
	})

	It("should count HostDevices", func() {
		vmi := vmiWithMemoryAndDevices("1Gi", v1.Devices{
			HostDevices: []v1.HostDevice{{Name: "dev0"}},
		})
		Expect(vfio.CountDevices(vmi)).To(Equal(1))
	})

	It("should count SRIOV interfaces", func() {
		vmi := vmiWithMemoryAndDevices("1Gi", v1.Devices{
			Interfaces: []v1.Interface{
				{Name: "sriov0", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}},
				{Name: "sriov1", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}},
			},
		})
		Expect(vfio.CountDevices(vmi)).To(Equal(2))
	})

	It("should not count non-SRIOV interfaces", func() {
		vmi := vmiWithMemoryAndDevices("1Gi", v1.Devices{
			Interfaces: []v1.Interface{
				{Name: "bridge0", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
			},
		})
		Expect(vfio.CountDevices(vmi)).To(Equal(0))
	})

	It("should sum GPUs, HostDevices, and SRIOV interfaces", func() {
		vmi := vmiWithMemoryAndDevices("1Gi", v1.Devices{
			GPUs:        []v1.GPU{{Name: "gpu0"}},
			HostDevices: []v1.HostDevice{{Name: "dev0"}},
			Interfaces: []v1.Interface{
				{Name: "sriov0", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}},
			},
		})
		Expect(vfio.CountDevices(vmi)).To(Equal(3))
	})
})

var _ = Describe("CalculateMemlockLimit", func() {
	const oneGiB = int64(vfio.MMIOOverheadBytes)
	const eightGiB = int64(8 * 1024 * 1024 * 1024)

	It("should return 0 for no VFIO devices", func() {
		vmi := vmiWithMemoryAndDevices("8Gi", v1.Devices{})
		Expect(vfio.CalculateMemlockLimit(vmi)).To(Equal(int64(0)))
	})

	DescribeTable("should match libvirt formula: numDevices * guestMemory + 1GiB",
		func(devices v1.Devices, expectedBytes int64) {
			vmi := vmiWithMemoryAndDevices("8Gi", devices)
			Expect(vfio.CalculateMemlockLimit(vmi)).To(Equal(expectedBytes))
		},
		Entry("1 GPU",
			v1.Devices{GPUs: []v1.GPU{{Name: "gpu0"}}},
			1*eightGiB+oneGiB),
		Entry("2 GPUs",
			v1.Devices{GPUs: []v1.GPU{{Name: "gpu0"}, {Name: "gpu1"}}},
			2*eightGiB+oneGiB),
		Entry("3 GPUs",
			v1.Devices{GPUs: []v1.GPU{{Name: "gpu0"}, {Name: "gpu1"}, {Name: "gpu2"}}},
			3*eightGiB+oneGiB),
		Entry("1 GPU + 1 HostDevice",
			v1.Devices{
				GPUs:        []v1.GPU{{Name: "gpu0"}},
				HostDevices: []v1.HostDevice{{Name: "dev0"}},
			},
			2*eightGiB+oneGiB),
		Entry("2 SRIOV interfaces",
			v1.Devices{Interfaces: []v1.Interface{
				{Name: "sriov0", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}},
				{Name: "sriov1", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}},
			}},
			2*eightGiB+oneGiB),
	)

	It("should use guest memory when set", func() {
		guestMem := resource.MustParse("4Gi")
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{Guest: &guestMem},
					Devices: v1.Devices{
						GPUs: []v1.GPU{{Name: "gpu0"}},
					},
				},
			},
		}
		fourGiB := int64(4 * 1024 * 1024 * 1024)
		Expect(vfio.CalculateMemlockLimit(vmi)).To(Equal(1*fourGiB + oneGiB))
	})
})

var _ = Describe("CalculateMemlockExtraBytes", func() {
	const eightGiB = int64(8 * 1024 * 1024 * 1024)

	It("should return 0 for no VFIO devices", func() {
		vmi := vmiWithMemoryAndDevices("8Gi", v1.Devices{})
		Expect(vfio.CalculateMemlockExtraBytes(vmi)).To(Equal(int64(0)))
	})

	It("should return 0 for single device", func() {
		vmi := vmiWithMemoryAndDevices("8Gi", v1.Devices{
			GPUs: []v1.GPU{{Name: "gpu0"}},
		})
		Expect(vfio.CalculateMemlockExtraBytes(vmi)).To(Equal(int64(0)))
	})

	DescribeTable("should return (N-1) * guestMemory for N > 1",
		func(devices v1.Devices, expectedBytes int64) {
			vmi := vmiWithMemoryAndDevices("8Gi", devices)
			Expect(vfio.CalculateMemlockExtraBytes(vmi)).To(Equal(expectedBytes))
		},
		Entry("2 GPUs",
			v1.Devices{GPUs: []v1.GPU{{Name: "gpu0"}, {Name: "gpu1"}}},
			1*eightGiB),
		Entry("3 GPUs",
			v1.Devices{GPUs: []v1.GPU{{Name: "gpu0"}, {Name: "gpu1"}, {Name: "gpu2"}}},
			2*eightGiB),
		Entry("1 GPU + 1 HostDevice",
			v1.Devices{
				GPUs:        []v1.GPU{{Name: "gpu0"}},
				HostDevices: []v1.HostDevice{{Name: "dev0"}},
			},
			1*eightGiB),
	)
})
