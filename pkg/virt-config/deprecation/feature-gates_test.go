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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-config/deprecation"
)

var _ = Describe("Feature Gate", func() {
	It("register a simple FG, expect default warning", func() {
		fg := deprecation.FeatureGate{Name: "my-fg", State: deprecation.GA}

		deprecation.RegisterFeatureGate(fg)
		DeferCleanup(deprecation.UnregisterFeatureGate, fg.Name)

		Expect(deprecation.FeatureGateInfo(fg.Name)).To(Equal(&deprecation.FeatureGate{
			Name:    fg.Name,
			State:   fg.State,
			Message: fmt.Sprintf(deprecation.WarningPattern, fg.Name, fg.State),
		}))
	})

	It("register a FG with an explicit warning", func() {
		const message = "my-message"
		fg := deprecation.FeatureGate{Name: "my-fg", State: deprecation.Deprecated, Message: message}

		deprecation.RegisterFeatureGate(fg)
		DeferCleanup(deprecation.UnregisterFeatureGate, fg.Name)

		Expect(deprecation.FeatureGateInfo(fg.Name)).To(Equal(&deprecation.FeatureGate{
			Name:    fg.Name,
			State:   fg.State,
			Message: message,
		}))
	})

	It("register multiple unique FGs", func() {
		fg1 := deprecation.FeatureGate{Name: "my-fg1", State: deprecation.GA, Message: "my-message"}
		fg2 := deprecation.FeatureGate{Name: "my-fg2", State: deprecation.GA, Message: "my-message"}

		deprecation.RegisterFeatureGate(fg1)
		deprecation.RegisterFeatureGate(fg2)
		DeferCleanup(deprecation.UnregisterFeatureGate, fg1.Name)
		DeferCleanup(deprecation.UnregisterFeatureGate, fg2.Name)

		Expect(deprecation.FeatureGateInfo(fg1.Name)).To(Equal(&fg1))
		Expect(deprecation.FeatureGateInfo(fg2.Name)).To(Equal(&fg2))
	})

	It("register FG that overrides an existing one", func() {
		fg1 := deprecation.FeatureGate{Name: "my-fg1", State: deprecation.GA, Message: "my-message"}
		fg2 := deprecation.FeatureGate{Name: "my-fg2", State: deprecation.GA, Message: "my-message"}
		fg1clone := deprecation.FeatureGate{Name: "my-fg1", State: deprecation.GA, Message: "my-other-message"}

		deprecation.RegisterFeatureGate(fg1)
		deprecation.RegisterFeatureGate(fg2)
		deprecation.RegisterFeatureGate(fg1clone)
		DeferCleanup(deprecation.UnregisterFeatureGate, fg1.Name)
		DeferCleanup(deprecation.UnregisterFeatureGate, fg2.Name)
		DeferCleanup(deprecation.UnregisterFeatureGate, fg1clone.Name)

		Expect(deprecation.FeatureGateInfo(fg1.Name)).NotTo(Equal(&fg1))
		Expect(deprecation.FeatureGateInfo(fg1.Name)).To(Equal(&fg1clone))
		Expect(deprecation.FeatureGateInfo(fg2.Name)).To(Equal(&fg2))
	})
})
