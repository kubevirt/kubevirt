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

package kvm

import (
	"fmt"
	"strconv"

	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/pointer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("GetMemoryOverhead calculation", func() {
	// VirtLauncherMonitorOverhead + VirtLauncherOverhead + VirtlogdOverhead + VirtqemudOverhead + QemuOverhead + IothreadsOverhead
	const staticOverheadString = "223Mi"
	var (
		vmi                     *v1.VirtualMachineInstance
		staticOverhead          *resource.Quantity
		baseOverhead            *resource.Quantity
		coresOverhead           *resource.Quantity
		videoRAMOverhead        *resource.Quantity
		cpuArchOverhead         *resource.Quantity
		vfioOverhead            *resource.Quantity
		downwardmetricsOverhead *resource.Quantity
		sevOverhead             *resource.Quantity
		tpmOverhead             *resource.Quantity
	)

	BeforeEach(func() {
		vmi = &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "test-vmi"},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Resources: v1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceMemory: resource.MustParse("1Gi"),
						},
						Limits: k8sv1.ResourceList{},
					},
				},
			},
		}
		staticOverhead = pointer.P(resource.MustParse(staticOverheadString))
		// MemoryReq / 512bit
		baseOverhead = pointer.P(resource.MustParse("7Mi"))
		coresOverhead = pointer.P(resource.MustParse("8Mi"))
		videoRAMOverhead = pointer.P(resource.MustParse("32Mi"))
		cpuArchOverhead = pointer.P(resource.MustParse("128Mi"))
		vfioOverhead = pointer.P(resource.MustParse("1Gi"))
		downwardmetricsOverhead = pointer.P(resource.MustParse("1Mi"))
		sevOverhead = pointer.P(resource.MustParse("256Mi"))
		tpmOverhead = pointer.P(resource.MustParse("53Mi"))
	})

	When("the vmi is not requesting any specific device or cpu or whatever", func() {
		It("should return base overhead+static+8Mi", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			// 8Mi*1core(default)
			expected.Add(*coresOverhead)
			overhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the vmi requests the specific cpu", func() {
		BeforeEach(func() {
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:   2,
				Threads: 2,
				Sockets: 2,
			}
		})

		It("should adjust overhead based on the cores/threads/sockets", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			// (2cores* 2threads *2sockets)
			value := coresOverhead.Value() * 8
			expected.Add(*resource.NewQuantity(value, coresOverhead.Format))
			overhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the vmi requests cpu resource", func() {
		DescribeTable("should adjust overhead", func(requests, limits string, coresMultiplier int) {
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse(requests)
			if limits != "" {
				vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = resource.MustParse(limits)
			}

			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			value := coresOverhead.Value() * int64(coresMultiplier)
			expected.Add(*resource.NewQuantity(value, coresOverhead.Format))
			overhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		},
			Entry("based on the limits if both requests and limits are provided", "3", "5", 5),
			Entry("based on the requests if only requests are provided", "3", "", 3),
		)

	})

	When("the vmi does not require auto attach graphics device", func() {
		BeforeEach(func() {
			vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = pointer.P(false)
		})

		It("should not add videoRAMOverhead", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*coresOverhead)
			overhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the cpu arch is arm64", func() {
		It("should add arm64 overhead", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(*cpuArchOverhead)
			overhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, "arm64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the vmi requests a VFIO device", func() {
		DescribeTable("should add vfio overhead", func(devices v1.Devices) {
			vmi.Spec.Domain.Devices = devices
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(*vfioOverhead)
			overhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		},
			Entry("with hostDEV", v1.Devices{HostDevices: []v1.HostDevice{{Name: "test"}}}),
			Entry("with GPU", v1.Devices{GPUs: []v1.GPU{{Name: "test"}}}),
			Entry("with SRIOV", v1.Devices{Interfaces: []v1.Interface{{Name: "test", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}}}),
		)
	})

	When("the vmi has a downward metrics volume", func() {
		BeforeEach(func() {
			vmi.Spec.Volumes = []v1.Volume{{VolumeSource: v1.VolumeSource{DownwardMetrics: &v1.DownwardMetricsVolumeSource{}}}}
		})
		It("should add downwardMetrics overhead", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(*downwardmetricsOverhead)
			overhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the vmi has probes fields", func() {
		DescribeTable("should add probes overhead", func(livenessProbe, readinessProbe *v1.Probe, probeOverhead resource.Quantity) {
			vmi.Spec.LivenessProbe = livenessProbe
			vmi.Spec.ReadinessProbe = readinessProbe
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(probeOverhead)

			overhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		},
			Entry("with livenessProbe only", &v1.Probe{Handler: v1.Handler{Exec: &k8sv1.ExecAction{}}}, nil, resource.MustParse("110Mi")),
			Entry("with readinessProbe only", nil, &v1.Probe{Handler: v1.Handler{Exec: &k8sv1.ExecAction{}}}, resource.MustParse("110Mi")),
			Entry("with both readinessProbe adn livenessProbe", &v1.Probe{Handler: v1.Handler{Exec: &k8sv1.ExecAction{}}}, &v1.Probe{Handler: v1.Handler{Exec: &k8sv1.ExecAction{}}}, resource.MustParse("120Mi")),
		)
	})

	When("the vmi requests AMD SEV", func() {
		BeforeEach(func() {
			vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
				SEV: &v1.SEV{},
			}
		})

		It("should add SEV overhead", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(*sevOverhead)
			overhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the vmi requests TPM device", func() {
		BeforeEach(func() {
			vmi.Spec.Domain.Devices = v1.Devices{
				TPM: &v1.TPMDevice{},
			}
		})

		It("should add SEV overhead", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(*tpmOverhead)
			overhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the additionalOverheadRatio is provided", func() {
		DescribeTable("should adjust the overhead using the given ratio", func(additionalOverheadRatio string, expectParseError bool) {
			base := resource.NewScaledQuantity(0, resource.Kilo)
			base.Add(*baseOverhead)
			base.Add(*staticOverhead)
			base.Add(*videoRAMOverhead)
			base.Add(*coresOverhead)
			var expected resource.Quantity
			if expectParseError {
				expected = *base
			} else {
				ratio, _ := strconv.ParseFloat(additionalOverheadRatio, 64)
				expected = multiplyMemory(*base, ratio)
			}

			overhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, "amd64", pointer.P(additionalOverheadRatio))
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		},
			Entry("with the given value if the given value is a float", "3.2", false),
			Entry("with no value if the given value is not a float", "no_float", true),
		)
	})

	When("the vmi is requesting dedicated CPU or wants to have QOSGuaranteed", func() {
		DescribeTable("should add 100Mi of overhead", func(requestDedicatedCPU, wantsQOSGuaranteed bool) {
			vmi.Spec.Domain.CPU = &v1.CPU{Cores: 1}
			if requestDedicatedCPU {
				vmi.Spec.Domain.CPU.DedicatedCPUPlacement = true
			}
			if wantsQOSGuaranteed {
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")
				vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("1Gi")
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("4")
				vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("4")
			}
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(resource.MustParse("100Mi"))

			overhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		},
			Entry("with DedicatedCPU", true, false),
			Entry("when wants QOSGuaranteed", false, true),
		)
	})

	When("reservedOverhead value is provided", func() {
		DescribeTable("should be considered as a resource overhead", func(overhead *resource.Quantity) {
			cpuArch := "amd64"
			vmi.Spec.Domain.Memory = &v1.Memory{}
			vmi.Spec.Domain.Memory.ReservedOverhead = &v1.ReservedOverhead{
				AddedOverhead: overhead,
			}
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(*overhead)
			result := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, cpuArch, nil)
			Expect(result.Value()).To(BeEquivalentTo(expected.Value()))

		},
			Entry("with some value", resource.NewScaledQuantity(100, resource.Giga)),
			Entry("with zero value", &resource.Quantity{}),
		)
	})

	Context("Template with guest-to-request memory headroom", func() {
		defaultArch := "amd64"
		newVmi := func() *v1.VirtualMachineInstance {
			vmi := api.NewMinimalVMI("test-vmi")

			vmi.Spec.Domain.Resources = v1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("1G"),
					k8sv1.ResourceCPU:    resource.MustParse("1"),
				},
			}

			return vmi
		}

		DescribeTable("should add guest-to-memory headroom when configured with ratio", func(ratioStr string) {
			vmi := newVmi()

			ratio, err := strconv.ParseFloat(ratioStr, 64)
			Expect(err).ToNot(HaveOccurred())

			originalOverhead := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, defaultArch, nil)
			actualOverheadWithHeadroom := NewKvmHypervisorBackend().GetMemoryOverhead(vmi, defaultArch, pointer.P(ratioStr))
			expectedOverheadWithHeadroom := multiplyMemory(originalOverhead, ratio)

			const errFmt = "overhead without headroom: %s, ratio: %s, actual overhead with headroom: %s, expected overhead with headroom: %s"
			Expect(newVmi()).To(Equal(vmi), "vmi object should not be changed")
			Expect(actualOverheadWithHeadroom.Cmp(expectedOverheadWithHeadroom)).To(Equal(0),
				fmt.Sprintf(errFmt, originalOverhead.String(), ratioStr, actualOverheadWithHeadroom.String(), expectedOverheadWithHeadroom.String()))
		},
			Entry("2.332", "2.332"),
			Entry("1.234", "1.234"),
			Entry("1.0", "1.0"),
		)
	})
})
