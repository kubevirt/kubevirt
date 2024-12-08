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

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/prometheus/client_golang/prometheus"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = BeforeSuite(func() {
})

var _ = Describe("VMI Stats Collector", func() {
	clusterConfig, _, _ = testutils.NewFakeClusterConfigUsingKV(&k6tv1.KubeVirt{})

	Context("VMI info", func() {
		setupTestCollector()

		It("should handle no VMIs", func() {
			cr := reportVmisStats([]*k6tv1.VirtualMachineInstance{})
			Expect(cr).To(BeEmpty())
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

			var crs []operatormetrics.CollectorResult
			for _, vmi := range vmis {
				crs = append(crs, collectVMIInfo(vmi))
			}

			Expect(crs).To(HaveLen(5))

			for i, cr := range crs {
				Expect(cr).ToNot(BeNil())
				Expect(cr.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vmi_info"))
				Expect(cr.Value).To(BeEquivalentTo(1))
				Expect(cr.Labels).To(HaveLen(17))

				Expect(cr.Labels[3]).To(Equal(getVMIPhase(vmis[i])))
				os, workload, flavor := getSystemInfoFromAnnotations(vmis[i].Annotations)
				Expect(cr.Labels[4]).To(Equal(os))
				Expect(cr.Labels[5]).To(Equal(workload))
				Expect(cr.Labels[6]).To(Equal(flavor))
				Expect(cr.Labels[16]).To(Equal(getVMIPod(vmis[i])))
			}
		})

		It("should update the vmi_pod label correctly after migration", func() {
			originalPod := &k8sv1.Pod{
				ObjectMeta: newPodMetaForInformer("virt-launcher-originalpod", "test-ns", "test-vmi-uid"),
				Spec: k8sv1.PodSpec{
					NodeName: "initial-node",
				},
				Status: k8sv1.PodStatus{
					Phase: k8sv1.PodRunning,
				},
			}
			targetPod := &k8sv1.Pod{
				ObjectMeta: newPodMetaForInformer("virt-launcher-targetpod", "test-ns", "test-vmi-uid"),
				Spec: k8sv1.PodSpec{
					NodeName: "target-node",
				},
				Status: k8sv1.PodStatus{
					Phase: k8sv1.PodRunning,
				},
			}

			_ = kvPodInformer.GetStore().Add(originalPod)
			_ = kvPodInformer.GetStore().Add(targetPod)

			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "test-ns",
					UID:       "test-vmi-uid",
				},
				Status: k6tv1.VirtualMachineInstanceStatus{
					Phase:    "Running",
					NodeName: "target-node",
					MigrationState: &k6tv1.VirtualMachineInstanceMigrationState{
						TargetPod: "virt-launcher-targetpod",
						Completed: true,
						Failed:    false,
					},
				},
			}

			cr := collectVMIInfo(vmi)

			Expect(cr).ToNot(BeNil())
			Expect(cr.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vmi_info"))
			Expect(cr.Value).To(BeEquivalentTo(1))
			Expect(cr.Labels).To(HaveLen(17))
			Expect(cr.Labels[16]).To(Equal("virt-launcher-targetpod"))
		})

		It("should return the original pod when migration failed", func() {
			originalPod := &k8sv1.Pod{
				ObjectMeta: newPodMetaForInformer("virt-launcher-originalpod", "test-ns", "test-vmi-uid"),
				Spec: k8sv1.PodSpec{
					NodeName: "initial-node",
				},
				Status: k8sv1.PodStatus{
					Phase: k8sv1.PodRunning,
				},
			}
			targetPod := &k8sv1.Pod{
				ObjectMeta: newPodMetaForInformer("virt-launcher-targetpod", "test-ns", "test-vmi-uid"),
				Spec: k8sv1.PodSpec{
					NodeName: "target-node",
				},
				Status: k8sv1.PodStatus{
					Phase: k8sv1.PodRunning,
				},
			}

			_ = kvPodInformer.GetStore().Add(originalPod)
			_ = kvPodInformer.GetStore().Add(targetPod)

			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "test-ns",
					UID:       "test-vmi-uid",
				},
				Status: k6tv1.VirtualMachineInstanceStatus{
					Phase:    "Running",
					NodeName: "initial-node",
					MigrationState: &k6tv1.VirtualMachineInstanceMigrationState{
						TargetPod: "virt-launcher-targetpod",
						Completed: true,
						Failed:    true,
					},
				},
			}

			cr := collectVMIInfo(vmi)

			Expect(cr).ToNot(BeNil())
			Expect(cr.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vmi_info"))
			Expect(cr.Value).To(BeEquivalentTo(1))
			Expect(cr.Labels).To(HaveLen(17))
			Expect(cr.Labels[16]).To(Equal("virt-launcher-originalpod"))
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
						Annotations: annotations,
					},
				},
			}

			var crs []operatormetrics.CollectorResult
			for _, vmi := range vmis {
				crs = append(crs, collectVMIInfo(vmi))
			}
			Expect(crs).To(HaveLen(1), "Expected 1 metric")

			cr := crs[0]
			Expect(cr).ToNot(BeNil())
			Expect(cr.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vmi_info"))
			Expect(cr.Value).To(BeEquivalentTo(1))
			Expect(cr.Labels).To(HaveLen(17))
			Expect(cr.Labels[7]).To(Equal(expected))
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
						Annotations: annotations,
					},
				},
			}

			var crs []operatormetrics.CollectorResult
			for _, vmi := range vmis {
				crs = append(crs, collectVMIInfo(vmi))
			}
			Expect(crs).To(HaveLen(1), "Expected 1 metric")

			cr := crs[0]

			Expect(cr.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vmi_info"))
			Expect(cr.Value).To(BeEquivalentTo(1))
			Expect(cr.Labels).To(HaveLen(17))
			Expect(cr.Labels[8]).To(Equal(expected))
		},
			Entry("with no preference expect <none>", k6tv1.PreferenceAnnotation, "", "<none>"),
			Entry("with managed preference expect its name", k6tv1.PreferenceAnnotation, "p-managed", "p-managed"),
			Entry("with custom preference expect <other>", k6tv1.PreferenceAnnotation, "p-unmanaged", "<other>"),
			Entry("with no cluster preference expect <none>", k6tv1.ClusterPreferenceAnnotation, "", "<none>"),
			Entry("with managed cluster preference expect its name", k6tv1.ClusterPreferenceAnnotation, "cp-managed", "cp-managed"),
			Entry("with custom cluster preference expect <other>", k6tv1.ClusterPreferenceAnnotation, "cp-unmanaged", "<other>"),
		)
	})

	Context("VMI Eviction blocker", func() {

		liveMigrateEvictPolicy := k6tv1.EvictionStrategyLiveMigrate
		DescribeTable("Add eviction alert metrics", func(evictionPolicy *k6tv1.EvictionStrategy, migrateCondStatus k8sv1.ConditionStatus, expectedVal float64) {
			vmiInformer, _ = testutils.NewFakeInformerFor(&k6tv1.VirtualMachineInstance{})

			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			vmi := createVMIForEviction(evictionPolicy, migrateCondStatus)

			evictionBlockerResultMetric := getEvictionBlocker(vmi)
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

	Context("VMI Interfaces info", func() {
		DescribeTable("kubevirt_vmi_status_addresses metrics", func(ifaceValues [][]string) {
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "testvmi",
				},
				Status: k6tv1.VirtualMachineInstanceStatus{
					NodeName:   "testNode",
					Interfaces: interfacesFor(ifaceValues),
				},
			}

			metrics := collectVMIInterfacesInfo(vmi)
			Expect(metrics).To(HaveLen(len(ifaceValues)))

			for i, labelValues := range ifaceValues {
				values := append([]string{"testNode", "test-ns", "testvmi"}, labelValues...)
				Expect(metrics[i].Labels).To(Equal(values))
			}
		},
			Entry("no interfaces", [][]string{}),
			Entry("one interface", [][]string{{"192.168.1.2", "InternalIP"}}),
			Entry("two interfaces", [][]string{
				{"170.170.170.170", "InternalIP"},
				{"180.180.180.180", "InternalIP"},
			}),
		)
	})
})

func interfacesFor(values [][]string) []k6tv1.VirtualMachineInstanceNetworkInterface {
	interfaces := make([]k6tv1.VirtualMachineInstanceNetworkInterface, len(values))
	for i, v := range values {
		interfaces[i] = k6tv1.VirtualMachineInstanceNetworkInterface{
			IP: v[0],
		}
	}
	return interfaces
}

func createVMIForEviction(evictionStrategy *k6tv1.EvictionStrategy, migratableCondStatus k8sv1.ConditionStatus) *k6tv1.VirtualMachineInstance {
	vmi := &k6tv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "testvmi",
		},
		Status: k6tv1.VirtualMachineInstanceStatus{
			NodeName: "testNode",
		},
	}

	if migratableCondStatus != k8sv1.ConditionUnknown {
		vmi.Status.Conditions = []k6tv1.VirtualMachineInstanceCondition{
			{
				Type:   k6tv1.VirtualMachineInstanceIsMigratable,
				Status: migratableCondStatus,
			},
		}
	}

	vmi.Spec.EvictionStrategy = evictionStrategy

	return vmi
}

func setupTestCollector() {
	instanceTypeInformer, _ = testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineInstancetype{})
	clusterInstanceTypeInformer, _ = testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterInstancetype{})
	preferenceInformer, _ = testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachinePreference{})
	clusterPreferenceInformer, _ = testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterPreference{})
	kvPodInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Pod{})

	_ = instanceTypeInformer.GetStore().Add(&instancetypev1beta1.VirtualMachineInstancetype{
		ObjectMeta: newObjectMetaForInstancetypes("i-managed", "kubevirt.io"),
	})
	_ = instanceTypeInformer.GetStore().Add(&instancetypev1beta1.VirtualMachineInstancetype{
		ObjectMeta: newObjectMetaForInstancetypes("i-unmanaged", "some-user"),
	})

	_ = clusterInstanceTypeInformer.GetStore().Add(&instancetypev1beta1.VirtualMachineClusterInstancetype{
		ObjectMeta: newObjectMetaForInstancetypes("ci-managed", "kubevirt.io"),
	})
	_ = clusterInstanceTypeInformer.GetStore().Add(&instancetypev1beta1.VirtualMachineClusterInstancetype{
		ObjectMeta: newObjectMetaForInstancetypes("ci-unmanaged", ""),
	})

	_ = preferenceInformer.GetStore().Add(&instancetypev1beta1.VirtualMachinePreference{
		ObjectMeta: newObjectMetaForInstancetypes("p-managed", "kubevirt.io"),
	})
	_ = preferenceInformer.GetStore().Add(&instancetypev1beta1.VirtualMachinePreference{
		ObjectMeta: newObjectMetaForInstancetypes("p-unmanaged", "some-vendor.com"),
	})

	_ = clusterPreferenceInformer.GetStore().Add(&instancetypev1beta1.VirtualMachineClusterPreference{
		ObjectMeta: newObjectMetaForInstancetypes("cp-managed", "kubevirt.io"),
	})

	_ = kvPodInformer.GetStore().Add(&k8sv1.Pod{
		ObjectMeta: newPodMetaForInformer("virt-launcher-testpod", "test-ns", "test-vmi-uid"),
	})
}

func newObjectMetaForInstancetypes(name, vendor string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:   name,
		Labels: map[string]string{instancetypeVendorLabel: vendor},
	}
}

func newPodMetaForInformer(name, namespace, createdByUID string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    map[string]string{"kubevirt.io/created-by": createdByUID},
	}
}
