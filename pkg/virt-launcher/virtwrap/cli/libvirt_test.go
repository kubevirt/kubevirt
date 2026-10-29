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
			conn = &LibvirtConnection{
				alive:         true,
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
		AfterEach(func() {
			newLibvirtConnection = newConnection
			closeLibvirtConnection = func(c *libvirt.Connect) error {
				if c == nil {
					return nil
				}
				_, err := c.Close()
				return err
			}
		})

		It("should close the previous connection so abandoned callbacks cannot keep delivering", func() {
			oldConn := &libvirt.Connect{}
			newConn := &libvirt.Connect{}
			var closed *libvirt.Connect

			newLibvirtConnection = func(string, string, string) (*libvirt.Connect, error) {
				return newConn, nil
			}
			closeLibvirtConnection = func(c *libvirt.Connect) error {
				closed = c
				return nil
			}

			conn := &LibvirtConnection{
				Connect:       oldConn,
				alive:         false,
				reconnectLock: &sync.Mutex{},
				uri:           "test:///default",
			}

			Expect(conn.reconnectIfNecessaryLocked()).To(Succeed())
			Expect(conn.Connect).To(BeIdenticalTo(newConn))
			Expect(conn.alive).To(BeTrue())
			Expect(closed).To(BeIdenticalTo(oldConn))
		})

		It("should not close a connection when reconnect is unnecessary", func() {
			existing := &libvirt.Connect{}
			closeCalled := false
			closeLibvirtConnection = func(*libvirt.Connect) error {
				closeCalled = true
				return nil
			}
			newLibvirtConnection = func(string, string, string) (*libvirt.Connect, error) {
				Fail("new connection should not be opened when already alive")
				return nil, nil
			}

			conn := &LibvirtConnection{
				Connect:       existing,
				alive:         true,
				reconnectLock: &sync.Mutex{},
			}

			Expect(conn.reconnectIfNecessaryLocked()).To(Succeed())
			Expect(conn.Connect).To(BeIdenticalTo(existing))
			Expect(closeCalled).To(BeFalse())
		})

		It("should leave the previous connection in place if opening a new one fails", func() {
			oldConn := &libvirt.Connect{}
			closeCalled := false
			closeLibvirtConnection = func(*libvirt.Connect) error {
				closeCalled = true
				return nil
			}
			newLibvirtConnection = func(string, string, string) (*libvirt.Connect, error) {
				return nil, fmt.Errorf("dial failed")
			}

			conn := &LibvirtConnection{
				Connect:       oldConn,
				alive:         false,
				reconnectLock: &sync.Mutex{},
				uri:           "test:///default",
			}

			Expect(conn.reconnectIfNecessaryLocked()).To(MatchError("dial failed"))
			Expect(conn.Connect).To(BeIdenticalTo(oldConn))
			Expect(conn.alive).To(BeFalse())
			Expect(closeCalled).To(BeFalse())
		})
	})
})
