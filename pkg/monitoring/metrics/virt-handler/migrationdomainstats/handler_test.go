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
	})
})
