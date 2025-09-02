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

package virt_controller

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

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
			vms := cloneVM(createVM(status, t))

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
			vms := cloneVM(&k6tv1.VirtualMachine{
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
								Machine: &k6tv1.Machine{Type: "q35"},
							},
						},
					},
				},
				Status: k6tv1.VirtualMachineStatus{PrintableStatus: k6tv1.VirtualMachineStatusCrashLoopBackOff},
			})
			vms[1].Spec.Template.ObjectMeta.Annotations[annotationPrefix+"phase"] = "dummy"

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

				Expect(cr.GetLabelValue("status")).To(Equal("crashloopbackoff"))
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

			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns"},
				Spec:       k6tv1.VirtualMachineSpec{Instancetype: instanceType},
			}
			crs := CollectVMsInfo(cloneVM(vm))
			Expect(crs).To(HaveLen(2), "Expected 2 metrics")

			for _, cr := range crs {
				Expect(cr).ToNot(BeNil())
				Expect(cr.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_info"))
				Expect(cr.Value).To(BeEquivalentTo(1))
				Expect(cr.Labels).To(HaveLen(10))
				Expect(cr.GetLabelValue("instance_type")).To(Equal(expected))
			}
		},
			Entry("with no instance type expect empty string", "VirtualMachineInstancetype", "", ""),
			Entry("with managed instance type expect its name", "VirtualMachineInstancetype", "i-managed", "i-managed"),
			Entry("with custom instance type expect <other>", "VirtualMachineInstancetype", "i-unmanaged", "<other>"),
			Entry("with no cluster instance type expect empty string", "VirtualMachineClusterInstancetype", "", ""),
			Entry("with managed cluster instance type expect its name", "VirtualMachineClusterInstancetype", "ci-managed", "ci-managed"),
			Entry("with custom cluster instance type expect <other>", "VirtualMachineClusterInstancetype", "ci-unmanaged", "<other>"),
			Entry("with an instance type which no longer exists expect <other>", "VirtualMachineInstancetype", "i-gone", "<other>"),
			Entry("with a cluster instance type which no longer exists expect <other>", "VirtualMachineClusterInstancetype", "ci-gone", "<other>"),
			Entry("with lowercase instance type expect the same behavior", "virtualmachineinstancetype", "i-managed", "i-managed"),
			Entry("with lowercase cluster instance type expect the same behavior", "virtualmachineclusterinstancetype", "ci-managed", "ci-managed"),
		)

		DescribeTable("should show preference value correctly", func(preferenceAnnotationKey string, preferenceName string, expected string) {
			var preference *k6tv1.PreferenceMatcher
			if preferenceName != "" {
				preference = &k6tv1.PreferenceMatcher{
					Kind: preferenceAnnotationKey,
					Name: preferenceName,
				}
			}

			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns"},
				Spec:       k6tv1.VirtualMachineSpec{Preference: preference},
			}
			crs := CollectVMsInfo(cloneVM(vm))
			Expect(crs).To(HaveLen(2), "Expected 2 metrics")

			for _, cr := range crs {
				Expect(cr.Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_info"))
				Expect(cr.Value).To(BeEquivalentTo(1))
				Expect(cr.Labels).To(HaveLen(10))
				Expect(cr.GetLabelValue("preference")).To(Equal(expected))
			}
		},
			Entry("with no preference expect empty string", "VirtualMachinePreference", "", ""),
			Entry("with managed preference expect its name", "VirtualMachinePreference", "p-managed", "p-managed"),
			Entry("with custom preference expect <other>", "VirtualMachinePreference", "p-unmanaged", "<other>"),
			Entry("with no cluster preference expect empty string", "VirtualMachineClusterPreference", "", ""),
			Entry("with managed cluster preference expect its name", "VirtualMachineClusterPreference", "cp-managed", "cp-managed"),
			Entry("with custom cluster preference expect <other>", "VirtualMachineClusterPreference", "cp-unmanaged", "<other>"),
			Entry("with an preference which no longer exists expect <other>", "VirtualMachinePreference", "p-gone", "<other>"),
			Entry("with a cluster preference which no longer exists expect <other>", "VirtualMachineClusterPreference", "cp-gone", "<other>"),
			Entry("with lowercase preference expect the same behavior", "virtualmachinepreference", "p-managed", "p-managed"),
			Entry("with lowercase cluster preference expect the same behavior", "virtualmachineclusterpreference", "cp-managed", "cp-managed"),
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

			crs := CollectResourceRequestsAndLimits(cloneVM(vm))
			expectVMs(crs, "testvm", "test-ns", func(crs []operatormetrics.CollectorResult, name, ns string) {
				Expect(crs).To(HaveLen(3), "Expected 3 metrics")
				Expect(crs[0].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[0].Value).To(BeEquivalentTo(1))
				Expect(crs[0].Labels).To(Equal([]string{name, ns, "cpu", "cores", "default"}))
				Expect(crs[1].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[1].Value).To(BeEquivalentTo(1))
				Expect(crs[1].Labels).To(Equal([]string{name, ns, "cpu", "threads", "default"}))
				Expect(crs[2].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[2].Value).To(BeEquivalentTo(1))
				Expect(crs[2].Labels).To(Equal([]string{name, ns, "cpu", "sockets", "default"}))
			})
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

			crs := CollectResourceRequestsAndLimits(cloneVM(vm))

			expectVM := func(crs []operatormetrics.CollectorResult, name, ns string) {
				By("checking the resource requests")
				Expect(crs[0].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[0].Value).To(BeEquivalentTo(1024))
				Expect(crs[0].Labels).To(Equal([]string{name, ns, "memory", "bytes", "domain"}))

				By("checking the resource limits")
				Expect(crs[1].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_limits"))
				Expect(crs[1].Value).To(BeEquivalentTo(2048))
				Expect(crs[1].Labels).To(Equal([]string{name, ns, "memory", "bytes"}))

				Expect(crs[2].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[2].Value).To(BeEquivalentTo(1))
				Expect(crs[2].Labels).To(Equal([]string{name, ns, "cpu", "cores", "default"}))
				Expect(crs[3].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[3].Value).To(BeEquivalentTo(1))
				Expect(crs[3].Labels).To(Equal([]string{name, ns, "cpu", "threads", "default"}))
				Expect(crs[4].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[4].Value).To(BeEquivalentTo(1))
				Expect(crs[4].Labels).To(Equal([]string{name, ns, "cpu", "sockets", "default"}))
			}

			expectVMs(crs, "testvm", "test-ns", expectVM)
		})

		It("should collect VM memory guest and hugepages requests", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "testvm",
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Domain: k6tv1.DomainSpec{
								Memory: &k6tv1.Memory{
									Guest: resource.NewQuantity(1024, resource.BinarySI),
									Hugepages: &k6tv1.Hugepages{
										PageSize: "2Mi",
									},
								},
							},
						},
					},
				},
			}

			crs := CollectResourceRequestsAndLimits(cloneVM(vm))

			expectVM := func(crs []operatormetrics.CollectorResult, name, ns string) {
				By("checking the memory guest requests")
				Expect(crs[0].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[0].Value).To(BeEquivalentTo(1024))
				Expect(crs[0].Labels).To(Equal([]string{name, ns, "memory", "bytes", "guest"}))

				By("checking the memory hugepages requests")
				Expect(crs[1].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[1].Value).To(BeEquivalentTo(2097152))
				Expect(crs[1].Labels).To(Equal([]string{name, ns, "memory", "bytes", "hugepages"}))

				Expect(crs[2].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[2].Value).To(BeEquivalentTo(1))
				Expect(crs[2].Labels).To(Equal([]string{name, ns, "cpu", "cores", "default"}))
				Expect(crs[3].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[3].Value).To(BeEquivalentTo(1))
				Expect(crs[3].Labels).To(Equal([]string{name, ns, "cpu", "threads", "default"}))
				Expect(crs[4].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[4].Value).To(BeEquivalentTo(1))
				Expect(crs[4].Labels).To(Equal([]string{name, ns, "cpu", "sockets", "default"}))
			}

			expectVMs(crs, "testvm", "test-ns", expectVM)
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

			crs := CollectResourceRequestsAndLimits(cloneVM(vm))

			expectVM := func(crs []operatormetrics.CollectorResult, name, ns string) {
				Expect(crs).To(HaveLen(2), "Expected 2 metrics")
				By("checking the resource requests")
				Expect(crs[0].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[0].Value).To(BeEquivalentTo(0.5))
				Expect(crs[0].Labels).To(Equal([]string{name, ns, "cpu", "cores", "requests"}))
				By("checking the resource limits")
				Expect(crs[1].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_limits"))
				Expect(crs[1].Value).To(BeEquivalentTo(1))
				Expect(crs[1].Labels).To(Equal([]string{name, ns, "cpu", "cores"}))
			}

			expectVMs(crs, "testvm", "test-ns", expectVM)
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

			crs := CollectResourceRequestsAndLimits(cloneVM(vm))
			vm1 := filterResultsByVM(crs, "testvm", "test-ns")
			Expect(vm1).To(HaveLen(3), "Expected 3 metrics")

			Expect(vm1[0].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(vm1[0].Value).To(BeEquivalentTo(2))
			Expect(vm1[0].Labels).To(Equal([]string{"testvm", "test-ns", "cpu", "cores", "domain"}))

			Expect(vm1[1].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(vm1[1].Value).To(BeEquivalentTo(4))
			Expect(vm1[1].Labels).To(Equal([]string{"testvm", "test-ns", "cpu", "threads", "domain"}))

			Expect(vm1[2].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(vm1[2].Value).To(BeEquivalentTo(1))
			Expect(vm1[2].Labels).To(Equal([]string{"testvm", "test-ns", "cpu", "sockets", "domain"}))

			vm2 := filterResultsByVM(crs, "testvm-2", "test-ns")
			Expect(vm2).To(HaveLen(3), "Expected 3 metrics")
			Expect(vm2[0].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(vm2[0].Value).To(BeEquivalentTo(2))
			Expect(vm2[0].Labels).To(Equal([]string{"testvm-2", "test-ns", "cpu", "cores", "domain"}))
			Expect(vm2[1].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(vm2[1].Value).To(BeEquivalentTo(4))
			Expect(vm2[1].Labels).To(Equal([]string{"testvm-2", "test-ns", "cpu", "threads", "domain"}))
			Expect(vm2[2].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
			Expect(vm2[2].Value).To(BeEquivalentTo(1))
			Expect(vm2[2].Labels).To(Equal([]string{"testvm-2", "test-ns", "cpu", "sockets", "domain"}))
		})

		It("should collect VM CPU and Memory resource requests from Instance Type", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "testvm",
				},
				Spec: k6tv1.VirtualMachineSpec{
					Instancetype: &k6tv1.InstancetypeMatcher{
						Kind: "VirtualMachineClusterInstancetype",
						Name: "ci-managed",
					},
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{},
				},
			}

			crs := CollectResourceRequestsAndLimits(cloneVM(vm))

			expectVM := func(crs []operatormetrics.CollectorResult, name, ns string) {
				Expect(crs).To(HaveLen(4), "Expected 4 metrics")
				// memory from instancetype
				Expect(crs[0].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[0].Value).To(BeEquivalentTo(2048))
				Expect(crs[0].Labels).To(Equal([]string{name, ns, "memory", "bytes", "guest"}))
				// cpu cores/threads/sockets from instancetype
				Expect(crs[1].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[1].Value).To(BeEquivalentTo(1))
				Expect(crs[1].Labels).To(Equal([]string{name, ns, "cpu", "cores", "domain"}))
				Expect(crs[2].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[2].Value).To(BeEquivalentTo(1))
				Expect(crs[2].Labels).To(Equal([]string{name, ns, "cpu", "threads", "domain"}))
				Expect(crs[3].Metric.GetOpts().Name).To(ContainSubstring("kubevirt_vm_resource_requests"))
				Expect(crs[3].Value).To(BeEquivalentTo(2))
				Expect(crs[3].Labels).To(Equal([]string{name, ns, "cpu", "sockets", "domain"}))
			}

			expectVMs(crs, "testvm", "test-ns", expectVM)
		})
	})

	Context("PVC allocated size metric collection", func() {
		BeforeEach(func() {
			pvcInformer, _ := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
			stores.PersistentVolumeClaim = pvcInformer.GetIndexer()
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
			err := stores.PersistentVolumeClaim.Add(pvc)
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

			results := CollectDiskAllocatedSize(cloneVM(vm))

			Expect(results).ToNot(BeEmpty())
			expectVM := func(crs []operatormetrics.CollectorResult, name, ns string) {
				Expect(crs).To(HaveLen(1))
				Expect(crs[0].Metric.GetOpts().Name).To(Equal("kubevirt_vm_disk_allocated_size_bytes"))
				Expect(crs[0].Value).To(Equal(float64(5 * 1024 * 1024 * 1024)))
				Expect(crs[0].Labels).To(Equal([]string{name, ns, "test-vm-pvc", "Filesystem", "rootdisk"}))
			}
			expectVMs(results, "test-vm", "default", expectVM)
		})

		It("should handle PVC with nil volume mode", func() {
			pvc := createPVC("default", "test-vm-pvc-nil-mode", resource.MustParse("3Gi"), nil)
			err := stores.PersistentVolumeClaim.Add(pvc)
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

			results := CollectDiskAllocatedSize(cloneVM(vm))

			Expect(results).ToNot(BeEmpty())
			expectVM := func(crs []operatormetrics.CollectorResult, name, ns string) {
				Expect(crs).To(HaveLen(1))
				Expect(crs[0].Metric.GetOpts().Name).To(Equal("kubevirt_vm_disk_allocated_size_bytes"))
				Expect(crs[0].Value).To(Equal(float64(3 * 1024 * 1024 * 1024)))
				Expect(crs[0].Labels).To(Equal([]string{name, ns, "test-vm-pvc-nil-mode", "", "rootdisk"}))
			}
			expectVMs(results, "test-vm-nil-mode", "default", expectVM)
		})

		It("should prioritize DataVolume template size over PVC size", func() {
			pvc := createPVC("default", "test-dv-pvc", resource.MustParse("5Gi"), pointer.P(k8sv1.PersistentVolumeFilesystem))
			err := stores.PersistentVolumeClaim.Add(pvc)
			Expect(err).ToNot(HaveOccurred())

			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-vm-dv",
				},
				Spec: k6tv1.VirtualMachineSpec{
					DataVolumeTemplates: []k6tv1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-dv-pvc",
							},
							Spec: cdiv1.DataVolumeSpec{
								PVC: &k8sv1.PersistentVolumeClaimSpec{
									Resources: k8sv1.VolumeResourceRequirements{
										Requests: k8sv1.ResourceList{
											k8sv1.ResourceStorage: resource.MustParse("2Gi"),
										},
									},
								},
							},
						},
					},
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Volumes: []k6tv1.Volume{
								{
									Name: "datavolumedisk",
									VolumeSource: k6tv1.VolumeSource{
										DataVolume: &k6tv1.DataVolumeSource{
											Name: "test-dv-pvc",
										},
									},
								},
							},
						},
					},
				},
			}

			results := CollectDiskAllocatedSize(cloneVM(vm))

			Expect(results).ToNot(BeEmpty())
			expectVM := func(crs []operatormetrics.CollectorResult, name, ns string) {
				Expect(crs).To(HaveLen(1))
				Expect(crs[0].Metric.GetOpts().Name).To(Equal("kubevirt_vm_disk_allocated_size_bytes"))
				Expect(crs[0].Value).To(Equal(float64(2 * 1024 * 1024 * 1024)))
				Expect(crs[0].Labels).To(Equal([]string{name, ns, "test-dv-pvc", "Filesystem", "datavolumedisk"}))
			}
			expectVMs(results, "test-vm-dv", "default", expectVM)
		})
	})

	Context("VM creation time metric collection", func() {
		It("should collect VM creation time correctly", func() {
			testTime := time.Now()

			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         "test-ns",
					Name:              "test-vm",
					CreationTimestamp: metav1.NewTime(testTime),
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{},
				},
			}

			results := collectVMCreationTimestamp(cloneVM(vm))

			Expect(results).ToNot(BeEmpty())
			expectVM := func(crs []operatormetrics.CollectorResult, name, ns string) {
				Expect(crs).To(HaveLen(1))
				Expect(crs[0].Metric.GetOpts().Name).To(Equal("kubevirt_vm_create_date_timestamp_seconds"))
				Expect(crs[0].Value).To(Equal(float64(testTime.Unix())))
				Expect(crs[0].Labels).To(Equal([]string{name, ns}))
			}
			expectVMs(results, "test-vm", "test-ns", expectVM)
		})

		It("should collect correct creation times for multiple VMs", func() {
			vm1 := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         "test-ns",
					Name:              "test-vm1",
					CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Hour)),
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{},
				},
			}
			vm2 := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         "test-ns",
					Name:              "test-vm2",
					CreationTimestamp: metav1.NewTime(time.Now().Add(-2 * time.Hour)),
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{},
				},
			}

			vms := []*k6tv1.VirtualMachine{vm1, vm2}
			results := collectVMCreationTimestamp(vms)

			Expect(results).To(HaveLen(2))
			Expect(results[0].Value).To(Equal(float64(vm1.CreationTimestamp.Unix())))
			Expect(results[1].Value).To(Equal(float64(vm2.CreationTimestamp.Unix())))
		})

		It("metric should not exists if the VM creation timestamp is zero", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         "test-ns",
					Name:              "test-vm-zero-time",
					CreationTimestamp: metav1.Time{},
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{},
				},
			}

			results := collectVMCreationTimestamp(cloneVM(vm))

			Expect(filterResultsByVM(results, "test-vm-zero-time", "test-ns")).To(BeEmpty())
			Expect(filterResultsByVM(results, "test-vm-zero-time-2", "test-ns")).To(BeEmpty())
		})
	})

	Context("VM vNIC info", func() {
		It("should collect metrics for vNICs with various binding types, including PluginBinding", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test-vm",
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Domain: k6tv1.DomainSpec{
								Devices: k6tv1.Devices{
									Interfaces: []k6tv1.Interface{
										{
											Name: "iface1",
											InterfaceBindingMethod: k6tv1.InterfaceBindingMethod{
												Bridge: &k6tv1.InterfaceBridge{},
											},
											Model: "virtio",
										},
										{
											Name: "iface2",
											InterfaceBindingMethod: k6tv1.InterfaceBindingMethod{
												Masquerade: &k6tv1.InterfaceMasquerade{},
											},
											Model: "e1000e",
										},
										{
											Name: "iface3",
											InterfaceBindingMethod: k6tv1.InterfaceBindingMethod{
												SRIOV: &k6tv1.InterfaceSRIOV{},
											},
										},
										{
											Name:    "iface4",
											Binding: &k6tv1.PluginBinding{Name: "custom-plugin"},
										},
									},
								},
							},
							Networks: []k6tv1.Network{
								{
									Name:          "iface1",
									NetworkSource: k6tv1.NetworkSource{Pod: &k6tv1.PodNetwork{}},
								},
								{
									Name:          "iface2",
									NetworkSource: k6tv1.NetworkSource{Pod: &k6tv1.PodNetwork{}},
								},
								{
									Name:          "iface3",
									NetworkSource: k6tv1.NetworkSource{Multus: &k6tv1.MultusNetwork{NetworkName: "multus-net"}},
								},
								{
									Name:          "iface4",
									NetworkSource: k6tv1.NetworkSource{Multus: &k6tv1.MultusNetwork{NetworkName: "custom-net"}},
								},
							},
						},
					},
				},
			}

			metrics := CollectVmsVnicInfo(cloneVM(vm))
			Expect(metrics).To(HaveLen(8), "Expected metrics for all vNICs in both VMs")

			vm1 := filterResultsByVM(metrics, "test-vm", "test-ns")
			Expect(vm1).To(HaveLen(4))
			Expect(vm1[0].Labels).To(Equal([]string{"test-vm", "test-ns", "iface1", "core", "pod networking", "bridge", "virtio"}))
			Expect(vm1[1].Labels).To(Equal([]string{"test-vm", "test-ns", "iface2", "core", "pod networking", "masquerade", "e1000e"}))
			Expect(vm1[2].Labels).To(Equal([]string{"test-vm", "test-ns", "iface3", "core", "multus-net", "sriov", "<none>"}))
			Expect(vm1[3].Labels).To(Equal([]string{"test-vm", "test-ns", "iface4", "plugin", "custom-net", "custom-plugin", "<none>"}))

			vm2 := filterResultsByVM(metrics, "test-vm-2", "test-ns")
			Expect(vm2).To(HaveLen(4))
			Expect(vm2[0].Labels).To(Equal([]string{"test-vm-2", "test-ns", "iface1", "core", "pod networking", "bridge", "virtio"}))
			Expect(vm2[1].Labels).To(Equal([]string{"test-vm-2", "test-ns", "iface2", "core", "pod networking", "masquerade", "e1000e"}))
			Expect(vm2[2].Labels).To(Equal([]string{"test-vm-2", "test-ns", "iface3", "core", "multus-net", "sriov", "<none>"}))
			Expect(vm2[3].Labels).To(Equal([]string{"test-vm-2", "test-ns", "iface4", "plugin", "custom-net", "custom-plugin", "<none>"}))
		})
		It("should not collect kubevirt_vm_vnic_info metric if no network defined", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test-vm",
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Domain: k6tv1.DomainSpec{
								Devices: k6tv1.Devices{
									Interfaces: []k6tv1.Interface{
										{
											Name: "iface1",
											InterfaceBindingMethod: k6tv1.InterfaceBindingMethod{
												Bridge: &k6tv1.InterfaceBridge{},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			metrics := CollectVmsVnicInfo(cloneVM(vm))
			Expect(filterResultsByVM(metrics, "test-vm", "test-ns")).To(BeEmpty())
			Expect(filterResultsByVM(metrics, "test-vm-2", "test-ns")).To(BeEmpty())
		})
	})
})

// Add helpers for two-VM testing
func cloneVM(vm *k6tv1.VirtualMachine) []*k6tv1.VirtualMachine {
	vm2 := vm.DeepCopy()
	if vm2.Name == "" {
		vm2.Name = vm.Name + "-2"
	} else {
		vm2.Name = vm2.Name + "-2"
	}
	if vm2.Namespace == "" {
		vm2.Namespace = vm.Namespace
	}
	return []*k6tv1.VirtualMachine{vm, vm2}
}

func filterResultsByVM(crs []operatormetrics.CollectorResult, name, namespace string) []operatormetrics.CollectorResult {
	var out []operatormetrics.CollectorResult
	for _, r := range crs {
		if len(r.Labels) >= 2 && r.Labels[0] == name && r.Labels[1] == namespace {
			out = append(out, r)
		}
	}
	return out
}

func expectVMs(
	results []operatormetrics.CollectorResult,
	name, namespace string,
	expectVM func([]operatormetrics.CollectorResult, string, string),
) {
	expectVM(filterResultsByVM(results, name, namespace), name, namespace)
	expectVM(filterResultsByVM(results, name+"-2", namespace), name+"-2", namespace)
}
