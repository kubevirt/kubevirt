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
