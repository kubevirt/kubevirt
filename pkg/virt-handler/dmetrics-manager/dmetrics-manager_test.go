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

package dmetrics_manager

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("DownwardMetrics Manager", func() {
	var manager *DownwardMetricsManager
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		manager = NewDownwardMetricsManager("test-node")
		vmi = &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi",
				Namespace: "default",
				UID:       types.UID("test-uid"),
			},
		}
	})

	Context("Server restart after unexpected exit", func() {
		It("should remove stale entry when server exits", func() {
			cancelCtx := func() {}

			// Simulate server entry in map
			manager.stopServer[vmi.UID] = cancelCtx
			Expect(manager.stopServer).To(HaveLen(1))

			// Simulate onExit callback when server exits
			onExit := func() {
				manager.lock.Lock()
				defer manager.lock.Unlock()
				if manager.done {
					return
				}
				delete(manager.stopServer, vmi.UID)
			}
			onExit()

			// Verify stale entry is removed
			Expect(manager.stopServer).To(BeEmpty())
		})

		It("should not remove entry if manager is shutting down", func() {
			manager.stopServer[vmi.UID] = func() {}
			manager.done = true

			// Simulate onExit during shutdown
			onExit := func() {
				manager.lock.Lock()
				defer manager.lock.Unlock()
				if manager.done {
					return
				}
				delete(manager.stopServer, vmi.UID)
			}
			onExit()

			// Verify entry is NOT removed (shutdown is in progress)
			Expect(manager.stopServer).To(HaveLen(1))
		})

		It("should handle concurrent server exits safely", func() {
			vmiCount := 10
			vmis := make([]*v1.VirtualMachineInstance, vmiCount)

			// Create separate VMI for each server
			for i := range vmiCount {
				extraVmi := &v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-vmi-" + string(rune(48+i)),
						Namespace: "default",
						UID:       types.UID("uid-" + string(rune(48+i))),
					},
				}
				vmis[i] = extraVmi

				manager.lock.Lock()
				manager.stopServer[extraVmi.UID] = func() {}
				manager.lock.Unlock()
			}

			Expect(manager.stopServer).To(HaveLen(vmiCount))

			// Simulate concurrent exits for each VMI
			done := make(chan struct{})
			for i := range vmiCount {
				go func(idx int) {
					defer GinkgoRecover()

					onExit := func() {
						manager.lock.Lock()
						defer manager.lock.Unlock()
						if manager.done {
							return
						}
						delete(manager.stopServer, vmis[idx].UID)
					}
					onExit()

					if idx == vmiCount-1 {
						close(done)
					}
				}(i)
			}

			Eventually(done, time.Second).Should(BeClosed())

			// All entries should be removed
			manager.lock.Lock()
			defer manager.lock.Unlock()
			Expect(manager.stopServer).To(BeEmpty())
		})
	})

	Context("StopServer behavior", func() {
		It("should stop server for any VMI reference", func() {
			cancelCalled := false
			cancelCtx := func() { cancelCalled = true }

			manager.stopServer[vmi.UID] = cancelCtx

			// StopServer should work for the VMI
			manager.StopServer(vmi)

			Expect(cancelCalled).To(BeTrue())
			Expect(manager.stopServer).To(BeEmpty())
		})

		It("should handle StopServer during manager shutdown", func() {
			manager.done = true
			manager.stopServer[vmi.UID] = func() {}

			// StopServer should return immediately if manager is shutting down
			manager.StopServer(vmi)

			// Entry should still exist (not cleaned up during shutdown)
			Expect(manager.stopServer).To(HaveLen(1))
		})

		It("should handle StopServer for non-existent server", func() {
			// Should not panic for VMI without a running server
			Expect(func() {
				manager.StopServer(vmi)
			}).NotTo(Panic())
		})
	})

	Context("Race conditions", func() {
		It("should handle race between StopServer and onExit", func() {
			manager.stopServer[vmi.UID] = func() {}

			done := make(chan struct{})

			// Goroutine: Server exits and calls onExit
			go func() {
				onExit := func() {
					manager.lock.Lock()
					defer manager.lock.Unlock()
					if manager.done {
						return
					}
					delete(manager.stopServer, vmi.UID)
				}
				onExit()
				close(done)
			}()

			// Meanwhile: Main thread calls StopServer simultaneously
			manager.StopServer(vmi)

			Eventually(done, time.Second).Should(BeClosed())

			// Verify no corruption, entry should be gone
			Expect(manager.stopServer).To(BeEmpty())
		})

		It("should handle server exit during shutdown sequence", func() {
			manager.stopServer[vmi.UID] = func() {}

			done := make(chan struct{})

			// Goroutine: Simulate manager shutdown iterating and deleting
			go func() {
				manager.lock.Lock()
				for uid := range manager.stopServer {
					delete(manager.stopServer, uid)
				}
				manager.lock.Unlock()
				close(done)
			}()

			// Meanwhile: Server is exiting and calling onExit concurrently
			onExit := func() {
				manager.lock.Lock()
				defer manager.lock.Unlock()
				if manager.done {
					return
				}
				delete(manager.stopServer, vmi.UID)
			}
			onExit()

			Eventually(done, time.Second).Should(BeClosed())

			// Verify no corruption
			Expect(manager.stopServer).To(BeEmpty())
		})
	})

	Context("Manager shutdown", func() {
		It("should stop all running servers on shutdown", func() {
			extraVmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi-extra",
					Namespace: "default",
					UID:       types.UID("uid-extra"),
				},
			}

			cancelCount := 0
			cancelCtx1 := func() { cancelCount++ }
			cancelCtx2 := func() { cancelCount++ }

			manager.lock.Lock()
			manager.stopServer[vmi.UID] = cancelCtx1
			manager.stopServer[extraVmi.UID] = cancelCtx2
			manager.lock.Unlock()

			stopCh := make(chan struct{})
			go manager.Run(stopCh)

			// Trigger shutdown
			close(stopCh)

			Eventually(func() int {
				return cancelCount
			}, time.Second).Should(Equal(2))
		})
	})
})
