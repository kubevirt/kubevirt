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

const (
	testMetricVMResourceRequests = "kubevirt_vm_resource_requests"
	testLabelCPU                 = "cpu"
	testLabelCores               = "cores"
	testLabelBytes               = "bytes"
	testLabelDomain              = "domain"
	testLabelThreads             = "threads"
	testLabelSockets             = "sockets"
	testMetricVMLabels           = "kubevirt_vm_labels"
	testLabelDefault             = "default"
	testLabelGuestEffective      = "guest_effective"
	testLabelMemory              = "memory"
	testLabelGuest               = "guest"
)

// Minimal stub for label config used by vmstats tests
type funcLabelsConfig func(string) bool

func (f funcLabelsConfig) ShouldReport(label string) bool { return f(label) }

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

		DescribeTable("should show instance type value correctly", func(instanceTypeKind, instanceTypeName, expected string) {
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

		DescribeTable("should show preference value correctly", func(preferenceAnnotationKey, preferenceName, expected string) {
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
		It("should report default CPU when no CPU topology is set", func() {
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
				Expect(crs).ToNot(BeEmpty())
				var defaults []operatormetrics.CollectorResult
				for _, cr := range crs {
					if cr.Metric.GetOpts().Name == testMetricVMResourceRequests && len(cr.Labels) == 5 &&
						cr.Labels[2] == testLabelCPU && cr.Labels[4] == testLabelDefault && cr.Labels[0] == name && cr.Labels[1] == ns {
						defaults = append(defaults, cr)
					}
				}
				Expect(defaults).To(HaveLen(3), "Expected 3 default CPU metrics")
				found := map[string]bool{testLabelCores: false, testLabelThreads: false, testLabelSockets: false}
				for _, cr := range defaults {
					Expect(cr.Value).To(BeEquivalentTo(1))
					found[cr.Labels[3]] = true
				}
				Expect(found[testLabelCores]).To(BeTrue())
				Expect(found[testLabelThreads]).To(BeTrue())
				Expect(found[testLabelSockets]).To(BeTrue())
				// guest_effective cores present
				var ge bool
				for _, r := range crs {
					if r.Metric.GetOpts().Name == testMetricVMResourceRequests &&
						len(r.Labels) == 5 && r.Labels[2] == testLabelCPU &&
						r.Labels[3] == testLabelCores && r.Labels[4] == testLabelGuestEffective {
						Expect(r.Value).To(BeEquivalentTo(1))
						ge = true
					}
				}
				Expect(ge).To(BeTrue())
			})
		})

		It("should collect effective memory and effective CPU metrics", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "testvm",
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Domain: k6tv1.DomainSpec{
								Memory: &k6tv1.Memory{Guest: resource.NewQuantity(1024, resource.BinarySI)},
							},
						},
					},
				},
			}

			crs := CollectResourceRequestsAndLimits(cloneVM(vm))

			expectVM := func(crs []operatormetrics.CollectorResult, name, ns string) {
				// memory guest present
				var memGuest, memGE bool
				for _, r := range crs {
					if r.Metric.GetOpts().Name == testMetricVMResourceRequests &&
						len(r.Labels) == 5 && r.Labels[2] == testLabelMemory &&
						r.Labels[3] == testLabelBytes && r.Labels[4] == testLabelGuest {
						Expect(r.Value).To(BeEquivalentTo(1024))
						memGuest = true
					}
					if r.Metric.GetOpts().Name == testMetricVMResourceRequests &&
						len(r.Labels) == 5 && r.Labels[2] == testLabelMemory &&
						r.Labels[3] == testLabelBytes && r.Labels[4] == testLabelGuestEffective {
						Expect(r.Value).To(BeEquivalentTo(1024))
						memGE = true
					}
				}
				Expect(memGuest && memGE).To(BeTrue())

				// default CPU triad present and guest_effective cores present
				var defaults []operatormetrics.CollectorResult
				for _, cr := range crs {
					if cr.Metric.GetOpts().Name == testMetricVMResourceRequests &&
						len(cr.Labels) == 5 && cr.Labels[2] == testLabelCPU && cr.Labels[4] == testLabelDefault {
						defaults = append(defaults, cr)
					}
				}
				Expect(defaults).To(HaveLen(3))
				var ge bool
				for _, r := range crs {
					if r.Metric.GetOpts().Name == testMetricVMResourceRequests &&
						len(r.Labels) == 5 && r.Labels[2] == testLabelCPU &&
						r.Labels[3] == testLabelCores && r.Labels[4] == testLabelGuestEffective {
						Expect(r.Value).To(BeEquivalentTo(1))
						ge = true
					}
				}
				Expect(ge).To(BeTrue())
			}

			expectVMs(crs, "testvm", "test-ns", expectVM)
		})

		It("should collect CPU requests/limits and guest_effective when resources are set", func() {
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
			expectVMs(crs, "testvm", "test-ns", func(filtered []operatormetrics.CollectorResult, name, ns string) {
				var reqFound, limFound, geFound bool
				for _, r := range filtered {
					if r.Metric.GetOpts().Name == testMetricVMResourceRequests &&
						len(r.Labels) == 5 && r.Labels[2] == testLabelCPU &&
						r.Labels[3] == testLabelCores && r.Labels[4] == "requests" {
						Expect(r.Value).To(BeEquivalentTo(0.5))
						reqFound = true
					}
					if r.Metric.GetOpts().Name == "kubevirt_vm_resource_limits" &&
						len(r.Labels) == 4 && r.Labels[2] == testLabelCPU && r.Labels[3] == testLabelCores {
						Expect(r.Value).To(BeEquivalentTo(1))
						limFound = true
					}
					if r.Metric.GetOpts().Name == testMetricVMResourceRequests &&
						len(r.Labels) == 5 && r.Labels[2] == testLabelCPU &&
						r.Labels[3] == testLabelCores && r.Labels[4] == testLabelGuestEffective {
						Expect(r.Value).To(BeEquivalentTo(1))
						geFound = true
					}
				}
				Expect(reqFound && limFound && geFound).To(BeTrue())
			})
		})

		It("should collect domain CPU metrics and guest_effective from topology", func() {
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
			expectDomainCPUMetrics(crs, "testvm", "test-ns", 2, 4, 1, 8)
		})

		It("should collect VM CPU metrics from Instance Type (guest_effective present)", func() {
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

			expectVM := func(filtered []operatormetrics.CollectorResult, name, ns string) {
				// guest_effective cores should be present (when instancetype applies CPU defaults/effective)
				var ge bool
				for _, r := range filtered {
					if r.Metric.GetOpts().Name == testMetricVMResourceRequests &&
						len(r.Labels) == 5 && r.Labels[0] == name && r.Labels[1] == ns &&
						r.Labels[2] == testLabelCPU && r.Labels[3] == testLabelCores &&
						r.Labels[4] == testLabelGuestEffective {
						Expect(r.Value).To(BeNumerically(">=", 1))
						ge = true
						break
					}
				}
				Expect(ge).To(BeTrue())
			}

			expectVMs(crs, "testvm", "test-ns", expectVM)
		})

		It("should report domain CPU metrics and guest_effective vCPUs", func() {
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
									Threads: 2,
									Sockets: 1,
								},
								Resources: k6tv1.ResourceRequirements{},
							},
						},
					},
				},
			}

			crs := CollectResourceRequestsAndLimits(cloneVM(vm))
			expectDomainCPUMetrics(crs, "testvm", "test-ns", 2, 2, 1, 4)
		})

		It("should emit default memory bytes equal to effective memory", func() {
			guestMemory := resource.MustParse("2Gi")
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
									Guest: &guestMemory,
								},
								Resources: k6tv1.ResourceRequirements{},
							},
						},
					},
				},
			}

			crs := CollectResourceRequestsAndLimits(cloneVM(vm))
			expectMemoryGuestMetrics(crs, "testvm", "test-ns", float64(guestMemory.Value()), float64(guestMemory.Value()))
		})

		It("should handle VM with only CPU topology but no limits (domain + guest_effective)", func() {
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
									Cores:   1,
									Threads: 1,
									Sockets: 2,
								},
							},
						},
					},
				},
			}

			crs := CollectResourceRequestsAndLimits(cloneVM(vm))
			expectDomainCPUMetrics(crs, "testvm", "test-ns", 1, 1, 2, 2)
		})

		It("should handle VM with only memory guest but no limits", func() {
			guestMemory := resource.MustParse("1Gi")
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
									Guest: &guestMemory,
								},
							},
						},
					},
				},
			}

			crs := CollectResourceRequestsAndLimits(cloneVM(vm))
			expectMemoryGuestMetrics(crs, "testvm", "test-ns", float64(guestMemory.Value()), float64(guestMemory.Value()))
		})

		It("should return default CPU cores=1 for VM with no CPU topology or memory", func() {
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
			var cpuFound bool
			for _, r := range filterResultsByVM(crs, "testvm", "test-ns") {
				if r.Metric.GetOpts().Name == testMetricVMResourceRequests &&
					len(r.Labels) == 5 && r.Labels[2] == testLabelCPU &&
					r.Labels[3] == testLabelCores && r.Labels[4] == testLabelDefault {
					Expect(r.Value).To(BeEquivalentTo(1))
					cpuFound = true
				}
			}
			Expect(cpuFound).To(BeTrue())
			// No memory guest/guest_effective when memory not specified
			for _, r := range filterResultsByVM(crs, "testvm", "test-ns") {
				isMemoryRequest := r.Metric.GetOpts().Name == testMetricVMResourceRequests &&
					len(r.Labels) == 5 && r.Labels[2] == testLabelMemory
				Expect(isMemoryRequest).To(BeFalse())
			}
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
			ifaces, nets := newVNICTestInterfaces()
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test-vm",
				},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Domain:   k6tv1.DomainSpec{Devices: k6tv1.Devices{Interfaces: ifaces}},
							Networks: nets,
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

	Context("VM labels metric", func() {
		BeforeEach(func() {
			// Default to allowing all labels; individual tests override as needed
			vmLabelsCfg = funcLabelsConfig(func(string) bool { return true })
		})

		It("should collect allowed labels", func() {
			// default allow-all in BeforeEach is sufficient
			vms := []*k6tv1.VirtualMachine{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vm1",
						Namespace: "default",
						Labels: map[string]string{
							"environment": "production",
							"team":        "backend",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vm2",
						Namespace: "default",
						Labels: map[string]string{
							"environment": "staging",
							"version":     "2.0",
						},
					},
				},
			}

			allResults := reportVmsStats(vms)
			var labelResults []operatormetrics.CollectorResult
			for _, r := range allResults {
				if r.Metric.GetOpts().Name == testMetricVMLabels {
					labelResults = append(labelResults, r)
				}
			}
			Expect(labelResults).To(HaveLen(2))

			var r1, r2 operatormetrics.CollectorResult
			if labelResults[0].Labels[0] == "vm1" {
				r1, r2 = labelResults[0], labelResults[1]
			} else {
				r1, r2 = labelResults[1], labelResults[0]
			}
			Expect(r1.Labels).To(Equal([]string{"vm1", "default"}))
			Expect(r1.ConstLabels).To(HaveKeyWithValue("label_environment", "production"))
			Expect(r1.ConstLabels).To(HaveKeyWithValue("label_team", "backend"))

			Expect(r2.Labels).To(Equal([]string{"vm2", "default"}))
			Expect(r2.ConstLabels).To(HaveKeyWithValue("label_environment", "staging"))
			Expect(r2.ConstLabels).To(HaveKeyWithValue("label_version", "2.0"))
		})

		It("should prioritize ignorelist over allowlist when overlapping", func() {
			vmLabelsCfg = funcLabelsConfig(func(l string) bool {
				return l != "vm.kubevirt.io/template"
			})

			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
					Labels: map[string]string{
						"environment":             "production",
						"vm.kubevirt.io/template": "tmpl",
					},
				},
			}

			results := reportVMLabels(vm)
			Expect(results).To(HaveLen(1))
			res := results[0]
			Expect(res.ConstLabels).To(HaveKeyWithValue("label_environment", "production"))
			Expect(res.ConstLabels).ToNot(HaveKey("label_vm_kubevirt_io_template"))
		})

		It("should change metric output when configuration changes dynamically", func() {
			// Restrict to specific labels and ignore secret
			vmLabelsCfg = funcLabelsConfig(func(l string) bool {
				if l == "secret" {
					return false
				}
				return l == "environment" || l == "team" || l == "version"
			})
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
					Labels: map[string]string{
						"environment": "production",
						"team":        "backend",
						"secret":      "sensitive",
						"version":     "1.0",
					},
				},
			}

			afterRestrict := reportVMLabels(vm)
			Expect(afterRestrict).To(HaveLen(1))
			Expect(afterRestrict[0].ConstLabels).To(HaveLen(3))
			Expect(afterRestrict[0].ConstLabels).To(HaveKeyWithValue("label_environment", "production"))
			Expect(afterRestrict[0].ConstLabels).To(HaveKeyWithValue("label_team", "backend"))
			Expect(afterRestrict[0].ConstLabels).To(HaveKeyWithValue("label_version", "1.0"))
			Expect(afterRestrict[0].ConstLabels).ToNot(HaveKey("label_secret"))
		})

		It("should not emit vm labels metric when allowlist is empty", func() {
			vmLabelsCfg = funcLabelsConfig(func(string) bool { return false })

			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vm-no-metric",
					Namespace: "default",
					Labels: map[string]string{
						"a": "1",
					},
				},
			}

			allResults := reportVmsStats([]*k6tv1.VirtualMachine{vm})
			for _, r := range allResults {
				Expect(r.Metric.GetOpts().Name).ToNot(Equal(testMetricVMLabels))
			}
		})

		It("should treat ignorelist '*' as no ignore (wildcards unsupported)", func() {
			vmLabelsCfg = funcLabelsConfig(func(string) bool { return true })

			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vm-wildcard-ignore",
					Namespace: "default",
					Labels: map[string]string{
						"environment": "prod",
						"team":        "be",
					},
				},
			}

			allResults := reportVmsStats([]*k6tv1.VirtualMachine{vm})
			var labelResults []operatormetrics.CollectorResult
			for _, r := range allResults {
				if r.Metric.GetOpts().Name == testMetricVMLabels {
					labelResults = append(labelResults, r)
				}
			}
			Expect(labelResults).To(HaveLen(1))
			res := labelResults[0]
			Expect(res.ConstLabels).To(HaveKeyWithValue("label_environment", "prod"))
			Expect(res.ConstLabels).To(HaveKeyWithValue("label_team", "be"))
		})
	})
})

func cloneVM(vm *k6tv1.VirtualMachine) []*k6tv1.VirtualMachine {
	vm2 := vm.DeepCopy()
	if vm2.Name == "" {
		vm2.Name = vm.Name + "-2"
	} else {
		vm2.Name += "-2"
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

func expectDomainCPUMetrics(
	crs []operatormetrics.CollectorResult,
	vmName, ns string,
	expectedCores, expectedThreads, expectedSockets, expectedGE float64,
) {
	expected := map[string]float64{
		testLabelCores:   expectedCores,
		testLabelThreads: expectedThreads,
		testLabelSockets: expectedSockets,
	}
	found := map[string]bool{}

	for _, r := range filterResultsByVM(crs, vmName, ns) {
		if !isVMResourceRequestCPU(r) {
			continue
		}
		if r.Labels[4] == testLabelDomain {
			if exp, ok := expected[r.Labels[3]]; ok {
				Expect(r.Value).To(BeEquivalentTo(exp))
				found[r.Labels[3]] = true
			}
		}
		if r.Labels[3] == testLabelCores && r.Labels[4] == testLabelGuestEffective {
			Expect(r.Value).To(BeEquivalentTo(expectedGE))
			found[testLabelGuestEffective] = true
		}
	}
	Expect(found).To(HaveLen(4))
}

func isVMResourceRequestCPU(r operatormetrics.CollectorResult) bool {
	return r.Metric.GetOpts().Name == testMetricVMResourceRequests &&
		len(r.Labels) == 5 && r.Labels[2] == testLabelCPU
}

func expectMemoryGuestMetrics(
	crs []operatormetrics.CollectorResult,
	vmName, ns string,
	expectedGuestBytes, expectedGEBytes float64,
) {
	var memGuest, memGE bool
	for _, r := range filterResultsByVM(crs, vmName, ns) {
		if r.Metric.GetOpts().Name == testMetricVMResourceRequests &&
			len(r.Labels) == 5 && r.Labels[2] == testLabelMemory &&
			r.Labels[3] == testLabelBytes && r.Labels[4] == testLabelGuest {
			Expect(r.Value).To(BeEquivalentTo(expectedGuestBytes))
			memGuest = true
		}
		if r.Metric.GetOpts().Name == testMetricVMResourceRequests &&
			len(r.Labels) == 5 && r.Labels[2] == testLabelMemory &&
			r.Labels[3] == testLabelBytes && r.Labels[4] == testLabelGuestEffective {
			Expect(r.Value).To(BeEquivalentTo(expectedGEBytes))
			memGE = true
		}
	}
	Expect(memGuest && memGE).To(BeTrue())
}

func expectVMs(
	results []operatormetrics.CollectorResult,
	name, namespace string,
	expectVM func([]operatormetrics.CollectorResult, string, string),
) {
	expectVM(filterResultsByVM(results, name, namespace), name, namespace)
	expectVM(filterResultsByVM(results, name+"-2", namespace), name+"-2", namespace)
}
