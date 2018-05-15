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

var _ = Describe("OfflineVirtualMachine", func() {

	Context("One valid OfflineVirtualMachine controller given", func() {

		var ctrl *gomock.Controller
		var vmInterface *kubecli.MockVMInterface
		var ovmInterface *kubecli.MockOfflineVirtualMachineInterface
		var vmSource *framework.FakeControllerSource
		var ovmSource *framework.FakeControllerSource
		var vmInformer cache.SharedIndexInformer
		var ovmInformer cache.SharedIndexInformer
		var stop chan struct{}
		var controller *OVMController
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
			vmInterface = kubecli.NewMockVMInterface(ctrl)
			ovmInterface = kubecli.NewMockOfflineVirtualMachineInterface(ctrl)

			vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
			ovmInformer, ovmSource = testutils.NewFakeInformerFor(&v1.OfflineVirtualMachine{})
			recorder = record.NewFakeRecorder(100)

			controller = NewOVMController(vmInformer, ovmInformer, recorder, virtClient)
			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockQueue = testutils.NewMockWorkQueue(controller.Queue)
			controller.Queue = mockQueue
			vmFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmSource)

			// Set up mock client
			virtClient.EXPECT().VM(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()
			virtClient.EXPECT().OfflineVirtualMachine(metav1.NamespaceDefault).Return(ovmInterface).AnyTimes()
		})

		addOfflineVirtualMachine := func(ovm *v1.OfflineVirtualMachine) {
			syncCaches(stop)
			mockQueue.ExpectAdds(1)
			ovmSource.Add(ovm)
			mockQueue.Wait()
		}

		It("should create missing VM", func() {
			ovm, vm := DefaultOfflineVirtualMachine(true)

			addOfflineVirtualMachine(ovm)

			// expect creation called
			vmInterface.EXPECT().Create(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.Name).To(Equal("testvm"))
			}).Return(vm, nil)

			// expect update status is called
			ovmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.OfflineVirtualMachine).Status.Created).To(BeFalse())
				Expect(arg.(*v1.OfflineVirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should update status to created if the vm exists", func() {
			ovm, vm := DefaultOfflineVirtualMachine(true)
			vm.Status.Phase = v1.Scheduled

			addOfflineVirtualMachine(ovm)
			vmFeeder.Add(vm)

			// expect update status is called
			ovmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.OfflineVirtualMachine).Status.Created).To(BeTrue())
				Expect(arg.(*v1.OfflineVirtualMachine).Status.Ready).To(BeFalse())
			}).Return(nil, nil)

			controller.Execute()
		})

		It("should update status to created and ready when vm is running", func() {
			ovm, vm := DefaultOfflineVirtualMachine(true)

			addOfflineVirtualMachine(ovm)
			vmFeeder.Add(vm)

			// expect update status is called
			ovmInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.OfflineVirtualMachine).Status.Created).To(BeTrue())
				Expect(arg.(*v1.OfflineVirtualMachine).Status.Ready).To(BeTrue())
			}).Return(nil, nil)

			controller.Execute()
		})

		It("should have stable firmware UUIDs", func() {
			ovm1, _ := DefaultOfflineVirtualMachineWithNames(true, "testovm1", "testvm1")
			vm1 := controller.setupVMFromOVM(ovm1)

			// intentionally use the same names
			ovm2, _ := DefaultOfflineVirtualMachineWithNames(true, "testovm1", "testvm1")
			vm2 := controller.setupVMFromOVM(ovm2)
			Expect(vm1.Spec.Domain.Firmware.UUID).To(Equal(vm2.Spec.Domain.Firmware.UUID))

			// now we want different names
			ovm3, _ := DefaultOfflineVirtualMachineWithNames(true, "testovm3", "testvm3")
			vm3 := controller.setupVMFromOVM(ovm3)
			Expect(vm1.Spec.Domain.Firmware.UUID).NotTo(Equal(vm3.Spec.Domain.Firmware.UUID))
		})

		It("should honour any firmware UUID present in the template", func() {
			uid := uuid.NewRandom().String()
			ovm1, _ := DefaultOfflineVirtualMachineWithNames(true, "testovm1", "testvm1")
			ovm1.Spec.Template.Spec.Domain.Firmware = &virtv1.Firmware{UUID: types.UID(uid)}

			vm1 := controller.setupVMFromOVM(ovm1)
			Expect(string(vm1.Spec.Domain.Firmware.UUID)).To(Equal(uid))
		})

		It("should delete VM when stopped", func() {
			ovm, vm := DefaultOfflineVirtualMachine(false)

			addOfflineVirtualMachine(ovm)
			vmFeeder.Add(vm)

			// ovmInterface.EXPECT().Update(gomock.Any()).Return(ovm, nil)
			vmInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

			ovmInterface.EXPECT().Update(gomock.Any()).Times(1).Return(ovm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should ignore non-matching VMs", func() {
			ovm, vm := DefaultOfflineVirtualMachine(true)

			nonMatchingVM := v1.NewMinimalVM("testvm1")
			nonMatchingVM.ObjectMeta.Labels = map[string]string{"test": "test1"}

			addOfflineVirtualMachine(ovm)

			// We still expect three calls to create VMs, since VM does not meet the requirements
			vmSource.Add(nonMatchingVM)

			vmInterface.EXPECT().Create(gomock.Any()).Return(vm, nil)
			ovmInterface.EXPECT().Update(gomock.Any()).Times(2).Return(ovm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should detect that it is orphan deleted and remove the owner reference on the remaining VM", func() {
			ovm, vm := DefaultOfflineVirtualMachine(true)

			// Mark it as orphan deleted
			now := metav1.Now()
			ovm.ObjectMeta.DeletionTimestamp = &now
			ovm.ObjectMeta.Finalizers = []string{metav1.FinalizerOrphanDependents}

			addOfflineVirtualMachine(ovm)
			vmFeeder.Add(vm)

			vmInterface.EXPECT().Patch(vm.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that a VM already exists and adopt it", func() {
			ovm, vm := DefaultOfflineVirtualMachine(true)
			vm.OwnerReferences = []metav1.OwnerReference{}

			addOfflineVirtualMachine(ovm)
			vmFeeder.Add(vm)

			ovmInterface.EXPECT().Get(ovm.ObjectMeta.Name, gomock.Any()).Return(ovm, nil)
			ovmInterface.EXPECT().Update(gomock.Any()).Return(ovm, nil)
			vmInterface.EXPECT().Patch(vm.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that it has nothing to do beside updating the status", func() {
			ovm, vm := DefaultOfflineVirtualMachine(true)

			addOfflineVirtualMachine(ovm)
			vmFeeder.Add(vm)

			ovmInterface.EXPECT().Update(gomock.Any()).Return(ovm, nil)

			controller.Execute()
		})

		It("should add a fail condition if start up fails", func() {
			ovm, vm := DefaultOfflineVirtualMachine(true)

			addOfflineVirtualMachine(ovm)
			// vmFeeder.Add(vm)

			vmInterface.EXPECT().Create(gomock.Any()).Return(vm, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			ovmInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objOVM := obj.(*v1.OfflineVirtualMachine)
				Expect(objOVM.Status.Conditions).To(HaveLen(1))
				cond := objOVM.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.OfflineVirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedCreate"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			}).Return(ovm, nil)

			controller.Execute()

			testutils.ExpectEvents(recorder, FailedCreateVirtualMachineReason)
		})

		It("should add a fail condition if deletion fails", func() {
			ovm, vm := DefaultOfflineVirtualMachine(false)

			addOfflineVirtualMachine(ovm)
			vmFeeder.Add(vm)

			vmInterface.EXPECT().Delete(vm.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("failure"))

			ovmInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objOVM := obj.(*v1.OfflineVirtualMachine)
				Expect(objOVM.Status.Conditions).To(HaveLen(1))
				cond := objOVM.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.OfflineVirtualMachineFailure))
				Expect(cond.Reason).To(Equal("FailedDelete"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			})

			controller.Execute()

			testutils.ExpectEvents(recorder, FailedDeleteVirtualMachineReason)
		})
	})
})

func OfflineVirtualMachineFromVM(name string, vm *v1.VirtualMachine, started bool) *v1.OfflineVirtualMachine {
	ovm := &v1.OfflineVirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vm.ObjectMeta.Namespace, ResourceVersion: "1"},
		Spec: v1.OfflineVirtualMachineSpec{
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
	return ovm
}

func DefaultOfflineVirtualMachineWithNames(started bool, ovmName string, vmName string) (*v1.OfflineVirtualMachine, *v1.VirtualMachine) {
	vm := v1.NewMinimalVM(vmName)
	vm.Status.Phase = v1.Running
	ovm := OfflineVirtualMachineFromVM(ovmName, vm, started)
	t := true
	vm.OwnerReferences = []metav1.OwnerReference{{
		APIVersion:         v1.OfflineVirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               v1.OfflineVirtualMachineGroupVersionKind.Kind,
		Name:               ovm.ObjectMeta.Name,
		UID:                ovm.ObjectMeta.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}}
	return ovm, vm
}

func DefaultOfflineVirtualMachine(started bool) (*v1.OfflineVirtualMachine, *v1.VirtualMachine) {
	return DefaultOfflineVirtualMachineWithNames(started, "testvm", "testvm")
}
