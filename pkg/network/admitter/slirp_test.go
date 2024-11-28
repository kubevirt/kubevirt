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

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validate interface with SLIRP binding", func() {
	It("should be rejected if not enabled in the Kubevirt CR", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{DeprecatedSlirp: &v1.DeprecatedInterfaceSlirp{}},
			}),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		validator := admitter.NewValidator(stubClusterConfigChecker{})
		causes := validator.Validate(k8sfield.NewPath("fake"), &vmi.Spec)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("Slirp interface is not enabled in kubevirt-config"))
	})

	It("should be rejected without a pod network", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{DeprecatedSlirp: &v1.DeprecatedInterfaceSlirp{}},
			}),
			libvmi.WithNetwork(libvmi.MultusNetwork("default", "net")),
		)
		config := stubClusterConfigChecker{slirpEnabled: true}
		validator := admitter.NewValidator(config)
		causes := validator.Validate(k8sfield.NewPath("fake"), &vmi.Spec)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("Slirp interface only implemented with pod network"))
	})

	It("should be accepted with a pod network when SLIRP is enabled in the Kubevirt CR", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{DeprecatedSlirp: &v1.DeprecatedInterfaceSlirp{}},
			}),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		config := stubClusterConfigChecker{slirpEnabled: true}
		validator := admitter.NewValidator(config)
		Expect(validator.Validate(k8sfield.NewPath("fake"), &vmi.Spec)).To(BeEmpty())
	})
})

var _ = Describe("Validate creation of interface with SLIRP binding", func() {
	It("should be rejected", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{DeprecatedSlirp: &v1.DeprecatedInterfaceSlirp{}},
			}),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		validator := admitter.NewValidator(stubClusterConfigChecker{})
		causes := validator.ValidateCreation(k8sfield.NewPath("fake"), &vmi.Spec)
		Expect(causes).To(
			ConsistOf(metav1.StatusCause{
				Type:    "FieldValueInvalid",
				Message: "Slirp interface support has been discontinued since v1.3",
				Field:   "fake.domain.devices.interfaces[0].slirp",
			}),
		)
	})
})
