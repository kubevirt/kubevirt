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
 * Copyright 2025 Red Hat, Inc.
 *
*/

package launchsecurity_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/launchsecurity"
)

var _ = Describe("LaunchSecurity: Intel TDX", func() {
	Context("TDX policy conversion", func() {
		var policy uint64

		BeforeEach(func() {
			policy = 1 << 0 // debug mode
		})

		It("should always set TDXPolicyNoDebug", func() {
			Expect(launchsecurity.TDXPolicyToBits(policy)).To(Equal(launchsecurity.TDXPolicyNoDebug))
			Expect(launchsecurity.TDXPolicyToBits(0)).To(Equal(launchsecurity.TDXPolicyNoDebug))
			Expect(launchsecurity.TDXPolicyToBits(0xffffffffffffffff)).To(Equal(launchsecurity.TDXPolicyNoDebug | launchsecurity.TDXDisableVEConversion))
		})
	})
})
