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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package virt_controller

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("VM Stats Collector", func() {
	Context("VM status collector", func() {
		createVM := func(status k6tv1.VirtualMachinePrintableStatus, vmLastTransitionsTime time.Time) *k6tv1.VirtualMachine {
			return &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-vm"},
				Status: k6tv1.VirtualMachineStatus{
					PrintableStatus: status,
					Conditions: []k6tv1.VirtualMachineCondition{
						{
							Type:               k6tv1.VirtualMachineFailure,
							Status:             "any",
							Reason:             "any",
							LastTransitionTime: metav1.NewTime(vmLastTransitionsTime.Add(-20 * time.Second)),
						},
						{
							Type:               k6tv1.VirtualMachineReady,
							Status:             "any",
							Reason:             "any",
							LastTransitionTime: metav1.NewTime(vmLastTransitionsTime),
						},
						{
							Type:               k6tv1.VirtualMachinePaused,
							Status:             "any",
							Reason:             "any",
							LastTransitionTime: metav1.NewTime(vmLastTransitionsTime.Add(-40 * time.Second)),
						},
					},
				},
			}
		}

		DescribeTable("Add VM status metrics", func(status k6tv1.VirtualMachinePrintableStatus, metric operatormetrics.Metric) {
			t := time.Now()
			vms := []*k6tv1.VirtualMachine{
				createVM(status, t),
			}

			cr := reportVmsStats(vms)

			containsStateMetric := false

			for _, result := range cr {
				if strings.Contains(result.Metric.GetOpts().Name, metric.GetOpts().Name) {
					containsStateMetric = true
					Expect(result.Value).To(Equal(float64(t.Unix())))
				} else {
					Expect(result.Value).To(BeZero())
				}
			}

			Expect(containsStateMetric).To(BeTrue())
		},
			Entry("Starting VM", k6tv1.VirtualMachineStatusProvisioning, startingTimestamp),
			Entry("Running VM", k6tv1.VirtualMachineStatusRunning, runningTimestamp),
			Entry("Migrating VM", k6tv1.VirtualMachineStatusMigrating, migratingTimestamp),
			Entry("Non running VM", k6tv1.VirtualMachineStatusStopped, nonRunningTimestamp),
			Entry("Errored VM", k6tv1.VirtualMachineStatusCrashLoopBackOff, errorTimestamp),
		)
	})

	Context("VM Resource Requests", func() {
		It("should ignore VM with empty memory resource requests", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "testvm",
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Domain: k6tv1.DomainSpec{},
						},
					},
				},
			}

			cr := CollectResourceRequests([]*k6tv1.VirtualMachine{vm})
			Expect(cr).To(BeZero())
		})

		It("should collect VM memory resource requests", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "testvm",
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Domain: k6tv1.DomainSpec{
								Resources: k6tv1.ResourceRequirements{
									Requests: k8sv1.ResourceList{
										k8sv1.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
									},
								},
							},
						},
					},
				},
			}

			crs := CollectResourceRequests([]*k6tv1.VirtualMachine{vm})
			Expect(crs).To(HaveLen(1))
			Expect(crs[0].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(crs[0].Value).To(BeEquivalentTo(1024))
			Expect(crs[0].Labels).To(Equal([]string{"testvm", "test-ns", "memory", "bytes"}))
		})

		It("should collect VM CPU resource requests", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "testvm",
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Domain: k6tv1.DomainSpec{
								Resources: k6tv1.ResourceRequirements{
									Requests: k8sv1.ResourceList{
										k8sv1.ResourceCPU: *resource.NewMilliQuantity(500, resource.BinarySI),
									},
								},
							},
						},
					},
				},
			}

			crs := CollectResourceRequests([]*k6tv1.VirtualMachine{vm})
			Expect(crs).To(HaveLen(1), "Expected 1 metric")
			Expect(crs[0].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(crs[0].Value).To(BeEquivalentTo(0.5))
			Expect(crs[0].Labels).To(Equal([]string{"testvm", "test-ns", "cpu", "cores"}))
		})

		It("should collect VM CPU resource requests from domain", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "testvm",
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Domain: k6tv1.DomainSpec{
								CPU: &k6tv1.CPU{
									Cores:   2,
									Threads: 4,
									Sockets: 1,
								},
							},
						},
					},
				},
			}

			crs := CollectResourceRequests([]*k6tv1.VirtualMachine{vm})
			Expect(crs).To(HaveLen(3), "Expected 1 metric")

			Expect(crs[0].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(crs[0].Value).To(BeEquivalentTo(2))
			Expect(crs[0].Labels).To(Equal([]string{"testvm", "test-ns", "cpu", "cores"}))

			Expect(crs[1].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(crs[1].Value).To(BeEquivalentTo(4))
			Expect(crs[1].Labels).To(Equal([]string{"testvm", "test-ns", "cpu", "threads"}))

			Expect(crs[2].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(crs[2].Value).To(BeEquivalentTo(1))
			Expect(crs[2].Labels).To(Equal([]string{"testvm", "test-ns", "cpu", "sockets"}))
		})
	})
})
