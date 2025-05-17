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

package featuregate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Validator", func() {
	const (
		fgName    = "test"
		fgWarning = "test warning message"
	)

	DescribeTable("validate feature gate", func(fgState featuregate.State, expected []metav1.StatusCause) {
		featuregate.RegisterFeatureGate(featuregate.FeatureGate{
			Name:        fgName,
			State:       featuregate.State(fgState),
			VmiSpecUsed: func(_ *v1.VirtualMachineInstanceSpec) bool { return true },
			Message:     fgWarning,
		})
		DeferCleanup(featuregate.UnregisterFeatureGate, fgName)
		vmi := libvmi.New()

		Expect(featuregate.ValidateFeatureGates([]string{fgName}, &vmi.Spec)).To(ConsistOf(expected))
	},
		Entry("that is GA", featuregate.GA, nil),
		Entry("that is Deprecated", featuregate.Deprecated, nil),
		Entry("that is Discontinued", featuregate.Discontinued,
			[]metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: fgWarning,
			}},
		),
	)
})

var _ = Describe("Feature gate enablement", func() {

	Context("feature gate parsing", func() {
		It("should return true for implicitly enabled feature gates", func() {
			const fgName = "fg123"
			enabledFgs, err := featuregate.ParseEnableFeatureGates([]string{fgName})
			Expect(err).ToNot(HaveOccurred())
			Expect(enabledFgs).To(ContainElement(fgName))
		})

		It("should return true for explicitly enabled feature gates", func() {
			const fgName = "fg123"
			enabledFgs, err := featuregate.ParseEnableFeatureGates([]string{fgName + "=true"})
			Expect(err).ToNot(HaveOccurred())
			Expect(enabledFgs).To(ContainElement(fgName))
		})

		It("should return false for explicitly disabled feature gates", func() {
			const fgName = "fg123"
			enabledFgs, err := featuregate.ParseEnableFeatureGates([]string{fgName + "=false"})
			Expect(err).ToNot(HaveOccurred())
			Expect(enabledFgs).ToNot(ContainElement(fgName))
		})

		It("should give precedence to explicitly enabled feature gates", func() {
			const fgName = "fg123"
			enabledFgs, err := featuregate.ParseEnableFeatureGates([]string{fgName, fgName + "=false"})
			Expect(err).ToNot(HaveOccurred())
			Expect(enabledFgs).ToNot(ContainElement(fgName))
		})

		It("should raise an error when the same feature gate is explicitly both disabled and enabled", func() {
			const fgName = "fg123"
			enabledFgs, err := featuregate.ParseEnableFeatureGates([]string{fgName + "=false", fgName + "=true"})
			Expect(err).To(HaveOccurred())
			Expect(enabledFgs).ToNot(ContainElement(fgName))
		})

		It("should raise an error for a non-bool value", func() {
			const fgName = "fg123"
			enabledFgs, err := featuregate.ParseEnableFeatureGates([]string{fgName + "=notabool"})
			Expect(err).To(HaveOccurred())
			Expect(enabledFgs).ToNot(ContainElement(fgName))
		})
	})

})
