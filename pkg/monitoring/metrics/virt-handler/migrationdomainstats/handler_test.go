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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	ioprometheusclient "github.com/prometheus/client_model/go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

type possiblyFinishedQueue struct {
	isFinished bool
	results    []result
}

func (q *possiblyFinishedQueue) all() ([]result, bool) {
	return q.results, q.isFinished
}

func (*possiblyFinishedQueue) startPolling() {
	panic("not implemented")
}

func completedMigrationDomain(downtime uint64, migrationUID string) *api.Domain {
	return &api.Domain{
		ObjectMeta: metav1.ObjectMeta{Name: "test-vmi", Namespace: "default"},
		Spec: api.DomainSpec{
			Metadata: api.Metadata{KubeVirt: api.KubeVirtMetadata{
				Migration: &api.MigrationMetadata{UID: types.UID(migrationUID)},
			}},
		},
		Status: api.DomainStatus{
			MigrationStats: &stats.DomainJobInfo{DowntimeSet: true, Downtime: downtime},
		},
	}
}

var _ = Describe("Handler", func() {
	var h *handler

	BeforeEach(func() {
		migrationDowntime.Reset()
		h = &handler{
			vmiStats:       make(map[string]vmiQueue),
			nodeName:       "source-node",
			completedStats: make(map[string]completedMigrationResult),
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

	Describe("completed migration results", func() {
		var vmiStore cache.Store
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmiStore = cache.NewStore(cache.MetaNamespaceKeyFunc)
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vmi", Namespace: "default", UID: "vmi-uid"},
				Status: v1.VirtualMachineInstanceStatus{
					MigrationState: &v1.VirtualMachineInstanceMigrationState{MigrationUID: "migration-1"},
				},
			}
			Expect(vmiStore.Add(vmi)).To(Succeed())
			h.globalVMIStore = vmiStore
		})

		It("should store completed downtime without modifying the active queue", func() {
			q := &possiblyFinishedQueue{
				results: []result{{domainJobInfo: stats.DomainJobInfo{DataTotalSet: true, DataTotal: 123}}},
			}
			h.vmiStats["default/test-vmi"] = q

			domain := completedMigrationDomain(70, "migration-1")
			domain.Status.MigrationStats.DataTotalSet = true
			domain.Status.MigrationStats.DataTotal = 999
			h.handleDomainCompletedMigrationStats(domain)

			Expect(q.isFinished).To(BeFalse())
			Expect(h.vmiStats["default/test-vmi"]).To(BeIdenticalTo(q))

			completedStats := h.completedStats["default/test-vmi"]
			Expect(completedStats.node).To(Equal("source-node"))
			Expect(completedStats.vmiUID).To(Equal("vmi-uid"))
			Expect(completedStats.migrationUID).To(Equal("migration-1"))
			Expect(completedStats.domainJobInfo.Downtime).To(Equal(uint64(70)))
			Expect(completedStats.domainJobInfo.DataTotalSet).To(BeFalse())
			Expect(migrationDowntimeSampleCount()).To(Equal(uint64(1)))

			results := h.Collect()
			Expect(results).To(HaveLen(2))
			Expect(results[0].domainJobInfo).To(Equal(stats.DomainJobInfo{DataTotalSet: true, DataTotal: 123}))
			Expect(results[1].domainJobInfo).To(Equal(completedStats.domainJobInfo))
			Expect(results[1].timestamp).To(BeZero())

			h.handleDomainCompletedMigrationStats(completedMigrationDomain(80, "migration-1"))
			Expect(h.completedStats["default/test-vmi"].domainJobInfo.Downtime).To(Equal(uint64(80)))
			Expect(migrationDowntimeSampleCount()).To(Equal(uint64(1)))

			h.handleDomainCompletedMigrationStats(completedMigrationDomain(90, "migration-2"))
			Expect(migrationDowntimeSampleCount()).To(Equal(uint64(2)))
		})

		It("should store completed migration stats when the domain update arrives before the VMI update creates a queue", func() {
			h.handleDomainCompletedMigrationStats(completedMigrationDomain(70, "migration-1"))

			results := h.Collect()
			Expect(results).To(HaveLen(1))
			Expect(results[0].domainJobInfo.DowntimeSet).To(BeTrue())
			Expect(results[0].domainJobInfo.Downtime).To(Equal(uint64(70)))
			Expect(h.vmiStats).To(BeEmpty())
			Expect(h.completedStats).To(HaveKey("default/test-vmi"))

			results = h.Collect()
			Expect(results).To(HaveLen(1))
			Expect(results[0].domainJobInfo.DowntimeSet).To(BeTrue())
			Expect(results[0].domainJobInfo.Downtime).To(Equal(uint64(70)))

			Expect(vmiStore.Delete(vmi)).To(Succeed())
			h.Collect()
			Expect(h.completedStats).To(BeEmpty())
		})

		It("should discard stats when a VMI is recreated with the same name", func() {
			vmi.UID = "old-vmi-uid"
			Expect(vmiStore.Update(vmi)).To(Succeed())
			h.handleDomainCompletedMigrationStats(completedMigrationDomain(70, "migration-1"))

			Expect(vmiStore.Update(&v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vmi", Namespace: "default", UID: "new-vmi-uid"},
			})).To(Succeed())

			Expect(h.Collect()).To(BeEmpty())
			Expect(h.completedStats).To(BeEmpty())
		})

		DescribeTable("should retain only the latest successful migration result", func(failed bool, expectedResults int) {
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{MigrationUID: "migration-1"}
			Expect(vmiStore.Update(vmi)).To(Succeed())
			h.handleDomainCompletedMigrationStats(completedMigrationDomain(70, "migration-1"))

			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "migration-2",
				Completed:    true,
				Failed:       failed,
			}
			Expect(vmiStore.Update(vmi)).To(Succeed())

			Expect(h.Collect()).To(HaveLen(expectedResults))
		},
			Entry("discarding the prior result after a newer success", false, 0),
			Entry("retaining the prior result after a newer failure", true, 1),
		)
	})
})

func migrationDowntimeSampleCount() uint64 {
	metrics := make(chan prometheus.Metric, 10)
	migrationDowntime.Collect(metrics)
	close(metrics)

	var count uint64
	for metric := range metrics {
		value := &ioprometheusclient.Metric{}
		ExpectWithOffset(1, metric.Write(value)).To(Succeed())
		count += value.GetHistogram().GetSampleCount()
	}
	return count
}
