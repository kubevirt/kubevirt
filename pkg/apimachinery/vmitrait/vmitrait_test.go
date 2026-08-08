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

package vmitrait_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/vmitrait"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("VMI traits", func() {
	Context("IsNonRoot", func() {
		It("should return false when annotation is absent and RuntimeUser is 0", func() {
			Expect(vmitrait.IsNonRoot(&v1.VirtualMachineInstance{})).To(BeFalse())
		})

		DescribeTable("should return true", func(annotations map[string]string, runtimeUser uint64) {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Annotations = annotations
			vmi.Status.RuntimeUser = runtimeUser
			Expect(vmitrait.IsNonRoot(vmi)).To(BeTrue())
		},
			Entry("when the deprecated non-root annotation is present",
				map[string]string{v1.DeprecatedNonRootVMIAnnotation: ""},
				uint64(0),
			),
			Entry("when RuntimeUser is non-zero",
				nil,
				uint64(107),
			),
			Entry("when both annotation and non-zero RuntimeUser are present",
				map[string]string{v1.DeprecatedNonRootVMIAnnotation: ""},
				uint64(107),
			),
		)
	})

	Context("HasVFIO", func() {
		DescribeTable("should return true when a VFIO device is present", func(vmi *v1.VirtualMachineInstance) {
			Expect(vmitrait.HasVFIO(vmi)).To(BeTrue())
		},
			Entry("with a GPU",
				libvmi.New(libvmi.WithGPU(v1.GPU{Name: "gpu1", DeviceName: "nvidia.com/gpu"})),
			),
			Entry("with a host device",
				libvmi.New(libvmi.WithHostDevice(v1.HostDevice{Name: "dev1", DeviceName: "vendor.com/device"})),
			),
			Entry("with an SRIOV interface",
				libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding("sriov1")),
					libvmi.WithNetwork(&v1.Network{Name: "sriov1", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "sriov-net"}}}),
				),
			),
		)

		It("should return false when no VFIO devices are present", func() {
			Expect(vmitrait.HasVFIO(libvmi.New())).To(BeFalse())
		})
	})
})
