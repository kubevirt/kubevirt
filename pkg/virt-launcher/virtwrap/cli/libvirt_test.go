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

package cli

import (
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"libvirt.org/go/libvirt"
)

var _ = Describe("Libvirt Suite", func() {
	Context("Upon attempt to connect to Libvirt", func() {
		It("should time out while waiting for libvirt", func() {
			_, err := NewConnectionWithTimeout("http://", "", "", 1*time.Microsecond, 100*time.Millisecond, 500*time.Millisecond)
			Expect(err).To(MatchError("cannot connect to libvirt daemon: context deadline exceeded"))
		})
	})

	Context("Upon registering event callbacks", func() {
		const registrations = 50

		var conn *LibvirtConnection

		noopCallback := func(_ *libvirt.Connect, _ *libvirt.Domain) {}

		BeforeEach(func() {
			virConn, err := libvirt.NewConnect("test:///default")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(virConn.Close)
			conn = &LibvirtConnection{
				Connect:       virConn,
				state:         &connState{},
				reconnectLock: &sync.Mutex{},
			}
		})

		It("should store the callback before registering it and hold the reconnect lock for both", func() {
			registerErr := fmt.Errorf("register failed")
			register := func(_ libvirt.DomainEventGenericCallback) (int, error) {
				Expect(conn.reconnectLock.TryLock()).To(BeFalse())
				Expect(conn.domainRebootEventCallbacks).To(HaveLen(1))
				return 0, registerErr
			}

			Expect(registerEventCallback(conn, &conn.domainRebootEventCallbacks, noopCallback, register)).To(MatchError(registerErr))
			Expect(conn.domainRebootEventCallbacks).To(HaveLen(1))
		})

		It("should not race with callback replay", func() {
			register := func(_ libvirt.DomainEventGenericCallback) (int, error) {
				return 0, nil
			}

			var wg sync.WaitGroup
			wg.Add(1)
			// Mimic the replay loop of reconnectIfNecessaryLocked.
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				for range registrations {
					conn.reconnectLock.Lock()
					for _, callback := range conn.domainRebootEventCallbacks {
						Expect(callback).ToNot(BeNil())
					}
					conn.reconnectLock.Unlock()
				}
			}()

			for range registrations {
				wg.Add(1)
				go func() {
					defer GinkgoRecover()
					defer wg.Done()
					Expect(registerEventCallback(conn, &conn.domainRebootEventCallbacks, noopCallback, register)).To(Succeed())
				}()
			}
			wg.Wait()

			Expect(conn.domainRebootEventCallbacks).To(HaveLen(registrations))
		})
	})

	Context("Upon reconnect", func() {
		It("should replace an unusable connection and replay its callbacks", func() {
			Expect(libvirt.EventRegisterDefaultImpl()).To(Succeed())

			unusable, err := libvirt.NewConnect("test:///default")
			Expect(err).ToNot(HaveOccurred())
			Expect(unusable.Close()).To(BeZero())

			events := make(chan libvirt.DomainEventType, 10)
			conn := &LibvirtConnection{
				Connect:       unusable,
				state:         &connState{},
				uri:           "test:///default",
				reconnectLock: &sync.Mutex{},
				domainEventCallbacks: []libvirt.DomainEventLifecycleCallback{
					func(_ *libvirt.Connect, _ *libvirt.Domain, event *libvirt.DomainEventLifecycle) {
						events <- event.Event
					},
				},
			}

			dom, err := conn.LookupDomainByName("test")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(conn.Connect.Close)
			DeferCleanup(dom.Free)
			Expect(conn.Connect).ToNot(BeIdenticalTo(unusable))

			Expect(dom.Suspend()).To(Succeed())
			DeferCleanup(dom.Resume)

			timer, err := libvirt.EventAddTimeout(0, func(int) {})
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(libvirt.EventRemoveTimeout, timer)
			Expect(libvirt.EventRunDefaultImpl()).To(Succeed())

			Expect(events).To(HaveLen(1))
			Expect(events).To(Receive(Equal(libvirt.DOMAIN_EVENT_SUSPENDED)))
		})

		It("should keep a connection that is still alive after a failed call", func() {
			conn, err := NewConnection("test:///default", "", "", time.Hour)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(conn.Close)

			lvConn := conn.(*LibvirtConnection)
			previous := lvConn.Connect
			_, err = conn.LookupDomainByName("missing")
			Expect(err).To(HaveOccurred())
			Expect(lvConn.current()).To(BeIdenticalTo(previous))
		})
	})
})
