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
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/launchsecurity"
)

var _ = Describe("LaunchSecurity: AMD Secure Encrypted Virtualization (SEV)", func() {
	Context("SEV policy conversion", func() {
		var policy v1.SEVPolicy

		BeforeEach(func() {
			policy = v1.SEVPolicy{}
		})

		It("should always set NoDebug", func() {
			Expect(launchsecurity.SEVPolicyToBits(nil)).To(Equal(launchsecurity.SEVPolicyNoDebug))
			Expect(launchsecurity.SEVPolicyToBits(&policy)).To(Equal(launchsecurity.SEVPolicyNoDebug))

			policy = v1.SEVPolicy{
				NoKeysSharing:  pointer.BoolPtr(false),
				EncryptedState: pointer.BoolPtr(false),
				NoSend:         pointer.BoolPtr(false),
				Domain:         pointer.BoolPtr(false),
				SEV:            pointer.BoolPtr(false),
			}
			Expect(launchsecurity.SEVPolicyToBits(&policy)).To(Equal(launchsecurity.SEVPolicyNoDebug))

			policy = v1.SEVPolicy{
				NoKeysSharing:  pointer.BoolPtr(true),
				EncryptedState: pointer.BoolPtr(true),
				NoSend:         pointer.BoolPtr(true),
				Domain:         pointer.BoolPtr(true),
				SEV:            pointer.BoolPtr(true),
			}
			Expect(launchsecurity.SEVPolicyToBits(&policy)).ToNot(Equal(launchsecurity.SEVPolicyNoDebug))
			Expect(launchsecurity.SEVPolicyToBits(&policy) & launchsecurity.SEVPolicyNoDebug).To(Equal(launchsecurity.SEVPolicyNoDebug))
		})

		table.DescribeTable("should correctly set individual bits:", func(expectedBit uint, field **bool) {
			Expect(field).ToNot(BeNil())
			*field = nil
			Expect(launchsecurity.SEVPolicyToBits(&policy)).To(Equal(launchsecurity.SEVPolicyNoDebug))
			*field = pointer.BoolPtr(true)
			Expect(launchsecurity.SEVPolicyToBits(&policy)).To(Equal(launchsecurity.SEVPolicyNoDebug | expectedBit))
			*field = pointer.BoolPtr(false)
			Expect(launchsecurity.SEVPolicyToBits(&policy)).To(Equal(launchsecurity.SEVPolicyNoDebug))
		},
			table.Entry("NoKeysSharing ", launchsecurity.SEVPolicyNoKeysSharing, &policy.NoKeysSharing),
			table.Entry("EncryptedState", launchsecurity.SEVPolicyEncryptedState, &policy.EncryptedState),
			table.Entry("NoSend        ", launchsecurity.SEVPolicyNoSend, &policy.NoSend),
			table.Entry("Domain        ", launchsecurity.SEVPolicyDomain, &policy.Domain),
			table.Entry("SEV           ", launchsecurity.SEVPolicySEV, &policy.SEV),
		)
	})
})
