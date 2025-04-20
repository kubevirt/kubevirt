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
 */

package apply_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("instancetype.spec.CPU and preference.spec.CPU", func() {
	var (
		vmi              *virtv1.VirtualMachineInstance
		instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec
		preferenceSpec   *v1beta1.VirtualMachinePreferenceSpec

		vmiApplier = apply.NewVMIApplier()
		field      = k8sfield.NewPath("spec", "template", "spec")
	)

	BeforeEach(func() {
		vmi = libvmi.New()

		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			CPU: v1beta1.CPUInstancetype{
				Guest:                 uint32(2),
				Model:                 pointer.P("host-passthrough"),
				DedicatedCPUPlacement: pointer.P(true),
				IsolateEmulatorThread: pointer.P(true),
				NUMA: &virtv1.NUMA{
					GuestMappingPassthrough: &virtv1.NUMAGuestMappingPassthrough{},
				},
				Realtime: &virtv1.Realtime{
					Mask: "0-3,^1",
				},
				MaxSockets: pointer.P(uint32(6)),
			},
		}

		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			CPU: &v1beta1.CPUPreferences{},
		}
	})

	It("should default to PreferSockets", func() {
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetypeSpec.CPU.Guest))
		Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
		Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
		Expect(vmi.Spec.Domain.CPU.Model).To(Equal(*instancetypeSpec.CPU.Model))
		Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(*instancetypeSpec.CPU.DedicatedCPUPlacement))
		Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(*instancetypeSpec.CPU.IsolateEmulatorThread))
		Expect(vmi.Spec.Domain.CPU.NUMA).To(HaveValue(Equal(*instancetypeSpec.CPU.NUMA)))
		Expect(vmi.Spec.Domain.CPU.Realtime).To(HaveValue(Equal(*instancetypeSpec.CPU.Realtime)))
		Expect(vmi.Spec.Domain.CPU.MaxSockets).To(Equal(*instancetypeSpec.CPU.MaxSockets))
	})

	It("should default to Sockets, when instancetype is used with PreferAny", func() {
		preferenceSpec.CPU.PreferredCPUTopology = pointer.P(v1beta1.Any)

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetypeSpec.CPU.Guest))
		Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
		Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
	})

	It("should apply in full with PreferCores selected", func() {
		preferenceSpec.CPU.PreferredCPUTopology = pointer.P(v1beta1.Cores)

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(1)))
		Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(instancetypeSpec.CPU.Guest))
		Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
		Expect(vmi.Spec.Domain.CPU.Model).To(Equal(*instancetypeSpec.CPU.Model))
		Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(*instancetypeSpec.CPU.DedicatedCPUPlacement))
		Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(*instancetypeSpec.CPU.IsolateEmulatorThread))
		Expect(vmi.Spec.Domain.CPU.NUMA).To(HaveValue(Equal(*instancetypeSpec.CPU.NUMA)))
		Expect(vmi.Spec.Domain.CPU.Realtime).To(HaveValue(Equal(*instancetypeSpec.CPU.Realtime)))
	})

	It("should apply in full with PreferThreads selected", func() {
		preferenceSpec.CPU.PreferredCPUTopology = pointer.P(v1beta1.Threads)

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(1)))
		Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
		Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(instancetypeSpec.CPU.Guest))
		Expect(vmi.Spec.Domain.CPU.Model).To(Equal(*instancetypeSpec.CPU.Model))
		Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(*instancetypeSpec.CPU.DedicatedCPUPlacement))
		Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(*instancetypeSpec.CPU.IsolateEmulatorThread))
		Expect(vmi.Spec.Domain.CPU.NUMA).To(HaveValue(Equal(*instancetypeSpec.CPU.NUMA)))
		Expect(vmi.Spec.Domain.CPU.Realtime).To(HaveValue(Equal(*instancetypeSpec.CPU.Realtime)))
	})

	It("should apply in full with PreferSockets selected", func() {
		preferenceSpec.CPU.PreferredCPUTopology = pointer.P(v1beta1.Sockets)

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetypeSpec.CPU.Guest))
		Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
		Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
		Expect(vmi.Spec.Domain.CPU.Model).To(Equal(*instancetypeSpec.CPU.Model))
		Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(*instancetypeSpec.CPU.DedicatedCPUPlacement))
		Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(*instancetypeSpec.CPU.IsolateEmulatorThread))
		Expect(vmi.Spec.Domain.CPU.NUMA).To(HaveValue(Equal(*instancetypeSpec.CPU.NUMA)))
		Expect(vmi.Spec.Domain.CPU.Realtime).To(HaveValue(Equal(*instancetypeSpec.CPU.Realtime)))
	})

	Context("with PreferSpread", func() {
		DescribeTable("should spread", func(vCPUs uint32, preferenceSpec v1beta1.VirtualMachinePreferenceSpec, expectedCPU virtv1.CPU) {
			instancetypeSpec.CPU.Guest = vCPUs
			if preferenceSpec.CPU == nil {
				preferenceSpec.CPU = &v1beta1.CPUPreferences{}
			}
			preferenceSpec.CPU.PreferredCPUTopology = pointer.P(v1beta1.Spread)

			Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, &preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
			Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(expectedCPU.Sockets))
			Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(expectedCPU.Cores))
			Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(expectedCPU.Threads))
		},
			Entry("by default to SocketsCores with a default topology for 1 vCPU",
				uint32(1),
				v1beta1.VirtualMachinePreferenceSpec{},
				virtv1.CPU{Sockets: 1, Cores: 1, Threads: 1},
			),
			Entry("by default to SocketsCores with 2 vCPUs and a default ratio of 1:2:1",
				uint32(2),
				v1beta1.VirtualMachinePreferenceSpec{},
				virtv1.CPU{Sockets: 1, Cores: 2, Threads: 1},
			),
			Entry("by default to SocketsCores with 4 vCPUs and a default ratio of 1:2:1",
				uint32(4),
				v1beta1.VirtualMachinePreferenceSpec{},
				virtv1.CPU{Sockets: 2, Cores: 2, Threads: 1},
			),
			Entry("by default to SocketsCores with 6 vCPUs and a default ratio of 1:2:1",
				uint32(6),
				v1beta1.VirtualMachinePreferenceSpec{},
				virtv1.CPU{Sockets: 3, Cores: 2, Threads: 1},
			),
			Entry("by default to SocketsCores with 8 vCPUs and a default ratio of 1:2:1",
				uint32(8),
				v1beta1.VirtualMachinePreferenceSpec{},
				virtv1.CPU{Sockets: 4, Cores: 2, Threads: 1},
			),
			Entry("by default to SocketsCores with 3 vCPUs and a ratio of 1:3:1",
				uint32(3),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(3)),
						},
					},
				},
				virtv1.CPU{Sockets: 1, Cores: 3, Threads: 1},
			),
			Entry("by default to SocketsCores with 6 vCPUs and a ratio of 1:3:1",
				uint32(6),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(3)),
						},
					},
				},
				virtv1.CPU{Sockets: 2, Cores: 3, Threads: 1},
			),
			Entry("by default to SocketsCores with 9 vCPUs and a ratio of 1:3:1",
				uint32(9),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(3)),
						},
					},
				},
				virtv1.CPU{Sockets: 3, Cores: 3, Threads: 1},
			),
			Entry("by default to SocketsCores with 12 vCPUs and a ratio of 1:3:1",
				uint32(12),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(3)),
						},
					},
				},
				virtv1.CPU{Sockets: 4, Cores: 3, Threads: 1},
			),
			Entry("by default to SocketsCores with 4 vCPUs and a ratio of 1:4:1",
				uint32(4),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(4)),
						},
					},
				},
				virtv1.CPU{Sockets: 1, Cores: 4, Threads: 1},
			),
			Entry("by default to SocketsCores with 8 vCPUs and a ratio of 1:4:1",
				uint32(8),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(4)),
						},
					},
				},
				virtv1.CPU{Sockets: 2, Cores: 4, Threads: 1},
			),
			Entry("by default to SocketsCores with 12 vCPUs and a ratio of 1:4:1",
				uint32(12),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(4)),
						},
					},
				},
				virtv1.CPU{Sockets: 3, Cores: 4, Threads: 1},
			),
			Entry("by default to SocketsCores with 16 vCPUs and a ratio of 1:4:1",
				uint32(16),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(4)),
						},
					},
				},
				virtv1.CPU{Sockets: 4, Cores: 4, Threads: 1},
			),
			Entry("to SocketsCoresThreads with 4 vCPUs and a default ratio of 1:2:2",
				uint32(4),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
						},
					},
				},
				virtv1.CPU{Sockets: 1, Cores: 2, Threads: 2},
			),
			Entry("to SocketsCoresThreads with 8 vCPUs and a default ratio of 1:2:2",
				uint32(8),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
						},
					},
				},
				virtv1.CPU{Sockets: 2, Cores: 2, Threads: 2},
			),
			Entry("to SocketsCoresThreads with 12 vCPUs and a default ratio of 1:2:2",
				uint32(12),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
						},
					},
				},
				virtv1.CPU{Sockets: 3, Cores: 2, Threads: 2},
			),
			Entry("to SocketsCoresThreads with 16 vCPUs and a default ratio of 1:2:2",
				uint32(16),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
						},
					},
				},
				virtv1.CPU{Sockets: 4, Cores: 2, Threads: 2},
			),
			Entry("to SocketsCoresThreads with 6 vCPUs and a ratio of 1:3:2",
				uint32(6),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
							Ratio:  pointer.P(uint32(3)),
						},
					},
				},
				virtv1.CPU{Sockets: 1, Cores: 3, Threads: 2},
			),
			Entry("to SocketsCoresThreads with 12 vCPUs and a ratio of 1:3:2",
				uint32(12),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
							Ratio:  pointer.P(uint32(3)),
						},
					},
				},
				virtv1.CPU{Sockets: 2, Cores: 3, Threads: 2},
			),
			Entry("to SocketsCoresThreads with 18 vCPUs and a ratio of 1:3:2",
				uint32(18),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
							Ratio:  pointer.P(uint32(3)),
						},
					},
				},
				virtv1.CPU{Sockets: 3, Cores: 3, Threads: 2},
			),
			Entry("to SocketsCoresThreads with 24 vCPUs and a ratio of 1:3:2",
				uint32(24),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
							Ratio:  pointer.P(uint32(3)),
						},
					},
				},
				virtv1.CPU{Sockets: 4, Cores: 3, Threads: 2},
			),
			Entry("to SocketsCoresThreads with 8 vCPUs and a ratio of 1:4:2",
				uint32(8),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
							Ratio:  pointer.P(uint32(4)),
						},
					},
				},
				virtv1.CPU{Sockets: 1, Cores: 4, Threads: 2},
			),
			Entry("to SocketsCoresThreads with 16 vCPUs and a ratio of 1:4:2",
				uint32(16),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
							Ratio:  pointer.P(uint32(4)),
						},
					},
				},
				virtv1.CPU{Sockets: 2, Cores: 4, Threads: 2},
			),
			Entry("to SocketsCoresThreads with 24 vCPUs and a ratio of 1:4:2",
				uint32(24),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
							Ratio:  pointer.P(uint32(4)),
						},
					},
				},
				virtv1.CPU{Sockets: 3, Cores: 4, Threads: 2},
			),
			Entry("to SocketsCoresThreads with 36 vCPUs and a ratio of 1:4:2",
				uint32(36),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
							Ratio:  pointer.P(uint32(4)),
						},
					},
				},
				virtv1.CPU{Sockets: 4, Cores: 4, Threads: 2},
			),
			Entry("to CoresThreads with 2 vCPUs and a default ratio of 1:2",
				uint32(2),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
				virtv1.CPU{Sockets: 1, Cores: 1, Threads: 2},
			),
			Entry("to CoresThreads with 4 vCPUs and a default ratio of 1:2",
				uint32(4),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
				virtv1.CPU{Sockets: 1, Cores: 2, Threads: 2},
			),
			Entry("to CoresThreads with 6 vCPUs and a default ratio of 1:2",
				uint32(6),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
				virtv1.CPU{Sockets: 1, Cores: 3, Threads: 2},
			),
			Entry("to CoresThreads with 8 vCPUs and a default ratio of 1:2",
				uint32(8),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
				virtv1.CPU{Sockets: 1, Cores: 4, Threads: 2},
			),
		)
	})

	It("should return a conflict if vmi.Spec.Domain.CPU already defined", func() {
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			CPU: v1beta1.CPUInstancetype{
				Guest: uint32(2),
			},
		}

		vmi.Spec.Domain.CPU = &virtv1.CPU{
			Cores:   4,
			Sockets: 1,
			Threads: 1,
		}

		conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
		Expect(conflicts).To(HaveLen(3))
		Expect(conflicts).To(Equal(conflict.Conflicts{
			conflict.New("spec", "template", "spec", "domain", "cpu", "sockets"),
			conflict.New("spec", "template", "spec", "domain", "cpu", "cores"),
			conflict.New("spec", "template", "spec", "domain", "cpu", "threads"),
		}))
	})

	It("should return a conflict if vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] already defined", func() {
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			CPU: v1beta1.CPUInstancetype{
				Guest: uint32(2),
			},
		}

		vmi.Spec.Domain.Resources = virtv1.ResourceRequirements{
			Requests: k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("1"),
			},
		}

		conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
		Expect(conflicts).To(HaveLen(1))
		Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.resources.requests.cpu"))
	})

	It("should return a conflict if vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] already defined", func() {
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			CPU: v1beta1.CPUInstancetype{
				Guest: uint32(2),
			},
		}

		vmi.Spec.Domain.Resources = virtv1.ResourceRequirements{
			Limits: k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("1"),
			},
		}

		conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
		Expect(conflicts).To(HaveLen(1))
		Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.resources.limits.cpu"))
	})

	It("should apply PreferredCPUFeatures", func() {
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			CPU: &v1beta1.CPUPreferences{
				PreferredCPUFeatures: []virtv1.CPUFeature{
					{
						Name:   "foo",
						Policy: "require",
					},
					{
						Name:   "bar",
						Policy: "force",
					},
				},
			},
		}
		vmi.Spec.Domain.CPU = &virtv1.CPU{
			Features: []virtv1.CPUFeature{
				{
					Name:   "bar",
					Policy: "optional",
				},
			},
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Domain.CPU.Features).To(HaveLen(2))
		Expect(vmi.Spec.Domain.CPU.Features).To(ContainElements([]virtv1.CPUFeature{
			{
				Name:   "foo",
				Policy: "require",
			},
			{
				Name:   "bar",
				Policy: "optional",
			},
		}))
	})
})
