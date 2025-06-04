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

	Context("Load averaging functionality", func() {
		var fakeLoad api.Load

		BeforeEach(func() {
			fakeLoad = api.Load{
				Load1m:  1.23,
				Load5m:  2.45,
				Load15m: 3.67,
			}
		})

		Context("convertToLoad", func() {
			It("should convert libvirt DomainGuestInfo load to api.Load", func() {
				guestInfo := &libvirt.DomainGuestInfo{
					Load: &libvirt.DomainGuestInfoLoad{
						Load1M:  1.23,
						Load5M:  2.45,
						Load15M: 3.67,
					},
				}

				load := convertToLoad(guestInfo)

				Expect(load.Load1m).To(Equal(1.23))
				Expect(load.Load5m).To(Equal(2.45))
				Expect(load.Load15m).To(Equal(3.67))
			})

			It("should return empty Load when guestInfo.Load is nil", func() {
				guestInfo := &libvirt.DomainGuestInfo{
					Load: nil,
				}

				load := convertToLoad(guestInfo)

				Expect(load.Load1m).To(Equal(0.0))
				Expect(load.Load5m).To(Equal(0.0))
				Expect(load.Load15m).To(Equal(0.0))
			})

			It("should handle zero load values", func() {
				guestInfo := &libvirt.DomainGuestInfo{
					Load: &libvirt.DomainGuestInfoLoad{
						Load1M:  0.0,
						Load5M:  0.0,
						Load15M: 0.0,
					},
				}

				load := convertToLoad(guestInfo)

				Expect(load.Load1m).To(Equal(0.0))
				Expect(load.Load5m).To(Equal(0.0))
				Expect(load.Load15m).To(Equal(0.0))
			})

			It("should handle high load values", func() {
				guestInfo := &libvirt.DomainGuestInfo{
					Load: &libvirt.DomainGuestInfoLoad{
						Load1M:  100.50,
						Load5M:  200.75,
						Load15M: 300.99,
					},
				}

				load := convertToLoad(guestInfo)

				Expect(load.Load1m).To(Equal(100.50))
				Expect(load.Load5m).To(Equal(200.75))
				Expect(load.Load15m).To(Equal(300.99))
			})
		})

		Context("Load storage in AsyncAgentStore", func() {
			It("should store and retrieve load data", func() {
				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_LOAD, fakeLoad)

				sysInfo := agentStore.GetSysInfo()
				Expect(sysInfo.Load.Load1m).To(Equal(fakeLoad.Load1m))
				Expect(sysInfo.Load.Load5m).To(Equal(fakeLoad.Load5m))
				Expect(sysInfo.Load.Load15m).To(Equal(fakeLoad.Load15m))
			})

			It("should fire an event for new load data", func() {
				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_LOAD, fakeLoad)

				Expect(agentStore.AgentUpdated).To(Receive(Equal(AgentUpdatedEvent{
					DomainInfo: api.DomainGuestInfo{
						Interfaces:     nil,
						FSFreezeStatus: nil,
						OSInfo:         nil,
					},
				})))
			})

			It("should not fire an event for the same load data", func() {
				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_LOAD, fakeLoad)

				// Consume the first event
				Expect(agentStore.AgentUpdated).To(Receive())

				// Store the same load data again
				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_LOAD, fakeLoad)
				Expect(agentStore.AgentUpdated).ToNot(Receive())
			})

			It("should fire an event for updated load data", func() {
				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_LOAD, fakeLoad)

				// Consume the first event
				Expect(agentStore.AgentUpdated).To(Receive())

				updatedLoad := api.Load{
					Load1m:  5.12,
					Load5m:  6.34,
					Load15m: 7.56,
				}

				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_LOAD, updatedLoad)

				Expect(agentStore.AgentUpdated).To(Receive(Equal(AgentUpdatedEvent{
					DomainInfo: api.DomainGuestInfo{
						Interfaces:     nil,
						FSFreezeStatus: nil,
						OSInfo:         nil,
					},
				})))
			})

			It("should return empty Load when no load data exists", func() {
				sysInfo := agentStore.GetSysInfo()

				Expect(sysInfo.Load.Load1m).To(Equal(0.0))
				Expect(sysInfo.Load.Load5m).To(Equal(0.0))
				Expect(sysInfo.Load.Load15m).To(Equal(0.0))
			})
		})

		Context("fetchAndStoreGuestInfo with load data", func() {
			It("should fetch and store load info when requested", func() {
				guestInfo := &libvirt.DomainGuestInfo{
					Load: &libvirt.DomainGuestInfoLoad{
						Load1M:  2.34,
						Load5M:  3.45,
						Load15M: 4.56,
					},
				}
				agentPoller := &AgentPoller{
					Connection: mockConnection,
					domainName: "fake",
					agentStore: &agentStore,
				}

				mockDomain.EXPECT().Free()
				mockDomain.EXPECT().GetGuestInfo(libvirt.DOMAIN_GUEST_INFO_LOAD, uint32(0)).Return(guestInfo, nil)

				fetchAndStoreGuestInfo(libvirt.DOMAIN_GUEST_INFO_LOAD, agentPoller)

				sysInfo := agentStore.GetSysInfo()
				Expect(sysInfo.Load.Load1m).To(Equal(2.34))
				Expect(sysInfo.Load.Load5m).To(Equal(3.45))
				Expect(sysInfo.Load.Load15m).To(Equal(4.56))
			})

			It("should handle load info combined with other guest info types", func() {
				guestInfo := &libvirt.DomainGuestInfo{
					Hostname: "test-hostname",
					Load: &libvirt.DomainGuestInfoLoad{
						Load1M:  1.11,
						Load5M:  2.22,
						Load15M: 3.33,
					},
				}
				agentPoller := &AgentPoller{
					Connection: mockConnection,
					domainName: "fake",
					agentStore: &agentStore,
				}

				libvirtTypes := libvirt.DOMAIN_GUEST_INFO_HOSTNAME | libvirt.DOMAIN_GUEST_INFO_LOAD

				mockDomain.EXPECT().Free()
				mockDomain.EXPECT().GetGuestInfo(libvirtTypes, uint32(0)).Return(guestInfo, nil)

				fetchAndStoreGuestInfo(libvirtTypes, agentPoller)

				sysInfo := agentStore.GetSysInfo()
				Expect(sysInfo.Hostname).To(Equal("test-hostname"))
				Expect(sysInfo.Load.Load1m).To(Equal(1.11))
				Expect(sysInfo.Load.Load5m).To(Equal(2.22))
				Expect(sysInfo.Load.Load15m).To(Equal(3.33))
			})

			It("should store empty load when libvirt returns nil load", func() {
				guestInfo := &libvirt.DomainGuestInfo{
					Load: nil,
				}
				agentPoller := &AgentPoller{
					Connection: mockConnection,
					domainName: "fake",
					agentStore: &agentStore,
				}

				mockDomain.EXPECT().Free()
				mockDomain.EXPECT().GetGuestInfo(libvirt.DOMAIN_GUEST_INFO_LOAD, uint32(0)).Return(guestInfo, nil)

				fetchAndStoreGuestInfo(libvirt.DOMAIN_GUEST_INFO_LOAD, agentPoller)

				sysInfo := agentStore.GetSysInfo()
				Expect(sysInfo.Load.Load1m).To(Equal(0.0))
				Expect(sysInfo.Load.Load5m).To(Equal(0.0))
				Expect(sysInfo.Load.Load15m).To(Equal(0.0))
			})
		})

		Context("GetSysInfo with load data", func() {
			It("should include load data in sysinfo when available", func() {
				fakeOSInfo := api.GuestOSInfo{
					Name:    "TestOS",
					Version: "1.0",
				}
				fakeTimezone := api.Timezone{
					Zone:   "UTC",
					Offset: 0,
				}

				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_OS, fakeOSInfo)
				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_HOSTNAME, "test-host")
				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_TIMEZONE, fakeTimezone)
				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_LOAD, fakeLoad)

				sysInfo := agentStore.GetSysInfo()

				Expect(sysInfo.OSInfo.Name).To(Equal("TestOS"))
				Expect(sysInfo.OSInfo.Version).To(Equal("1.0"))
				Expect(sysInfo.Hostname).To(Equal("test-host"))
				Expect(sysInfo.Timezone.Zone).To(Equal("UTC"))
				Expect(sysInfo.Timezone.Offset).To(Equal(0))
				Expect(sysInfo.Load.Load1m).To(Equal(1.23))
				Expect(sysInfo.Load.Load5m).To(Equal(2.45))
				Expect(sysInfo.Load.Load15m).To(Equal(3.67))
			})

			It("should return empty load when only other sysinfo is available", func() {
				fakeOSInfo := api.GuestOSInfo{
					Name:    "TestOS",
					Version: "1.0",
				}

				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_OS, fakeOSInfo)
				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_HOSTNAME, "test-host")

				sysInfo := agentStore.GetSysInfo()

				Expect(sysInfo.OSInfo.Name).To(Equal("TestOS"))
				Expect(sysInfo.Hostname).To(Equal("test-host"))
				Expect(sysInfo.Load.Load1m).To(Equal(0.0))
				Expect(sysInfo.Load.Load5m).To(Equal(0.0))
				Expect(sysInfo.Load.Load15m).To(Equal(0.0))
			})
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
