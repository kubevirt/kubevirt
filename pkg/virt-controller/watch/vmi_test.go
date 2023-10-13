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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"k8s.io/utils/pointer"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegaTypes "github.com/onsi/gomega/types"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	fakenetworkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	kvcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/network/sriov"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
)

type PodVmIfaceStatus struct {
	podIfaceStatus *networkv1.NetworkStatus
	vmIfaceStatus  *virtv1.VirtualMachineInstanceNetworkInterface
}

var _ = Describe("VirtualMachineInstance watcher", func() {
	var config *virtconfig.ClusterConfig

	var ctrl *gomock.Controller
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var vmiSource *framework.FakeControllerSource
	var vmSource *framework.FakeControllerSource
	var podSource *framework.FakeControllerSource
	var vmiInformer cache.SharedIndexInformer
	var vmInformer cache.SharedIndexInformer
	var podInformer cache.SharedIndexInformer
	var stop chan struct{}
	var controller *VMIController
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue
	var podFeeder *testutils.PodFeeder
	var virtClient *kubecli.MockKubevirtClient
	var kubeClient *fake.Clientset
	var networkClient *fakenetworkclient.Clientset
	var pvcInformer cache.SharedIndexInformer
	var rqInformer cache.SharedIndexInformer
	var nsInformer cache.SharedIndexInformer
	var kvInformer cache.SharedIndexInformer

	var dataVolumeSource *framework.FakeControllerSource
	var dataVolumeInformer cache.SharedIndexInformer
	var cdiInformer cache.SharedIndexInformer
	var cdiConfigInformer cache.SharedIndexInformer
	var dataVolumeFeeder *testutils.DataVolumeFeeder
	var qemuGid int64 = 107
	controllerOf := true

	shouldExpectMatchingPodCreation := func(uid types.UID, matchers ...gomegaTypes.GomegaMatcher) {
		// Expect pod creation
		kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			update, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(update.GetObject().(*k8sv1.Pod).Labels[virtv1.CreatedByLabel]).To(Equal(string(uid)))
			Expect(update.GetObject().(*k8sv1.Pod)).To(SatisfyAny(matchers...))
			return true, update.GetObject(), nil
		})
	}

	shouldExpectPodCreation := func(uid types.UID) {
		// Expect pod creation
		kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			update, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(update.GetObject().(*k8sv1.Pod).Labels[virtv1.CreatedByLabel]).To(Equal(string(uid)))
			return true, update.GetObject(), nil
		})
	}

	shouldExpectMultiplePodDeletions := func(pod *k8sv1.Pod, deletionCount *int) {
		// Expect pod deletion
		kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			update, ok := action.(testing.DeleteAction)
			Expect(ok).To(BeTrue())
			Expect(pod.Namespace).To(Equal(update.GetNamespace()))
			*deletionCount++
			return true, nil, nil
		})
	}

	shouldExpectPodDeletion := func(pod *k8sv1.Pod) {
		// Expect pod deletion
		kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			update, ok := action.(testing.DeleteAction)
			Expect(ok).To(BeTrue())
			Expect(pod.Namespace).To(Equal(update.GetNamespace()))
			return true, nil, nil
		})
	}

	prependInjectPodPatch := func(pod *k8sv1.Pod) {
		kubeClient.Fake.PrependReactor("patch", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			originalPodBytes, err := json.Marshal(pod)
			if err != nil {
				return false, nil, err
			}

			patch := action.(testing.PatchAction)
			patchBytes := patch.GetPatch()

			patchJSON, err := jsonpatch.DecodePatch(patchBytes)
			if err != nil {
				return false, nil, err
			}
			newPodBytes, err := patchJSON.Apply(originalPodBytes)
			if err != nil {
				return false, nil, err
			}

			var newPod *k8sv1.Pod
			if err = json.Unmarshal(newPodBytes, &newPod); err != nil {
				return false, nil, err
			}

			return true, newPod, nil
		})
	}

	shouldExpectVirtualMachineHandover := func(vmi *virtv1.VirtualMachineInstance) {
		vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.Phase).To(Equal(virtv1.Scheduled))
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.PhaseTransitionTimestamps).ToNot(BeEmpty())
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.PhaseTransitionTimestamps).ToNot(HaveLen(len(vmi.Status.PhaseTransitionTimestamps)))
		}).Return(vmi, nil)
	}

	ignorePodUpdates := func() {
		kubeClient.Fake.PrependReactor("update", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			update, _ := action.(testing.UpdateAction)
			return true, update.GetObject(), nil
		})
	}

	shouldExpectVirtualMachineSchedulingState := func(vmi *virtv1.VirtualMachineInstance) {
		vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.Phase).To(Equal(virtv1.Scheduling))
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ConsistOf(MatchFields(IgnoreExtras,
				Fields{"Type": Equal(virtv1.VirtualMachineInstanceReady)})))
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.PhaseTransitionTimestamps).ToNot(BeEmpty())
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.PhaseTransitionTimestamps).ToNot(HaveLen(len(vmi.Status.PhaseTransitionTimestamps)))
		}).Return(vmi, nil)
	}

	shouldExpectVirtualMachineScheduledState := func(vmi *virtv1.VirtualMachineInstance) {
		vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.Phase).To(Equal(virtv1.Scheduled))
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ConsistOf(MatchFields(IgnoreExtras,
				Fields{"Type": Equal(virtv1.VirtualMachineInstanceReady)})))
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.PhaseTransitionTimestamps).ToNot(BeEmpty())
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.PhaseTransitionTimestamps).ToNot(HaveLen(len(vmi.Status.PhaseTransitionTimestamps)))
		}).Return(vmi, nil)
	}

	shouldExpectVirtualMachineFailedState := func(vmi *virtv1.VirtualMachineInstance) {
		vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.Phase).To(Equal(virtv1.Failed))
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ConsistOf(MatchFields(IgnoreExtras,
				Fields{"Type": Equal(virtv1.VirtualMachineInstanceReady)})))
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.PhaseTransitionTimestamps).ToNot(BeEmpty())
			Expect(arg.(*virtv1.VirtualMachineInstance).Status.PhaseTransitionTimestamps).ToNot(HaveLen(len(vmi.Status.PhaseTransitionTimestamps)))
		}).Return(vmi, nil)
	}

	shouldExpectHotplugPod := func() {
		// Expect pod creation
		kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			update, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(update.GetObject().(*k8sv1.Pod).GenerateName).To(Equal("hp-volume-"))
			return true, update.GetObject(), nil
		})
	}

	syncCaches := func(stop chan struct{}) {
		go vmiInformer.Run(stop)
		go vmInformer.Run(stop)
		go podInformer.Run(stop)
		go pvcInformer.Run(stop)

		go dataVolumeInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop,
			vmiInformer.HasSynced,
			vmInformer.HasSynced,
			podInformer.HasSynced,
			pvcInformer.HasSynced,
			dataVolumeInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		vmiInformer, vmiSource = testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachineInstance{}, kvcontroller.GetVMIInformerIndexers())
		vmInformer, vmSource = testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachine{}, kvcontroller.GetVirtualMachineInformerIndexers())
		podInformer, podSource = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		dataVolumeInformer, dataVolumeSource = testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		kubevirtFakeConfig := &virtv1.KubeVirtConfiguration{
			DeveloperConfiguration: &virtv1.DeveloperConfiguration{
				MinimumClusterTSCFrequency: pointer.Int64(12345),
			},
		}

		config, _, kvInformer = testutils.NewFakeClusterConfigUsingKVConfig(kubevirtFakeConfig)
		pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		cdiInformer, _ = testutils.NewFakeInformerFor(&cdiv1.CDIConfig{})
		cdiConfigInformer, _ = testutils.NewFakeInformerFor(&cdiv1.CDIConfig{})
		rqInformer, _ = testutils.NewFakeInformerFor(&k8sv1.ResourceQuota{})
		nsInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Namespace{})
		controller, _ = NewVMIController(
			services.NewTemplateService("a", 240, "b", "c", "d", "e", "f", "g", pvcInformer.GetStore(), virtClient, config, qemuGid, "h", rqInformer.GetStore(), nsInformer.GetStore()),
			vmiInformer,
			vmInformer,
			podInformer,
			pvcInformer,
			recorder,
			virtClient,
			dataVolumeInformer,
			cdiInformer,
			cdiConfigInformer,
			config,
			topology.NewTopologyHinter(&cache.FakeCustomStore{}, &cache.FakeCustomStore{}, config),
		)
		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
		podFeeder = testutils.NewPodFeeder(mockQueue, podSource)
		dataVolumeFeeder = testutils.NewDataVolumeFeeder(mockQueue, dataVolumeSource)

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstance(k8sv1.NamespaceDefault).Return(vmiInterface).AnyTimes()
		kubeClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		networkClient = fakenetworkclient.NewSimpleClientset()
		virtClient.EXPECT().NetworkClient().Return(networkClient).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		syncCaches(stop)
	})
	AfterEach(func() {
		close(stop)
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
	})

	addVirtualMachine := func(vmi *virtv1.VirtualMachineInstance) {
		mockQueue.ExpectAdds(1)
		vmiSource.Add(vmi)
		mockQueue.Wait()
	}

	Context("On valid VirtualMachineInstance given with DataVolume source", func() {

		dvVolumeSource := virtv1.VolumeSource{
			DataVolume: &virtv1.DataVolumeSource{
				Name: "test1",
			},
		}
		It("should create a corresponding Pod on VMI creation when DataVolume is ready", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})

			dvPVC := NewPvc(vmi.Namespace, "test1")
			// we are mocking a successful DataVolume. we expect the PVC to
			// be available in the store if DV is successful.
			Expect(pvcInformer.GetIndexer().Add(dvPVC)).To(Succeed())

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.Succeeded)

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)
			shouldExpectPodCreation(vmi.UID)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})

		It("should create a doppleganger Pod on VMI creation when DataVolume is in WaitForFirstConsumer state", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)

			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, &controllerOf)
			dvPVC.Status.Phase = k8sv1.ClaimPending
			// we are mocking a DataVolume in WFFC phase. we expect the PVC to
			// be in available but in the Pending state.
			Expect(pvcInformer.GetIndexer().Add(dvPVC)).To(Succeed())

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
					Fields{"Type": Equal(virtv1.VirtualMachineInstanceProvisioning)})))
			}).Return(vmi, nil)

			IsPodWithoutVmPayload := WithTransform(
				func(pod *k8sv1.Pod) string {
					for _, c := range pod.Spec.Containers {
						if c.Name == "compute" {
							return strings.Join(c.Command, " ")
						}
					}

					return ""
				},
				Equal("/bin/bash -c echo bound PVCs"))
			shouldExpectMatchingPodCreation(vmi.UID, IsPodWithoutVmPayload)
			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				Fail("here")
				return true, dvPVC, nil
			})

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})

		It("should not delete a doppleganger Pod on VMI creation when DataVolume is in WaitForFirstConsumer state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Annotations[virtv1.EphemeralProvisioningObject] = "true"

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})

			pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
				Name: "test1",
			})

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)

			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, &controllerOf)
			dvPVC.Status.Phase = k8sv1.ClaimPending
			// we are mocking a DataVolume in WFFC phase. we expect the PVC to
			// be in available but in the Pending state.
			Expect(pvcInformer.GetIndexer().Add(dvPVC)).To(Succeed())

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")
			dataVolumeFeeder.Add(dataVolume)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Phase).To(Equal(virtv1.Pending))
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).
					To(ContainElement(virtv1.VirtualMachineInstanceCondition{
						Type:   virtv1.VirtualMachineInstanceProvisioning,
						Status: k8sv1.ConditionTrue,
					}))
			}).Return(vmi, nil)
			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, dvPVC, nil
			})
			controller.Execute()
		})

		It("should delete a doppleganger Pod on VMI creation when DataVolume is no longer in WaitForFirstConsumer state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Annotations[virtv1.EphemeralProvisioningObject] = "true"

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})
			pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
				Name: "test1",
			})

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.Succeeded)
			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, &controllerOf)
			// we are mocking a DataVolume in Succeeded phase. we expect the PVC to
			// be in available
			Expect(pvcInformer.GetIndexer().Add(dvPVC)).To(Succeed())

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")
			dataVolumeFeeder.Add(dataVolume)
			shouldExpectPodDeletion(pod)

			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, dvPVC, nil
			})
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)
		})

		It("should only get WFFC pods that are associated with current VMI", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Annotations[virtv1.EphemeralProvisioningObject] = "true"
			Expect(podInformer.GetIndexer().Add(pod)).To(Succeed())

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
			Expect(podInformer.GetIndexer().Add(nonOwnedPod)).To(Succeed())

			res, err := controller.waitForFirstConsumerTemporaryPods(vmi, pod)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
		})

		DescribeTable("VMI should handle doppleganger Pod status while DV is in WaitForFirstConsumer phase",
			func(phase k8sv1.PodPhase, conditions []k8sv1.PodCondition, expectedPhase virtv1.VirtualMachineInstancePhase) {
				vmi := NewPendingVirtualMachine("testvmi")
				pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
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

				dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)
				dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, &controllerOf)
				dvPVC.Status.Phase = k8sv1.ClaimBound
				Expect(pvcInformer.GetIndexer().Add(dvPVC)).To(Succeed())

				addVirtualMachine(vmi)
				podFeeder.Add(pod)
				addActivePods(vmi, pod.UID, "")
				dataVolumeFeeder.Add(dataVolume)

				vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachineInstance).Status.Phase).To(Equal(expectedPhase))
				}).Return(vmi, nil)
				kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
					return true, dvPVC, nil
				})

				controller.Execute()
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
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.Pending)
			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, &controllerOf)
			dvPVC.Status.Phase = k8sv1.ClaimPending
			Expect(pvcInformer.GetIndexer().Add(dvPVC)).To(Succeed())

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)

			controller.Execute()
		})
	})

	Context("On valid VirtualMachineInstance given with PVC source, ownedRef of DataVolume", func() {

		pvcVolumeSource := virtv1.VolumeSource{
			PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: "test1",
			}},
		}
		It("should create a corresponding Pod on VMI creation when DataVolume is ready", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", "test1", &controllerOf)
			// we are mocking a successful DataVolume. we expect the PVC to
			// be available in the store if DV is successful.
			Expect(pvcInformer.GetIndexer().Add(dvPVC)).To(Succeed())

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.Succeeded)

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)
			shouldExpectPodCreation(vmi.UID)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})

		It("should create a doppleganger Pod on VMI creation when DataVolume is in WaitForFirstConsumer state", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", "test1", &controllerOf)
			dvPVC.Status.Phase = k8sv1.ClaimPending
			// we are mocking a DataVolume in WFFC phase. we expect the PVC to
			// be in available but in the Pending state.
			Expect(pvcInformer.GetIndexer().Add(dvPVC)).To(Succeed())

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
					Fields{"Type": Equal(virtv1.VirtualMachineInstanceProvisioning)})))
			}).Return(vmi, nil)

			IsPodWithoutVmPayload := WithTransform(
				func(pod *k8sv1.Pod) string {
					for _, c := range pod.Spec.Containers {
						if c.Name == "compute" {
							return strings.Join(c.Command, " ")
						}
					}

					return ""
				},
				Equal("/bin/bash -c echo bound PVCs"))
			shouldExpectMatchingPodCreation(vmi.UID, IsPodWithoutVmPayload)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})

		It("should not delete a doppleganger Pod on VMI creation when DataVolume is in WaitForFirstConsumer state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Annotations[virtv1.EphemeralProvisioningObject] = "true"

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
				Name: "test1",
			})

			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", "test1", &controllerOf)
			dvPVC.Status.Phase = k8sv1.ClaimPending
			// we are mocking a DataVolume in WFFC phase. we expect the PVC to
			// be in available but in the Pending state.
			Expect(pvcInformer.GetIndexer().Add(dvPVC)).To(Succeed())

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")
			dataVolumeFeeder.Add(dataVolume)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Phase).To(Equal(virtv1.Pending))
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).
					To(ContainElement(virtv1.VirtualMachineInstanceCondition{
						Type:   virtv1.VirtualMachineInstanceProvisioning,
						Status: k8sv1.ConditionTrue,
					}))
			}).Return(vmi, nil)
			controller.Execute()
		})

		It("should delete a doppleganger Pod on VMI creation when DataVolume is no longer in WaitForFirstConsumer state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Annotations[virtv1.EphemeralProvisioningObject] = "true"

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
				Name: "test1",
			})

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.Succeeded)
			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, &controllerOf)
			dvPVC.Status.Phase = k8sv1.ClaimBound
			// we are mocking a DataVolume in Succeeded phase. we expect the PVC to
			// be in available
			Expect(pvcInformer.GetIndexer().Add(dvPVC)).To(Succeed())

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")
			dataVolumeFeeder.Add(dataVolume)
			shouldExpectPodDeletion(pod)
			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, dvPVC, nil
			})

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)
		})

		It("should not create a corresponding Pod on VMI creation when DataVolume is pending", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", "test1", &controllerOf)
			dvPVC.Status.Phase = k8sv1.ClaimPending
			// we are mocking a successful DataVolume. we expect the PVC to
			// be available in the store if DV is successful.
			Expect(pvcInformer.GetIndexer().Add(dvPVC)).To(Succeed())

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.Pending)

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)

			controller.Execute()
		})

		It("should create a corresponding Pod on VMI creation when PVC is not controlled by a DataVolume", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			pvc := NewPvc(vmi.Namespace, "test1")
			Expect(pvcInformer.GetIndexer().Add(pvc)).To(Succeed())

			addVirtualMachine(vmi)
			shouldExpectPodCreation(vmi.UID)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})

	})

	Context("with network-status annotation", func() {
		const (
			defaultNetworkName = "default"
			sriovNetworkName   = "network1"
			netAttachDefName   = "default/nad1"
			selectedPCIAddress = "0000:04:02.5"
		)

		It("should not patch network-pci-map when no SR-IOV networks exist", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Running
			vmi = addDefaultNetwork(vmi, defaultNetworkName)
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")
			pod.Annotations[networkv1.NetworkStatusAnnot] = `
			[
			{
			"name": "kindnet",
			"interface": "eth0",
			"ips": [
			  "10.244.2.131"
			],
			"mac": "82:cf:7c:98:43:7e",
			"default": true,
			"dns": {}
			}
			]`

			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), &metav1.PatchOptions{}).Return(vmi, nil)
			controller.Execute()
			Expect(pod.Annotations).ToNot(HaveKey(sriov.NetworkPCIMapAnnot))
		})
		It("should patch network-pci-map when SR-IOV networks exist", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Running
			vmi = addDefaultNetwork(vmi, defaultNetworkName)
			vmi = addDefaultNetworkStatus(vmi, defaultNetworkName)
			vmi = addSRIOVNetwork(vmi, sriovNetworkName, netAttachDefName)
			vmi = addDefaultNetworkStatus(vmi, sriovNetworkName)
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")
			pod.Annotations[networkv1.NetworkStatusAnnot] = `
			[
			{
			"name": "kindnet",
			"interface": "eth0",
			"ips": [
			  "10.244.2.131"
			],
			"mac": "82:cf:7c:98:43:7e",
			"default": true,
			"dns": {}
			},
			{
			"name": "` + netAttachDefName + `",
			"interface": "poda7662f44d65",
			"dns": {},
			"device-info": {
			  "type": "pci",
			  "version": "1.0.0",
			  "pci": {
			    "pci-address": "` + selectedPCIAddress + `"
			  }
			}
			}
			]`

			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), &metav1.PatchOptions{}).Return(vmi, nil)
			prependInjectPodPatch(pod)
			controller.Execute()
			Expect(pod.Annotations).To(HaveKeyWithValue(sriov.NetworkPCIMapAnnot, `{"`+sriovNetworkName+`":"`+selectedPCIAddress+`"}`))
		})
	})

	Context("On valid VirtualMachineInstance given", func() {
		It("should create a corresponding Pod on VirtualMachineInstance creation", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			addVirtualMachine(vmi)

			shouldExpectPodCreation(vmi.UID)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})
		DescribeTable("should delete the corresponding Pods on VirtualMachineInstance deletion with vmi", func(phase virtv1.VirtualMachineInstancePhase) {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Status.Phase = phase
			vmi.DeletionTimestamp = now()

			if vmi.IsRunning() {
				setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodConditionMissingReason)
			} else {
				setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			}

			// 2 pods are owned by VMI
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.UID = "456-456-456"
			pod2 := pod.DeepCopy()
			pod2.UID = "123-123-123"
			pod2.Name = "test2"

			// The 3rd one is not.
			pod3 := pod.DeepCopy()
			pod3.UID = "123-123-123"
			pod3.Name = "test2"
			pod3.Annotations = make(map[string]string)
			pod3.Labels = make(map[string]string)

			addActivePods(vmi, pod.UID, "")
			addActivePods(vmi, pod2.UID, "")

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			podFeeder.Add(pod2)

			deletionCount := 0
			shouldExpectMultiplePodDeletions(pod, &deletionCount)

			if vmi.IsUnprocessed() {
				shouldExpectVirtualMachineSchedulingState(vmi)
			}

			//shouldExpectVirtualMachineFailedState(vmi)

			controller.Execute()

			// Verify only the 2 pods are deleted
			Expect(deletionCount).To(Equal(2))

			testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)
			testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)
		},
			Entry("in running state", virtv1.Running),
			Entry("in unset state", virtv1.VmPhaseUnset),
			Entry("in pending state", virtv1.Pending),
			Entry("in succeeded state", virtv1.Succeeded),
			Entry("in failed state", virtv1.Failed),
			Entry("in scheduled state", virtv1.Scheduled),
			Entry("in scheduling state", virtv1.Scheduling),
		)
		It("should not try to delete a pod again, which is already marked for deletion and go to failed state, when in scheduling state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)

			vmi.Status.Phase = virtv1.Scheduling
			vmi.DeletionTimestamp = now()
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			shouldExpectPodDeletion(pod)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)

			modifiedPod := pod.DeepCopy()
			modifiedPod.DeletionTimestamp = now()

			podFeeder.Modify(modifiedPod)

			shouldExpectVirtualMachineFailedState(vmi)

			controller.Execute()
		})
		DescribeTable("should not delete the corresponding Pod if the vmi is in", func(phase virtv1.VirtualMachineInstancePhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)

			vmi.Status.Phase = phase
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			controller.Execute()
		},
			Entry("succeeded state", virtv1.Failed),
			Entry("failed state", virtv1.Succeeded),
		)
		It("should do nothing if the vmi is in final state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Failed
			vmi.Finalizers = []string{}

			addVirtualMachine(vmi)

			controller.Execute()
		})
		It("should set an error condition if creating the pod fails", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			addVirtualMachine(vmi)

			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
						"Status": Equal(k8sv1.ConditionFalse),
						"Reason": Equal("FailedCreate"),
					})))
			}).Return(vmi, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedCreatePodReason)
		})
		It("should back-off if a sync error occurs", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			addVirtualMachine(vmi)

			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
						"Status": Equal(k8sv1.ConditionFalse),
						"Reason": Equal("FailedCreate"),
					})))
			}).Return(vmi, nil)

			controller.Execute()
			Expect(controller.Queue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(1))

			testutils.ExpectEvent(recorder, FailedCreatePodReason)
		})
		It("should remove the error condition if the sync finally succeeds", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{Type: virtv1.VirtualMachineInstanceSynchronized})

			addVirtualMachine(vmi)

			// Expect pod creation
			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				update, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				Expect(update.GetObject().(*k8sv1.Pod).Labels[virtv1.CreatedByLabel]).To(Equal(string(vmi.UID)))
				return true, update.GetObject(), nil
			})

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).ToNot(ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Type": Equal(virtv1.VirtualMachineInstanceSynchronized),
					})))
			}).Return(vmi, nil)

			controller.Execute()
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
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

				vmi := NewPendingVirtualMachine("testvmi")
				setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodNotExistsReason)

				vmi.Spec.Volumes = []virtv1.Volume{
					{
						Name:         "test",
						VolumeSource: volumeSource,
					},
				}
				addVirtualMachine(vmi)
				update := vmiInterface.EXPECT().Update(context.Background(), gomock.Any())
				update.Do(func(ctx context.Context, vmi *virtv1.VirtualMachineInstance) {
					expectConditions(vmi)
					_ = vmiInformer.GetStore().Update(vmi)
					key := kvcontroller.VirtualMachineInstanceKey(vmi)
					controller.vmiExpectations.LowerExpectations(key, 1, 0)
					update.Return(vmi, nil)
				})

				controller.Execute()
				Expect(controller.Queue.Len()).To(Equal(0))
				// When the PVC appears, we will automatically be retriggered, it is not an error condition
				Expect(mockQueue.GetRateLimitedEnqueueCount()).To(BeZero())
				addVirtualMachine(vmi)

				// make sure that during next iteration we do not add the same condition again
				vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, vmi *virtv1.VirtualMachineInstance) {
					expectConditions(vmi)
				}).Return(vmi, nil)

				controller.Execute()
				Expect(controller.Queue.Len()).To(Equal(0))
				Expect(mockQueue.GetRateLimitedEnqueueCount()).To(BeZero())
			},
			Entry("when PVC does not exist", FailedPvcNotFoundReason,
				virtv1.VolumeSource{
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "something",
						},
					},
				}),
			Entry("when DataVolume does not exist", FailedPvcNotFoundReason,
				virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: "something",
					},
				}),
		)

		DescribeTable("should move the vmi to scheduling state if a pod exists", func(phase k8sv1.PodPhase, isReady bool) {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, phase)
			pod.Status.ContainerStatuses[0].Ready = isReady

			unreadyReason := virtv1.GuestNotRunningReason
			if phase == k8sv1.PodSucceeded || phase == k8sv1.PodFailed {
				unreadyReason = virtv1.PodTerminatingReason
			}
			setReadyCondition(vmi, k8sv1.ConditionFalse, unreadyReason)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			shouldExpectVirtualMachineSchedulingState(vmi)

			controller.Execute()
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
				vmi := NewPendingVirtualMachine("testvmi")
				setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
				vmi.Status.Phase = virtv1.Scheduling

				pod := NewPodForVirtualMachine(vmi, k8sv1.PodPending)

				pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{
					Message: "Insufficient memory",
					Reason:  "Unschedulable",
					Status:  k8sv1.ConditionFalse,
					Type:    k8sv1.PodScheduled,
				})

				addVirtualMachine(vmi)
				podFeeder.Add(pod)

				vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachineInstance).Status.Phase).To(Equal(virtv1.Scheduling))
					Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
						Fields{
							"Type":    Equal(virtv1.VirtualMachineInstanceConditionType((k8sv1.PodScheduled))),
							"Status":  Equal(k8sv1.ConditionFalse),
							"Reason":  Equal("Unschedulable"),
							"Message": Equal("Insufficient memory"),
						})))
				}).Return(vmi, nil)

				controller.Execute()
			})
		})

		Context("when Pod recovers from scheduling issues", func() {
			DescribeTable("it should remove scheduling pod condition from the VirtualMachineInstance if the pod", func(owner string, podPhase k8sv1.PodPhase) {
				vmi := NewPendingVirtualMachine("testvmi")
				setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodConditionMissingReason)
				vmi.Status.Phase = virtv1.Scheduling

				pod := NewPodForVirtualMachine(vmi, podPhase)

				vmi.Status.Conditions = append(vmi.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
					Message: "Insufficient memory",
					Reason:  "Unschedulable",
					Status:  k8sv1.ConditionFalse,
					Type:    virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled),
				})

				addVirtualMachine(vmi)
				podFeeder.Add(pod)

				vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).ToNot(ContainElement(MatchFields(IgnoreExtras,
						Fields{
							"Type": Equal(virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled)),
						})))
				}).Return(vmi, nil)

				ignorePodUpdates()

				controller.Execute()

				testutils.IgnoreEvents(recorder)
			},
				Entry("is owned by virt-handler and is running", "virt-handler", k8sv1.PodRunning),
				Entry("is owned by virt-controller and is running", "virt-controller", k8sv1.PodRunning),
				Entry("is owned by virt-controller and is pending", "virt-controller", k8sv1.PodPending),
			)
		})

		It("should move the vmi to failed state if the pod disappears and the vmi is in scheduling state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Scheduling

			addVirtualMachine(vmi)

			shouldExpectVirtualMachineFailedState(vmi)

			controller.Execute()
		})
		It("should move the vmi to failed state if the vmi is pending, no pod exists yet and gets deleted", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.DeletionTimestamp = now()

			addVirtualMachine(vmi)

			shouldExpectVirtualMachineFailedState(vmi)

			controller.Execute()
		})
		DescribeTable("should hand over pod to virt-handler if pod is ready and running", func(containerStatus []k8sv1.ContainerStatus) {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.ContainerStatuses = containerStatus

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			if containerStatus[0].Ready {
				addActivePods(vmi, pod.UID, "")
			}

			shouldExpectVirtualMachineScheduledState(vmi)

			controller.Execute()
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
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.ContainerStatuses = containerStatus

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			controller.Execute()
		},
			Entry("with not ready infra container and not ready compute container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: false}, {Name: "kubevirt-infra", Ready: false}},
			),
			Entry("with not ready compute container and no infra container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: false}},
			),
		)

		DescribeTable("With a virt-launcher pod and an attachment pod, it", func(attachmentPodPhase k8sv1.PodPhase, expectedPhase virtv1.VirtualMachineInstancePhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			pvc := NewHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimBound)
			Expect(pvcInformer.GetIndexer().Add(pvc)).To(Succeed())
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dv",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Pending,
				},
			}
			Expect(dataVolumeInformer.GetIndexer().Add(dv)).To(Succeed())
			Expect(pvcInformer.GetIndexer().Add(pvc)).To(Succeed())
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
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{
				Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
			}}
			attachmentPod := NewPodForVirtlauncher(pod, "hp-test", "abcd", attachmentPodPhase)
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
			podFeeder.Add(pod)
			podFeeder.Add(attachmentPod)

			switch expectedPhase {
			case virtv1.Scheduled:
				shouldExpectVirtualMachineScheduledState(vmi)
			case virtv1.Scheduling:
				vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
					Expect(arg.(*virtv1.VirtualMachineInstance).Status.Phase).To(Equal(virtv1.Scheduling))
					Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ConsistOf(MatchFields(IgnoreExtras,
						Fields{"Type": Equal(virtv1.VirtualMachineInstanceReady)})))
					Expect(arg.(*virtv1.VirtualMachineInstance).Status.PhaseTransitionTimestamps).To(BeEmpty())
				}).Return(vmi, nil)
			}

			controller.Execute()

			switch expectedPhase {
			case virtv1.Scheduled:
				testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
			case virtv1.Scheduling:
				break
			}
		},
			Entry("should hand over pods if both are ready and running", k8sv1.PodRunning, virtv1.Scheduled),
			Entry("should hand over pods even if the attachment pod is not ready and running", k8sv1.PodPending, virtv1.Scheduled),
		)

		It("should ignore migration target pods", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodConditionMissingReason)
			vmi.Status.Phase = virtv1.Running
			vmi.Status.NodeName = "curnode"

			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
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
			podFeeder.Add(failedTargetPod)
			podFeeder.Add(pod)
			podFeeder.Add(failedTargetPod2)

			selectedPod, err := kvcontroller.CurrentVMIPod(vmi, podInformer)
			Expect(err).ToNot(HaveOccurred())

			Expect(selectedPod.UID).To(Equal(pod.UID))
		})

		It("should set an error condition if deleting the virtual machine pod fails", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.DeletionTimestamp = now()
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			// Expect pod delete
			kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
						"Status": Equal(k8sv1.ConditionFalse),
						"Reason": Equal(FailedDeletePodReason),
					})))
			}).Return(vmi, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedDeletePodReason)
		})
		It("should set an error condition if when pod cannot guarantee resources when cpu pinning has been requested", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			vmi.Spec.Domain.CPU = &virtv1.CPU{DedicatedCPUPlacement: true}
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
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
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
						"Status": Equal(k8sv1.ConditionFalse),
						"Reason": Equal(FailedGuaranteePodResourcesReason),
					})))
			}).Return(vmi, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedGuaranteePodResourcesReason)
		})
		It("should set an error condition if creating the virtual machine pod fails", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			// Expect pod creation
			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			addVirtualMachine(vmi)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
						"Status": Equal(k8sv1.ConditionFalse),
						"Reason": Equal(FailedCreatePodReason),
					})))
			}).Return(vmi, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedCreatePodReason)
		})
		It("should update the virtual machine to scheduled if pod is ready, runnning and handed over to virt-handler", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			shouldExpectVirtualMachineHandover(vmi)

			controller.Execute()
		})
		It("should update the virtual machine QOS class if the pod finally has a QOS class assigned", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.QOSClass = k8sv1.PodQOSGuaranteed

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(*arg.(*virtv1.VirtualMachineInstance).Status.QOSClass).To(Equal(k8sv1.PodQOSGuaranteed))
			}).Return(vmi, nil)

			controller.Execute()
		})
		It("should update the virtual machine to scheduled if pod is ready, triggered by pod change", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodPending)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			controller.Execute()

			pod = NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			podFeeder.Modify(pod)

			shouldExpectVirtualMachineHandover(vmi)

			controller.Execute()
		})
		It("should update the virtual machine to failed if pod was not ready, triggered by pod delete", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodPending)
			vmi.Status.Phase = virtv1.Scheduling

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			controller.Execute()

			podFeeder.Delete(pod)

			shouldExpectVirtualMachineFailedState(vmi)

			controller.Execute()
		})
		It("should update the virtual machine to failed if compute container is terminated while still scheduling", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodTerminatingReason)
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodPending)
			pod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{Name: "compute", State: k8sv1.ContainerState{Terminated: &k8sv1.ContainerStateTerminated{}}}}
			vmi.Status.Phase = virtv1.Scheduling

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			shouldExpectVirtualMachineFailedState(vmi)

			controller.Execute()
		})
		DescribeTable("should remove the fore ground finalizer if no pod is present and the vmi is in ", func(phase virtv1.VirtualMachineInstancePhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = phase
			vmi.Status.LauncherContainerImageVersion = "madeup"
			vmi.Labels = map[string]string{}
			vmi.Labels[virtv1.OutdatedLauncherImageLabel] = ""
			vmi.Finalizers = append(vmi.Finalizers, virtv1.VirtualMachineInstanceFinalizer)
			Expect(vmi.Finalizers).To(ContainElement(virtv1.VirtualMachineInstanceFinalizer))

			addVirtualMachine(vmi)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				// outdated launcher label should get cleared after being finalized
				_, ok := arg.(*virtv1.VirtualMachineInstance).Labels[virtv1.OutdatedLauncherImageLabel]
				Expect(ok).To(BeFalse())
				// outdated launcher image should get cleared after being finalized
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.LauncherContainerImageVersion).To(Equal(""))

				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Phase).To(Equal(phase))
				Expect(arg.(*virtv1.VirtualMachineInstance).Finalizers).ToNot(ContainElement(virtv1.VirtualMachineInstanceFinalizer))
			}).Return(vmi, nil)

			controller.Execute()
		},
			Entry("succeeded state", virtv1.Succeeded),
			Entry("failed state", virtv1.Failed),
		)

		DescribeTable("VM controller finalizer", func(hasOwner, vmExists, expectFinalizer bool) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Finalizers = append(vmi.Finalizers, virtv1.VirtualMachineControllerFinalizer, virtv1.VirtualMachineInstanceFinalizer)
			vmi.Status.Phase = virtv1.Succeeded
			Expect(vmi.Finalizers).To(ContainElement(virtv1.VirtualMachineControllerFinalizer))

			vm := VirtualMachineFromVMI(vmi.Name, vmi, true)
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
				vmSource.Add(vm)
				// the controller isn't using informer callbacks for the VM informer
				// so add a sleep here to ensure the informer has time to cache up before
				// we call Execute()
				time.Sleep(1 * time.Second)
			}

			addVirtualMachine(vmi)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				if !expectFinalizer {
					Expect(arg.(*virtv1.VirtualMachineInstance).Finalizers).ToNot(ContainElement(virtv1.VirtualMachineControllerFinalizer))
				} else {
					Expect(arg.(*virtv1.VirtualMachineInstance).Finalizers).To(ContainElement(virtv1.VirtualMachineControllerFinalizer))
				}
				Expect(arg.(*virtv1.VirtualMachineInstance).Finalizers).ToNot(ContainElement(virtv1.VirtualMachineInstanceFinalizer))
			}).Return(vmi, nil)

			controller.Execute()
		},
			Entry("should be removed if no owner exists", false, false, false),
			Entry("should be removed if VM owner is set but VM is deleted", false, true, false),
			Entry("should be left if VM owner is present", true, true, true),
		)

		DescribeTable("should do nothing if pod is handed to virt-handler", func(phase k8sv1.PodPhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Scheduled
			pod := NewPodForVirtualMachine(vmi, phase)

			unreadyReason := virtv1.GuestNotRunningReason
			if phase == k8sv1.PodSucceeded || phase == k8sv1.PodFailed {
				unreadyReason = virtv1.PodTerminatingReason
			}
			setReadyCondition(vmi, k8sv1.ConditionFalse, unreadyReason)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			controller.Execute()
		},
			Entry("and in running state", k8sv1.PodRunning),
			Entry("and in unknown state", k8sv1.PodUnknown),
			Entry("and in succeeded state", k8sv1.PodSucceeded),
			Entry("and in failed state", k8sv1.PodFailed),
			Entry("and in pending state", k8sv1.PodPending),
		)

		It("should add outdated label if pod's image is outdated and VMI is in running state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionTrue, "")
			vmi.Status.Phase = virtv1.Running
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})

			pod.Spec.Containers = append(pod.Spec.Containers, k8sv1.Container{
				Image: "madeup",
				Name:  "compute",
			})

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			podFeeder.Add(pod)

			vmiPatch := `[{ "op": "add", "path": "/status/launcherContainerImageVersion", "value": "madeup" }, { "op": "add", "path": "/metadata/labels", "value": {"kubevirt.io/outdatedLauncherImage":""} }]`

			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(vmiPatch), &metav1.PatchOptions{}).Return(vmi, nil)

			podPatch := `[{ "op": "test", "path": "/metadata/labels", "value": {"kubevirt.io":"virt-launcher","kubevirt.io/created-by":"1234"} }, { "op": "replace", "path": "/metadata/labels", "value": {"kubevirt.io":"virt-launcher","kubevirt.io/created-by":"1234","kubevirt.io/outdatedLauncherImage":""} }]`
			kubeClient.Fake.PrependReactor("patch", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				p, _ := action.(testing.PatchAction)
				Expect(podPatch).To(Equal(string(p.GetPatch())))
				return true, nil, nil
			})
			controller.Execute()
		})
		It("should remove outdated label if pod's image up-to-date and VMI is in running state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionTrue, "")
			vmi.Status.Phase = virtv1.Running
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
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
			podFeeder.Add(pod)

			patch := `[{ "op": "add", "path": "/status/launcherContainerImageVersion", "value": "a" }, { "op": "test", "path": "/metadata/labels", "value": {"kubevirt.io/outdatedLauncherImage":""} }, { "op": "replace", "path": "/metadata/labels", "value": {} }]`

			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vmi, nil)

			controller.Execute()
		})

		It("should add a ready condition if it is present on the pod and the VMI is in running state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Conditions = nil
			vmi.Status.Phase = virtv1.Running
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			podFeeder.Add(pod)

			patch := `[{ "op": "test", "path": "/status/conditions", "value": null }, { "op": "replace", "path": "/status/conditions", "value": [{"type":"Ready","status":"True","lastProbeTime":null,"lastTransitionTime":null}] }]`
			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vmi, nil)

			controller.Execute()
		})

		It("should indicate on the ready condition if the pod is terminating", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Conditions = nil
			vmi.Status.Phase = virtv1.Running
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.DeletionTimestamp = now()
			pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), &metav1.PatchOptions{}).DoAndReturn(func(ctx context.Context, _ string, _ interface{}, patchBytes []byte, options interface{}, _ ...string) (*virtv1.VirtualMachineInstance, error) {
				patch, err := jsonpatch.DecodePatch(patchBytes)
				Expect(err).ToNot(HaveOccurred())
				vmiBytes, err := json.Marshal(vmi)
				Expect(err).ToNot(HaveOccurred())
				vmiBytes, err = patch.Apply(vmiBytes)
				Expect(err).ToNot(HaveOccurred())
				patchedVMI := &virtv1.VirtualMachineInstance{}
				err = json.Unmarshal(vmiBytes, patchedVMI)
				Expect(err).ToNot(HaveOccurred())
				Expect(patchedVMI.Status.Conditions).To(HaveLen(1))
				cond := patchedVMI.Status.Conditions[0]
				Expect(cond.Reason).To(Equal(virtv1.PodTerminatingReason))
				Expect(string(cond.Type)).To(Equal(string(k8sv1.PodReady)))
				Expect(cond.Status).To(Equal(k8sv1.ConditionFalse))
				return patchedVMI, nil
			})
			controller.Execute()
		})

		It("should add active pods to status if VMI is in running state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodConditionMissingReason)
			vmi.Status.Phase = virtv1.Running
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.UID = "someUID"
			pod.Spec.NodeName = "someHost"

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			patch := `[{ "op": "test", "path": "/status/activePods", "value": {} }, { "op": "replace", "path": "/status/activePods", "value": {"someUID":"someHost"} }]`
			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vmi, nil)

			controller.Execute()
		})

		It("should not remove sync conditions from virt-handler if it is in scheduled state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduled
			vmi.Status.Conditions = append(vmi.Status.Conditions,
				virtv1.VirtualMachineInstanceCondition{Type: virtv1.VirtualMachineInstanceSynchronized, Status: k8sv1.ConditionFalse})
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			controller.Execute()
		})

		DescribeTable("should do nothing if the vmi is handed over to virt-handler, the pod disappears", func(phase virtv1.VirtualMachineInstancePhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = phase

			addVirtualMachine(vmi)

			controller.Execute()
		},
			Entry("and the vmi is in running state", virtv1.Running),
			Entry("and the vmi is in scheduled state", virtv1.Scheduled),
		)

		DescribeTable("should move the vmi to failed if pod is not handed over", func(phase k8sv1.PodPhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodTerminatingReason)
			vmi.Status.Phase = virtv1.Scheduling
			pod := NewPodForVirtualMachine(vmi, phase)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			shouldExpectVirtualMachineFailedState(vmi)

			controller.Execute()
		},
			Entry("and in succeeded state", k8sv1.PodSucceeded),
			Entry("and in failed state", k8sv1.PodFailed),
		)

		DescribeTable("should set a Synchronized=False condition when pod runs into image pull errors", func(initContainer bool, containerWaitingReason string) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodPending)

			containerStatus := &pod.Status.ContainerStatuses[0]
			if initContainer {
				pod.Status.InitContainerStatuses = []k8sv1.ContainerStatus{{}}
				containerStatus = &pod.Status.InitContainerStatuses[0]
			}
			containerStatus.State.Waiting = &k8sv1.ContainerStateWaiting{
				Reason: containerWaitingReason,
			}

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
						"Status": Equal(k8sv1.ConditionFalse),
						"Reason": Equal(containerWaitingReason),
					})))
			})

			controller.Execute()
		},
			Entry("ErrImagePull in init container", true, ErrImagePullReason),
			Entry("ImagePullBackOff in init container", true, ImagePullBackOffReason),
			Entry("ErrImagePull in compute container", false, ErrImagePullReason),
			Entry("ImagePullBackOff in compute container", false, ImagePullBackOffReason),
		)

		DescribeTable("should override Synchronized=False condition reason when it's already set", func(prevReason, newReason string) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodPending)

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
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Type":   Equal(virtv1.VirtualMachineInstanceSynchronized),
						"Status": Equal(k8sv1.ConditionFalse),
						"Reason": Equal(newReason),
					})))
			})

			controller.Execute()
		},
			Entry("ErrImagePull --> ImagePullBackOff", ErrImagePullReason, ImagePullBackOffReason),
			Entry("ImagePullBackOff --> ErrImagePull", ImagePullBackOffReason, ErrImagePullReason),
		)
		It("should add MigrationTransport to VMI status if MigrationTransportUnixAnnotation was set", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Annotations[virtv1.MigrationTransportUnixAnnotation] = "true"

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*virtv1.VirtualMachineInstance).Status.MigrationTransport).
					To(Equal(virtv1.MigrationTransportUnix))
			}).Return(vmi, nil)
			controller.Execute()
		})

		Context("should update pod labels", func() {

			type testData struct {
				vmiLabels     map[string]string
				podLabels     map[string]string
				expectedPatch string
			}
			DescribeTable("when VMI dynamic label set changes", func(td *testData) {
				vmi := NewPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running

				pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

				vmi.Labels = td.vmiLabels
				for key, val := range td.podLabels {
					pod.Labels[key] = val
				}

				addVirtualMachine(vmi)
				addActivePods(vmi, pod.UID, "")
				podFeeder.Add(pod)

				vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), &metav1.PatchOptions{}).Return(vmi, nil)

				patch := ""
				kubeClient.Fake.PrependReactor("patch", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
					p, _ := action.(testing.PatchAction)
					patch = string(p.GetPatch())
					return true, nil, nil
				})

				key := kvcontroller.VirtualMachineInstanceKey(vmi)
				err := controller.execute(key)
				Expect(err).ToNot(HaveOccurred())

				Expect(patch).To(Equal(td.expectedPatch))
			},
				Entry("when VMI and pod labels differ",
					&testData{
						vmiLabels: map[string]string{
							virtv1.NodeNameLabel: "node2",
						},
						podLabels: map[string]string{
							virtv1.NodeNameLabel: "node1",
						},
						expectedPatch: `[{ "op": "test", "path": "/metadata/labels", "value": {"kubevirt.io":"virt-launcher","kubevirt.io/created-by":"1234","kubevirt.io/nodeName":"node1"} }, { "op": "replace", "path": "/metadata/labels", "value": {"kubevirt.io":"virt-launcher","kubevirt.io/created-by":"1234","kubevirt.io/nodeName":"node2"} }]`,
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
						expectedPatch: "",
					},
				),
				Entry("when POD label doesn't exist",

					&testData{
						vmiLabels: map[string]string{
							virtv1.NodeNameLabel: "node1",
						},
						podLabels:     map[string]string{},
						expectedPatch: `[{ "op": "test", "path": "/metadata/labels", "value": {"kubevirt.io":"virt-launcher","kubevirt.io/created-by":"1234"} }, { "op": "replace", "path": "/metadata/labels", "value": {"kubevirt.io":"virt-launcher","kubevirt.io/created-by":"1234","kubevirt.io/nodeName":"node1"} }]`,
					},
				),
				Entry("when neither POD or VMI label exists",
					&testData{
						vmiLabels:     map[string]string{},
						podLabels:     map[string]string{},
						expectedPatch: "",
					},
				),
				Entry("when POD label exists and VMI does not",
					&testData{
						vmiLabels: map[string]string{},
						podLabels: map[string]string{
							virtv1.OutdatedLauncherImageLabel: "",
						},
						expectedPatch: `[{ "op": "test", "path": "/metadata/labels", "value": {"kubevirt.io":"virt-launcher","kubevirt.io/created-by":"1234","kubevirt.io/outdatedLauncherImage":""} }, { "op": "replace", "path": "/metadata/labels", "value": {"kubevirt.io":"virt-launcher","kubevirt.io/created-by":"1234"} }]`,
					},
				),
			)
		})

		DescribeTable("should set VirtualMachineUnpaused=False pod condition when VMI is paused", func(currUnpausedStatus k8sv1.ConditionStatus) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Running
			kvcontroller.NewVirtualMachineInstanceConditionManager().UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
				Type:   virtv1.VirtualMachineInstancePaused,
				Status: k8sv1.ConditionTrue,
			})

			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			kvcontroller.NewPodConditionManager().RemoveCondition(pod, virtv1.VirtualMachineUnpaused)
			if currUnpausedStatus != k8sv1.ConditionUnknown {
				pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{
					Type:   virtv1.VirtualMachineUnpaused,
					Status: currUnpausedStatus,
				})
			}

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), &metav1.PatchOptions{}).Return(vmi, nil)
			kubeClient.Fake.PrependReactor("patch", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				patch, _ := action.(testing.PatchAction)
				Expect(string(patch.GetPatch())).To(ContainSubstring(fmt.Sprintf(`"type":"%s"`, virtv1.VirtualMachineUnpaused)))
				Expect(string(patch.GetPatch())).To(ContainSubstring(fmt.Sprintf(`"status":"%s"`, k8sv1.ConditionFalse)))

				return true, nil, nil
			})

			controller.Execute()
		},
			Entry("when VirtualMachineUnpaused=True", k8sv1.ConditionTrue),
			Entry("when VirtualMachineUnpaused condition is unset", k8sv1.ConditionUnknown),
		)

		DescribeTable("should set VirtualMachineUnpaused=True pod condition when VMI is not paused", func(currUnpausedStatus k8sv1.ConditionStatus) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = virtv1.Running
			kvcontroller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, virtv1.VirtualMachineInstancePaused)

			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			kvcontroller.NewPodConditionManager().RemoveCondition(pod, virtv1.VirtualMachineUnpaused)
			if currUnpausedStatus != k8sv1.ConditionUnknown {
				pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{
					Type:   virtv1.VirtualMachineUnpaused,
					Status: currUnpausedStatus,
				})
			}

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), &metav1.PatchOptions{}).Return(vmi, nil)
			kubeClient.Fake.PrependReactor("patch", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				patch, _ := action.(testing.PatchAction)
				Expect(string(patch.GetPatch())).To(ContainSubstring(fmt.Sprintf(`"type":"%s"`, virtv1.VirtualMachineUnpaused)))
				Expect(string(patch.GetPatch())).To(ContainSubstring(fmt.Sprintf(`"status":"%s"`, k8sv1.ConditionTrue)))

				return true, nil, nil
			})

			controller.Execute()
		},
			Entry("when VirtualMachineUnpaused=True", k8sv1.ConditionTrue),
			Entry("when VirtualMachineUnpaused condition is unset", k8sv1.ConditionUnknown),
		)

		Context("with memory hotplug enabled", func() {
			It("should add MemoryChange conditon when guest memory changes", func() {
				currentGuestMemory := resource.MustParse("128Mi")
				requestedGuestMemory := resource.MustParse("512Mi")

				vmi := NewPendingVirtualMachine("testvmi")
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

				pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

				addVirtualMachine(vmi)
				podFeeder.Add(pod)
				addActivePods(vmi, pod.UID, "")

				vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), &metav1.PatchOptions{}).Do(func(ctx context.Context, name, patchType, patch, opts interface{}, subs ...interface{}) {
					originalVMIBytes, err := json.Marshal(vmi)
					Expect(err).ToNot(HaveOccurred())
					patchBytes := patch.([]byte)

					patchJSON, err := jsonpatch.DecodePatch(patchBytes)
					Expect(err).ToNot(HaveOccurred())
					newVMIBytes, err := patchJSON.Apply(originalVMIBytes)
					Expect(err).ToNot(HaveOccurred())

					var newVMI *virtv1.VirtualMachineInstance
					err = json.Unmarshal(newVMIBytes, &newVMI)
					Expect(err).ToNot(HaveOccurred())

					memoryChange := kvcontroller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(newVMI, virtv1.VirtualMachineInstanceMemoryChange, k8sv1.ConditionTrue)
					Expect(memoryChange).To(BeTrue())

				})

				controller.Execute()
			})

			It("should store guestMemoryOverheadRatio if used during memory hotplug", func() {
				currentGuestMemory := resource.MustParse("128Mi")
				requestedGuestMemory := resource.MustParse("512Mi")

				vmi := NewPendingVirtualMachine("testvmi")
				vmi.Status.Phase = virtv1.Running
				vmi.Status.Memory = &virtv1.MemoryStatus{
					GuestCurrent: &currentGuestMemory,
				}
				vmi.Spec.Domain.Memory = &virtv1.Memory{
					Guest:    &requestedGuestMemory,
					MaxGuest: &requestedGuestMemory,
				}
				kvCR := testutils.GetFakeKubeVirtClusterConfig(kvInformer)
				overheadRatio := "2"
				kvCR.Spec.Configuration.AdditionalGuestMemoryOverheadRatio = &overheadRatio
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvCR)

				controller.syncMemoryHotplug(vmi)

				Expect(vmi.Labels).To(HaveKeyWithValue(virtv1.MemoryHotplugOverheadRatioLabel, overheadRatio))
			})
		})
	})

	Context("hotplug volume", func() {
		It("Should find vmi, from virt-launcher pod", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			controllerRef := kvcontroller.GetControllerOf(pod)
			addVirtualMachine(vmi)

			result := controller.resolveControllerRef(k8sv1.NamespaceDefault, controllerRef)
			Expect(equality.Semantic.DeepEqual(result, vmi)).To(BeTrue(), "result: %v, should equal %v", result, vmi)
		})

		It("Should find vmi, from attachment pod", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			attachmentPod := NewPodForVirtlauncher(pod, "hp-test", "abcd", k8sv1.PodRunning)
			controllerRef := kvcontroller.GetControllerOf(attachmentPod)
			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			result := controller.resolveControllerRef(k8sv1.NamespaceDefault, controllerRef)
			Expect(equality.Semantic.DeepEqual(result, vmi)).To(BeTrue(), "result: %v, should equal %v", result, vmi)
		})

		It("DeleteAllAttachmentPods should return success if no virt launcher pod exists", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			addVirtualMachine(vmi)
			err := controller.deleteAllAttachmentPods(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("DeleteAllAttachmentPods should return success if there are no attachment pods", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
			err := controller.deleteAllAttachmentPods(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("DeleteAllAttachmentPods should return success if there are attachment pods", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			attachmentPod1 := NewPodForVirtlauncher(virtlauncherPod, "pod1", "abcd", k8sv1.PodRunning)
			attachmentPod2 := NewPodForVirtlauncher(virtlauncherPod, "pod2", "abcd", k8sv1.PodRunning)
			attachmentPod3 := NewPodForVirtlauncher(virtlauncherPod, "pod3", "abcd", k8sv1.PodRunning)
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
			podFeeder.Add(attachmentPod1)
			podFeeder.Add(attachmentPod2)
			podFeeder.Add(attachmentPod3)
			shouldExpectPodDeletion(attachmentPod3)
			shouldExpectPodDeletion(attachmentPod1)
			shouldExpectPodDeletion(attachmentPod2)
			err := controller.deleteAllAttachmentPods(vmi)
			testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)
			testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)
			testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)
			Expect(err).ToNot(HaveOccurred())
		})

		It("CreateAttachmentPodTemplate should return error if volume is not DV or PVC", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
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
			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, k8serrors.NewNotFound(k8sv1.Resource("persistentvolumeclaim"), "noclaim")
			})
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
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
			kubeClient.Fake.PrependReactor("get", "datavolumes", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, k8serrors.NewNotFound(k8sv1.Resource("datavolumes"), "test-dv")
			})
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pvc := NewHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimPending)
			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, pvc, nil
			})
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
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
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pvc := NewHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimPending)
			Expect(pvcInformer.GetIndexer().Add(pvc)).To(Succeed())
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dv",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Pending,
				},
			}
			Expect(dataVolumeInformer.GetIndexer().Add(dv)).To(Succeed())
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
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
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, virtv1.VolumeStatus{
				Name:          "test-pvc-volume",
				HotplugVolume: &virtv1.HotplugVolumeStatus{},
				Phase:         virtv1.VolumePending,
				Message:       "some technical reason",
				Reason:        PVCNotReadyReason,
			})
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pvc := NewHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimBound)
			Expect(pvcInformer.GetIndexer().Add(pvc)).To(Succeed())
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dv",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Pending,
				},
			}
			Expect(dataVolumeInformer.GetIndexer().Add(dv)).To(Succeed())
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
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
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.SelinuxContext = "system_u:system_r:container_file_t:s0:c1,c2"
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pvc := NewHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimBound)
			Expect(pvcInformer.GetIndexer().Add(pvc)).To(Succeed())
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dv",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Succeeded,
				},
			}
			Expect(dataVolumeInformer.GetIndexer().Add(dv)).To(Succeed())
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
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
			pod := NewPodForVirtlauncher(virtlauncherPod, "test-pod", "abcd", k8sv1.PodRunning)
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
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			return makePodWithVirtlauncher(virtlauncherPod, indexes...)
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
			for i := 0; i < total; i++ {
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

		DescribeTable("should calculate deleted volumes based on current pods and volumes", func(hotplugPods []*k8sv1.Pod, hotplugVolumes []*virtv1.Volume, expected []k8sv1.Volume) {
			res := controller.getDeletedHotplugVolumes(hotplugPods, hotplugVolumes)
			Expect(res).To(HaveLen(len(expected)))
			for i, volume := range res {
				Expect(equality.Semantic.DeepEqual(volume, expected[i])).To(BeTrue(), "%v does not match %v", volume, expected[i])
			}
		},
			Entry("should return empty if pods and volumes is empty", make([]*k8sv1.Pod, 0), make([]*virtv1.Volume, 0), make([]k8sv1.Volume, 0)),
			Entry("should return empty if pods and volumes match", makePods(1), makeVolumes(1), make([]k8sv1.Volume, 0)),
			Entry("should return empty if pods < volumes", makePods(1), makeVolumes(1, 2), make([]k8sv1.Volume, 0)),
		)

		DescribeTable("should calculate new volumes based on current pods and volumes", func(hotplugPods []*k8sv1.Pod, hotplugVolumes []*virtv1.Volume, expected []*virtv1.Volume) {
			res := controller.getNewHotplugVolumes(hotplugPods, hotplugVolumes)
			Expect(res).To(HaveLen(len(expected)))
			for i, volume := range res {
				Expect(equality.Semantic.DeepEqual(volume, expected[i])).To(BeTrue(), "%v does not match %v", volume, expected[i])
			}
		},
			Entry("should return empty if pods and volumes is empty", make([]*k8sv1.Pod, 0), make([]*virtv1.Volume, 0), make([]*virtv1.Volume, 0)),
			Entry("should return empty if pods and volumes match", makePods(1), makeVolumes(1), make([]*virtv1.Volume, 0)),
			Entry("should return empty if pods > volumes", makePods(1), makeVolumes(1, 2), makeVolumes(2)),
			Entry("should return 1 value if pods < volumes by 1", makePods(1, 2), makeVolumes(1), make([]*virtv1.Volume, 0)),
		)

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
				pvc := NewHotplugPVC(fmt.Sprintf("claim%d", index), k8sv1.NamespaceDefault, k8sv1.ClaimBound)
				kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
					return true, pvc, nil
				})
				dv := &cdiv1.DataVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("claim%d", index),
						Namespace: k8sv1.NamespaceDefault,
					},
					Status: cdiv1.DataVolumeStatus{
						Phase: cdiv1.Succeeded,
					},
				}
				Expect(dataVolumeInformer.GetIndexer().Add(dv)).To(Succeed())
				Expect(pvcInformer.GetIndexer().Add(pvc)).To(Succeed())
			}
		}

		DescribeTable("handleHotplugVolumes should properly react to input", func(hotplugVolumes []*virtv1.Volume, hotplugAttachmentPods []*k8sv1.Pod, createPodReaction func(*k8sv1.Pod, ...int), pvcFunc func(...int), pvcIndexes []int, orgStatus []virtv1.VolumeStatus, expectedEvent string, expectedErr syncError) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.VolumeStatus = orgStatus
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
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
			podFeeder.Add(virtlauncherPod)
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
			res := controller.needsHandleHotplug(hotplugVolumes, hotplugAttachmentPods)
			Expect(res).To(Equal(expected))
		},
			Entry("should return false if volumes and attachmentpods are empty", makeVolumes(), makePods(), false),
			Entry("should return false if volumes and attachmentpods match", makeVolumes(1), makePods(1), false),
			Entry("should return true if volumes > attachmentpods", makeVolumes(1, 2), makePods(1), true),
			Entry("should return true if volumes < attachmentpods", makeVolumes(1), makePods(1, 2), true),
			Entry("should return true if len(volumes) == len(attachmentpods), but contents differ", makeVolumes(1, 3), makePods(1, 2), true),
		)

		DescribeTable("virtlauncherAttachmentPods", func(podCount int) {
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			for i := 0; i < podCount; i++ {
				attachmentPod := NewPodForVirtlauncher(virtlauncherPod, fmt.Sprintf("test-pod%d", i), fmt.Sprintf("abcd%d", i), k8sv1.PodRunning)
				Expect(podInformer.GetIndexer().Add(attachmentPod)).To(Succeed())
			}
			// Add some non owned pods.
			for i := 0; i < 5; i++ {
				pod := &k8sv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("unowned-test-pod%d", i),
						Namespace: vmi.Namespace,
					},
					Status: k8sv1.PodStatus{
						Phase: k8sv1.PodRunning,
					},
				}
				Expect(podInformer.GetIndexer().Add(pod)).To(Succeed())
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
			Expect(podInformer.GetIndexer().Add(otherPod)).To(Succeed())
			for i := 10; i < 10+podCount; i++ {
				attachmentPod := NewPodForVirtlauncher(otherPod, fmt.Sprintf("test-pod%d", i), fmt.Sprintf("abcde%d", i), k8sv1.PodRunning)
				Expect(podInformer.GetIndexer().Add(attachmentPod)).To(Succeed())
			}

			res, err := kvcontroller.AttachmentPods(virtlauncherPod, podInformer)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(podCount))
		},
			Entry("should return number (0) of pods passed in", 0),
			Entry("should return number (1) of pods passed in", 1),
			Entry("should return number (2) of pods passed in", 2),
		)

		DescribeTable("getHotplugVolumes", func(virtlauncherVolumes []k8sv1.Volume, vmiVolumes []*virtv1.Volume, expectedIndexes ...int) {
			vmi := NewPendingVirtualMachine("testvmi")
			for _, volume := range vmiVolumes {
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, *volume)
			}
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			virtlauncherPod.Spec.Volumes = virtlauncherVolumes
			res := getHotplugVolumes(vmi, virtlauncherPod)
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

		truncateSprintf := func(str string, args ...interface{}) string {
			n := strings.Count(str, "%d")
			return fmt.Sprintf(str, args[:n]...)
		}

		makeVolumeStatusesForUpdateWithMessage := func(podName, podUID string, phase virtv1.VolumePhase, message, reason string, indexes ...int) []virtv1.VolumeStatus {
			res := make([]virtv1.VolumeStatus, 0)
			for _, index := range indexes {
				fsOverhead := storagetypes.DefaultFSOverhead
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
						AccessModes: []k8sv1.PersistentVolumeAccessMode{
							k8sv1.ReadOnlyMany,
						},
						FilesystemOverhead: &fsOverhead,
					},
				})
			}
			return res
		}

		makeVolumeStatusesForUpdateWithMemoryDump := func(dumpIndex int, indexes ...int) []virtv1.VolumeStatus {
			res := makeVolumeStatusesForUpdateWithMessage("test-pod", "abcd", virtv1.HotplugVolumeAttachedToNode, "Created hotplug attachment pod test-pod, for volume volume%d", SuccessfulCreatePodReason, indexes...)
			res[dumpIndex].MemoryDumpVolume = &virtv1.DomainMemoryDumpInfo{
				ClaimName: fmt.Sprintf("volume%d", dumpIndex),
			}
			return res
		}

		makeVolumeStatusesForUpdate := func(indexes ...int) []virtv1.VolumeStatus {
			return makeVolumeStatusesForUpdateWithMessage("test-pod", "abcd", virtv1.HotplugVolumeAttachedToNode, "Created hotplug attachment pod test-pod, for volume volume%d", SuccessfulCreatePodReason, indexes...)
		}

		DescribeTable("updateVolumeStatus", func(oldStatus []virtv1.VolumeStatus, specVolumes []*virtv1.Volume, podIndexes []int, pvcIndexes []int, expectedStatus []virtv1.VolumeStatus, expectedEvents []string) {
			vmi := NewPendingVirtualMachine("testvmi")
			volumes := make([]virtv1.Volume, 0)
			for _, volume := range specVolumes {
				volumes = append(volumes, *volume)
			}
			vmi.Spec.Volumes = volumes
			vmi.Status.VolumeStatus = oldStatus
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			attachmentPods := makePodWithVirtlauncher(virtlauncherPod, podIndexes...)
			now := metav1.Now()
			for _, pod := range attachmentPods {
				pod.DeletionTimestamp = &now
				Expect(podInformer.GetIndexer().Add(pod)).To(Succeed())
			}
			for _, pvcIndex := range pvcIndexes {
				pvc := NewHotplugPVC(fmt.Sprintf("claim%d", pvcIndex), k8sv1.NamespaceDefault, k8sv1.ClaimBound)
				Expect(pvcInformer.GetIndexer().Add(pvc)).To(Succeed())
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
				makeVolumeStatusesForUpdate(0),
				[]string{SuccessfulCreatePodReason}),
			Entry("should update volume status, if a new volume is added, and pod does not exist",
				makeVolumeStatusesForUpdate(),
				makeVolumes(0),
				[]int{},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("", "", virtv1.VolumeBound, "PVC is in phase Bound", PVCNotReadyReason, 0),
				[]string{}),
			Entry("should update volume status, if a existing volume is changed, and pod does not exist",
				makeVolumeStatusesForUpdateWithMessage("", "", virtv1.VolumePending, "PVC is in phase Pending", PVCNotReadyReason, 0),
				makeVolumes(0),
				[]int{},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("", "", virtv1.VolumeBound, "PVC is in phase Bound", PVCNotReadyReason, 0),
				[]string{}),
			Entry("should keep status, if volume removed and if pod still exists",
				makeVolumeStatusesForUpdate(0),
				makeVolumes(),
				[]int{0},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("test-pod", "abcd", virtv1.HotplugVolumeDetaching, "Deleted hotplug attachment pod test-pod, for volume volume0", SuccessfulDeletePodReason, 0),
				[]string{SuccessfulDeletePodReason}),
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
				[]string{SuccessfulCreatePodReason}),
		)

		DescribeTable("Should properly calculate if it needs to handle hotplug volumes", func(hotplugVolumes []*virtv1.Volume, attachmentPods []*k8sv1.Pod, match gomegaTypes.GomegaMatcher) {
			Expect(controller.needsHandleHotplug(hotplugVolumes, attachmentPods)).To(match)
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
			active, old := controller.getActiveAndOldAttachmentPods(hotplugVolumes, attachmentPods)
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
			Expect(cdiInformer.GetIndexer().Add(&cdi)).To(Succeed())
			Expect(cdiInformer.GetIndexer().Add(&cdi2)).To(Succeed())

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

			Expect(cdiInformer.GetIndexer().Add(&cdi)).To(Succeed())

			fsOverhead, err := controller.getFilesystemOverhead(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Failed to find CDIConfig but CDI exists"))
			Expect(fsOverhead).To(Equal(cdiv1.Percent("0")))
		})

		DescribeTable("getFilesystemOverhead", func(volumeMode k8sv1.PersistentVolumeMode, scName string, expectedOverhead cdiv1.Percent) {
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

			Expect(cdiInformer.GetIndexer().Add(&cdi)).To(Succeed())
			Expect(cdiConfigInformer.GetIndexer().Add(&cfg)).To(Succeed())

			fsOverhead, err := controller.getFilesystemOverhead(pvc)
			Expect(err).ToNot(HaveOccurred())
			Expect(fsOverhead).To(Equal(expectedOverhead))
		},
			Entry("should not get any fs overhead if the PVC has volumeMode block", k8sv1.PersistentVolumeBlock, "default", cdiv1.Percent("0")),
			Entry("should return global fs overhead if there's no storageClassName", k8sv1.PersistentVolumeFilesystem, "", cdiv1.Percent("0.5")),
			Entry("should return global fs overhead if the storageClassName is invalid", k8sv1.PersistentVolumeFilesystem, "nonValid", cdiv1.Percent("0.5")),
			Entry("should return the appropiate overhead when using a valid storageClassName", k8sv1.PersistentVolumeFilesystem, "default", cdiv1.Percent("0.8")),
		)

		It("Should properly create attachmentpod, if correct volume and disk are added", func() {
			vmi := NewPendingVirtualMachine("testvmi")
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
						Bus: v1.DiskBusVirtio,
					},
				},
			})
			disks = append(disks, virtv1.Disk{
				Name: "hotplug",
				DiskDevice: virtv1.DiskDevice{
					Disk: &virtv1.DiskTarget{
						Bus: v1.DiskBusSATA,
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = disks
			vmi.Status.Phase = virtv1.Running
			vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, virtv1.VolumeStatus{
				Name:   "existing",
				Target: "",
			})
			vmi.Status.ActivePods["virt-launch-uid"] = ""
			vmi.Status.SelinuxContext = "system_u:system_r:container_file_t:s0:c1,c2"
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

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
			Expect(pvcInformer.GetIndexer().Add(existingPVC)).To(Succeed())
			Expect(pvcInformer.GetIndexer().Add(hpPVC)).To(Succeed())
			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, existingPVC, nil
			})
			shouldExpectHotplugPod()
			addVirtualMachine(vmi)
			Expect(podInformer.GetIndexer().Add(virtlauncherPod)).To(Succeed())
			//Modify by adding a new hotplugged disk
			patch := `[{ "op": "test", "path": "/status/volumeStatus", "value": [{"name":"existing","target":""}] }, { "op": "replace", "path": "/status/volumeStatus", "value": [{"name":"existing","target":"","persistentVolumeClaimInfo":{"filesystemOverhead":"0.055"}},{"name":"hotplug","target":"","phase":"Bound","reason":"PVCNotReady","message":"PVC is in phase Bound","persistentVolumeClaimInfo":{"filesystemOverhead":"0.055"},"hotplugVolume":{}}] }]`
			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vmi, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
			Expect(vmi.Status.Phase).To(Equal(virtv1.Running))
		})

		It("Should properly delete attachment, if volume and disk are removed", func() {
			vmi := NewPendingVirtualMachine("testvmi")
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
						Bus: v1.DiskBusVirtio,
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = disks
			vmi.Status.Phase = virtv1.Running
			vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, virtv1.VolumeStatus{
				Name:   "existing",
				Target: "",
			})
			vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, virtv1.VolumeStatus{
				Name:   "hotplug",
				Target: "",
				HotplugVolume: &virtv1.HotplugVolumeStatus{
					AttachPodName: "hp-volume-hotplug",
					AttachPodUID:  "abcd",
				},
			})
			vmi.Status.ActivePods["virt-launch-uid"] = ""
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

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
			hpPod := NewPodForVirtlauncher(virtlauncherPod, "hp-volume-hotplug", "abcd", k8sv1.PodRunning)
			hpPod.Spec.Volumes = append(hpPod.Spec.Volumes, k8sv1.Volume{
				Name: "hotplug",
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{},
				},
			})
			Expect(podInformer.GetIndexer().Add(hpPod)).To(Succeed())
			Expect(pvcInformer.GetIndexer().Add(existingPVC)).To(Succeed())
			Expect(pvcInformer.GetIndexer().Add(hpPVC)).To(Succeed())
			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, existingPVC, nil
			})
			shouldExpectPodDeletion(hpPod)
			addVirtualMachine(vmi)
			Expect(podInformer.GetIndexer().Add(virtlauncherPod)).To(Succeed())
			//Modify by adding a new hotplugged disk
			patch := `[{ "op": "test", "path": "/status/volumeStatus", "value": [{"name":"existing","target":""},{"name":"hotplug","target":"","hotplugVolume":{"attachPodName":"hp-volume-hotplug","attachPodUID":"abcd"}}] }, { "op": "replace", "path": "/status/volumeStatus", "value": [{"name":"existing","target":"","persistentVolumeClaimInfo":{"filesystemOverhead":"0.055"}},{"name":"hotplug","target":"","phase":"Detaching","hotplugVolume":{"attachPodName":"hp-volume-hotplug","attachPodUID":"abcd"}}] }]`
			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &metav1.PatchOptions{}).Return(vmi, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)
			Expect(vmi.Status.Phase).To(Equal(virtv1.Running))
		})
	})

	Context("topology hints", func() {

		getVmiWithInvTsc := func() *virtv1.VirtualMachineInstance {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Spec.Architecture = "amd64"
			vmi.Spec.Domain.CPU = &v1.CPU{
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
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Spec.Architecture = "amd64"
			vmi.Spec.Domain.Features = &v1.Features{
				Hyperv: &v1.FeatureHyperv{
					Reenlightenment: &v1.FeatureState{Enabled: pointer.Bool(true)},
				},
			}

			return vmi
		}

		expectTopologyHintsUpdate := func() {
			var vmi *virtv1.VirtualMachineInstance
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				vmi = arg.(*virtv1.VirtualMachineInstance)
				Expect(topology.AreTSCFrequencyTopologyHintsDefined(vmi)).To(BeTrue())
			}).Return(vmi, nil)
		}

		Context("needs to be set when", func() {

			runController := func(vmi *virtv1.VirtualMachineInstance) {
				addVirtualMachine(vmi)
				shouldExpectPodCreation(vmi.UID)
				controller.Execute()
			}

			It("invtsc feature exists", func() {
				vmi := getVmiWithInvTsc()

				expectTopologyHintsUpdate()
				runController(vmi)
			})

			Context("HyperV reenlightenment is enabled", func() {
				var vmi *v1.VirtualMachineInstance

				BeforeEach(func() {
					vmi = getVmiWithReenlightenment()
				})

				When("TSC frequency is exposed", func() {
					It("topology hints need to be set", func() {
						expectTopologyHintsUpdate()
						runController(vmi)
					})
				})

				When("TSC frequency is not exposed", func() {
					unexposeTscFrequency := func(requirement topology.TscFrequencyRequirementType) {
						mockHinter := topology.NewMockHinter(ctrl)
						mockHinter.EXPECT().TopologyHintsForVMI(gomock.Any()).Return(nil, requirement, fmt.Errorf("tsc frequency is not exposed on the cluster")).AnyTimes()
						mockHinter.EXPECT().IsTscFrequencyRequired(gomock.Any()).Return(requirement == topology.RequiredForBoot)
						controller.topologyHinter = mockHinter
					}

					It("topology hints don't need to be set and VMI needs to be non-migratable", func() {
						unexposeTscFrequency(topology.RequiredForMigration)
						// Make sure no update occurs
						runController(vmi)

						testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
					})
				})

			})

		})

		Context("pod creation", func() {

			runController := func(vmi *virtv1.VirtualMachineInstance) {
				addVirtualMachine(vmi)
				controller.Execute()
			}

			It("does not need to happen if tsc requiredment is of type RequiredForBoot", func() {
				vmi := getVmiWithInvTsc()
				Expect(topology.GetTscFrequencyRequirement(vmi).Type).To(Equal(topology.RequiredForBoot))

				expectTopologyHintsUpdate()
				runController(vmi)
			})

			It("does not need to happen if tsc requiredment is of type RequiredForMigration", func() {
				vmi := getVmiWithReenlightenment()
				Expect(topology.GetTscFrequencyRequirement(vmi).Type).To(Equal(topology.RequiredForMigration))

				expectTopologyHintsUpdate()
				runController(vmi)
			})

			It("does not need to happen if tsc requiredment is of type NotRequired", func() {
				vmi := NewPendingVirtualMachine("testvmi")
				Expect(topology.GetTscFrequencyRequirement(vmi).Type).To(Equal(topology.NotRequired))

				shouldExpectPodCreation(vmi.UID)
				runController(vmi)
				testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
			})
		})
	})

	Context("auto attach VSOCK", func() {
		It("should allocate CID when VirtualMachineInstance is scheduled", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.GuestNotRunningReason)
			vmi.Status.Phase = virtv1.Scheduling
			vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.Bool(true)
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg *virtv1.VirtualMachineInstance) {
				Expect(arg.Status.Phase).To(Equal(virtv1.Scheduled))
				Expect(arg.Status.PhaseTransitionTimestamps).ToNot(BeEmpty())
				Expect(arg.Status.PhaseTransitionTimestamps).ToNot(HaveLen(len(vmi.Status.PhaseTransitionTimestamps)))
				Expect(arg.Status.VSOCKCID).NotTo(BeNil())
			}).Return(vmi, nil)
			controller.Execute()
		})

		It("should recycle the CID when the pods are deleted", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.Bool(true)
			Expect(controller.cidsMap.Allocate(vmi)).To(Succeed())
			vmi.Status.Phase = virtv1.Succeeded
			addVirtualMachine(vmi)

			Expect(vmiInformer.GetIndexer().Delete(vmi)).To(Succeed())
			controller.Execute()

			Expect(controller.cidsMap.cids).To(BeEmpty())
			Expect(controller.cidsMap.reverse).To(BeEmpty())
		})
	})

	Context("dynamic interface attachment", func() {
		const vmName = "vm1"

		var (
			pod *k8sv1.Pod
			vmi *virtv1.VirtualMachineInstance
		)

		expectPodStatusUpdateFailed := func(pod *k8sv1.Pod) {
			kubeClient.Fake.PrependReactor("patch", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				_, ok := action.(testing.PatchAction)
				Expect(ok).To(BeTrue())
				return true, pod, fmt.Errorf("your cluster is doomed")
			})
		}

		const (
			firstVMInterface  = "oldIface1"
			firstVMNetwork    = "oldnet1"
			secondVMInterface = "oldiface2"
			secondVMNetwork   = "oldnet2"
		)

		Context("k8s API is down", func() {
			BeforeEach(func() {
				vmi = appendNetworkToVMI(
					appendNetworkToVMI(
						api.NewMinimalVMI(vmName), firstVMNetwork, firstVMInterface),
					secondVMNetwork, secondVMInterface)

				var err error
				pod, err = NewPodForVirtualMachineWithMultusAnnotations(vmi, k8sv1.PodRunning, config)
				Expect(err).ToNot(HaveOccurred())

				expectPodStatusUpdateFailed(pod)
			})

			It("pod multus status cannot be updated", func() {
				Expect(controller.updateMultusAnnotation(
					vmi.Namespace, vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, pod)).To(HaveOccurred())
			})
		})

		Context("pod status update", func() {
			const mac1 = "abc"
			const mac2 = "bcd"
			BeforeEach(func() {
				vmi = api.NewMinimalVMI(vmName)
				pod = NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
				Expect(pod.Annotations).NotTo(HaveKey(networkv1.NetworkAttachmentAnnot))
				prependInjectPodPatch(pod)
			})

			DescribeTable("update pods network annotation with", func(networks []v1.Network, interfaces []v1.Interface, matchers ...gomegaTypes.GomegaMatcher) {
				vmi.Spec.Networks = networks
				vmi.Spec.Domain.Devices.Interfaces = interfaces
				Expect(controller.updateMultusAnnotation(
					vmi.Namespace, vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, pod)).To(Succeed())
				for _, matcher := range matchers {
					Expect(pod.Annotations).To(matcher)
				}
			},
				Entry("a single interface",
					[]v1.Network{{
						Name:          "iface1",
						NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "net1"}},
					}},
					[]v1.Interface{{
						Name: "iface1",
					}},
					HaveKeyWithValue(
						networkv1.NetworkAttachmentAnnot,
						`[{"name":"net1","namespace":"default","interface":"pod7e0055a6880"}]`)),
				Entry("multiple interfaces",
					[]v1.Network{{
						Name:          "iface1",
						NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "net1"}},
					}, {
						Name:          "iface2",
						NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "net1"}},
					}},
					[]v1.Interface{{
						Name:       "iface1",
						MacAddress: "mac1",
					}, {
						Name:       "iface2",
						MacAddress: "mac2",
					}},
					HaveKeyWithValue(
						networkv1.NetworkAttachmentAnnot,
						`[{"name":"net1","namespace":"default","mac":"mac1","interface":"pod7e0055a6880"},{"name":"net1","namespace":"default","mac":"mac2","interface":"pod48802102d24"}]`)),
			)
			DescribeTable("the subject interface name, in the pod networks annotation, should be in similar form as other interfaces",
				func(testPodNetworkStatus []networkv1.NetworkStatus, expectedMultusNetworksAnnotation string) {
					vmi = api.NewMinimalVMI(vmName)
					vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
						Name:                   "red",
						InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}, {
						Name:                   "blue",
						InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}

					vmi.Spec.Networks = []v1.Network{{
						Name:          "red",
						NetworkSource: v1.NetworkSource{Multus: &virtv1.MultusNetwork{NetworkName: "red-net"}}}, {
						Name:          "blue",
						NetworkSource: v1.NetworkSource{Multus: &virtv1.MultusNetwork{NetworkName: "blue-net"}}}}

					pod, err := NewPodForVirtualMachineWithMultusAnnotations(vmi, k8sv1.PodRunning, config, testPodNetworkStatus...)
					Expect(err).ToNot(HaveOccurred())

					prependInjectPodPatch(pod)

					Expect(controller.updateMultusAnnotation(vmi.Namespace, vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, pod)).To(Succeed())

					Expect(pod.Annotations).To(HaveKey(networkv1.NetworkAttachmentAnnot))
					Expect(pod.Annotations[networkv1.NetworkAttachmentAnnot]).To(MatchJSON(expectedMultusNetworksAnnotation))
				},
				Entry("when Multus network-status annotation interfaces has ordinal names",
					[]networkv1.NetworkStatus{
						{
							Interface: "net1", Name: "red-net",
						},
					},
					// expected Multus network annotation
					`[
							{"interface":"net1", "name":"red-net", "namespace": "default"},
							{"interface":"net2", "name":"blue-net", "namespace": "default"}
					]`,
				),
				Entry("when Multus network-status annotation interfaces has hashed names",
					[]networkv1.NetworkStatus{
						{
							Interface: "podb1f51a511f1", Name: "red-net",
						},
					},
					// expected Multus network annotation
					`[
							{"interface":"podb1f51a511f1", "name":"red-net", "namespace": "default"},
							{"interface":"pod16477688c0e", "name":"blue-net", "namespace": "default"}
					]`,
				),
			)
		})

		Context("interface status", func() {
			const (
				ifaceName   = "iface1"
				networkName = "meganet"
			)

			DescribeTable("updateInterfaceStatus", func(vmi *virtv1.VirtualMachineInstance, ifaceStatus ...PodVmIfaceStatus) {
				var (
					podIfaceStatus []networkv1.NetworkStatus
					vmIfaceStatus  []virtv1.VirtualMachineInstanceNetworkInterface
				)
				for i, ifaceState := range ifaceStatus {
					if ifaceState.podIfaceStatus != nil {
						podIfaceStatus = append(podIfaceStatus, *ifaceStatus[i].podIfaceStatus)
					}
					if ifaceState.vmIfaceStatus != nil {
						vmIfaceStatus = append(vmIfaceStatus, *ifaceStatus[i].vmIfaceStatus)
					}
				}
				pod, err := NewPodForVirtualMachineWithMultusAnnotations(vmi, k8sv1.PodRunning, config, podIfaceStatus...)
				Expect(err).ToNot(HaveOccurred())

				Expect(controller.updateInterfaceStatus(vmi, pod)).To(Succeed())
				Expect(vmi.Status.Interfaces).To(ConsistOf(vmIfaceStatus))
			},
				Entry("VMI without interfaces on spec does not generate new interface status", api.NewMinimalVMI(vmName)),
				Entry("VMI with an interface on spec (not matched on status) generates new interface status",
					newVMIWithOneIface(api.NewMinimalVMI(vmName), networkName, ifaceName),
					PodVmIfaceStatus{}),
				Entry("VMI with an interface on spec (matched on status) does not generate new interface status",
					newVMIWithOneIfaceStatus(newVMIWithOneIface(api.NewMinimalVMI(vmName), networkName, ifaceName), ifaceName),
					PodVmIfaceStatus{
						vmIfaceStatus: simpleIfaceStatus(ifaceName),
					}),
				Entry("VMI with an interface on spec (matched on status) with the pod interface ready",
					newVMIWithOneIface(api.NewMinimalVMI(vmName), networkName, ifaceName),
					PodVmIfaceStatus{
						vmIfaceStatus: readyHotpluggedIfaceStatus(ifaceName),
						podIfaceStatus: &networkv1.NetworkStatus{
							Name:      networkName,
							Interface: "pod7e0055a6880",
						},
					}),
				Entry("VMI with an interface on spec, (matched on status), with the pod interface ready and ordinal name",
					newVMIWithOneIface(api.NewMinimalVMI(vmName), networkName, ifaceName),
					PodVmIfaceStatus{
						vmIfaceStatus: readyHotpluggedIfaceStatus(ifaceName),
						podIfaceStatus: &networkv1.NetworkStatus{
							Name:      networkName,
							Interface: "net1",
						},
					}),
				Entry("VMI with a guest agent interface",
					newVMIWithGuestAgentInterface(
						newVMIWithOneIface(api.NewMinimalVMI(vmName), networkName, ifaceName),
						ifaceName,
					),
					PodVmIfaceStatus{
						vmIfaceStatus: &virtv1.VirtualMachineInstanceNetworkInterface{
							InterfaceName: ifaceName,
							InfoSource:    vmispec.InfoSourceGuestAgent,
						},
					}),
				Entry("VMI with an interface on spec (matched on status) with pod interface *not* ready",
					newVMIWithOneIfaceStatus(newVMIWithOneIface(api.NewMinimalVMI(vmName), networkName, ifaceName), ifaceName),
					PodVmIfaceStatus{
						vmIfaceStatus: simpleIfaceStatus(ifaceName),
					}),
			)
		})
	})
})

func NewDv(namespace string, name string, phase cdiv1.DataVolumePhase) *cdiv1.DataVolume {
	return &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: cdiv1.DataVolumeStatus{
			Phase: phase,
		},
	}
}

func NewPvc(namespace string, name string) *k8sv1.PersistentVolumeClaim {
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

func NewPvcWithOwner(namespace string, name string, ownerName string, isController *bool) *k8sv1.PersistentVolumeClaim {
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

func NewPendingVirtualMachine(name string) *virtv1.VirtualMachineInstance {
	vmi := api.NewMinimalVMI(name)
	vmi.UID = "1234"
	vmi.Status.Phase = virtv1.Pending
	setReadyCondition(vmi, k8sv1.ConditionFalse, virtv1.PodNotExistsReason)
	kvcontroller.SetLatestApiVersionAnnotation(vmi)
	vmi.Status.ActivePods = make(map[types.UID]string)
	vmi.Status.VolumeStatus = make([]virtv1.VolumeStatus, 0)
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

func NewPodForVirtualMachine(vmi *virtv1.VirtualMachineInstance, phase k8sv1.PodPhase) *k8sv1.Pod {
	podAnnotations := map[string]string{
		virtv1.DomainAnnotation: vmi.Name,
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

func NewPodForVirtualMachineWithMultusAnnotations(vmi *virtv1.VirtualMachineInstance, phase k8sv1.PodPhase, config *virtconfig.ClusterConfig, podNetworkStatus ...networkv1.NetworkStatus) (*k8sv1.Pod, error) {
	pod := NewPodForVirtualMachine(vmi, phase)

	multusAnnotations, err := services.GenerateMultusCNIAnnotation(vmi.Namespace, vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, config)
	if err != nil {
		return nil, err
	}
	pod.Annotations[networkv1.NetworkAttachmentAnnot] = multusAnnotations

	if len(podNetworkStatus) > 0 {
		podCurrentNetworksJSON, err := json.Marshal(podNetworkStatus)
		if err != nil {
			return nil, err
		}
		pod.Annotations[networkv1.NetworkStatusAnnot] = string(podCurrentNetworksJSON)
	}

	return pod, nil
}

func NewHotplugPVC(name, namespace string, phase k8sv1.PersistentVolumeClaimPhase) *k8sv1.PersistentVolumeClaim {
	t := true
	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1alpha1",
					Kind:       "DataVolume",
					Name:       name,
					Controller: &t,
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

func NewPodForVirtlauncher(virtlauncher *k8sv1.Pod, name, uid string, phase k8sv1.PodPhase) *k8sv1.Pod {
	t := true
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
					Controller: &t,
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

func now() *metav1.Time {
	now := metav1.Now()
	return &now
}

func markAsReady(vmi *virtv1.VirtualMachineInstance) {
	kvcontroller.NewVirtualMachineInstanceConditionManager().AddPodCondition(vmi, &k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})
}

func markAsPodTerminating(vmi *virtv1.VirtualMachineInstance) {
	kvcontroller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodReady))
	kvcontroller.NewVirtualMachineInstanceConditionManager().AddPodCondition(vmi, &k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionFalse, Reason: virtv1.PodTerminatingReason})
}

func markAsNonReady(vmi *virtv1.VirtualMachineInstance) {
	kvcontroller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodReady))
	kvcontroller.NewVirtualMachineInstanceConditionManager().AddPodCondition(vmi, &k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionFalse})
}

func unmarkReady(vmi *virtv1.VirtualMachineInstance) {
	kvcontroller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodReady))
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

func addDefaultNetwork(vmi *virtv1.VirtualMachineInstance, networkName string) *virtv1.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, newMasqueradePrimaryInterface(networkName))
	vmi.Spec.Networks = append(vmi.Spec.Networks, newMasqueradeDefaultNetwork(networkName))
	return vmi
}

func addDefaultNetworkStatus(vmi *virtv1.VirtualMachineInstance, networkName string) *virtv1.VirtualMachineInstance {
	vmi.Status.Interfaces = append(vmi.Status.Interfaces, virtv1.VirtualMachineInstanceNetworkInterface{Name: networkName})
	return vmi
}

func addSRIOVNetwork(vmi *virtv1.VirtualMachineInstance, networkName, nadName string) *virtv1.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, newSRIOVInterface(networkName))
	vmi.Spec.Networks = append(vmi.Spec.Networks, newMultusNetwork(networkName, nadName))
	return vmi
}

func newSRIOVInterface(name string) virtv1.Interface {
	return virtv1.Interface{
		Name:                   name,
		InterfaceBindingMethod: virtv1.InterfaceBindingMethod{SRIOV: &virtv1.InterfaceSRIOV{}},
	}
}

func newMasqueradePrimaryInterface(name string) virtv1.Interface {
	return virtv1.Interface{
		Name:                   name,
		InterfaceBindingMethod: virtv1.InterfaceBindingMethod{Masquerade: &virtv1.InterfaceMasquerade{}},
	}
}

func newMasqueradeDefaultNetwork(name string) virtv1.Network {
	return virtv1.Network{
		Name: name,
		NetworkSource: virtv1.NetworkSource{
			Pod: &virtv1.PodNetwork{},
		},
	}
}

func newMultusNetwork(name, networkName string) virtv1.Network {
	return virtv1.Network{
		Name: name,
		NetworkSource: virtv1.NetworkSource{
			Multus: &virtv1.MultusNetwork{
				NetworkName: networkName,
			},
		},
	}
}

func appendNetworkToVMI(vmi *virtv1.VirtualMachineInstance, netAttachDefName string, name string) *virtv1.VirtualMachineInstance {
	vmi.Spec.Networks = append(
		vmi.Spec.Networks,
		virtv1.Network{
			Name: name,
			NetworkSource: virtv1.NetworkSource{
				Multus: &virtv1.MultusNetwork{
					NetworkName: netAttachDefName,
				},
			},
		})
	return vmi
}

func newVMIWithOneIface(vmi *virtv1.VirtualMachineInstance, networkName string, ifaceName string) *virtv1.VirtualMachineInstance {
	vmi.Spec.Networks = append(
		vmi.Spec.Networks,
		virtv1.Network{
			Name: ifaceName,
			NetworkSource: virtv1.NetworkSource{
				Multus: &virtv1.MultusNetwork{
					NetworkName: networkName,
				},
			},
		})
	vmi.Spec.Domain.Devices.Interfaces = append(
		vmi.Spec.Domain.Devices.Interfaces,
		virtv1.Interface{
			Name: ifaceName,
		})
	return vmi
}

func newVMIWithOneIfaceStatus(vmi *virtv1.VirtualMachineInstance, ifaceName string) *virtv1.VirtualMachineInstance {
	vmi.Status.Interfaces = append(vmi.Status.Interfaces, *simpleIfaceStatus(ifaceName))
	return vmi
}

func simpleIfaceStatus(ifaceName string) *virtv1.VirtualMachineInstanceNetworkInterface {
	return &virtv1.VirtualMachineInstanceNetworkInterface{
		Name:          ifaceName,
		InterfaceName: ifaceName,
	}
}

func readyHotpluggedIfaceStatus(ifaceName string) *virtv1.VirtualMachineInstanceNetworkInterface {
	return &virtv1.VirtualMachineInstanceNetworkInterface{
		Name:       ifaceName,
		InfoSource: vmispec.InfoSourceMultusStatus,
	}
}

func newVMIWithGuestAgentInterface(vmi *virtv1.VirtualMachineInstance, ifaceName string) *virtv1.VirtualMachineInstance {
	vmi.Status.Interfaces = append(vmi.Status.Interfaces, virtv1.VirtualMachineInstanceNetworkInterface{
		InterfaceName: ifaceName,
		InfoSource:    vmispec.InfoSourceGuestAgent,
	})
	return vmi
}
