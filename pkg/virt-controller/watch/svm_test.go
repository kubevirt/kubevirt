package watch

import (
	"fmt"

	"github.com/pborman/uuid"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	// "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("StatefulVirtualMachine", func() {

	Context("One valid StatefulVirtualMachine controller given", func() {

		var ctrl *gomock.Controller
		var vmInterface *kubecli.MockVMInterface
		var svmInterface *kubecli.MockStatefulVirtualMachineInterface
		var vmSource *framework.FakeControllerSource
		var svmSource *framework.FakeControllerSource
		var vmInformer cache.SharedIndexInformer
		var svmInformer cache.SharedIndexInformer
		var stop chan struct{}
		var controller *SVMController
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue
		var vmFeeder *testutils.VirtualMachineFeeder

		syncCaches := func(stop chan struct{}) {
			go vmInformer.Run(stop)
			go svmInformer.Run(stop)
			Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, svmInformer.HasSynced)).To(BeTrue())
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			vmInterface = kubecli.NewMockVMInterface(ctrl)
			svmInterface = kubecli.NewMockStatefulVirtualMachineInterface(ctrl)

			vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
			svmInformer, svmSource = testutils.NewFakeInformerFor(&v1.StatefulVirtualMachine{})
			recorder = record.NewFakeRecorder(100)

			controller = NewSVMController(vmInformer, svmInformer, recorder, virtClient)
			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockQueue = testutils.NewMockWorkQueue(controller.Queue)
			controller.Queue = mockQueue
			vmFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmSource)

			// Set up mock client
			virtClient.EXPECT().VM(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()
			virtClient.EXPECT().StatefulVirtualMachine(metav1.NamespaceDefault).Return(svmInterface).AnyTimes()
		})

		addStatefulVirtualMachine := func(svm *v1.StatefulVirtualMachine) {
			syncCaches(stop)
			mockQueue.ExpectAdds(1)
			svmSource.Add(svm)
			mockQueue.Wait()
		}

		It("should create missing VM", func() {
			svm, vm := DefaultStatefulVirtualMachine(true)

			addStatefulVirtualMachine(svm)

			// expect creation called
			vmInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.Name).To(Equal("testvm"))
			}).Return(vm, nil)

			// expect update status is called
			svmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.StatefulVirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*v1.StatefulVirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should update status to created if the vm exists", func() {
			svm, vm := DefaultStatefulVirtualMachine(true)
			vm.Status.Phase = v1.Scheduled

			addStatefulVirtualMachine(svm)
			vmFeeder.Add(vm)

			// expect update status is called
			svmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.StatefulVirtualMachine).Status.Created).To(BeTrue())
				Expect(arg.(*v1.StatefulVirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()
		})

		It("should update status to created and ready when vm is running", func() {
			svm, vm := DefaultStatefulVirtualMachine(true)

			addStatefulVirtualMachine(svm)
			vmFeeder.Add(vm)

			// expect update status is called
			svmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.StatefulVirtualMachine).Status.Created).To(BeTrue())
				Expect(arg.(*v1.StatefulVirtualMachine).Status.Ready).To(BeTrue())
			}).Return(nil, nil)

			controller.Execute()
		})

		It("should have stable firmware UUIDs", func() {
			svm1, _ := DefaultStatefulVirtualMachineWithNames(true, "testsvm1", "testvm1")
			vm1 := controller.setupVMFromSVM(svm1)

			// intentionally use the same names
			svm2, _ := DefaultStatefulVirtualMachineWithNames(true, "testsvm1", "testvm1")
			vm2 := controller.setupVMFromSVM(svm2)
			Expect(vm1.Spec.Domain.Firmware.UUID).To(Equal(vm2.Spec.Domain.Firmware.UUID))

			// now we want different names
			svm3, _ := DefaultStatefulVirtualMachineWithNames(true, "testsvm3", "testvm3")
			vm3 := controller.setupVMFromSVM(svm3)
			Expect(vm1.Spec.Domain.Firmware.UUID).NotTo(Equal(vm3.Spec.Domain.Firmware.UUID))
		})

		It("should honour any firmware UUID present in the template", func() {
			uid := uuid.NewRandom().String()
			svm1, _ := DefaultStatefulVirtualMachineWithNames(true, "testsvm1", "testvm1")
			svm1.Spec.Template.Spec.Domain.Firmware = &virtv1.Firmware{UUID: types.UID(uid)}

			vm1 := controller.setupVMFromSVM(svm1)
			Expect(string(vm1.Spec.Domain.Firmware.UUID)).To(Equal(uid))
		})

		It("should delete VM when stopped", func() {
			svm, vm := DefaultStatefulVirtualMachine(false)

			addStatefulVirtualMachine(svm)
			vmFeeder.Add(vm)

			// svmInterface.EXPECT().Update(gomock.Any()).Return(svm, nil)
			vmInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

			svmInterface.EXPECT().Update(gomock.Any()).Times(1).Return(svm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should ignore non-matching VMs", func() {
			svm, vm := DefaultStatefulVirtualMachine(true)

			nonMatchingVM := v1.NewMinimalVM("testvm1")
			nonMatchingVM.ObjectMeta.Labels = map[string]string{"test": "test1"}

			addStatefulVirtualMachine(svm)

			// We still expect three calls to create VMs, since VM does not meet the requirements
			vmSource.Add(nonMatchingVM)

			vmInterface.EXPECT().Create(gomock.Any()).Return(vm, nil)
			svmInterface.EXPECT().Update(gomock.Any()).Times(2).Return(svm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should detect that it is orphan deleted and remove the owner reference on the remaining VM", func() {
			svm, vm := DefaultStatefulVirtualMachine(true)

			// Mark it as orphan deleted
			now := metav1.Now()
			svm.ObjectMeta.DeletionTimestamp = &now
			svm.ObjectMeta.Finalizers = []string{metav1.FinalizerOrphanDependents}

			addStatefulVirtualMachine(svm)
			vmFeeder.Add(vm)

			vmInterface.EXPECT().Patch(vm.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that a VM already exists and adopt it", func() {
			svm, vm := DefaultStatefulVirtualMachine(true)
			vm.OwnerReferences = []metav1.OwnerReference{}

			addStatefulVirtualMachine(svm)
			vmFeeder.Add(vm)

			svmInterface.EXPECT().Get(svm.ObjectMeta.Name, gomock.Any()).Return(svm, nil)
			svmInterface.EXPECT().Update(gomock.Any()).Return(svm, nil)
			vmInterface.EXPECT().Patch(vm.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that it has nothing to do beside updating the status", func() {
			svm, vm := DefaultStatefulVirtualMachine(true)

			addStatefulVirtualMachine(svm)
			vmFeeder.Add(vm)

			svmInterface.EXPECT().Update(gomock.Any()).Return(svm, nil)

			controller.Execute()
		})

		It("should add a fail condition if start up fails", func() {
			svm, vm := DefaultStatefulVirtualMachine(true)

			addStatefulVirtualMachine(svm)
			// vmFeeder.Add(vm)

			vmInterface.EXPECT().Create(gomock.Any()).Return(vm, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			svmInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objSVM := obj.(*v1.StatefulVirtualMachine)
				Expect(objSVM.Status.Conditions).To(HaveLen(1))
				cond := objSVM.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.StatefulVirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedCreate"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			}).Return(svm, nil)

			controller.Execute()

			testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
		})

		It("should add a fail condition if deletion fails", func() {
			svm, vm := DefaultStatefulVirtualMachine(false)

			addStatefulVirtualMachine(svm)
			vmFeeder.Add(vm)

			vmInterface.EXPECT().Delete(vm.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("failure"))

			svmInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objSVM := obj.(*v1.StatefulVirtualMachine)
				Expect(objSVM.Status.Conditions).To(HaveLen(1))
				cond := objSVM.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.StatefulVirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedDelete"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			})

			controller.Execute()

			testutils.ExpectEvents(recorder, FailedDeleteVirtualMachineReason)
		})
	})
})

func StatefulVirtualMachineFromVM(name string, vm *v1.VirtualMachine, started bool) *v1.StatefulVirtualMachine {
	svm := &v1.StatefulVirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vm.ObjectMeta.Namespace, ResourceVersion: "1"},
		Spec: v1.StatefulVirtualMachineSpec{
			Running: started,
			Template: &v1.VMTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vm.ObjectMeta.Name,
					Labels: vm.ObjectMeta.Labels,
				},
				Spec: vm.Spec,
			},
		},
	}
	return svm
}

func DefaultStatefulVirtualMachineWithNames(started bool, svmName string, vmName string) (*v1.StatefulVirtualMachine, *v1.VirtualMachine) {
	vm := v1.NewMinimalVM(vmName)
	vm.Status.Phase = v1.Running
	svm := StatefulVirtualMachineFromVM(svmName, vm, started)
	t := true
	vm.OwnerReferences = []metav1.OwnerReference{{
		APIVersion:         v1.StatefulVirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               v1.StatefulVirtualMachineGroupVersionKind.Kind,
		Name:               svm.ObjectMeta.Name,
		UID:                svm.ObjectMeta.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}}
	return svm, vm
}

func DefaultStatefulVirtualMachine(started bool) (*v1.StatefulVirtualMachine, *v1.VirtualMachine) {
	return DefaultStatefulVirtualMachineWithNames(started, "testvm", "testvm")
}
