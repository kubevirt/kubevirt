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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package admitter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validate interface with SLIRP binding", func() {
	It("should be rejected if not enabled in the Kubevirt CR", func() {
		vmi := v1.VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name: "default",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{
				DeprecatedSlirp: &v1.DeprecatedInterfaceSlirp{},
			},
		}}
		vmi.Spec.Networks = []v1.Network{{
			Name:          "default",
			NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
		}}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("Slirp interface is not enabled in kubevirt-config"))
	})

	It("should be rejected without a pod network", func() {
		vmi := v1.VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name: "default",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{
				DeprecatedSlirp: &v1.DeprecatedInterfaceSlirp{},
			},
		}}
		vmi.Spec.Networks = []v1.Network{{
			Name:          "default",
			NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "net"}},
		}}

		config := stubClusterConfigChecker{slirpEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, config)
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("Slirp interface only implemented with pod network"))
	})

	It("should be accepted with a pod network when SLIRP is enabled in the Kubevirt CR", func() {
		vmi := v1.VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name: "default",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{
				DeprecatedSlirp: &v1.DeprecatedInterfaceSlirp{},
			},
		}}
		vmi.Spec.Networks = []v1.Network{{
			Name:          "default",
			NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
		}}

		config := stubClusterConfigChecker{slirpEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, config)
		Expect(validator.Validate()).To(BeEmpty())
	})
})

var _ = Describe("Validate creation of interface with SLIRP binding", func() {
	It("should be rejected", func() {
		vmi := v1.VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name: "default",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{
				DeprecatedSlirp: &v1.DeprecatedInterfaceSlirp{},
			},
		}}
		vmi.Spec.Networks = []v1.Network{{
			Name:          "default",
			NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
		}}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{})
		causes := validator.ValidateCreation()
		Expect(causes).To(
			ConsistOf(metav1.StatusCause{
				Type:    "FieldValueInvalid",
				Message: "Slirp interface support has been discontinued since v1.3",
				Field:   "fake.domain.devices.interfaces[0].slirp",
			}),
		)
	})
})
