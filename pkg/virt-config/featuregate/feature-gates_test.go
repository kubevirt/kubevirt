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

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Feature Gate", func() {
	It("register a simple FG, expect default warning", func() {
		fg := featuregate.FeatureGate{Name: "my-fg", State: featuregate.GA}

		featuregate.RegisterFeatureGate(fg)
		DeferCleanup(featuregate.UnregisterFeatureGate, fg.Name)

		Expect(featuregate.FeatureGateInfo(fg.Name)).To(Equal(&featuregate.FeatureGate{
			Name:    fg.Name,
			State:   fg.State,
			Message: fmt.Sprintf(featuregate.WarningPattern, fg.Name, fg.State),
		}))
	})

	It("register a FG with an explicit warning", func() {
		const message = "my-message"
		fg := featuregate.FeatureGate{Name: "my-fg", State: featuregate.Deprecated, Message: message}

		featuregate.RegisterFeatureGate(fg)
		DeferCleanup(featuregate.UnregisterFeatureGate, fg.Name)

		Expect(featuregate.FeatureGateInfo(fg.Name)).To(Equal(&featuregate.FeatureGate{
			Name:    fg.Name,
			State:   fg.State,
			Message: message,
		}))
	})

	It("register multiple unique FGs", func() {
		fg1 := featuregate.FeatureGate{Name: "my-fg1", State: featuregate.GA, Message: "my-message"}
		fg2 := featuregate.FeatureGate{Name: "my-fg2", State: featuregate.GA, Message: "my-message"}

		featuregate.RegisterFeatureGate(fg1)
		featuregate.RegisterFeatureGate(fg2)
		DeferCleanup(featuregate.UnregisterFeatureGate, fg1.Name)
		DeferCleanup(featuregate.UnregisterFeatureGate, fg2.Name)

		Expect(featuregate.FeatureGateInfo(fg1.Name)).To(Equal(&fg1))
		Expect(featuregate.FeatureGateInfo(fg2.Name)).To(Equal(&fg2))
	})

	Context("IsFeatureGateEnabled", func() {
		const (
			testGAGate         = "test-ga-gate"
			testBetaGate       = "test-beta-gate"
			testAlphaGate      = "test-alpha-gate"
			testDeprecatedGate = "test-deprecated-gate"
			testUnknownGate    = "test-unknown-unregistered-gate"
		)

		BeforeEach(func() {
			featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: testGAGate, State: featuregate.GA})
			featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: testBetaGate, State: featuregate.Beta})
			featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: testAlphaGate, State: featuregate.Alpha})
			featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: testDeprecatedGate, State: featuregate.Deprecated})
		})

		AfterEach(func() {
			featuregate.UnregisterFeatureGate(testGAGate)
			featuregate.UnregisterFeatureGate(testBetaGate)
			featuregate.UnregisterFeatureGate(testAlphaGate)
			featuregate.UnregisterFeatureGate(testDeprecatedGate)
		})

		DescribeTable("should return the expected result",
			func(gate string, devConfig *v1.DeveloperConfiguration, expected bool) {
				Expect(featuregate.IsFeatureGateEnabled(gate, devConfig)).To(Equal(expected))
			},
			Entry("GA gate with nil config", testGAGate, nil, true),
			Entry("GA gate in disabled list is still enabled",
				testGAGate, &v1.DeveloperConfiguration{DisabledFeatureGates: []string{testGAGate}}, true),
			Entry("Beta gate with nil config defaults to on", testBetaGate, nil, true),
			Entry("Beta gate explicitly disabled",
				testBetaGate, &v1.DeveloperConfiguration{DisabledFeatureGates: []string{testBetaGate}}, false),
			Entry("Beta gate explicitly enabled",
				testBetaGate, &v1.DeveloperConfiguration{FeatureGates: []string{testBetaGate}}, true),
			Entry("Alpha gate with nil config defaults to off", testAlphaGate, nil, false),
			Entry("Alpha gate explicitly enabled",
				testAlphaGate, &v1.DeveloperConfiguration{FeatureGates: []string{testAlphaGate}}, true),
			Entry("Alpha gate explicitly disabled",
				testAlphaGate, &v1.DeveloperConfiguration{DisabledFeatureGates: []string{testAlphaGate}}, false),
			Entry("Deprecated gate with nil config defaults to off", testDeprecatedGate, nil, false),
			Entry("Unregistered gate with nil config defaults to off", testUnknownGate, nil, false),
			Entry("Unregistered gate explicitly disabled",
				testUnknownGate, &v1.DeveloperConfiguration{FeatureGates: []string{testUnknownGate}}, false),
		)
	})

	It("register FG that overrides an existing one", func() {
		fg1 := featuregate.FeatureGate{Name: "my-fg1", State: featuregate.GA, Message: "my-message"}
		fg2 := featuregate.FeatureGate{Name: "my-fg2", State: featuregate.GA, Message: "my-message"}
		fg1clone := featuregate.FeatureGate{Name: "my-fg1", State: featuregate.GA, Message: "my-other-message"}

		featuregate.RegisterFeatureGate(fg1)
		featuregate.RegisterFeatureGate(fg2)
		featuregate.RegisterFeatureGate(fg1clone)
		DeferCleanup(featuregate.UnregisterFeatureGate, fg1.Name)
		DeferCleanup(featuregate.UnregisterFeatureGate, fg1.Name)
		DeferCleanup(featuregate.UnregisterFeatureGate, fg1clone.Name)

		Expect(featuregate.FeatureGateInfo(fg1.Name)).NotTo(Equal(&fg1))
		Expect(featuregate.FeatureGateInfo(fg1.Name)).To(Equal(&fg1clone))
		Expect(featuregate.FeatureGateInfo(fg2.Name)).To(Equal(&fg2))
	})
})
