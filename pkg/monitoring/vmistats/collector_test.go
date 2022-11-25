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

package vmistats

import (
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = BeforeSuite(func() {
})

var _ = Describe("VMI Stats Collector", func() {

	Context("should push metrics", func() {

		var (
			vmis    []*k6tv1.VirtualMachineInstance
			now     metav1.Time
			ch      chan prometheus.Metric
			scraper *vmiPrometheusScraper
		)

		BeforeEach(func() {
			now = metav1.Now()
			vmis = []*k6tv1.VirtualMachineInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "test-ns",
						Name:              "testvmi",
						CreationTimestamp: now,
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						NodeName: "testNode",
					},
				},
			}
			vmiInformer, _ := testutils.NewFakeInformerFor(&k6tv1.VirtualMachineInstance{})
			clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKV(&k6tv1.KubeVirt{})
			collector := &VMICollector{
				vmiInformer:   vmiInformer,
				clusterConfig: clusterConfig,
			}
			ch = make(chan prometheus.Metric, 2)
			scraper = VmiPrometheusScraper(
				ch, collector.clusterConfig, vmis)
		})

		liveMigrateEvictPolicy := k6tv1.EvictionStrategyLiveMigrate

		DescribeTable("regarding eviction alert when", func(evictionPolicy *k6tv1.EvictionStrategy, migrateCondStatus k8sv1.ConditionStatus, expectedVal float64) {
			defer close(ch)

			addEvictionStrategyAndConditions(vmis, evictionPolicy, migrateCondStatus)
			scraper.updateVMIMetrics()

			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_non_evictable"))
			Expect(dto.Gauge.GetValue()).To(BeEquivalentTo(expectedVal))
		},
			Entry("VMI Eviction policy set to LiveMigration and vm is not migratable", &liveMigrateEvictPolicy, k8sv1.ConditionFalse, 1.0),
			Entry("VMI Eviction policy set to LiveMigration and vm migratable status is not known", &liveMigrateEvictPolicy, k8sv1.ConditionUnknown, 1.0),
			Entry("VMI Eviction policy set to LiveMigration and vm is migratable", &liveMigrateEvictPolicy, k8sv1.ConditionTrue, 0.0),
			Entry("VMI Eviction policy is not set and vm is not migratable", nil, k8sv1.ConditionFalse, 0.0),
			Entry("VMI Eviction policy is not set and vm is migratable", nil, k8sv1.ConditionTrue, 0.0),
			Entry("VMI Eviction policy is not set and vm migratable status is not known", nil, k8sv1.ConditionUnknown, 0.0),
		)

		It("regarding creation timestamp metric", func() {
			defer close(ch)
			scraper.updateVMIMetrics()

			<-ch
			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring(vmiCreationTimestamp))
			Expect(dto.Counter.GetValue()).To(BeEquivalentTo(now.Unix()))
		})
	})
})

func addEvictionStrategyAndConditions(vmis []*k6tv1.VirtualMachineInstance, evictionStrategy *k6tv1.EvictionStrategy, migratableCondStatus k8sv1.ConditionStatus) {

	if migratableCondStatus != k8sv1.ConditionUnknown {
		vmis[0].Status.Conditions = []k6tv1.VirtualMachineInstanceCondition{
			{
				Type:   k6tv1.VirtualMachineInstanceIsMigratable,
				Status: migratableCondStatus,
			},
		}
	}

	vmis[0].Spec.EvictionStrategy = evictionStrategy
}

var _ = Describe("Utility functions", func() {
	Context("VMI Count map reporting", func() {
		It("should handle missing VMs", func() {
			var countMap map[vmiCountMetric]uint64

			countMap = makeVMICountMetricMap(nil)
			Expect(countMap).NotTo(BeNil())
			Expect(countMap).To(BeEmpty())

			vmis := []*k6tv1.VirtualMachineInstance{}
			countMap = makeVMICountMetricMap(vmis)
			Expect(countMap).NotTo(BeNil())
			Expect(countMap).To(BeEmpty())
		})

		It("should handle different VMI phases", func() {
			vmis := []*k6tv1.VirtualMachineInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "running#0",
						Annotations: map[string]string{
							annotationPrefix + "os":       "centos8",
							annotationPrefix + "workload": "server",
							annotationPrefix + "flavor":   "tiny",
						},
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Running",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "running#1",
						Annotations: map[string]string{
							annotationPrefix + "os":       "centos8",
							annotationPrefix + "workload": "server",
							annotationPrefix + "flavor":   "tiny",
						},
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Running",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pending#0",
						Annotations: map[string]string{
							annotationPrefix + "os":       "fedora33",
							annotationPrefix + "workload": "workstation",
							annotationPrefix + "flavor":   "large",
						},
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Pending",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "scheduling#0",
						Annotations: map[string]string{
							annotationPrefix + "os":       "centos7",
							annotationPrefix + "workload": "server",
							annotationPrefix + "flavor":   "medium",
							annotationPrefix + "dummy":    "dummy",
						},
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Scheduling",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "scheduling#1",
						Annotations: map[string]string{
							annotationPrefix + "os":       "centos7",
							annotationPrefix + "workload": "server",
							annotationPrefix + "flavor":   "medium",
							annotationPrefix + "phase":    "dummy",
						},
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Scheduling",
					},
				},
			}

			countMap := makeVMICountMetricMap(vmis)
			Expect(countMap).NotTo(BeNil())
			Expect(countMap).To(HaveLen(3))

			running := vmiCountMetric{
				Phase:    "running",
				OS:       "centos8",
				Workload: "server",
				Flavor:   "tiny",
			}
			pending := vmiCountMetric{
				Phase:    "pending",
				OS:       "fedora33",
				Workload: "workstation",
				Flavor:   "large",
			}
			scheduling := vmiCountMetric{
				Phase:    "scheduling",
				OS:       "centos7",
				Workload: "server",
				Flavor:   "medium",
			}
			bogus := vmiCountMetric{
				Phase: "bogus",
			}
			Expect(countMap[running]).To(Equal(uint64(2)))
			Expect(countMap[pending]).To(Equal(uint64(1)))
			Expect(countMap[scheduling]).To(Equal(uint64(2)))
			Expect(countMap[bogus]).To(Equal(uint64(0))) // intentionally bogus key
		})
	})
})
