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
	"fmt"

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

	Context("dependencies", func() {
		const (
			dependentFG = "test-dependent"
			depFG       = "test-dependency"
		)

		registerDependent := func(fgState, depState featuregate.State) {
			featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: depFG, State: depState})
			featuregate.RegisterFeatureGate(featuregate.FeatureGate{
				Name: dependentFG, State: fgState, Dependencies: []string{depFG},
			})
			DeferCleanup(featuregate.UnregisterFeatureGate, depFG)
			DeferCleanup(featuregate.UnregisterFeatureGate, dependentFG)
		}

		DescribeTable("should warn about a missing dependency", func(fgState, depState featuregate.State, devConfig *v1.DeveloperConfiguration, expectWarning bool) {
			registerDependent(fgState, depState)

			warnings := featuregate.WarnMissingDependencies(devConfig)

			if expectWarning {
				Expect(warnings).To(ConsistOf(fmt.Sprintf(featuregate.DependencyWarningPattern, dependentFG, depFG)))
			} else {
				Expect(warnings).To(BeEmpty())
			}
		},
			Entry("when an Alpha dependency is not enabled",
				featuregate.Alpha, featuregate.Alpha,
				&v1.DeveloperConfiguration{FeatureGates: []string{dependentFG}}, true),
			Entry("not when an Alpha dependency is explicitly enabled",
				featuregate.Alpha, featuregate.Alpha,
				&v1.DeveloperConfiguration{FeatureGates: []string{dependentFG, depFG}}, false),
			Entry("not when a Beta dependency is enabled by default",
				featuregate.Alpha, featuregate.Beta,
				&v1.DeveloperConfiguration{FeatureGates: []string{dependentFG}}, false),
			Entry("when a Beta dependency is explicitly disabled",
				featuregate.Alpha, featuregate.Beta,
				&v1.DeveloperConfiguration{FeatureGates: []string{dependentFG}, DisabledFeatureGates: []string{depFG}}, true),
			Entry("not when a GA dependency is not listed",
				featuregate.Alpha, featuregate.GA,
				&v1.DeveloperConfiguration{FeatureGates: []string{dependentFG}}, false),
			Entry("when a Beta feature gate enabled by default has an unmet Alpha dependency",
				featuregate.Beta, featuregate.Alpha,
				&v1.DeveloperConfiguration{}, true),
			Entry("when a Beta feature gate enabled by default has an unmet Alpha dependency and a nil config",
				featuregate.Beta, featuregate.Alpha, nil, true),
		)

		It("should not warn when the feature gate is not enabled", func() {
			registerDependent(featuregate.Alpha, featuregate.Alpha)

			Expect(featuregate.WarnMissingDependencies(&v1.DeveloperConfiguration{})).To(BeEmpty())
		})

		It("should not warn for a nil developer configuration", func() {
			Expect(featuregate.WarnMissingDependencies(nil)).To(BeEmpty())
		})
	})
})
