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
	"kubevirt.io/kubevirt/pkg/pointer"
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

	DescribeTable("get enabled feature gates", func(featureGates []v1.FeatureGateConfiguration, legacyFeatureGates []string, expected []string) {
		Expect(featuregate.GetEnabledFeatureGates(featureGates, legacyFeatureGates)).To(BeEquivalentTo(expected))
	},
		Entry("with no feature gates", nil, nil, nil),
		Entry("with only legacy feature gates", nil, []string{"legacy1", "legacy2"}, []string{"legacy1", "legacy2"}),
		Entry("with only feature gate configurations", []v1.FeatureGateConfiguration{{Name: "fgconfig1"}, {Name: "fgconfig2"}}, nil, []string{"fgconfig1", "fgconfig2"}),
		Entry("with feature gate configurations being disabled", []v1.FeatureGateConfiguration{{Name: "fg1"}, {Name: "fg2", Enabled: pointer.P(false)}}, nil, []string{"fg1"}),
		Entry("new feature gate configuration should take precedence over legacy feature gates", []v1.FeatureGateConfiguration{{Name: "fg1"}, {Name: "fg2", Enabled: pointer.P(false)}}, []string{"fg2"}, []string{"fg1"}),
		Entry("new feature gate configuration is enabled and also the legacy feature gates", []v1.FeatureGateConfiguration{{Name: "fg1"}, {Name: "fg2", Enabled: pointer.P(true)}}, []string{"fg2"}, []string{"fg1", "fg2"}),
	)
})
