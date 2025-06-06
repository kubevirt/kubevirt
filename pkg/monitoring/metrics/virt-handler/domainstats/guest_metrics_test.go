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
 *
 */

package domainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/testing"
)

var _ = Describe("guest metrics", func() {
	Context("Collect", func() {
		It("should collect guest hostname as info metric", func() {
			vmiReport := &VirtualMachineInstanceReport{
				vmi: &k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-vmi-1",
						Namespace: "test-ns-1",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						NodeName: "test-node-1",
					},
				},
				vmiStats: &VirtualMachineInstanceStats{
					GuestAgentInfo: &k6tv1.VirtualMachineInstanceGuestAgentInfo{
						Hostname: "test-hostname",
					},
				},
			}
			vmiReport.buildRuntimeLabels()

			metrics := guestMetrics{}
			crs := metrics.Collect(vmiReport)

			Expect(crs).To(HaveLen(1))
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(guestHostname, 1.0)))

			cr := crs[0]
			Expect(cr.ConstLabels).To(HaveKeyWithValue("hostname", "test-hostname"))
		})

		It("should not collect hostname metric when hostname is empty", func() {
			vmiReport := &VirtualMachineInstanceReport{
				vmi: &k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-vmi-1",
						Namespace: "test-ns-1",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						NodeName: "test-node-1",
					},
				},
				vmiStats: &VirtualMachineInstanceStats{
					GuestAgentInfo: &k6tv1.VirtualMachineInstanceGuestAgentInfo{
						Hostname: "",
					},
				},
			}
			vmiReport.buildRuntimeLabels()

			metrics := guestMetrics{}
			crs := metrics.Collect(vmiReport)

			Expect(crs).To(BeEmpty())
		})

		It("should collect guest load average metrics", func() {
			vmiReport := &VirtualMachineInstanceReport{
				vmi: &k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-vmi-1",
						Namespace: "test-ns-1",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						NodeName: "test-node-1",
					},
				},
				vmiStats: &VirtualMachineInstanceStats{
					GuestAgentInfo: &k6tv1.VirtualMachineInstanceGuestAgentInfo{
						Load: k6tv1.VirtualMachineInstanceGuestOSLoad{
							Load1m:  1.5,
							Load5m:  2.5,
							Load15m: 3.5,
						},
					},
				},
			}
			vmiReport.buildRuntimeLabels()

			metrics := guestMetrics{}
			crs := metrics.Collect(vmiReport)

			Expect(crs).To(HaveLen(3))
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(guestLoad1M, 1.5)))
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(guestLoad5M, 2.5)))
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(guestLoad15M, 3.5)))
		})

		It("should not collect metrics when guest agent info is nil", func() {
			vmiReport := &VirtualMachineInstanceReport{
				vmi: &k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-vmi-1",
						Namespace: "test-ns-1",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						NodeName: "test-node-1",
					},
				},
				vmiStats: &VirtualMachineInstanceStats{
					GuestAgentInfo: nil,
				},
			}
			vmiReport.buildRuntimeLabels()

			metrics := guestMetrics{}
			crs := metrics.Collect(vmiReport)

			Expect(crs).To(BeEmpty())
		})

		It("should only collect non-zero load metrics", func() {
			vmiReport := &VirtualMachineInstanceReport{
				vmi: &k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-vmi-1",
						Namespace: "test-ns-1",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						NodeName: "test-node-1",
					},
				},
				vmiStats: &VirtualMachineInstanceStats{
					GuestAgentInfo: &k6tv1.VirtualMachineInstanceGuestAgentInfo{
						Load: k6tv1.VirtualMachineInstanceGuestOSLoad{
							Load1m:  1.5, // Only 1M load is non-zero
							Load5m:  0,
							Load15m: 0,
						},
					},
				},
			}
			vmiReport.buildRuntimeLabels()

			metrics := guestMetrics{}
			crs := metrics.Collect(vmiReport)

			Expect(crs).To(HaveLen(1))
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(guestLoad1M, 1.5)))
		})
	})
})
