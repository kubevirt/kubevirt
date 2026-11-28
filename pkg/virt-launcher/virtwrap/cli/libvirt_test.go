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

	Context("Upon deleting a domain checkpoint", func() {
		var conn *LibvirtConnection
		var domain *libvirt.Domain

		BeforeEach(func() {
			native, err := libvirt.NewConnect("test:///default")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() {
				Expect(native.Close()).To(BeZero())
			})
			conn = &LibvirtConnection{
				Connect:       native,
				uri:           "test:///default",
				alive:         true,
				reconnectLock: &sync.Mutex{},
			}
			domain, err = native.LookupDomainByName("test")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(domain.Free)
			checkpoint, err := domain.CreateCheckpointXML("<domaincheckpoint><name>old</name></domaincheckpoint>", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(checkpoint.Free()).To(Succeed())
		})

		It("should delete the checkpoint and keep its child", func() {
			child, err := domain.CreateCheckpointXML("<domaincheckpoint><name>new</name></domaincheckpoint>", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(child.Free()).To(Succeed())

			Expect(conn.DeleteDomainCheckpoint("test", "old")).To(Succeed())
			Expect(conn.alive).To(BeTrue())
			_, err = domain.CheckpointLookupByName("old", 0)
			Expect(err).To(HaveField("Code", libvirt.ERR_NO_DOMAIN_CHECKPOINT))
			child, err = domain.CheckpointLookupByName("new", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(child.Free()).To(Succeed())
		})

		DescribeTable("should return lookup errors",
			func(domainName, checkpointName string, code libvirt.ErrorNumber) {
				Expect(conn.DeleteDomainCheckpoint(domainName, checkpointName)).To(HaveField("Code", code))
				Expect(conn.alive).To(BeTrue())
			},
			Entry("domain missing", "missing", "old", libvirt.ERR_NO_DOMAIN),
			Entry("checkpoint missing", "test", "missing", libvirt.ERR_NO_DOMAIN_CHECKPOINT),
		)

		It("should return the deletion error and keep the checkpoint", func() {
			readOnly, err := libvirt.NewConnectReadOnly("test:///default")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() {
				Expect(readOnly.Close()).To(BeZero())
			})
			conn.Connect = readOnly

			Expect(conn.DeleteDomainCheckpoint("test", "old")).To(HaveField("Code", libvirt.ERR_OPERATION_DENIED))
			Expect(conn.alive).To(BeTrue())
			checkpoint, err := domain.CheckpointLookupByName("old", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(checkpoint.Free()).To(Succeed())
		})

		It("should reconnect before deleting the checkpoint", func() {
			original := conn.Connect
			DeferCleanup(func() {
				if conn.Connect != nil && conn.Connect != original {
					Expect(conn.Connect.Close()).To(BeZero())
				}
			})
			conn.alive = false

			Expect(conn.DeleteDomainCheckpoint("test", "old")).To(Succeed())
			Expect(conn.alive).To(BeTrue())
			Expect(conn.Connect).ToNot(BeIdenticalTo(original))
			_, err := domain.CheckpointLookupByName("old", 0)
			Expect(err).To(HaveField("Code", libvirt.ERR_NO_DOMAIN_CHECKPOINT))
		})

		It("should return the error when reconnecting fails", func() {
			conn.alive = false
			conn.uri = "unsupported:///"

			Expect(conn.DeleteDomainCheckpoint("test", "old")).To(HaveOccurred())
			Expect(conn.alive).To(BeFalse())
			checkpoint, err := domain.CheckpointLookupByName("old", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(checkpoint.Free()).To(Succeed())
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
})
