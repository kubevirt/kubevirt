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
 */

package synchronization

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("migrationTunnel", func() {
	It("enforces the per-tunnel channel concurrency semaphore", func() {
		tunnel := &migrationTunnel{
			channelSem: make(chan struct{}, 2),
		}

		By("acquiring up to the limit")
		Expect(tunnel.tryAcquireChannelSlot()).To(BeTrue())
		Expect(tunnel.tryAcquireChannelSlot()).To(BeTrue())

		By("rejecting an acquire beyond the limit")
		Expect(tunnel.tryAcquireChannelSlot()).To(BeFalse())

		By("releasing a slot and acquiring again")
		tunnel.releaseChannelSlot()
		Expect(tunnel.tryAcquireChannelSlot()).To(BeTrue())
	})

	It("closes listeners under lock and clears port maps", func() {
		By("attaching a live listener to the tunnel")
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		port := ln.Addr().(*net.TCPAddr).Port
		tunnel := &migrationTunnel{
			listeners:     []net.Listener{ln},
			listenerPorts: map[int]int{port: 49152},
		}

		By("closing listeners and verifying the maps are cleared")
		tunnel.closeListeners()
		Expect(tunnel.listeners).To(BeNil())
		Expect(tunnel.listenerPorts).To(BeEmpty())
		_, err = ln.Accept()
		Expect(err).To(HaveOccurred())
	})
})
