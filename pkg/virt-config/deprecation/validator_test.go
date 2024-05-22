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

package deprecation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/deprecation"
)

var _ = Describe("Validator", func() {
	DescribeTable("validate feature gate", func(fgName string, vmi *v1.VirtualMachineInstance, expected []metav1.StatusCause) {
		Expect(deprecation.ValidateFeatureGates([]string{fgName}, &vmi.Spec)).To(ConsistOf(expected))
	},
		Entry("that is GA", deprecation.LiveMigrationGate, libvmi.New(), nil),
		Entry(
			"that is Deprecated",
			deprecation.PasstGate,
			libvmi.New(
				libvmi.WithInterface(v1.Interface{InterfaceBindingMethod: v1.InterfaceBindingMethod{DeprecatedPasst: &v1.DeprecatedInterfacePasst{}}}),
				libvmi.WithNetwork(&v1.Network{}),
			),
			nil,
		),
		Entry(
			"that is Discontinued",
			deprecation.MacvtapGate,
			libvmi.New(
				libvmi.WithInterface(v1.Interface{
					InterfaceBindingMethod: v1.InterfaceBindingMethod{DeprecatedMacvtap: &v1.DeprecatedInterfaceMacvtap{}},
				}),
				libvmi.WithNetwork(&v1.Network{}),
			),
			[]metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: deprecation.MacvtapDiscontinueMessage,
			}},
		),
	)
})
