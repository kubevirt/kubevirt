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

package prometheus

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k6tv1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("Collector", func() {
	var socketToVMI vmiSocketMap

	BeforeEach(func() {
		socketToVMI = make(vmiSocketMap)
		socketToVMI["a"] = &k6tv1.VirtualMachineInstance{}
		socketToVMI["b"] = &k6tv1.VirtualMachineInstance{}
		socketToVMI["c"] = &k6tv1.VirtualMachineInstance{}

	})

	Context("on running source", func() {
		It("should scrape all the sources", func() {
			fs := newFakeScraper()
			cc := NewConcurrentCollector(1)

			skipped, completed := cc.Collect(socketToVMI, fs, 1*time.Second)

			Expect(len(skipped)).To(Equal(0))
			Expect(completed).To(BeTrue())
		})
	})

	Context("on blocked source", func() {
		It("should gather the available data", func() {
			fs := newFakeScraper()
			fs.Block("a")
			cc := NewConcurrentCollector(1)

			skipped, completed := cc.Collect(socketToVMI, fs, 1*time.Second)

			Expect(len(skipped)).To(Equal(0))
			Expect(completed).To(BeFalse())
		})

		It("should skip it on later collections", func() {
			fs := newFakeScraper()
			fs.Block("a")
			cc := NewConcurrentCollector(2)

			By("Doing a first collection")
			skipped, completed := cc.Collect(socketToVMI, fs, 1*time.Second)
			// first collection is not aware of the blocked source
			Expect(len(skipped)).To(Equal(0))
			Expect(completed).To(BeFalse())

			By("Doing a second collection")
			skipped, completed = cc.Collect(socketToVMI, fs, 1*time.Second)
			// second collection is not aware of the blocked source
			Expect(len(skipped)).To(Equal(0))
			Expect(completed).To(BeFalse())

			By("Collecting again with a blocked source")
			skipped, completed = cc.Collect(socketToVMI, fs, 1*time.Second)
			// second collection is aware of the blocked source
			Expect(len(skipped)).To(Equal(1))
			Expect(skipped[0]).To(Equal("a"))
			Expect(completed).To(BeTrue())

		})

		It("should resume scraping when unblocks", func() {
			fs := newFakeScraper()
			fs.Block("b")
			cc := NewConcurrentCollector(1)

			By("Doing a first collection")
			skipped, completed := cc.Collect(socketToVMI, fs, 1*time.Second)
			// first collection is not aware of the blocked source
			Expect(len(skipped)).To(Equal(0))
			Expect(completed).To(BeFalse())

			By("Collecting again with a blocked source")
			skipped, completed = cc.Collect(socketToVMI, fs, 1*time.Second)
			// second collection is aware of the blocked source
			Expect(len(skipped)).To(Equal(1))
			Expect(skipped[0]).To(Equal("b"))
			Expect(completed).To(BeTrue())

			By("Unblocking the stuck source")
			ready := fs.Wakeup("b")
			<-ready
			fs.Unblock("b")

			By("Restored a clean state")
			skipped, completed = cc.Collect(socketToVMI, fs, 1*time.Second)
			Expect(len(skipped)).To(Equal(0))
			Expect(completed).To(BeTrue())
		})
	})
})

type fakeScraper struct {
	ready   map[string]chan bool
	blocked map[string]chan bool
}

func newFakeScraper() *fakeScraper {
	return &fakeScraper{
		blocked: make(map[string]chan bool),
		ready:   make(map[string]chan bool),
	}
}

// Unblock makes sure Scrape() WILL block. Call it when no goroutines are running (e.g. when concurrentCollector is known idle.)
func (fs *fakeScraper) Block(key string) {
	fs.blocked[key] = make(chan bool)
	fs.ready[key] = make(chan bool)
}

// Unblock makes sure Scrape() won't block. Call it when no goroutines are running (e.g. when concurrentCollector is known idle.)
func (fs *fakeScraper) Unblock(key string) {
	delete(fs.blocked, key)
	delete(fs.ready, key)
}

// Wakeup awakens a blocked Scrape(). Can be called when concurrentCollector is running.
func (fs *fakeScraper) Wakeup(key string) chan bool {
	if c, ok := fs.blocked[key]; ok {
		c <- true
		return fs.ready[key]
	}
	return nil
}

func (fs *fakeScraper) Scrape(key string, _ *k6tv1.VirtualMachineInstance) {
	if c, ok := fs.blocked[key]; ok {
		<-c
		fs.ready[key] <- true
	}
}
