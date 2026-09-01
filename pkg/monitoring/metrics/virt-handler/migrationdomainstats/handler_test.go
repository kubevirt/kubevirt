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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	return completedMigrationDomainAt(downtime, migrationUID, time.Unix(100, 0))
}

func completedMigrationDomainAt(downtime uint64, migrationUID string, start time.Time) *api.Domain {
	return &api.Domain{
		ObjectMeta: metav1.ObjectMeta{Name: "test-vmi", Namespace: "default", UID: "vmi-uid"},
		Spec: api.DomainSpec{
			Metadata: api.Metadata{KubeVirt: api.KubeVirtMetadata{
				UID: "vmi-uid",
				Migration: &api.MigrationMetadata{
					UID:            types.UID(migrationUID),
					StartTimestamp: &metav1.Time{Time: start},
				},
			}},
		},
		Status: api.DomainStatus{
			CompletedMigrationStats: &api.CompletedMigrationStats{DowntimeSet: true, Downtime: downtime},
		},
	}
}

var _ = Describe("Handler", func() {
	var h *handler

	BeforeEach(func() {
		h = &handler{
			vmiStats:                make(map[string]vmiQueue),
			nodeName:                "source-node",
			completedMigrationStats: make(map[string]completedMigrationResult),
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
					MigrationState: &v1.VirtualMachineInstanceMigrationState{
						MigrationUID:   "migration-1",
						StartTimestamp: &metav1.Time{Time: time.Unix(100, 0)},
					},
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
			h.handleDomainCompletedMigrationStats(domain)

			Expect(q.isFinished).To(BeFalse())
			Expect(h.vmiStats["default/test-vmi"]).To(BeIdenticalTo(q))

			completedStats := h.completedMigrationStats["default/test-vmi"]
			Expect(completedStats.node).To(Equal("source-node"))
			Expect(completedStats.vmiUID).To(Equal("vmi-uid"))
			Expect(completedStats.migrationUID).To(Equal("migration-1"))
			Expect(completedStats.downtime).To(Equal(uint64(70)))

			results := h.Collect()
			Expect(results).To(HaveLen(2))
			Expect(results[0].domainJobInfo).To(Equal(stats.DomainJobInfo{DataTotalSet: true, DataTotal: 123}))
			Expect(results[1].domainJobInfo).To(Equal(completedStats.domainJobInfo))
			Expect(results[1].timestamp).To(BeZero())

			h.handleDomainCompletedMigrationStats(completedMigrationDomain(80, "migration-1"))
			Expect(h.completedMigrationStats["default/test-vmi"].downtime).To(Equal(uint64(70)))

			h.handleDomainCompletedMigrationStats(completedMigrationDomainAt(90, "migration-2", time.Unix(200, 0)))
			Expect(h.completedMigrationStats["default/test-vmi"].migrationUID).To(Equal("migration-2"))
			Expect(h.completedMigrationStats["default/test-vmi"].downtime).To(Equal(uint64(90)))

			results = h.Collect()
			Expect(results).To(HaveLen(2))
			Expect(results[1].downtimeSet).To(BeTrue())
			Expect(results[1].downtime).To(Equal(uint64(90)))
		})

		It("should drop stats when the migration metadata is missing", func() {
			domain := completedMigrationDomain(70, "migration-1")
			domain.Spec.Metadata.KubeVirt.Migration = nil

			h.handleDomainCompletedMigrationStats(domain)

			Expect(h.completedMigrationStats).To(BeEmpty())
		})

		It("should drop stats when the migration UID is empty", func() {
			h.handleDomainCompletedMigrationStats(completedMigrationDomain(70, ""))

			Expect(h.completedMigrationStats).To(BeEmpty())
		})

		It("should drop stats when the migration start timestamp is missing", func() {
			domain := completedMigrationDomain(70, "migration-1")
			domain.Spec.Metadata.KubeVirt.Migration.StartTimestamp = nil

			h.handleDomainCompletedMigrationStats(domain)

			Expect(h.completedMigrationStats).To(BeEmpty())
		})

		It("should drop stats from a domain belonging to another VMI instance", func() {
			domain := completedMigrationDomain(70, "migration-1")
			domain.Spec.Metadata.KubeVirt.UID = "old-vmi-uid"

			h.handleDomainCompletedMigrationStats(domain)

			Expect(h.completedMigrationStats).To(BeEmpty())
		})

		It("should drop stats when the domain UID is missing", func() {
			domain := completedMigrationDomain(70, "migration-1")
			domain.Spec.Metadata.KubeVirt.UID = ""

			h.handleDomainCompletedMigrationStats(domain)

			Expect(h.completedMigrationStats).To(BeEmpty())
		})

		It("should retain new completed stats when the VMI informer still has the prior migration", func() {
			vmi.Status.MigrationState.Completed = true
			Expect(vmiStore.Update(vmi)).To(Succeed())

			h.handleDomainCompletedMigrationStats(completedMigrationDomainAt(90, "migration-2", time.Unix(200, 0)))

			Expect(h.Collect()).To(HaveLen(1))
			Expect(h.completedMigrationStats["default/test-vmi"].migrationUID).To(Equal("migration-2"))
		})

		It("should drop an older domain replay after a newer successful migration", func() {
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID:   "migration-2",
				StartTimestamp: &metav1.Time{Time: time.Unix(200, 0)},
				Completed:      true,
			}
			Expect(vmiStore.Update(vmi)).To(Succeed())

			h.handleDomainCompletedMigrationStats(completedMigrationDomain(70, "migration-1"))

			Expect(h.completedMigrationStats).To(BeEmpty())
		})

		It("should not replace stored stats with an older domain replay", func() {
			h.handleDomainCompletedMigrationStats(completedMigrationDomainAt(90, "migration-2", time.Unix(200, 0)))

			h.handleDomainCompletedMigrationStats(completedMigrationDomain(70, "migration-1"))

			Expect(h.completedMigrationStats["default/test-vmi"].migrationUID).To(Equal("migration-2"))
			Expect(h.completedMigrationStats["default/test-vmi"].downtime).To(Equal(uint64(90)))
		})

		It("should store completed migration stats when the domain update arrives before the VMI update creates a queue", func() {
			h.handleDomainCompletedMigrationStats(completedMigrationDomain(70, "migration-1"))

			results := h.Collect()
			Expect(results).To(HaveLen(1))
			Expect(results[0].downtimeSet).To(BeTrue())
			Expect(results[0].downtime).To(Equal(uint64(70)))
			Expect(h.vmiStats).To(BeEmpty())
			Expect(h.completedMigrationStats).To(HaveKey("default/test-vmi"))

			results = h.Collect()
			Expect(results).To(HaveLen(1))
			Expect(results[0].downtimeSet).To(BeTrue())
			Expect(results[0].downtime).To(Equal(uint64(70)))

			Expect(vmiStore.Delete(vmi)).To(Succeed())
			h.Collect()
			Expect(h.completedMigrationStats).To(BeEmpty())
		})

		It("should discard stats when a VMI is recreated with the same name", func() {
			vmi.UID = "old-vmi-uid"
			Expect(vmiStore.Update(vmi)).To(Succeed())
			domain := completedMigrationDomain(70, "migration-1")
			domain.Spec.Metadata.KubeVirt.UID = vmi.UID
			h.handleDomainCompletedMigrationStats(domain)

			Expect(vmiStore.Update(&v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vmi", Namespace: "default", UID: "new-vmi-uid"},
			})).To(Succeed())

			Expect(h.Collect()).To(BeEmpty())
			Expect(h.completedMigrationStats).To(BeEmpty())
		})

		DescribeTable("should retain only the latest successful migration result", func(failed bool, expectedResults int, expectedMigrationUID string, expectedDowntime uint64) {
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID:   "migration-1",
				StartTimestamp: &metav1.Time{Time: time.Unix(100, 0)},
			}
			Expect(vmiStore.Update(vmi)).To(Succeed())
			h.handleDomainCompletedMigrationStats(completedMigrationDomain(70, "migration-1"))

			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID:   "migration-2",
				StartTimestamp: &metav1.Time{Time: time.Unix(200, 0)},
				Completed:      true,
				Failed:         failed,
			}
			Expect(vmiStore.Update(vmi)).To(Succeed())

			results := h.Collect()
			Expect(results).To(HaveLen(expectedResults))
			if expectedResults > 0 {
				Expect(h.completedMigrationStats["default/test-vmi"].migrationUID).To(Equal(expectedMigrationUID))
				Expect(results[0].downtime).To(Equal(expectedDowntime))
			}
		},
			Entry("discarding the prior result after a newer success", false, 0, "", uint64(0)),
			Entry("retaining the prior result after a newer failure", true, 1, "migration-1", uint64(70)),
		)
	})
})
