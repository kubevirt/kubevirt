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

var _ = Describe("HypervisorFeatures Domain Configurator", func() {
	It("should initialize Features and return nil when VMI has no features", func() {
		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithFeatures(&api.Features{})))
	})

	DescribeTable("should convert ACPI feature", func(enabled *bool, expectedFeatures *api.Features) {
		vmi := libvmi.New(withFeatures(v1.Features{
			ACPI: v1.FeatureState{Enabled: enabled},
		}))
		var domain api.Domain

		configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithFeatures(expectedFeatures)))
	},
		Entry("nil (defaults to enabled)", nil, &api.Features{ACPI: &api.FeatureEnabled{}}),
		Entry("explicitly enabled", new(true), &api.Features{ACPI: &api.FeatureEnabled{}}),
		Entry("explicitly disabled", new(false), &api.Features{}),
	)

	DescribeTable("should convert SMM feature", func(enabled *bool, expectedFeatures *api.Features) {
		vmi := libvmi.New(withFeatures(v1.Features{
			SMM: &v1.FeatureState{Enabled: enabled},
		}))
		var domain api.Domain

		configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithFeatures(expectedFeatures)))
	},
		Entry("nil (defaults to enabled)", nil, &api.Features{ACPI: &api.FeatureEnabled{}, SMM: &api.FeatureEnabled{}}),
		Entry("explicitly enabled", new(true), &api.Features{ACPI: &api.FeatureEnabled{}, SMM: &api.FeatureEnabled{}}),
		Entry("explicitly disabled", new(false), &api.Features{ACPI: &api.FeatureEnabled{}}),
	)

	DescribeTable("should convert APIC feature", func(enabled *bool, expectedFeatures *api.Features) {
		vmi := libvmi.New(withFeatures(v1.Features{
			APIC: &v1.FeatureAPIC{FeatureState: v1.FeatureState{Enabled: enabled}},
		}))
		var domain api.Domain

		configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithFeatures(expectedFeatures)))
	},
		Entry("nil (defaults to enabled)", nil, &api.Features{ACPI: &api.FeatureEnabled{}, APIC: &api.FeatureEnabled{}}),
		Entry("explicitly enabled", new(true), &api.Features{ACPI: &api.FeatureEnabled{}, APIC: &api.FeatureEnabled{}}),
		Entry("explicitly disabled", new(false), &api.Features{ACPI: &api.FeatureEnabled{}}),
	)

	DescribeTable("should convert KVM hidden feature", func(hidden bool, expectedFeatures *api.Features) {
		vmi := libvmi.New(withFeatures(v1.Features{
			KVM: &v1.FeatureKVM{Hidden: hidden},
		}))
		var domain api.Domain

		configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithFeatures(expectedFeatures)))
	},
		Entry("hidden true", true, &api.Features{
			ACPI: &api.FeatureEnabled{},
			KVM:  &api.FeatureKVM{Hidden: &api.FeatureState{State: "on"}},
		}),
		Entry("hidden false", false, &api.Features{
			ACPI: &api.FeatureEnabled{},
			KVM:  &api.FeatureKVM{Hidden: &api.FeatureState{State: "off"}},
		}),
	)

	DescribeTable("should convert Pvspinlock feature", func(enabled *bool, expectedFeatures *api.Features) {
		vmi := libvmi.New(withFeatures(v1.Features{
			Pvspinlock: &v1.FeatureState{Enabled: enabled},
		}))
		var domain api.Domain

		configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithFeatures(expectedFeatures)))
	},
		Entry("nil (defaults to on)", nil, &api.Features{
			ACPI:       &api.FeatureEnabled{},
			PVSpinlock: &api.FeaturePVSpinlock{State: "on"},
		}),
		Entry("explicitly enabled", new(true), &api.Features{
			ACPI:       &api.FeatureEnabled{},
			PVSpinlock: &api.FeaturePVSpinlock{State: "on"},
		}),
		Entry("explicitly disabled", new(false), &api.Features{
			ACPI:       &api.FeatureEnabled{},
			PVSpinlock: &api.FeaturePVSpinlock{State: "off"},
		}),
	)

	Context("HypervPassthrough", func() {
		It("should configure hyperv passthrough mode", func() {
			vmi := libvmi.New(withFeatures(v1.Features{
				HypervPassthrough: &v1.HyperVPassthrough{Enabled: new(true)},
			}))
			var domain api.Domain

			configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain).To(Equal(newDomainWithFeatures(&api.Features{
				ACPI:   &api.FeatureEnabled{},
				Hyperv: &api.FeatureHyperv{Mode: api.HypervModePassthrough},
			})))
		})

		It("should not configure hyperv passthrough when disabled", func() {
			vmi := libvmi.New(withFeatures(v1.Features{
				HypervPassthrough: &v1.HyperVPassthrough{Enabled: new(false)},
			}))
			var domain api.Domain

			configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain).To(Equal(newDomainWithFeatures(&api.Features{ACPI: &api.FeatureEnabled{}})))
		})
	})

	Context("Hyperv", func() {
		It("should convert all hyperv features", func() {
			retries := uint32(4096)
			vmi := libvmi.New(withFeatures(v1.Features{
				Hyperv: &v1.FeatureHyperv{
					Relaxed:         &v1.FeatureState{Enabled: new(true)},
					VAPIC:           &v1.FeatureState{Enabled: new(true)},
					VPIndex:         &v1.FeatureState{Enabled: new(true)},
					Runtime:         &v1.FeatureState{Enabled: new(true)},
					SyNIC:           &v1.FeatureState{Enabled: new(true)},
					Reset:           &v1.FeatureState{Enabled: new(true)},
					Frequencies:     &v1.FeatureState{Enabled: new(true)},
					Reenlightenment: &v1.FeatureState{Enabled: new(true)},
					IPI:             &v1.FeatureState{Enabled: new(true)},
					EVMCS:           &v1.FeatureState{Enabled: new(true)},
					Spinlocks: &v1.FeatureSpinlocks{
						FeatureState: v1.FeatureState{Enabled: new(true)},
						Retries:      &retries,
					},
					VendorID: &v1.FeatureVendorID{
						FeatureState: v1.FeatureState{Enabled: new(true)},
						VendorID:     "myvendor",
					},
					SyNICTimer: &v1.SyNICTimer{
						FeatureState: v1.FeatureState{Enabled: new(true)},
						Direct:       &v1.FeatureState{Enabled: new(true)},
					},
					TLBFlush: &v1.TLBFlush{
						FeatureState: v1.FeatureState{Enabled: new(true)},
						Direct:       &v1.FeatureState{Enabled: new(true)},
						Extended:     &v1.FeatureState{Enabled: new(true)},
					},
				},
			}))
			var domain api.Domain

			configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain).To(Equal(newDomainWithFeatures(&api.Features{
				ACPI: &api.FeatureEnabled{},
				Hyperv: &api.FeatureHyperv{
					Relaxed:         &api.FeatureState{State: "on"},
					VAPIC:           &api.FeatureState{State: "on"},
					VPIndex:         &api.FeatureState{State: "on"},
					Runtime:         &api.FeatureState{State: "on"},
					SyNIC:           &api.FeatureState{State: "on"},
					Reset:           &api.FeatureState{State: "on"},
					Frequencies:     &api.FeatureState{State: "on"},
					Reenlightenment: &api.FeatureState{State: "on"},
					IPI:             &api.FeatureState{State: "on"},
					EVMCS:           &api.FeatureState{State: "on"},
					Spinlocks:       &api.FeatureSpinlocks{State: "on", Retries: &retries},
					VendorID:        &api.FeatureVendorID{State: "on", Value: "myvendor"},
					SyNICTimer: &api.SyNICTimer{
						State:  "on",
						Direct: &api.FeatureState{State: "on"},
					},
					TLBFlush: &api.TLBFlush{
						State:    "on",
						Direct:   &api.FeatureState{State: "on"},
						Extended: &api.FeatureState{State: "on"},
					},
				},
			})))
		})

		It("should default to on when Enabled is nil", func() {
			vmi := libvmi.New(withFeatures(v1.Features{
				Hyperv: &v1.FeatureHyperv{
					Relaxed:    &v1.FeatureState{},
					Spinlocks:  &v1.FeatureSpinlocks{},
					SyNICTimer: &v1.SyNICTimer{},
					TLBFlush:   &v1.TLBFlush{},
				},
			}))
			var domain api.Domain

			configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain).To(Equal(newDomainWithFeatures(&api.Features{
				ACPI: &api.FeatureEnabled{},
				Hyperv: &api.FeatureHyperv{
					Relaxed:    &api.FeatureState{State: "on"},
					Spinlocks:  &api.FeatureSpinlocks{State: "on"},
					SyNICTimer: &api.SyNICTimer{State: "on"},
					TLBFlush:   &api.TLBFlush{State: "on"},
				},
			})))
		})

		It("should convert disabled features to off", func() {
			vmi := libvmi.New(withFeatures(v1.Features{
				Hyperv: &v1.FeatureHyperv{
					Relaxed: &v1.FeatureState{Enabled: new(false)},
					VAPIC:   &v1.FeatureState{Enabled: new(false)},
				},
			}))
			var domain api.Domain

			configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain).To(Equal(newDomainWithFeatures(&api.Features{
				ACPI: &api.FeatureEnabled{},
				Hyperv: &api.FeatureHyperv{
					Relaxed: &api.FeatureState{State: "off"},
					VAPIC:   &api.FeatureState{State: "off"},
				},
			})))
		})

		It("should not set unspecified hyperv features", func() {
			vmi := libvmi.New(withFeatures(v1.Features{
				Hyperv: &v1.FeatureHyperv{},
			}))
			var domain api.Domain

			configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain).To(Equal(newDomainWithFeatures(&api.Features{
				ACPI:   &api.FeatureEnabled{},
				Hyperv: &api.FeatureHyperv{},
			})))
		})

		It("should take precedence over HypervPassthrough", func() {
			vmi := libvmi.New(withFeatures(v1.Features{
				Hyperv: &v1.FeatureHyperv{
					Relaxed: &v1.FeatureState{Enabled: new(true)},
				},
				HypervPassthrough: &v1.HyperVPassthrough{Enabled: new(true)},
			}))
			var domain api.Domain

			configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, false)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain).To(Equal(newDomainWithFeatures(&api.Features{
				ACPI: &api.FeatureEnabled{},
				Hyperv: &api.FeatureHyperv{
					Relaxed: &api.FeatureState{State: "on"},
				},
			})))
		})
	})

	It("should set VMPort off when hasVMPort is true", func() {
		vmi := libvmi.New(withFeatures(v1.Features{}))
		var domain api.Domain

		configurator := compute.NewHypervisorFeaturesDomainConfigurator(true, false)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithFeatures(&api.Features{
			ACPI:   &api.FeatureEnabled{},
			VMPort: &api.FeatureState{State: "off"},
		})))
	})

	It("should set PMU off when useLaunchSecurityTDX is true", func() {
		vmi := libvmi.New(withFeatures(v1.Features{}))
		var domain api.Domain

		configurator := compute.NewHypervisorFeaturesDomainConfigurator(false, true)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithFeatures(&api.Features{
			ACPI: &api.FeatureEnabled{},
			PMU:  &api.FeatureState{State: "off"},
		})))
	})
})

func withFeatures(features v1.Features) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Features = &features
	}
}

func newDomainWithFeatures(features *api.Features) api.Domain {
	return api.Domain{
		Spec: api.DomainSpec{
			Features: features,
		},
	}
}
