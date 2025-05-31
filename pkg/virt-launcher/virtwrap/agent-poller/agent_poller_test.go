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

package agentpoller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"libvirt.org/go/libvirt"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
)

var _ = Describe("Qemu agent poller", func() {
	var fakeInterfaces []api.InterfaceStatus
	var fakeFSFreezeStatus api.FSFreeze
	var fakeInfo api.GuestOSInfo
	var agentStore AsyncAgentStore
	var ctrl *gomock.Controller
	var mockLibvirt *testing.Libvirt

	BeforeEach(func() {
		fakeInterfaces = []api.InterfaceStatus{
			{
				Mac: "00:00:00:00:00:01",
			},
		}
		fakeFSFreezeStatus = api.FSFreeze{
			Status: "frozen",
		}
		fakeInfo = api.GuestOSInfo{
			Name:          "TestGuestOSName",
			KernelRelease: "1.1.0-Generic",
			Version:       "1.0.0",
			PrettyName:    "TestGuestOSName 1.0.0",
			VersionId:     "1.0.0",
			KernelVersion: "1.1.0",
			Machine:       "x86_64",
			Id:            "testguestos",
		}
		agentStore = NewAsyncAgentStore()
		ctrl = gomock.NewController(GinkgoT())
		mockLibvirt = testing.NewLibvirt(ctrl)
		mockLibvirt.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(mockLibvirt.VirtDomain, nil).AnyTimes()
	})

	Context("with libvirt API", func() {
		It("should store the retrieved guest info when requested", func() {
			guestInfo := &libvirt.DomainGuestInfo{
				Interfaces: []libvirt.DomainGuestInfoInterface{{Name: "net0"}},
				OS:         &libvirt.DomainGuestInfoOS{Name: "fedora"},
				Hostname:   "test-host",
				TimeZone:   &libvirt.DomainGuestInfoTimeZone{Name: "EST"},
				Users:      []libvirt.DomainGuestInfoUser{{Name: "admin"}},
			}
			agentPoller := &AgentPoller{
				Connection: mockLibvirt.VirtConnection,
				domainName: "fake",
				agentStore: &agentStore,
			}
			libvirtTypes := libvirt.DOMAIN_GUEST_INFO_INTERFACES |
				libvirt.DOMAIN_GUEST_INFO_OS |
				libvirt.DOMAIN_GUEST_INFO_HOSTNAME |
				libvirt.DOMAIN_GUEST_INFO_TIMEZONE |
				libvirt.DOMAIN_GUEST_INFO_USERS

			mockLibvirt.DomainEXPECT().Free()
			mockLibvirt.DomainEXPECT().GetGuestInfo(libvirtTypes, uint32(0)).Return(guestInfo, nil)

			fetchAndStoreGuestInfo(libvirtTypes, agentPoller)

			interfacesStatus := agentStore.GetInterfaceStatus()
			Expect(interfacesStatus[0].InterfaceName).To(Equal("net0"))

			osInfo := agentStore.GetGuestOSInfo()
			Expect(osInfo.Name).To(Equal("fedora"))

			sysInfo := agentStore.GetSysInfo()
			Expect(sysInfo.Hostname).To(Equal("test-host"))
			Expect(sysInfo.Timezone.Zone).To(Equal("EST"))

			users := agentStore.GetUsers(1)
			Expect(users[0].Name).To(Equal("admin"))
		})
	})

	Context("with AsyncAgentStore", func() {
		It("should store and load the data", func() {
			agentVersion := AgentInfo{Version: "4.1"}
			agentStore.Store(GetAgent, agentVersion)
			agent := agentStore.GetGA()

			Expect(agent).To(Equal(agentVersion))
		})

		It("should fire an event for new fsfreezestatus", func() {
			agentStore.Store(GetFSFreezeStatus, fakeFSFreezeStatus)

			Expect(agentStore.AgentUpdated).To(Receive(Equal(AgentUpdatedEvent{
				DomainInfo: api.DomainGuestInfo{
					Interfaces:     nil,
					FSFreezeStatus: &fakeFSFreezeStatus,
					OSInfo:         nil,
				},
			})))
		})

		It("should not fire an event for the same fsfreezestatus", func() {
			agentStore.Store(GetFSFreezeStatus, fakeFSFreezeStatus)

			Expect(agentStore.AgentUpdated).To(Receive(Equal(AgentUpdatedEvent{
				DomainInfo: api.DomainGuestInfo{
					Interfaces:     nil,
					FSFreezeStatus: &fakeFSFreezeStatus,
					OSInfo:         nil,
				},
			})))

			agentStore.Store(GetFSFreezeStatus, fakeFSFreezeStatus)
			Expect(agentStore.AgentUpdated).ToNot(Receive())
		})

		It("should fire an event for new sysinfo data", func() {
			agentStore.Store(libvirt.DOMAIN_GUEST_INFO_OS, fakeInfo)
			Expect(agentStore.AgentUpdated).To(Receive(Equal(AgentUpdatedEvent{
				DomainInfo: api.DomainGuestInfo{OSInfo: &fakeInfo},
			})))
		})

		It("should not fire an event for the same sysinfo data", func() {
			agentStore.Store(libvirt.DOMAIN_GUEST_INFO_OS, fakeInfo)
			Expect(agentStore.AgentUpdated).To(Receive(Equal(AgentUpdatedEvent{
				DomainInfo: api.DomainGuestInfo{OSInfo: &fakeInfo},
			})))

			agentStore.Store(libvirt.DOMAIN_GUEST_INFO_OS, fakeInfo)
			Expect(agentStore.AgentUpdated).ToNot(Receive())
		})

		It("should fire an event with new updated key and old non updated keys", func() {
			agentStore.Store(libvirt.DOMAIN_GUEST_INFO_INTERFACES, fakeInterfaces)
			Expect(agentStore.AgentUpdated).To(Receive(Equal(AgentUpdatedEvent{
				DomainInfo: api.DomainGuestInfo{
					Interfaces: fakeInterfaces,
				},
			})))

			agentStore.Store(GetFSFreezeStatus, fakeFSFreezeStatus)
			Expect(agentStore.AgentUpdated).To(Receive(Equal(AgentUpdatedEvent{
				DomainInfo: api.DomainGuestInfo{
					Interfaces:     fakeInterfaces,
					FSFreezeStatus: &fakeFSFreezeStatus,
				},
			})))

			agentStore.Store(libvirt.DOMAIN_GUEST_INFO_OS, fakeInfo)
			Expect(agentStore.AgentUpdated).To(Receive(Equal(AgentUpdatedEvent{
				DomainInfo: api.DomainGuestInfo{
					Interfaces:     fakeInterfaces,
					FSFreezeStatus: &fakeFSFreezeStatus,
					OSInfo:         &fakeInfo,
				},
			})))
		})

		It("should report nil slice when no interfaces exists", func() {
			interfacesStatus := agentStore.GetInterfaceStatus()

			Expect(interfacesStatus).To(BeNil())
		})

		It("should report interfaces info when interfaces exists", func() {
			agentStore.Store(libvirt.DOMAIN_GUEST_INFO_INTERFACES, fakeInterfaces)
			interfacesStatus := agentStore.GetInterfaceStatus()

			Expect(interfacesStatus).To(Equal(fakeInterfaces))
		})

		It("should report nil when no osInfo exists", func() {
			osInfo := agentStore.GetGuestOSInfo()

			Expect(osInfo).To(BeNil())
		})

		It("should report osInfo when osInfo exists", func() {
			agentStore.Store(libvirt.DOMAIN_GUEST_INFO_OS, fakeInfo)
			osInfo := agentStore.GetGuestOSInfo()

			Expect(*osInfo).To(Equal(fakeInfo))
		})
	})

	Context("PollerWorker", func() {
		It("executes the agent commands at least once", func() {
			const interval = 1
			const expectedExecutions = 1

			commandExecutions := runPollAndCountCommandExecution(interval, expectedExecutions, 0, pollInitialInterval)

			Expect(commandExecutions).To(Equal(expectedExecutions))
		})

		It("executes the agent commands based on the time interval specified", func() {
			const interval = 1
			const expectedExecutions = 3

			commandExecutions := runPollAndCountCommandExecution(interval, expectedExecutions, 0, pollInitialInterval)

			Expect(commandExecutions).To(Equal(expectedExecutions))
		})

		It("executes the agent commands based on the minimum interval at initial run", func() {
			const interval = 30
			const expectedExecutions = 3
			const unitTestPollInitialInterval = time.Second

			// Given the initial interval is 1sec, the code under test is expected to execute the commands at time:
			// 0, 1sec, 1sec + 2*1sec
			// Therefore, setting a timeout limit of 4sec+ should cover the first 3 executions.
			t := 10 * unitTestPollInitialInterval
			commandExecutions := runPollAndCountCommandExecution(interval, expectedExecutions, t, unitTestPollInitialInterval)

			Expect(commandExecutions).To(Equal(expectedExecutions))
		})
	})
})

// runPollAndCountCommandExecution runs a PollerWorker with the specified polling interval
// and counts the number of times the command has been executed.
// The operation is limited by the provided or self calculated timeout and the expected executions.
// The timeout needs to be large enough to allow the expected executions to occur and to accommodate the
// inaccuracy of the go-routine execution.
func runPollAndCountCommandExecution(interval, expectedExecutions int, timeout, initialInterval time.Duration) int {
	const fakeAgentCommandName = "foo"
	w := PollerWorker{
		CallTick:      time.Duration(interval),
		AgentCommands: []AgentCommand{fakeAgentCommandName},
	}
	// Closing the c channel assures go-routine termination.
	// The done channel is a receiver, therefore left to the gc for collection.
	c := make(chan struct{})
	defer close(c)
	done := make(chan struct{})

	go w.Poll(func() { done <- struct{}{} }, c, initialInterval)

	if timeout == 0 {
		// Calculate the time needed for the poll to execute the commands.
		// An additional interval is included intentionally to act as a timeout buffer.
		timeout = time.Duration((expectedExecutions)*interval) * time.Second
	}
	return countSignals(done, expectedExecutions, timeout)
}

// countSignals counts the number of signals received through the `done` channel
// Returns in case the timeout has been reached or maxSignals received.
func countSignals(done <-chan struct{}, maxSignals int, timeout time.Duration) int {
	var counter int
	t := time.After(timeout)
	for {
		select {
		case <-t:
			return counter
		case <-done:
			counter++
			if counter == maxSignals {
				return counter
			}
		}
	}
}
