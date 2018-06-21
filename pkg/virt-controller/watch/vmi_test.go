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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/cache"

	k8sv1 "k8s.io/api/core/v1"

	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	"fmt"

	"github.com/onsi/ginkgo/extensions/table"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/api/v1"
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
	var configMapInformer cache.SharedIndexInformer

	shouldExpectPodCreation := func(uid types.UID) {
		// Expect pod creation
		kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(update.GetObject().(*k8sv1.Pod).Annotations[v1.OwnedByAnnotation]).To(Equal("virt-controller"))
			Expect(update.GetObject().(*k8sv1.Pod).Annotations[v1.CreatedByAnnotation]).To(Equal(string(uid)))
			return true, update.GetObject(), nil
		})
	}

	shouldExpectPodDeletion := func(pod *k8sv1.Pod) {
		// Expect pod creation
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
			Expect(arg.(*v1.VirtualMachineInstance).Finalizers).To(ContainElement(v1.VirtualMachineInstanceFinalizer))
		}).Return(vmi, nil)
	}

	shouldExpectPodHandover := func() {
		kubeClient.Fake.PrependReactor("update", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			Expect(update.GetObject().(*k8sv1.Pod).Annotations[v1.OwnedByAnnotation]).To(Equal("virt-handler"))
			return true, update.GetObject(), nil
		})
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
			Expect(arg.(*v1.VirtualMachineInstance).Finalizers).To(ContainElement(v1.VirtualMachineInstanceFinalizer))
		}).Return(vmi, nil)
	}

	shouldExpectVirtualMachineFailedState := func(vmi *v1.VirtualMachineInstance) {
		vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
			Expect(arg.(*v1.VirtualMachineInstance).Status.Phase).To(Equal(v1.Failed))
			Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions).To(BeEmpty())
			Expect(arg.(*v1.VirtualMachineInstance).Finalizers).To(ContainElement(v1.VirtualMachineInstanceFinalizer))
		}).Return(vmi, nil)
	}

	syncCaches := func(stop chan struct{}) {
		go vmiInformer.Run(stop)
		go podInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmiInformer.HasSynced, podInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		vmiInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		podInformer, podSource = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		recorder = record.NewFakeRecorder(100)

		configMapInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		controller = NewVMIController(services.NewTemplateService("a", "b", "c", configMapInformer.GetStore()), vmiInformer, podInformer, recorder, virtClient, configMapInformer)
		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
		podFeeder = testutils.NewPodFeeder(mockQueue, podSource)

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstance(k8sv1.NamespaceDefault).Return(vmiInterface).AnyTimes()
		kubeClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

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

	Context("On valid VirtualMachineInstance given", func() {
		It("should create a corresponding Pod on VirtualMachineInstance creation", func() {
			vmi := NewPendingVirtualMachine("testvmi")

			addVirtualMachine(vmi)

			shouldExpectPodCreation(vmi.UID)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})
		table.DescribeTable("should delete the corresponding Pod on VirtualMachineInstance deletion with vmi", func(phase v1.VirtualMachineInstancePhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = phase
			vmi.DeletionTimestamp = now()
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			shouldExpectPodDeletion(pod)

			if vmi.IsUnprocessed() {
				shouldExpectVirtualMachineSchedulingState(vmi)
			}

			controller.Execute()

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
		It("should not try to delete a pod again, which is already marked for deletion and go to failed state, when in sheduling state", func() {
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
		It("should remove the error condition if the sync finally succeeds", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{Type: v1.VirtualMachineInstanceSynchronized}}

			addVirtualMachine(vmi)

			// Expect pod creation
			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				Expect(update.GetObject().(*k8sv1.Pod).Annotations[v1.OwnedByAnnotation]).To(Equal("virt-controller"))
				Expect(update.GetObject().(*k8sv1.Pod).Annotations[v1.CreatedByAnnotation]).To(Equal(string(vmi.UID)))
				return true, update.GetObject(), nil
			})

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions).To(BeEmpty())
			}).Return(vmi, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})
		table.DescribeTable("should move the vmi to scheduling state if a pod exists", func(phase k8sv1.PodPhase, isReady bool) {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, phase)
			pod.Status.ContainerStatuses[0].Ready = isReady

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			shouldExpectVirtualMachineSchedulingState(vmi)

			if phase == k8sv1.PodRunning && isReady {
				shouldExpectPodHandover()
			}

			controller.Execute()

			if phase == k8sv1.PodRunning && isReady {
				testutils.ExpectEvent(recorder, SuccessfulHandOverPodReason)
			}
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
					Expect(arg.(*v1.VirtualMachineInstance).Finalizers).To(ContainElement(v1.VirtualMachineInstanceFinalizer))
				}).Return(vmi, nil)

				controller.Execute()
			})
		})

		Context("when Pod recovers from scheduling issues", func() {
			table.DescribeTable("it should remove scheduling pod condition from the VirtualMachineInstance if the pod", func(owner string, podPhase k8sv1.PodPhase) {
				vmi := NewPendingVirtualMachine("testvmi")
				vmi.Status.Phase = v1.Scheduling

				pod := NewPodForVirtualMachine(vmi, podPhase)
				pod.Annotations[v1.OwnedByAnnotation] = owner

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
		It("should hand over pod to virt-handler if pod is ready and running", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			shouldExpectPodHandover()

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulHandOverPodReason)
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
		It("should set an error condition if handing over the pod to virt-handler fails", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			pod := NewPodForVirtualMachine(vmi, k8sv1.PodRunning)

			// Expect pod hand over
			kubeClient.Fake.PrependReactor("update", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			addVirtualMachine(vmi)
			podFeeder.Add(pod)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.Conditions[0].Reason).To(Equal(FailedHandOverPodReason))
			}).Return(vmi, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedHandOverPodReason)
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
			pod.Annotations[v1.OwnedByAnnotation] = "virt-handler"

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
			pod.Annotations[v1.OwnedByAnnotation] = "virt-handler"

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
		table.DescribeTable("should remove the finalizer if no pod is present and the vmi is in ", func(phase v1.VirtualMachineInstancePhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = phase
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
			pod.Annotations[v1.OwnedByAnnotation] = "virt-handler"

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
		It("should do nothing if the vmi is handed over to virt-handler and the pod disappears", func() {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduled

			addVirtualMachine(vmi)

			controller.Execute()
		})
		table.DescribeTable("should move the vmi to failed if pod is not handed over", func(phase k8sv1.PodPhase) {
			vmi := NewPendingVirtualMachine("testvmi")
			vmi.Status.Phase = v1.Scheduling
			Expect(vmi.Finalizers).To(ContainElement(v1.VirtualMachineInstanceFinalizer))
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
})

func NewPendingVirtualMachine(name string) *v1.VirtualMachineInstance {
	vmi := v1.NewMinimalVMI(name)
	vmi.UID = "1234"
	vmi.Status.Phase = v1.Pending
	addInitializedAnnotation(vmi)
	return vmi
}

func NewPodForVirtualMachine(vmi *v1.VirtualMachineInstance, phase k8sv1.PodPhase) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: vmi.Namespace,
			Labels: map[string]string{
				v1.AppLabel:    "virt-launcher",
				v1.DomainLabel: vmi.Name,
			},
			Annotations: map[string]string{
				v1.CreatedByAnnotation: string(vmi.UID),
				v1.OwnedByAnnotation:   "virt-controller",
			},
		},
		Status: k8sv1.PodStatus{
			Phase: phase,
			ContainerStatuses: []k8sv1.ContainerStatus{
				{Ready: true},
			},
		},
	}
}

func now() *metav1.Time {
	now := metav1.Now()
	return &now
}
