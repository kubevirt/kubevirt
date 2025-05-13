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
			vm = libvmi.NewVirtualMachine(vmi)
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
			BeforeEach(func() {
				labelSelector := &metav1.LabelSelector{
					MatchLabels: backup.CBTLabel,
				}
				kv := &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							ChangedBlockTrackingLabelSelectors: &v1.ChangedBlockTrackingSelectors{
								VirtualMachineLabelSelector: labelSelector,
							},
						},
					},
				}
				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)

				vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
				vm = libvmi.NewVirtualMachine(vmi, libvmi.WithLabels(backup.CBTLabel))
			})

			It("should set CBT state to Initializing if VMI does not exist", func() {
				backup.SyncVMChangedBlockTrackingState(vm, nil, config, nsStore)
				Expect(vm.Status.ChangedBlockTracking).To(Equal(v1.ChangedBlockTrackingInitializing))
			})

			It("should set CBT state to PendingRestart if VMI exists with no CBT status", func() {
				backup.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
				Expect(vm.Status.ChangedBlockTracking).To(Equal(v1.ChangedBlockTrackingPendingRestart))
			})
		})

		Context("VM namespace matches Namespace Label Selector", func() {
			BeforeEach(func() {
				labelSelector := &metav1.LabelSelector{
					MatchLabels: map[string]string{"cbt-ns": "enabled"},
				}
				kv := &v1.KubeVirt{
					Spec: v1.KubeVirtSpec{
						Configuration: v1.KubeVirtConfiguration{
							ChangedBlockTrackingLabelSelectors: &v1.ChangedBlockTrackingSelectors{
								NamespaceLabelSelector: labelSelector,
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

			It("should set CBT state to Initializing for VM when namespace matches if VMI does not exist", func() {
				backup.SyncVMChangedBlockTrackingState(vm, nil, config, nsStore)
				Expect(vm.Status.ChangedBlockTracking).To(Equal(v1.ChangedBlockTrackingInitializing))
			})
			It("should set CBT state to PendingRestart for VM when namespace matches if VMI exist", func() {
				backup.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
				Expect(vm.Status.ChangedBlockTracking).To(Equal(v1.ChangedBlockTrackingPendingRestart))
			})
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
						},
					},
				}
				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
				vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
				vm = libvmi.NewVirtualMachine(vmi)
			})

			It("should keep CBT empty if vm has empty CBT state", func() {
				vm.Status.ChangedBlockTracking = ""
				backup.SyncVMChangedBlockTrackingState(vm, nil, config, nsStore)
				Expect(vm.Status.ChangedBlockTracking).To(BeEmpty())
			})

			It("should disable CBT if VMI is nil and vm has CBT state", func() {
				vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingEnabled
				backup.SyncVMChangedBlockTrackingState(vm, nil, config, nsStore)
				Expect(vm.Status.ChangedBlockTracking).To(Equal(v1.ChangedBlockTrackingDisabled))
			})

			DescribeTable("should set CBT to", func(vmCBTState, expectedState v1.ChangedBlockTrackingState, vmiStateExists bool) {
				vm.Status.ChangedBlockTracking = vmCBTState
				if vmiStateExists {
					vmi.Status.ChangedBlockTracking = v1.ChangedBlockTrackingInitializing
				}
				backup.SyncVMChangedBlockTrackingState(vm, vmi, config, nsStore)
				Expect(vm.Status.ChangedBlockTracking).To(Equal(expectedState))
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
