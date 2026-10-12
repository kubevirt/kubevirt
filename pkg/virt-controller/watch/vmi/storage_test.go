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

package vmi

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegaTypes "github.com/onsi/gomega/types"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/libvmi"

	kvcontroller "kubevirt.io/kubevirt/pkg/controller"
	controllertesting "kubevirt.io/kubevirt/pkg/controller/testing"
	"kubevirt.io/kubevirt/pkg/pointer"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
	watchtesting "kubevirt.io/kubevirt/pkg/virt-controller/watch/testing"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
)

var _ = Describe("VirtualMachineInstance watcher storage", func() {
	var config *virtconfig.ClusterConfig

	var controller *Controller
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue[string]
	var virtClientset *kubevirtfake.Clientset
	var kubeClient *fake.Clientset
	var storageClassStore, storageProfileStore cache.Store
	var kvStore cache.Store

	expectMatchingPodCreation := func(vmi *virtv1.VirtualMachineInstance, matchers ...gomegaTypes.GomegaMatcher) {
		pods, err := kubeClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", virtv1.CreatedByLabel, string(vmi.UID))})
		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).To(HaveLen(1))
		Expect(pods.Items[0].Labels[virtv1.CreatedByLabel]).To(Equal(string(vmi.UID)))
		if len(matchers) > 0 {
			Expect(&pods.Items[0]).To(SatisfyAny(matchers...))
		}
	}

	expectPodDoesNotExist := func(namespace, name string) {
		_, err := kubeClient.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
	}

	expectPodExists := func(namespace, name string) {
		pod, err := kubeClient.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(pod).ToNot(BeNil())
	}

	expectVMIBeInPhase := func(namespace, name string, phase virtv1.VirtualMachineInstancePhase) {
		updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVmi.Status.Phase).To(BeEquivalentTo(phase))
	}

	expectVMIWithMatcherConditions := func(namespace, name string, matchers ...gomegaTypes.GomegaMatcher) {
		updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVmi.Status.Conditions).To(SatisfyAll(matchers...))
	}

	expectVMIDataVolumeReadyCondition := func(namespace, name string, expectedStatus k8sv1.ConditionStatus) {
		expectVMIWithMatcherConditions(namespace, name,
			ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Type":   BeEquivalentTo(virtv1.VirtualMachineInstanceDataVolumesReady),
					"Status": BeEquivalentTo(expectedStatus)},
				),
			),
		)
	}

	expectVirtualMachinePendingState := func(namespace, name string) {
		expectVMIBeInPhase(namespace, name, virtv1.Pending)
		expectVMIWithMatcherConditions(namespace, name, ContainElement(MatchFields(IgnoreExtras,
			Fields{"Type": Equal(virtv1.VirtualMachineInstanceReady)})))
	}

	expectHotplugPod := func() {
		pods, err := kubeClient.CoreV1().Pods(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).To(
			ContainElement(
				WithTransform(func(pod k8sv1.Pod) string {
					return pod.GenerateName
				}, Equal("hp-volume-")),
			),
		)
	}

	sanityExecute := func() {}

	BeforeEach(func() {
		virtClient := kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		virtClientset = kubevirtfake.NewSimpleClientset()

		vmiInformer, _ := testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachineInstance{}, kvcontroller.GetVMIInformerIndexers())

		vmInformer, _ := testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachine{}, kvcontroller.GetVirtualMachineInformerIndexers())
		podInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})
		dataVolumeInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		storageProfileInformer, _ := testutils.NewFakeInformerFor(&cdiv1.StorageProfile{})
		storageProfileStore = storageProfileInformer.GetStore()
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		kubevirtFakeConfig := &virtv1.KubeVirtConfiguration{
			DeveloperConfiguration: &virtv1.DeveloperConfiguration{
				MinimumClusterTSCFrequency: new(int64(12345)),
			},
		}

		config, _, kvStore = testutils.NewFakeClusterConfigUsingKVConfig(kubevirtFakeConfig)
		pvcInformer, _ := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		migrationInformer, _ := testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachineInstanceMigration{}, kvcontroller.GetVirtualMachineInstanceMigrationInformerIndexers())
		storageClassInformer, _ := testutils.NewFakeInformerFor(&storagev1.StorageClass{})
		storageClassStore = storageClassInformer.GetStore()
		cdiInformer, _ := testutils.NewFakeInformerFor(&cdiv1.CDIConfig{})
		cdiConfigInformer, _ := testutils.NewFakeInformerFor(&cdiv1.CDIConfig{})
		kubeVirtInformer, _ := testutils.NewFakeInformerFor(&virtv1.KubeVirt{})
		rqInformer, _ := testutils.NewFakeInformerFor(&k8sv1.ResourceQuota{})
		nsInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Namespace{})
		var qemuGid int64 = 107

		stubNetStatusUpdate := func(vmi *virtv1.VirtualMachineInstance, _ *k8sv1.Pod) error {
			vmi.Status.Interfaces = append(vmi.Status.Interfaces, virtv1.VirtualMachineInstanceNetworkInterface{Name: "stubNetStatusUpdate"})
			return nil
		}

		mockQueue = testutils.NewMockWorkQueue(testutils.NewFrozenClockRateLimitingQueue("virt-controller-vmi"))
		controller, _ = NewController(mockQueue,
			services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", pvcInformer.GetStore(), virtClient, config, qemuGid, "g", rqInformer.GetStore(), nsInformer.GetStore()),
			vmiInformer,
			vmInformer,
			podInformer,
			pvcInformer,
			migrationInformer,
			storageClassInformer,
			recorder,
			virtClient,
			dataVolumeInformer,
			storageProfileInformer,
			cdiInformer,
			cdiConfigInformer,
			kubeVirtInformer,
			config,
			topology.NewTopologyHinter(&cache.FakeCustomStore{}, &cache.FakeCustomStore{}, config),
			stubNetworkAnnotationsGenerator{},
			stubStorageAnnotationsGenerator{},
			stubNetStatusUpdate,
			validateNetVMISpecStub(),
			stubMigrationEvaluator{result: k8sv1.ConditionUnknown},
			&fakeAllocator{},
			[]string{},
			[]string{},
		)

		sanityExecute = func() {
			controllertesting.SanityExecute(controller, []cache.Store{
				controller.vmiIndexer, controller.vmStore, controller.podIndexer, controller.pvcIndexer,
				controller.dataVolumeIndexer, controller.cdiStore, controller.cdiConfigStore,
				storageProfileInformer.GetStore(),
				storageClassStore,
			}, Default)
		}

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstance(k8sv1.NamespaceDefault).Return(
			virtClientset.KubevirtV1().VirtualMachineInstances(k8sv1.NamespaceDefault),
		).AnyTimes()
		kubeClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
	})

	AfterEach(func() {
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
	})

	addVirtualMachine := func(vmi *virtv1.VirtualMachineInstance) {
		controller.vmiIndexer.Add(vmi)
		key, err := kvcontroller.KeyFunc(vmi)
		Expect(err).To(Not(HaveOccurred()))
		mockQueue.Add(key)
		_, err = virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	addPod := func(pod *k8sv1.Pod) {
		Expect(controller.podIndexer.Add(pod)).To(Succeed())
		_, err := kubeClient.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	addDataVolumePVC := func(dvPVC *k8sv1.PersistentVolumeClaim) {
		Expect(controller.pvcIndexer.Add(dvPVC)).To(Succeed())
		_, err := kubeClient.CoreV1().PersistentVolumeClaims(dvPVC.Namespace).Create(context.Background(), dvPVC, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	addDataVolume := func(dv *cdiv1.DataVolume) {
		Expect(controller.dataVolumeIndexer.Add(dv)).To(Succeed())
	}

	Context("On valid VirtualMachineInstance given with DataVolume source", func() {

		dvVolumeSource := virtv1.VolumeSource{
			DataVolume: &virtv1.DataVolumeSource{
				Name: "test1",
			},
		}
		It("should create a corresponding Pod on VMI creation when DataVolume is ready", func() {
			vmi := newPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})

			dvPVC := newPvc(vmi.Namespace, "test1")
			dataVolume := newDv(vmi.Namespace, "test1", cdiv1.Succeeded)

			addVirtualMachine(vmi)
			addDataVolumePVC(dvPVC)
			addDataVolume(dataVolume)

			sanityExecute()
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
			expectMatchingPodCreation(vmi)
			expectVMIDataVolumeReadyCondition(vmi.Namespace, vmi.Name, k8sv1.ConditionTrue)
		})

		It("should create a doppleganger Pod on VMI creation when DataVolume is in WaitForFirstConsumer state", func() {
			vmi := newPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})

			dataVolume := newDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)

			dvPVC := newPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, new(true))
			dvPVC.Status.Phase = k8sv1.ClaimPending
			addDataVolumePVC(dvPVC)
			addVirtualMachine(vmi)
			addDataVolume(dataVolume)

			IsPodWithoutVmPayload := WithTransform(
				func(pod *k8sv1.Pod) string {
					for _, c := range pod.Spec.Containers {
						if c.Name == "compute" {
							return strings.Join(append(c.Command, c.Args...), " ")
						}
					}

					return ""
				},
				Equal("/bin/bash -c echo bound PVCs"),
			)

			sanityExecute()
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)

			expectMatchingPodCreation(vmi, IsPodWithoutVmPayload)
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name,
				ContainElements(
					MatchFields(IgnoreExtras, Fields{"Type": BeEquivalentTo(virtv1.VirtualMachineInstanceProvisioning)}),
					MatchFields(IgnoreExtras, Fields{"Type": BeEquivalentTo(virtv1.VirtualMachineInstanceDataVolumesReady), "Status": BeEquivalentTo(k8sv1.ConditionFalse)}),
				),
			)
		})

		It("should not delete a doppleganger Pod on VMI creation when DataVolume is in WaitForFirstConsumer state", func() {
			vmi := newPendingVirtualMachine("testvmi")
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Annotations[virtv1.EphemeralProvisioningObject] = "true"

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})

			pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
				Name: "test1",
			})

			dataVolume := newDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)

			dvPVC := newPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, new(true))
			dvPVC.Status.Phase = k8sv1.ClaimPending

			addDataVolumePVC(dvPVC)
			addActivePods(vmi, pod.UID, "")
			addVirtualMachine(vmi)
			addPod(pod)
			addDataVolume(dataVolume)
			sanityExecute()
			expectVMIBeInPhase(vmi.Namespace, vmi.Name, virtv1.Pending)
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(virtv1.VirtualMachineInstanceCondition{
				Type:   virtv1.VirtualMachineInstanceProvisioning,
				Status: k8sv1.ConditionTrue,
			}))
		})

		It("should delete a doppleganger Pod on VMI creation when DataVolume is no longer in WaitForFirstConsumer state", func() {
			vmi := newPendingVirtualMachine("testvmi")
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Annotations[virtv1.EphemeralProvisioningObject] = "true"

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})
			pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
				Name: "test1",
			})

			dataVolume := newDv(vmi.Namespace, "test1", cdiv1.Succeeded)
			dvPVC := newPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, new(true))
			addDataVolumePVC(dvPVC)
			addVirtualMachine(vmi)
			addPod(pod)
			addActivePods(vmi, pod.UID, "")
			addDataVolume(dataVolume)

			sanityExecute()
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulDeletePodReason)
			expectVMIDataVolumeReadyCondition(vmi.Namespace, vmi.Name, k8sv1.ConditionTrue)
			expectPodDoesNotExist(pod.Namespace, pod.Name)
		})

		It("should only get WFFC pods that are associated with current VMI", func() {
			vmi := newPendingVirtualMachine("testvmi")
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Annotations[virtv1.EphemeralProvisioningObject] = "true"
			Expect(controller.podIndexer.Add(pod)).To(Succeed())

			// Non owned pod that shouldn't be found by waitForFirstConsumerTemporaryPods
			nonOwnedPod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unowned-test-pod",
					Namespace: vmi.Namespace,
					Annotations: map[string]string{
						virtv1.EphemeralProvisioningObject: "true",
					},
				},
				Status: k8sv1.PodStatus{
					Phase: k8sv1.PodRunning,
				},
			}
			Expect(controller.podIndexer.Add(nonOwnedPod)).To(Succeed())

			res, err := controller.waitForFirstConsumerTemporaryPods(vmi, pod)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
		})

		DescribeTable("VMI should handle doppleganger Pod status while DV is in WaitForFirstConsumer phase",
			func(phase k8sv1.PodPhase, conditions []k8sv1.PodCondition, expectedPhase virtv1.VirtualMachineInstancePhase) {
				vmi := newPendingVirtualMachine("testvmi")
				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				pod.Annotations[virtv1.EphemeralProvisioningObject] = "true"
				pod.Status.Phase = phase
				pod.Status.Conditions = conditions

				vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
					Name:         "test1",
					VolumeSource: dvVolumeSource,
				})

				pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
					Name: "test1",
				})

				dataVolume := newDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)
				dvPVC := newPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, new(true))
				dvPVC.Status.Phase = k8sv1.ClaimBound
				Expect(controller.pvcIndexer.Add(dvPVC)).To(Succeed())

				addVirtualMachine(vmi)
				addPod(pod)
				addDataVolumePVC(dvPVC)
				addActivePods(vmi, pod.UID, "")
				addDataVolume(dataVolume)

				sanityExecute()
				expectVMIBeInPhase(vmi.Namespace, vmi.Name, expectedPhase)
			},
			Entry("fail if pod in failed state", k8sv1.PodFailed, nil, virtv1.Failed),
			Entry("do nothing if pod succeed", k8sv1.PodSucceeded, nil, virtv1.Pending),
			//The PodReasonUnschedulable is a transient condition. It can clear up if more resources are added to the cluster
			Entry("do nothing if pod Pending Unschedulable",
				k8sv1.PodPending,
				[]k8sv1.PodCondition{{
					Type:   k8sv1.PodScheduled,
					Status: k8sv1.ConditionFalse,
					Reason: k8sv1.PodReasonUnschedulable}}, virtv1.Pending),
		)

		It("should not create a corresponding Pod on VMI creation when DataVolume is pending", func() {
			vmi := newPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})

			dataVolume := newDv(vmi.Namespace, "test1", cdiv1.Pending)
			dvPVC := newPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, new(true))
			dvPVC.Status.Phase = k8sv1.ClaimPending

			addVirtualMachine(vmi)
			addDataVolumePVC(dvPVC)
			addDataVolume(dataVolume)
			sanityExecute()
			expectVMIDataVolumeReadyCondition(vmi.Namespace, vmi.Name, k8sv1.ConditionFalse)
		})

		When("backend storage is pending", func() {
			var vmi *virtv1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = newPendingVirtualMachine("testvmi")
				vmi.Spec.Domain.Firmware = &virtv1.Firmware{
					Bootloader: &virtv1.Bootloader{
						EFI: &virtv1.EFI{
							Persistent: new(true),
						},
					},
				}
				addVirtualMachine(vmi)
			})

			DescribeTable("should not create a corresponding Pod on VMI creation with a storage class in Immediate mode", func(mode *storagev1.VolumeBindingMode) {
				sc := &storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testsc123",
					},
					VolumeBindingMode: mode,
				}
				Expect(storageClassStore.Add(sc)).To(Succeed())

				pvc := newPvc(vmi.Namespace, "persistent-state-for-"+vmi.Name+"-12345")
				pvc.ObjectMeta.Labels = map[string]string{"persistent-state-for": vmi.Name}
				pvc.Status.Phase = k8sv1.ClaimPending
				pvc.Spec.StorageClassName = new("testsc123")
				pvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteMany}
				addDataVolumePVC(pvc)

				sanityExecute()
				expectVirtualMachinePendingState(vmi.Namespace, vmi.Name)
			},
				Entry("using explicit Immediate mode", pointer.P(storagev1.VolumeBindingImmediate)),
				Entry("using nil mode", nil))

			It("should create a corresponding Pod on VMI creation with a storage class in WaitForFirstConsumer mode", func() {
				sc := &storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testsc456",
					},
					VolumeBindingMode: pointer.P(storagev1.VolumeBindingWaitForFirstConsumer),
				}
				Expect(storageClassStore.Add(sc)).To(Succeed())

				pvc := newPvc(vmi.Namespace, "persistent-state-for-"+vmi.Name+"-67890")
				pvc.ObjectMeta.Labels = map[string]string{"persistent-state-for": vmi.Name}
				pvc.Status.Phase = k8sv1.ClaimPending
				pvc.Spec.StorageClassName = new("testsc456")
				pvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteMany}
				addDataVolumePVC(pvc)

				sanityExecute()
				testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
				expectMatchingPodCreation(vmi)
			})

		})

		When("backend storage no RWX support", func() {
			var vmi *virtv1.VirtualMachineInstance

			DescribeTable("should create a corresponding RWO PVC on VMI creation", func(alterVMI func(*virtv1.VirtualMachineInstance)) {
				vmi = newPendingVirtualMachine("testvmi")
				alterVMI(vmi)
				addVirtualMachine(vmi)

				sc := &storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testsc123",
					},
					VolumeBindingMode: pointer.P(storagev1.VolumeBindingImmediate),
				}
				sp := &cdiv1.StorageProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name: sc.Name,
					},
					Spec: cdiv1.StorageProfileSpec{
						ClaimPropertySets: []cdiv1.ClaimPropertySet{
							{AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce}, VolumeMode: pointer.P(k8sv1.PersistentVolumeFilesystem)},
						},
					},
					Status: cdiv1.StorageProfileStatus{
						StorageClass: new(sc.Name),
						ClaimPropertySets: []cdiv1.ClaimPropertySet{
							{AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce}, VolumeMode: pointer.P(k8sv1.PersistentVolumeFilesystem)},
						},
					},
				}
				Expect(storageClassStore.Add(sc)).To(Succeed())
				Expect(storageProfileStore.Add(sp)).To(Succeed())
				kvCR := testutils.GetFakeKubeVirtClusterConfig(kvStore)
				kvCR.Spec.Configuration.VMStateStorageClass = "testsc123"
				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvCR)

				sanityExecute()
				expectVirtualMachinePendingState(vmi.Namespace, vmi.Name)
				pvcs, err := kubeClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: "persistent-state-for=" + vmi.Name,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(pvcs.Items).To(HaveLen(1))
				pvc := pvcs.Items[0]
				Expect(pvc.Spec.StorageClassName).To(HaveValue(Equal(sc.Name)))
				Expect(pvc.Spec.AccessModes).To(HaveLen(1))
				Expect(pvc.Spec.AccessModes[0]).To(Equal(k8sv1.ReadWriteOnce))
			},
				Entry("with EFI", func(vmi *virtv1.VirtualMachineInstance) {
					vmi.Spec.Domain.Firmware = &virtv1.Firmware{
						Bootloader: &virtv1.Bootloader{
							EFI: &virtv1.EFI{
								Persistent: new(true),
							},
						},
					}
				}),
				Entry("with CBT", func(vmi *virtv1.VirtualMachineInstance) {
					cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingInitializing)
				}),
			)
			It("should delete backend PVC if cbt is no longer enabled on VMI creation", func() {
				vmi = newPendingVirtualMachine("testvmi")
				cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingDisabled)
				addVirtualMachine(vmi)
				pvc := newPvc(vmi.Namespace, "vm-state-pvc")
				pvc.Labels = map[string]string{"persistent-state-for": vmi.Name}
				addDataVolumePVC(pvc)
				pvcs, err := kubeClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: "persistent-state-for=" + vmi.Name,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(pvcs.Items).To(HaveLen(1))

				sanityExecute()
				testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
				pvcs, err = kubeClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: "persistent-state-for=" + vmi.Name,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(pvcs.Items).To(BeEmpty())
			})
		})
	})

	Context("On valid VirtualMachineInstance given with PVC source, ownedRef of DataVolume", func() {

		pvcVolumeSource := virtv1.VolumeSource{
			PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: "test1",
			}},
		}
		It("should create a corresponding Pod on VMI creation when DataVolume is ready", func() {
			vmi := newPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			dvPVC := newPvcWithOwner(vmi.Namespace, "test1", "test1", new(true))
			dataVolume := newDv(vmi.Namespace, "test1", cdiv1.Succeeded)

			addVirtualMachine(vmi)
			addDataVolumePVC(dvPVC)
			addDataVolume(dataVolume)

			sanityExecute()
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
			expectMatchingPodCreation(vmi)
			expectVMIDataVolumeReadyCondition(vmi.Namespace, vmi.Name, k8sv1.ConditionTrue)
		})

		It("should create a doppleganger Pod on VMI creation when DataVolume is in WaitForFirstConsumer state", func() {
			vmi := newPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			dvPVC := newPvcWithOwner(vmi.Namespace, "test1", "test1", new(true))
			dvPVC.Status.Phase = k8sv1.ClaimPending
			dataVolume := newDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)

			addVirtualMachine(vmi)
			addDataVolumePVC(dvPVC)
			addDataVolume(dataVolume)

			sanityExecute()
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
			IsPodWithoutVmPayload := WithTransform(
				func(pod *k8sv1.Pod) string {
					for _, c := range pod.Spec.Containers {
						if c.Name == "compute" {
							return strings.Join(append(c.Command, c.Args...), " ")
						}
					}

					return ""
				},
				Equal("/bin/bash -c echo bound PVCs"))
			expectMatchingPodCreation(vmi, IsPodWithoutVmPayload)
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
				Fields{"Type": Equal(virtv1.VirtualMachineInstanceProvisioning)})),
			)
		})

		It("should not delete a doppleganger Pod on VMI creation when DataVolume is in WaitForFirstConsumer state", func() {
			vmi := newPendingVirtualMachine("testvmi")
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Annotations[virtv1.EphemeralProvisioningObject] = "true"

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
				Name: "test1",
			})

			dvPVC := newPvcWithOwner(vmi.Namespace, "test1", "test1", new(true))
			dvPVC.Status.Phase = k8sv1.ClaimPending
			dataVolume := newDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)

			addVirtualMachine(vmi)
			addDataVolumePVC(dvPVC)
			addPod(pod)
			addActivePods(vmi, pod.UID, "")
			addDataVolume(dataVolume)

			sanityExecute()
			expectVMIBeInPhase(vmi.Namespace, vmi.Name, virtv1.Pending)
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(virtv1.VirtualMachineInstanceCondition{
				Type:   virtv1.VirtualMachineInstanceProvisioning,
				Status: k8sv1.ConditionTrue,
			}))
		})

		It("should delete a doppleganger Pod on VMI creation when DataVolume is no longer in WaitForFirstConsumer state", func() {
			vmi := newPendingVirtualMachine("testvmi")
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Annotations[virtv1.EphemeralProvisioningObject] = "true"

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
				Name: "test1",
			})

			dataVolume := newDv(vmi.Namespace, "test1", cdiv1.Succeeded)
			dvPVC := newPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, new(true))
			dvPVC.Status.Phase = k8sv1.ClaimBound
			addVirtualMachine(vmi)
			addDataVolumePVC(dvPVC)
			addPod(pod)
			addActivePods(vmi, pod.UID, "")
			addDataVolume(dataVolume)

			sanityExecute()
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulDeletePodReason)
			expectPodDoesNotExist(pod.Namespace, pod.Name)
			expectVMIDataVolumeReadyCondition(vmi.Namespace, vmi.Name, k8sv1.ConditionTrue)

		})

		It("should not create a corresponding Pod on VMI creation when DataVolume is pending", func() {
			vmi := newPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			dvPVC := newPvcWithOwner(vmi.Namespace, "test1", "test1", new(true))
			dvPVC.Status.Phase = k8sv1.ClaimPending
			dataVolume := newDv(vmi.Namespace, "test1", cdiv1.Pending)

			addVirtualMachine(vmi)
			addDataVolumePVC(dvPVC)
			addDataVolume(dataVolume)

			sanityExecute()
			expectVMIDataVolumeReadyCondition(vmi.Namespace, vmi.Name, k8sv1.ConditionFalse)
		})

		It("should create a corresponding Pod on VMI creation when PVC is not controlled by a DataVolume", func() {
			vmi := newPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			pvc := newPvc(vmi.Namespace, "test1")

			addVirtualMachine(vmi)
			addDataVolumePVC(pvc)

			sanityExecute()
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
			expectMatchingPodCreation(vmi)
		})

	})

	Context("hotplug volume", func() {
		It("Should find vmi, from virt-launcher pod", func() {
			vmi := newPendingVirtualMachine("testvmi")
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			controllerRef := metav1.GetControllerOf(pod)
			addVirtualMachine(vmi)

			result := controller.resolveControllerRef(k8sv1.NamespaceDefault, controllerRef)
			Expect(equality.Semantic.DeepEqual(result, vmi)).To(BeTrue(), "result: %v, should equal %v", result, vmi)
		})

		It("Should find vmi, from attachment pod", func() {
			vmi := newPendingVirtualMachine("testvmi")
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			attachmentPod := newPodForVirtlauncher(pod, "hp-test", "abcd", k8sv1.PodRunning)
			controllerRef := metav1.GetControllerOf(attachmentPod)
			addVirtualMachine(vmi)
			addPod(pod)

			result := controller.resolveControllerRef(k8sv1.NamespaceDefault, controllerRef)
			Expect(equality.Semantic.DeepEqual(result, vmi)).To(BeTrue(), "result: %v, should equal %v", result, vmi)
		})

		It("DeleteAllAttachmentPods should return success if no virt launcher pod exists", func() {
			vmi := newPendingVirtualMachine("testvmi")
			addVirtualMachine(vmi)
			err := controller.deleteAllAttachmentPods(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("DeleteAllAttachmentPods should return success if there are no attachment pods", func() {
			vmi := newPendingVirtualMachine("testvmi")
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addVirtualMachine(vmi)
			addPod(virtlauncherPod)
			err := controller.deleteAllAttachmentPods(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("DeleteAllAttachmentPods should return success if there are attachment pods", func() {
			vmi := newPendingVirtualMachine("testvmi")
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			attachmentPod1 := newPodForVirtlauncher(virtlauncherPod, "pod1", "abcd", k8sv1.PodRunning)
			attachmentPod2 := newPodForVirtlauncher(virtlauncherPod, "pod2", "efgh", k8sv1.PodRunning)
			attachmentPod3 := newPodForVirtlauncher(virtlauncherPod, "pod3", "ilmn", k8sv1.PodRunning)
			addVirtualMachine(vmi)
			addPod(virtlauncherPod)
			addPod(attachmentPod1)
			addPod(attachmentPod2)
			addPod(attachmentPod3)
			err := controller.deleteAllAttachmentPods(vmi)
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulDeletePodReason)
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulDeletePodReason)
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulDeletePodReason)
			Expect(err).ToNot(HaveOccurred())
			expectPodDoesNotExist(attachmentPod3.Namespace, attachmentPod3.Name)
			expectPodDoesNotExist(attachmentPod1.Namespace, attachmentPod1.Name)
			expectPodDoesNotExist(attachmentPod2.Namespace, attachmentPod2.Name)
		})

		It("CreateAttachmentPodTemplate should return error if volume is not DV or PVC", func() {
			vmi := newPendingVirtualMachine("testvmi")
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addVirtualMachine(vmi)
			addPod(virtlauncherPod)
			invalidVolume := &virtv1.Volume{
				Name: "fake",
				VolumeSource: virtv1.VolumeSource{
					ConfigMap: &virtv1.ConfigMapVolumeSource{},
				},
			}
			pod, err := controller.createAttachmentPodTemplate(vmi, virtlauncherPod, []*virtv1.Volume{invalidVolume})
			Expect(pod).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("CreateAttachmentPodTemplate should return error if volume has PVC that doesn't exist", func() {
			vmi := newPendingVirtualMachine("testvmi")
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addVirtualMachine(vmi)
			addPod(virtlauncherPod)
			nopvcVolume := &virtv1.Volume{
				Name: "nopvc",
				VolumeSource: virtv1.VolumeSource{
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "noclaim",
						},
					},
				},
			}
			pod, err := controller.createAttachmentPodTemplate(vmi, virtlauncherPod, []*virtv1.Volume{nopvcVolume})
			Expect(pod).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("CreateAttachmentPodTemplate should return nil pod if only one DV exists and owning PVC doesn't exist", func() {
			vmi := newPendingVirtualMachine("testvmi")
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pvc := newHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimPending)

			addVirtualMachine(vmi)
			addPod(virtlauncherPod)
			addDataVolumePVC(pvc)
			volume := &virtv1.Volume{
				Name: "test-pvc-volume",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "test-dv",
					},
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-dv",
						},
					},
				},
			}
			pod, err := controller.createAttachmentPodTemplate(vmi, virtlauncherPod, []*virtv1.Volume{volume})
			Expect(pod).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("CreateAttachmentPodTemplate should set status to pending if DV owning PVC is not ready", func() {
			vmi := newPendingVirtualMachine("testvmi")
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pvc := newHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimPending)
			addDataVolumePVC(pvc)
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dv",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Pending,
				},
			}
			addDataVolume(dv)
			addVirtualMachine(vmi)
			addPod(virtlauncherPod)
			volume := &virtv1.Volume{
				Name: "test-pvc-volume",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "test-dv",
					},
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-dv",
						},
					},
				},
			}
			pod, err := controller.createAttachmentPodTemplate(vmi, virtlauncherPod, []*virtv1.Volume{volume})
			Expect(pod).To(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("CreateAttachmentPodTemplate should update status to bound if DV owning PVC is bound but not ready", func() {
			vmi := newPendingVirtualMachine("testvmi")
			addVolumeStatuses(vmi, virtv1.VolumeStatus{
				Name:          "test-pvc-volume",
				HotplugVolume: &virtv1.HotplugVolumeStatus{},
				Phase:         virtv1.VolumePending,
				Message:       "some technical reason",
				Reason:        kvcontroller.PVCNotReadyReason,
			})
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pvc := newHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimBound)
			addDataVolumePVC(pvc)
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dv",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Pending,
				},
			}
			addDataVolume(dv)
			addVirtualMachine(vmi)
			addPod(virtlauncherPod)
			volume := &virtv1.Volume{
				Name: "test-pvc-volume",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "test-dv",
					},
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-dv",
						},
					},
				},
			}
			pod, err := controller.createAttachmentPodTemplate(vmi, virtlauncherPod, []*virtv1.Volume{volume})
			Expect(pod).To(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("CreateAttachmentPodTemplate should create a pod template if DV of owning PVC is ready", func() {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.SelinuxContext = "system_u:system_r:container_file_t:s0:c1,c2"
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pvc := newHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimBound)
			addDataVolumePVC(pvc)
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dv",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Succeeded,
				},
			}
			addDataVolume(dv)
			addVirtualMachine(vmi)
			addPod(virtlauncherPod)
			volume := &virtv1.Volume{
				Name: "test-pvc-volume",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "test-dv",
					},
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-dv",
						},
					},
				},
			}
			pod, err := controller.createAttachmentPodTemplate(vmi, virtlauncherPod, []*virtv1.Volume{volume})
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.GenerateName).To(Equal("hp-volume-"))
			found := false
			for _, podVolume := range pod.Spec.Volumes {
				if volume.Name == podVolume.Name {
					found = true
				}
			}
			Expect(found).To(BeTrue())
		})

		makePodWithVirtlauncher := func(virtlauncherPod *k8sv1.Pod, indexes ...int) []*k8sv1.Pod {
			res := make([]*k8sv1.Pod, 0)
			pod := newPodForVirtlauncher(virtlauncherPod, "test-pod", "abcd", k8sv1.PodRunning)
			pod.Spec.Volumes = make([]k8sv1.Volume, 0)
			pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
				Name: "emptydir",
				VolumeSource: k8sv1.VolumeSource{
					EmptyDir: &k8sv1.EmptyDirVolumeSource{},
				},
			})
			pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
				Name: "token",
				VolumeSource: k8sv1.VolumeSource{
					Secret: &k8sv1.SecretVolumeSource{},
				},
			})
			for _, index := range indexes {
				pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
					Name: fmt.Sprintf("volume%d", index),
					VolumeSource: k8sv1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("claim%d", index),
						},
					},
				})
				pod.Status.ContainerStatuses = []k8sv1.ContainerStatus{
					{
						Ready: true,
					},
				}
				res = append(res, pod)
			}
			return res
		}

		makePods := func(indexes ...int) []*k8sv1.Pod {
			vmi := newPendingVirtualMachine("testvmi")
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			return makePodWithVirtlauncher(virtlauncherPod, indexes...)
		}

		makePodsWithDeletion := func(indexes ...int) []*k8sv1.Pod {
			attachmentPods := makePods(indexes...)
			for _, pod := range attachmentPods {
				pod.DeletionTimestamp = new(metav1.Now())
			}
			return attachmentPods
		}

		makeVolumes := func(indexes ...int) []*virtv1.Volume {
			res := make([]*virtv1.Volume, 0)
			for _, index := range indexes {
				res = append(res, &virtv1.Volume{
					Name: fmt.Sprintf("volume%d", index),
					VolumeSource: virtv1.VolumeSource{
						PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: fmt.Sprintf("claim%d", index),
							},
						},
					},
				})
			}
			return res
		}

		makeVolumesWithMemoryDump := func(total int, indexes ...int) []*virtv1.Volume {
			res := make([]*virtv1.Volume, 0)
			for i := range total {
				memoryDump := false
				for _, index := range indexes {
					if i == index {
						memoryDump = true
						res = append(res, &virtv1.Volume{
							Name: fmt.Sprintf("volume%d", index),
							VolumeSource: virtv1.VolumeSource{
								MemoryDump: testutils.NewFakeMemoryDumpSource(fmt.Sprintf("claim%d", i)),
							},
						})
					}
				}
				if !memoryDump {
					res = append(res, &virtv1.Volume{
						Name: fmt.Sprintf("volume%d", i),
						VolumeSource: virtv1.VolumeSource{
							PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
								PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("claim%d", i),
								},
							},
						},
					})
				}
			}
			return res
		}

		makeK8sVolumes := func(indexes ...int) []k8sv1.Volume {
			res := make([]k8sv1.Volume, 0)
			for _, index := range indexes {
				res = append(res, k8sv1.Volume{
					Name: fmt.Sprintf("volume%d", index),
					VolumeSource: k8sv1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("claim%d", index),
						},
					},
				})
			}
			return res
		}

		makeVolumeStatuses := func() []virtv1.VolumeStatus {
			volumeStatuses := make([]virtv1.VolumeStatus, 0)
			volumeStatuses = append(volumeStatuses, virtv1.VolumeStatus{
				Name: "test-volume",
				HotplugVolume: &virtv1.HotplugVolumeStatus{
					AttachPodName: "attach-pod",
					AttachPodUID:  "abcd",
				},
			})
			volumeStatuses = append(volumeStatuses, virtv1.VolumeStatus{
				Name:          "test-volume2",
				HotplugVolume: &virtv1.HotplugVolumeStatus{},
			})
			return volumeStatuses
		}

		DescribeTable("should determine if status contains the volume including a pod", func(volumeStatuses []virtv1.VolumeStatus, volume *virtv1.Volume, expected bool) {
			res := controller.volumeStatusContainsVolumeAndPod(volumeStatuses, volume)
			Expect(res).To(Equal(expected))
		},
			Entry("should return true if the volume with pod is in the status", makeVolumeStatuses(), &virtv1.Volume{
				Name: "test-volume",
			}, true),
			Entry("should return false if the volume with pod is in the status", makeVolumeStatuses(), &virtv1.Volume{
				Name: "test-volume2",
			}, false),
		)

		preparePVC := func(indexes ...int) {
			for _, index := range indexes {
				pvc := newHotplugPVC(fmt.Sprintf("claim%d", index), k8sv1.NamespaceDefault, k8sv1.ClaimBound)
				dv := &cdiv1.DataVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("claim%d", index),
						Namespace: k8sv1.NamespaceDefault,
					},
					Status: cdiv1.DataVolumeStatus{
						Phase: cdiv1.Succeeded,
					},
				}

				addDataVolume(dv)
				addDataVolumePVC(pvc)
			}
		}

		DescribeTable("handleHotplugVolumes should properly react to input", func(hotplugVolumes []*virtv1.Volume, hotplugAttachmentPods []*k8sv1.Pod, createPodReaction func(*k8sv1.Pod, ...int), pvcFunc func(...int), pvcIndexes []int, orgStatus []virtv1.VolumeStatus, expectedEvent string, expectedErr common.SyncError) {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.VolumeStatus = orgStatus
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pvcFunc(pvcIndexes...)
			datavolumes := []*cdiv1.DataVolume{}
			for _, volume := range hotplugVolumes {
				datavolumes = append(datavolumes, &cdiv1.DataVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name: volume.Name,
					},
					Status: cdiv1.DataVolumeStatus{
						Phase: cdiv1.Succeeded,
					},
				})
			}
			addVirtualMachine(vmi)
			addPod(virtlauncherPod)
			if createPodReaction != nil {
				createPodReaction(virtlauncherPod, pvcIndexes...)
			}
			syncError := controller.handleHotplugVolumes(hotplugVolumes, hotplugAttachmentPods, vmi, virtlauncherPod, datavolumes)
			if expectedErr != nil {
				Expect(syncError).To(BeEquivalentTo(expectedErr))
			} else {
				Expect(syncError).ToNot(HaveOccurred())
			}
			if expectedEvent != "" {
				testutils.ExpectEvent(recorder, expectedEvent)
			}
		},
			Entry("when volumes and pods match, the status should remain the same",
				makeVolumes(1),
				makePods(1),
				nil,
				preparePVC,
				[]int{1},
				makeVolumeStatuses(),
				"",
				nil),
		)

		DescribeTable("needsHandleHotplug", func(hotplugVolumes []*virtv1.Volume, hotplugAttachmentPods []*k8sv1.Pod, expected bool) {
			res := needsHandleHotplug(hotplugVolumes, hotplugAttachmentPods)
			Expect(res).To(Equal(expected))
		},
			Entry("should return false if volumes and attachmentpods are empty", makeVolumes(), makePods(), false),
			Entry("should return false if volumes and attachmentpods match", makeVolumes(1), makePods(1), false),
			Entry("should return true if volumes > attachmentpods", makeVolumes(1, 2), makePods(1), true),
			Entry("should return true if volumes < attachmentpods", makeVolumes(1), makePods(1, 2), true),
			Entry("should return true if len(volumes) == len(attachmentpods), but contents differ", makeVolumes(1, 3), makePods(1, 2), true),
		)

		DescribeTable("virtlauncherAttachmentPods", func(podCount int) {
			vmi := newPendingVirtualMachine("testvmi")
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			for i := range podCount {
				attachmentPod := newPodForVirtlauncher(virtlauncherPod, fmt.Sprintf("test-pod%d", i), fmt.Sprintf("abcd%d", i), k8sv1.PodRunning)
				Expect(controller.podIndexer.Add(attachmentPod)).To(Succeed())
			}
			// Add some non owned pods.
			for i := range 5 {
				pod := &k8sv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("unowned-test-pod%d", i),
						Namespace: vmi.Namespace,
					},
					Status: k8sv1.PodStatus{
						Phase: k8sv1.PodRunning,
					},
				}
				Expect(controller.podIndexer.Add(pod)).To(Succeed())
			}
			otherPod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "owning-test-pod",
					Namespace: vmi.Namespace,
					UID:       "something",
				},
				Status: k8sv1.PodStatus{
					Phase: k8sv1.PodRunning,
				},
			}
			Expect(controller.podIndexer.Add(otherPod)).To(Succeed())
			for i := 10; i < 10+podCount; i++ {
				attachmentPod := newPodForVirtlauncher(otherPod, fmt.Sprintf("test-pod%d", i), fmt.Sprintf("abcde%d", i), k8sv1.PodRunning)
				Expect(controller.podIndexer.Add(attachmentPod)).To(Succeed())
			}

			res, err := kvcontroller.AttachmentPods(virtlauncherPod, controller.podIndexer)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(podCount))
		},
			Entry("should return number (0) of pods passed in", 0),
			Entry("should return number (1) of pods passed in", 1),
			Entry("should return number (2) of pods passed in", 2),
		)

		DescribeTable("getHotplugVolumes", func(virtlauncherVolumes []k8sv1.Volume, vmiVolumes []*virtv1.Volume, expectedIndexes ...int) {
			vmi := newPendingVirtualMachine("testvmi")
			for _, volume := range vmiVolumes {
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, *volume)
			}
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			virtlauncherPod.Spec.Volumes = virtlauncherVolumes
			res := storagetypes.GetHotplugVolumes(vmi, virtlauncherPod)
			Expect(res).To(HaveLen(len(expectedIndexes)))
			for _, index := range expectedIndexes {
				found := false
				for _, volume := range res {
					if volume.Name == fmt.Sprintf("volume%d", index) {
						found = true
					}
				}
				Expect(found).To(BeTrue())
			}
		},
			Entry("should return no volumes if vmi and virtlauncher have no volumes", makeK8sVolumes(), makeVolumes()),
			Entry("should return a volume if vmi has one more than virtlauncher", makeK8sVolumes(), makeVolumes(1), 1),
			Entry("should return a volume if vmi has one more than virtlauncher, with matching volumes", makeK8sVolumes(1, 3), makeVolumes(1, 2, 3), 2),
			Entry("should return multiple volumes if vmi has multiple more than virtlauncher, with matching volumes", makeK8sVolumes(1, 3), makeVolumes(1, 2, 3, 4, 5), 2, 4, 5),
			Entry("should return a memory dump volume if vmi has memory dump volume not on virtlauncher", makeK8sVolumes(0, 2), makeVolumesWithMemoryDump(3, 1), 1),
		)

		truncateSprintf := func(str string, args ...any) string {
			n := strings.Count(str, "%d")
			return fmt.Sprintf(str, args[:n]...)
		}

		makeVolumeStatusesForUpdateWithMessage := func(podName, podUID string, phase virtv1.VolumePhase, message, reason string, indexes ...int) []virtv1.VolumeStatus {
			res := make([]virtv1.VolumeStatus, 0)
			for _, index := range indexes {
				res = append(res, virtv1.VolumeStatus{
					Name: fmt.Sprintf("volume%d", index),
					HotplugVolume: &virtv1.HotplugVolumeStatus{
						AttachPodName: truncateSprintf(podName, index),
						AttachPodUID:  types.UID(truncateSprintf(podUID, index)),
					},
					Phase:   phase,
					Message: truncateSprintf(message, index, index),
					Reason:  reason,
					PersistentVolumeClaimInfo: &virtv1.PersistentVolumeClaimInfo{
						ClaimName: fmt.Sprintf("claim%d", index),
						AccessModes: []k8sv1.PersistentVolumeAccessMode{
							k8sv1.ReadOnlyMany,
						},
						FilesystemOverhead: pointer.P(storagetypes.DefaultFSOverhead),
					},
				})
			}
			return res
		}

		makeVolumeStatusesForUpdateWithMemoryDump := func(dumpIndex int, indexes ...int) []virtv1.VolumeStatus {
			res := makeVolumeStatusesForUpdateWithMessage("", "", virtv1.VolumeBound, "PVC is in phase Bound", kvcontroller.PVCNotReadyReason, indexes...)
			res[dumpIndex].MemoryDumpVolume = &virtv1.DomainMemoryDumpInfo{
				ClaimName: fmt.Sprintf("volume%d", dumpIndex),
			}
			return res
		}

		makeVolumeStatusesForUpdate := func(indexes ...int) []virtv1.VolumeStatus {
			return makeVolumeStatusesForUpdateWithMessage("test-pod", "abcd", virtv1.HotplugVolumeAttachedToNode, "Created hotplug attachment pod test-pod, for volume volume%d", kvcontroller.SuccessfulCreatePodReason, indexes...)
		}

		makeDetachVolumeStatus := func(indexes ...int) []virtv1.VolumeStatus {
			return makeVolumeStatusesForUpdateWithMessage("test-pod", "abcd", virtv1.HotplugVolumeDetaching, "Deleted hotplug attachment pod test-pod, for volume%d", kvcontroller.SuccessfulDeletePodReason, indexes...)
		}

		DescribeTable("updateVolumeStatus", func(oldStatus []virtv1.VolumeStatus, specVolumes []*virtv1.Volume, podIndexes []int, pvcIndexes []int, expectedStatus []virtv1.VolumeStatus, expectedEvents []string) {
			vmi := newPendingVirtualMachine("testvmi")
			volumes := make([]virtv1.Volume, 0)
			for _, volume := range specVolumes {
				volumes = append(volumes, *volume)
			}
			vmi.Spec.Volumes = volumes
			vmi.Status.VolumeStatus = oldStatus
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			attachmentPods := makePodWithVirtlauncher(virtlauncherPod, podIndexes...)

			for _, pod := range attachmentPods {
				pod.DeletionTimestamp = new(metav1.Now())
				Expect(controller.podIndexer.Add(pod)).To(Succeed())
			}
			for _, pvcIndex := range pvcIndexes {
				pvc := newHotplugPVC(fmt.Sprintf("claim%d", pvcIndex), k8sv1.NamespaceDefault, k8sv1.ClaimBound)
				Expect(controller.pvcIndexer.Add(pvc)).To(Succeed())
			}

			err := controller.updateVolumeStatus(vmi, virtlauncherPod)
			testutils.ExpectEvents(recorder, expectedEvents...)
			Expect(err).ToNot(HaveOccurred())
			Expect(equality.Semantic.DeepEqual(expectedStatus, vmi.Status.VolumeStatus)).To(BeTrue(), "status: %v, expected: %v", vmi.Status.VolumeStatus, expectedStatus)
		},
			Entry("should not update volume status, if no volumes changed",
				makeVolumeStatusesForUpdate(),
				makeVolumes(),
				[]int{},
				[]int{},
				makeVolumeStatusesForUpdate(),
				[]string{}),
			Entry("should update volume status, if a new volume is added, and pod exists",
				makeVolumeStatusesForUpdate(),
				makeVolumes(0),
				[]int{0},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("", "", virtv1.VolumeBound, "PVC is in phase Bound", kvcontroller.PVCNotReadyReason, 0),
				[]string{}),
			Entry("should update volume status, if a new volume is added, and pod does not exist",
				makeVolumeStatusesForUpdate(),
				makeVolumes(0),
				[]int{},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("", "", virtv1.VolumeBound, "PVC is in phase Bound", kvcontroller.PVCNotReadyReason, 0),
				[]string{}),
			Entry("should update volume status, if a existing volume is changed, and pod does not exist",
				makeVolumeStatusesForUpdateWithMessage("", "", virtv1.VolumePending, "PVC is in phase Pending", kvcontroller.PVCNotReadyReason, 0),
				makeVolumes(0),
				[]int{},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("", "", virtv1.VolumeBound, "PVC is in phase Bound", kvcontroller.PVCNotReadyReason, 0),
				[]string{}),
			Entry("should keep status, if volume removed and if pod still exists",
				makeVolumeStatusesForUpdate(0),
				makeVolumes(),
				[]int{0},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("test-pod", "abcd", virtv1.HotplugVolumeDetaching, "Deleted hotplug attachment pod test-pod, for volume volume0", kvcontroller.SuccessfulDeletePodReason, 0),
				[]string{kvcontroller.SuccessfulDeletePodReason}),
			Entry("should update volume status, if volume gets stuck in Detaching state due to it being removed and re-added",
				makeDetachVolumeStatus(),
				makeVolumes(0),
				[]int{0},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("", "", virtv1.VolumeBound, "PVC is in phase Bound", kvcontroller.PVCNotReadyReason, 0),
				[]string{}),
			Entry("should keep status and not update phase if volume is still Ready",
				makeVolumeStatusesForUpdateWithMessage("", "", virtv1.VolumeReady, "", "", 0),
				makeVolumes(),
				[]int{0},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("test-pod", "abcd", virtv1.VolumeReady, "", "", 0),
				[]string{}),
			Entry("should keep status and not update phase if volume is still mounted",
				makeVolumeStatusesForUpdateWithMessage("", "", virtv1.HotplugVolumeMounted, "", "", 0),
				makeVolumes(),
				[]int{0},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("test-pod", "abcd", virtv1.HotplugVolumeMounted, "", "", 0),
				[]string{}),
			Entry("should remove volume status, if volume is removed and pod is gone",
				makeVolumeStatusesForUpdate(0),
				makeVolumes(),
				[]int{},
				[]int{},
				makeVolumeStatusesForUpdate(),
				[]string{}),
			Entry("should update volume status with memory dump, if a new memory dump volume is added",
				makeVolumeStatusesForUpdate(),
				makeVolumesWithMemoryDump(1, 0),
				[]int{0},
				[]int{0},
				makeVolumeStatusesForUpdateWithMemoryDump(0, 0),
				[]string{}),
		)

		DescribeTable("Should properly calculate if it needs to handle hotplug volumes", func(hotplugVolumes []*virtv1.Volume, attachmentPods []*k8sv1.Pod, match gomegaTypes.GomegaMatcher) {
			Expect(needsHandleHotplug(hotplugVolumes, attachmentPods)).To(match)
		},
			Entry("nil volumes, nil attachmentPods", nil, nil, BeFalse()),
			Entry("empty volumes, empty attachmentPods", []*virtv1.Volume{}, []*k8sv1.Pod{}, BeFalse()),
			Entry("single volume, empty attachmentPods", []*virtv1.Volume{
				{
					Name: "test",
				},
			}, []*k8sv1.Pod{}, BeTrue()),
			Entry("no volume, single attachmentPod", []*virtv1.Volume{}, makePods(0), BeTrue()),
			Entry("matching volume, single attachmentPod", []*virtv1.Volume{
				{
					Name: "volume0",
				},
			}, makePods(0), BeFalse()),
			Entry("mismatched volume, single attachmentPod", []*virtv1.Volume{
				{
					Name: "invalid",
				},
			}, makePods(0), BeTrue()),
			Entry("matching volume, multiple attachmentPods", []*virtv1.Volume{
				{
					Name: "volume0",
				},
			}, []*k8sv1.Pod{makePods(0)[0], makePods(1)[0]}, BeTrue()),
		)

		DescribeTable("Should find active and old pods", func(hotplugVolumes []*virtv1.Volume, attachmentPods []*k8sv1.Pod, expectedActive *k8sv1.Pod, expectedOld []*k8sv1.Pod) {
			active, old := getActiveAndOldAttachmentPods(hotplugVolumes, attachmentPods)
			Expect(active).To(Equal(expectedActive))
			Expect(old).To(ContainElements(expectedOld))
		},
			Entry("nil volumes, nil attachmentPods", nil, nil, nil, nil),
			Entry("empty volumes, empty attachmentPods", []*virtv1.Volume{}, []*k8sv1.Pod{}, nil, []*k8sv1.Pod{}),
			Entry("matching volume, single attachmentPod", []*virtv1.Volume{
				{
					Name: "volume0",
				},
			}, makePods(0), makePods(0)[0], []*k8sv1.Pod{}),
			Entry("matching volume, nil attachmentPods since marked for deletion", []*virtv1.Volume{
				{
					Name: "volume0",
				},
			}, makePodsWithDeletion(0), nil, []*k8sv1.Pod{}),
			Entry("matching volume, multiple attachmentPods, first pod matches", []*virtv1.Volume{
				{
					Name: "volume0",
				},
			}, []*k8sv1.Pod{makePods(0)[0], makePods(1)[0]}, makePods(0)[0], makePods(1)),
			Entry("matching volume, multiple attachmentPods, second pod matches", []*virtv1.Volume{
				{
					Name: "volume1",
				},
			}, []*k8sv1.Pod{makePods(0)[0], makePods(1)[0]}, makePods(1)[0], makePods(0)),
		)

		It("Should get default filesystem overhead if there are multiple CDI instances", func() {
			cdi := cdiv1.CDI{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testCDI1",
				},
			}
			cdi2 := cdiv1.CDI{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testCDI2",
				},
			}
			// We add multiple CDI instances to trigger the error
			Expect(controller.cdiStore.Add(&cdi)).To(Succeed())
			Expect(controller.cdiStore.Add(&cdi2)).To(Succeed())

			fsOverhead, err := controller.getFilesystemOverhead(&k8sv1.PersistentVolumeClaim{})
			Expect(err).ToNot(HaveOccurred())
			Expect(fsOverhead).To(Equal(storagetypes.DefaultFSOverhead))
		})

		It("Should get default filesystem overhead if there is no CDI available", func() {
			fsOverhead, err := controller.getFilesystemOverhead(&k8sv1.PersistentVolumeClaim{})
			Expect(err).ToNot(HaveOccurred())
			Expect(fsOverhead).To(Equal(storagetypes.DefaultFSOverhead))
		})

		It("Should fail to get filesystem overhead if there's no valid CDI config available", func() {
			cdi := cdiv1.CDI{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testCDI",
				},
			}

			Expect(controller.cdiStore.Add(&cdi)).To(Succeed())

			fsOverhead, err := controller.getFilesystemOverhead(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Failed to find CDIConfig but CDI exists"))
			Expect(fsOverhead).To(Equal(virtv1.Percent("0")))
		})

		DescribeTable("getFilesystemOverhead", func(volumeMode k8sv1.PersistentVolumeMode, scName string, expectedOverhead virtv1.Percent) {
			cdi := cdiv1.CDI{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testCDI",
				},
			}
			cfg := cdiv1.CDIConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "config",
				},
				Status: cdiv1.CDIConfigStatus{
					FilesystemOverhead: &cdiv1.FilesystemOverhead{
						Global:       cdiv1.Percent("0.5"),
						StorageClass: map[string]cdiv1.Percent{"default": cdiv1.Percent("0.8")},
					},
				},
			}
			pvc := &k8sv1.PersistentVolumeClaim{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PersistentVolumeClaim",
					APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test"},
				Spec: k8sv1.PersistentVolumeClaimSpec{},
			}

			pvc.Spec.VolumeMode = &volumeMode
			if scName != "" {
				pvc.Spec.StorageClassName = &scName
			}

			Expect(controller.cdiStore.Add(&cdi)).To(Succeed())
			Expect(controller.cdiConfigStore.Add(&cfg)).To(Succeed())

			fsOverhead, err := controller.getFilesystemOverhead(pvc)
			Expect(err).ToNot(HaveOccurred())
			Expect(fsOverhead).To(Equal(expectedOverhead))
		},
			Entry("should not get any fs overhead if the PVC has volumeMode block", k8sv1.PersistentVolumeBlock, "default", virtv1.Percent("0")),
			Entry("should return global fs overhead if there's no storageClassName", k8sv1.PersistentVolumeFilesystem, "", virtv1.Percent("0.5")),
			Entry("should return global fs overhead if the storageClassName is invalid", k8sv1.PersistentVolumeFilesystem, "nonValid", virtv1.Percent("0.5")),
			Entry("should return the appropiate overhead when using a valid storageClassName", k8sv1.PersistentVolumeFilesystem, "default", virtv1.Percent("0.8")),
		)

		It("Should set error for utility volume with block mode PVC", func() {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Spec.UtilityVolumes = []virtv1.UtilityVolume{
				{
					Name: "utility-vol",
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "block-pvc",
					},
				},
			}

			blockMode := k8sv1.PersistentVolumeBlock
			blockPVC := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "block-pvc",
					Namespace: k8sv1.NamespaceDefault,
				},
				Spec: k8sv1.PersistentVolumeClaimSpec{
					VolumeMode: &blockMode,
					AccessModes: []k8sv1.PersistentVolumeAccessMode{
						k8sv1.ReadWriteOnce,
					},
				},
				Status: k8sv1.PersistentVolumeClaimStatus{
					Phase: k8sv1.ClaimBound,
				},
			}

			Expect(controller.pvcIndexer.Add(blockPVC)).To(Succeed())

			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			err := controller.updateVolumeStatus(vmi, virtlauncherPod)
			Expect(err).ToNot(HaveOccurred())

			Expect(vmi.Status.VolumeStatus).To(HaveLen(1))
			volumeStatus := vmi.Status.VolumeStatus[0]
			Expect(volumeStatus.Name).To(Equal("utility-vol"))
			Expect(volumeStatus.Phase).To(Equal(virtv1.VolumePending))
			Expect(volumeStatus.Reason).To(Equal(kvcontroller.PVCNotReadyReason))
			Expect(volumeStatus.Message).To(ContainSubstring("must be filesystem mode, not block mode"))
		})

		It("Should accept utility volume with filesystem mode PVC", func() {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Spec.UtilityVolumes = []virtv1.UtilityVolume{
				{
					Name: "utility-vol",
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "filesystem-pvc",
					},
				},
			}

			filesystemPVC := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "filesystem-pvc",
					Namespace: k8sv1.NamespaceDefault,
				},
				Spec: k8sv1.PersistentVolumeClaimSpec{
					AccessModes: []k8sv1.PersistentVolumeAccessMode{
						k8sv1.ReadWriteOnce,
					},
				},
				Status: k8sv1.PersistentVolumeClaimStatus{
					Phase: k8sv1.ClaimBound,
				},
			}

			Expect(controller.pvcIndexer.Add(filesystemPVC)).To(Succeed())

			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			err := controller.updateVolumeStatus(vmi, virtlauncherPod)
			Expect(err).ToNot(HaveOccurred())

			Expect(vmi.Status.VolumeStatus).To(HaveLen(1))
			volumeStatus := vmi.Status.VolumeStatus[0]
			Expect(volumeStatus.Name).To(Equal("utility-vol"))
			Expect(volumeStatus.Phase).To(Equal(virtv1.VolumeBound))
			Expect(volumeStatus.Reason).To(Equal(kvcontroller.PVCNotReadyReason))
			Expect(volumeStatus.Message).To(Equal("PVC is in phase Bound"))
			Expect(volumeStatus.HotplugVolume).ToNot(BeNil())
			Expect(volumeStatus.PersistentVolumeClaimInfo).ToNot(BeNil())
			Expect(volumeStatus.PersistentVolumeClaimInfo.ClaimName).To(Equal("filesystem-pvc"))
		})

		// TODO drop test with legacy code https://github.com/kubevirt/kubevirt/issues/17369
		It("Should normalize legacy backend storage volume name on upgrade", func() {
			vmi := newPendingVirtualMachine("testvmi")
			legacyPVCName := "persistent-state-for-testvmi"
			// Simulate a VMI whose status was written by an older controller:
			// the volume status entry uses the PVC name as the Name field.
			vmi.Status.VolumeStatus = []virtv1.VolumeStatus{
				{
					Name: legacyPVCName,
					PersistentVolumeClaimInfo: &virtv1.PersistentVolumeClaimInfo{
						ClaimName: legacyPVCName,
					},
				},
			}

			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      legacyPVCName,
					Namespace: k8sv1.NamespaceDefault,
					Labels:    map[string]string{backendstorage.PVCPrefix: "testvmi"},
				},
			}
			Expect(controller.pvcIndexer.Add(pvc)).To(Succeed())

			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			Expect(controller.updateVolumeStatus(vmi, virtlauncherPod)).To(Succeed())

			// The legacy entry must be carried forward under the new static name,
			// so that UpdateVolumeStatus can update it in-place on the same reconcile.
			Expect(vmi.Status.VolumeStatus).To(HaveLen(1))
			Expect(vmi.Status.VolumeStatus[0].Name).To(Equal(backendstorage.VolumeName))
			Expect(vmi.Status.VolumeStatus[0].PersistentVolumeClaimInfo.ClaimName).To(Equal(legacyPVCName))
		})

		Context("isUtilityVolumeWithBlockPVC", func() {
			It("should return true for a utility volume with block mode PVC", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Spec.UtilityVolumes = []virtv1.UtilityVolume{
					{
						Name: "utility-vol",
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "block-pvc",
						},
					},
				}

				blockMode := k8sv1.PersistentVolumeBlock
				blockPVC := &k8sv1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "block-pvc",
						Namespace: k8sv1.NamespaceDefault,
					},
					Spec: k8sv1.PersistentVolumeClaimSpec{
						VolumeMode: &blockMode,
						AccessModes: []k8sv1.PersistentVolumeAccessMode{
							k8sv1.ReadWriteOnce,
						},
					},
					Status: k8sv1.PersistentVolumeClaimStatus{
						Phase: k8sv1.ClaimBound,
					},
				}

				Expect(controller.pvcIndexer.Add(blockPVC)).To(Succeed())

				utilityVolume := &virtv1.Volume{
					Name: "utility-vol",
					VolumeSource: virtv1.VolumeSource{
						PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "block-pvc",
							},
							Hotpluggable: true,
						},
					},
				}

				isBlock, err := controller.isUtilityVolumeWithBlockPVC(vmi, utilityVolume)
				Expect(err).ToNot(HaveOccurred())
				Expect(isBlock).To(BeTrue())
			})

			It("should return false for a non-utility volume with block mode PVC", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Spec.UtilityVolumes = []virtv1.UtilityVolume{
					{
						Name: "utility-vol",
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "block-pvc",
						},
					},
				}

				blockMode := k8sv1.PersistentVolumeBlock
				blockPVC := &k8sv1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "block-pvc",
						Namespace: k8sv1.NamespaceDefault,
					},
					Spec: k8sv1.PersistentVolumeClaimSpec{
						VolumeMode: &blockMode,
						AccessModes: []k8sv1.PersistentVolumeAccessMode{
							k8sv1.ReadWriteOnce,
						},
					},
					Status: k8sv1.PersistentVolumeClaimStatus{
						Phase: k8sv1.ClaimBound,
					},
				}

				Expect(controller.pvcIndexer.Add(blockPVC)).To(Succeed())

				nonUtilityVolume := &virtv1.Volume{
					Name: "non-utility-vol",
					VolumeSource: virtv1.VolumeSource{
						PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "block-pvc",
							},
							Hotpluggable: true,
						},
					},
				}

				isBlock, err := controller.isUtilityVolumeWithBlockPVC(vmi, nonUtilityVolume)
				Expect(err).ToNot(HaveOccurred())
				Expect(isBlock).To(BeFalse())
			})

			It("should return an error when PVC does not exist", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Spec.UtilityVolumes = []virtv1.UtilityVolume{
					{
						Name: "utility-vol",
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "missing-pvc",
						},
					},
				}

				missingPVCVolume := &virtv1.Volume{
					Name: "utility-vol",
					VolumeSource: virtv1.VolumeSource{
						PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "missing-pvc",
							},
							Hotpluggable: true,
						},
					},
				}

				_, err := controller.isUtilityVolumeWithBlockPVC(vmi, missingPVCVolume)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("does not exist"))
			})

			It("should return false for a utility volume with filesystem PVC", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Spec.UtilityVolumes = []virtv1.UtilityVolume{
					{
						Name: "utility-vol",
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "filesystem-pvc",
						},
					},
				}

				filesystemPVC := &k8sv1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "filesystem-pvc",
						Namespace: k8sv1.NamespaceDefault,
					},
					Spec: k8sv1.PersistentVolumeClaimSpec{
						AccessModes: []k8sv1.PersistentVolumeAccessMode{
							k8sv1.ReadWriteOnce,
						},
					},
					Status: k8sv1.PersistentVolumeClaimStatus{
						Phase: k8sv1.ClaimBound,
					},
				}

				Expect(controller.pvcIndexer.Add(filesystemPVC)).To(Succeed())

				filesystemVolume := &virtv1.Volume{
					Name: "utility-vol",
					VolumeSource: virtv1.VolumeSource{
						PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "filesystem-pvc",
							},
							Hotpluggable: true,
						},
					},
				}

				isBlock, err := controller.isUtilityVolumeWithBlockPVC(vmi, filesystemVolume)
				Expect(err).ToNot(HaveOccurred())
				Expect(isBlock).To(BeFalse())
			})
		})

		It("Should not create attachment pod for utility volume with block mode PVC", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodConditionMissingReason)
			vmi.Spec.UtilityVolumes = []virtv1.UtilityVolume{
				{
					Name: "utility-vol",
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "block-pvc",
					},
				},
			}
			vmi.Status.Phase = virtv1.Running
			addActivePods(vmi, "virt-launch-uid", "")

			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

			blockMode := k8sv1.PersistentVolumeBlock
			blockPVC := &k8sv1.PersistentVolumeClaim{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PersistentVolumeClaim",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmi.Namespace,
					Name:      "block-pvc",
				},
				Spec: k8sv1.PersistentVolumeClaimSpec{
					VolumeMode: &blockMode,
					AccessModes: []k8sv1.PersistentVolumeAccessMode{
						k8sv1.ReadWriteOnce,
					},
				},
				Status: k8sv1.PersistentVolumeClaimStatus{
					Phase: k8sv1.ClaimBound,
				},
			}
			addDataVolumePVC(blockPVC)
			addVirtualMachine(vmi)
			addPod(virtlauncherPod)

			sanityExecute()

			podList, err := kubeClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			attachmentPodCount := 0
			for _, pod := range podList.Items {
				if strings.HasPrefix(pod.Name, "hp-volume-") {
					attachmentPodCount++
				}
			}
			Expect(attachmentPodCount).To(Equal(0))
		})

		It("Should properly create attachmentpod, if correct volume and disk are added", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodConditionMissingReason)
			volumes := make([]virtv1.Volume, 0)
			volumes = append(volumes, virtv1.Volume{
				Name: "existing",
				VolumeSource: virtv1.VolumeSource{
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "existing",
						},
					},
				},
			})
			volumes = append(volumes, virtv1.Volume{
				Name: "hotplug",
				VolumeSource: virtv1.VolumeSource{
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "hotplug",
						},
					},
				},
			})
			vmi.Spec.Volumes = volumes
			disks := make([]virtv1.Disk, 0)
			disks = append(disks, virtv1.Disk{
				Name: "existing",
				DiskDevice: virtv1.DiskDevice{
					Disk: &virtv1.DiskTarget{
						Bus: virtv1.DiskBusVirtio,
					},
				},
			})
			disks = append(disks, virtv1.Disk{
				Name: "hotplug",
				DiskDevice: virtv1.DiskDevice{
					Disk: &virtv1.DiskTarget{
						Bus: virtv1.DiskBusSATA,
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = disks
			vmi.Status.Phase = virtv1.Running
			addVolumeStatuses(vmi, virtv1.VolumeStatus{
				Name:   "existing",
				Target: "",
			})
			addActivePods(vmi, "virt-launch-uid", "")
			vmi.Status.SelinuxContext = "system_u:system_r:container_file_t:s0:c1,c2"
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

			existingPVC := &k8sv1.PersistentVolumeClaim{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PersistentVolumeClaim",
					APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmi.Namespace,
					Name:      "existing"},
			}
			hpPVC := &k8sv1.PersistentVolumeClaim{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PersistentVolumeClaim",
					APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmi.Namespace,
					Name:      "hotplug"},
				Status: k8sv1.PersistentVolumeClaimStatus{
					Phase: k8sv1.ClaimBound,
				},
			}
			virtlauncherPod.Spec.Volumes = append(virtlauncherPod.Spec.Volumes, k8sv1.Volume{
				Name: "existing",
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "existing",
					},
				},
			})
			addDataVolumePVC(existingPVC)
			addDataVolumePVC(hpPVC)
			addVirtualMachine(vmi)
			addPod(virtlauncherPod)

			sanityExecute()
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
			expectHotplugPod()
			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVmi.Status.Phase).To(Equal(virtv1.Running))
			Expect(updatedVmi.Status.VolumeStatus).To(BeEquivalentTo([]virtv1.VolumeStatus{
				{
					Name:   "existing",
					Target: "",
					PersistentVolumeClaimInfo: &virtv1.PersistentVolumeClaimInfo{
						ClaimName:          "existing",
						FilesystemOverhead: pointer.P(virtv1.Percent("0.055")),
					},
				},
				{
					Name:    "hotplug",
					Target:  "",
					Phase:   virtv1.VolumeBound,
					Reason:  "PVCNotReady",
					Message: "PVC is in phase Bound",
					PersistentVolumeClaimInfo: &virtv1.PersistentVolumeClaimInfo{
						ClaimName:          "hotplug",
						FilesystemOverhead: pointer.P(virtv1.Percent("0.055")),
					},
					HotplugVolume: &virtv1.HotplugVolumeStatus{},
				},
			}))
		})

		It("Should properly delete attachment, if volume and disk are removed", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodConditionMissingReason)
			volumes := make([]virtv1.Volume, 0)
			volumes = append(volumes, virtv1.Volume{
				Name: "existing",
				VolumeSource: virtv1.VolumeSource{
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "existing",
					}},
				},
			})
			vmi.Spec.Volumes = volumes
			disks := make([]virtv1.Disk, 0)
			disks = append(disks, virtv1.Disk{
				Name: "existing",
				DiskDevice: virtv1.DiskDevice{
					Disk: &virtv1.DiskTarget{
						Bus: virtv1.DiskBusVirtio,
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = disks
			vmi.Status.Phase = virtv1.Running
			addVolumeStatuses(vmi,
				virtv1.VolumeStatus{
					Name:   "existing",
					Target: "",
				}, virtv1.VolumeStatus{
					Name:   "hotplug",
					Target: "",
					HotplugVolume: &virtv1.HotplugVolumeStatus{
						AttachPodName: "hp-volume-hotplug",
						AttachPodUID:  "abcd",
					},
				},
			)
			addActivePods(vmi, "virt-launch-uid", "")
			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

			existingPVC := &k8sv1.PersistentVolumeClaim{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PersistentVolumeClaim",
					APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmi.Namespace,
					Name:      "existing"},
			}
			hpPVC := &k8sv1.PersistentVolumeClaim{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PersistentVolumeClaim",
					APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmi.Namespace,
					Name:      "hotplug"},
				Status: k8sv1.PersistentVolumeClaimStatus{
					Phase: k8sv1.ClaimBound,
				},
			}
			virtlauncherPod.Spec.Volumes = append(virtlauncherPod.Spec.Volumes, k8sv1.Volume{
				Name: "existing",
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "existing",
					},
				},
			})
			hpPod := newPodForVirtlauncher(virtlauncherPod, "hp-volume-hotplug", "abcd", k8sv1.PodRunning)
			hpPod.Spec.Volumes = append(hpPod.Spec.Volumes, k8sv1.Volume{
				Name: "hotplug",
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{},
				},
			})
			addDataVolumePVC(existingPVC)
			addDataVolumePVC(hpPVC)
			addVirtualMachine(vmi)
			addPod(virtlauncherPod)
			addPod(hpPod)
			sanityExecute()

			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulDeletePodReason)
			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVmi.Status.Phase).To(Equal(virtv1.Running))
			Expect(updatedVmi.Status.VolumeStatus).To(BeEquivalentTo([]virtv1.VolumeStatus{
				{
					Name:   "existing",
					Target: "",
					PersistentVolumeClaimInfo: &virtv1.PersistentVolumeClaimInfo{
						ClaimName:          "existing",
						FilesystemOverhead: pointer.P(virtv1.Percent("0.055"))},
				},
				{
					Name:   "hotplug",
					Target: "",
					Phase:  virtv1.HotplugVolumeDetaching,
					HotplugVolume: &virtv1.HotplugVolumeStatus{
						AttachPodName: "hp-volume-hotplug",
						AttachPodUID:  "abcd",
					},
				},
			}))
			expectPodDoesNotExist(hpPod.Namespace, hpPod.Name)
		})

		DescribeTable("Should requeue if pod is not created before", func(pod *k8sv1.Pod, creationTimeAdd, requeueTime time.Duration, expected bool) {
			var pods []*k8sv1.Pod
			if pod != nil {
				pod.CreationTimestamp.Time = time.Now().Add(creationTimeAdd)
				pods = append(pods, pod)
			}
			requeue, threshold := controller.requeueAfter(pods, requeueTime)
			Expect(requeue).To(Equal(expected))
			if expected {
				Expect(threshold).To(BeNumerically(">", 0))
			}
		},
			Entry("nil pod, should not requeue", nil, time.Duration(0), time.Second, false),
			Entry("created 5 seconds ago, with 1 second requeu time, should not requeue", &k8sv1.Pod{}, -5*time.Second, time.Second, false),
			Entry("created now, with second requeue time, should requeue in second", &k8sv1.Pod{}, time.Duration(0), time.Second, true),
			Entry("created in 2 seconds, with second requeue time, should requeue now", &k8sv1.Pod{}, 2*time.Second, time.Second, true),
		)

		DescribeTable("should clean up attachment pods, based on the number of volumes", func(readyVolumes []*virtv1.Volume, currentPod *k8sv1.Pod, oldPods []*k8sv1.Pod, expectDelete bool) {
			// TODO should disappear
			shouldExpectPodDeletion := func(pod *k8sv1.Pod) {
				// Expect pod deletion
				kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
					update, ok := action.(testing.DeleteAction)
					Expect(ok).To(BeTrue())
					Expect(pod.Namespace).To(Equal(update.GetNamespace()))
					return true, nil, nil
				})
			}

			vmi := watchtesting.NewRunningVirtualMachine("testvmi", &k8sv1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testnode",
				},
			})
			if currentPod == nil && len(oldPods) > 0 {
				currentPod = oldPods[0]
			}
			if expectDelete {
				if currentPod != nil {
					shouldExpectPodDeletion(currentPod)
				}
				if len(oldPods) > 0 {
					for i := range oldPods {
						if i > 0 {
							shouldExpectPodDeletion(oldPods[i])
						}
					}
				}
			}
			err := controller.cleanupAttachmentPods(currentPod, oldPods, vmi, len(readyVolumes))
			Expect(err).ToNot(HaveOccurred())
			if expectDelete {
				eventReasons := []string{}
				if len(oldPods) > 0 {
					for range oldPods {
						eventReasons = append(eventReasons, kvcontroller.SuccessfulDeletePodReason)
					}
				}
				By(fmt.Sprintf("expected events %v", eventReasons))
				testutils.ExpectEvents(recorder, eventReasons...)
			}
		},
			Entry("no volumes, no pods", nil, nil, nil, false),
			Entry("no volumes, pending current pod", nil, &k8sv1.Pod{
				Status: k8sv1.PodStatus{
					Phase: k8sv1.PodPending,
				},
			}, nil, true),
			Entry("no volumes, running current pod", nil, &k8sv1.Pod{
				Status: k8sv1.PodStatus{
					Phase: k8sv1.PodRunning,
				},
			}, nil, true),
			Entry("a volume, running current pod", []*virtv1.Volume{{}}, &k8sv1.Pod{
				Status: k8sv1.PodStatus{
					Phase: k8sv1.PodRunning,
				},
			}, nil, false),
			Entry("no volumes, no current pod, pending oldpod", nil, nil, []*k8sv1.Pod{
				{
					Status: k8sv1.PodStatus{
						Phase: k8sv1.PodPending,
					},
				},
			}, true),
			Entry("no volumes, no current pod, multiple oldpods", nil, nil, []*k8sv1.Pod{
				{
					Status: k8sv1.PodStatus{
						Phase: k8sv1.PodRunning,
					},
				},
				{
					Status: k8sv1.PodStatus{
						Phase: k8sv1.PodRunning,
					},
				},
			}, true),
		)

		DescribeTable("should handle old running pod kept as fallback based on volume status", func(volName string, volPhase virtv1.VolumePhase, inSpec bool, expectDelete bool) {
			vmi := watchtesting.NewRunningVirtualMachine("testvmi", &k8sv1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testnode",
				},
			})
			vmi.Status.VolumeStatus = []virtv1.VolumeStatus{
				{
					Name:  volName,
					Phase: volPhase,
					HotplugVolume: &virtv1.HotplugVolumeStatus{
						AttachPodName: "old-pod",
						AttachPodUID:  "old-uid",
					},
				},
			}
			if inSpec {
				vmi.Spec.Volumes = []virtv1.Volume{
					{
						Name: volName,
						VolumeSource: virtv1.VolumeSource{
							PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
								PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
									ClaimName: volName + "-pvc",
								},
								Hotpluggable: true,
							},
						},
					},
				}
			} else {
				vmi.Spec.Volumes = []virtv1.Volume{}
			}

			readyVolumes := []*virtv1.Volume{{}}
			oldPod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "old-pod",
					Namespace: vmi.Namespace,
				},
				Spec: k8sv1.PodSpec{
					Volumes: []k8sv1.Volume{
						{Name: volName},
					},
				},
				Status: k8sv1.PodStatus{
					Phase: k8sv1.PodRunning,
				},
			}
			addPod(oldPod)

			err := controller.cleanupAttachmentPods(nil, []*k8sv1.Pod{oldPod}, vmi, len(readyVolumes))
			Expect(err).ToNot(HaveOccurred())
			if expectDelete {
				testutils.ExpectEvent(recorder, kvcontroller.SuccessfulDeletePodReason)
				expectPodDoesNotExist(oldPod.Namespace, oldPod.Name)
			} else {
				expectPodExists(oldPod.Namespace, oldPod.Name)
			}
		},
			Entry("should delete when all volumes are Detaching",
				"detaching-vol", virtv1.HotplugVolumeDetaching, false, true),
			Entry("should keep when volume is still in spec",
				"active-vol", virtv1.VolumeReady, true, false),
			Entry("should keep when removed volume is in deletion-blocking phase",
				"blocking-vol", virtv1.VolumeReady, false, false),
		)

		It("should keep old running pod as fallback when it has both a Detaching and an in-spec volume", func() {
			vmi := watchtesting.NewRunningVirtualMachine("testvmi", &k8sv1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testnode",
				},
			})
			vmi.Spec.Volumes = []virtv1.Volume{
				{
					Name: "active-vol",
					VolumeSource: virtv1.VolumeSource{
						PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "active-pvc",
							},
							Hotpluggable: true,
						},
					},
				},
			}
			vmi.Status.VolumeStatus = []virtv1.VolumeStatus{
				{
					Name:  "detaching-vol",
					Phase: virtv1.HotplugVolumeDetaching,
					HotplugVolume: &virtv1.HotplugVolumeStatus{
						AttachPodName: "old-pod",
						AttachPodUID:  "old-uid",
					},
				},
				{
					Name:  "active-vol",
					Phase: virtv1.VolumeReady,
					HotplugVolume: &virtv1.HotplugVolumeStatus{
						AttachPodName: "old-pod",
						AttachPodUID:  "old-uid",
					},
				},
			}

			readyVolumes := []*virtv1.Volume{{}}
			oldPod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "old-pod",
					Namespace: vmi.Namespace,
				},
				Spec: k8sv1.PodSpec{
					Volumes: []k8sv1.Volume{
						{Name: "detaching-vol"},
						{Name: "active-vol"},
					},
				},
				Status: k8sv1.PodStatus{
					Phase: k8sv1.PodRunning,
				},
			}
			addPod(oldPod)

			err := controller.cleanupAttachmentPods(nil, []*k8sv1.Pod{oldPod}, vmi, len(readyVolumes))
			Expect(err).ToNot(HaveOccurred())
			expectPodExists(oldPod.Namespace, oldPod.Name)
		})

		DescribeTable("should clean up utility volume attachment pods when replacement pod is running",
			func(includeUtilityPVC bool, includeUtilityMount bool, expectDelete bool) {
				vmi := watchtesting.NewRunningVirtualMachine("testvmi", &k8sv1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testnode",
					},
				})
				vmi.Spec.UtilityVolumes = []virtv1.UtilityVolume{
					{
						Name: "utility-vol",
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "utility-pvc",
						},
					},
				}
				vmi.Status.VolumeStatus = []virtv1.VolumeStatus{
					{
						Name:  "utility-vol",
						Phase: virtv1.HotplugVolumeMounted,
						HotplugVolume: &virtv1.HotplugVolumeStatus{
							AttachPodName: "new-pod",
							AttachPodUID:  "new-uid",
						},
					},
				}

				oldPod := &k8sv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "old-pod",
						Namespace: vmi.Namespace,
					},
					Spec: k8sv1.PodSpec{
						Volumes: []k8sv1.Volume{
							{
								Name: "utility-vol",
								VolumeSource: k8sv1.VolumeSource{
									PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
										ClaimName: "utility-pvc",
									},
								},
							},
						},
					},
					Status: k8sv1.PodStatus{
						Phase: k8sv1.PodRunning,
					},
				}

				currentPodSpec := k8sv1.PodSpec{
					Volumes: []k8sv1.Volume{
						{
							Name: "hotplug-vol",
							VolumeSource: k8sv1.VolumeSource{
								PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
									ClaimName: "hotplug-pvc",
								},
							},
						},
					},
				}
				if includeUtilityPVC {
					currentPodSpec.Volumes = append([]k8sv1.Volume{
						{
							Name: "utility-vol",
							VolumeSource: k8sv1.VolumeSource{
								PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
									ClaimName: "utility-pvc",
								},
							},
						},
					}, currentPodSpec.Volumes...)
				}
				if includeUtilityMount {
					currentPodSpec.Containers = []k8sv1.Container{
						{
							Name: "hotplug-disk",
							VolumeMounts: []k8sv1.VolumeMount{
								{
									Name:      "utility-vol",
									MountPath: "/utility-vol",
								},
							},
						},
					}
				}
				currentPod := &k8sv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "new-pod",
						Namespace: vmi.Namespace,
					},
					Spec: currentPodSpec,
					Status: k8sv1.PodStatus{
						Phase: k8sv1.PodRunning,
					},
				}

				addPod(oldPod)

				err := controller.cleanupAttachmentPods(currentPod, []*k8sv1.Pod{oldPod}, vmi, 2)
				Expect(err).ToNot(HaveOccurred())
				if expectDelete {
					testutils.ExpectEvent(recorder, kvcontroller.SuccessfulDeletePodReason)
					expectPodDoesNotExist(oldPod.Namespace, oldPod.Name)
				} else {
					expectPodExists(oldPod.Namespace, oldPod.Name)
				}
			},
			Entry("delete when replacement exposes utility volume", true, true, true),
			Entry("keep when replacement is missing utility volume", false, false, false),
			Entry("keep when replacement has utility PVC without container mount", true, false, false),
		)

		DescribeTable("volumeHandledByCurrentPod",
			func(currentPod *k8sv1.Pod, volumeName string, specVolumes map[string]struct{}, expected bool) {
				Expect(volumeHandledByCurrentPod(volumeName, currentPod, specVolumes)).To(Equal(expected))
			},
			Entry("nil pod", nil, "vol", map[string]struct{}{"vol": {}}, false),
			Entry("pending pod", &k8sv1.Pod{Status: k8sv1.PodStatus{Phase: k8sv1.PodPending}}, "vol", map[string]struct{}{"vol": {}}, false),
			Entry("volume not in spec map", &k8sv1.Pod{Status: k8sv1.PodStatus{Phase: k8sv1.PodRunning}}, "vol", map[string]struct{}{}, false),
			Entry("pvc without container mount", &k8sv1.Pod{
				Status: k8sv1.PodStatus{Phase: k8sv1.PodRunning},
				Spec: k8sv1.PodSpec{
					Volumes: []k8sv1.Volume{
						{
							Name: "vol",
							VolumeSource: k8sv1.VolumeSource{
								PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc"},
							},
						},
					},
				},
			}, "vol", map[string]struct{}{"vol": {}}, false),
			Entry("pvc with volume mount", &k8sv1.Pod{
				Status: k8sv1.PodStatus{Phase: k8sv1.PodRunning},
				Spec: k8sv1.PodSpec{
					Containers: []k8sv1.Container{
						{
							Name: "hotplug-disk",
							VolumeMounts: []k8sv1.VolumeMount{
								{Name: "vol", MountPath: "/vol"},
							},
						},
					},
					Volumes: []k8sv1.Volume{
						{
							Name: "vol",
							VolumeSource: k8sv1.VolumeSource{
								PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc"},
							},
						},
					},
				},
			}, "vol", map[string]struct{}{"vol": {}}, true),
			Entry("pvc with volume device", &k8sv1.Pod{
				Status: k8sv1.PodStatus{Phase: k8sv1.PodRunning},
				Spec: k8sv1.PodSpec{
					Containers: []k8sv1.Container{
						{
							Name: "hotplug-disk",
							VolumeDevices: []k8sv1.VolumeDevice{
								{Name: "vol", DevicePath: "/path/vol/uid"},
							},
						},
					},
					Volumes: []k8sv1.Volume{
						{
							Name: "vol",
							VolumeSource: k8sv1.VolumeSource{
								PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc"},
							},
						},
					},
				},
			}, "vol", map[string]struct{}{"vol": {}}, true),
		)
	})

	Context("Aggregating DataVolume conditions", func() {

		dvVolumeSource1 := virtv1.VolumeSource{
			DataVolume: &virtv1.DataVolumeSource{
				Name: "test1",
			},
		}
		dvVolumeSource2 := virtv1.VolumeSource{
			DataVolume: &virtv1.DataVolumeSource{
				Name: "test2",
			},
		}
		dvVolumeSource3 := virtv1.VolumeSource{
			DataVolume: &virtv1.DataVolumeSource{
				Name: "test3",
			},
		}
		DescribeTable("Should aggregate conditions from 3 DataVolumes on VMI status",
			func(dvConds1, dvConds2, dvConds3 []cdiv1.DataVolumeCondition, expectedStatus k8sv1.ConditionStatus, expectedMessage string) {
				vmi := newPendingVirtualMachine("testvmi")

				vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
					Name:         "test1",
					VolumeSource: dvVolumeSource1,
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
					Name:         "test2",
					VolumeSource: dvVolumeSource2,
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
					Name:         "test3",
					VolumeSource: dvVolumeSource3,
				})

				dvPVC1 := newPvc(vmi.Namespace, "test1")
				dvPVC2 := newPvc(vmi.Namespace, "test2")
				dvPVC3 := newPvc(vmi.Namespace, "test3")
				// we are mocking a successful DataVolume. we expect the PVC to
				// be available in the store if DV is successful.
				addDataVolumePVC(dvPVC1)
				addDataVolumePVC(dvPVC2)
				addDataVolumePVC(dvPVC3)

				dataVolume1 := newDv(vmi.Namespace, "test1", cdiv1.Unknown)
				dataVolume2 := newDv(vmi.Namespace, "test2", cdiv1.Unknown)
				dataVolume3 := newDv(vmi.Namespace, "test3", cdiv1.Unknown)
				for _, c := range dvConds1 {
					setDataVolumeCondition(dataVolume1, c)
				}
				for _, c := range dvConds2 {
					setDataVolumeCondition(dataVolume2, c)
				}
				for _, c := range dvConds3 {
					setDataVolumeCondition(dataVolume3, c)
				}

				addVirtualMachine(vmi)
				addDataVolume(dataVolume1)
				addDataVolume(dataVolume2)
				addDataVolume(dataVolume3)

				sanityExecute()
				testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
				expectMatchingPodCreation(vmi)
				expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name,
					ContainElement(
						MatchFields(IgnoreExtras,
							Fields{
								"Type":    BeEquivalentTo(virtv1.VirtualMachineInstanceDataVolumesReady),
								"Status":  BeEquivalentTo(expectedStatus),
								"Message": BeEquivalentTo(expectedMessage),
							},
						),
					),
				)
			},
			Entry("3 ready DataVolumes",
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionTrue},
				},
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionTrue},
				},
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionTrue},
				},
				k8sv1.ConditionTrue,
				"All of the VMI's DVs are bound and ready",
			),
			Entry("2 ready DataVolumes, 1 not ready",
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionTrue},
				},
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionFalse},
				},
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionTrue},
				},
				k8sv1.ConditionFalse,
				"Not all of the VMI's DVs are ready",
			),
			Entry("3 Unknown DataVolumes",
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionUnknown},
				},
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionUnknown},
				},
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionUnknown},
				},
				k8sv1.ConditionUnknown,
				"Not all of the VMI's DVs are ready",
			),
			Entry("2 ready, 1 unknown",
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionTrue},
				},
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionTrue},
				},
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionUnknown},
				},
				k8sv1.ConditionUnknown,
				"Not all of the VMI's DVs are ready",
			),
			Entry("1 ready, 1 not ready, 1 unknown",
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionTrue},
				},
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionFalse},
				},
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionUnknown},
				},
				k8sv1.ConditionFalse,
				"Not all of the VMI's DVs are ready",
			),
			Entry("1 ready, 1 unknown, 1 not ready",
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionTrue},
				},
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionUnknown},
				},
				[]cdiv1.DataVolumeCondition{
					{Type: cdiv1.DataVolumeReady, Status: k8sv1.ConditionFalse},
				},
				k8sv1.ConditionFalse,
				"Not all of the VMI's DVs are ready",
			),
		)
	})

	Context("Event handling", func() {
		indexVMI := func(vmi *virtv1.VirtualMachineInstance) {
			Expect(controller.vmiIndexer.Add(vmi)).To(Succeed())
			_, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		pvcWith := func(requestSize, capacity, rv string) *k8sv1.PersistentVolumeClaim {
			pvc := newPvc("default", "testpvc")
			pvc.Spec.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceStorage: resource.MustParse(requestSize),
			}
			pvc.Status.Capacity = k8sv1.ResourceList{
				k8sv1.ResourceStorage: resource.MustParse(capacity),
			}
			pvc.ObjectMeta.ResourceVersion = rv
			return pvc
		}

		DescribeTable("updatePVC", func(newReq, newCap, newRV string, expectEnqueue bool) {
			vmi := libvmi.New(
				libvmi.WithName("testvmi"),
				libvmi.WithNamespace("default"),
				libvmi.WithDataVolume("myvolume", "testpvc"),
			)
			indexVMI(vmi)

			oldPVC := pvcWith("50Gi", "50Gi", "1")
			newPVC := pvcWith(newReq, newCap, newRV)

			addDataVolumePVC(oldPVC)
			controller.updatePVC(oldPVC, newPVC)

			if expectEnqueue {
				Eventually(func() int {
					return mockQueue.Len()
				}, time.Second, 100*time.Millisecond).Should(Equal(1))

				item, _ := mockQueue.Get()
				Expect(item).To(Equal("default/testvmi"))
			} else {
				Consistently(func() int {
					return mockQueue.Len()
				}, time.Second, 100*time.Millisecond).Should(Equal(0))
			}
		},
			Entry("should enqueue when requested size changes", "100Gi", "50Gi", "2", true),
			Entry("should enqueue when capacity changes", "50Gi", "100Gi", "2", true),
			Entry("should not enqueue when nothing changed", "50Gi", "50Gi", "2", false),
			Entry("should not enqueue when resource version is the same", "10Gi", "50Gi", "1", false),
		)
	})

})
