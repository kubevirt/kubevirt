/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package domainstats

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/testing"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/collector"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("domain stats collector", func() {
	Context("Collect", func() {
		vmis := []*k6tv1.VirtualMachineInstance{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi-1",
					Namespace: "test-ns-1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi-1",
					Namespace: "test-ns-1",
				},
			},
		}

		vmiStats := []*VirtualMachineInstanceStats{
			{
				DomainStats: &stats.DomainStats{
					Memory: &stats.DomainStatsMemory{
						RSSSet: true,
						RSS:    1,
					},
				},
			},
			{
				DomainStats: &stats.DomainStats{
					Memory: &stats.DomainStatsMemory{
						RSSSet: true,
						RSS:    2,
					},
				},
			},
		}

		It("should collect metrics for each vmi", func() {
			concCollector := fakeCollector{
				vmis:     vmis,
				vmiStats: vmiStats,
			}
			crs := execDomainStatsCollector(concCollector, vmis)
			Expect(crs).To(HaveLen(2))
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(memoryResident, kibibytesToBytes(1))))
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(memoryResident, kibibytesToBytes(2))))
		})
	})
})

type fakeCollector struct {
	collector.Collector

	vmis     []*k6tv1.VirtualMachineInstance
	vmiStats []*VirtualMachineInstanceStats
}

func (fc fakeCollector) Collect(_ []*k6tv1.VirtualMachineInstance, scraper collector.MetricsScraper, _ time.Duration) ([]string, bool) {
	var busyScrapers sync.WaitGroup

	for i, vmi := range fc.vmis {
		busyScrapers.Add(1)
		go fakeCollect(scraper, &busyScrapers, vmi, fc.vmiStats[i])
	}

	c := make(chan struct{})
	go func() {
		busyScrapers.Wait()
		c <- struct{}{}
	}()
	select {
	case <-c:
		scraper.Complete()
	case <-time.After(collector.CollectionTimeout):
		Fail("timeout")
	}

	return nil, true
}

func fakeCollect(
	scraper collector.MetricsScraper, wg *sync.WaitGroup,
	vmi *k6tv1.VirtualMachineInstance, vmiStats *VirtualMachineInstanceStats,
) {
	defer wg.Done()

	dScraper := scraper.(*DomainstatsScraper)
	report(vmi, vmiStats, dScraper.ch)
}
