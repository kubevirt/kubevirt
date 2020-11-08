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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package agentpoller

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Qemu agent poller", func() {
	Context("with AsyncAgentStore", func() {

		It("should store and load the data", func() {
			var agentStore = NewAsyncAgentStore()
			agentVersion := "4.1"
			agentStore.Store(GET_AGENT, agentVersion)
			agent := agentStore.GetGA()

			Expect(agent).To(Equal(agentVersion))
		})

		It("should fire an event for new sysinfo data", func() {
			var agentStore = NewAsyncAgentStore()

			fakeInfo := api.GuestOSInfo{
				Name:          "TestGuestOSName",
				KernelRelease: "1.1.0-Generic",
				Version:       "1.0.0",
				PrettyName:    "TestGuestOSName 1.0.0",
				VersionId:     "1.0.0",
				KernelVersion: "1.1.0",
				Machine:       "x86_64",
				Id:            "testguestos",
			}
			agentStore.Store(GET_OSINFO, fakeInfo)

			Expect(agentStore.AgentUpdated).To(Receive(Equal(AgentUpdatedEvent{
				Type:       GET_OSINFO,
				DomainInfo: api.DomainGuestInfo{OSInfo: &fakeInfo},
			})))
		})

		It("should not fire an event for the same sysinfo data", func() {
			var agentStore = NewAsyncAgentStore()
			fakeInfo := api.GuestOSInfo{
				Name:          "TestGuestOSName",
				KernelRelease: "1.1.0-Generic",
				Version:       "1.0.0",
				PrettyName:    "TestGuestOSName 1.0.0",
				VersionId:     "1.0.0",
				KernelVersion: "1.1.0",
				Machine:       "x86_64",
				Id:            "testguestos",
			}

			agentStore.Store(GET_OSINFO, fakeInfo)
			Expect(agentStore.AgentUpdated).To(Receive(Equal(AgentUpdatedEvent{
				Type:       GET_OSINFO,
				DomainInfo: api.DomainGuestInfo{OSInfo: &fakeInfo},
			})))

			agentStore.Store(GET_OSINFO, fakeInfo)
			Expect(agentStore.AgentUpdated).ToNot(Receive())
		})
	})
})
