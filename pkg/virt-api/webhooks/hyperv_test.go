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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package webhooks

import (
	"fmt"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/client-go/api/v1"
)

var _false bool = false

var _ = Describe("Mutating Webhook HyperV utils", func() {

	It("Should not mutate VMIs without HyperV configuration", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		Expect(vmi.Spec.Domain.Features).To(BeNil())
		err := SetVirtualMachineInstanceHypervFeatureDependencies(vmi)
		Expect(err).To(BeNil())
		Expect(vmi.Spec.Domain.Features).To(BeNil())
	})

	It("Should not mutate VMIs with empty HyperV configuration", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{},
		}
		err := SetVirtualMachineInstanceHypervFeatureDependencies(vmi)
		Expect(err).To(BeNil())
		hyperv := v1.FeatureHyperv{}
		ok := reflect.DeepEqual(*vmi.Spec.Domain.Features.Hyperv, hyperv)
		if !ok {
			// debug aid
			fmt.Fprintf(GinkgoWriter, "got: %#v\n", *vmi.Spec.Domain.Features.Hyperv)
			fmt.Fprintf(GinkgoWriter, "exp: %#v\n", hyperv)
		}
		Expect(ok).To(BeTrue())
	})

	It("Should not mutate VMIs with hyperv configuration without deps", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: &_true,
				},
				Runtime: &v1.FeatureState{
					Enabled: &_true,
				},
				Reset: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		err := SetVirtualMachineInstanceHypervFeatureDependencies(vmi)
		Expect(err).To(BeNil())

		hyperv := v1.FeatureHyperv{
			Relaxed: &v1.FeatureState{
				Enabled: &_true,
			},
			Runtime: &v1.FeatureState{
				Enabled: &_true,
			},
			Reset: &v1.FeatureState{
				Enabled: &_true,
			},
		}

		ok := reflect.DeepEqual(*vmi.Spec.Domain.Features.Hyperv, hyperv)
		if !ok {
			// debug aid
			fmt.Fprintf(GinkgoWriter, "got: %#v\n", *vmi.Spec.Domain.Features.Hyperv)
			fmt.Fprintf(GinkgoWriter, "exp: %#v\n", hyperv)
		}
		Expect(ok).To(BeTrue())
	})

	It("Should mutate VMIs with hyperv configuration to fix deps", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: &_true,
				},
				SyNICTimer: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		err := SetVirtualMachineInstanceHypervFeatureDependencies(vmi)
		Expect(err).To(BeNil())

		hyperv := v1.FeatureHyperv{
			Relaxed: &v1.FeatureState{
				Enabled: &_true,
			},
			VPIndex: &v1.FeatureState{
				Enabled: &_true,
			},
			SyNIC: &v1.FeatureState{
				Enabled: &_true,
			},
			SyNICTimer: &v1.FeatureState{
				Enabled: &_true,
			},
		}

		ok := reflect.DeepEqual(*vmi.Spec.Domain.Features.Hyperv, hyperv)
		if !ok {
			// debug aid
			fmt.Fprintf(GinkgoWriter, "got: %#v\n", *vmi.Spec.Domain.Features.Hyperv)
			fmt.Fprintf(GinkgoWriter, "exp: %#v\n", hyperv)
		}
		Expect(ok).To(BeTrue())
	})

	It("Should partially mutate VMIs with explicit hyperv configuration", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				VPIndex: &v1.FeatureState{
					Enabled: &_false,
				},
				// should enable SyNIC
				SyNICTimer: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		SetVirtualMachineInstanceHypervFeatureDependencies(vmi)
		// we MUST report the error in mutation, but production code is
		// supposed to ignore it to fullfill the design semantics, see
		// the discussion in https://github.com/kubevirt/kubevirt/pull/2408

		hyperv := v1.FeatureHyperv{
			VPIndex: &v1.FeatureState{
				Enabled: &_false,
			},
			SyNIC: &v1.FeatureState{
				Enabled: &_true,
			},
			SyNICTimer: &v1.FeatureState{
				Enabled: &_true,
			},
		}

		ok := reflect.DeepEqual(*vmi.Spec.Domain.Features.Hyperv, hyperv)
		if !ok {
			// debug aid
			fmt.Fprintf(GinkgoWriter, "got: %#v\n", *vmi.Spec.Domain.Features.Hyperv)
			fmt.Fprintf(GinkgoWriter, "exp: %#v\n", hyperv)
		}
		Expect(ok).To(BeTrue())
	})

})

var _ = Describe("Validating Webhook HyperV utils", func() {

	It("Should validate VMIs without HyperV configuration", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		Expect(vmi.Spec.Domain.Features).To(BeNil())
		path := k8sfield.NewPath("spec")
		causes := ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(0))
	})

	It("Should validate VMIs with empty HyperV configuration", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{},
		}
		path := k8sfield.NewPath("spec")
		causes := ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(0))
	})

	It("Should validate VMIs with hyperv configuration without deps", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: &_true,
				},
				Runtime: &v1.FeatureState{
					Enabled: &_true,
				},
				Reset: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(0))
	})

	It("Should not validate VMIs with broken hyperv deps", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: &_true,
				},
				SyNIC: &v1.FeatureState{
					Enabled: &_true,
				},
				SyNICTimer: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		path := k8sfield.NewPath("spec")
		causes := ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(BeNumerically(">=", 1))
	})

	It("Should validate VMIs with correct hyperv deps", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: &_true,
				},
				VPIndex: &v1.FeatureState{
					Enabled: &_true,
				},
				SyNIC: &v1.FeatureState{
					Enabled: &_true,
				},
				SyNICTimer: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}

		path := k8sfield.NewPath("spec")
		causes := ValidateVirtualMachineInstanceHypervFeatureDependencies(path, &vmi.Spec)
		Expect(len(causes)).To(Equal(0))
	})
})
