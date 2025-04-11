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

package admitter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validating macvtap core binding", func() {
	It("should reject a macvtap interface on a network different than multus", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   "default",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{DeprecatedMacvtap: &v1.DeprecatedInterfaceMacvtap{}},
		}}
		spec.Networks = []v1.Network{{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

		clusterConfig := stubClusterConfigChecker{macvtapFeatureGateEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, clusterConfig)
		causes := validator.Validate()

		Expect(causes).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueInvalid",
			Message: "Macvtap interface only implemented with Multus network",
			Field:   "fake.domain.devices.interfaces[0].name",
		}))
	})

	It("should reject a macvtap interface on a multus network when the feature is inactive", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   "default",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{DeprecatedMacvtap: &v1.DeprecatedInterfaceMacvtap{}},
		}}
		spec.Networks = []v1.Network{{
			Name:          "default",
			NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "test"}},
		}}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()

		Expect(causes).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueInvalid",
			Message: "Macvtap feature gate is not enabled",
			Field:   "fake.domain.devices.interfaces[0].name",
		}))
	})

	It("should accept a macvtap interface on a multus network when FG is active", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   "default",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{DeprecatedMacvtap: &v1.DeprecatedInterfaceMacvtap{}},
		}}
		spec.Networks = []v1.Network{{
			Name:          "default",
			NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "test"}},
		}}

		clusterConfig := stubClusterConfigChecker{macvtapFeatureGateEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, clusterConfig)
		Expect(validator.Validate()).To(BeEmpty())
	})
})
