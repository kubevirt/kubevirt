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

package virt_controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = BeforeSuite(func() {
})

var _ = Describe("VMI Stats Collector", func() {

	Context("VMI Eviction blocker", func() {

		liveMigrateEvictPolicy := k6tv1.EvictionStrategyLiveMigrate
		DescribeTable("Add eviction alert metrics", func(evictionPolicy *k6tv1.EvictionStrategy, migrateCondStatus k8sv1.ConditionStatus, expectedVal float64) {
			vmiInformer, _ = testutils.NewFakeInformerFor(&k6tv1.VirtualMachineInstance{})
			clusterConfig, _, _ = testutils.NewFakeClusterConfigUsingKV(&k6tv1.KubeVirt{})

			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			vmis := createVMISForEviction(evictionPolicy, migrateCondStatus)

			evictionBlockerResults := getEvictionBlocker(vmis)
			Expect(evictionBlockerResults).To(HaveLen(1), "Expected 1 metric")

			evictionBlockerResultMetric := evictionBlockerResults[0]
			Expect(evictionBlockerResultMetric).ToNot(BeNil())
			Expect(evictionBlockerResultMetric.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vmi_non_evictable"))
			Expect(evictionBlockerResultMetric.Value).To(BeEquivalentTo(expectedVal))
		},
			Entry("VMI Eviction policy set to LiveMigration and vm is not migratable", &liveMigrateEvictPolicy, k8sv1.ConditionFalse, 1.0),
			Entry("VMI Eviction policy set to LiveMigration and vm migratable status is not known", &liveMigrateEvictPolicy, k8sv1.ConditionUnknown, 1.0),
			Entry("VMI Eviction policy set to LiveMigration and vm is migratable", &liveMigrateEvictPolicy, k8sv1.ConditionTrue, 0.0),
			Entry("VMI Eviction policy is not set and vm is not migratable", nil, k8sv1.ConditionFalse, 0.0),
			Entry("VMI Eviction policy is not set and vm is migratable", nil, k8sv1.ConditionTrue, 0.0),
			Entry("VMI Eviction policy is not set and vm migratable status is not known", nil, k8sv1.ConditionUnknown, 0.0),
		)
	})
})

func createVMISForEviction(evictionStrategy *k6tv1.EvictionStrategy, migratableCondStatus k8sv1.ConditionStatus) []*k6tv1.VirtualMachineInstance {

	vmis := []*k6tv1.VirtualMachineInstance{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test-ns",
				Name:      "testvmi",
			},
			Status: k6tv1.VirtualMachineInstanceStatus{
				NodeName: "testNode",
			},
		},
	}

	if migratableCondStatus != k8sv1.ConditionUnknown {
		vmis[0].Status.Conditions = []k6tv1.VirtualMachineInstanceCondition{
			{
				Type:   k6tv1.VirtualMachineInstanceIsMigratable,
				Status: migratableCondStatus,
			},
		}
	}

	vmis[0].Spec.EvictionStrategy = evictionStrategy

	return vmis
}

var _ = Describe("Utility functions", func() {
	setupTestCollector()

	Context("VMI Count map reporting", func() {
		It("should handle missing VMs", func() {
			var countMap map[vmiCountMetric]uint64

			countMap = makeVMICountMetricMap(nil)
			Expect(countMap).NotTo(BeNil())
			Expect(countMap).To(BeEmpty())

			var vmis []*k6tv1.VirtualMachineInstance
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
						GuestOSInfo: k6tv1.VirtualMachineInstanceGuestOSInfo{
							KernelRelease: "6.5.6-300.fc39.x86_64",
							Machine:       "x86_64",
							Name:          "Fedora Linux",
							VersionID:     "39",
						},
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
				Phase:                "running",
				OS:                   "centos8",
				Workload:             "server",
				Flavor:               "tiny",
				GuestOSKernelRelease: "<none>",
				GuestOSMachine:       "<none>",
				GuestOSName:          "<none>",
				GuestOSVersionID:     "<none>",
				InstanceType:         "<none>",
				Preference:           "<none>",
			}
			pending := vmiCountMetric{
				Phase:                "pending",
				OS:                   "fedora33",
				Workload:             "workstation",
				Flavor:               "large",
				InstanceType:         "<none>",
				Preference:           "<none>",
				GuestOSKernelRelease: "6.5.6-300.fc39.x86_64",
				GuestOSMachine:       "x86_64",
				GuestOSName:          "Fedora Linux",
				GuestOSVersionID:     "39",
			}
			scheduling := vmiCountMetric{
				Phase:                "scheduling",
				OS:                   "centos7",
				Workload:             "server",
				Flavor:               "medium",
				GuestOSKernelRelease: "<none>",
				GuestOSMachine:       "<none>",
				GuestOSName:          "<none>",
				GuestOSVersionID:     "<none>",
				InstanceType:         "<none>",
				Preference:           "<none>",
			}
			bogus := vmiCountMetric{
				Phase: "bogus",
			}
			Expect(countMap[running]).To(Equal(uint64(2)))
			Expect(countMap[pending]).To(Equal(uint64(1)))
			Expect(countMap[scheduling]).To(Equal(uint64(2)))
			Expect(countMap[bogus]).To(Equal(uint64(0))) // intentionally bogus key
		})

		DescribeTable("should show instance type value correctly", func(instanceTypeAnnotationKey string, instanceType string, expected string) {
			annotations := map[string]string{}
			if instanceType != "" {
				annotations[instanceTypeAnnotationKey] = instanceType
			}

			vmis := []*k6tv1.VirtualMachineInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "running",
						Namespace:   "test-ns",
						Annotations: annotations,
					},
				},
			}

			phaseResults := getVmisPhase(vmis)
			Expect(phaseResults).To(HaveLen(1), "Expected 1 metric")

			phaseResultMetric := phaseResults[0]
			Expect(phaseResultMetric).ToNot(BeNil())
			Expect(phaseResultMetric.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vmi_phase_count"))
			Expect(phaseResultMetric.Value).To(BeEquivalentTo(1))
			Expect(phaseResultMetric.Labels).To(HaveLen(11))
			Expect(phaseResultMetric.Labels[5]).To(Equal(expected))
		},
			Entry("with no instance type expect <none>", k6tv1.InstancetypeAnnotation, "", "<none>"),
			Entry("with managed instance type expect its name", k6tv1.InstancetypeAnnotation, "i-managed", "i-managed"),
			Entry("with custom instance type expect <other>", k6tv1.InstancetypeAnnotation, "i-unmanaged", "<other>"),
			Entry("with no cluster instance type expect <none>", k6tv1.ClusterInstancetypeAnnotation, "", "<none>"),
			Entry("with managed cluster instance type expect its name", k6tv1.ClusterInstancetypeAnnotation, "ci-managed", "ci-managed"),
			Entry("with custom cluster instance type expect <other>", k6tv1.ClusterInstancetypeAnnotation, "ci-unmanaged", "<other>"),
		)

		DescribeTable("should show preference value correctly", func(preferenceAnnotationKey string, preference string, expected string) {
			annotations := map[string]string{}
			if preference != "" {
				annotations[preferenceAnnotationKey] = preference
			}

			vmis := []*k6tv1.VirtualMachineInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "running",
						Namespace:   "test-ns",
						Annotations: annotations,
					},
				},
			}

			phaseResults := getVmisPhase(vmis)
			Expect(phaseResults).To(HaveLen(1), "Expected 1 metric")

			phaseResultMetric := phaseResults[0]

			Expect(phaseResultMetric.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vmi_phase_count"))
			Expect(phaseResultMetric.Value).To(BeEquivalentTo(1))
			Expect(phaseResultMetric.Labels).To(HaveLen(11))
			Expect(phaseResultMetric.Labels[6]).To(Equal(expected))
		},
			Entry("with no preference expect <none>", k6tv1.PreferenceAnnotation, "", "<none>"),
			Entry("with managed preference expect its name", k6tv1.PreferenceAnnotation, "p-managed", "p-managed"),
			Entry("with custom preference expect <other>", k6tv1.PreferenceAnnotation, "p-unmanaged", "<other>"),
			Entry("with no cluster preference expect <none>", k6tv1.ClusterPreferenceAnnotation, "", "<none>"),
			Entry("with managed cluster preference expect its name", k6tv1.ClusterPreferenceAnnotation, "cp-managed", "cp-managed"),
			Entry("with custom cluster preference expect <other>", k6tv1.ClusterPreferenceAnnotation, "cp-unmanaged", "<other>"),
		)
	})
})

func setupTestCollector() {
	instanceTypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineInstancetype{})
	clusterInstanceTypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterInstancetype{})
	preferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachinePreference{})
	clusterPreferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterPreference{})

	_ = instanceTypeInformer.GetStore().Add(&instancetypev1beta1.VirtualMachineInstancetype{
		ObjectMeta: newObjectMetaForInstancetypes("i-managed", "test-ns", "kubevirt.io"),
	})

	_ = instanceTypeInformer.GetStore().Add(&instancetypev1beta1.VirtualMachineInstancetype{
		ObjectMeta: newObjectMetaForInstancetypes("i-unmanaged", "test-ns", "some-user"),
	})

	_ = clusterInstanceTypeInformer.GetStore().Add(&instancetypev1beta1.VirtualMachineClusterInstancetype{
		ObjectMeta: newObjectMetaForInstancetypes("ci-managed", "", "kubevirt.io"),
		Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
			CPU: instancetypev1beta1.CPUInstancetype{
				Guest: 2,
			},
			Memory: instancetypev1beta1.MemoryInstancetype{
				Guest: *resource.NewQuantity(2048, resource.BinarySI),
			},
		},
	})

	_ = clusterInstanceTypeInformer.GetStore().Add(&instancetypev1beta1.VirtualMachineClusterInstancetype{
		ObjectMeta: newObjectMetaForInstancetypes("ci-unmanaged", "", ""),
	})

	_ = preferenceInformer.GetStore().Add(&instancetypev1beta1.VirtualMachinePreference{
		ObjectMeta: newObjectMetaForInstancetypes("p-managed", "test-ns", "kubevirt.io"),
	})

	_ = preferenceInformer.GetStore().Add(&instancetypev1beta1.VirtualMachinePreference{
		ObjectMeta: newObjectMetaForInstancetypes("p-unmanaged", "test-ns", "some-vendor.com"),
	})

	_ = clusterPreferenceInformer.GetStore().Add(&instancetypev1beta1.VirtualMachineClusterPreference{
		ObjectMeta: newObjectMetaForInstancetypes("cp-managed", "", "kubevirt.io"),
	})

	instancetypeMethods = &instancetype.InstancetypeMethods{
		InstancetypeStore:        instanceTypeInformer.GetStore(),
		ClusterInstancetypeStore: clusterInstanceTypeInformer.GetStore(),
		PreferenceStore:          preferenceInformer.GetStore(),
		ClusterPreferenceStore:   clusterPreferenceInformer.GetStore(),
	}

	vmiMigrationInformer, _ = testutils.NewFakeInformerFor(&k6tv1.VirtualMachineInstanceMigration{})

	_ = vmiMigrationInformer.GetStore().Add(&k6tv1.VirtualMachineInstanceMigration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-migration",
			Namespace: "test-ns",
			UID:       "test-migration-uid",
		},
	})
}

func newObjectMetaForInstancetypes(name, namespace, vendor string) metav1.ObjectMeta {
	om := metav1.ObjectMeta{
		Name:   name,
		Labels: map[string]string{instancetypeVendorLabel: vendor},
	}

	if namespace != "" {
		om.Namespace = namespace
	}

	return om
}
