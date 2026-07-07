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

package migrationdomainstats

import (
	"container/ring"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"

	domstatsCollector "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/collector"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

type possiblyFinishedQueue struct {
	isFinished     bool
	completedStats *stats.DomainJobInfo
}

func (q *possiblyFinishedQueue) addCompletedStats(domainJobInfo stats.DomainJobInfo) {
	q.isFinished = true
	q.completedStats = &domainJobInfo
}

func (q *possiblyFinishedQueue) all() ([]result, bool) {
	return nil, q.isFinished
}

func (*possiblyFinishedQueue) startPolling() {
	panic("not implemented")
}

type noOpCollector struct{}

func (noOpCollector) Collect(_ []*v1.VirtualMachineInstance, scraper domstatsCollector.MetricsScraper, _ time.Duration) ([]string, bool) {
	scraper.Complete()
	return nil, true
}

var _ = Describe("Handler", func() {
	var h *handler

	BeforeEach(func() {
		h = &handler{
			vmiStats: make(map[string]vmiQueue),
		}
	})

	Describe("Collect", func() {
		Context("when queue is finished", func() {
			It("should delete the queue after collecting results", func() {
				h.vmiStats["default/test-vmi"] = &possiblyFinishedQueue{
					isFinished: true,
				}
				h.Collect()

				Expect(h.vmiStats).To(BeEmpty())
			})
		})

		Context("when queue is not finished", func() {
			It("should keep the queue after collecting results", func() {
				h.vmiStats["default/test-vmi"] = &possiblyFinishedQueue{
					isFinished: false,
				}
				h.Collect()

				Expect(h.vmiStats).To(HaveKey("default/test-vmi"))
			})
		})
	})

	Describe("queue results", func() {
		It("should store completed migration stats from domain updates", func() {
			q := &possiblyFinishedQueue{}
			h.vmiStats["default/test-vmi"] = q

			h.handleDomainAdd(&api.Domain{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vmi", Namespace: "default"},
				Status: api.DomainStatus{
					MigrationStats: &stats.DomainJobInfo{DowntimeSet: true, Downtime: 70},
				},
			})

			Expect(q.isFinished).To(BeTrue())
			Expect(q.completedStats).ToNot(BeNil())
			Expect(q.completedStats.Downtime).To(Equal(uint64(70)))
		})

		It("should store completed migration stats when the domain update arrives before the VMI update creates a queue", func() {
			vmiStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
			vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "test-vmi", Namespace: "default"}}
			Expect(vmiStore.Add(vmi)).To(Succeed())
			h.vmiStore = vmiStore

			h.handleDomainAdd(&api.Domain{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vmi", Namespace: "default"},
				Status: api.DomainStatus{
					MigrationStats: &stats.DomainJobInfo{DowntimeSet: true, Downtime: 70},
				},
			})

			results := h.Collect()
			Expect(results).To(HaveLen(1))
			Expect(results[0].domainJobInfo.DowntimeSet).To(BeTrue())
			Expect(results[0].domainJobInfo.Downtime).To(Equal(uint64(70)))
			Expect(h.vmiStats).To(BeEmpty())
		})

		It("should identify completed downtime stats", func() {
			Expect(hasCompletedDowntimeStats(stats.DomainJobInfo{})).To(BeFalse())
			Expect(hasCompletedDowntimeStats(stats.DomainJobInfo{DowntimeSet: true})).To(BeTrue())
			Expect(hasCompletedDowntimeStats(stats.DomainJobInfo{DowntimeNetSet: true})).To(BeTrue())
		})

		It("should stop polling after completed downtime stats are observed", func() {
			q := &queue{}

			Expect(q.shouldStopPolling(false, stats.DomainJobInfo{})).To(BeFalse())
			Expect(q.shouldStopPolling(true, stats.DomainJobInfo{})).To(BeFalse())
			Expect(q.shouldStopPolling(true, stats.DomainJobInfo{DowntimeSet: true})).To(BeTrue())
		})

		It("should stop polling after completed downtime stats timeout", func() {
			completedAt := time.Now().Add(-completedStatsTimeout)
			q := &queue{completedAt: &completedAt}

			Expect(q.shouldStopPolling(true, stats.DomainJobInfo{})).To(BeTrue())
		})

		It("should report finished only after the final collection was stored", func() {
			q := &queue{results: ring.New(bufferSize)}
			q.results.Value = result{vmi: "active"}

			results, finished := q.all()
			Expect(results).To(HaveLen(1))
			Expect(finished).To(BeFalse())

			q = &queue{results: ring.New(bufferSize), finished: true}
			q.results.Value = result{vmi: "final"}

			results, finished = q.all()
			Expect(results).To(HaveLen(1))
			Expect(finished).To(BeTrue())
		})

		It("should not report finished before the final collection is stored", func() {
			q := &queue{results: ring.New(bufferSize)}

			results, finished := q.all()
			Expect(results).To(BeEmpty())
			Expect(finished).To(BeFalse())
		})

		It("should not overwrite completed stats after the queue is finished", func() {
			q := &queue{
				vmi:     &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "test-vmi", Namespace: "default"}},
				results: ring.New(bufferSize),
			}
			q.addCompletedStats(stats.DomainJobInfo{DowntimeSet: true, Downtime: 70})

			q.collect()

			results, finished := q.all()
			Expect(results).To(HaveLen(1))
			Expect(finished).To(BeTrue())
			Expect(results[0].domainJobInfo.DowntimeSet).To(BeTrue())
			Expect(results[0].domainJobInfo.Downtime).To(Equal(uint64(70)))
		})

		It("should stop polling after completed stats timeout even when scraping fails", func() {
			completedAt := time.Now().Add(-completedStatsTimeout)
			vmiStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
			q := &queue{
				vmiStore:    vmiStore,
				vmi:         &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "test-vmi", Namespace: "default"}},
				collector:   noOpCollector{},
				results:     ring.New(bufferSize),
				completedAt: &completedAt,
			}

			q.collect()

			_, finished := q.all()
			Expect(finished).To(BeTrue())
		})
	})
})
