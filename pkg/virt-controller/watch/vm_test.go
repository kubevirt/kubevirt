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

var _ = Describe("VM watcher", func() {
	log.Log.SetIOWriter(GinkgoWriter)

	var ctrl *gomock.Controller
	var vmInterface *kubecli.MockVMInterface
	var vmSource *framework.FakeControllerSource
	var podSource *framework.FakeControllerSource
	var vmInformer cache.SharedIndexInformer
	var podInformer cache.SharedIndexInformer
	var stop chan struct{}
	var controller *VMController
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue
	var podFeeder *testutils.PodFeeder
	var virtClient *kubecli.MockKubevirtClient
	var kubeClient *fake.Clientset

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

	shouldExpectVirtualMachineHandover := func(vm *v1.VirtualMachine) {
		vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
			Expect(arg.(*v1.VirtualMachine).Status.Phase).To(Equal(v1.Scheduled))
			Expect(arg.(*v1.VirtualMachine).Status.Conditions).To(BeEmpty())
			Expect(arg.(*v1.VirtualMachine).Finalizers).To(ContainElement(v1.VirtualMachineFinalizer))
		}).Return(vm, nil)
	}

	shouldExpectPodHandover := func() {
		kubeClient.Fake.PrependReactor("update", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())
			Expect(update.GetObject().(*k8sv1.Pod).Annotations[v1.OwnedByAnnotation]).To(Equal("virt-handler"))
			return true, update.GetObject(), nil
		})
	}

	shouldExpectVirtualMachineSchedulingState := func(vm *v1.VirtualMachine) {
		vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
			Expect(arg.(*v1.VirtualMachine).Status.Phase).To(Equal(v1.Scheduling))
			Expect(arg.(*v1.VirtualMachine).Status.Conditions).To(BeEmpty())
			Expect(arg.(*v1.VirtualMachine).Finalizers).To(ContainElement(v1.VirtualMachineFinalizer))
		}).Return(vm, nil)
	}

	shouldExpectVirtualMachineFailedState := func(vm *v1.VirtualMachine) {
		vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
			Expect(arg.(*v1.VirtualMachine).Status.Phase).To(Equal(v1.Failed))
			Expect(arg.(*v1.VirtualMachine).Status.Conditions).To(BeEmpty())
			Expect(arg.(*v1.VirtualMachine).Finalizers).To(ContainElement(v1.VirtualMachineFinalizer))
		}).Return(vm, nil)
	}

	syncCaches := func(stop chan struct{}) {
		go vmInformer.Run(stop)
		go podInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, podInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVMInterface(ctrl)

		vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
		podInformer, podSource = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		recorder = record.NewFakeRecorder(100)

		controller = NewVMController(services.NewTemplateService("a", "b", "c"), vmInformer, podInformer, recorder, virtClient)
		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
		podFeeder = testutils.NewPodFeeder(mockQueue, podSource)

		// Set up mock client
		virtClient.EXPECT().VM(k8sv1.NamespaceDefault).Return(vmInterface).AnyTimes()
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

	addVirtualMachine := func(vm *v1.VirtualMachine) {
		mockQueue.ExpectAdds(1)
		vmSource.Add(vm)
		mockQueue.Wait()
	}

	Context("On valid VirtualMachine given", func() {
		It("should create a corresponding Pod on VirtualMachine creation", func() {
			vm := NewPendingVirtualMachine("testvm")

			addVirtualMachine(vm)

			shouldExpectPodCreation(vm.UID)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})
		table.DescribeTable("should delete the corresponding Pod on VirtualMachine deletion with vm", func(phase v1.VMPhase) {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = phase
			vm.DeletionTimestamp = now()
			pod := NewPodForVirtualMachine(vm, k8sv1.PodRunning)

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			shouldExpectPodDeletion(pod)

			if vm.IsUnprocessed() {
				shouldExpectVirtualMachineSchedulingState(vm)
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
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = v1.Scheduling
			vm.DeletionTimestamp = now()
			pod := NewPodForVirtualMachine(vm, k8sv1.PodRunning)

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			shouldExpectPodDeletion(pod)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeletePodReason)

			modifiedPod := pod.DeepCopy()
			modifiedPod.DeletionTimestamp = now()

			podFeeder.Modify(modifiedPod)

			shouldExpectVirtualMachineFailedState(vm)

			controller.Execute()
		})
		table.DescribeTable("should not delete the corresponding Pod if the vm is in", func(phase v1.VMPhase) {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = phase
			pod := NewPodForVirtualMachine(vm, k8sv1.PodRunning)

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			controller.Execute()
		},
			table.Entry("succeeded state", v1.Failed),
			table.Entry("failed state", v1.Succeeded),
		)
		It("should do nothing if the vm is in final state", func() {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = v1.Failed
			vm.Finalizers = []string{}

			addVirtualMachine(vm)

			controller.Execute()
		})
		It("should set an error condition if creating the pod fails", func() {
			vm := NewPendingVirtualMachine("testvm")

			addVirtualMachine(vm)

			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Conditions[0].Reason).To(Equal("FailedCreate"))
			}).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedCreatePodReason)
		})
		It("should remove the error condition if the sync finally succeeds", func() {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Conditions = []v1.VirtualMachineCondition{{Type: v1.VirtualMachineSynchronized}}

			addVirtualMachine(vm)

			// Expect pod creation
			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				Expect(update.GetObject().(*k8sv1.Pod).Annotations[v1.OwnedByAnnotation]).To(Equal("virt-controller"))
				Expect(update.GetObject().(*k8sv1.Pod).Annotations[v1.CreatedByAnnotation]).To(Equal(string(vm.UID)))
				return true, update.GetObject(), nil
			})

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Conditions).To(BeEmpty())
			}).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})
		table.DescribeTable("should move the vm to scheduling state if a pod exists", func(phase k8sv1.PodPhase, isReady bool) {
			vm := NewPendingVirtualMachine("testvm")
			pod := NewPodForVirtualMachine(vm, phase)
			pod.Status.ContainerStatuses[0].Ready = isReady

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			shouldExpectVirtualMachineSchedulingState(vm)

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
		It("should move the vm to failed state if the pod disappears and the vm is in scheduling state", func() {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = v1.Scheduling

			addVirtualMachine(vm)

			shouldExpectVirtualMachineFailedState(vm)

			controller.Execute()
		})
		It("should move the vm to failed state if the vm is pending, no pod exists yet and gets deleted", func() {
			vm := NewPendingVirtualMachine("testvm")
			vm.DeletionTimestamp = now()

			addVirtualMachine(vm)

			shouldExpectVirtualMachineFailedState(vm)

			controller.Execute()
		})
		It("should hand over pod to virt-handler if pod is ready and running", func() {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = v1.Scheduling
			pod := NewPodForVirtualMachine(vm, k8sv1.PodRunning)

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			shouldExpectPodHandover()

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulHandOverPodReason)
		})
		It("should set an error condition if deleting the virtual machine pod fails", func() {
			vm := NewPendingVirtualMachine("testvm")
			vm.DeletionTimestamp = now()
			pod := NewPodForVirtualMachine(vm, k8sv1.PodRunning)

			// Expect pod delete
			kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Conditions[0].Reason).To(Equal(FailedDeletePodReason))
			}).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedDeletePodReason)
		})
		It("should set an error condition if handing over the pod to virt-handler fails", func() {
			vm := NewPendingVirtualMachine("testvm")
			pod := NewPodForVirtualMachine(vm, k8sv1.PodRunning)

			// Expect pod hand over
			kubeClient.Fake.PrependReactor("update", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Conditions[0].Reason).To(Equal(FailedHandOverPodReason))
			}).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedHandOverPodReason)
		})
		It("should set an error condition if creating the virtual machine pod fails", func() {
			vm := NewPendingVirtualMachine("testvm")

			// Expect pod creation
			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			addVirtualMachine(vm)

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Conditions[0].Reason).To(Equal(FailedCreatePodReason))
			}).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedCreatePodReason)
		})
		It("should update the virtual machine to scheduled if pod is ready, runnning and handed over to virt-handler", func() {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = v1.Scheduling
			pod := NewPodForVirtualMachine(vm, k8sv1.PodRunning)
			pod.Annotations[v1.OwnedByAnnotation] = "virt-handler"

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			shouldExpectVirtualMachineHandover(vm)

			controller.Execute()
		})
		It("should update the virtual machine to scheduled if pod is ready, triggered by pod change", func() {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = v1.Scheduling
			pod := NewPodForVirtualMachine(vm, k8sv1.PodPending)

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			controller.Execute()

			pod = NewPodForVirtualMachine(vm, k8sv1.PodRunning)
			pod.Annotations[v1.OwnedByAnnotation] = "virt-handler"

			podFeeder.Modify(pod)

			shouldExpectVirtualMachineHandover(vm)

			controller.Execute()
		})
		It("should update the virtual machine to failed if pod was not ready, triggered by pod delete", func() {
			vm := NewPendingVirtualMachine("testvm")
			pod := NewPodForVirtualMachine(vm, k8sv1.PodPending)
			vm.Status.Phase = v1.Scheduling

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			controller.Execute()

			podFeeder.Delete(pod)

			shouldExpectVirtualMachineFailedState(vm)

			controller.Execute()
		})
		table.DescribeTable("should remove the finalizer if no pod is present and the vm is in ", func(phase v1.VMPhase) {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = phase
			Expect(vm.Finalizers).To(ContainElement(v1.VirtualMachineFinalizer))

			addVirtualMachine(vm)

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Phase).To(Equal(phase))
				Expect(arg.(*v1.VirtualMachine).Status.Conditions).To(BeEmpty())
				Expect(arg.(*v1.VirtualMachine).Finalizers).ToNot(ContainElement(v1.VirtualMachineFinalizer))
			}).Return(vm, nil)

			controller.Execute()
		},
			table.Entry("failed state", v1.Succeeded),
			table.Entry("succeeded state", v1.Failed),
		)
		table.DescribeTable("should do nothing if pod is handed to virt-handler", func(phase k8sv1.PodPhase) {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = v1.Scheduled
			pod := NewPodForVirtualMachine(vm, phase)
			pod.Annotations[v1.OwnedByAnnotation] = "virt-handler"

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			controller.Execute()
		},
			table.Entry("and in running state", k8sv1.PodRunning),
			table.Entry("and in unknown state", k8sv1.PodUnknown),
			table.Entry("and in succeeded state", k8sv1.PodSucceeded),
			table.Entry("and in failed state", k8sv1.PodFailed),
			table.Entry("and in pending state", k8sv1.PodPending),
		)
		It("should do nothing if the vm is handed over to virt-handler and the pod disappears", func() {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = v1.Scheduled

			addVirtualMachine(vm)

			controller.Execute()
		})
		table.DescribeTable("should move the vm to failed if pod is not handed over", func(phase k8sv1.PodPhase) {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = v1.Scheduling
			Expect(vm.Finalizers).To(ContainElement(v1.VirtualMachineFinalizer))
			pod := NewPodForVirtualMachine(vm, phase)

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			shouldExpectVirtualMachineFailedState(vm)

			controller.Execute()
		},
			table.Entry("and in succeeded state", k8sv1.PodSucceeded),
			table.Entry("and in failed state", k8sv1.PodFailed),
		)
	})
})

func NewPendingVirtualMachine(name string) *v1.VirtualMachine {
	vm := v1.NewMinimalVM(name)
	vm.UID = "1234"
	vm.Status.Phase = v1.Pending
	addInitializedAnnotation(vm)
	return vm
}

func NewPodForVirtualMachine(vm *v1.VirtualMachine, phase k8sv1.PodPhase) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: vm.Namespace,
			Labels: map[string]string{
				v1.AppLabel:    "virt-launcher",
				v1.DomainLabel: vm.Name,
			},
			Annotations: map[string]string{
				v1.CreatedByAnnotation: string(vm.UID),
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
