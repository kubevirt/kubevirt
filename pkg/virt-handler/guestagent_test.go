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

package virthandler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Guest Agent", func() {

	Context("guestAgentCommandSubsetSupported", func() {
		It("should return true when all required commands are present and enabled", func() {
			available := map[string]bool{
				"guest-ping": true,
				"guest-info": true,
			}
			Expect(guestAgentCommandSubsetSupported([]string{"guest-ping", "guest-info"}, available)).To(BeTrue())
		})

		It("should return false when a required command is missing", func() {
			available := map[string]bool{
				"guest-ping": true,
			}
			Expect(guestAgentCommandSubsetSupported([]string{"guest-ping", "guest-info"}, available)).To(BeFalse())
		})

		It("should return false when a required command is present but disabled", func() {
			available := map[string]bool{
				"guest-ping": true,
				"guest-info": false,
			}
			Expect(guestAgentCommandSubsetSupported([]string{"guest-ping", "guest-info"}, available)).To(BeFalse())
		})

		It("should return true for an empty required commands list", func() {
			available := map[string]bool{"guest-ping": true}
			Expect(guestAgentCommandSubsetSupported([]string{}, available)).To(BeTrue())
		})

		It("should return true for an empty required list and empty available map", func() {
			Expect(guestAgentCommandSubsetSupported([]string{}, map[string]bool{})).To(BeTrue())
		})
	})

	Context("sshRelatedCommandsSupported", func() {
		It("should return true when new SSH commands are available", func() {
			available := map[string]bool{}
			for _, cmd := range sshRelatedGuestAgentCommands {
				available[cmd] = true
			}
			Expect(sshRelatedCommandsSupported(available)).To(BeTrue())
		})

		It("should return true when old SSH commands are available", func() {
			available := map[string]bool{}
			for _, cmd := range oldSSHRelatedGuestAgentCommands {
				available[cmd] = true
			}
			Expect(sshRelatedCommandsSupported(available)).To(BeTrue())
		})

		It("should return false when neither new nor old SSH commands are available", func() {
			available := map[string]bool{
				"guest-ping": true,
			}
			Expect(sshRelatedCommandsSupported(available)).To(BeFalse())
		})

		It("should return false when SSH commands are present but disabled", func() {
			available := map[string]bool{}
			for _, cmd := range sshRelatedGuestAgentCommands {
				available[cmd] = false
			}
			for _, cmd := range oldSSHRelatedGuestAgentCommands {
				available[cmd] = false
			}
			Expect(sshRelatedCommandsSupported(available)).To(BeFalse())
		})

		It("should return true when only partial old commands plus full new commands are available", func() {
			available := map[string]bool{
				// Only partial old commands
				"guest-exec": true,
			}
			// Add all new SSH commands
			for _, cmd := range sshRelatedGuestAgentCommands {
				available[cmd] = true
			}
			Expect(sshRelatedCommandsSupported(available)).To(BeTrue())
		})
	})
})
