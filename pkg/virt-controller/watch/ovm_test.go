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

var _ = Describe("VirtualMachine", func() {

	Context("One valid VirtualMachine controller given", func() {

		var ctrl *gomock.Controller
		var vmInterface *kubecli.MockVirtualMachineInstanceInterface
		var ovmInterface *kubecli.MockVirtualMachineInterface
		var vmSource *framework.FakeControllerSource
		var ovmSource *framework.FakeControllerSource
		var vmInformer cache.SharedIndexInformer
		var ovmInformer cache.SharedIndexInformer
		var stop chan struct{}
		var controller *VMController
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue
		var vmFeeder *testutils.VirtualMachineFeeder

		syncCaches := func(stop chan struct{}) {
			go vmInformer.Run(stop)
			go ovmInformer.Run(stop)
			Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, ovmInformer.HasSynced)).To(BeTrue())
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			vmInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			ovmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)

			vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			ovmInformer, ovmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
			recorder = record.NewFakeRecorder(100)

			controller = NewVMController(vmInformer, ovmInformer, recorder, virtClient)
			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockQueue = testutils.NewMockWorkQueue(controller.Queue)
			controller.Queue = mockQueue
			vmFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmSource)

			// Set up mock client
			virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()
			virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(ovmInterface).AnyTimes()
		})

		addVirtualMachine := func(ovm *v1.VirtualMachine) {
			syncCaches(stop)
			mockQueue.ExpectAdds(1)
			ovmSource.Add(ovm)
			mockQueue.Wait()
		}

		It("should create missing VirtualMachineInstance", func() {
			ovm, vm := DefaultVirtualMachine(true)

			addVirtualMachine(ovm)

			// expect creation called
			vmInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).ObjectMeta.Name).To(Equal("testvm"))
			}).Return(vm, nil)

			// expect update status is called
			ovmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*v1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should update status to created if the vm exists", func() {
			ovm, vm := DefaultVirtualMachine(true)
			vm.Status.Phase = v1.Scheduled

			addVirtualMachine(ovm)
			vmFeeder.Add(vm)

			// expect update status is called
			ovmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Created).To(BeTrue())
				Expect(arg.(*v1.VirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()
		})

		It("should update status to created and ready when vm is running", func() {
			ovm, vm := DefaultVirtualMachine(true)

			addVirtualMachine(ovm)
			vmFeeder.Add(vm)

			// expect update status is called
			ovmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).Status.Created).To(BeTrue())
				Expect(arg.(*v1.VirtualMachine).Status.Ready).To(BeTrue())
			}).Return(nil, nil)

			controller.Execute()
		})

		It("should have stable firmware UUIDs", func() {
			ovm1, _ := DefaultVirtualMachineWithNames(true, "testovm1", "testvm1")
			vm1 := controller.setupVMIFromVM(ovm1)

			// intentionally use the same names
			ovm2, _ := DefaultVirtualMachineWithNames(true, "testovm1", "testvm1")
			vm2 := controller.setupVMIFromVM(ovm2)
			Expect(vm1.Spec.Domain.Firmware.UUID).To(Equal(vm2.Spec.Domain.Firmware.UUID))

			// now we want different names
			ovm3, _ := DefaultVirtualMachineWithNames(true, "testovm3", "testvm3")
			vm3 := controller.setupVMIFromVM(ovm3)
			Expect(vm1.Spec.Domain.Firmware.UUID).NotTo(Equal(vm3.Spec.Domain.Firmware.UUID))
		})

		It("should honour any firmware UUID present in the template", func() {
			uid := uuid.NewRandom().String()
			ovm1, _ := DefaultVirtualMachineWithNames(true, "testovm1", "testvm1")
			ovm1.Spec.Template.Spec.Domain.Firmware = &virtv1.Firmware{UUID: types.UID(uid)}

			vm1 := controller.setupVMIFromVM(ovm1)
			Expect(string(vm1.Spec.Domain.Firmware.UUID)).To(Equal(uid))
		})

		It("should delete VirtualMachineInstance when stopped", func() {
			ovm, vm := DefaultVirtualMachine(false)

			addVirtualMachine(ovm)
			vmFeeder.Add(vm)

			// ovmInterface.EXPECT().Update(gomock.Any()).Return(ovm, nil)
			vmInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

			ovmInterface.EXPECT().Update(gomock.Any()).Times(1).Return(ovm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should ignore non-matching VMIs", func() {
			ovm, vm := DefaultVirtualMachine(true)

			nonMatchingVMI := v1.NewMinimalVMI("testvm1")
			nonMatchingVMI.ObjectMeta.Labels = map[string]string{"test": "test1"}

			addVirtualMachine(ovm)

			// We still expect three calls to create VMIs, since VirtualMachineInstance does not meet the requirements
			vmSource.Add(nonMatchingVMI)

			vmInterface.EXPECT().Create(gomock.Any()).Return(vm, nil)
			ovmInterface.EXPECT().Update(gomock.Any()).Times(2).Return(ovm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should detect that it is orphan deleted and remove the owner reference on the remaining VirtualMachineInstance", func() {
			ovm, vm := DefaultVirtualMachine(true)

			// Mark it as orphan deleted
			now := metav1.Now()
			ovm.ObjectMeta.DeletionTimestamp = &now
			ovm.ObjectMeta.Finalizers = []string{metav1.FinalizerOrphanDependents}

			addVirtualMachine(ovm)
			vmFeeder.Add(vm)

			vmInterface.EXPECT().Patch(vm.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that a VirtualMachineInstance already exists and adopt it", func() {
			ovm, vm := DefaultVirtualMachine(true)
			vm.OwnerReferences = []metav1.OwnerReference{}

			addVirtualMachine(ovm)
			vmFeeder.Add(vm)

			ovmInterface.EXPECT().Get(ovm.ObjectMeta.Name, gomock.Any()).Return(ovm, nil)
			ovmInterface.EXPECT().Update(gomock.Any()).Return(ovm, nil)
			vmInterface.EXPECT().Patch(vm.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that it has nothing to do beside updating the status", func() {
			ovm, vm := DefaultVirtualMachine(true)

			addVirtualMachine(ovm)
			vmFeeder.Add(vm)

			ovmInterface.EXPECT().Update(gomock.Any()).Return(ovm, nil)

			controller.Execute()
		})

		It("should add a fail condition if start up fails", func() {
			ovm, vm := DefaultVirtualMachine(true)

			addVirtualMachine(ovm)
			// vmFeeder.Add(vm)

			vmInterface.EXPECT().Create(gomock.Any()).Return(vm, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			ovmInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*v1.VirtualMachine)
				Expect(objVM.Status.Conditions).To(HaveLen(1))
				cond := objVM.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.VirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedCreate"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			}).Return(ovm, nil)

			controller.Execute()

			testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
		})

		It("should add a fail condition if deletion fails", func() {
			ovm, vm := DefaultVirtualMachine(false)

			addVirtualMachine(ovm)
			vmFeeder.Add(vm)

			vmInterface.EXPECT().Delete(vm.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("failure"))

			ovmInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objVM := obj.(*v1.VirtualMachine)
				Expect(objVM.Status.Conditions).To(HaveLen(1))
				cond := objVM.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.VirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedDelete"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			})

			controller.Execute()

			testutils.ExpectEvents(recorder, FailedDeleteVirtualMachineReason)
		})
	})
})

func VirtualMachineFromVMI(name string, vm *v1.VirtualMachineInstance, started bool) *v1.VirtualMachine {
	ovm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vm.ObjectMeta.Namespace, ResourceVersion: "1"},
		Spec: v1.VirtualMachineSpec{
			Running: started,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vm.ObjectMeta.Name,
					Labels: vm.ObjectMeta.Labels,
				},
				Spec: vm.Spec,
			},
		},
	}
	return ovm
}

func DefaultVirtualMachineWithNames(started bool, ovmName string, vmName string) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	vm := v1.NewMinimalVMI(vmName)
	vm.Status.Phase = v1.Running
	ovm := VirtualMachineFromVMI(ovmName, vm, started)
	t := true
	vm.OwnerReferences = []metav1.OwnerReference{{
		APIVersion:         v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               v1.VirtualMachineGroupVersionKind.Kind,
		Name:               ovm.ObjectMeta.Name,
		UID:                ovm.ObjectMeta.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}}
	return ovm, vm
}

func DefaultVirtualMachine(started bool) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	return DefaultVirtualMachineWithNames(started, "testvm", "testvm")
}
