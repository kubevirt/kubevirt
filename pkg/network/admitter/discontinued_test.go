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

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validate discontinued bindings", func() {
	DescribeTable("should be rejected",
		func(interfaceBindingMethod v1.InterfaceBindingMethod, expectedMessage, expectedField string) {
			vmi := libvmi.New(
				libvmi.WithInterface(v1.Interface{
					Name:                   "default",
					InterfaceBindingMethod: interfaceBindingMethod,
				}),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{})
			causes := validator.ValidateCreation()
			Expect(causes).To(
				ConsistOf(metav1.StatusCause{
					Type:    "FieldValueInvalid",
					Message: expectedMessage,
					Field:   expectedField,
				}),
			)
		},
		Entry("SLIRP binding",
			v1.InterfaceBindingMethod{DeprecatedSlirp: &v1.DeprecatedInterfaceSlirp{}},
			"Slirp interface support has been discontinued since v1.3",
			"fake.domain.devices.interfaces[0].slirp",
		),
		Entry("Discontinued Passt binding",
			v1.InterfaceBindingMethod{DeprecatedPasst: &v1.DeprecatedInterfacePasst{}},
			"Passt network binding has been discontinued since v1.3",
			"fake.domain.devices.interfaces[0].passt",
		),
		Entry("Discontinued macvtap binding",
			v1.InterfaceBindingMethod{DeprecatedMacvtap: &v1.DeprecatedInterfaceMacvtap{}},
			"Macvtap network binding has been discontinued since v1.3",
			"fake.domain.devices.interfaces[0].macvtap",
		),
	)
})
