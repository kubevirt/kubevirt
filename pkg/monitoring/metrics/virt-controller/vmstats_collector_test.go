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

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
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

	Context("VM info", func() {
		setupTestCollector()

		It("should handle no VMs", func() {
			cr := CollectVMsInfo([]*k6tv1.VirtualMachine{})
			Expect(cr).To(BeEmpty())
		})

		It("should show annotation types and machine_type labels correctly", func() {
			vms := []*k6tv1.VirtualMachine{
				{
					Spec: k6tv1.VirtualMachineSpec{
						Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									annotationPrefix + "os":       "centos8",
									annotationPrefix + "workload": "server",
									annotationPrefix + "flavor":   "tiny",
									"other":                       "other",
								},
							},
							Spec: k6tv1.VirtualMachineInstanceSpec{
								Domain: k6tv1.DomainSpec{
									Machine: &k6tv1.Machine{
										Type: "q35",
									},
								},
							},
						},
					},
					Status: k6tv1.VirtualMachineStatus{
						PrintableStatus: k6tv1.VirtualMachineStatusCrashLoopBackOff,
					},
				},
				{
					Spec: k6tv1.VirtualMachineSpec{
						Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									annotationPrefix + "os":       "centos8",
									annotationPrefix + "workload": "server",
									annotationPrefix + "flavor":   "tiny",
									annotationPrefix + "phase":    "dummy",
								},
							},
							Spec: k6tv1.VirtualMachineInstanceSpec{
								Domain: k6tv1.DomainSpec{
									Machine: &k6tv1.Machine{
										Type: "q35",
									},
								},
							},
						},
					},
					Status: k6tv1.VirtualMachineStatus{
						PrintableStatus: k6tv1.VirtualMachineStatusCrashLoopBackOff,
					},
				},
			}

			crs := CollectVMsInfo(vms)
			Expect(crs).To(HaveLen(2), "Expected 2 metrics")

			for i, cr := range crs {
				Expect(cr).ToNot(BeNil())
				Expect(cr.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_info"))
				Expect(cr.Value).To(BeEquivalentTo(1))
				Expect(cr.Labels).To(HaveLen(10))

				Expect(cr.GetLabelValue("name")).To(Equal(vms[i].ObjectMeta.Name))
				Expect(cr.GetLabelValue("namespace")).To(Equal(vms[i].ObjectMeta.Namespace))

				os, workload, flavor := getSystemInfoFromAnnotations(vms[i].Spec.Template.ObjectMeta.Annotations)
				Expect(cr.GetLabelValue("os")).To(Equal(os))
				Expect(cr.GetLabelValue("workload")).To(Equal(workload))
				Expect(cr.GetLabelValue("flavor")).To(Equal(flavor))

				Expect(cr.GetLabelValue("machine_type")).To(Equal(vms[i].Spec.Template.Spec.Domain.Machine.Type))

				Expect(cr.GetLabelValue("status")).To(Equal("CrashLoopBackOff"))
				Expect(cr.GetLabelValue("status_group")).To(Equal("error"))
			}
		})

		DescribeTable("should show instance type value correctly", func(instanceTypeKind string, instanceTypeName string, expected string) {
			var instanceType *k6tv1.InstancetypeMatcher
			if instanceTypeName != "" {
				instanceType = &k6tv1.InstancetypeMatcher{
					Kind: instanceTypeKind,
					Name: instanceTypeName,
				}
			}

			vms := []*k6tv1.VirtualMachine{{Spec: k6tv1.VirtualMachineSpec{Instancetype: instanceType}}}
			crs := CollectVMsInfo(vms)
			Expect(crs).To(HaveLen(1), "Expected 1 metric")

			cr := crs[0]
			Expect(cr).ToNot(BeNil())
			Expect(cr.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_info"))
			Expect(cr.Value).To(BeEquivalentTo(1))
			Expect(cr.Labels).To(HaveLen(10))
			Expect(cr.GetLabelValue("instance_type")).To(Equal(expected))
		},
			Entry("with no instance type expect <none>", "VirtualMachineInstancetype", "", "<none>"),
			Entry("with managed instance type expect its name", "VirtualMachineInstancetype", "i-managed", "i-managed"),
			Entry("with custom instance type expect <other>", "VirtualMachineInstancetype", "i-unmanaged", "<other>"),
			Entry("with no cluster instance type expect <none>", "VirtualMachineClusterInstancetype", "", "<none>"),
			Entry("with managed cluster instance type expect its name", "VirtualMachineClusterInstancetype", "ci-managed", "ci-managed"),
			Entry("with custom cluster instance type expect <other>", "VirtualMachineClusterInstancetype", "ci-unmanaged", "<other>"),
		)

		DescribeTable("should show preference value correctly", func(preferenceAnnotationKey string, preferenceName string, expected string) {
			var preference *k6tv1.PreferenceMatcher
			if preferenceName != "" {
				preference = &k6tv1.PreferenceMatcher{
					Kind: preferenceAnnotationKey,
					Name: preferenceName,
				}
			}

			vms := []*k6tv1.VirtualMachine{{Spec: k6tv1.VirtualMachineSpec{Preference: preference}}}
			crs := CollectVMsInfo(vms)
			Expect(crs).To(HaveLen(1), "Expected 1 metric")

			cr := crs[0]

			Expect(cr.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_info"))
			Expect(cr.Value).To(BeEquivalentTo(1))
			Expect(cr.Labels).To(HaveLen(10))
			Expect(cr.GetLabelValue("preference")).To(Equal(expected))
		},
			Entry("with no preference expect <none>", "VirtualMachinePreference", "", "<none>"),
			Entry("with managed preference expect its name", "VirtualMachinePreference", "p-managed", "p-managed"),
			Entry("with custom preference expect <other>", "VirtualMachinePreference", "p-unmanaged", "<other>"),
			Entry("with no cluster preference expect <none>", "VirtualMachineClusterPreference", "", "<none>"),
			Entry("with managed cluster preference expect its name", "VirtualMachineClusterPreference", "cp-managed", "cp-managed"),
			Entry("with custom cluster preference expect <other>", "VirtualMachineClusterPreference", "cp-unmanaged", "<other>"),
		)
	})

	Context("VM Resource Requests", func() {
		It("should ignore VM with empty memory resource requests and limits", func() {
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

			cr := CollectResourceRequestsAndLimits([]*k6tv1.VirtualMachine{vm})
			Expect(cr).To(BeZero())
		})

		It("should collect VM memory resource requests and limits", func() {
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
									Limits: k8sv1.ResourceList{
										k8sv1.ResourceMemory: *resource.NewQuantity(2048, resource.BinarySI),
									},
								},
							},
						},
					},
				},
			}

			crs := CollectResourceRequestsAndLimits([]*k6tv1.VirtualMachine{vm})
			Expect(crs).To(HaveLen(2), "Expected 2 metrics")

			By("checking the resource requests")
			Expect(crs[0].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(crs[0].Value).To(BeEquivalentTo(1024))
			Expect(crs[0].Labels).To(Equal([]string{"testvm", "test-ns", "memory", "bytes"}))

			By("checking the resource limits")
			Expect(crs[1].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_limits"))
			Expect(crs[1].Value).To(BeEquivalentTo(2048))
			Expect(crs[1].Labels).To(Equal([]string{"testvm", "test-ns", "memory", "bytes"}))
		})

		It("should collect VM CPU resource requests and limits", func() {
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
									Limits: k8sv1.ResourceList{
										k8sv1.ResourceCPU: *resource.NewMilliQuantity(1000, resource.BinarySI),
									},
								},
							},
						},
					},
				},
			}

			crs := CollectResourceRequestsAndLimits([]*k6tv1.VirtualMachine{vm})
			Expect(crs).To(HaveLen(2), "Expected 2 metrics")

			By("checking the resource requests")
			Expect(crs[0].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(crs[0].Value).To(BeEquivalentTo(0.5))
			Expect(crs[0].Labels).To(Equal([]string{"testvm", "test-ns", "cpu", "cores"}))

			By("checking the resource limits")
			Expect(crs[1].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_limits"))
			Expect(crs[1].Value).To(BeEquivalentTo(1))
			Expect(crs[1].Labels).To(Equal([]string{"testvm", "test-ns", "cpu", "cores"}))
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

			crs := CollectResourceRequestsAndLimits([]*k6tv1.VirtualMachine{vm})
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

	Context("PVC allocated size metric collection", func() {
		BeforeEach(func() {
			persistentVolumeClaimInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		})

		createPVC := func(namespace, name string, size resource.Quantity, volumeMode *k8sv1.PersistentVolumeMode) *k8sv1.PersistentVolumeClaim {
			storageClassName := "rook-ceph-block"

			return &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
				Spec: k8sv1.PersistentVolumeClaimSpec{
					AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
					VolumeMode:  volumeMode,
					Resources: k8sv1.VolumeResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceStorage: size,
						},
					},
					StorageClassName: &storageClassName,
				},
			}
		}

		It("should collect PVC size metrics correctly", func() {
			pvc := createPVC("default", "test-vm-pvc", resource.MustParse("5Gi"), pointer.P(k8sv1.PersistentVolumeFilesystem))
			err := persistentVolumeClaimInformer.GetIndexer().Add(pvc)
			Expect(err).ToNot(HaveOccurred())

			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-vm",
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Volumes: []k6tv1.Volume{
								{
									Name: "rootdisk",
									VolumeSource: k6tv1.VolumeSource{
										PersistentVolumeClaim: &k6tv1.PersistentVolumeClaimVolumeSource{
											PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
												ClaimName: "test-vm-pvc",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			results := CollectDiskAllocatedSize([]*k6tv1.VirtualMachine{vm})

			Expect(results).ToNot(BeEmpty())
			Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vm_disk_allocated_size_bytes"))
			Expect(results[0].Value).To(Equal(float64(5 * 1024 * 1024 * 1024)))
			Expect(results[0].Labels).To(Equal([]string{"test-vm", "default", "test-vm-pvc", "Filesystem", "rootdisk"}))
		})

		It("should handle PVC with nil volume mode", func() {
			pvc := createPVC("default", "test-vm-pvc-nil-mode", resource.MustParse("3Gi"), nil)
			err := persistentVolumeClaimInformer.GetIndexer().Add(pvc)
			Expect(err).ToNot(HaveOccurred())

			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-vm-nil-mode",
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Volumes: []k6tv1.Volume{
								{
									Name: "rootdisk",
									VolumeSource: k6tv1.VolumeSource{
										PersistentVolumeClaim: &k6tv1.PersistentVolumeClaimVolumeSource{
											PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
												ClaimName: "test-vm-pvc-nil-mode",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			results := CollectDiskAllocatedSize([]*k6tv1.VirtualMachine{vm})

			Expect(results).ToNot(BeEmpty())
			Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vm_disk_allocated_size_bytes"))
			Expect(results[0].Value).To(Equal(float64(3 * 1024 * 1024 * 1024)))
			Expect(results[0].Labels).To(Equal([]string{"test-vm-nil-mode", "default", "test-vm-pvc-nil-mode", "null", "rootdisk"}))
		})
	})
})
