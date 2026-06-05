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

package compute_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("CPU Domain Configurator", func() {
	const (
		hotplugSupported         = true
		requiresMPXCPUValidation = true
	)

	Context("CPU topology", func() {
		DescribeTable("should set topology and VCPU count from VMI CPU spec",
			func(vmi *v1.VirtualMachineInstance, expectedDomain api.Domain) {
				var domain api.Domain

				configurator := compute.NewCPUDomainConfigurator(!hotplugSupported, !requiresMPXCPUValidation)
				Expect(configurator.Configure(vmi, &domain)).To(Succeed())

				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("with no CPU spec",
				libvmi.New(),
				api.Domain{Spec: api.DomainSpec{
					CPU:  api.CPU{Mode: v1.CPUModeHostModel, Topology: &api.CPUTopology{Sockets: 1, Cores: 1, Threads: 1}},
					VCPU: &api.VCPU{Placement: "static", CPUs: 1},
				}},
			),
			Entry("with explicit cores, threads, sockets",
				libvmi.New(libvmi.WithCPUCount(4, 2, 6)),
				api.Domain{Spec: api.DomainSpec{
					CPU:  api.CPU{Mode: v1.CPUModeHostModel, Topology: &api.CPUTopology{Sockets: 6, Cores: 4, Threads: 2}},
					VCPU: &api.VCPU{Placement: "static", CPUs: 48},
				}},
			),
			Entry("with CPU resource request only",
				libvmi.New(libvmi.WithCPURequest("4")),
				api.Domain{Spec: api.DomainSpec{
					CPU:  api.CPU{Mode: v1.CPUModeHostModel, Topology: &api.CPUTopology{Sockets: 4, Cores: 1, Threads: 1}},
					VCPU: &api.VCPU{Placement: "static", CPUs: 4},
				}},
			),
			Entry("with CPU resource limit only",
				libvmi.New(libvmi.WithCPULimit("8")),
				api.Domain{Spec: api.DomainSpec{
					CPU:  api.CPU{Mode: v1.CPUModeHostModel, Topology: &api.CPUTopology{Sockets: 8, Cores: 1, Threads: 1}},
					VCPU: &api.VCPU{Placement: "static", CPUs: 8},
				}},
			),
		)
	})

	Context("CPU model", func() {
		DescribeTable("should set CPU mode and model",
			func(vmi *v1.VirtualMachineInstance, expectedDomain api.Domain) {
				var domain api.Domain

				configurator := compute.NewCPUDomainConfigurator(!hotplugSupported, !requiresMPXCPUValidation)
				Expect(configurator.Configure(vmi, &domain)).To(Succeed())

				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("defaults to host-model when model is empty",
				libvmi.New(libvmi.WithCPUCount(1, 1, 1)),
				api.Domain{Spec: api.DomainSpec{
					CPU:  api.CPU{Mode: v1.CPUModeHostModel, Topology: &api.CPUTopology{Sockets: 1, Cores: 1, Threads: 1}},
					VCPU: &api.VCPU{Placement: "static", CPUs: 1},
				}},
			),
			Entry("host-model sets mode directly",
				libvmi.New(libvmi.WithCPUModel(v1.CPUModeHostModel)),
				api.Domain{Spec: api.DomainSpec{
					CPU:  api.CPU{Mode: v1.CPUModeHostModel, Topology: &api.CPUTopology{Sockets: 1, Cores: 1, Threads: 1}},
					VCPU: &api.VCPU{Placement: "static", CPUs: 1},
				}},
			),
			Entry("host-passthrough sets mode directly",
				libvmi.New(libvmi.WithCPUModel(v1.CPUModeHostPassthrough)),
				api.Domain{Spec: api.DomainSpec{
					CPU:  api.CPU{Mode: v1.CPUModeHostPassthrough, Topology: &api.CPUTopology{Sockets: 1, Cores: 1, Threads: 1}},
					VCPU: &api.VCPU{Placement: "static", CPUs: 1},
				}},
			),
			Entry("custom model sets mode to custom with model name",
				libvmi.New(libvmi.WithCPUModel("Skylake-Server")),
				api.Domain{Spec: api.DomainSpec{
					CPU:  api.CPU{Mode: "custom", Model: "Skylake-Server", Topology: &api.CPUTopology{Sockets: 1, Cores: 1, Threads: 1}},
					VCPU: &api.VCPU{Placement: "static", CPUs: 1},
				}},
			),
		)
	})

	Context("CPU features", func() {
		It("should set features from VMI spec", func() {
			vmi := libvmi.New(
				libvmi.WithCPUModel("Skylake-Server"),
				libvmi.WithCPUFeature("avx2", "require"),
				libvmi.WithCPUFeature("vmx", "disable"),
			)
			var domain api.Domain

			configurator := compute.NewCPUDomainConfigurator(!hotplugSupported, !requiresMPXCPUValidation)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{Spec: api.DomainSpec{
				CPU: api.CPU{
					Mode:     "custom",
					Model:    "Skylake-Server",
					Topology: &api.CPUTopology{Sockets: 1, Cores: 1, Threads: 1},
					Features: []api.CPUFeature{
						{Name: "avx2", Policy: "require"},
						{Name: "vmx", Policy: "disable"},
					},
				},
				VCPU: &api.VCPU{Placement: "static", CPUs: 1},
			}}
			Expect(domain).To(Equal(expectedDomain))
		})
	})

	Context("MPX CPU validation", func() {
		DescribeTable("should add mpx disable feature only when required",
			func(vmi *v1.VirtualMachineInstance, requiresMPX bool, expectedDomain api.Domain) {
				var domain api.Domain

				configurator := compute.NewCPUDomainConfigurator(!hotplugSupported, requiresMPX)
				Expect(configurator.Configure(vmi, &domain)).To(Succeed())

				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("adds mpx disable for custom model when MPX validation is required",
				libvmi.New(libvmi.WithCPUModel("Skylake-Server")),
				requiresMPXCPUValidation,
				api.Domain{Spec: api.DomainSpec{
					CPU: api.CPU{
						Mode:     "custom",
						Model:    "Skylake-Server",
						Topology: &api.CPUTopology{Sockets: 1, Cores: 1, Threads: 1},
						Features: []api.CPUFeature{{Name: "mpx", Policy: "disable"}},
					},
					VCPU: &api.VCPU{Placement: "static", CPUs: 1},
				}},
			),
			Entry("does not add mpx for host-model",
				libvmi.New(libvmi.WithCPUModel(v1.CPUModeHostModel)),
				requiresMPXCPUValidation,
				api.Domain{Spec: api.DomainSpec{
					CPU:  api.CPU{Mode: v1.CPUModeHostModel, Topology: &api.CPUTopology{Sockets: 1, Cores: 1, Threads: 1}},
					VCPU: &api.VCPU{Placement: "static", CPUs: 1},
				}},
			),
			Entry("does not add mpx for host-passthrough",
				libvmi.New(libvmi.WithCPUModel(v1.CPUModeHostPassthrough)),
				requiresMPXCPUValidation,
				api.Domain{Spec: api.DomainSpec{
					CPU:  api.CPU{Mode: v1.CPUModeHostPassthrough, Topology: &api.CPUTopology{Sockets: 1, Cores: 1, Threads: 1}},
					VCPU: &api.VCPU{Placement: "static", CPUs: 1},
				}},
			),
			Entry("does not add mpx when user already specifies it",
				libvmi.New(
					libvmi.WithCPUModel("Skylake-Server"),
					libvmi.WithCPUFeature("mpx", "require"),
				),
				requiresMPXCPUValidation,
				api.Domain{Spec: api.DomainSpec{
					CPU: api.CPU{
						Mode:     "custom",
						Model:    "Skylake-Server",
						Topology: &api.CPUTopology{Sockets: 1, Cores: 1, Threads: 1},
						Features: []api.CPUFeature{{Name: "mpx", Policy: "require"}},
					},
					VCPU: &api.VCPU{Placement: "static", CPUs: 1},
				}},
			),
		)
	})

	Context("CPU hotplug", func() {
		It("should configure VCPUs for hotplug when MaxSockets is set and hotplug is supported", func() {
			vmi := libvmi.New(
				libvmi.WithCPUCount(1, 1, 2),
				withMaxSockets(4),
			)
			var domain api.Domain

			configurator := compute.NewCPUDomainConfigurator(hotplugSupported, !requiresMPXCPUValidation)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{Spec: api.DomainSpec{
				CPU:  api.CPU{Mode: v1.CPUModeHostModel, Topology: &api.CPUTopology{Sockets: 4, Cores: 1, Threads: 1}},
				VCPU: &api.VCPU{Placement: "static", CPUs: 4},
				VCPUs: &api.VCPUs{VCPU: []api.VCPUsVCPU{
					{ID: 0, Enabled: "yes", Hotpluggable: "no"},
					{ID: 1, Enabled: "yes", Hotpluggable: "yes"},
					{ID: 2, Enabled: "no", Hotpluggable: "yes"},
					{ID: 3, Enabled: "no", Hotpluggable: "yes"},
				}},
			}}
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should not configure VCPUs for hotplug when hotplug is not supported", func() {
			vmi := libvmi.New(
				libvmi.WithCPUCount(1, 1, 2),
				withMaxSockets(4),
			)
			var domain api.Domain

			configurator := compute.NewCPUDomainConfigurator(!hotplugSupported, !requiresMPXCPUValidation)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{Spec: api.DomainSpec{
				CPU:  api.CPU{Mode: v1.CPUModeHostModel, Topology: &api.CPUTopology{Sockets: 2, Cores: 1, Threads: 1}},
				VCPU: &api.VCPU{Placement: "static", CPUs: 2},
			}}
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should not configure VCPUs for hotplug when MaxSockets is zero", func() {
			vmi := libvmi.New(libvmi.WithCPUCount(1, 1, 2))
			var domain api.Domain

			configurator := compute.NewCPUDomainConfigurator(hotplugSupported, !requiresMPXCPUValidation)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{Spec: api.DomainSpec{
				CPU:  api.CPU{Mode: v1.CPUModeHostModel, Topology: &api.CPUTopology{Sockets: 2, Cores: 1, Threads: 1}},
				VCPU: &api.VCPU{Placement: "static", CPUs: 2},
			}}
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should mark first socket vCPUs as non-hotpluggable", func() {
			vmi := libvmi.New(
				libvmi.WithCPUCount(2, 1, 1),
				withMaxSockets(3),
			)
			var domain api.Domain

			configurator := compute.NewCPUDomainConfigurator(hotplugSupported, !requiresMPXCPUValidation)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{Spec: api.DomainSpec{
				CPU:  api.CPU{Mode: v1.CPUModeHostModel, Topology: &api.CPUTopology{Sockets: 3, Cores: 2, Threads: 1}},
				VCPU: &api.VCPU{Placement: "static", CPUs: 6},
				VCPUs: &api.VCPUs{VCPU: []api.VCPUsVCPU{
					{ID: 0, Enabled: "yes", Hotpluggable: "no"},
					{ID: 1, Enabled: "yes", Hotpluggable: "no"},
					{ID: 2, Enabled: "no", Hotpluggable: "yes"},
					{ID: 3, Enabled: "no", Hotpluggable: "yes"},
					{ID: 4, Enabled: "no", Hotpluggable: "yes"},
					{ID: 5, Enabled: "no", Hotpluggable: "yes"},
				}},
			}}
			Expect(domain).To(Equal(expectedDomain))
		})
	})
})

func withMaxSockets(maxSockets uint32) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.CPU == nil {
			vmi.Spec.Domain.CPU = &v1.CPU{}
		}
		vmi.Spec.Domain.CPU.MaxSockets = maxSockets
	}
}
