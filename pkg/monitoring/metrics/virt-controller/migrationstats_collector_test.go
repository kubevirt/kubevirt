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

package virtcontroller

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Migration Stats Collector", func() {
	getVMIM := func(phase k6tv1.VirtualMachineInstanceMigrationPhase) *k6tv1.VirtualMachineInstanceMigration {
		return &k6tv1.VirtualMachineInstanceMigration{
			Status: k6tv1.VirtualMachineInstanceMigrationStatus{
				Phase: phase,
			},
		}
	}

	It("should set all metrics to 0 when no migrations", func() {
		var vmims []*k6tv1.VirtualMachineInstanceMigration

		cr := reportMigrationStats(vmims)

		for _, result := range cr {
			Expect(result.Value).Should(BeZero())
		}
	})

	DescribeTable("should set correct metric for each phase",
		func(phase k6tv1.VirtualMachineInstanceMigrationPhase, metric operatormetrics.Metric) {
			vmims := []*k6tv1.VirtualMachineInstanceMigration{
				getVMIM(phase),
			}

			cr := reportMigrationStats(vmims)

			containsMetric := false

			for _, result := range cr {
				if isMigrationInfoMetric(result) {
					Expect(result.Value).To(Equal(1.0))
					continue
				}
				if strings.Contains(result.Metric.GetOpts().Name, metric.GetOpts().Name) {
					containsMetric = true
					Expect(result.Value).To(Equal(1.0))
				} else {
					Expect(result.Value).To(BeZero())
				}
			}

			Expect(containsMetric).To(BeTrue())
		},
		Entry("Failed migration", k6tv1.MigrationFailed, failedMigration),
		Entry("Pending migration", k6tv1.MigrationPending, pendingMigrations),
		Entry("Running migration", k6tv1.MigrationRunning, runningMigrations),
		Entry("Scheduling migration", k6tv1.MigrationScheduling, schedulingMigrations),
		Entry("Succeeded migration", k6tv1.MigrationSucceeded, succeededMigration),
		Entry("Undefined migration", k6tv1.MigrationPhaseUnset, unsetMigration),
	)

	It("should set succeeded and pending to 1 and others to 0 with 1 successful and 1 pending", func() {
		vmims := []*k6tv1.VirtualMachineInstanceMigration{
			getVMIM(k6tv1.MigrationSucceeded),
			getVMIM(k6tv1.MigrationPending),
		}

		cr := reportMigrationStats(vmims)

		for _, result := range cr {
			if isMigrationInfoMetric(result) {
				Expect(result.Value).To(Equal(1.0))
				continue
			}
			if strings.Contains(result.Metric.GetOpts().Name, "succeeded") {
				Expect(result.Value).To(Equal(1.0))
			} else if strings.Contains(result.Metric.GetOpts().Name, "pending") {
				Expect(result.Value).To(Equal(1.0))
			} else {
				Expect(result.Value).To(BeZero())
			}
		}
	})

	Context("kubevirt_vmi_migration_info", func() {
		It("should emit one series per VMIM with user trigger for an in-progress migration", func() {
			vmim := &k6tv1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmim",
					Namespace: "test-ns",
					UID:       types.UID("test-vmim-uid"),
				},
				Spec: k6tv1.VirtualMachineInstanceMigrationSpec{
					VMIName: "test-vmi",
				},
				Status: k6tv1.VirtualMachineInstanceMigrationStatus{
					Phase: k6tv1.MigrationRunning,
				},
			}

			results := migrationInfoResults(reportMigrationStats([]*k6tv1.VirtualMachineInstanceMigration{vmim}))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Value).To(Equal(1.0))
			Expect(results[0].Labels).To(Equal([]string{
				"test-ns", "test-vmi", "test-vmim", "test-vmim-uid",
				"", "",
				"running", migrationTriggerUser, migrationResultInProgress, migrationReasonNone,
				"", "", "", "",
			}))
		})

		DescribeTable("should map trigger from VMIM annotations", func(annotations map[string]string, trigger string) {
			vmim := &k6tv1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-vmim",
					Namespace:   "test-ns",
					Annotations: annotations,
				},
				Spec: k6tv1.VirtualMachineInstanceMigrationSpec{
					VMIName: "test-vmi",
				},
				Status: k6tv1.VirtualMachineInstanceMigrationStatus{
					Phase: k6tv1.MigrationPending,
				},
			}

			results := migrationInfoResults(reportMigrationStats([]*k6tv1.VirtualMachineInstanceMigration{vmim}))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Labels[7]).To(Equal(trigger))
		},
			Entry("user", nil, migrationTriggerUser),
			Entry("evacuation", map[string]string{k6tv1.EvacuationMigrationAnnotation: "node-1"}, migrationTriggerEvacuation),
			Entry("workload update", map[string]string{k6tv1.WorkloadUpdateMigrationAnnotation: ""}, migrationTriggerWorkloadUpdate),
		)

		It("should prefer evacuation over workload update when both annotations are set", func() {
			vmim := &k6tv1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmim",
					Namespace: "test-ns",
					Annotations: map[string]string{
						k6tv1.EvacuationMigrationAnnotation:     "node-1",
						k6tv1.WorkloadUpdateMigrationAnnotation: "",
					},
				},
				Spec: k6tv1.VirtualMachineInstanceMigrationSpec{
					VMIName: "test-vmi",
				},
				Status: k6tv1.VirtualMachineInstanceMigrationStatus{
					Phase: k6tv1.MigrationPending,
				},
			}

			results := migrationInfoResults(reportMigrationStats([]*k6tv1.VirtualMachineInstanceMigration{vmim}))
			Expect(results[0].Labels[7]).To(Equal(migrationTriggerEvacuation))
		})

		It("should include source and target nodes and succeeded result", func() {
			vmim := &k6tv1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmim",
					Namespace: "test-ns",
					UID:       types.UID("test-vmim-uid"),
				},
				Spec: k6tv1.VirtualMachineInstanceMigrationSpec{
					VMIName: "test-vmi",
				},
				Status: k6tv1.VirtualMachineInstanceMigrationStatus{
					Phase: k6tv1.MigrationSucceeded,
					MigrationState: &k6tv1.VirtualMachineInstanceMigrationState{
						SourceNode: "node-a",
						TargetNode: "node-b",
					},
				},
			}

			results := migrationInfoResults(reportMigrationStats([]*k6tv1.VirtualMachineInstanceMigration{vmim}))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Labels).To(Equal([]string{
				"test-ns", "test-vmi", "test-vmim", "test-vmim-uid",
				"node-a", "node-b",
				"succeeded", migrationTriggerUser, migrationResultSucceeded, migrationReasonNone,
				"", "", "", "",
			}))
		})

		DescribeTable("should map failed reason from status", func(vmim *k6tv1.VirtualMachineInstanceMigration, reason string) {
			results := migrationInfoResults(reportMigrationStats([]*k6tv1.VirtualMachineInstanceMigration{vmim}))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Labels[6]).To(Equal("failed"))
			Expect(results[0].Labels[8]).To(Equal(migrationResultFailed))
			Expect(results[0].Labels[9]).To(Equal(reason))
		},
			Entry("generic failure",
				failedMigrationInfoVMIM(&k6tv1.VirtualMachineInstanceMigrationState{
					FailureReason: "live migration failed",
				}, nil),
				migrationReasonFailed,
			),
			Entry("timeout",
				failedMigrationInfoVMIM(&k6tv1.VirtualMachineInstanceMigrationState{
					FailureReason: "pending pod default/virt-launcher-x timeout period exceeded",
				}, nil),
				migrationReasonTimeout,
			),
			Entry("unschedulable timeout prefers unschedulable",
				failedMigrationInfoVMIM(&k6tv1.VirtualMachineInstanceMigrationState{
					FailureReason: "unschedulable pod default/virt-launcher-x timeout period exceeded",
				}, nil),
				migrationReasonUnschedulable,
			),
			Entry("resource quota",
				failedMigrationInfoVMIM(nil, []k6tv1.VirtualMachineInstanceMigrationCondition{
					{
						Type:   k6tv1.VirtualMachineInstanceMigrationRejectedByResourceQuota,
						Status: k8sv1.ConditionTrue,
					},
				}),
				migrationReasonUnschedulable,
			),
			Entry("abort requested condition",
				failedMigrationInfoVMIM(nil, []k6tv1.VirtualMachineInstanceMigrationCondition{
					{
						Type:   k6tv1.VirtualMachineInstanceMigrationAbortRequested,
						Status: k8sv1.ConditionTrue,
					},
				}),
				migrationReasonCanceled,
			),
			Entry("abort status",
				failedMigrationInfoVMIM(&k6tv1.VirtualMachineInstanceMigrationState{
					AbortStatus:   k6tv1.MigrationAbortSucceeded,
					FailureReason: "live migration has been aborted",
				}, nil),
				migrationReasonCanceled,
			),
		)

		It("should not classify in-progress abort as a failed reason", func() {
			vmim := &k6tv1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmim",
					Namespace: "test-ns",
				},
				Spec: k6tv1.VirtualMachineInstanceMigrationSpec{
					VMIName: "test-vmi",
				},
				Status: k6tv1.VirtualMachineInstanceMigrationStatus{
					Phase: k6tv1.MigrationRunning,
					Conditions: []k6tv1.VirtualMachineInstanceMigrationCondition{
						{
							Type:   k6tv1.VirtualMachineInstanceMigrationAbortRequested,
							Status: k8sv1.ConditionTrue,
						},
					},
				},
			}

			results := migrationInfoResults(reportMigrationStats([]*k6tv1.VirtualMachineInstanceMigration{vmim}))
			Expect(results[0].Labels[8]).To(Equal(migrationResultInProgress))
			Expect(results[0].Labels[9]).To(Equal(migrationReasonNone))
		})

		It("should distinguish VMIMs that reuse a name by uid", func() {
			first := failedMigrationInfoVMIM(nil, nil)
			first.UID = "uid-1"
			second := failedMigrationInfoVMIM(nil, nil)
			second.UID = "uid-2"

			results := migrationInfoResults(reportMigrationStats([]*k6tv1.VirtualMachineInstanceMigration{first, second}))
			Expect(results).To(HaveLen(2))
			Expect(results[0].Labels[3]).To(Equal("uid-1"))
			Expect(results[1].Labels[3]).To(Equal("uid-2"))
		})

		DescribeTable("should map mode, priority, policy, and network type",
			func(mutate func(*k6tv1.VirtualMachineInstanceMigration), mode, priority, policy, networkType string) {
				vmim := &k6tv1.VirtualMachineInstanceMigration{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-vmim",
						Namespace: "test-ns",
						UID:       types.UID("test-vmim-uid"),
					},
					Spec: k6tv1.VirtualMachineInstanceMigrationSpec{
						VMIName: "test-vmi",
					},
					Status: k6tv1.VirtualMachineInstanceMigrationStatus{
						Phase: k6tv1.MigrationRunning,
					},
				}
				mutate(vmim)

				results := migrationInfoResults(reportMigrationStats([]*k6tv1.VirtualMachineInstanceMigration{vmim}))
				Expect(results).To(HaveLen(1))
				Expect(results[0].Labels[10]).To(Equal(mode))
				Expect(results[0].Labels[11]).To(Equal(priority))
				Expect(results[0].Labels[12]).To(Equal(policy))
				Expect(results[0].Labels[13]).To(Equal(networkType))
			},
			Entry("unset", func(*k6tv1.VirtualMachineInstanceMigration) {}, "", "", "", ""),
			Entry("precopy over pod", func(vmim *k6tv1.VirtualMachineInstanceMigration) {
				vmim.Spec.Priority = pointer.P(k6tv1.PriorityUserTriggered)
				vmim.Status.MigrationState = &k6tv1.VirtualMachineInstanceMigrationState{
					Mode:                 k6tv1.MigrationPreCopy,
					MigrationPolicyName:  pointer.P("policy-a"),
					MigrationNetworkType: k6tv1.Pod,
				}
			}, "precopy", "user-triggered", "policy-a", "pod"),
			Entry("postcopy over migration network", func(vmim *k6tv1.VirtualMachineInstanceMigration) {
				vmim.Spec.Priority = pointer.P(k6tv1.PrioritySystemCritical)
				vmim.Status.MigrationState = &k6tv1.VirtualMachineInstanceMigrationState{
					Mode:                 k6tv1.MigrationPostCopy,
					MigrationNetworkType: k6tv1.Migration,
				}
			}, "postcopy", "system-critical", "", "migration"),
			Entry("paused system-maintenance", func(vmim *k6tv1.VirtualMachineInstanceMigration) {
				vmim.Spec.Priority = pointer.P(k6tv1.PrioritySystemMaintenance)
				vmim.Status.MigrationState = &k6tv1.VirtualMachineInstanceMigrationState{
					Mode: k6tv1.MigrationPaused,
				}
			}, "paused", "system-maintenance", "", ""),
			Entry("empty policy name", func(vmim *k6tv1.VirtualMachineInstanceMigration) {
				vmim.Status.MigrationState = &k6tv1.VirtualMachineInstanceMigrationState{
					MigrationPolicyName: pointer.P(""),
				}
			}, "", "", "", ""),
		)
	})
})

func isMigrationInfoMetric(result operatormetrics.CollectorResult) bool {
	return result.Metric.GetOpts().Name == migrationInfo.GetOpts().Name
}

func migrationInfoResults(cr []operatormetrics.CollectorResult) []operatormetrics.CollectorResult {
	var results []operatormetrics.CollectorResult
	for _, result := range cr {
		if isMigrationInfoMetric(result) {
			results = append(results, result)
		}
	}
	return results
}

func failedMigrationInfoVMIM(
	state *k6tv1.VirtualMachineInstanceMigrationState,
	conditions []k6tv1.VirtualMachineInstanceMigrationCondition,
) *k6tv1.VirtualMachineInstanceMigration {
	return &k6tv1.VirtualMachineInstanceMigration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vmim",
			Namespace: "test-ns",
			UID:       types.UID("test-vmim-uid"),
		},
		Spec: k6tv1.VirtualMachineInstanceMigrationSpec{
			VMIName: "test-vmi",
		},
		Status: k6tv1.VirtualMachineInstanceMigrationStatus{
			Phase:          k6tv1.MigrationFailed,
			MigrationState: state,
			Conditions:     conditions,
		},
	}
}
