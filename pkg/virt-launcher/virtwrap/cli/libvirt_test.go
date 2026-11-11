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
		It("should not race with event dispatch", func() {
			const registrations = 50
			conn := &LibvirtConnection{}
			noopCallback := func(_ *libvirt.Connect, _ *libvirt.Domain) {}

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				for range registrations {
					for _, callback := range conn.reboot.callbacks() {
						Expect(callback).ToNot(BeNil())
					}
				}
			}()

			for range registrations {
				wg.Add(1)
				go func() {
					defer GinkgoRecover()
					defer wg.Done()
					Expect(conn.DomainEventRebootRegister(noopCallback)).To(Succeed())
				}()
			}
			wg.Wait()

			Expect(conn.reboot.callbacks()).To(HaveLen(registrations))
		})
	})

	Context("Upon reconnect", func() {
		It("should deliver events to current subscribers through the new connection", func() {
			Expect(libvirt.EventRegisterDefaultImpl()).To(Succeed())

			unusable := &libvirt.Connect{}
			conn := &LibvirtConnection{
				Connect:       unusable,
				uri:           "test:///default",
				reconnectLock: &sync.Mutex{},
			}

			kept := make(chan libvirt.DomainEventType, 10)
			Expect(conn.DomainEventLifecycleRegister(func(_ *libvirt.Connect, _ *libvirt.Domain, event *libvirt.DomainEventLifecycle) {
				kept <- event.Event
			})).To(Succeed())
			deregistered := make(chan libvirt.DomainEventType, 10)
			unsubscribe := conn.lifecycle.subscribe(func(_ *libvirt.Connect, _ *libvirt.Domain, event *libvirt.DomainEventLifecycle) {
				deregistered <- event.Event
			})

			dom, err := conn.LookupDomainByName("test")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(conn.Connect.Close)
			DeferCleanup(dom.Free)
			Expect(conn.Connect).ToNot(BeIdenticalTo(unusable))

			unsubscribe()

			Expect(dom.Suspend()).To(Succeed())
			DeferCleanup(dom.Resume)

			timer, err := libvirt.EventAddTimeout(0, func(int) {})
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(libvirt.EventRemoveTimeout, timer)
			Expect(libvirt.EventRunDefaultImpl()).To(Succeed())

			Expect(kept).To(HaveLen(1))
			Expect(kept).To(Receive(Equal(libvirt.DOMAIN_EVENT_SUSPENDED)))
			Expect(deregistered).To(BeEmpty())
		})
	})
})
