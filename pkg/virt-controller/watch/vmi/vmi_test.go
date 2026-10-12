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
	"maps"
	"slices"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegaTypes "github.com/onsi/gomega/types"
	"go.uber.org/mock/gomock"

	"kubevirt.io/kubevirt/tests/framework/matcher"

	k8sv1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/storage/velero"

	kvcontroller "kubevirt.io/kubevirt/pkg/controller"
	controllertesting "kubevirt.io/kubevirt/pkg/controller/testing"
	storageannotations "kubevirt.io/kubevirt/pkg/storage/pod/annotations"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	watchtesting "kubevirt.io/kubevirt/pkg/virt-controller/watch/testing"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/descheduler"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
)

var _ = Describe("VirtualMachineInstance watcher", func() {
	var config *virtconfig.ClusterConfig

	var controller *Controller
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue[string]
	var virtClientset *kubevirtfake.Clientset
	var kubeClient *fake.Clientset
	// We pass the store to backend storage and we don't have direct access
	var storageClassStore cache.Store
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

	expectPodAnnotations := func(pod *k8sv1.Pod, matchers ...gomegaTypes.GomegaMatcher) {
		retrievedPod, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(retrievedPod.Annotations).To(SatisfyAll(matchers...))
	}

	ignorePodUpdates := func() {
		kubeClient.Fake.PrependReactor("update", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			update, _ := action.(testing.UpdateAction)
			return true, update.GetObject(), nil
		})
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

	expectVMITimestamp := func(namespace, name string, matchers ...gomegaTypes.GomegaMatcher) {
		updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVmi.Status.PhaseTransitionTimestamps).To(SatisfyAll(matchers...))
	}

	expectVMISchedulingState := func(vmi *virtv1.VirtualMachineInstance) {
		expectVMIBeInPhase(vmi.Namespace, vmi.Name, virtv1.Scheduling)
		expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
			Fields{"Type": Equal(virtv1.VirtualMachineInstanceReady)})))
		expectVMITimestamp(vmi.Namespace, vmi.Name, Not(BeEmpty()), Not(HaveLen(len(vmi.Status.PhaseTransitionTimestamps))))
	}

	expectVMIScheduledState := func(vmi *virtv1.VirtualMachineInstance) {
		expectVMIBeInPhase(vmi.Namespace, vmi.Name, virtv1.Scheduled)
		expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
			Fields{"Type": Equal(virtv1.VirtualMachineInstanceReady)})))
		expectVMITimestamp(vmi.Namespace, vmi.Name, Not(BeEmpty()), Not(HaveLen(len(vmi.Status.PhaseTransitionTimestamps))))
	}

	expectVMIFailedState := func(vmi *virtv1.VirtualMachineInstance) {
		expectVMIBeInPhase(vmi.Namespace, vmi.Name, virtv1.Failed)
		expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
			Fields{"Type": Equal(virtv1.VirtualMachineInstanceReady)})))
		expectVMITimestamp(vmi.Namespace, vmi.Name, Not(BeEmpty()), Not(HaveLen(len(vmi.Status.PhaseTransitionTimestamps))))
	}

	// Needs to be set in BeforeEach in order to not expose storageClass/Profile informer/store
	sanityExecute := func() {}

	BeforeEach(func() {
		virtClient := kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		virtClientset = kubevirtfake.NewSimpleClientset()

		vmiInformer, _ := testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachineInstance{}, kvcontroller.GetVMIInformerIndexers())

		vmInformer, _ := testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachine{}, kvcontroller.GetVirtualMachineInformerIndexers())
		podInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})
		dataVolumeInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		storageProfileInformer, _ := testutils.NewFakeInformerFor(&cdiv1.StorageProfile{})
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

	Describe("recovering a VMI from pod owner annotations", func() {
		It("should recover the VMI when the annotated owner identity matches", func() {
			vmi := newPendingVirtualMachine("testvmi")
			Expect(controller.vmiIndexer.Add(vmi)).To(Succeed())

			pod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "attachment-pod",
					Namespace: vmi.Namespace,
					Annotations: map[string]string{
						virtv1.OwnerVMINameAnnotation: vmi.Name,
						virtv1.OwnerVMIUIDAnnotation:  string(vmi.UID),
					},
				},
			}

			Expect(controller.recoverVMIFromPodAnnotations(pod)).To(Equal(vmi))
		})

		It("should not recover the VMI when the annotated owner UID does not match", func() {
			vmi := newPendingVirtualMachine("testvmi")
			Expect(controller.vmiIndexer.Add(vmi)).To(Succeed())

			pod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "attachment-pod",
					Namespace: vmi.Namespace,
					Annotations: map[string]string{
						virtv1.OwnerVMINameAnnotation: vmi.Name,
						virtv1.OwnerVMIUIDAnnotation:  "other-uid",
					},
				},
			}

			Expect(controller.recoverVMIFromPodAnnotations(pod)).To(BeNil())
		})

		It("should observe an attachment pod deletion when the virt-launcher pod is gone from the cache", func() {
			vmi := newPendingVirtualMachine("testvmi")
			Expect(controller.vmiIndexer.Add(vmi)).To(Succeed())
			vmiKey := kvcontroller.VirtualMachineInstanceKey(vmi)

			virtlauncherPod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			attachmentPod := newPodForVirtlauncher(virtlauncherPod, "attachment-pod", "attachment-uid", k8sv1.PodRunning)
			attachmentPod.Annotations = map[string]string{
				virtv1.OwnerVMINameAnnotation: vmi.Name,
				virtv1.OwnerVMIUIDAnnotation:  string(vmi.UID),
			}

			controller.podExpectations.ExpectDeletions(vmiKey, []string{kvcontroller.PodKey(attachmentPod)})
			Expect(controller.podExpectations.SatisfiedExpectations(vmiKey)).To(BeFalse())

			controller.onPodDelete(attachmentPod)

			Expect(controller.podExpectations.SatisfiedExpectations(vmiKey)).To(BeTrue())
			Expect(mockQueue.Len()).To(Equal(1))
			key, shutdown := mockQueue.Get()
			Expect(shutdown).To(BeFalse())
			Expect(key).To(Equal(vmiKey))
			mockQueue.Done(key)
		})
	})

	Context("network annotations generation", func() {
		const (
			key1   = "key1"
			value1 = "value1"
			key2   = "key2"
			value2 = "value2"
		)

		It("should patch the pod's annotations when network annotations generator returns custom annotations", func() {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Running

			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			addPod(pod)
			addActivePods(vmi, pod.UID, "")

			controller.netAnnotationsGenerator = stubNetworkAnnotationsGenerator{
				annotations: map[string]string{key1: value1, key2: value2},
			}
			sanityExecute()
			expectPodAnnotations(pod, HaveKeyWithValue(key1, value1), HaveKeyWithValue(key2, value2))
		})

		It("should not patch the pod's annotations when network annotations generator returns an empty map", func() {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Running

			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			addPod(pod)
			addActivePods(vmi, pod.UID, "")

			controller.netAnnotationsGenerator = stubNetworkAnnotationsGenerator{annotations: map[string]string{}}
			sanityExecute()

			expectPodAnnotations(pod, Not(HaveKeyWithValue(key1, value1)), Not(HaveKeyWithValue(key2, value2)))
		})
	})

	Context("VMI patch", func() {
		var (
			oldVMI *virtv1.VirtualMachineInstance
			newVMI *virtv1.VirtualMachineInstance
		)
		BeforeEach(func() {
			oldVMI = newPendingVirtualMachine("oldvmi")
			newVMI = newPendingVirtualMachine("newvmi")

			oldVMI.Status.Interfaces = []virtv1.VirtualMachineInstanceNetworkInterface{
				{Name: "stubNetStatusUpdate1"},
				{Name: "stubNetStatusUpdate2"},
			}
		})
		It("should create empty status network interfaces patch, regardless of interfaces order", func() {

			// Reverse the order of interfaces in the new VMI
			newVMI.Status.Interfaces = slices.Clone(oldVMI.Status.Interfaces)
			slices.Reverse(newVMI.Status.Interfaces)

			patch := prepareVMIPatch(oldVMI, newVMI)

			Expect(patch).ToNot(BeNil(), "Patch should not be nil")
			Expect(patch.IsEmpty()).To(BeTrue(), "Patch should be empty")
		})
		It("should create non empty status network interfaces patch", func() {

			// Add a new interface to the new VMI
			newVMI.Status.Interfaces = append(oldVMI.Status.Interfaces,
				virtv1.VirtualMachineInstanceNetworkInterface{Name: "stubNetStatusUpdate3"},
			)

			patch := prepareVMIPatch(oldVMI, newVMI)

			Expect(patch).ToNot(BeNil(), "Patch should not be nil")
			Expect(patch.IsEmpty()).To(BeFalse(), "Patch should not be empty")
		})
	})

	It("should fail which when network VMI spec validator fail", func() {
		controller.validateNetworkSpec = validateNetVMISpecStub(metav1.StatusCause{Type: "test", Message: "test", Field: "test"})

		vmi := newPendingVirtualMachine("testvmi")
		addVirtualMachine(vmi)

		sanityExecute()

		vmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Conditions).To(SatisfyAll(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type":    Equal(virtv1.VirtualMachineInstanceSynchronized),
			"Status":  Equal(k8sv1.ConditionFalse),
			"Reason":  Equal("FailedCreateValidation"),
			"Message": Equal("failed create validation: &StatusCause{Type:test,Message:test,Field:test,}"),
		}))))
	})

	It("should fail when VMI has ContainerPath volumes but feature gate is disabled", func() {
		vmi := newPendingVirtualMachine("testvmi")

		// Add ContainerPath volume with matching filesystem
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
			Name: "token-volume",
			VolumeSource: virtv1.VolumeSource{
				ContainerPath: &virtv1.ContainerPathVolumeSource{
					Path: "/var/run/secrets/token",
				},
			},
		})
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, virtv1.Filesystem{
			Name:     "token-volume",
			Virtiofs: &virtv1.FilesystemVirtiofs{},
		})

		addVirtualMachine(vmi)

		// Feature gate is disabled by default in tests
		sanityExecute()

		testutils.ExpectEvent(recorder, virtv1.ContainerPathVolumesDisabledReason)
		vmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
			"Status": Equal(k8sv1.ConditionFalse),
			"Reason": Equal(virtv1.ContainerPathVolumesDisabledReason),
		})))
	})

	It("should fail when pod is missing virtiofs containers for ContainerPath volumes", func() {
		vmi := newPendingVirtualMachine("testvmi")
		vmi.Status.Phase = virtv1.Scheduling

		// Add ContainerPath volume with matching filesystem
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
			Name: "token-volume",
			VolumeSource: virtv1.VolumeSource{
				ContainerPath: &virtv1.ContainerPathVolumeSource{
					Path: "/var/run/secrets/token",
				},
			},
		})
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, virtv1.Filesystem{
			Name:     "token-volume",
			Virtiofs: &virtv1.FilesystemVirtiofs{},
		})

		// Create pod WITHOUT the expected virtiofs container
		pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
		addActivePods(vmi, pod.UID, "")
		addVirtualMachine(vmi)
		addPod(pod)

		sanityExecute()

		testutils.ExpectEvent(recorder, virtv1.MissingVirtiofsContainersReason)
		vmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
			"Status": Equal(k8sv1.ConditionFalse),
			"Reason": Equal(virtv1.MissingVirtiofsContainersReason),
		})))
	})

	Context("On valid VirtualMachineInstance given", func() {
		It("should create a corresponding Pod on VirtualMachineInstance creation with proper annotation", func() {
			vmi := newPendingVirtualMachine("testvmi")

			addVirtualMachine(vmi)

			sanityExecute()

			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
			expectMatchingPodCreation(vmi, WithTransform(
				func(pod *k8sv1.Pod) map[string]string {
					return pod.Annotations
				},
				HaveKey(descheduler.EvictOnlyAnnotation),
			))
		})

		It("should add request-evict-only annotation to the virt-launcher pod if annotation does not exist", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionTrue, "")
			vmi.Status.Phase = virtv1.Running
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			delete(pod.Annotations, descheduler.EvictOnlyAnnotation)

			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()

			expectPodAnnotations(pod, HaveKey(descheduler.EvictOnlyAnnotation))
		})

		DescribeTable("should delete the corresponding Pods on VirtualMachineInstance deletion with vmi", func(phase virtv1.VirtualMachineInstancePhase) {
			vmi := newPendingVirtualMachine("testvmi")

			vmi.Status.Phase = phase
			vmi.DeletionTimestamp = new(metav1.Now())

			if vmi.IsRunning() {
				setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodConditionMissingReason)
			} else {
				setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			}

			// 2 pods are owned by VMI
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.UID = "456-456-456"
			pod2 := pod.DeepCopy()
			pod2.UID = "123-123-123"
			pod2.Name = "test2"

			// The 3rd one is not.
			pod3 := pod.DeepCopy()
			pod3.UID = "789-789-789"
			pod3.Name = "test3"
			pod3.Annotations = map[string]string{
				virtv1.DomainAnnotation: "another-vmi",
			}
			pod3.Labels = map[string]string{
				virtv1.AppLabel:       "virt-launcher",
				virtv1.CreatedByLabel: "another-vmi-uid",
			}

			addActivePods(vmi, pod.UID, "")
			addActivePods(vmi, pod2.UID, "")

			addVirtualMachine(vmi)
			addPod(pod)
			addPod(pod2)

			_, err := kubeClient.CoreV1().Pods(pod3.Namespace).Create(context.Background(), pod3, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			sanityExecute()

			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulDeletePodReason)
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulDeletePodReason)
			// Verify only the 2 pods are deleted
			expectPodDoesNotExist(pod.Namespace, pod.Name)
			expectPodDoesNotExist(pod2.Namespace, pod2.Name)
			expectPodExists(pod3.Namespace, pod3.Name)

			if vmi.IsUnprocessed() {
				expectVMISchedulingState(vmi)
			}
		},
			Entry("in running state", virtv1.Running),
			Entry("in unset state", virtv1.VmPhaseUnset),
			Entry("in pending state", virtv1.Pending),
			Entry("in succeeded state", virtv1.Succeeded),
			Entry("in failed state", virtv1.Failed),
			Entry("in scheduled state", virtv1.Scheduled),
			Entry("in scheduling state", virtv1.Scheduling),
		)

		It("should delete finalized pods on VMI deletion", func() {
			vmi := newPendingVirtualMachine("testvmi")

			vmi.Status.Phase = virtv1.Failed
			vmi.DeletionTimestamp = new(metav1.Now())

			finalizedPod := newPodForVirtualMachine(vmi, k8sv1.PodSucceeded)
			finalizedPod.UID = "finalized-123"
			finalizedPod.Name = "finalized-pod"

			addVirtualMachine(vmi)
			addPod(finalizedPod)
			addActivePods(vmi, finalizedPod.UID, "")

			sanityExecute()

			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulDeletePodReason)

			expectPodDoesNotExist(finalizedPod.Namespace, finalizedPod.Name)
		})

		It("should not try to delete a pod again, which is already marked for deletion and go to failed state, when in scheduling state", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)

			vmi.Status.Phase = virtv1.Scheduling
			vmi.DeletionTimestamp = new(metav1.Now())
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			addPod(pod)
			addActivePods(vmi, pod.UID, "")

			sanityExecute()

			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulDeletePodReason)
			expectPodDoesNotExist(pod.Namespace, pod.Name)

			modifiedPod := pod.DeepCopy()
			modifiedPod.DeletionTimestamp = new(metav1.Now())

			key, err := kvcontroller.KeyFunc(vmi)
			Expect(err).To(Not(HaveOccurred()))
			mockQueue.Add(key)
			controller.podIndexer.Update(modifiedPod)
			controller.podExpectations.SetExpectations(key, 0, 0)

			sanityExecute()
			expectVMIFailedState(vmi)
		})
		DescribeTable("should not delete the corresponding Pod if the vmi is in", func(phase virtv1.VirtualMachineInstancePhase) {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)

			vmi.Status.Phase = phase
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			addPod(pod)
			addActivePods(vmi, pod.UID, "")

			sanityExecute()
			expectPodExists(pod.Namespace, pod.Name)
		},
			Entry("succeeded state", virtv1.Failed),
			Entry("failed state", virtv1.Succeeded),
		)
		It("should do nothing if the vmi is in final state", func() {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Failed
			vmi.Finalizers = []string{}

			addVirtualMachine(vmi)

			sanityExecute()
			Expect(virtClientset.Actions()).To(HaveLen(1))
			Expect(virtClientset.Actions()[0].GetVerb()).To(Equal("create"))
			Expect(virtClientset.Actions()[0].GetResource().Resource).To(Equal("virtualmachineinstances"))
			Expect(kubeClient.Actions()).To(BeEmpty())
		})

		It("should set an error condition if creating the pod fails", func() {
			vmi := newPendingVirtualMachine("testvmi")

			addVirtualMachine(vmi)

			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			sanityExecute()

			testutils.ExpectEvent(recorder, kvcontroller.FailedCreatePodReason)
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
				Fields{
					"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
					"Status": Equal(k8sv1.ConditionFalse),
					"Reason": Equal("FailedCreate"),
				})),
			)
		})
		It("should back-off if a sync error occurs", func() {
			vmi := newPendingVirtualMachine("testvmi")

			addVirtualMachine(vmi)

			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			sanityExecute()
			Expect(controller.Queue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(1))

			testutils.ExpectEvent(recorder, kvcontroller.FailedCreatePodReason)
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
				Fields{
					"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
					"Status": Equal(k8sv1.ConditionFalse),
					"Reason": Equal("FailedCreate"),
				})),
			)
		})
		It("should requeue after the expectations timeout when expectations are not met", func() {
			vmi := newPendingVirtualMachine("testvmi")
			addVirtualMachine(vmi)
			key, err := kvcontroller.KeyFunc(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(mockQueue.GetAddAfterEnqueueCount()).To(BeZero())

			// A pending creation expectation that no event will satisfy.
			controller.podExpectations.SetExpectations(key, 1, 0)

			sanityExecute()
			Expect(mockQueue.GetAddAfterEnqueueCount()).To(Equal(1))
		})

		It("should remove the error condition if the sync finally succeeds", func() {
			vmi := newPendingVirtualMachine("testvmi")

			vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{Type: virtv1.VirtualMachineInstanceSynchronized})

			addVirtualMachine(vmi)

			sanityExecute()
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
			expectMatchingPodCreation(vmi)
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name,
				Not(ContainElement(
					MatchFields(IgnoreExtras,
						Fields{
							"Type": Equal(virtv1.VirtualMachineInstanceSynchronized),
						},
					),
				)),
			)
		})
		DescribeTable("should never proceed to creating the launcher pod if not all PVCs are there to determine if they are WFFC and/or an import is done",
			func(syncReason string, volumeSource virtv1.VolumeSource) {

				expectConditions := func(vmi *virtv1.VirtualMachineInstance) {
					// PodScheduled and Synchronized (as well as Ready)
					Expect(vmi.Status.Conditions).To(HaveLen(3), "there should be exactly 3 conditions")

					getType := func(c virtv1.VirtualMachineInstanceCondition) string { return string(c.Type) }
					getReason := func(c virtv1.VirtualMachineInstanceCondition) string { return c.Reason }
					Expect(vmi.Status.Conditions).To(
						And(
							ContainElement(
								And(
									WithTransform(getType, Equal(string(virtv1.VirtualMachineInstanceSynchronized))),
									WithTransform(getReason, Equal(syncReason)),
								),
							),
							ContainElement(
								And(
									WithTransform(getType, Equal(string(k8sv1.PodScheduled))),
									WithTransform(getReason, Equal(k8sv1.PodReasonUnschedulable)),
								),
							),
						),
					)

					testutils.ExpectEvent(recorder, syncReason)
				}

				vmi := newPendingVirtualMachine("testvmi")
				setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodNotExistsReason)

				vmi.Spec.Volumes = []virtv1.Volume{
					{
						Name:         "test",
						VolumeSource: volumeSource,
					},
				}
				addVirtualMachine(vmi)

				sanityExecute()
				updatedVMI, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				expectConditions(updatedVMI)
				Expect(controller.Queue.Len()).To(Equal(0))
				// When the PVC appears, we will automatically be retriggered, it is not an error condition
				Expect(mockQueue.GetRateLimitedEnqueueCount()).To(BeZero())

				controller.vmiIndexer.Add(vmi)
				key, err := kvcontroller.KeyFunc(vmi)
				Expect(err).To(Not(HaveOccurred()))
				mockQueue.Add(key)
				// in previous loop we have updated the status
				controller.vmiExpectations.SetExpectations(key, 0, 0)

				sanityExecute()
				Expect(controller.Queue.Len()).To(Equal(0))
				Expect(mockQueue.GetRateLimitedEnqueueCount()).To(BeZero())
				// make sure that during next iteration we do not add the same condition again
				updatedVMI, err = virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				expectConditions(updatedVMI)
			},
			Entry("when PVC does not exist", kvcontroller.FailedPvcNotFoundReason,
				virtv1.VolumeSource{
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "something",
						},
					},
				}),
			Entry("when DataVolume does not exist", kvcontroller.FailedPvcNotFoundReason,
				virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "something",
					},
				}),
		)

		DescribeTable("should move the vmi to scheduling state if a pod exists", func(phase k8sv1.PodPhase, isReady bool) {
			vmi := newPendingVirtualMachine("testvmi")
			pod := newPodForVirtualMachine(vmi, phase)
			pod.Status.ContainerStatuses[0].Ready = isReady

			unreadyReason := virtv1.GuestNotRunningReason
			if phase == k8sv1.PodSucceeded || phase == k8sv1.PodFailed {
				unreadyReason = virtv1.PodTerminatingReason
			}
			setReadyCondition(vmi, k8sv1.ConditionFalse, unreadyReason)

			addVirtualMachine(vmi)
			addPod(pod)
			addActivePods(vmi, pod.UID, "")

			sanityExecute()
			expectVMISchedulingState(vmi)
		},
			Entry(", not ready and in running state", k8sv1.PodRunning, false),
			Entry(", not ready and in unknown state", k8sv1.PodUnknown, false),
			Entry(", not ready and in succeeded state", k8sv1.PodSucceeded, false),
			Entry(", not ready and in failed state", k8sv1.PodFailed, false),
			Entry(", not ready and in pending state", k8sv1.PodPending, false),
			Entry(", ready and in running state", k8sv1.PodRunning, true),
			Entry(", ready and in unknown state", k8sv1.PodUnknown, true),
			Entry(", ready and in succeeded state", k8sv1.PodSucceeded, true),
			Entry(", ready and in failed state", k8sv1.PodFailed, true),
			Entry(", ready and in pending state", k8sv1.PodPending, true),
		)

		Context("when pod failed to schedule", func() {
			It("should set scheduling pod condition on the VirtualMachineInstance", func() {
				vmi := newPendingVirtualMachine("testvmi")
				setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
				vmi.Status.Phase = virtv1.Scheduling

				pod := newPodForVirtualMachine(vmi, k8sv1.PodPending)

				pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{
					Message: "Insufficient memory",
					Reason:  "Unschedulable",
					Status:  k8sv1.ConditionFalse,
					Type:    k8sv1.PodScheduled,
				})

				addVirtualMachine(vmi)
				addPod(pod)

				sanityExecute()
				expectVMIBeInPhase(vmi.Namespace, vmi.Name, virtv1.Scheduling)
				expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Type":    Equal(virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled)),
						"Status":  Equal(k8sv1.ConditionFalse),
						"Reason":  Equal("Unschedulable"),
						"Message": Equal("Insufficient memory"),
					})))
			})
		})

		Context("when Pod recovers from scheduling issues", func() {
			DescribeTable("it should remove scheduling pod condition from the VirtualMachineInstance if the pod", func(owner string, podPhase k8sv1.PodPhase) {
				vmi := newPendingVirtualMachine("testvmi")
				setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodConditionMissingReason)
				vmi.Status.Phase = virtv1.Scheduling

				pod := newPodForVirtualMachine(vmi, podPhase)

				vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
					Message: "Insufficient memory",
					Reason:  "Unschedulable",
					Status:  k8sv1.ConditionFalse,
					Type:    virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled),
				})

				addVirtualMachine(vmi)
				addPod(pod)

				ignorePodUpdates()

				sanityExecute()
				testutils.IgnoreEvents(recorder)
				expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, Not(ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Type": Equal(virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled)),
					}))),
				)
			},
				Entry("is owned by virt-handler and is running", "virt-handler", k8sv1.PodRunning),
				Entry("is owned by virt-controller and is running", "virt-controller", k8sv1.PodRunning),
				Entry("is owned by virt-controller and is pending", "virt-controller", k8sv1.PodPending),
			)
		})

		It("should move the vmi to failed state if the pod disappears and the vmi is in scheduling state", func() {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Scheduling

			addVirtualMachine(vmi)

			sanityExecute()
			expectVMIFailedState(vmi)
		})
		It("should move the vmi to failed state if the vmi is pending, no pod exists yet and gets deleted", func() {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.DeletionTimestamp = new(metav1.Now())

			addVirtualMachine(vmi)

			sanityExecute()
			expectVMIFailedState(vmi)
		})

		DescribeTable("should hand over pod to virt-handler if pod is ready and running", func(containerStatus []k8sv1.ContainerStatus) {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.ContainerStatuses = containerStatus

			addVirtualMachine(vmi)
			addPod(pod)
			if containerStatus[0].Ready {
				addActivePods(vmi, pod.UID, "")
			}

			sanityExecute()
			expectVMIScheduledState(vmi)

			updatedVMI, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedVMI.Status.Interfaces).To(Equal([]virtv1.VirtualMachineInstanceNetworkInterface{{Name: "stubNetStatusUpdate"}}), "Network status update wasn't called")
		},
			Entry("with running compute container and no infra container",
				[]k8sv1.ContainerStatus{{
					Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
				}},
			),
			Entry("with running compute container and no ready istio-proxy container",
				[]k8sv1.ContainerStatus{{
					Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
				}, {Name: "istio-proxy", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}}, Ready: false}},
			),
		)
		DescribeTable("should not hand over pod to virt-handler if pod is ready and running", func(containerStatus []k8sv1.ContainerStatus) {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.ContainerStatuses = containerStatus

			addVirtualMachine(vmi)
			addPod(pod)
			addActivePods(vmi, pod.UID, "")

			sanityExecute()
			expectVMIBeInPhase(vmi.Namespace, vmi.Name, virtv1.Scheduling)
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
				Fields{"Type": Equal(virtv1.VirtualMachineInstanceReady), "Status": Equal(k8sv1.ConditionFalse), "Reason": Equal(virtv1.GuestNotRunningReason)})))
			expectVMITimestamp(vmi.Namespace, vmi.Name, BeEmpty())
		},
			Entry("with not ready infra container and not ready compute container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: false}, {Name: "kubevirt-infra", Ready: false}},
			),
			Entry("with not ready compute container and no infra container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: false}},
			),
		)

		DescribeTable("With a virt-launcher pod and an attachment pod, it", func(attachmentPodPhase k8sv1.PodPhase, expectedPhase virtv1.VirtualMachineInstancePhase) {
			vmi := newPendingVirtualMachine("testvmi")
			pvc := newHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimBound)
			Expect(controller.pvcIndexer.Add(pvc)).To(Succeed())
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dv",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Pending,
				},
			}
			Expect(controller.dataVolumeIndexer.Add(dv)).To(Succeed())
			Expect(controller.pvcIndexer.Add(pvc)).To(Succeed())
			volume := virtv1.Volume{
				Name: "test-dv",
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name:         "test-dv",
						Hotpluggable: true,
					},
				},
			}
			vmi.Spec.Volumes = []virtv1.Volume{volume}
			vmi.Status.Phase = virtv1.Scheduling
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{
				Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
			}}
			attachmentPod := newPodForVirtlauncher(pod, "hp-test", "abcd", attachmentPodPhase)
			attachmentPod.Spec.Volumes = append(attachmentPod.Spec.Volumes, k8sv1.Volume{
				Name: "test-dv",
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "test-dv",
						ReadOnly:  false,
					},
				},
			})
			addVirtualMachine(vmi)
			addPod(pod)
			addPod(attachmentPod)

			sanityExecute()

			switch expectedPhase {
			case virtv1.Scheduled:
				testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
				expectVMIScheduledState(vmi)
			case virtv1.Scheduling:
				expectVMIBeInPhase(vmi.Namespace, vmi.Name, virtv1.Scheduling)
				expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
					Fields{"Type": Equal(virtv1.VirtualMachineInstanceReady)})))
				expectVMITimestamp(vmi.Namespace, vmi.Name, BeEmpty())
			}
		},
			Entry("should hand over pods if both are ready and running", k8sv1.PodRunning, virtv1.Scheduled),
			Entry("should hand over pods even if the attachment pod is not ready and running", k8sv1.PodPending, virtv1.Scheduled),
		)

		It("should ignore migration target pods", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodConditionMissingReason)
			vmi.Status.Phase = virtv1.Running
			vmi.Status.NodeName = "curnode"

			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Spec.NodeName = "curnode"

			failedTargetPod := pod.DeepCopy()
			failedTargetPod.Name = "targetfailure"
			failedTargetPod.UID = "123-123-123-123"
			failedTargetPod.Spec.NodeName = "someothernode"

			failedTargetPod2 := pod.DeepCopy()
			failedTargetPod2.Name = "targetfailure2"
			failedTargetPod2.UID = "111-111-111-111"
			failedTargetPod2.Spec.NodeName = "someothernode"

			addVirtualMachine(vmi)
			addPod(failedTargetPod)
			addPod(pod)
			addPod(failedTargetPod2)

			selectedPod, err := kvcontroller.CurrentVMIPod(vmi, controller.podIndexer)
			Expect(err).ToNot(HaveOccurred())

			Expect(selectedPod.UID).To(Equal(pod.UID))
		})

		It("should set an error condition if deleting the virtual machine pod fails", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.DeletionTimestamp = new(metav1.Now())
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

			// Expect pod delete
			kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()

			testutils.ExpectEvent(recorder, kvcontroller.FailedDeletePodReason)
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
				Fields{
					"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
					"Status": Equal(k8sv1.ConditionFalse),
					"Reason": Equal(kvcontroller.FailedDeletePodReason),
				})),
			)
		})
		It("should set an error condition if when pod cannot guarantee resources when cpu pinning has been requested", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			vmi.Spec.Domain.CPU = &virtv1.CPU{DedicatedCPUPlacement: true}
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Spec = k8sv1.PodSpec{
				Containers: []k8sv1.Container{
					{
						Name: "test",
						Resources: k8sv1.ResourceRequirements{
							Requests: k8sv1.ResourceList{
								k8sv1.ResourceCPU: resource.MustParse("800m"),
							},
						},
					},
				},
			}

			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()

			testutils.ExpectEvent(recorder, kvcontroller.FailedGuaranteePodResourcesReason)
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
				Fields{
					"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
					"Status": Equal(k8sv1.ConditionFalse),
					"Reason": Equal(kvcontroller.FailedGuaranteePodResourcesReason),
				})),
			)
		})
		It("should set an error condition if creating the virtual machine pod fails", func() {
			vmi := newPendingVirtualMachine("testvmi")

			// Expect pod creation
			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			addVirtualMachine(vmi)

			sanityExecute()

			testutils.ExpectEvent(recorder, kvcontroller.FailedCreatePodReason)
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
				Fields{
					"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
					"Status": Equal(k8sv1.ConditionFalse),
					"Reason": Equal(kvcontroller.FailedCreatePodReason),
				})),
			)
		})
		It("should update the virtual machine to scheduled if pod is ready, runnning and handed over to virt-handler", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()
			expectVMIScheduledState(vmi)
		})
		It("should update the virtual machine QOS class if the pod finally has a QOS class assigned", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.QOSClass = k8sv1.PodQOSGuaranteed

			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()

			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(*updatedVmi.Status.QOSClass).To(Equal(k8sv1.PodQOSGuaranteed))
		})
		It("should update the virtual machine to scheduled if pod is ready, triggered by pod change", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			pod := newPodForVirtualMachine(vmi, k8sv1.PodPending)

			addVirtualMachine(vmi)
			addPod(pod)
			addActivePods(vmi, pod.UID, "")

			sanityExecute()

			pod = newPodForVirtualMachine(vmi, k8sv1.PodRunning)

			key, err := kvcontroller.KeyFunc(vmi)
			Expect(err).To(Not(HaveOccurred()))
			mockQueue.Add(key)

			controller.podIndexer.Update(pod)
			controller.podExpectations.SetExpectations(key, 0, 0)

			sanityExecute()
			expectVMIScheduledState(vmi)
		})
		It("should update the virtual machine to failed if pod was not ready, triggered by pod delete", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			pod := newPodForVirtualMachine(vmi, k8sv1.PodPending)
			vmi.Status.Phase = virtv1.Scheduling

			addVirtualMachine(vmi)
			addPod(pod)
			addActivePods(vmi, pod.UID, "")

			sanityExecute()

			key, err := kvcontroller.KeyFunc(vmi)
			Expect(err).To(Not(HaveOccurred()))
			mockQueue.Add(key)

			controller.podIndexer.Delete(pod)

			sanityExecute()
			expectVMIFailedState(vmi)
		})
		It("should update the virtual machine to failed if compute container is terminated while still scheduling", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodTerminatingReason)
			pod := newPodForVirtualMachine(vmi, k8sv1.PodPending)
			pod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{Name: "compute", State: k8sv1.ContainerState{Terminated: &k8sv1.ContainerStateTerminated{}}}}
			vmi.Status.Phase = virtv1.Scheduling

			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()
			expectVMIFailedState(vmi)
		})
		It("should replace the deprecated finalizer with the domain-qualified one during reconciliation", func() {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Running
			vmi.Finalizers = []string{virtv1.DeprecatedVirtualMachineInstanceFinalizer}

			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addActivePods(vmi, pod.UID, "")

			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()

			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVmi.Finalizers).ToNot(ContainElement(virtv1.DeprecatedVirtualMachineInstanceFinalizer))
			Expect(updatedVmi.Finalizers).To(ContainElement(virtv1.VirtualMachineInstanceFinalizer))
		})
		DescribeTable("should remove the fore ground finalizer if no pod is present and the vmi is in ", func(phase virtv1.VirtualMachineInstancePhase, finalizer string) {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = phase
			vmi.Status.LauncherContainerImageVersion = "madeup"
			vmi.Labels = map[string]string{}
			vmi.Labels[virtv1.OutdatedLauncherImageLabel] = ""
			vmi.Finalizers = append(vmi.Finalizers, finalizer)
			Expect(vmi.Finalizers).To(ContainElement(finalizer))

			addVirtualMachine(vmi)

			sanityExecute()

			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVmi.Labels).To(Not(HaveKey(virtv1.OutdatedLauncherImageLabel)))
			Expect(updatedVmi.Status.LauncherContainerImageVersion).To(BeEmpty())
			Expect(updatedVmi.Status.Phase).To(Equal(phase))
			Expect(updatedVmi.Finalizers).ToNot(ContainElement(finalizer))
		},
			Entry("succeeded state with deprecated finalizer", virtv1.Succeeded, virtv1.DeprecatedVirtualMachineInstanceFinalizer),
			Entry("failed state with deprecated finalizer", virtv1.Failed, virtv1.DeprecatedVirtualMachineInstanceFinalizer),
			Entry("succeeded state with domain-qualified finalizer", virtv1.Succeeded, virtv1.VirtualMachineInstanceFinalizer),
			Entry("failed state with domain-qualified finalizer", virtv1.Failed, virtv1.VirtualMachineInstanceFinalizer),
		)

		DescribeTable("VM controller finalizer", func(hasOwner, vmExists, expectFinalizer bool) {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Finalizers = append(
				vmi.Finalizers,
				virtv1.VirtualMachineControllerFinalizer,
				virtv1.DeprecatedVirtualMachineInstanceFinalizer,
				virtv1.VirtualMachineInstanceFinalizer,
			)
			vmi.Status.Phase = virtv1.Succeeded
			Expect(vmi.Finalizers).To(ContainElement(virtv1.VirtualMachineControllerFinalizer))

			vm := watchtesting.VirtualMachineFromVMI(vmi.Name, vmi, true)
			vm.UID = "123"
			if hasOwner {
				t := true
				vmi.OwnerReferences = []metav1.OwnerReference{{
					APIVersion:         virtv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
					Kind:               virtv1.VirtualMachineGroupVersionKind.Kind,
					Name:               vm.ObjectMeta.Name,
					UID:                vm.ObjectMeta.UID,
					Controller:         &t,
					BlockOwnerDeletion: &t,
				}}
			}

			if vmExists {
				controller.vmStore.Add(vm)
			}

			addVirtualMachine(vmi)

			sanityExecute()

			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVmi.Finalizers).ToNot(ContainElement(virtv1.DeprecatedVirtualMachineInstanceFinalizer))
			Expect(updatedVmi.Finalizers).ToNot(ContainElement(virtv1.VirtualMachineInstanceFinalizer))
			if !expectFinalizer {
				Expect(updatedVmi.Finalizers).ToNot(ContainElement(virtv1.VirtualMachineControllerFinalizer))
			} else {
				Expect(updatedVmi.Finalizers).To(ContainElement(virtv1.VirtualMachineControllerFinalizer))
			}
		},
			Entry("should be removed if no owner exists", false, false, false),
			Entry("should be removed if VM owner is set but VM is deleted", false, true, false),
			Entry("should be left if VM owner is present", true, true, true),
		)

		DescribeTable("should do nothing if pod is handed to virt-handler", func(phase k8sv1.PodPhase, expectedVirtActions []expectedAction) {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Scheduled
			pod := newPodForVirtualMachine(vmi, phase)

			unreadyReason := virtv1.GuestNotRunningReason
			if phase == k8sv1.PodSucceeded || phase == k8sv1.PodFailed {
				unreadyReason = virtv1.PodTerminatingReason
			}
			setReadyCondition(vmi, k8sv1.ConditionFalse, unreadyReason)

			addActivePods(vmi, pod.UID, "")
			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()
			Expect(virtClientset.Actions()).To(WithTransform(actionsSummary, ConsistOf(expectedVirtActions)))
			Expect(kubeClient.Actions()).To(HaveLen(1))
			Expect(kubeClient.Actions()[0].GetVerb()).To(Equal("create"))
			Expect(kubeClient.Actions()[0].GetResource().Resource).To(Equal("pods"))
		},
			Entry("and in running state", k8sv1.PodRunning, []expectedAction{
				{Verb: "create", Resource: "virtualmachineinstances"},
			}),
			Entry("and in unknown state", k8sv1.PodUnknown, []expectedAction{
				{Verb: "create", Resource: "virtualmachineinstances"},
			}),
			Entry("and in succeeded state", k8sv1.PodSucceeded, []expectedAction{
				{Verb: "create", Resource: "virtualmachineinstances"},
				{Verb: "patch", Resource: "virtualmachineinstances"},
			}),
			Entry("and in failed state", k8sv1.PodFailed, []expectedAction{
				{Verb: "create", Resource: "virtualmachineinstances"},
				{Verb: "patch", Resource: "virtualmachineinstances"},
			}),
			Entry("and in pending state", k8sv1.PodPending, []expectedAction{
				{Verb: "create", Resource: "virtualmachineinstances"},
			}),
		)

		It("should add outdated label if pod's image is outdated and VMI is in running state", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionTrue, "")
			vmi.Status.Phase = virtv1.Running
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})

			pod.Spec.Containers = append(pod.Spec.Containers, k8sv1.Container{
				Image: "madeup",
				Name:  "compute",
			})

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			addPod(pod)

			sanityExecute()

			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVmi.Status.LauncherContainerImageVersion).To(BeEquivalentTo("madeup"))
			Expect(updatedVmi.Labels).To(HaveKeyWithValue(virtv1.OutdatedLauncherImageLabel, ""))

			updatedPod, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedPod.Labels).To(HaveKeyWithValue("kubevirt.io", "virt-launcher"))
			Expect(updatedPod.Labels).To(HaveKeyWithValue("kubevirt.io/created-by", "1234"))
			Expect(updatedPod.Labels).To(HaveKeyWithValue(virtv1.OutdatedLauncherImageLabel, ""))
		})
		It("should remove outdated label if pod's image up-to-date and VMI is in running state", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionTrue, "")
			vmi.Status.Phase = virtv1.Running
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})

			if vmi.Labels == nil {
				vmi.Labels = make(map[string]string)
			}

			vmi.Labels[virtv1.OutdatedLauncherImageLabel] = ""

			pod.Spec.Containers = append(pod.Spec.Containers, k8sv1.Container{
				Image: controller.templateService.GetLauncherImage(),
				Name:  "compute",
			})

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			addPod(pod)

			sanityExecute()

			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVmi.Status.LauncherContainerImageVersion).To(BeEquivalentTo("a"))
			Expect(updatedVmi.Labels).To(BeEmpty())
		})

		It("should add a ready condition if it is present on the pod and the VMI is in running state", func() {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Conditions = nil
			vmi.Status.Phase = virtv1.Running
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			addPod(pod)

			sanityExecute()
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name,
				ContainElement(
					MatchFields(IgnoreExtras, Fields{
						"Type":   BeEquivalentTo(virtv1.VirtualMachineInstanceReady),
						"Status": BeEquivalentTo(k8sv1.ConditionTrue),
					}),
				))
		})

		It("should indicate on the ready condition if the pod is terminating", func() {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Conditions = nil
			vmi.Status.Phase = virtv1.Running
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.DeletionTimestamp = new(metav1.Now())
			pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			addPod(pod)

			sanityExecute()
			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVmi.Status.Conditions).To(HaveLen(1))
			cond := updatedVmi.Status.Conditions[0]
			Expect(cond.Reason).To(Equal(virtv1.PodTerminatingReason))
			Expect(cond.Type).To(Equal(virtv1.VirtualMachineInstanceReady))
			Expect(cond.Status).To(Equal(k8sv1.ConditionFalse))
		})

		It("should add active pods to status if VMI is in running state", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodConditionMissingReason)
			vmi.Status.Phase = virtv1.Running
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.UID = "someUID"
			pod.Spec.NodeName = "someHost"

			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()
			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVmi.Status.ActivePods).To(HaveLen(1))
			Expect(updatedVmi.Status.ActivePods).To(HaveKeyWithValue(types.UID("someUID"), "someHost"))
		})

		It("should not remove sync conditions from virt-handler if it is in scheduled state", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduled
			vmi.Status.Conditions = append(vmi.Status.Conditions,
				virtv1.VirtualMachineInstanceCondition{Type: virtv1.VirtualMachineInstanceSynchronized, Status: k8sv1.ConditionFalse})
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})
			addActivePods(vmi, pod.UID, "")

			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
				Fields{"Type": Equal(virtv1.VirtualMachineInstanceSynchronized), "Status": Equal(k8sv1.ConditionFalse)})))
		})

		DescribeTable("should mark the vmi as Failed if launcher-pod does not exist", func(phase virtv1.VirtualMachineInstancePhase) {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = phase

			addVirtualMachine(vmi)

			sanityExecute()
			expectVMIBeInPhase(vmi.Namespace, vmi.Name, virtv1.Failed)
		},
			Entry("and the vmi is in running state", virtv1.Running),
			Entry("and the vmi is in scheduled state", virtv1.Scheduled),
		)

		DescribeTable("should move the vmi to failed if pod is not handed over", func(phase k8sv1.PodPhase) {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodTerminatingReason)
			vmi.Status.Phase = virtv1.Scheduling
			pod := newPodForVirtualMachine(vmi, phase)

			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()
			expectVMIFailedState(vmi)
		},
			Entry("and in succeeded state", k8sv1.PodSucceeded),
			Entry("and in failed state", k8sv1.PodFailed),
		)

		DescribeTable("should set a Synchronized=False condition when pod runs into image pull errors", func(initContainer bool, containerWaitingReason string) {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Scheduling
			pod := newPodForVirtualMachine(vmi, k8sv1.PodPending)

			containerStatus := &pod.Status.ContainerStatuses[0]
			if initContainer {
				pod.Status.InitContainerStatuses = []k8sv1.ContainerStatus{{}}
				containerStatus = &pod.Status.InitContainerStatuses[0]
			}
			containerStatus.State.Waiting = &k8sv1.ContainerStateWaiting{
				Reason: containerWaitingReason,
			}

			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()
			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
				Fields{
					"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
					"Status": Equal(k8sv1.ConditionFalse),
					"Reason": Equal(containerWaitingReason),
				})),
			)
		},
			Entry("ErrImagePull in init container", true, kvcontroller.ErrImagePullReason),
			Entry("ImagePullBackOff in init container", true, kvcontroller.ImagePullBackOffReason),
			Entry("InvalidImageName in init container", true, kvcontroller.InvalidImageNameReason),
			Entry("ErrImagePull in compute container", false, kvcontroller.ErrImagePullReason),
			Entry("ImagePullBackOff in compute container", false, kvcontroller.ImagePullBackOffReason),
			Entry("InvalidImageName in compute container", false, kvcontroller.InvalidImageNameReason),
		)

		DescribeTable("should override Synchronized=False condition reason when it's already set", func(prevReason, newReason string) {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Scheduling
			pod := newPodForVirtualMachine(vmi, k8sv1.PodPending)

			vmi.Status.Conditions = []virtv1.VirtualMachineInstanceCondition{
				{
					Type:   virtv1.VirtualMachineInstanceSynchronized,
					Status: k8sv1.ConditionFalse,
					Reason: prevReason,
				},
			}

			pod.Status.InitContainerStatuses = []k8sv1.ContainerStatus{{
				State: k8sv1.ContainerState{
					Waiting: &k8sv1.ContainerStateWaiting{
						Reason: newReason,
					},
				},
			}}

			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()

			expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
				Fields{
					"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
					"Status": Equal(k8sv1.ConditionFalse),
					"Reason": Equal(newReason),
				})),
			)
		},
			Entry("ErrImagePull --> ImagePullBackOff", kvcontroller.ErrImagePullReason, kvcontroller.ImagePullBackOffReason),
			Entry("ImagePullBackOff --> ErrImagePull", kvcontroller.ImagePullBackOffReason, kvcontroller.ErrImagePullReason),
		)
		It("should add MigrationTransport to VMI status", func() {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Scheduling
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			addPod(pod)
			addActivePods(vmi, pod.UID, "")

			sanityExecute()
			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVmi.Status.MigrationTransport).To(Equal(virtv1.MigrationTransportUnix))
		})

		Context("should update pod annotations and labels", func() {
			var (
				initialAdditionalLauncherAnnotationsSync []string
				initialAdditionalLauncherLabelsSync      []string
			)

			BeforeEach(func() {
				initialAdditionalLauncherAnnotationsSync = controller.additionalLauncherAnnotationsSync
				initialAdditionalLauncherLabelsSync = controller.additionalLauncherLabelsSync
			})

			AfterEach(func() {
				controller.additionalLauncherAnnotationsSync = initialAdditionalLauncherAnnotationsSync
				controller.additionalLauncherLabelsSync = initialAdditionalLauncherLabelsSync
			})

			type testData struct {
				vmiAnnotations                    map[string]string
				podAnnotations                    map[string]string
				vmiLabels                         map[string]string
				podLabels                         map[string]string
				expectedPatch                     bool
				expectedAnnotations               map[string]string
				expectedLabels                    map[string]string
				additionalLauncherAnnotationsSync []string
				additionalLauncherLabelsSync      []string
			}
			const (
				wildcardAnnotationPrefix = "custom/wildcard-annotation/"
				wildcardLabelPrefix      = "custom/wildcard-label/"
			)
			DescribeTable("when VMI dynamic annotations and label sets changes", func(td *testData) {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running

				controller.additionalLauncherAnnotationsSync = td.additionalLauncherAnnotationsSync
				controller.additionalLauncherLabelsSync = td.additionalLauncherLabelsSync

				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

				vmi.Labels = td.vmiLabels
				maps.Copy(vmi.Annotations, td.vmiAnnotations)
				maps.Copy(pod.Annotations, td.podAnnotations)
				maps.Copy(pod.Labels, td.podLabels)

				addVirtualMachine(vmi)
				addActivePods(vmi, pod.UID, "")
				addPod(pod)

				sanityExecute()

				updatedPod, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedPod.Annotations).To(BeEquivalentTo(td.expectedAnnotations))
				Expect(updatedPod.Labels).To(BeEquivalentTo(td.expectedLabels))
				if td.expectedPatch {
					Expect(kubeClient.Actions()).To(HaveLen(3)) // 0: create, 1: patch, 2: get
					Expect(kubeClient.Actions()[1].GetVerb()).To(Equal("patch"))
				} else {
					Expect(kubeClient.Actions()).To(HaveLen(2)) // 0: create, 1: get
				}
			},
				Entry("when VMI and pod labels differ",
					&testData{
						vmiLabels: map[string]string{
							virtv1.NodeNameLabel: "node2",
						},
						podLabels: map[string]string{
							virtv1.NodeNameLabel: "node1",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
							virtv1.NodeNameLabel:     "node2",
						},
						expectedPatch: true,
					},
				),
				Entry("when VMI and pod label are the same",
					&testData{
						vmiLabels: map[string]string{
							virtv1.NodeNameLabel: "node1",
						},
						podLabels: map[string]string{
							virtv1.NodeNameLabel: "node1",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
							virtv1.NodeNameLabel:     "node1",
						},
						expectedPatch: false,
					},
				),
				Entry("when POD label doesn't exist",
					&testData{
						vmiLabels: map[string]string{
							virtv1.NodeNameLabel: "node1",
						},
						podLabels: map[string]string{},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
							virtv1.NodeNameLabel:     "node1",
						},
						expectedPatch: true,
					},
				),
				Entry("when neither POD or VMI annotations and label exists",
					&testData{
						vmiLabels: map[string]string{},
						podLabels: map[string]string{},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
						},
						expectedPatch: false,
					},
				),
				Entry("when POD label exists and VMI does not",
					&testData{
						vmiLabels: map[string]string{},
						podLabels: map[string]string{
							virtv1.OutdatedLauncherImageLabel: "",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
						},
						expectedPatch: true,
					},
				),
				Entry("when VMI and pod annotations differ",
					&testData{
						vmiAnnotations: map[string]string{
							descheduler.EvictPodAnnotationKeyAlphaPreferNoEviction: "false",
						},
						podAnnotations: map[string]string{
							descheduler.EvictPodAnnotationKeyAlphaPreferNoEviction: "true",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":                                   "testvmi",
							descheduler.EvictOnlyAnnotation:                        "",
							descheduler.EvictPodAnnotationKeyAlphaPreferNoEviction: "false",
						},
						expectedPatch: true,
					},
				),
				Entry("when VMI and pod annotations are the same",
					&testData{
						vmiAnnotations: map[string]string{
							descheduler.EvictPodAnnotationKeyAlphaPreferNoEviction: "false",
						},
						podAnnotations: map[string]string{
							descheduler.EvictPodAnnotationKeyAlphaPreferNoEviction: "false",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":                                   "testvmi",
							descheduler.EvictOnlyAnnotation:                        "",
							descheduler.EvictPodAnnotationKeyAlphaPreferNoEviction: "false",
						},
						expectedPatch: false,
					},
				),
				Entry("when POD annotation doesn't exist",
					&testData{
						vmiAnnotations: map[string]string{
							descheduler.EvictPodAnnotationKeyAlphaPreferNoEviction: "false",
						},
						podAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":                                   "testvmi",
							descheduler.EvictOnlyAnnotation:                        "",
							descheduler.EvictPodAnnotationKeyAlphaPreferNoEviction: "false",
						},
						expectedPatch: true,
					},
				),
				Entry("when POD annotation exists and VMI does not",
					&testData{
						vmiAnnotations: map[string]string{},
						podAnnotations: map[string]string{
							"kubevirt.io/domain":                                   "testvmi",
							descheduler.EvictOnlyAnnotation:                        "",
							descheduler.EvictPodAnnotationKeyAlphaPreferNoEviction: "false",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedPatch: true,
					},
				),
				Entry("when both annotations and labels differ between VMI and pod",
					&testData{
						vmiAnnotations: map[string]string{
							descheduler.EvictPodAnnotationKeyAlphaPreferNoEviction: "false",
						},
						podAnnotations: map[string]string{
							descheduler.EvictPodAnnotationKeyAlphaPreferNoEviction: "true",
						},
						vmiLabels: map[string]string{
							virtv1.NodeNameLabel: "node2",
						},
						podLabels: map[string]string{
							virtv1.NodeNameLabel: "node1",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
							virtv1.NodeNameLabel:     "node2",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":                                   "testvmi",
							descheduler.EvictOnlyAnnotation:                        "",
							descheduler.EvictPodAnnotationKeyAlphaPreferNoEviction: "false",
						},
						expectedPatch: true,
					},
				),
				Entry("when VMI and pod custom annotations differ",
					&testData{
						vmiAnnotations: map[string]string{
							"custom/annotation": "false",
						},
						podAnnotations: map[string]string{
							"custom/annotation": "true",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
							"custom/annotation":             "false",
						},
						expectedPatch:                     true,
						additionalLauncherAnnotationsSync: []string{"custom/annotation"},
					},
				),
				Entry("when VMI and pod custom labels differ",
					&testData{
						vmiLabels: map[string]string{
							"custom/label": "node2",
						},
						podLabels: map[string]string{
							"custom/label": "node1",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
							"custom/label":           "node2",
						},
						expectedPatch:                true,
						additionalLauncherLabelsSync: []string{"custom/label"},
					},
				),
				Entry("when VMI and pod wildcard annotations differ",
					&testData{
						vmiAnnotations: map[string]string{
							wildcardAnnotationPrefix + "one": "value-one",
							wildcardAnnotationPrefix + "two": "value-two",
						},
						podAnnotations: map[string]string{
							wildcardAnnotationPrefix + "one": "stale",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":             "testvmi",
							descheduler.EvictOnlyAnnotation:  "",
							wildcardAnnotationPrefix + "one": "value-one",
							wildcardAnnotationPrefix + "two": "value-two",
						},
						expectedPatch:                     true,
						additionalLauncherAnnotationsSync: []string{wildcardAnnotationPrefix + "*"},
					},
				),
				Entry("when pod has wildcard annotations missing on VMI",
					&testData{
						vmiAnnotations: map[string]string{
							"kubevirt.io/domain": "testvmi",
						},
						podAnnotations: map[string]string{
							"kubevirt.io/domain":             "testvmi",
							descheduler.EvictOnlyAnnotation:  "",
							wildcardAnnotationPrefix + "one": "stale",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedPatch:                     true,
						additionalLauncherAnnotationsSync: []string{wildcardAnnotationPrefix + "*"},
					},
				),
				Entry("when VMI and pod wildcard labels differ",
					&testData{
						vmiLabels: map[string]string{
							wildcardLabelPrefix + "one": "nodeA",
							wildcardLabelPrefix + "two": "nodeB",
						},
						podLabels: map[string]string{
							wildcardLabelPrefix + "one": "nodeOld",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":               "virt-launcher",
							"kubevirt.io/created-by":    "1234",
							wildcardLabelPrefix + "one": "nodeA",
							wildcardLabelPrefix + "two": "nodeB",
						},
						expectedPatch:                true,
						additionalLauncherLabelsSync: []string{wildcardLabelPrefix + "*"},
					},
				),
				Entry("when pod has wildcard labels missing on VMI",
					&testData{
						vmiLabels: map[string]string{
							"kubevirt.io":        "virt-launcher",
							virtv1.NodeNameLabel: "node1",
						},
						podLabels: map[string]string{
							"kubevirt.io":               "virt-launcher",
							wildcardLabelPrefix + "one": "nodeOld",
							virtv1.NodeNameLabel:        "node1",
							"kubevirt.io/created-by":    "1234",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
							virtv1.NodeNameLabel:     "node1",
						},
						expectedPatch:                true,
						additionalLauncherLabelsSync: []string{wildcardLabelPrefix + "*"},
					},
				),
				Entry("when bare '*' annotation pattern is used it should be ignored",
					&testData{
						vmiAnnotations: map[string]string{
							"custom/something": "value",
						},
						podAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
						},
						expectedPatch:                     false,
						additionalLauncherAnnotationsSync: []string{"*"},
					},
				),
				Entry("when bare '*' label pattern is used it should be ignored",
					&testData{
						vmiLabels: map[string]string{
							"kubevirt.io":        "virt-launcher",
							"custom/some-label":  "value",
							virtv1.NodeNameLabel: "node1",
						},
						podLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							virtv1.NodeNameLabel:     "node1",
							"kubevirt.io/created-by": "1234",
						},
						expectedAnnotations: map[string]string{
							"kubevirt.io/domain":            "testvmi",
							descheduler.EvictOnlyAnnotation: "",
						},
						expectedLabels: map[string]string{
							"kubevirt.io":            "virt-launcher",
							"kubevirt.io/created-by": "1234",
							virtv1.NodeNameLabel:     "node1",
						},
						expectedPatch:                false,
						additionalLauncherLabelsSync: []string{"*"},
					},
				),
			)
		})

		Context("Velero backup hook annotations", func() {
			It("should remove Velero annotations when skip annotation is added to VMI", func() {
				controller.storageAnnotationsGenerator = storageannotations.NewGenerator(config)

				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				// Start without skip annotation so Velero annotations are generated
				delete(vmi.Annotations, velero.SkipHooksAnnotation)

				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				// Start with Velero annotations on pod
				pod.Annotations[velero.PreBackupHookContainerAnnotation] = "compute"
				pod.Annotations[velero.PreBackupHookCommandAnnotation] = `["/usr/bin/virt-freezer", "--freeze", "--name", "testvmi", "--namespace", "default"]`
				pod.Annotations[velero.PreBackupHookTimeoutAnnotation] = "60s"
				pod.Annotations[velero.PostBackupHookContainerAnnotation] = "compute"
				pod.Annotations[velero.PostBackupHookCommandAnnotation] = `["/usr/bin/virt-freezer", "--unfreeze", "--name", "testvmi", "--namespace", "default"]`
				addVirtualMachine(vmi)
				addActivePods(vmi, pod.UID, "")
				addPod(pod)

				// Add skip annotation to VMI
				vmi.Annotations[velero.SkipHooksAnnotation] = "true"
				controller.vmiIndexer.Update(vmi)

				sanityExecute()

				updatedPod, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedPod.Annotations).ToNot(HaveKey(velero.PreBackupHookContainerAnnotation))
				Expect(updatedPod.Annotations).ToNot(HaveKey(velero.PreBackupHookCommandAnnotation))
				Expect(updatedPod.Annotations).ToNot(HaveKey(velero.PreBackupHookTimeoutAnnotation))
				Expect(updatedPod.Annotations).ToNot(HaveKey(velero.PostBackupHookContainerAnnotation))
				Expect(updatedPod.Annotations).ToNot(HaveKey(velero.PostBackupHookCommandAnnotation))
				// Other annotations should remain
				Expect(updatedPod.Annotations).To(HaveKeyWithValue("kubevirt.io/domain", "testvmi"))
			})

			It("should add Velero annotations when skip annotation is removed from VMI", func() {
				controller.storageAnnotationsGenerator = storageannotations.NewGenerator(config)

				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				vmi.Annotations[velero.SkipHooksAnnotation] = "true"

				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

				addVirtualMachine(vmi)
				addActivePods(vmi, pod.UID, "")
				addPod(pod)

				// Remove skip annotation from VMI
				delete(vmi.Annotations, velero.SkipHooksAnnotation)
				controller.vmiIndexer.Update(vmi)

				sanityExecute()

				updatedPod, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedPod.Annotations).To(HaveKeyWithValue(velero.PreBackupHookContainerAnnotation, "compute"))
				Expect(updatedPod.Annotations).To(HaveKey(velero.PreBackupHookCommandAnnotation))
				Expect(updatedPod.Annotations).To(HaveKeyWithValue(velero.PreBackupHookTimeoutAnnotation, "60s"))
				Expect(updatedPod.Annotations).To(HaveKeyWithValue(velero.PostBackupHookContainerAnnotation, "compute"))
				Expect(updatedPod.Annotations).To(HaveKey(velero.PostBackupHookCommandAnnotation))
				Expect(updatedPod.Annotations).To(HaveKeyWithValue(velero.PostBackupHookTimeoutAnnotation, "60s"))
			})

			It("should not patch pod when Velero annotations are already in sync", func() {
				controller.storageAnnotationsGenerator = storageannotations.NewGenerator(config)

				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				// Remove skip annotation so Velero annotations are generated
				delete(vmi.Annotations, velero.SkipHooksAnnotation)

				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				// Pod already has the expected Velero annotations
				pod.Annotations[velero.PreBackupHookContainerAnnotation] = "compute"
				pod.Annotations[velero.PreBackupHookCommandAnnotation] = `["/usr/bin/virt-freezer", "--freeze", "--name", "testvmi", "--namespace", "default"]`
				pod.Annotations[velero.PreBackupHookTimeoutAnnotation] = "60s"
				pod.Annotations[velero.PostBackupHookContainerAnnotation] = "compute"
				pod.Annotations[velero.PostBackupHookCommandAnnotation] = `["/usr/bin/virt-freezer", "--unfreeze", "--name", "testvmi", "--namespace", "default"]`
				pod.Annotations[velero.PostBackupHookTimeoutAnnotation] = "60s"

				addVirtualMachine(vmi)
				addActivePods(vmi, pod.UID, "")
				addPod(pod)

				sanityExecute()

				// Verify no patch action occurred
				patchActions := 0
				for _, action := range kubeClient.Actions() {
					if action.GetVerb() == "patch" {
						patchActions++
					}
				}
				Expect(patchActions).To(Equal(0), "Expected no patch actions but found %d", patchActions)
			})
		})

		Context("Descheduler annotations", func() {
			It("should add eviction-in-progress annotation in case of VMI marked for eviction", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				vmi.Status.EvacuationNodeName = "test"
				vmi.Status.NodeName = "test"

				sourcePod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				sourcePod.Spec.NodeName = "test"
				addActivePods(vmi, sourcePod.UID, "")

				targetPod := sourcePod.DeepCopy()
				targetPod.Name = "targetfailure"
				targetPod.UID = "123-123-123-123"
				targetPod.Spec.NodeName = "someothernode"

				Expect(sourcePod.Annotations).ToNot(HaveKey(descheduler.EvictionInProgressAnnotation))
				Expect(targetPod.Annotations).ToNot(HaveKey(descheduler.EvictionInProgressAnnotation))

				addVirtualMachine(vmi)
				addPod(sourcePod)
				addPod(targetPod)

				sanityExecute()

				By("Checking that the source/active pod is correctly annotated")
				updatedSourcePod, err := kubeClient.CoreV1().Pods(sourcePod.Namespace).Get(context.Background(), sourcePod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedSourcePod.Annotations).To(HaveKey(descheduler.EvictionInProgressAnnotation))

				By("Checking that a migration target pod is not annotated to prevent double counting")
				updatedTargetPod, err := kubeClient.CoreV1().Pods(targetPod.Namespace).Get(context.Background(), targetPod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedTargetPod.Annotations).ToNot(HaveKey(descheduler.EvictionInProgressAnnotation))
			})

			It("should remove eviction-in-progress annotation in case of VMI not marked for eviction", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				vmi.Status.EvacuationNodeName = ""

				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				pod.ObjectMeta.Annotations[descheduler.EvictionInProgressAnnotation] = "kubevirt"
				addActivePods(vmi, pod.UID, "")
				addVirtualMachine(vmi)
				addPod(pod)

				sanityExecute()

				updatedPod, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedPod.Annotations).ToNot(HaveKey(descheduler.EvictionInProgressAnnotation))
			})
		})

		Context("Eviction condition", func() {
			It("should set VirtualMachineInstanceEvictionRequested condition when VMI is marked for eviction", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				vmi.Status.EvacuationNodeName = "test"
				vmi.Status.NodeName = "test"

				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				addActivePods(vmi, pod.UID, "")
				addVirtualMachine(vmi)
				addPod(pod)

				sanityExecute()

				updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi).To(matcher.HaveConditionTrueWithReason(virtv1.VirtualMachineInstanceEvictionRequested, virtv1.VirtualMachineInstanceReasonEvictionRequested))
			})

			It("should remove VirtualMachineInstanceEvictionRequested condition when VMI is not marked for eviction", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				vmi.Status.EvacuationNodeName = ""

				// Add the condition first
				kvcontroller.NewVirtualMachineInstanceConditionManager().UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
					Type:    virtv1.VirtualMachineInstanceEvictionRequested,
					Status:  k8sv1.ConditionTrue,
					Reason:  virtv1.VirtualMachineInstanceReasonEvictionRequested,
					Message: "VMI is marked for eviction",
				})

				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				addActivePods(vmi, pod.UID, "")
				addVirtualMachine(vmi)
				addPod(pod)

				sanityExecute()

				updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi).To(matcher.HaveConditionMissingOrFalse(virtv1.VirtualMachineInstanceEvictionRequested))
			})

			It("should not set VirtualMachineInstanceEvictionRequested condition when VMI is not marked for eviction", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				vmi.Status.EvacuationNodeName = ""

				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				addActivePods(vmi, pod.UID, "")
				addVirtualMachine(vmi)
				addPod(pod)

				sanityExecute()

				updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi).To(matcher.HaveConditionMissingOrFalse(virtv1.VirtualMachineInstanceEvictionRequested))
			})
		})

		DescribeTable("should set VirtualMachineUnpaused=False pod condition when VMI is paused", func(currUnpausedStatus k8sv1.ConditionStatus) {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Running
			kvcontroller.NewVirtualMachineInstanceConditionManager().UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
				Type:   virtv1.VirtualMachineInstancePaused,
				Status: k8sv1.ConditionTrue,
			})

			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addActivePods(vmi, pod.UID, "")
			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()

			updatedPod, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedPod.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
				Fields{
					"Type":   Equal(virtv1.VirtualMachineUnpaused),
					"Status": Equal(k8sv1.ConditionFalse),
				})),
			)
		},
			Entry("when VirtualMachineUnpaused=True", k8sv1.ConditionTrue),
			Entry("when VirtualMachineUnpaused condition is unset", k8sv1.ConditionUnknown),
		)

		DescribeTable("should set VirtualMachineUnpaused=True pod condition when VMI is not paused", func(currUnpausedStatus k8sv1.ConditionStatus) {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Running
			kvcontroller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, virtv1.VirtualMachineInstancePaused)

			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addActivePods(vmi, pod.UID, "")
			addVirtualMachine(vmi)
			addPod(pod)

			sanityExecute()

			updatedPod, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedPod.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
				Fields{
					"Type":   Equal(virtv1.VirtualMachineUnpaused),
					"Status": Equal(k8sv1.ConditionTrue),
				})),
			)
		},
			Entry("when VirtualMachineUnpaused=True", k8sv1.ConditionTrue),
			Entry("when VirtualMachineUnpaused condition is unset", k8sv1.ConditionUnknown),
		)

		Context("with memory hotplug enabled", func() {
			It("should add MemoryChange condition when guest memory changes", func() {
				currentGuestMemory := resource.MustParse("128Mi")
				requestedGuestMemory := resource.MustParse("512Mi")

				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				vmi.Status.Memory = &virtv1.MemoryStatus{
					GuestAtBoot:    &currentGuestMemory,
					GuestCurrent:   &currentGuestMemory,
					GuestRequested: &currentGuestMemory,
				}
				vmi.Spec.Domain.Memory = &virtv1.Memory{
					Guest:    &requestedGuestMemory,
					MaxGuest: &requestedGuestMemory,
				}

				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				addActivePods(vmi, pod.UID, "")

				addVirtualMachine(vmi)
				addPod(pod)

				sanityExecute()
				expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Type":   BeEquivalentTo(virtv1.VirtualMachineInstanceMemoryChange),
						"Status": Equal(k8sv1.ConditionTrue),
					})),
				)
			})

			It("should store guestMemoryOverheadRatio if used during memory hotplug", func() {
				currentGuestMemory := resource.MustParse("128Mi")
				requestedGuestMemory := resource.MustParse("512Mi")

				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				vmi.Status.Memory = &virtv1.MemoryStatus{
					GuestCurrent: &currentGuestMemory,
				}
				vmi.Spec.Domain.Memory = &virtv1.Memory{
					Guest:    &requestedGuestMemory,
					MaxGuest: &requestedGuestMemory,
				}
				kvCR := testutils.GetFakeKubeVirtClusterConfig(kvStore)
				overheadRatio := "2"
				kvCR.Spec.Configuration.AdditionalGuestMemoryOverheadRatio = &overheadRatio
				testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvCR)

				controller.syncMemoryHotplug(vmi)

				Expect(vmi.Labels).To(HaveKeyWithValue(virtv1.MemoryHotplugOverheadRatioLabel, overheadRatio))
			})

			It("should update memory overhead in VMI status from pod annotation", func() {
				memoryQuantity := resource.NewQuantity(256*1024*1024, resource.BinarySI) // 256Mi
				memory := strconv.FormatInt(memoryQuantity.Value(), 10)

				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				pod.Annotations[virtv1.MemoryOverheadAnnotationBytes] = memory

				controller.updateMemoryOverheadStatusFromPod(vmi, pod)

				Expect(vmi.Status.Memory).ToNot(BeNil(), "Memory status should be populated")
				Expect(vmi.Status.Memory.MemoryOverhead).ToNot(BeNil(), "MemoryOverhead should be set")
				Expect(vmi.Status.Memory.MemoryOverhead.Value()).To(Equal(memoryQuantity.Value()), "MemoryOverhead should match annotation value")
			})

			It("should not update memory overhead if pod annotation is missing", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				delete(pod.Annotations, virtv1.MemoryOverheadAnnotationBytes)

				controller.updateMemoryOverheadStatusFromPod(vmi, pod)

				Expect(vmi.Status.Memory).To(BeNil(), "Memory status should remain nil")
			})
		})
	})

	Context("topology hints", func() {

		getVmiWithInvTsc := func() *virtv1.VirtualMachineInstance {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Spec.Architecture = "amd64"
			vmi.Spec.Domain.CPU = &virtv1.CPU{
				Features: []virtv1.CPUFeature{
					{
						Name:   "invtsc",
						Policy: "require",
					},
				},
			}

			return vmi
		}

		getVmiWithReenlightenment := func() *virtv1.VirtualMachineInstance {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Spec.Architecture = "amd64"
			vmi.Spec.Domain.Features = &virtv1.Features{
				Hyperv: &virtv1.FeatureHyperv{
					Reenlightenment: &virtv1.FeatureState{Enabled: new(true)},
				},
			}

			return vmi
		}

		expectTopologyHintsDefined := func(vmi *virtv1.VirtualMachineInstance, value gomegaTypes.GomegaMatcher) {
			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(topology.AreTSCFrequencyTopologyHintsDefined(updatedVmi)).To(value)
		}

		Context("needs to be set when", func() {
			It("invtsc feature exists", func() {
				vmi := getVmiWithInvTsc()
				addVirtualMachine(vmi)
				sanityExecute()
				expectTopologyHintsDefined(vmi, BeTrue())
			})

			Context("HyperV reenlightenment is enabled", func() {
				var vmi *virtv1.VirtualMachineInstance

				BeforeEach(func() {
					vmi = getVmiWithReenlightenment()
				})

				When("TSC frequency is exposed", func() {
					It("topology hints need to be set", func() {
						addVirtualMachine(vmi)
						sanityExecute()
						expectTopologyHintsDefined(vmi, BeTrue())
					})
				})

				When("TSC frequency is not exposed", func() {
					unexposeTscFrequency := func(requirement topology.TscFrequencyRequirementType) {
						mockHinter := topology.NewMockHinter(gomock.NewController(GinkgoT()))
						mockHinter.EXPECT().TopologyHintsForVMI(gomock.Any()).Return(nil, requirement, fmt.Errorf("tsc frequency is not exposed on the cluster")).AnyTimes()
						mockHinter.EXPECT().IsTscFrequencyRequired(gomock.Any()).Return(requirement == topology.RequiredForBoot)
						controller.topologyHinter = mockHinter
					}

					It("topology hints don't need to be set and VMI needs to be non-migratable", func() {
						unexposeTscFrequency(topology.RequiredForMigration)
						// Make sure no update occurs
						addVirtualMachine(vmi)
						sanityExecute()
						expectTopologyHintsDefined(vmi, BeFalse())
						expectMatchingPodCreation(vmi)
						testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
					})
				})

			})

		})

		Context("pod creation", func() {

			It("does not need to happen if tsc requirement is of type RequiredForBoot", func() {
				vmi := getVmiWithInvTsc()
				Expect(topology.GetTscFrequencyRequirement(vmi).Type).To(Equal(topology.RequiredForBoot))

				addVirtualMachine(vmi)
				sanityExecute()
				expectTopologyHintsDefined(vmi, BeTrue())
			})

			It("does not need to happen if tsc requirement is of type RequiredForMigration", func() {
				vmi := getVmiWithReenlightenment()
				Expect(topology.GetTscFrequencyRequirement(vmi).Type).To(Equal(topology.RequiredForMigration))
				addVirtualMachine(vmi)

				sanityExecute()
				expectTopologyHintsDefined(vmi, BeTrue())
			})

			It("does not need to happen if tsc requirement is of type NotRequired", func() {
				vmi := newPendingVirtualMachine("testvmi")
				Expect(topology.GetTscFrequencyRequirement(vmi).Type).To(Equal(topology.NotRequired))
				addVirtualMachine(vmi)

				sanityExecute()

				testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
				expectMatchingPodCreation(vmi)
			})
		})

		Context("decentralized live migration", func() {
			It("should set the topology hints when the VMI is created", func() {
				vmi := getVmiWithReenlightenment()
				vmi.Annotations[virtv1.CreateMigrationTarget] = "true"
				addVirtualMachine(vmi)
				sanityExecute()
				expectTopologyHintsDefined(vmi, BeTrue())
				updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi.Status.Phase).To(Equal(virtv1.WaitingForSync))
			})

			DescribeTable("should transition migration target to WaitingForSync when pod is down", func(vmiPhase virtv1.VirtualMachineInstancePhase, podPhase k8sv1.PodPhase) {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = vmiPhase
				vmi.Status.NodeName = "targetnode"
				if vmi.Annotations == nil {
					vmi.Annotations = make(map[string]string)
				}
				vmi.Annotations[virtv1.CreateMigrationTarget] = "true"
				vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
					TargetNode: "targetnode",
					SourceNode: "sourcenode",
				}

				pod := newPodForVirtualMachine(vmi, podPhase)
				pod.Spec.NodeName = "targetnode"

				addActivePods(vmi, pod.UID, "targetnode")
				addVirtualMachine(vmi)
				addPod(pod)

				sanityExecute()

				updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi.Status.Phase).To(Equal(virtv1.WaitingForSync))
			},
				Entry("Scheduled VMI with Failed pod", virtv1.Scheduled, k8sv1.PodFailed),
				Entry("Scheduled VMI with Succeeded pod", virtv1.Scheduled, k8sv1.PodSucceeded),
			)

			It("should transition scheduled migration target to WaitingForSync when migration failed even if pod is running", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Scheduled
				vmi.Status.NodeName = "targetnode"
				if vmi.Annotations == nil {
					vmi.Annotations = make(map[string]string)
				}
				vmi.Annotations[virtv1.CreateMigrationTarget] = "true"
				vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
					TargetNode: "targetnode",
					SourceNode: "sourcenode",
					Failed:     true,
					Completed:  true,
				}

				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				pod.Spec.NodeName = "targetnode"

				addVirtualMachine(vmi)
				addActivePods(vmi, pod.UID, "targetnode")
				addPod(pod)

				sanityExecute()

				updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi.Status.Phase).To(Equal(virtv1.WaitingForSync))
			})

			DescribeTable("should keep migration target in WaitingForSync when pod is down", func(podPhase k8sv1.PodPhase) {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.WaitingForSync
				vmi.Status.NodeName = "targetnode"
				if vmi.Annotations == nil {
					vmi.Annotations = make(map[string]string)
				}
				vmi.Annotations[virtv1.CreateMigrationTarget] = "true"

				pod := newPodForVirtualMachine(vmi, podPhase)
				pod.Spec.NodeName = "targetnode"

				addVirtualMachine(vmi)
				addActivePods(vmi, pod.UID, "targetnode")
				addPod(pod)

				sanityExecute()

				updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi.Status.Phase).To(Equal(virtv1.WaitingForSync))
			},
				Entry("Failed pod", k8sv1.PodFailed),
				Entry("Succeeded pod", k8sv1.PodSucceeded),
			)

			It("should keep migration target in WaitingForSync when migration failed and pod is running", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.WaitingForSync
				vmi.Status.NodeName = "targetnode"
				if vmi.Annotations == nil {
					vmi.Annotations = make(map[string]string)
				}
				vmi.Annotations[virtv1.CreateMigrationTarget] = "true"
				vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{
					TargetNode: "targetnode",
					SourceNode: "sourcenode",
					Failed:     true,
					Completed:  true,
				}

				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)
				pod.Spec.NodeName = "targetnode"

				addVirtualMachine(vmi)
				addActivePods(vmi, pod.UID, "targetnode")
				addPod(pod)

				sanityExecute()

				updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi.Status.Phase).To(Equal(virtv1.WaitingForSync))
			})

			It("should transition running non-target VMI to Failed when pod disappears", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				vmi.Status.NodeName = "somenode"

				addVirtualMachine(vmi)

				sanityExecute()
				expectVMIBeInPhase(vmi.Namespace, vmi.Name, virtv1.Failed)
			})

			DescribeTable("should not let migration target escape WaitingForSync via Scheduling", func(podPhase k8sv1.PodPhase) {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.WaitingForSync
				vmi.Status.NodeName = "targetnode"
				if vmi.Annotations == nil {
					vmi.Annotations = make(map[string]string)
				}
				vmi.Annotations[virtv1.CreateMigrationTarget] = "true"

				pod := newPodForVirtualMachine(vmi, podPhase)
				pod.Spec.NodeName = "targetnode"

				addVirtualMachine(vmi)
				addActivePods(vmi, pod.UID, "targetnode")
				addPod(pod)

				sanityExecute()

				updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi.Status.Phase).To(Equal(virtv1.WaitingForSync))
			},
				Entry("pod in Failed phase", k8sv1.PodFailed),
				Entry("pod in Succeeded phase", k8sv1.PodSucceeded),
			)

			DescribeTable("should transition scheduling migration target to WaitingForSync when pod is down", func(podPhase k8sv1.PodPhase) {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Scheduling
				vmi.Status.NodeName = "targetnode"
				if vmi.Annotations == nil {
					vmi.Annotations = make(map[string]string)
				}
				vmi.Annotations[virtv1.CreateMigrationTarget] = "true"

				pod := newPodForVirtualMachine(vmi, podPhase)
				pod.Spec.NodeName = "targetnode"

				addVirtualMachine(vmi)
				addPod(pod)

				sanityExecute()

				updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi.Status.Phase).To(Equal(virtv1.WaitingForSync))
			},
				Entry("pod in Failed phase", k8sv1.PodFailed),
				Entry("pod in Succeeded phase", k8sv1.PodSucceeded),
			)

			It("should transition scheduling migration target to WaitingForSync when pod disappears", func() {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Scheduling
				if vmi.Annotations == nil {
					vmi.Annotations = make(map[string]string)
				}
				vmi.Annotations[virtv1.CreateMigrationTarget] = "true"

				addVirtualMachine(vmi)

				sanityExecute()

				updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi.Status.Phase).To(Equal(virtv1.WaitingForSync))
			})
		})
	})

	Context("auto attach VSOCK", func() {
		It("should allocate CID when VirtualMachineInstance is scheduled", func() {
			vmi := newPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			vmi.Spec.Domain.Devices.AutoattachVSOCK = new(true)
			pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			addPod(pod)
			sanityExecute()

			expectVMIScheduledState(vmi)
			updatedVmi, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVmi.Status.VSOCKCID).NotTo(BeNil())
		})

		It("should recycle the CID when the pods are deleted", func() {
			alc := &fakeAllocator{}
			controller.cidsMap = alc

			vmi := newPendingVirtualMachine("testvmi")
			vmi.Spec.Domain.Devices.AutoattachVSOCK = new(true)
			Expect(controller.cidsMap.Allocate(vmi)).To(Succeed())
			vmi.Status.Phase = virtv1.Succeeded
			addVirtualMachine(vmi)

			Expect(controller.vmiIndexer.Delete(vmi)).To(Succeed())
			sanityExecute()

			Expect(alc.calls).To(ConsistOf([]string{"Allocate", "Remove"}))
		})

	})

	Context("When a migration exists", func() {
		It("should delay pod creation if the migration is running", func() {
			vmi := newPendingVirtualMachine("testvmi")

			addVirtualMachine(vmi)

			migration := &virtv1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: vmi.Name + "-",
					Namespace:    vmi.Namespace,
				},
				Spec: virtv1.VirtualMachineInstanceMigrationSpec{
					VMIName: vmi.Name,
				},
				Status: virtv1.VirtualMachineInstanceMigrationStatus{
					Phase: virtv1.MigrationRunning,
				},
			}
			Expect(controller.migrationIndexer.Add(migration)).To(Succeed())

			sanityExecute()
			// Not expecting pod creation event, test will fail if one gets created
			pods, err := kubeClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", virtv1.CreatedByLabel, string(vmi.UID))})
			Expect(err).ToNot(HaveOccurred())
			Expect(pods.Items).To(BeEmpty())
		})

		It("should ignore the migration if it's not running", func() {
			vmi := newPendingVirtualMachine("testvmi")

			addVirtualMachine(vmi)

			migration := &virtv1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: vmi.Name + "-",
					Namespace:    vmi.Namespace,
				},
				Spec: virtv1.VirtualMachineInstanceMigrationSpec{
					VMIName: vmi.Name,
				},
				Status: virtv1.VirtualMachineInstanceMigrationStatus{
					Phase: virtv1.MigrationSucceeded,
				},
			}
			Expect(controller.migrationIndexer.Add(migration)).To(Succeed())

			sanityExecute()
			testutils.ExpectEvent(recorder, kvcontroller.SuccessfulCreatePodReason)
			expectMatchingPodCreation(vmi)
		})
	})

	Context("Automatic Migration Requirement", func() {
		noConditionMatcher := Not(ContainElement(MatchFields(IgnoreExtras,
			Fields{
				"Type": Equal(virtv1.VirtualMachineInstanceMigrationRequired),
			})))

		falseConditionMatcher := ContainElement(MatchFields(IgnoreExtras,
			Fields{
				"Type":   Equal(virtv1.VirtualMachineInstanceMigrationRequired),
				"Status": Equal(k8sv1.ConditionFalse),
				"Reason": Equal(virtv1.VirtualMachineInstanceReasonAutoMigrationPending),
			}))

		trueConditionMatcher := ContainElement(MatchFields(IgnoreExtras,
			Fields{
				"Type":   Equal(virtv1.VirtualMachineInstanceMigrationRequired),
				"Status": Equal(k8sv1.ConditionTrue),
				"Reason": Equal(virtv1.VirtualMachineInstanceReasonAutoMigrationDueToLiveUpdate),
			}))

		DescribeTable("Auto migration logic",
			func(existingCondition *virtv1.VirtualMachineInstanceCondition, evaluator migrationEvaluator, matcher gomegaTypes.GomegaMatcher) {
				vmi := newPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running

				if existingCondition != nil {
					vmi.Status.Conditions = append(vmi.Status.Conditions, *existingCondition)
				}

				pod := newPodForVirtualMachine(vmi, k8sv1.PodRunning)

				addVirtualMachine(vmi)
				addPod(pod)
				addActivePods(vmi, pod.UID, "")

				controller.netMigrationEvaluator = evaluator
				sanityExecute()

				expectVMIWithMatcherConditions(vmi.Namespace, vmi.Name, matcher)
			},
			Entry("should not set the condition when there is no condition and evaluator returns Unknown",
				nil,
				stubMigrationEvaluator{result: k8sv1.ConditionUnknown},
				noConditionMatcher,
			),
			Entry("should set the condition to false when there is no condition and evaluator returns False",
				nil,
				stubMigrationEvaluator{result: k8sv1.ConditionFalse},
				falseConditionMatcher,
			),
			Entry("should set the condition to false when there is no condition and evaluator returns True",
				nil,
				stubMigrationEvaluator{result: k8sv1.ConditionTrue},
				trueConditionMatcher,
			),
			Entry("should keep the False condition when there is a False condition and evaluator returns False",
				&virtv1.VirtualMachineInstanceCondition{
					Type:   virtv1.VirtualMachineInstanceMigrationRequired,
					Status: k8sv1.ConditionFalse,
					Reason: virtv1.VirtualMachineInstanceReasonAutoMigrationPending,
				},
				stubMigrationEvaluator{result: k8sv1.ConditionFalse},
				falseConditionMatcher,
			),
			Entry("should keep the True condition when there is a True condition and evaluator returns False",
				&virtv1.VirtualMachineInstanceCondition{
					Type:   virtv1.VirtualMachineInstanceMigrationRequired,
					Status: k8sv1.ConditionTrue,
					Reason: virtv1.VirtualMachineInstanceReasonAutoMigrationDueToLiveUpdate,
				},
				stubMigrationEvaluator{result: k8sv1.ConditionFalse},
				trueConditionMatcher,
			),
			Entry("should set the condition to True when there is a False condition and evaluator returns True",
				&virtv1.VirtualMachineInstanceCondition{
					Type:   virtv1.VirtualMachineInstanceMigrationRequired,
					Status: k8sv1.ConditionFalse,
					Reason: virtv1.VirtualMachineInstanceReasonAutoMigrationPending,
				},
				stubMigrationEvaluator{result: k8sv1.ConditionTrue},
				trueConditionMatcher,
			),
			Entry("should remove the condition when there is a False condition and evaluator returns Unknown",
				&virtv1.VirtualMachineInstanceCondition{
					Type:   virtv1.VirtualMachineInstanceMigrationRequired,
					Status: k8sv1.ConditionFalse,
					Reason: virtv1.VirtualMachineInstanceReasonAutoMigrationPending,
				},
				stubMigrationEvaluator{result: k8sv1.ConditionUnknown},
				noConditionMatcher,
			),
			Entry("should remove the condition when there is a True condition and evaluator returns Unknown",
				&virtv1.VirtualMachineInstanceCondition{
					Type:   virtv1.VirtualMachineInstanceMigrationRequired,
					Status: k8sv1.ConditionTrue,
					Reason: virtv1.VirtualMachineInstanceReasonAutoMigrationDueToLiveUpdate,
				},
				stubMigrationEvaluator{result: k8sv1.ConditionUnknown},
				noConditionMatcher,
			),
		)
	})

	Context("setActivePods", func() {
		type podSpec struct {
			phase    k8sv1.PodPhase
			uid      string
			name     string
			nodeName string
		}

		DescribeTable("should only include non-terminated pods", func(pods []podSpec, expectedLen int, expectedEntries map[types.UID]string) {
			vmi := newPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Running

			for _, ps := range pods {
				pod := newPodForVirtualMachine(vmi, ps.phase)
				pod.UID = types.UID(ps.uid)
				pod.Name = ps.name
				pod.Spec.NodeName = ps.nodeName
				Expect(controller.podIndexer.Add(pod)).To(Succeed())
			}

			updatedVMI, err := controller.setActivePods(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVMI.Status.ActivePods).To(HaveLen(expectedLen))
			for uid, node := range expectedEntries {
				Expect(updatedVMI.Status.ActivePods).To(HaveKeyWithValue(uid, node))
			}
		},
			Entry("excludes succeeded pods",
				[]podSpec{
					{k8sv1.PodRunning, "running-uid", "running-pod", "node1"},
					{k8sv1.PodSucceeded, "succeeded-uid", "succeeded-pod", "node2"},
				},
				1, map[types.UID]string{"running-uid": "node1"},
			),
			Entry("excludes failed pods",
				[]podSpec{
					{k8sv1.PodRunning, "running-uid", "running-pod", "node1"},
					{k8sv1.PodFailed, "failed-uid", "failed-pod", "node2"},
				},
				1, map[types.UID]string{"running-uid": "node1"},
			),
			Entry("keeps only target pod after migration",
				[]podSpec{
					{k8sv1.PodSucceeded, "source-uid", "source-pod", "source-node"},
					{k8sv1.PodRunning, "target-uid", "target-pod", "target-node"},
				},
				1, map[types.UID]string{"target-uid": "target-node"},
			),
			Entry("includes pending pods",
				[]podSpec{
					{k8sv1.PodPending, "pending-uid", "pending-pod", "node1"},
				},
				1, map[types.UID]string{"pending-uid": "node1"},
			),
			Entry("includes unknown pods",
				[]podSpec{
					{k8sv1.PodUnknown, "unknown-uid", "unknown-pod", "node1"},
				},
				1, map[types.UID]string{"unknown-uid": "node1"},
			),
			Entry("produces empty map when all pods are terminated",
				[]podSpec{
					{k8sv1.PodSucceeded, "succeeded-uid", "succeeded-pod", "node1"},
				},
				0, map[types.UID]string{},
			),
		)
	})
})

func newDv(namespace string, name string, phase cdiv1.DataVolumePhase) *cdiv1.DataVolume {
	dv := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: cdiv1.DataVolumeStatus{
			Phase: phase,
		},
	}

	// mock phase coherent conditions
	switch phase {
	case cdiv1.Pending,
		cdiv1.WaitForFirstConsumer,
		cdiv1.PendingPopulation:
		dv.Status.Conditions = []cdiv1.DataVolumeCondition{
			{
				Type:   cdiv1.DataVolumeBound,
				Status: k8sv1.ConditionFalse,
			},
			{
				Type:   cdiv1.DataVolumeRunning,
				Status: k8sv1.ConditionFalse,
			},
			{
				Type:   cdiv1.DataVolumeReady,
				Status: k8sv1.ConditionFalse,
			},
		}
	case cdiv1.PVCBound:
		dv.Status.Conditions = []cdiv1.DataVolumeCondition{
			{
				Type:   cdiv1.DataVolumeBound,
				Status: k8sv1.ConditionTrue,
			},
			{
				Type:   cdiv1.DataVolumeRunning,
				Status: k8sv1.ConditionFalse,
			},
			{
				Type:   cdiv1.DataVolumeReady,
				Status: k8sv1.ConditionFalse,
			},
		}
	case cdiv1.ImportScheduled,
		cdiv1.ImportInProgress,
		cdiv1.CloneScheduled,
		cdiv1.CloneInProgress,
		cdiv1.SnapshotForSmartCloneInProgress,
		cdiv1.CloneFromSnapshotSourceInProgress,
		cdiv1.SmartClonePVCInProgress,
		cdiv1.CSICloneInProgress,
		cdiv1.ExpansionInProgress,
		cdiv1.NamespaceTransferInProgress,
		cdiv1.UploadScheduled,
		cdiv1.UploadReady:
		dv.Status.Conditions = []cdiv1.DataVolumeCondition{
			{
				Type:   cdiv1.DataVolumeBound,
				Status: k8sv1.ConditionTrue,
			},
			{
				Type:   cdiv1.DataVolumeRunning,
				Status: k8sv1.ConditionTrue,
			},
			{
				Type:   cdiv1.DataVolumeReady,
				Status: k8sv1.ConditionFalse,
			},
		}
	case cdiv1.Succeeded:
		dv.Status.Conditions = []cdiv1.DataVolumeCondition{
			{
				Type:   cdiv1.DataVolumeBound,
				Status: k8sv1.ConditionTrue,
			},
			{
				Type:   cdiv1.DataVolumeRunning,
				Status: k8sv1.ConditionFalse,
			},
			{
				Type:   cdiv1.DataVolumeReady,
				Status: k8sv1.ConditionTrue,
			},
		}
	case cdiv1.Failed, cdiv1.Paused:
		dv.Status.Conditions = []cdiv1.DataVolumeCondition{
			{
				Type:   cdiv1.DataVolumeBound,
				Status: k8sv1.ConditionTrue,
			},
			{
				Type:   cdiv1.DataVolumeRunning,
				Status: k8sv1.ConditionFalse,
			},
			{
				Type:   cdiv1.DataVolumeReady,
				Status: k8sv1.ConditionFalse,
			},
		}
	case cdiv1.Unknown:
	default:
		dv.Status.Conditions = []cdiv1.DataVolumeCondition{
			{
				Type:   cdiv1.DataVolumeBound,
				Status: k8sv1.ConditionUnknown,
			},
			{
				Type:   cdiv1.DataVolumeRunning,
				Status: k8sv1.ConditionUnknown,
			},
			{
				Type:   cdiv1.DataVolumeReady,
				Status: k8sv1.ConditionUnknown,
			},
		}
	}

	return dv
}

func newPvc(namespace string, name string) *k8sv1.PersistentVolumeClaim {
	return &k8sv1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func newPvcWithOwner(namespace string, name string, ownerName string, isController *bool) *k8sv1.PersistentVolumeClaim {
	return &k8sv1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "DataVolume",
					Name:       ownerName,
					Controller: isController,
				},
			},
		},
	}
}

func newPendingVirtualMachine(name string) *virtv1.VirtualMachineInstance {
	vmi := api.NewMinimalVMI(name)
	vmi.UID = "1234"
	vmi.Status.Phase = virtv1.Pending
	setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodNotExistsReason)
	kvcontroller.SetLatestApiVersionAnnotation(vmi)
	return vmi
}

func setReadyCondition(vmi *virtv1.VirtualMachineInstance, status k8sv1.ConditionStatus, reason string) {
	kvcontroller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, virtv1.VirtualMachineInstanceReady)
	vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
		Type:   virtv1.VirtualMachineInstanceReady,
		Status: status,
		Reason: reason,
	})
}

func setDataVolumeCondition(dv *cdiv1.DataVolume, cond cdiv1.DataVolumeCondition) {
	for i := range dv.Status.Conditions {
		c := &dv.Status.Conditions[i]
		if c.Type == cond.Type {
			c.Status = cond.Status
			c.Message = cond.Message
			c.Reason = cond.Reason
			return
		}
	}
	dv.Status.Conditions = append(dv.Status.Conditions, cond)
}

func newPodForVirtualMachine(vmi *virtv1.VirtualMachineInstance, phase k8sv1.PodPhase) *k8sv1.Pod {
	podAnnotations := map[string]string{
		virtv1.DomainAnnotation:         vmi.Name,
		descheduler.EvictOnlyAnnotation: "",
	}
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: vmi.Namespace,
			UID:       "virt-launch-uid",
			Labels: map[string]string{
				virtv1.AppLabel:       "virt-launcher",
				virtv1.CreatedByLabel: string(vmi.UID),
			},
			Annotations: podAnnotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmi, virtv1.VirtualMachineInstanceGroupVersionKind),
			},
		},
		Status: k8sv1.PodStatus{
			Phase: phase,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: false, Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}}},
			},
			Conditions: []k8sv1.PodCondition{
				{
					Type:   virtv1.VirtualMachineUnpaused,
					Status: k8sv1.ConditionTrue,
				},
			},
		},
	}
}

func newHotplugPVC(name, namespace string, phase k8sv1.PersistentVolumeClaimPhase) *k8sv1.PersistentVolumeClaim {
	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1alpha1",
					Kind:       "DataVolume",
					Name:       name,
					Controller: new(true),
				},
			},
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{
				k8sv1.ReadOnlyMany,
			},
		},
		Status: k8sv1.PersistentVolumeClaimStatus{
			Phase: phase,
		},
	}
}

func newPodForVirtlauncher(virtlauncher *k8sv1.Pod, name, uid string, phase k8sv1.PodPhase) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: virtlauncher.Namespace,
			UID:       types.UID(uid),
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       virtlauncher.Name,
					Kind:       "Pod",
					APIVersion: "v1",
					Controller: new(true),
					UID:        virtlauncher.UID,
				},
			},
		},
		Status: k8sv1.PodStatus{
			Phase: phase,
		},
		Spec: k8sv1.PodSpec{
			// +2 for empty dir and token
			Volumes: []k8sv1.Volume{
				{
					Name: "hotplug-disks",
					VolumeSource: k8sv1.VolumeSource{
						EmptyDir: &k8sv1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "default-token",
					VolumeSource: k8sv1.VolumeSource{
						Secret: &k8sv1.SecretVolumeSource{},
					},
				},
			},
		},
	}
}

func addVolumeStatuses(vmi *virtv1.VirtualMachineInstance, volumeStatuses ...virtv1.VolumeStatus) *virtv1.VirtualMachineInstance {
	vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, volumeStatuses...)
	return vmi
}

func addActivePods(vmi *virtv1.VirtualMachineInstance, podUID types.UID, hostName string) *virtv1.VirtualMachineInstance {

	if vmi.Status.ActivePods != nil {
		vmi.Status.ActivePods[podUID] = hostName
	} else {
		vmi.Status.ActivePods = map[types.UID]string{
			podUID: hostName,
		}
	}
	return vmi
}

type fakeAllocator struct {
	calls []string
}

func (alc *fakeAllocator) Sync(_ []*virtv1.VirtualMachineInstance) {
	alc.calls = append(alc.calls, "Sync")
}

func (alc *fakeAllocator) Allocate(vmi *virtv1.VirtualMachineInstance) error {
	alc.calls = append(alc.calls, "Allocate")
	cid := uint32(3)
	vmi.Status.VSOCKCID = &cid
	return nil
}

func (alc *fakeAllocator) Remove(key string) {
	alc.calls = append(alc.calls, "Remove")
}

type stubNetworkAnnotationsGenerator struct {
	annotations map[string]string
}

func (s stubNetworkAnnotationsGenerator) GenerateFromActivePod(_ *virtv1.VirtualMachineInstance, _ *k8sv1.Pod) map[string]string {
	return s.annotations
}

type stubStorageAnnotationsGenerator struct {
	annotations map[string]string
	keys        []string
}

func (s stubStorageAnnotationsGenerator) Generate(_ *virtv1.VirtualMachineInstance) (map[string]string, error) {
	return s.annotations, nil
}

func (s stubStorageAnnotationsGenerator) ManagedAnnotationKeys() []string {
	return s.keys
}

func validateNetVMISpecStub(causes ...metav1.StatusCause) func(*k8sfield.Path, *virtv1.VirtualMachineInstanceSpec, *virtconfig.ClusterConfig) []metav1.StatusCause {
	return func(*k8sfield.Path, *virtv1.VirtualMachineInstanceSpec, *virtconfig.ClusterConfig) []metav1.StatusCause {
		return causes
	}
}

type stubMigrationEvaluator struct {
	result k8sv1.ConditionStatus
}

func (e stubMigrationEvaluator) Evaluate(_ *virtv1.VirtualMachineInstance, _ *k8sv1.Pod) k8sv1.ConditionStatus {
	return e.result
}

type expectedAction struct {
	Verb     string
	Resource string
}

func actionsSummary(actions []testing.Action) []expectedAction {
	result := make([]expectedAction, 0, len(actions))
	for _, a := range actions {
		result = append(result, expectedAction{Verb: a.GetVerb(), Resource: a.GetResource().Resource})
	}
	return result
}
