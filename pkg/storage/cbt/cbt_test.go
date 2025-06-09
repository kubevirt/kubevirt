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

package cbt_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var (
	labelSelector = &metav1.LabelSelector{
		MatchLabels: cbt.CBTLabel,
	}
)

var _ = Describe("CBT", func() {
	var (
		kv             *v1.KubeVirt
		k8sClient      *k8sfake.Clientset
		virtClient     *kubecli.MockKubevirtClient
		virtFakeClient *fake.Clientset
		config         *virtconfig.ClusterConfig
		kvStore        cache.Store
		nsStore        cache.Store
		vm             *v1.VirtualMachine
		vmi            *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		virtClient = kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		virtFakeClient = fake.NewSimpleClientset()
		config, _, kvStore = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(
			virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault),
		).AnyTimes()
		namespaceInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Namespace{})
		nsStore = namespaceInformer.GetStore()

		k8sClient = k8sfake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
	})

	updateKubeVirtWithLabelSelector := func(vmLabelSelector, nsLabelSelector *metav1.LabelSelector, enableFeatureGate bool) {
		kv = &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					ChangedBlockTrackingLabelSelectors: &v1.ChangedBlockTrackingSelectors{
						VirtualMachineLabelSelector: vmLabelSelector,
						NamespaceLabelSelector:      nsLabelSelector,
					},
				},
			},
		}
		if enableFeatureGate {
			kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{
				FeatureGates: []string{featuregate.IncrementalBackupGate},
			}
		}
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	}

	disableFeatureGate := func() {
		kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	}

	// Helper functions for test setup
	setupVMMatchingLabelSelector := func() {
		updateKubeVirtWithLabelSelector(labelSelector, nil, true)
		vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
		vm = libvmi.NewVirtualMachine(vmi, libvmi.WithLabels(cbt.CBTLabel))
	}

	setupVMNotMatchingSelector := func() {
		updateKubeVirtWithLabelSelector(labelSelector, nil, true)
		vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
		vm = libvmi.NewVirtualMachine(vmi) // No CBT labels
	}

	setupNamespaceMatchingSelector := func() {
		updateKubeVirtWithLabelSelector(nil, labelSelector, true)
		ns := &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   k8sv1.NamespaceDefault,
				Labels: cbt.CBTLabel,
			},
		}
		nsStore.Add(ns)
		vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
		vm = libvmi.NewVirtualMachine(vmi) // No CBT labels on VM
	}

	Context("SyncVMChangedBlockTrackingState", func() {
		DescribeTable("No kubevirt CR ChangedBlockTrackingLabelSelectors expect no updates", func(vmiExists bool) {
			kv = &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{},
				},
			}
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)

			vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vm = libvmi.NewVirtualMachine(vmi, libvmi.WithLabels(cbt.CBTLabel))
			updatedVM := vm.DeepCopy()
			var updatedVMI *v1.VirtualMachineInstance
			if vmiExists {
				updatedVMI = vmi.DeepCopy()
			}
			cbt.SyncVMChangedBlockTrackingState(updatedVM, updatedVMI, config, nsStore)
			Expect(updatedVM).To(Equal(vm))
			if vmiExists {
				Expect(updatedVMI).To(Equal(vmi))
			} else {
				Expect(updatedVMI).To(BeNil())
			}
		},
			Entry("VM only", false),
			Entry("VM and VMI", true),
		)

		Context("Enable CBT Transitions - VM matches Label Selector", func() {
			BeforeEach(setupVMMatchingLabelSelector)

			Context("No VMI scenarios", func() {
				DescribeTable("should transition VM state correctly when no VMI exists",
					func(initialVMState, expectedVMState v1.ChangedBlockTrackingState) {
						cbt.SetCBTState(&vm.Status.ChangedBlockTracking, initialVMState)
						cbt.SyncVMChangedBlockTrackingState(vm, nil, config, nsStore)
						Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(expectedVMState))
					},
					Entry("Undefined -> Initializing", v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingInitializing),
					Entry("PendingRestart -> Initializing", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingInitializing),
					Entry("Initializing -> Initializing", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingInitializing),
					Entry("Disabled -> Initializing", v1.ChangedBlockTrackingDisabled, v1.ChangedBlockTrackingInitializing),
					Entry("Enabled -> Enabled", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled),
				)
			})

			Context("VMI exists scenarios", func() {
				DescribeTable("should transition VM state correctly based on VMI state",
					func(initialVMState, vmiState, expectedVMState v1.ChangedBlockTrackingState) {
						cbt.SetCBTState(&vm.Status.ChangedBlockTracking, initialVMState)
						cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, vmiState)
						cbt.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
						Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(expectedVMState))
					},
					// From Undefined
					Entry("Undefined + any VMI -> PendingRestart", v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingPendingRestart),

					// From PendingRestart
					Entry("PendingRestart + VMI Enabled -> Enabled", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled),
					Entry("PendingRestart + VMI Initializing -> Initializing", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingInitializing),
					Entry("PendingRestart + VMI Undefined -> PendingRestart", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingPendingRestart),

					// From Initializing
					Entry("Initializing + VMI Enabled -> Enabled", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled),
					Entry("Initializing + VMI Initializing -> Initializing", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingInitializing),
					Entry("Initializing + VMI Undefined -> Initializing", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingInitializing),

					// From Enabled
					Entry("Enabled + VMI Enabled -> Enabled", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled),
					Entry("Enabled + VMI Initializing -> Initializing", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingInitializing),
					Entry("Enabled + VMI Undefined -> Initializing", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingInitializing),

					// From Disabled
					Entry("Disabled + VMI Enabled -> Enabled", v1.ChangedBlockTrackingDisabled, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled),
					Entry("Disabled + VMI Initializing -> Initializing", v1.ChangedBlockTrackingDisabled, v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingInitializing),
					Entry("Disabled + VMI Undefined -> PendingRestart", v1.ChangedBlockTrackingDisabled, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingPendingRestart),
				)
			})
		})

		Context("Enable CBT Transitions - Namespace matches Label Selector", func() {
			BeforeEach(setupNamespaceMatchingSelector)

			Context("No VMI scenarios", func() {
				DescribeTable("should transition VM state correctly when no VMI exists",
					func(initialVMState, expectedVMState v1.ChangedBlockTrackingState) {
						cbt.SetCBTState(&vm.Status.ChangedBlockTracking, initialVMState)
						cbt.SyncVMChangedBlockTrackingState(vm, nil, config, nsStore)
						Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(expectedVMState))
					},
					Entry("Undefined -> Initializing", v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingInitializing),
					Entry("PendingRestart -> Initializing", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingInitializing),
					Entry("Initializing -> Initializing", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingInitializing),
					Entry("Disabled -> Initializing", v1.ChangedBlockTrackingDisabled, v1.ChangedBlockTrackingInitializing),
					Entry("Enabled -> Enabled", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled),
				)
			})

			Context("VMI exists scenarios", func() {
				DescribeTable("should transition VM state correctly based on VMI state",
					func(initialVMState, vmiState, expectedVMState v1.ChangedBlockTrackingState) {
						cbt.SetCBTState(&vm.Status.ChangedBlockTracking, initialVMState)
						cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, vmiState)
						cbt.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
						Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(expectedVMState))
					},
					Entry("Undefined + VMI Undefined -> PendingRestart", v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingPendingRestart),
					Entry("PendingRestart + VMI Enabled -> Enabled", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled),
					Entry("Initializing + VMI Enabled -> Enabled", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled),
					Entry("Enabled + VMI Enabled -> Enabled", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled),
				)
			})
		})

		Context("Disable CBT Transitions - VM does not match Label Selector", func() {
			BeforeEach(setupVMNotMatchingSelector)

			Context("No VMI scenarios", func() {
				DescribeTable("should transition VM state correctly when no VMI exists",
					func(initialVMState, expectedVMState v1.ChangedBlockTrackingState) {
						cbt.SetCBTState(&vm.Status.ChangedBlockTracking, initialVMState)
						cbt.SyncVMChangedBlockTrackingState(vm, nil, config, nsStore)
						Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(expectedVMState))
					},
					Entry("Undefined -> Undefined", v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingUndefined),
					Entry("PendingRestart -> Disabled", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingDisabled),
					Entry("Initializing -> Disabled", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingDisabled),
					Entry("Enabled -> Disabled", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingDisabled),
					Entry("Disabled -> Disabled", v1.ChangedBlockTrackingDisabled, v1.ChangedBlockTrackingDisabled),
				)
			})

			Context("VMI exists scenarios", func() {
				DescribeTable("should transition VM state correctly based on VMI state",
					func(initialVMState, vmiState, expectedVMState v1.ChangedBlockTrackingState) {
						cbt.SetCBTState(&vm.Status.ChangedBlockTracking, initialVMState)
						cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, vmiState)
						cbt.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
						Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(expectedVMState))
					},
					// From Undefined - no change
					Entry("Undefined + any VMI -> Undefined", v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingUndefined),

					// Need restart when VMI has active CBT
					Entry("PendingRestart + VMI Enabled -> PendingRestart", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingPendingRestart),
					Entry("PendingRestart + VMI Initializing -> PendingRestart", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingPendingRestart),
					Entry("Initializing + VMI Enabled -> PendingRestart", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingPendingRestart),
					Entry("Initializing + VMI Initializing -> PendingRestart", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingPendingRestart),
					Entry("Enabled + VMI Enabled -> PendingRestart", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingPendingRestart),
					Entry("Enabled + VMI Initializing -> PendingRestart", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingPendingRestart),

					// Can disable immediately when VMI has no CBT
					Entry("PendingRestart + VMI Undefined -> Disabled", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingDisabled),
					Entry("Initializing + VMI Undefined -> Disabled", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingDisabled),
					Entry("Enabled + VMI Undefined -> Disabled", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingDisabled),

					// Disabled can disable immediately
					Entry("Disabled + VMI Enabled -> Disabled", v1.ChangedBlockTrackingDisabled, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingDisabled),
					Entry("Disabled + VMI Initializing -> Disabled", v1.ChangedBlockTrackingDisabled, v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingDisabled),
					Entry("Disabled + VMI Undefined -> Disabled", v1.ChangedBlockTrackingDisabled, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingDisabled),
				)
			})
		})

		Context("VM matches VM Label Selector", func() {
			BeforeEach(func() {
				setupVMMatchingLabelSelector()
			})

			DescribeTable("should set CBT state to ", func(vmiExists, fgDisabled bool, expectedStatus v1.ChangedBlockTrackingState) {
				if fgDisabled {
					disableFeatureGate()
				}
				if !vmiExists {
					vmi = nil
				}
				cbt.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
				Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(expectedStatus))
			},
				Entry("Initializing if VMI does not exist", false, false, v1.ChangedBlockTrackingInitializing),
				Entry("PendingRestart if VMI exists and cbtStatus is undefined", true, false, v1.ChangedBlockTrackingPendingRestart),
				Entry("IncrementalBackupFeatureGateDisabled if FG is disabled VMI does not exist", false, true, v1.ChangedBlockTrackingFGDisabled),
				Entry("IncrementalBackupFeatureGateDisabled if FG is disabled VMI exist", true, true, v1.ChangedBlockTrackingFGDisabled),
			)

			It("should set CBT state to enabled if vmi state is enabled", func() {
				cbt.SetCBTState(&vm.Status.ChangedBlockTracking, v1.ChangedBlockTrackingInitializing)
				cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingEnabled)
				cbt.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
				Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(v1.ChangedBlockTrackingEnabled))
			})
		})

		Context("VM namespace matches Namespace Label Selector", func() {
			BeforeEach(func() {
				setupNamespaceMatchingSelector()
			})

			It("should set VM CBT state to Initializing when namespace matches if VMI does not exist", func() {
				cbt.SyncVMChangedBlockTrackingState(vm, nil, config, nsStore)
				Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(v1.ChangedBlockTrackingInitializing))
			})

			DescribeTable("should set CBT state to ", func(vmiExists, fgDisabled bool, expectedStatus v1.ChangedBlockTrackingState) {
				if fgDisabled {
					disableFeatureGate()
				}
				if !vmiExists {
					vmi = nil
				}
				cbt.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
				Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(expectedStatus))
			},
				Entry("Initializing  for VM when namespace matches if VMI does not exist", false, false, v1.ChangedBlockTrackingInitializing),
				Entry("PendingRestart for VM when namespace matches if VMI exist", true, false, v1.ChangedBlockTrackingPendingRestart),
				Entry("IncrementalBackupFeatureGateDisabled if FG is disabled VMI does not exist", false, true, v1.ChangedBlockTrackingFGDisabled),
				Entry("IncrementalBackupFeatureGateDisabled if FG is disabled VMI exist", true, true, v1.ChangedBlockTrackingFGDisabled),
			)
		})

		Context("Edge Cases and Error Handling", func() {
			BeforeEach(setupVMMatchingLabelSelector)

			It("should reset invalid VM CBT state to Undefined", func() {
				cbt.SetCBTState(&vm.Status.ChangedBlockTracking, "invalid-state")
				cbt.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
				Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(v1.ChangedBlockTrackingUndefined))
			})

			Context("VM no longer matches Label Selector", func() {
				BeforeEach(func() {
					setupVMNotMatchingSelector()
				})

				It("should handle empty CBT state as Undefined", func() {
					cbt.SetCBTState(&vm.Status.ChangedBlockTracking, v1.ChangedBlockTrackingUndefined)
					cbt.SyncVMChangedBlockTrackingState(vm, nil, config, nsStore)
					Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(v1.ChangedBlockTrackingUndefined))
				})

				It("should disable CBT if VMI is nil and vm has CBT state", func() {
					cbt.SetCBTState(&vm.Status.ChangedBlockTracking, v1.ChangedBlockTrackingEnabled)
					cbt.SyncVMChangedBlockTrackingState(vm, nil, config, nsStore)
					Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(v1.ChangedBlockTrackingDisabled))
				})

				DescribeTable("should set CBT to", func(vmCBTState, expectedState v1.ChangedBlockTrackingState, vmiStateExists bool) {
					cbt.SetCBTState(&vm.Status.ChangedBlockTracking, vmCBTState)
					if vmiStateExists {
						cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingInitializing)
					}
					cbt.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
					Expect(cbt.CBTState(vm.Status.ChangedBlockTracking)).To(Equal(expectedState))
				},
					Entry("PendingRestart if VMI exists and has CBT enabled", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingPendingRestart, true),
					Entry("PendingRestart if VMI exists and has CBT Initializing", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingPendingRestart, true),
					Entry("PendingRestart if VMI exists and has CBT PendingRestart", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingPendingRestart, true),
					Entry("Disabled if VMI doesnt exists and has CBT enabled", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingDisabled, false),
					Entry("Disabled if VMI doesnt exists and has CBT Initializing", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingDisabled, false),
					Entry("Disabled if VMI doesnt exists and has CBT PendingRestart", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingDisabled, false),
				)
			})
		})
	})

	Context("SetChangedBlockTrackingOnVMI", func() {
		BeforeEach(func() {
			setupVMNotMatchingSelector()
		})
		It("IncrementalBackup featuregate disabled VMI cbt state should be undefined", func() {
			disableFeatureGate()
			vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vm = libvmi.NewVirtualMachine(vmi, libvmi.WithLabels(cbt.CBTLabel))
			cbt.SetChangedBlockTrackingOnVMI(vm, vmi, config, nsStore)
			Expect(cbt.CBTState(vmi.Status.ChangedBlockTracking)).To(Equal(v1.ChangedBlockTrackingUndefined))
		})
		It("VM matches VM Label Selector should set VMI state to Initializing", func() {
			setupVMMatchingLabelSelector()
			cbt.SetChangedBlockTrackingOnVMI(vm, vmi, config, nsStore)
			Expect(cbt.CBTState(vmi.Status.ChangedBlockTracking)).To(Equal(v1.ChangedBlockTrackingInitializing))
		})
		It("VM doesnt match VM Label Selector and VM CBT state exists should set VMI state to Disabled", func() {
			cbt.SetCBTState(&vm.Status.ChangedBlockTracking, v1.ChangedBlockTrackingEnabled)
			cbt.SetChangedBlockTrackingOnVMI(vm, vmi, config, nsStore)
			Expect(cbt.CBTState(vmi.Status.ChangedBlockTracking)).To(Equal(v1.ChangedBlockTrackingDisabled))
		})

		It("VM doesnt match VM Label Selector and VM CBT state doesnt exist shouldn't set VMI CBT state", func() {
			cbt.SetChangedBlockTrackingOnVMI(vm, vmi, config, nsStore)
			Expect(cbt.CBTState(vmi.Status.ChangedBlockTracking)).To(Equal(v1.ChangedBlockTrackingUndefined))
		})
	})

	Context("IsCBTEligibleVolume", func() {
		DescribeTable("should return correct eligibility for volume types",
			func(volumeSource v1.VolumeSource, expected bool) {
				volume := &v1.Volume{Name: "test-volume", VolumeSource: volumeSource}
				Expect(cbt.IsCBTEligibleVolume(volume)).To(Equal(expected))
			},
			Entry("PVC", v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "test-pvc"}}}, true),
			Entry("DataVolume", v1.VolumeSource{DataVolume: &v1.DataVolumeSource{Name: "test-dv"}}, true),
			Entry("HostDisk", v1.VolumeSource{HostDisk: &v1.HostDisk{Path: "/tmp/disk.img"}}, true),
			Entry("ContainerDisk", v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "test:latest"}}, false),
			Entry("ConfigMap", v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: k8sv1.LocalObjectReference{Name: "test-config"}}}, false),
			Entry("Secret", v1.VolumeSource{Secret: &v1.SecretVolumeSource{SecretName: "test-secret"}}, false),
			Entry("EmptyDisk", v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{Capacity: resource.MustParse("1Gi")}}, false),
			Entry("CloudInit", v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: "test"}}, false),
		)
	})

	Context("SetChangedBlockTrackingOnVMIFromDomain", func() {
		BeforeEach(func() {
			vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingInitializing)
		})

		pvcVolume := func(name, claimName string) v1.Volume {
			return v1.Volume{
				Name: name,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
					},
				},
			}
		}

		diskWithDataStore := func(name string, hasDataStore bool) api.Disk {
			disk := api.Disk{Alias: api.NewUserDefinedAlias(name), Source: api.DiskSource{}}
			if hasDataStore {
				disk.Source.DataStore = &api.DataStore{Type: "file"}
			}
			return disk
		}

		DescribeTable("should handle CBT enablement based on volume/disk configuration",
			func(volumes []v1.Volume, disks []api.Disk, expectedState v1.ChangedBlockTrackingState) {
				vmi.Spec.Volumes = volumes
				domain := &api.Domain{Spec: api.DomainSpec{Devices: api.Devices{Disks: disks}}}
				cbt.SetChangedBlockTrackingOnVMIFromDomain(vmi, domain)
				Expect(cbt.CBTState(vmi.Status.ChangedBlockTracking)).To(Equal(expectedState))
			},
			Entry("all eligible volumes have DataStore",
				[]v1.Volume{pvcVolume("pvc1", "test-pvc"), pvcVolume("pvc2", "test-pvc2")},
				[]api.Disk{diskWithDataStore("pvc1", true), diskWithDataStore("pvc2", true)},
				v1.ChangedBlockTrackingEnabled),
			Entry("one eligible volume lacks DataStore",
				[]v1.Volume{pvcVolume("pvc1", "test-pvc"), pvcVolume("pvc2", "test-pvc2")},
				[]api.Disk{diskWithDataStore("pvc1", true), diskWithDataStore("pvc2", false)},
				v1.ChangedBlockTrackingInitializing),
			Entry("mixed volumes, only eligible ones checked",
				[]v1.Volume{
					{Name: "container-disk", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "test:latest"}}},
					pvcVolume("pvc1", "test-pvc"),
				},
				[]api.Disk{diskWithDataStore("container-disk", false), diskWithDataStore("pvc1", true)},
				v1.ChangedBlockTrackingEnabled),
			Entry("only non-eligible volumes",
				[]v1.Volume{{Name: "container-disk", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "test:latest"}}}},
				[]api.Disk{diskWithDataStore("container-disk", false)},
				v1.ChangedBlockTrackingEnabled),
			Entry("eligible volume with no matching disk",
				[]v1.Volume{pvcVolume("pvc1", "test-pvc")},
				[]api.Disk{diskWithDataStore("different-disk", true)},
				v1.ChangedBlockTrackingInitializing),
		)

		DescribeTable("should handle edge cases",
			func(setupFunc func(), expectedState v1.ChangedBlockTrackingState) {
				setupFunc()
				Expect(cbt.CBTState(vmi.Status.ChangedBlockTracking)).To(Equal(expectedState))
			},
			Entry("domain is nil", func() {
				cbt.SetChangedBlockTrackingOnVMIFromDomain(vmi, nil)
			}, v1.ChangedBlockTrackingInitializing),
			Entry("VMI CBT status is nil", func() {
				vmi.Status.ChangedBlockTracking = nil
				domain := &api.Domain{Spec: api.DomainSpec{Devices: api.Devices{Disks: []api.Disk{diskWithDataStore("pvc1", true)}}}}
				cbt.SetChangedBlockTrackingOnVMIFromDomain(vmi, domain)
			}, v1.ChangedBlockTrackingUndefined),
		)
	})
})
