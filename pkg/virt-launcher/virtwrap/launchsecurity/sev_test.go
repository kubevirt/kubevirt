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
 * Copyright 2021
 *
 */

package launchsecurity_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/launchsecurity"
)

var _ = Describe("LaunchSecurity: AMD Secure Encrypted Virtualization (SEV)", func() {
	Context("SEV policy conversion", func() {
		It("should succeed when correct values are provided", func() {
			policy := []v1.SEVPolicy{
				v1.SEVPolicyNoDebug,
				v1.SEVPolicyNoKeysSharing,
				v1.SEVPolicyEncryptedState,
				v1.SEVPolicyNoSend,
				v1.SEVPolicyDomain,
				v1.SEVPolicySEV,
			}
			bits, err := launchsecurity.SEVPolicyToBits(policy)
			Expect(err).ToNot(HaveOccurred())
			Expect(bits).To(Equal(uint(0b111111)))
		})

		It("should fail when incorrect values are provided", func() {
			policy := []v1.SEVPolicy{
				"WrongPolicy",
			}
			bits, err := launchsecurity.SEVPolicyToBits(policy)
			Expect(err).To(HaveOccurred())
			Expect(bits).To(Equal(uint(0)))
		})
	})
})
