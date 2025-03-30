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

package collector

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Collector", func() {
	var vmis []*k6tv1.VirtualMachineInstance

	newCollector := func(retries int) Collector {
		fakeMapper := func(vmi []*k6tv1.VirtualMachineInstance) vmiSocketMap {
			m := vmiSocketMap{}
			for _, vmi := range vmi {
				m[vmi.Name] = vmi
			}
			return m
		}
		collector := NewConcurrentCollectorWithMapper(retries, fakeMapper)
		return collector
	}

	BeforeEach(func() {
		vmis = []*k6tv1.VirtualMachineInstance{
			{ObjectMeta: metav1.ObjectMeta{Name: "a"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "b"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "c"}},
		}
	})

	Context("on running source", func() {
		It("should scrape all the sources", func() {
			fs := newFakeScraper()
			cc := NewConcurrentCollector(1)

			skipped, completed := cc.Collect(vmis, fs, 1*time.Second)

			Expect(skipped).To(BeEmpty())
			Expect(completed).To(BeTrue())
		})
	})

	Context("on blocked source", func() {
		It("should gather the available data", func() {
			fs := newFakeScraper()
			fs.Block("a")
			cc := newCollector(1)

			skipped, completed := cc.Collect(vmis, fs, 1*time.Second)

			Expect(skipped).To(BeEmpty())
			Expect(completed).To(BeFalse())
		})

		It("should skip it on later collections", func() {
			fs := newFakeScraper()
			fs.Block("a")
			cc := newCollector(2)

			By("Doing a first collection")
			skipped, completed := cc.Collect(vmis, fs, 1*time.Second)
			// first collection is not aware of the blocked source
			Expect(skipped).To(BeEmpty())
			Expect(completed).To(BeFalse())

			By("Doing a second collection")
			skipped, completed = cc.Collect(vmis, fs, 1*time.Second)
			// second collection is not aware of the blocked source
			Expect(skipped).To(BeEmpty())
			Expect(completed).To(BeFalse())

			By("Collecting again with a blocked source")
			skipped, completed = cc.Collect(vmis, fs, 1*time.Second)
			// second collection is aware of the blocked source
			Expect(skipped).To(HaveLen(1))
			Expect(skipped[0]).To(Equal("a"))
			Expect(completed).To(BeTrue())
		})

		It("should resume scraping when unblocks", func() {
			fs := newFakeScraper()
			fs.Block("b")
			cc := newCollector(1)

			By("Doing a first collection")
			skipped, completed := cc.Collect(vmis, fs, 1*time.Second)
			// first collection is not aware of the blocked source
			Expect(skipped).To(BeEmpty())
			Expect(completed).To(BeFalse())

			By("Collecting again with a blocked source")
			skipped, completed = cc.Collect(vmis, fs, 1*time.Second)
			// second collection is aware of the blocked source
			Expect(skipped).To(HaveLen(1))
			Expect(skipped[0]).To(Equal("b"))
			Expect(completed).To(BeTrue())

			By("Unblocking the stuck source")
			ready := fs.Wakeup("b")
			<-ready
			fs.Unblock("b")

			By("Restored a clean state")
			skipped, completed = cc.Collect(vmis, fs, 1*time.Second)
			Expect(skipped).To(BeEmpty())
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

// Unblock makes sure Scrape() WILL block. Call it when no goroutines are running (e.g. when ConcurrentCollector is known idle.)
func (fs *fakeScraper) Block(key string) {
	fs.blocked[key] = make(chan bool)
	fs.ready[key] = make(chan bool)
}

// Unblock makes sure Scrape() won't block. Call it when no goroutines are running (e.g. when ConcurrentCollector is known idle.)
func (fs *fakeScraper) Unblock(key string) {
	delete(fs.blocked, key)
	delete(fs.ready, key)
}

// Wakeup awakens a blocked Scrape(). Can be called when ConcurrentCollector is running.
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

func (fs *fakeScraper) Complete() {}
