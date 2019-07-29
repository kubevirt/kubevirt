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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/prometheus/client_golang/prometheus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k6tv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = BeforeSuite(func() {
	log.Log.SetIOWriter(GinkgoWriter)
})

var _ = Describe("Prometheus", func() {
	Context("on blocked source", func() {
		It("should handle closed reporting socket", func() {
			ch := make(chan prometheus.Metric)
			close(ch)

			ps := prometheusScraper{ch: ch}

			testReportPanic := func() {
				vmStats := &stats.DomainStats{
					Cpu: &stats.DomainStatsCPU{},
					Memory: &stats.DomainStatsMemory{
						// trigger write on a socket. We need a value set - any value
						RSS:    1024,
						RSSSet: true,
					},
				}
				vmi := k6tv1.VirtualMachineInstance{}
				ps.Report("test", &vmi, vmStats)
			}
			Expect(testReportPanic).ToNot(Panic())
		})
	})
})

var _ = Describe("Utility functions", func() {
	Context("VMI Phases map reporting", func() {
		It("should handle missing VMs", func() {
			var phasesMap map[string]uint64

			phasesMap = makeVMIsPhasesMap(nil)
			Expect(phasesMap).NotTo(BeNil())
			Expect(len(phasesMap)).To(Equal(0))

			vmis := []*k6tv1.VirtualMachineInstance{}
			phasesMap = makeVMIsPhasesMap(vmis)
			Expect(phasesMap).NotTo(BeNil())
			Expect(len(phasesMap)).To(Equal(0))
		})

		It("should handle different VMI phases", func() {
			vmis := []*k6tv1.VirtualMachineInstance{
				&k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "running#0",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Running",
					},
				},
				&k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pending#0",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Pending",
					},
				},
				&k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "scheduling#0",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Scheduling",
					},
				},
				&k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "running#1",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Running",
					},
				},
			}

			phasesMap := makeVMIsPhasesMap(vmis)
			Expect(phasesMap).NotTo(BeNil())
			Expect(len(phasesMap)).To(Equal(3))
			Expect(phasesMap["running"]).To(Equal(uint64(2)))
			Expect(phasesMap["pending"]).To(Equal(uint64(1)))
			Expect(phasesMap["scheduling"]).To(Equal(uint64(1)))
			Expect(phasesMap["bogus"]).To(Equal(uint64(0))) // intentionally bogus key
		})
	})
})
