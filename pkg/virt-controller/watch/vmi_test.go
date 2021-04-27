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
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
	k8sv1 "k8s.io/api/core/v1"
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

	kvcontroller "kubevirt.io/kubevirt/pkg/controller"

	v1 "kubevirt.io/client-go/api/v1"
	fakenetworkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("VirtualMachineInstance watcher", func() {

	var ctrl *gomock.Controller
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var vmiSource *framework.FakeControllerSource
	var podSource *framework.FakeControllerSource
	var vmiInformer cache.SharedIndexInformer
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

	var dataVolumeSource *framework.FakeControllerSource
	var dataVolumeInformer cache.SharedIndexInformer
	var dataVolumeFeeder *testutils.DataVolumeFeeder
	var qemuGid int64 = 107
	controllerOf := true

	shouldExpectMatchingPodCreation := func(uid types.UID, matchers ...gomegaTypes.GomegaMatcher) {
		// Expect pod creation
		kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			update, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(update.GetObject().(*k8sv1.Pod).Labels[v1.CreatedByLabel]).To(Equal(string(uid)))
			Expect(update.GetObject().(*k8sv1.Pod)).To(SatisfyAny(matchers...))
			return true, update.GetObject(), nil
		})
	}

	shouldExpectPodCreation := func(uid types.UID) {
		// Expect pod creation
		kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			update, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(update.GetObject().(*k8sv1.Pod).Labels[v1.CreatedByLabel]).To(Equal(string(uid)))
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
			Expect(pod.Name).To(Equal(update.GetName()))
			return true, nil, nil
		})
	}

	shouldExpectVirtualMachineHandover := func(vmi *v1.VirtualMachineInstance) {
		vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
			Expect(arg.(*v1.VirtualMachineInstance).Status.Phase).To(Equal(v1.Scheduled))
			Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions).To(BeEmpty())
		}).Return(vmi, nil)
	}

	ignorePodUpdates := func() {
		kubeClient.Fake.PrependReactor("update", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
			update, _ := action.(testing.UpdateAction)
			return true, update.GetObject(), nil
		})
	}

	shouldExpectVirtualMachineSchedulingState := func(vmi *v1.VirtualMachineInstance) {
		vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
			Expect(arg.(*v1.VirtualMachineInstance).Status.Phase).To(Equal(v1.Scheduling))
			Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions).To(BeEmpty())
		}).Return(vmi, nil)
	}

	shouldExpectVirtualMachineScheduledState := func(vmi *v1.VirtualMachineInstance) {
		vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
			Expect(arg.(*v1.VirtualMachineInstance).Status.Phase).To(Equal(v1.Scheduled))
			Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions).To(BeEmpty())
		}).Return(vmi, nil)
	}

	shouldExpectVirtualMachineFailedState := func(vmi *v1.VirtualMachineInstance) {
		vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
			Expect(arg.(*v1.VirtualMachineInstance).Status.Phase).To(Equal(v1.Failed))
			Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions).To(BeEmpty())
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
		go podInformer.Run(stop)
		go pvcInformer.Run(stop)

		go dataVolumeInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop,
			vmiInformer.HasSynced,
			podInformer.HasSynced,
			pvcInformer.HasSynced,
			dataVolumeInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		vmiInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		podInformer, podSource = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		dataVolumeInformer, dataVolumeSource = testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		recorder = record.NewFakeRecorder(100)

		config, _, _, _ := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})
		pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		controller = NewVMIController(
			services.NewTemplateService("a", "b", "c", "d", "e", "f", "g", pvcInformer.GetStore(), virtClient, config, qemuGid, runtime.GOARCH),
			vmiInformer,
			podInformer,
			pvcInformer,
			recorder,
			virtClient,
			dataVolumeInformer,
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
		ctrl.Finish()
	})

	addVirtualMachine := func(vmi *v1.VirtualMachineInstance) {
		mockQueue.ExpectAdds(1)
		vmiSource.Add(vmi)
		mockQueue.Wait()
	}

	Context("On valid VirtualMachineInstance given with DataVolume source", func() {

		dvVolumeSource := v1.VolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name: "test1",
			},
		}
		It("should create a corresponding Pod on VMI creation when DataVolume is ready", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})

			dvPVC := NewPvc(vmi.Namespace, "test1")
			// we are mocking a successful DataVolume. we expect the PVC to
			// be available in the store if DV is successful.
			pvcInformer.GetIndexer().Add(dvPVC)

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.Succeeded)

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)
			shouldExpectPodCreation(vmi.UID)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})

		It("should create a doppleganger Pod on VMI creation when DataVolume is in WaitForFirstConsumer state", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)

			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, &controllerOf)
			dvPVC.Status.Phase = k8sv1.ClaimPending
			// we are mocking a DataVolume in WFFC phase. we expect the PVC to
			// be in available but in the Pending state.
			pvcInformer.GetIndexer().Add(dvPVC)

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)
			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions[0].Type).To(Equal(v1.VirtualMachineInstanceProvisioning))
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
			pod.Annotations[v1.EphemeralProvisioningObject] = "true"

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
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
			pvcInformer.GetIndexer().Add(dvPVC)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")
			dataVolumeFeeder.Add(dataVolume)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Phase).To(Equal(v1.Pending))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions).
					To(ContainElement(v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceProvisioning,
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
			pod.Annotations[v1.EphemeralProvisioningObject] = "true"

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
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
			pvcInformer.GetIndexer().Add(dvPVC)

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

		table.DescribeTable("VMI should handle doppleganger Pod status while DV is in WaitForFirstConsumer phase",
			func(phase k8sv1.PodPhase, conditions []k8sv1.PodCondition, expectedPhase v1.VirtualMachineInstancePhase) {
				vmi := NewPendingVirtualMachine("testvmi")
				pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
				pod.Annotations[v1.EphemeralProvisioningObject] = "true"
				pod.Status.Phase = phase
				pod.Status.Conditions = conditions

				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name:         "test1",
					VolumeSource: dvVolumeSource,
				})

				pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
					Name: "test1",
				})

				dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)
				dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, &controllerOf)
				dvPVC.Status.Phase = k8sv1.ClaimBound
				pvcInformer.GetIndexer().Add(dvPVC)

				addVirtualMachine(vmi)
				podFeeder.Add(pod)
				addActivePods(vmi, pod.UID, "")
				dataVolumeFeeder.Add(dataVolume)

				vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
					Expect(arg.(*v1.VirtualMachineInstance).Status.Phase).To(Equal(expectedPhase))
				}).Return(vmi, nil)
				kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
					return true, dvPVC, nil
				})

				controller.Execute()
			},
			table.Entry("fail if pod in failed state", k8sv1.PodFailed, nil, v1.Failed),
			table.Entry("fail if pod in succeded state", k8sv1.PodSucceeded, nil, v1.Failed),
			//The PodReasonUnschedulable is a transient condition. It can clear up if more resources are added to the cluster
			table.Entry("do nothing if pod Pending Unschedulable",
				k8sv1.PodPending,
				[]k8sv1.PodCondition{{
					Type:   k8sv1.PodScheduled,
					Status: k8sv1.ConditionFalse,
					Reason: k8sv1.PodReasonUnschedulable}}, v1.Pending),
		)

		It("should not create a corresponding Pod on VMI creation when DataVolume is pending", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name:         "test1",
				VolumeSource: dvVolumeSource,
			})

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.Pending)
			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", dataVolume.Name, &controllerOf)
			dvPVC.Status.Phase = k8sv1.ClaimPending
			pvcInformer.GetIndexer().Add(dvPVC)

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)

			controller.Execute()
		})
	})

	Context("On valid VirtualMachineInstance given with PVC source, ownedRef of DataVolume", func() {

		pvcVolumeSource := v1.VolumeSource{
			PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: "test1",
			},
		}
		It("should create a corresponding Pod on VMI creation when DataVolume is ready", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", "test1", &controllerOf)
			// we are mocking a successful DataVolume. we expect the PVC to
			// be available in the store if DV is successful.
			pvcInformer.GetIndexer().Add(dvPVC)

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.Succeeded)

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)
			shouldExpectPodCreation(vmi.UID)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})

		It("should create a doppleganger Pod on VMI creation when DataVolume is in WaitForFirstConsumer state", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", "test1", &controllerOf)
			dvPVC.Status.Phase = k8sv1.ClaimPending
			// we are mocking a DataVolume in WFFC phase. we expect the PVC to
			// be in available but in the Pending state.
			pvcInformer.GetIndexer().Add(dvPVC)

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)
			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions[0].Type).To(Equal(v1.VirtualMachineInstanceProvisioning))
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
			pod.Annotations[v1.EphemeralProvisioningObject] = "true"

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
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
			pvcInformer.GetIndexer().Add(dvPVC)

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.WaitForFirstConsumer)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")
			dataVolumeFeeder.Add(dataVolume)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Phase).To(Equal(v1.Pending))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions).
					To(ContainElement(v1.VirtualMachineInstanceCondition{
						Type:   v1.VirtualMachineInstanceProvisioning,
						Status: k8sv1.ConditionTrue,
					}))
			}).Return(vmi, nil)
			controller.Execute()
		})

		It("should delete a doppleganger Pod on VMI creation when DataVolume is no longer in WaitForFirstConsumer state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Annotations[v1.EphemeralProvisioningObject] = "true"

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
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
			pvcInformer.GetIndexer().Add(dvPVC)

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

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			dvPVC := NewPvcWithOwner(vmi.Namespace, "test1", "test1", &controllerOf)
			dvPVC.Status.Phase = k8sv1.ClaimPending
			// we are mocking a successful DataVolume. we expect the PVC to
			// be available in the store if DV is successful.
			pvcInformer.GetIndexer().Add(dvPVC)

			dataVolume := NewDv(vmi.Namespace, "test1", cdiv1.Pending)

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)

			controller.Execute()
		})

		It("should create a corresponding Pod on VMI creation when PVC is not controlled by a DataVolume", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name:         "test1",
				VolumeSource: pvcVolumeSource,
			})

			pvc := NewPvc(vmi.Namespace, "test1")
			pvcInformer.GetIndexer().Add(pvc)

			addVirtualMachine(vmi)
			shouldExpectPodCreation(vmi.UID)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
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
		table.DescribeTable("should delete the corresponding Pods on VirtualMachineInstance deletion with vmi", func(phase v1.VirtualMachineInstancePhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = phase
			vmi.DeletionTimestamp = now()

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
			table.Entry("in running state", v1.Running),
			table.Entry("in unset state", v1.VmPhaseUnset),
			table.Entry("in pending state", v1.Pending),
			table.Entry("in succeeded state", v1.Succeeded),
			table.Entry("in failed state", v1.Failed),
			table.Entry("in scheduled state", v1.Scheduled),
			table.Entry("in scheduling state", v1.Scheduling),
		)
		It("should not try to delete a pod again, which is already marked for deletion and go to failed state, when in scheduling state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
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
		table.DescribeTable("should not delete the corresponding Pod if the vmi is in", func(phase v1.VirtualMachineInstancePhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = phase
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			controller.Execute()
		},
			table.Entry("succeeded state", v1.Failed),
			table.Entry("failed state", v1.Succeeded),
		)
		It("should do nothing if the vmi is in final state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Failed
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

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions[0].Reason).To(Equal("FailedCreate"))
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

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions[0].Reason).To(Equal("FailedCreate"))
			}).Return(vmi, nil)

			controller.Execute()
			Expect(controller.Queue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(1))

			testutils.ExpectEvent(recorder, FailedCreatePodReason)
		})
		It("should remove the error condition if the sync finally succeeds", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{Type: v1.VirtualMachineInstanceSynchronized}}

			addVirtualMachine(vmi)

			// Expect pod creation
			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				update, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				Expect(update.GetObject().(*k8sv1.Pod).Labels[v1.CreatedByLabel]).To(Equal(string(vmi.UID)))
				return true, update.GetObject(), nil
			})

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions).To(BeEmpty())
			}).Return(vmi, nil)

			controller.Execute()
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})
		It("should create PodScheduled and Synchronized conditions exactly once each for repeated FailedPvcNotFoundReason sync error", func() {

			expectConditions := func(vmi *v1.VirtualMachineInstance) {
				// PodScheduled and Synchronized
				Expect(len(vmi.Status.Conditions)).To(Equal(2), "there should be exactly 2 conditions")

				getType := func(c v1.VirtualMachineInstanceCondition) string { return string(c.Type) }
				getReason := func(c v1.VirtualMachineInstanceCondition) string { return c.Reason }
				Expect(vmi.Status.Conditions).To(
					And(
						ContainElement(
							WithTransform(getType, Equal(string(v1.VirtualMachineInstanceSynchronized))),
						),
						ContainElement(
							And(
								WithTransform(getType, Equal(string(k8sv1.PodScheduled))),
								WithTransform(getReason, Equal(k8sv1.PodReasonUnschedulable)),
							),
						),
					),
				)

				testutils.ExpectEvent(recorder, FailedPvcNotFoundReason)
			}

			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "test",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "something",
						},
					},
				},
			}
			addVirtualMachine(vmi)
			update := vmiInterface.EXPECT().Update(gomock.Any())
			update.Do(func(vmi *v1.VirtualMachineInstance) {
				expectConditions(vmi)
				vmiInformer.GetStore().Update(vmi)
				update.Return(vmi, nil)
			})

			controller.Execute()
			Expect(controller.Queue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(1))

			// make sure that during next iteration we do not add the same condition again
			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance) {
				expectConditions(vmi)
			}).Return(vmi, nil)

			controller.Execute()
			Expect(controller.Queue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(2))
		})

		table.DescribeTable("should move the vmi to scheduling state if a pod exists", func(phase k8sv1.PodPhase, isReady bool) {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, phase)
			pod.Status.ContainerStatuses[0].Ready = isReady

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			shouldExpectVirtualMachineSchedulingState(vmi)

			controller.Execute()
		},
			table.Entry(", not ready and in running state", k8sv1.PodRunning, false),
			table.Entry(", not ready and in unknown state", k8sv1.PodUnknown, false),
			table.Entry(", not ready and in succeeded state", k8sv1.PodSucceeded, false),
			table.Entry(", not ready and in failed state", k8sv1.PodFailed, false),
			table.Entry(", not ready and in pending state", k8sv1.PodPending, false),
			table.Entry(", ready and in running state", k8sv1.PodRunning, true),
			table.Entry(", ready and in unknown state", k8sv1.PodUnknown, true),
			table.Entry(", ready and in succeeded state", k8sv1.PodSucceeded, true),
			table.Entry(", ready and in failed state", k8sv1.PodFailed, true),
			table.Entry(", ready and in pending state", k8sv1.PodPending, true),
		)

		Context("when pod failed to schedule", func() {
			It("should set scheduling pod condition on the VirtualMachineInstance", func() {
				vmi := NewPendingVirtualMachine("testvmi")
				vmi.Status.Phase = v1.Scheduling

				pod := NewPodForVirtualMachine(vmi, k8sv1.PodPending)

				pod.Status.Conditions = append(pod.Status.Conditions, k8sv1.PodCondition{
					Message: "Insufficient memory",
					Reason:  "Unschedulable",
					Status:  k8sv1.ConditionFalse,
					Type:    k8sv1.PodScheduled,
				})

				addVirtualMachine(vmi)
				podFeeder.Add(pod)

				vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
					Expect(arg.(*v1.VirtualMachineInstance).Status.Phase).To(Equal(v1.Scheduling))
					Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions).NotTo(BeEmpty())
					Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions[0].Message).To(Equal("Insufficient memory"))
					Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions[0].Reason).To(Equal("Unschedulable"))
					Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions[0].Status).To(Equal(k8sv1.ConditionFalse))
					Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions[0].Type).To(Equal(v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled)))
				}).Return(vmi, nil)

				controller.Execute()
			})
		})

		Context("when Pod recovers from scheduling issues", func() {
			table.DescribeTable("it should remove scheduling pod condition from the VirtualMachineInstance if the pod", func(owner string, podPhase k8sv1.PodPhase) {
				vmi := NewPendingVirtualMachine("testvmi")
				vmi.Status.Phase = v1.Scheduling

				pod := NewPodForVirtualMachine(vmi, podPhase)

				vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
					Message: "Insufficient memory",
					Reason:  "Unschedulable",
					Status:  k8sv1.ConditionFalse,
					Type:    v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled),
				})

				addVirtualMachine(vmi)
				podFeeder.Add(pod)

				vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
					Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions).To(BeEmpty())
				}).Return(vmi, nil)

				ignorePodUpdates()

				controller.Execute()

				testutils.IgnoreEvents(recorder)
			},
				table.Entry("is owned by virt-handler and is running", "virt-handler", k8sv1.PodRunning),
				table.Entry("is owned by virt-controller and is running", "virt-controller", k8sv1.PodRunning),
				table.Entry("is owned by virt-controller and is pending", "virt-controller", k8sv1.PodPending),
			)
		})

		It("should move the vmi to failed state if the pod disappears and the vmi is in scheduling state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling

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
		table.DescribeTable("should hand over pod to virt-handler if pod is ready and running", func(containerStatus []k8sv1.ContainerStatus) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
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
			table.Entry("with running compute container and no infra container",
				[]k8sv1.ContainerStatus{{
					Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
				}},
			),
			table.Entry("with running compute container and no ready istio-proxy container",
				[]k8sv1.ContainerStatus{{
					Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
				}, {Name: "istio-proxy", Ready: false}},
			),
		)
		table.DescribeTable("should not hand over pod to virt-handler if pod is ready and running", func(containerStatus []k8sv1.ContainerStatus) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.ContainerStatuses = containerStatus

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			controller.Execute()
		},
			table.Entry("with not ready infra container and not ready compute container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: false}, {Name: "kubevirt-infra", Ready: false}},
			),
			table.Entry("with not ready compute container and no infra container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: false}},
			),
		)

		It("should ignore migration target pods", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Running
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
			Expect(err).To(BeNil())

			Expect(selectedPod.UID).To(Equal(pod.UID))
		})

		It("should set an error condition if deleting the virtual machine pod fails", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.DeletionTimestamp = now()
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			// Expect pod delete
			kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions[0].Reason).To(Equal(FailedDeletePodReason))
			}).Return(vmi, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedDeletePodReason)
		})
		It("should set an error condition if when pod cannot guarantee resources when cpu pinning has been requested", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
			vmi.Spec.Domain.CPU = &v1.CPU{DedicatedCPUPlacement: true}
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

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions[0].Reason).To(Equal(FailedGuaranteePodResourcesReason))
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

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions[0].Reason).To(Equal(FailedCreatePodReason))
			}).Return(vmi, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedCreatePodReason)
		})
		It("should update the virtual machine to scheduled if pod is ready, runnning and handed over to virt-handler", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			shouldExpectVirtualMachineHandover(vmi)

			controller.Execute()
		})
		It("should update the virtual machine QOS class if the pod finally has a QOS class assigned", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.QOSClass = k8sv1.PodQOSGuaranteed

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(*arg.(*v1.VirtualMachineInstance).Status.QOSClass).To(Equal(k8sv1.PodQOSGuaranteed))
			}).Return(vmi, nil)

			controller.Execute()
		})
		It("should update the virtual machine to scheduled if pod is ready, triggered by pod change", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
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
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodPending)
			vmi.Status.Phase = v1.Scheduling

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
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodPending)
			pod.Status.ContainerStatuses = []k8sv1.ContainerStatus{{Name: "compute", State: k8sv1.ContainerState{Terminated: &k8sv1.ContainerStateTerminated{}}}}
			vmi.Status.Phase = v1.Scheduling

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			shouldExpectVirtualMachineFailedState(vmi)

			controller.Execute()
		})
		table.DescribeTable("should remove the finalizer if no pod is present and the vmi is in ", func(phase v1.VirtualMachineInstancePhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = phase
			vmi.Status.LauncherContainerImageVersion = "madeup"
			vmi.Labels = map[string]string{}
			vmi.Labels[v1.OutdatedLauncherImageLabel] = ""
			vmi.Finalizers = append(vmi.Finalizers, v1.VirtualMachineInstanceFinalizer)
			Expect(vmi.Finalizers).To(ContainElement(v1.VirtualMachineInstanceFinalizer))

			addVirtualMachine(vmi)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				// outdated launcher label should get cleared after being finalized
				_, ok := arg.(*v1.VirtualMachineInstance).Labels[v1.OutdatedLauncherImageLabel]
				Expect(ok).To(BeFalse())
				// outdated launcher image should get cleared after being finalized
				Expect(arg.(*v1.VirtualMachineInstance).Status.LauncherContainerImageVersion).To(Equal(""))

				Expect(arg.(*v1.VirtualMachineInstance).Status.Phase).To(Equal(phase))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions).To(BeEmpty())
				Expect(arg.(*v1.VirtualMachineInstance).Finalizers).ToNot(ContainElement(v1.VirtualMachineInstanceFinalizer))
			}).Return(vmi, nil)

			controller.Execute()
		},
			table.Entry("failed state", v1.Succeeded),
			table.Entry("succeeded state", v1.Failed),
		)
		table.DescribeTable("should do nothing if pod is handed to virt-handler", func(phase k8sv1.PodPhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduled
			pod := NewPodForVirtualMachine(vmi, phase)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			controller.Execute()
		},
			table.Entry("and in running state", k8sv1.PodRunning),
			table.Entry("and in unknown state", k8sv1.PodUnknown),
			table.Entry("and in succeeded state", k8sv1.PodSucceeded),
			table.Entry("and in failed state", k8sv1.PodFailed),
			table.Entry("and in pending state", k8sv1.PodPending),
		)

		It("should remove ready condition when VMI is in a finalized state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Succeeded

			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{Type: v1.VirtualMachineInstanceReady}}
			addVirtualMachine(vmi)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Conditions)).To(Equal(0))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should add outdated label if pod's image is outdated and VMI is in running state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Running
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = []k8sv1.PodCondition{{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue}}

			pod.Spec.Containers = append(pod.Spec.Containers, k8sv1.Container{
				Image: "madeup",
				Name:  "compute",
			})

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			podFeeder.Add(pod)

			patch := `[ { "op": "test", "path": "/status/conditions", "value": null }, { "op": "replace", "path": "/status/conditions", "value": [{"type":"Ready","status":"True","lastProbeTime":null,"lastTransitionTime":null}] }, { "op": "add", "path": "/status/launcherContainerImageVersion", "value": "madeup" }, { "op": "add", "path": "/metadata/labels", "value": {"kubevirt.io/outdatedLauncherImage":""} } ]`

			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, []byte(patch)).Return(vmi, nil)

			controller.Execute()
		})
		It("should remove outdated label if pod's image up-to-date and VMI is in running state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Running
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = []k8sv1.PodCondition{{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue}}

			if vmi.Labels == nil {
				vmi.Labels = make(map[string]string)
			}

			vmi.Labels[v1.OutdatedLauncherImageLabel] = ""

			pod.Spec.Containers = append(pod.Spec.Containers, k8sv1.Container{
				Image: controller.templateService.GetLauncherImage(),
				Name:  "compute",
			})

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			podFeeder.Add(pod)

			patch := `[ { "op": "test", "path": "/status/conditions", "value": null }, { "op": "replace", "path": "/status/conditions", "value": [{"type":"Ready","status":"True","lastProbeTime":null,"lastTransitionTime":null}] }, { "op": "add", "path": "/status/launcherContainerImageVersion", "value": "a" }, { "op": "test", "path": "/metadata/labels", "value": {"kubevirt.io/outdatedLauncherImage":""} }, { "op": "replace", "path": "/metadata/labels", "value": {} } ]`

			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, []byte(patch)).Return(vmi, nil)

			controller.Execute()
		})

		It("should add a ready condition if it is present on the pod and the VMI is in running state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Running
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = []k8sv1.PodCondition{{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue}}

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			podFeeder.Add(pod)

			patch := `[ { "op": "test", "path": "/status/conditions", "value": null }, { "op": "replace", "path": "/status/conditions", "value": [{"type":"Ready","status":"True","lastProbeTime":null,"lastTransitionTime":null}] } ]`
			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, []byte(patch)).Return(vmi, nil)

			controller.Execute()
		})

		It("should indicate on the ready condition if the pod is terminating", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Running
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.DeletionTimestamp = now()
			pod.Status.Conditions = []k8sv1.PodCondition{{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue}}

			addVirtualMachine(vmi)
			addActivePods(vmi, pod.UID, "")
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, gomock.Any()).DoAndReturn(func(_ string, _ interface{}, patchBytes []byte) (*v1.VirtualMachineInstance, error) {
				patch, err := jsonpatch.DecodePatch(patchBytes)
				Expect(err).ToNot(HaveOccurred())
				vmiBytes, err := json.Marshal(vmi)
				Expect(err).ToNot(HaveOccurred())
				vmiBytes, err = patch.Apply(vmiBytes)
				Expect(err).ToNot(HaveOccurred())
				patchedVMI := &v1.VirtualMachineInstance{}
				err = json.Unmarshal(vmiBytes, patchedVMI)
				Expect(err).ToNot(HaveOccurred())
				Expect(patchedVMI.Status.Conditions).To(HaveLen(1))
				cond := patchedVMI.Status.Conditions[0]
				Expect(cond.Reason).To(Equal(v1.PodTerminatingReason))
				Expect(string(cond.Type)).To(Equal(string(k8sv1.PodReady)))
				Expect(cond.Status).To(Equal(k8sv1.ConditionFalse))
				return patchedVMI, nil
			})
			controller.Execute()
			testutils.ExpectEvent(recorder, v1.PodTerminatingReason)
		})

		It("should add active pods to status if VMI is in running state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Running
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.UID = "someUID"
			pod.Spec.NodeName = "someHost"

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			patch := `[ { "op": "test", "path": "/status/activePods", "value": {} }, { "op": "replace", "path": "/status/activePods", "value": {"someUID":"someHost"} } ]`
			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, []byte(patch)).Return(vmi, nil)

			controller.Execute()
		})

		table.DescribeTable("should not add a ready condition if the vmi is", func(phase v1.VirtualMachineInstancePhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = phase
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = []k8sv1.PodCondition{{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue}}

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			controller.Execute()
		},
			table.Entry("in succeeded state", v1.Succeeded),
			table.Entry("in failed state", v1.Failed),
			table.Entry("in scheduled state", v1.Scheduled),
		)

		It("should not remove sync conditions from virt-handler if it is in scheduled state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduled
			vmi.Status.Conditions = append(vmi.Status.Conditions,
				v1.VirtualMachineInstanceCondition{Type: v1.VirtualMachineInstanceSynchronized, Status: k8sv1.ConditionFalse})
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = []k8sv1.PodCondition{{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue}}

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			addActivePods(vmi, pod.UID, "")

			controller.Execute()
		})

		table.DescribeTable("should do nothing if the vmi is handed over to virt-handler, the pod disappears", func(phase v1.VirtualMachineInstancePhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = phase

			addVirtualMachine(vmi)

			controller.Execute()
		},
			table.Entry("and the vmi is in running state", v1.Running),
			table.Entry("and the vmi is in scheduled state", v1.Scheduled),
		)

		table.DescribeTable("should move the vmi to failed if pod is not handed over", func(phase k8sv1.PodPhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
			pod := NewPodForVirtualMachine(vmi, phase)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			shouldExpectVirtualMachineFailedState(vmi)

			controller.Execute()
		},
			table.Entry("and in succeeded state", k8sv1.PodSucceeded),
			table.Entry("and in failed state", k8sv1.PodFailed),
		)
	})

	Context("hotplug volume", func() {
		It("Should find vmi, from virt-launcher pod", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			controllerRef := kvcontroller.GetControllerOf(pod)
			addVirtualMachine(vmi)

			result := controller.resolveControllerRef(k8sv1.NamespaceDefault, controllerRef)
			Expect(reflect.DeepEqual(result, vmi)).To(BeTrue(), "result: %v, should equal %v", result, vmi)
		})

		It("Should find vmi, from attachment pod", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			attachmentPod := NewPodForVirtlauncher(pod, "hp-test", "abcd", k8sv1.PodRunning)
			controllerRef := kvcontroller.GetControllerOf(attachmentPod)
			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			result := controller.resolveControllerRef(k8sv1.NamespaceDefault, controllerRef)
			Expect(reflect.DeepEqual(result, vmi)).To(BeTrue(), "result: %v, should equal %v", result, vmi)
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

		It("DeleteAllAttachmentPods should return success if there are no attachment pods", func() {
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
			err := controller.deleteAllAttachmentPods(vmi)
			Expect(err).ToNot(HaveOccurred())
			shouldExpectPodDeletion(attachmentPod1)
			shouldExpectPodDeletion(attachmentPod2)
			shouldExpectPodDeletion(attachmentPod3)
		})

		It("CreateAttachmentPodTemplate should return error if volume is not DV or PVC", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
			invalidVolume := &v1.Volume{
				Name: "fake",
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{},
				},
			}
			pod, err := controller.createAttachmentPodTemplate(vmi, virtlauncherPod, invalidVolume)
			Expect(pod).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to hotplug, claim not PVC or Datavolume"))
		})

		It("CreateAttachmentPodTemplate should return error if volume has PVC that doesn't exist", func() {
			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, k8serrors.NewNotFound(k8sv1.Resource("persistentvolumeclaim"), "noclaim")
			})
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
			nopvcVolume := &v1.Volume{
				Name: "nopvc",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "noclaim",
					},
				},
			}
			pod, err := controller.createAttachmentPodTemplate(vmi, virtlauncherPod, nopvcVolume)
			Expect(pod).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to hotplug, claim noclaim not found"))
		})

		It("CreateAttachmentPodTemplate should return error if DV owning PVC doesn't exist", func() {
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
			volume := &v1.Volume{
				Name: "test-pvc-volume",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "test-dv",
					},
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "test-dv",
					},
				},
			}
			pod, err := controller.createAttachmentPodTemplate(vmi, virtlauncherPod, volume)
			Expect(pod).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to hotplug, claim test-dv not found"))
		})

		It("CreateAttachmentPodTemplate should set status to pending if DV owning PVC is not ready", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pvc := NewHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimPending)
			pvcInformer.GetIndexer().Add(pvc)
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dv",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Pending,
				},
			}
			dataVolumeInformer.GetIndexer().Add(dv)
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
			volume := &v1.Volume{
				Name: "test-pvc-volume",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "test-dv",
					},
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "test-dv",
					},
				},
			}
			pod, err := controller.createAttachmentPodTemplate(vmi, virtlauncherPod, volume)
			Expect(pod).To(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("CreateAttachmentPodTemplate should update status to bound if DV owning PVC is bound but not ready", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, v1.VolumeStatus{
				Name:          "test-pvc-volume",
				HotplugVolume: &v1.HotplugVolumeStatus{},
				Phase:         v1.VolumePending,
				Message:       "some technical reason",
				Reason:        PVCNotReadyReason,
			})
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pvc := NewHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimBound)
			pvcInformer.GetIndexer().Add(pvc)
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dv",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Pending,
				},
			}
			dataVolumeInformer.GetIndexer().Add(dv)
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
			volume := &v1.Volume{
				Name: "test-pvc-volume",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "test-dv",
					},
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "test-dv",
					},
				},
			}
			pod, err := controller.createAttachmentPodTemplate(vmi, virtlauncherPod, volume)
			Expect(pod).To(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("CreateAttachmentPodTemplate should create a pod template if DV of owning PVC is ready", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pvc := NewHotplugPVC("test-dv", vmi.Namespace, k8sv1.ClaimBound)
			pvcInformer.GetIndexer().Add(pvc)
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-dv",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Succeeded,
				},
			}
			dataVolumeInformer.GetIndexer().Add(dv)
			addVirtualMachine(vmi)
			podFeeder.Add(virtlauncherPod)
			volume := &v1.Volume{
				Name: "test-pvc-volume",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "test-dv",
					},
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "test-dv",
					},
				},
			}
			pod, err := controller.createAttachmentPodTemplate(vmi, virtlauncherPod, volume)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.GenerateName).To(Equal("hp-volume-"))
			Expect(pod.Spec.Volumes[0].Name).To(Equal(volume.Name))
		})

		makePodsWithVirtlauncher := func(virtlauncherPod *k8sv1.Pod, indexes ...int) []*k8sv1.Pod {
			res := make([]*k8sv1.Pod, 0)
			for _, index := range indexes {
				pod := NewPodForVirtlauncher(virtlauncherPod, fmt.Sprintf("test-pod%d", index), fmt.Sprintf("abcd%d", index), k8sv1.PodRunning)
				pod.Spec.Volumes = make([]k8sv1.Volume, 0)
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
			return makePodsWithVirtlauncher(virtlauncherPod, indexes...)
		}

		makeVolumes := func(indexes ...int) []*v1.Volume {
			res := make([]*v1.Volume, 0)
			for _, index := range indexes {
				res = append(res, &v1.Volume{
					Name: fmt.Sprintf("volume%d", index),
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("claim%d", index),
						},
					},
				})
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

		table.DescribeTable("should calculate deleted volumes based on current pods and volumes", func(hotplugPods []*k8sv1.Pod, hotplugVolumes []*v1.Volume, expected []k8sv1.Volume) {
			res := controller.getDeletedHotplugVolumes(hotplugPods, hotplugVolumes)
			Expect(len(res)).To(Equal(len(expected)))
			for i, volume := range res {
				Expect(reflect.DeepEqual(volume, expected[i])).To(BeTrue(), "%v does not match %v", volume, expected[i])
			}
		},
			table.Entry("should return empty if pods and volumes is empty", make([]*k8sv1.Pod, 0), make([]*v1.Volume, 0), make([]k8sv1.Volume, 0)),
			table.Entry("should return empty if pods and volumes match", makePods(1), makeVolumes(1), make([]k8sv1.Volume, 0)),
			table.Entry("should return empty if pods < volumes", makePods(1), makeVolumes(1, 2), make([]k8sv1.Volume, 0)),
			table.Entry("should return 1 value if pods > volumes by 1", makePods(1, 2), makeVolumes(1), makeK8sVolumes(2)),
		)

		table.DescribeTable("should calculate new volumes based on current pods and volumes", func(hotplugPods []*k8sv1.Pod, hotplugVolumes []*v1.Volume, expected []*v1.Volume) {
			res := controller.getNewHotplugVolumes(hotplugPods, hotplugVolumes)
			Expect(len(res)).To(Equal(len(expected)))
			for i, volume := range res {
				Expect(reflect.DeepEqual(volume, expected[i])).To(BeTrue(), "%v does not match %v", volume, expected[i])
			}
		},
			table.Entry("should return empty if pods and volumes is empty", make([]*k8sv1.Pod, 0), make([]*v1.Volume, 0), make([]*v1.Volume, 0)),
			table.Entry("should return empty if pods and volumes match", makePods(1), makeVolumes(1), make([]*v1.Volume, 0)),
			table.Entry("should return empty if pods > volumes", makePods(1), makeVolumes(1, 2), makeVolumes(2)),
			table.Entry("should return 1 value if pods < volumes by 1", makePods(1, 2), makeVolumes(1), make([]*v1.Volume, 0)),
		)

		makeVolumeStatuses := func() []v1.VolumeStatus {
			volumeStatuses := make([]v1.VolumeStatus, 0)
			volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
				Name: "test-volume",
				HotplugVolume: &v1.HotplugVolumeStatus{
					AttachPodName: "attach-pod",
					AttachPodUID:  "abcd",
				},
			})
			volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
				Name:          "test-volume2",
				HotplugVolume: &v1.HotplugVolumeStatus{},
			})
			return volumeStatuses
		}

		table.DescribeTable("should determine if status contains the volume including a pod", func(volumeStatuses []v1.VolumeStatus, volume *v1.Volume, expected bool) {
			res := controller.volumeStatusContainsVolumeAndPod(volumeStatuses, volume)
			Expect(res).To(Equal(expected))
		},
			table.Entry("should return if the volume with pod is in the status", makeVolumeStatuses(), &v1.Volume{
				Name: "test-volume",
			}, true),
			table.Entry("should return if the volume with pod is in the status", makeVolumeStatuses(), &v1.Volume{
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
				dataVolumeInformer.GetIndexer().Add(dv)
				pvcInformer.GetIndexer().Add(pvc)
			}
		}

		makeExistingVolumeStatuses := func(indexes ...int) []v1.VolumeStatus {
			res := makeVolumeStatuses()
			for _, index := range indexes {
				res = append(res, v1.VolumeStatus{
					Name: fmt.Sprintf("volume%d", index),
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testing-pod",
						AttachPodUID:  "abcd",
					},
					Phase:   v1.HotplugVolumeAttachedToNode,
					Message: fmt.Sprintf("Created hotplug attachment pod , for volume volume%d", index),
					Reason:  SuccessfulCreatePodReason,
				})
			}
			return res
		}

		makePositiveCreatePod := func(virtlauncherPod *k8sv1.Pod, indexes ...int) {
			for i := 0; i < len(indexes); i++ {
				kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
					update, ok := action.(testing.CreateAction)
					Expect(ok).To(BeTrue())
					Expect(update.GetObject().(*k8sv1.Pod).GenerateName).To(Equal("hp-volume-"))
					Expect(update.GetObject().(*k8sv1.Pod).ObjectMeta.OwnerReferences[0].Name).To(Equal(virtlauncherPod.Name))
					Expect(update.GetObject().(*k8sv1.Pod).ObjectMeta.OwnerReferences[0].UID).To(Equal(virtlauncherPod.UID))
					return true, update.GetObject(), nil
				})
			}
		}

		makeExistingCreatePod := func(virtlauncherPod *k8sv1.Pod, indexes ...int) {
			for i := 0; i < len(indexes); i++ {
				kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
					return true, nil, k8serrors.NewAlreadyExists(k8sv1.Resource("pod"), "hp-volume")
				})
			}
		}

		makeFailureCreatePod := func(virtlauncherPod *k8sv1.Pod, indexes ...int) {
			for i := 0; i < len(indexes); i++ {
				kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
					return true, nil, fmt.Errorf("Error creating pod")
				})
			}
		}

		makePositiveDeletePod := func(virtlauncherPod *k8sv1.Pod, indexes ...int) {
			for i := 0; i < len(indexes); i++ {
				kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
					_, ok := action.(testing.DeleteAction)
					Expect(ok).To(BeTrue())
					return true, nil, nil
				})
			}
		}

		makeFailureDeletePod := func(virtlauncherPod *k8sv1.Pod, indexes ...int) {
			for i := 0; i < len(indexes); i++ {
				kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
					_, ok := action.(testing.DeleteAction)
					Expect(ok).To(BeTrue())
					return true, nil, fmt.Errorf("Error deleting pod")
				})
			}
		}

		table.DescribeTable("handleHotplugVolumes should properly react to input", func(hotplugVolumes []*v1.Volume, hotplugAttachmentPods []*k8sv1.Pod, createPodReaction func(*k8sv1.Pod, ...int), pvcFunc func(...int), pvcIndexes []int, orgStatus []v1.VolumeStatus, expectedEvent string, expectedErr syncError) {
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
			table.Entry("when volumes and pods match, the status should remain the same",
				makeVolumes(1),
				makePods(1),
				nil,
				preparePVC,
				[]int{},
				makeVolumeStatuses(),
				"",
				nil),
			table.Entry("when volumes > pods, a new pod should be created with a status",
				makeVolumes(1, 2),
				makePods(1),
				makePositiveCreatePod,
				preparePVC,
				[]int{2},
				makeVolumeStatuses(),
				SuccessfulCreatePodReason,
				nil),
			table.Entry("when volumes > pods, but an existing status with a pod exist, should error",
				makeVolumes(1, 2),
				makePods(1),
				makePositiveCreatePod,
				preparePVC,
				[]int{2},
				makeExistingVolumeStatuses(2),
				"",
				&syncErrorImpl{fmt.Errorf("Missing pod for hotplugged volume %s", "volume2"), MissingAttachmentPodReason}),
			table.Entry("when volumes > pods, and creating pod fails, should return the pod creation failure error",
				makeVolumes(1, 2),
				makePods(1),
				makeFailureCreatePod,
				preparePVC,
				[]int{2},
				makeVolumeStatuses(),
				FailedCreatePodReason,
				&syncErrorImpl{fmt.Errorf("Error creating attachment pod %v", fmt.Errorf("Error creating pod")), FailedCreatePodReason}),
			table.Entry("when volumes > pods, should return error if pod already exists.",
				makeVolumes(1, 2),
				makePods(1),
				makeExistingCreatePod,
				preparePVC,
				[]int{2},
				makeVolumeStatuses(),
				FailedCreatePodReason,
				&syncErrorImpl{fmt.Errorf("Error creating attachment pod %v", fmt.Errorf("pod \"hp-volume\" already exists")), FailedCreatePodReason}),
			table.Entry("when volumes < pods, the pod should be deleted",
				makeVolumes(1),
				makePods(1, 2),
				makePositiveDeletePod,
				preparePVC,
				[]int{2},
				makeVolumeStatuses(),
				SuccessfulDeletePodReason,
				nil),
			table.Entry("when volumes < pods, if pod deletion fails, it should return the error",
				makeVolumes(1),
				makePods(1, 2),
				makeFailureDeletePod,
				preparePVC,
				[]int{2},
				makeVolumeStatuses(),
				FailedDeletePodReason,
				&syncErrorImpl{fmt.Errorf("Error deleting attachment pod %v", fmt.Errorf("Error deleting pod")), FailedDeletePodReason}),
		)

		table.DescribeTable("needsHandleHotplug", func(hotplugVolumes []*v1.Volume, hotplugAttachmentPods []*k8sv1.Pod, expected bool) {
			res := controller.needsHandleHotplug(hotplugVolumes, hotplugAttachmentPods)
			Expect(res).To(Equal(expected))
		},
			table.Entry("should return false if volumes and attachmentpods are empty", makeVolumes(), makePods(), false),
			table.Entry("should return false if volumes and attachmentpods match", makeVolumes(1), makePods(1), false),
			table.Entry("should return true if volumes > attachmentpods", makeVolumes(1, 2), makePods(1), true),
			table.Entry("should return true if volumes < attachmentpods", makeVolumes(1), makePods(1, 2), true),
			table.Entry("should return true if len(volumes) == len(attachmentpods), but contents differ", makeVolumes(1, 3), makePods(1, 2), true),
		)

		table.DescribeTable("virtlauncherAttachmentPods", func(podCount int) {
			vmi := NewPendingVirtualMachine("testvmi")
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			for i := 0; i < podCount; i++ {
				attachmentPod := NewPodForVirtlauncher(virtlauncherPod, fmt.Sprintf("test-pod%d", i), fmt.Sprintf("abcd%d", i), k8sv1.PodRunning)
				podInformer.GetIndexer().Add(attachmentPod)
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
				podInformer.GetIndexer().Add(pod)
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
			podInformer.GetIndexer().Add(otherPod)
			for i := 10; i < 10+podCount; i++ {
				attachmentPod := NewPodForVirtlauncher(otherPod, fmt.Sprintf("test-pod%d", i), fmt.Sprintf("abcde%d", i), k8sv1.PodRunning)
				podInformer.GetIndexer().Add(attachmentPod)
			}

			res, err := controller.virtlauncherAttachmentPods(virtlauncherPod)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(res)).To(Equal(podCount))
		},
			table.Entry("should return number of pods passed in", 0),
			table.Entry("should return number of pods passed in", 1),
			table.Entry("should return number of pods passed in", 2),
		)

		table.DescribeTable("getHotplugVolumes", func(virtlauncherVolumes []k8sv1.Volume, vmiVolumes []*v1.Volume, expectedIndexes ...int) {
			vmi := NewPendingVirtualMachine("testvmi")
			for _, volume := range vmiVolumes {
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, *volume)
			}
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			virtlauncherPod.Spec.Volumes = virtlauncherVolumes
			res := controller.getHotplugVolumes(vmi, virtlauncherPod)
			Expect(len(res)).To(Equal(len(expectedIndexes)))
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
			table.Entry("should return no volumes if vmi and virtlauncher have no volumes", makeK8sVolumes(), makeVolumes()),
			table.Entry("should return a volume if vmi has one more than virtlauncher", makeK8sVolumes(), makeVolumes(1), 1),
			table.Entry("should return a volume if vmi has one more than virtlauncher, with matching volumes", makeK8sVolumes(1, 3), makeVolumes(1, 2, 3), 2),
			table.Entry("should return multiple volumes if vmi has multiple more than virtlauncher, with matching volumes", makeK8sVolumes(1, 3), makeVolumes(1, 2, 3, 4, 5), 2, 4, 5),
		)

		truncateSprintf := func(str string, args ...interface{}) string {
			n := strings.Count(str, "%d")
			return fmt.Sprintf(str, args[:n]...)
		}

		makeVolumeStatusesForUpdateWithMessage := func(podName, podUID string, phase v1.VolumePhase, message, reason string, indexes ...int) []v1.VolumeStatus {
			res := make([]v1.VolumeStatus, 0)
			for _, index := range indexes {
				res = append(res, v1.VolumeStatus{
					Name: fmt.Sprintf("volume%d", index),
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: truncateSprintf(podName, index),
						AttachPodUID:  types.UID(truncateSprintf(podUID, index)),
					},
					Phase:   phase,
					Message: truncateSprintf(message, index, index),
					Reason:  reason,
				})
			}
			return res
		}

		makeVolumeStatusesForUpdate := func(indexes ...int) []v1.VolumeStatus {
			return makeVolumeStatusesForUpdateWithMessage("test-pod%d", "abcd%d", v1.HotplugVolumeAttachedToNode, "Created hotplug attachment pod test-pod%d, for volume volume%d", SuccessfulCreatePodReason, indexes...)
		}

		table.DescribeTable("updateVolumeStatus", func(oldStatus []v1.VolumeStatus, specVolumes []*v1.Volume, podIndexes []int, pvcIndexes []int, expectedStatus []v1.VolumeStatus, expectedEvents []string) {
			vmi := NewPendingVirtualMachine("testvmi")
			volumes := make([]v1.Volume, 0)
			for _, volume := range specVolumes {
				volumes = append(volumes, *volume)
			}
			vmi.Spec.Volumes = volumes
			vmi.Status.VolumeStatus = oldStatus
			virtlauncherPod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			attachmentPods := makePodsWithVirtlauncher(virtlauncherPod, podIndexes...)
			now := metav1.Now()
			for _, pod := range attachmentPods {
				pod.DeletionTimestamp = &now
				podInformer.GetIndexer().Add(pod)
			}
			for _, pvcIndex := range pvcIndexes {
				pvc := NewHotplugPVC(fmt.Sprintf("claim%d", pvcIndex), k8sv1.NamespaceDefault, k8sv1.ClaimBound)
				pvcInformer.GetIndexer().Add(pvc)
			}
			err := controller.updateVolumeStatus(vmi, virtlauncherPod)
			testutils.ExpectEvents(recorder, expectedEvents...)
			Expect(err).ToNot(HaveOccurred())
			Expect(reflect.DeepEqual(expectedStatus, vmi.Status.VolumeStatus)).To(BeTrue())
		},
			table.Entry("should not update volume status, if no volumes changed",
				makeVolumeStatusesForUpdate(),
				makeVolumes(),
				[]int{},
				[]int{},
				makeVolumeStatusesForUpdate(),
				[]string{}),
			table.Entry("should update volume status, if a new volume is added, and pod exists",
				makeVolumeStatusesForUpdate(),
				makeVolumes(0),
				[]int{0},
				[]int{},
				makeVolumeStatusesForUpdate(0),
				[]string{SuccessfulCreatePodReason}),
			table.Entry("should update volume status, if a new volume is added, and pod does not exist",
				makeVolumeStatusesForUpdate(),
				makeVolumes(0),
				[]int{},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("", "", v1.VolumeBound, "PVC is in phase Bound", PVCNotReadyReason, 0),
				[]string{}),
			table.Entry("should update volume status, if a existing volume is changed, and pod does not exist",
				makeVolumeStatusesForUpdateWithMessage("", "", v1.VolumePending, "PVC is in phase Pending", PVCNotReadyReason, 0),
				makeVolumes(0),
				[]int{},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("", "", v1.VolumeBound, "PVC is in phase Bound", PVCNotReadyReason, 0),
				[]string{}),
			table.Entry("should keep status, if volume removed and if pod still exists",
				makeVolumeStatusesForUpdate(0),
				makeVolumes(),
				[]int{0},
				[]int{0},
				makeVolumeStatusesForUpdateWithMessage("test-pod0", "abcd0", v1.HotplugVolumeDetaching, "Deleted hotplug attachment pod test-pod0, for volume volume0", SuccessfulDeletePodReason, 0),
				[]string{SuccessfulDeletePodReason}),
			table.Entry("should remove volume status, if volume is removed and pod is gone",
				makeVolumeStatusesForUpdate(0),
				makeVolumes(),
				[]int{},
				[]int{},
				makeVolumeStatusesForUpdate(),
				[]string{}),
		)

		It("Should properly create attachmentpod, if correct volume and disk are added", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			volumes := make([]v1.Volume, 0)
			volumes = append(volumes, v1.Volume{
				Name: "existing",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "existing",
					},
				},
			})
			volumes = append(volumes, v1.Volume{
				Name: "hotplug",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "hotplug",
					},
				},
			})
			vmi.Spec.Volumes = volumes
			disks := make([]v1.Disk, 0)
			disks = append(disks, v1.Disk{
				Name: "existing",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: "virtio",
					},
				},
			})
			disks = append(disks, v1.Disk{
				Name: "hotplug",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: "scsi",
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = disks
			vmi.Status.Phase = v1.Running
			vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, v1.VolumeStatus{
				Name:   "existing",
				Target: "",
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
			pvcInformer.GetIndexer().Add(existingPVC)
			pvcInformer.GetIndexer().Add(hpPVC)
			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, existingPVC, nil
			})
			shouldExpectHotplugPod()
			addVirtualMachine(vmi)
			podInformer.GetIndexer().Add(virtlauncherPod)
			//Modify by adding a new hotplugged disk
			patch := `[ { "op": "test", "path": "/status/volumeStatus", "value": [{"name":"existing","target":""}] }, { "op": "replace", "path": "/status/volumeStatus", "value": [{"name":"existing","target":""},{"name":"hotplug","target":"","phase":"Bound","reason":"PVCNotReady","message":"PVC is in phase Bound","hotplugVolume":{}}] } ]`
			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, []byte(patch)).Return(vmi, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
			Expect(vmi.Status.Phase).To(Equal(v1.Running))
		})

		It("Should properly delete attachment, if volume and disk are removed", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			volumes := make([]v1.Volume, 0)
			volumes = append(volumes, v1.Volume{
				Name: "existing",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "existing",
					},
				},
			})
			vmi.Spec.Volumes = volumes
			disks := make([]v1.Disk, 0)
			disks = append(disks, v1.Disk{
				Name: "existing",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: "virtio",
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = disks
			vmi.Status.Phase = v1.Running
			vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, v1.VolumeStatus{
				Name:   "existing",
				Target: "",
			})
			vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, v1.VolumeStatus{
				Name:   "hotplug",
				Target: "",
				HotplugVolume: &v1.HotplugVolumeStatus{
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
			podInformer.GetIndexer().Add(hpPod)
			pvcInformer.GetIndexer().Add(existingPVC)
			pvcInformer.GetIndexer().Add(hpPVC)
			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, existingPVC, nil
			})
			shouldExpectPodDeletion(hpPod)
			addVirtualMachine(vmi)
			podInformer.GetIndexer().Add(virtlauncherPod)
			//Modify by adding a new hotplugged disk
			patch := `[ { "op": "test", "path": "/status/volumeStatus", "value": [{"name":"existing","target":""},{"name":"hotplug","target":"","hotplugVolume":{"attachPodName":"hp-volume-hotplug","attachPodUID":"abcd"}}] }, { "op": "replace", "path": "/status/volumeStatus", "value": [{"name":"existing","target":""},{"name":"hotplug","target":"","phase":"Detaching","hotplugVolume":{"attachPodName":"hp-volume-hotplug","attachPodUID":"abcd"}}] } ]`
			vmiInterface.EXPECT().Patch(vmi.Name, types.JSONPatchType, []byte(patch)).Return(vmi, nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)
			Expect(vmi.Status.Phase).To(Equal(v1.Running))
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

func NewPendingVirtualMachine(name string) *v1.VirtualMachineInstance {
	vmi := v1.NewMinimalVMI(name)
	vmi.UID = "1234"
	vmi.Status.Phase = v1.Pending
	kvcontroller.SetLatestApiVersionAnnotation(vmi)
	vmi.Status.ActivePods = make(map[types.UID]string)
	vmi.Status.VolumeStatus = make([]v1.VolumeStatus, 0)
	return vmi
}

func NewPodForVirtualMachine(vmi *v1.VirtualMachineInstance, phase k8sv1.PodPhase) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: vmi.Namespace,
			UID:       "virt-launch-uid",
			Labels: map[string]string{
				v1.AppLabel:       "virt-launcher",
				v1.CreatedByLabel: string(vmi.UID),
			},
			Annotations: map[string]string{
				v1.DomainAnnotation: vmi.Name,
			},
		},
		Status: k8sv1.PodStatus{
			Phase: phase,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: false, Name: "compute", State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}}},
			},
		},
	}
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
	}
}

func now() *metav1.Time {
	now := metav1.Now()
	return &now
}

func markAsReady(vmi *v1.VirtualMachineInstance) {
	kvcontroller.NewVirtualMachineInstanceConditionManager().AddPodCondition(vmi, &k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})
}

func markAsPodTerminating(vmi *v1.VirtualMachineInstance) {
	kvcontroller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, v1.VirtualMachineInstanceConditionType(k8sv1.PodReady))
	kvcontroller.NewVirtualMachineInstanceConditionManager().AddPodCondition(vmi, &k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionFalse, Reason: v1.PodTerminatingReason})
}

func markAsNonReady(vmi *v1.VirtualMachineInstance) {
	kvcontroller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, v1.VirtualMachineInstanceConditionType(k8sv1.PodReady))
	kvcontroller.NewVirtualMachineInstanceConditionManager().AddPodCondition(vmi, &k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionFalse})
}

func addActivePods(vmi *v1.VirtualMachineInstance, podUID types.UID, hostName string) *v1.VirtualMachineInstance {

	if vmi.Status.ActivePods != nil {
		vmi.Status.ActivePods[podUID] = hostName
	} else {
		vmi.Status.ActivePods = map[types.UID]string{
			podUID: hostName,
		}
	}
	return vmi
}
