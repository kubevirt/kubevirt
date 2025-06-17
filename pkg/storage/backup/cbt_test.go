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

package backup_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/storage/backup"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("CBT", func() {
	var (
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

	Context("SyncVMChangedBlockTrackingState", func() {
		DescribeTable("No kubevirt CR ChangedBlockTrackingLabelSelectors expect no updates", func(vmiExists bool) {
			kv := &v1.KubeVirt{Spec: v1.KubeVirtSpec{Configuration: v1.KubeVirtConfiguration{}}}
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)

			vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vm = libvmi.NewVirtualMachine(vmi, libvmi.WithLabels(backup.CBTLabel))
			updatedVM := vm.DeepCopy()
			var updatedVMI *v1.VirtualMachineInstance
			if vmiExists {
				updatedVMI = vmi.DeepCopy()
			}
			backup.SyncVMChangedBlockTrackingState(updatedVM, updatedVMI, config, nsStore)
			Expect(updatedVM).To(Equal(vm))
			if vmiExists {
				Expect(updatedVMI).To(Equal(vmi))
			} else {
				Expect(updatedVMI).To(BeNil())
			}
		},
			Entry("To VM", false),
			Entry("To VM and VMI", true),
		)

		Context("VM matches VM Label Selector", func() {
			var kv *v1.KubeVirt

			BeforeEach(func() {
				labelSelector := &metav1.LabelSelector{
					MatchLabels: backup.CBTLabel,
				}
				kv = &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							ChangedBlockTrackingLabelSelectors: &v1.ChangedBlockTrackingSelectors{
								VirtualMachineLabelSelector: labelSelector,
							},
							DeveloperConfiguration: &v1.DeveloperConfiguration{
								FeatureGates: []string{featuregate.IncrementalBackupGate},
							},
						},
					},
				}
				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)

				vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
				vm = libvmi.NewVirtualMachine(vmi, libvmi.WithLabels(backup.CBTLabel))
			})

			DescribeTable("should set CBT state to ", func(vmiExists, fgDisabled bool, expectedStatus v1.ChangedBlockTrackingState) {
				if fgDisabled {
					kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
					testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
				}
				if !vmiExists {
					vmi = nil
				}
				backup.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
				Expect(vm.Status.ChangedBlockTracking).To(Equal(expectedStatus))
			},
				Entry("Initializing if VMI does not exist", false, false, v1.ChangedBlockTrackingInitializing),
				Entry("PendingRestart if VMI exists and cbtStatus is undefined", true, false, v1.ChangedBlockTrackingPendingRestart),
				Entry("IncrementalBackupFeatureGateDisabled if FG is disabled VMI does not exist", false, true, v1.ChangedBlockTrackingFGDisabled),
				Entry("IncrementalBackupFeatureGateDisabled if FG is disabled VMI exist", true, true, v1.ChangedBlockTrackingFGDisabled),
			)

			It("should set CBT state to enabled if vmi state is enabled", func() {
				vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingInitializing
				vmi.Status.ChangedBlockTracking = v1.ChangedBlockTrackingEnabled
				backup.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
				Expect(vm.Status.ChangedBlockTracking).To(Equal(v1.ChangedBlockTrackingEnabled))
			})
		})

		Context("VM namespace matches Namespace Label Selector", func() {
			var kv *v1.KubeVirt

			BeforeEach(func() {
				labelSelector := &metav1.LabelSelector{
					MatchLabels: map[string]string{"cbt-ns": "enabled"},
				}
				kv = &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							ChangedBlockTrackingLabelSelectors: &v1.ChangedBlockTrackingSelectors{
								NamespaceLabelSelector: labelSelector,
							},
							DeveloperConfiguration: &v1.DeveloperConfiguration{
								FeatureGates: []string{featuregate.IncrementalBackupGate},
							},
						},
					},
				}
				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)

				ns := &k8sv1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   k8sv1.NamespaceDefault,
						Labels: map[string]string{"cbt-ns": "enabled"},
					},
				}
				nsStore.Add(ns)

				vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
				vm = libvmi.NewVirtualMachine(vmi)
			})

			DescribeTable("should set CBT state to ", func(vmiExists, fgDisabled bool, expectedStatus v1.ChangedBlockTrackingState) {
				if fgDisabled {
					kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
					testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
				}
				if !vmiExists {
					vmi = nil
				}
				backup.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
				Expect(vm.Status.ChangedBlockTracking).To(Equal(expectedStatus))
			},
				Entry("Initializing  for VM when namespace matches if VMI does not exist", false, false, v1.ChangedBlockTrackingInitializing),
				Entry("PendingRestart for VM when namespace matches if VMI exist", true, false, v1.ChangedBlockTrackingPendingRestart),
				Entry("IncrementalBackupFeatureGateDisabled if FG is disabled VMI does not exist", false, true, v1.ChangedBlockTrackingFGDisabled),
				Entry("IncrementalBackupFeatureGateDisabled if FG is disabled VMI exist", true, true, v1.ChangedBlockTrackingFGDisabled),
			)
		})

		Context("VM no longer matches Label Selector", func() {
			BeforeEach(func() {
				vmLabelSelector := &metav1.LabelSelector{
					MatchLabels: backup.CBTLabel,
				}
				nsLabelSelector := &metav1.LabelSelector{
					MatchLabels: map[string]string{"cbt-ns": "enabled"},
				}
				kv := &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							ChangedBlockTrackingLabelSelectors: &v1.ChangedBlockTrackingSelectors{
								VirtualMachineLabelSelector: vmLabelSelector,
								NamespaceLabelSelector:      nsLabelSelector,
							},
							DeveloperConfiguration: &v1.DeveloperConfiguration{
								FeatureGates: []string{featuregate.IncrementalBackupGate},
							},
						},
					},
				}
				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
				vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
				vm = libvmi.NewVirtualMachine(vmi)
			})

			DescribeTable("if vmi does not exist", func(cbtState, expectedState v1.ChangedBlockTrackingState) {
				vm.Status.ChangedBlockTracking = cbtState
				backup.SyncVMChangedBlockTrackingState(vm, nil, config, nsStore)
				Expect(vm.Status.ChangedBlockTracking).To(Equal(expectedState))
			},
				Entry("VM CBT state undefined, should keep CBT undefined", v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingUndefined),
				Entry("VM CBT state enabled, should disable CBT", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingDisabled),
				Entry("VM CBT state initalizing, should disable CBT", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingDisabled),
				Entry("VM CBT state pendingRestart, should disable CBT", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingDisabled),
				Entry("VM CBT state fgDisabled, should disable CBT", v1.ChangedBlockTrackingFGDisabled, v1.ChangedBlockTrackingDisabled),
				Entry("VM CBT state disabled, should keep disabled CBT", v1.ChangedBlockTrackingDisabled, v1.ChangedBlockTrackingDisabled),
			)

			DescribeTable("if vmi exist", func(vmCBTState, vmiCBTState, expectedState v1.ChangedBlockTrackingState) {
				vm.Status.ChangedBlockTracking = vmCBTState
				vmi.Status.ChangedBlockTracking = vmiCBTState
				backup.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
				Expect(vm.Status.ChangedBlockTracking).To(Equal(expectedState))
			},
				Entry("with cbt initialized and VM has CBT enabled, should set PendingRestart", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingPendingRestart),
				Entry("with cbt initialized and VM has CBT pendingRestart, should set PendingRestart", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingPendingRestart),
				Entry("with cbt initialized and VM has CBT initalizing, should set PendingRestart", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingPendingRestart),
				Entry("with cbt undefined and VM has CBT enabled, should set Disabled", v1.ChangedBlockTrackingEnabled, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingDisabled),
				Entry("with cbt undefined and VM has CBT pendingRestart, should set Disabled", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingDisabled),
				Entry("with cbt undefined and VM has CBT initalizing, should set Disabled", v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingUndefined, v1.ChangedBlockTrackingDisabled),
			)
		})
	})
	Context("SetChangedBlockTrackingOnVMI", func() {
		var kv *v1.KubeVirt

		BeforeEach(func() {
			kv = &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							FeatureGates: []string{featuregate.IncrementalBackupGate},
						},
					},
				},
			}
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
			vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vm = libvmi.NewVirtualMachine(vmi)
		})
		It("IncrementalBackup featuregate disabled VMI cbt state should be undefined", func() {
			kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
			vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vm = libvmi.NewVirtualMachine(vmi, libvmi.WithLabels(backup.CBTLabel))
			backup.SetChangedBlockTrackingOnVMI(vm, vmi, config, nsStore)
			Expect(vmi.Status.ChangedBlockTracking).To(Equal(v1.ChangedBlockTrackingUndefined))
		})
		It("VM matches VM Label Selector should set VMI state to Initializing", func() {
			labelSelector := &metav1.LabelSelector{
				MatchLabels: backup.CBTLabel,
			}
			kv.Spec.Configuration.ChangedBlockTrackingLabelSelectors = &v1.ChangedBlockTrackingSelectors{
				VirtualMachineLabelSelector: labelSelector,
			}
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
			libvmi.WithLabels(backup.CBTLabel)(vm)
			backup.SetChangedBlockTrackingOnVMI(vm, vmi, config, nsStore)
			Expect(vmi.Status.ChangedBlockTracking).To(Equal(v1.ChangedBlockTrackingInitializing))
		})
		It("VM doesnt match VM Label Selector and VM CBT state exists should set VMI state to Disabled", func() {
			vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingEnabled
			backup.SetChangedBlockTrackingOnVMI(vm, vmi, config, nsStore)
			Expect(vmi.Status.ChangedBlockTracking).To(Equal(v1.ChangedBlockTrackingDisabled))
		})

		It("VM doesnt match VM Label Selector and VM CBT state doesnt exist shouldn't set VMI CBT state", func() {
			backup.SetChangedBlockTrackingOnVMI(vm, vmi, config, nsStore)
			Expect(vmi.Status.ChangedBlockTracking).To(BeEmpty())
		})
	})
	Context("UpdateVMIChangedBlockTrackingFromDomain", func() {
		It("domain has cbt enabled should update VMI state to Enabled", func() {
			vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			domain := &api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Disks: []api.Disk{{
							Source: api.DiskSource{
								DataStore: &api.DataStore{
									Type: "file",
								},
							},
						}},
					},
				},
			}
			backup.UpdateVMIChangedBlockTrackingFromDomain(vmi, domain)
			Expect(vmi.Status.ChangedBlockTracking).To(Equal(v1.ChangedBlockTrackingEnabled))
		})
		It("domain doesnt have cbt shouldnt update VMI state", func() {
			vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			domain := &api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Disks: []api.Disk{{
							Source: api.DiskSource{},
						}},
					},
				},
			}
			backup.UpdateVMIChangedBlockTrackingFromDomain(vmi, domain)
			Expect(vmi.Status.ChangedBlockTracking).To(BeEmpty())
		})
	})
})
