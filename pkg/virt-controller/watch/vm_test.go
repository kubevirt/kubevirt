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

	kubev1 "k8s.io/api/core/v1"

	"github.com/golang/mock/gomock"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	"fmt"
	"github.com/onsi/ginkgo/extensions/table"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
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

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVMInterface(ctrl)

		vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
		podInformer, podSource = testutils.NewFakeInformerFor(&kubev1.Pod{})
		recorder = record.NewFakeRecorder(100)

		controller = NewVMController(services.NewTemplateService("a", "b", "c"), vmInformer, podInformer, recorder, virtClient)
		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
		podFeeder = testutils.NewPodFeeder(mockQueue, podSource)

		// Set up mock client
		virtClient.EXPECT().VM(kubev1.NamespaceDefault).Return(vmInterface).AnyTimes()
		kubeClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
	})
	AfterEach(func() {
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
		ctrl.Finish()
	})

	syncCaches := func(stop chan struct{}) {
		go vmInformer.Run(stop)
		go podInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, podInformer.HasSynced)).To(BeTrue())
	}

	addVirtualMachine := func(vm *v1.VirtualMachine) {
		syncCaches(stop)
		mockQueue.ExpectAdds(1)
		vmSource.Add(vm)
		mockQueue.Wait()
	}

	Context("On valid VirtualMachine given", func() {
		It("should create a corresponding Pod on VirtualMachineCreation", func() {
			vm := NewPendingVirtualMachine("testvm")

			addVirtualMachine(vm)

			// Expect pod creation
			kubeClient.Fake.PrependReactor("create", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				Expect(update.GetObject().(*kubev1.Pod).Annotations[v1.OwnedByAnnotation]).To(Equal("virt-controller"))
				Expect(update.GetObject().(*kubev1.Pod).Annotations[v1.CreatedByAnnotation]).To(Equal(string(vm.UID)))
				return true, update.GetObject(), nil
			})

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
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
				Expect(update.GetObject().(*kubev1.Pod).Annotations[v1.OwnedByAnnotation]).To(Equal("virt-controller"))
				Expect(update.GetObject().(*kubev1.Pod).Annotations[v1.CreatedByAnnotation]).To(Equal(string(vm.UID)))
				return true, update.GetObject(), nil
			})

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Conditions).To(BeEmpty())
			}).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreatePodReason)
		})
		It("should move the vm to scheduling state if a pod exists but is not yet ready", func() {
			vm := NewPendingVirtualMachine("testvm")
			pod := NewPodForVirtualMachine(vm, kubev1.PodRunning)
			pod.Status.ContainerStatuses[0].Ready = false

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Phase).To(Equal(v1.Scheduling))
				Expect(arg.(*v1.VirtualMachine).Status.Conditions).To(BeEmpty())
			}).Return(vm, nil)

			controller.Execute()
		})
		It("should hand over pod to virt-handler if pod is ready and running", func() {
			vm := NewPendingVirtualMachine("testvm")
			pod := NewPodForVirtualMachine(vm, kubev1.PodRunning)

			// Expect pod hand over
			kubeClient.Fake.PrependReactor("update", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.UpdateAction)
				Expect(ok).To(BeTrue())
				Expect(update.GetObject().(*kubev1.Pod).Annotations[v1.OwnedByAnnotation]).To(Equal("virt-handler"))
				return true, update.GetObject(), nil
			})

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulHandOverPodReason)
		})
		It("should set an error condition if handing over the pod to virt-handler fails", func() {
			vm := NewPendingVirtualMachine("testvm")
			pod := NewPodForVirtualMachine(vm, kubev1.PodRunning)

			// Expect pod hand over
			kubeClient.Fake.PrependReactor("update", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, fmt.Errorf("random error")
			})

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Conditions[0].Reason).To(Equal("FailedHandOver"))
			}).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedHandOverPodReason)
		})
		It("should update the virtual machine to scheduled if pod is ready, runnning and handed over to virt-handler", func() {
			vm := NewPendingVirtualMachine("testvm")
			pod := NewPodForVirtualMachine(vm, kubev1.PodRunning)
			pod.Annotations[v1.OwnedByAnnotation] = "virt-handler"

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Phase).To(Equal(v1.Scheduled))
				Expect(arg.(*v1.VirtualMachine).Status.Conditions).To(BeEmpty())
			}).Return(vm, nil)

			controller.Execute()
		})
		table.DescribeTable("should do nothing if pod is handed to virt-handler", func(phase kubev1.PodPhase) {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = v1.Scheduled
			pod := NewPodForVirtualMachine(vm, phase)
			pod.Annotations[v1.OwnedByAnnotation] = "virt-handler"

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			controller.Execute()
		},
			table.Entry("and in running state", kubev1.PodRunning),
			table.Entry("and in unknown state", kubev1.PodUnknown),
			table.Entry("and in succeeded state", kubev1.PodSucceeded),
			table.Entry("and in failed state", kubev1.PodFailed),
			table.Entry("and in pending state", kubev1.PodPending),
		)
		It("should do nothing if the vm is handed over to virt-handler and the pod disappears", func() {
			vm := NewPendingVirtualMachine("testvm")
			vm.Status.Phase = v1.Scheduled

			addVirtualMachine(vm)

			controller.Execute()
		})
		table.DescribeTable("should move the vm to failed if pod is not handed over", func(phase kubev1.PodPhase) {
			vm := NewPendingVirtualMachine("testvm")
			Expect(vm.Finalizers).To(ContainElement(v1.VirtualMachineFinalizer))
			pod := NewPodForVirtualMachine(vm, phase)

			addVirtualMachine(vm)
			podFeeder.Add(pod)

			vmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Phase).To(Equal(v1.Failed))
				Expect(arg.(*v1.VirtualMachine).Status.Conditions).To(BeEmpty())
				Expect(arg.(*v1.VirtualMachine).Finalizers).ToNot(ContainElement(v1.VirtualMachineFinalizer))
			}).Return(vm, nil)

			controller.Execute()
		},
			table.Entry("and in succeeded state", kubev1.PodSucceeded),
			table.Entry("and in failed state", kubev1.PodFailed),
		)
	})
})

func NewPendingVirtualMachine(name string) *v1.VirtualMachine {
	vm := v1.NewMinimalVM(name)
	vm.UID = "1234"
	addInitializedAnnotation(vm)
	return vm
}

func NewPodForVirtualMachine(vm *v1.VirtualMachine, phase kubev1.PodPhase) *kubev1.Pod {
	return &kubev1.Pod{
		ObjectMeta: v12.ObjectMeta{
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
		Status: kubev1.PodStatus{
			Phase: phase,
			ContainerStatuses: []kubev1.ContainerStatus{
				{Ready: true},
			},
		},
	}
}
