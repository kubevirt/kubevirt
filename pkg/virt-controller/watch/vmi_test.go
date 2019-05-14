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
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/controller"

	fakenetworkclient "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned/fake"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("VirtualMachineInstance watcher", func() {
	log.Log.SetIOWriter(GinkgoWriter)

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

	shouldExpectPodCreation := func(uid types.UID) {
		// Expect pod creation
		kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(update.GetObject().(*k8sv1.Pod).Labels[v1.CreatedByLabel]).To(Equal(string(uid)))
			return true, update.GetObject(), nil
		})
	}

	shouldExpectMultiplePodDeletions := func(pod *k8sv1.Pod, deletionCount *int) {
		// Expect pod deletion
		kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.DeleteAction)
			Expect(ok).To(BeTrue())
			Expect(pod.Namespace).To(Equal(update.GetNamespace()))
			*deletionCount++
			return true, nil, nil
		})
	}

	shouldExpectPodDeletion := func(pod *k8sv1.Pod) {
		// Expect pod deletion
		kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
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
		kubeClient.Fake.PrependReactor("update", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
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

	syncCaches := func(stop chan struct{}) {
		go vmiInformer.Run(stop)
		go podInformer.Run(stop)

		go dataVolumeInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop,
			vmiInformer.HasSynced,
			podInformer.HasSynced,
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

		config, _ := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})
		pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		controller = NewVMIController(
			services.NewTemplateService("a", "b", "c", "d", pvcInformer.GetStore(), virtClient, config),
			vmiInformer,
			podInformer,
			recorder,
			virtClient,
			dataVolumeInformer)
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
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
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

		It("should create a corresponding Pod on VMI creation when DataVolume is ready", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "test1",
					},
				},
			})

			dataVolume := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test1",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Succeeded,
				},
			}

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)
			shouldExpectPodCreation(vmi.UID)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})

		It("should not create a corresponding Pod on VMI creation when DataVolume is pending", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "test1",
					},
				},
			})

			dataVolume := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test1",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Pending,
				},
			}

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)

			controller.Execute()
		})

		It("VMI should fail if DataVolume fails to import", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "test1",
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "test1",
					},
				},
			})

			dataVolume := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test1",
					Namespace: vmi.Namespace,
				},
				Status: cdiv1.DataVolumeStatus{
					Phase: cdiv1.Failed,
				},
			}

			addVirtualMachine(vmi)
			dataVolumeFeeder.Add(dataVolume)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Phase).To(Equal(v1.Failed))
			}).Return(vmi, nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, FailedDataVolumeImportReason)
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
			pod2 := pod.DeepCopy()
			pod2.UID = "123-123-123"
			pod2.Name = "test2"

			// The 3rd one is not.
			pod3 := pod.DeepCopy()
			pod3.UID = "123-123-123"
			pod3.Name = "test2"
			pod3.Annotations = make(map[string]string)
			pod3.Labels = make(map[string]string)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)
			podFeeder.Add(pod2)

			deletionCount := 0
			shouldExpectMultiplePodDeletions(pod, &deletionCount)

			if vmi.IsUnprocessed() {
				shouldExpectVirtualMachineSchedulingState(vmi)
			}

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

			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
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

			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
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
			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
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
		table.DescribeTable("should move the vmi to scheduling state if a pod exists", func(phase k8sv1.PodPhase, isReady bool) {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, phase)
			pod.Status.ContainerStatuses[0].Ready = isReady

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

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

			shouldExpectVirtualMachineScheduledState(vmi)

			controller.Execute()
		},
			table.Entry("with ready infra container and not ready compute container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: false}, {Name: "kubevirt-infra", Ready: true}},
			),
			table.Entry("with ready compute container and no infra container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: true}},
			),
			table.Entry("with ready compute container and no ready istio-proxy container",
				[]k8sv1.ContainerStatus{{Name: "compute", Ready: true}, {Name: "istio-proxy", Ready: false}},
			),
		)
		table.DescribeTable("should not hand over pod to virt-handler if pod is ready and running", func(containerStatus []k8sv1.ContainerStatus) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.ContainerStatuses = containerStatus

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

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

			selectedPod, err := controller.currentPod(vmi)
			Expect(err).To(BeNil())

			Expect(selectedPod.UID).To(Equal(pod.UID))
		})

		It("should set an error condition if deleting the virtual machine pod fails", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.DeletionTimestamp = now()
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			// Expect pod delete
			kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
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
					k8sv1.Container{
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
			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
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
		It("should update the virtual machine to scheduled if pod is ready, triggered by pod change", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodPending)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

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
			vmi.Finalizers = append(vmi.Finalizers, v1.VirtualMachineInstanceFinalizer)
			Expect(vmi.Finalizers).To(ContainElement(v1.VirtualMachineInstanceFinalizer))

			addVirtualMachine(vmi)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
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

			controller.Execute()
		},
			table.Entry("and in running state", k8sv1.PodRunning),
			table.Entry("and in unknown state", k8sv1.PodUnknown),
			table.Entry("and in succeeded state", k8sv1.PodSucceeded),
			table.Entry("and in failed state", k8sv1.PodFailed),
			table.Entry("and in pending state", k8sv1.PodPending),
		)

		It("should add a ready condition if it is present on the pod and the VMI is in running state", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Running
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)
			pod.Status.Conditions = []k8sv1.PodCondition{{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue}}

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			patch := `[ { "op": "test", "path": "/status/conditions", "value": null }, { "op": "replace", "path": "/status/conditions", "value": [{"type":"Ready","status":"True","lastProbeTime":null,"lastTransitionTime":null}] } ]`
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

			controller.Execute()
		},
			table.Entry("in succeeded state", v1.Succeeded),
			table.Entry("in failed state", v1.Failed),
			table.Entry("in scheduled state", v1.Scheduled),
		)

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

	Context("When VirtualMachineInstance is connected to a network", func() {
		It("should report the status of this network", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			networkName := "test net"
			podIp := "1.1.1.1"
			pod.Status.PodIP = podIp
			vmi.Spec.Networks = []v1.Network{
				v1.Network{
					Name: networkName,
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{
							VMNetworkCIDR: "1.1.1.1",
						},
					},
				},
			}
			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Interfaces)).To(Equal(1))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].Name).To(Equal(networkName))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].IP).To(Equal(podIp))
			}).Return(vmi, nil)
			controller.Execute()
		})

		It("should only report the pod network in status", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			networkName := "test net"
			vmi.Spec.Networks = []v1.Network{
				v1.Network{
					Name: networkName,
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{
							VMNetworkCIDR: "1.1.1.1",
						},
					},
				},
				v1.Network{
					Name: networkName,
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: "multus",
						},
					},
				},
			}
			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Interfaces)).To(Equal(1))
			}).Return(vmi, nil)
			controller.Execute()
		})
	})
})

func NewPendingVirtualMachine(name string) *v1.VirtualMachineInstance {
	vmi := v1.NewMinimalVMI(name)
	vmi.UID = "1234"
	vmi.Status.Phase = v1.Pending
	return vmi
}

func NewPodForVirtualMachine(vmi *v1.VirtualMachineInstance, phase k8sv1.PodPhase) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: vmi.Namespace,
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
				{Ready: true, Name: "kubevirt-infra"},
				{Ready: false, Name: "compute"},
			},
		},
	}
}

func now() *metav1.Time {
	now := metav1.Now()
	return &now
}

func markAsReady(vmi *v1.VirtualMachineInstance) {
	controller.NewVirtualMachineInstanceConditionManager().AddPodCondition(vmi, &k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionTrue})
}

func markAsNonReady(vmi *v1.VirtualMachineInstance) {
	controller.NewVirtualMachineInstanceConditionManager().RemoveCondition(vmi, v1.VirtualMachineInstanceConditionType(k8sv1.PodReady))
	controller.NewVirtualMachineInstanceConditionManager().AddPodCondition(vmi, &k8sv1.PodCondition{Type: k8sv1.PodReady, Status: k8sv1.ConditionFalse})
}
