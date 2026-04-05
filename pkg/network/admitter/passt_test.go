/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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

var _ = Describe("Validating passtBinding core binding", func() {
	It("should reject networks with a multus network source and passtBinding interface", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   "default",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{PasstBinding: &v1.InterfacePasstBinding{}},
		}}
		spec.Networks = []v1.Network{{
			Name:          "default",
			NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "test"}},
		}}

		clusterConfig := stubClusterConfigChecker{passtBindingFeatureGateEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, clusterConfig)
		causes := validator.Validate()

		Expect(causes).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueInvalid",
			Message: "PasstBinding interface only implemented with pod network",
			Field:   "fake.domain.devices.interfaces[0].name",
		}))
	})

	It("should reject networks with a passtBinding interface and passtBinding feature gate disabled", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   "default",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{PasstBinding: &v1.InterfacePasstBinding{}},
		}}
		spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()

		Expect(causes).To(ConsistOf(metav1.StatusCause{
			Type:    "FieldValueInvalid",
			Message: "PasstBinding feature gate is not enabled",
			Field:   "fake.domain.devices.interfaces[0].name",
		}))
	})

	It("should accept networks with a pod network source and passtBinding interface", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:                   "default",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{PasstBinding: &v1.InterfacePasstBinding{}},
		}}
		spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

		clusterConfig := stubClusterConfigChecker{passtBindingFeatureGateEnabled: true}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, clusterConfig)
		Expect(validator.Validate()).To(BeEmpty())
	})
})
