package watch

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("VirtualMachineSnapshot", func() {

	Context("Valid VirtualMachineSnapshot controller given", func() {

		var ctrl *gomock.Controller
		var vmInterface *kubecli.MockVirtualMachineInterface
		var vmsInterface *kubecli.MockVirtualMachineSnapshotInterface
		var vmSource *framework.FakeControllerSource
		var vmsSource *framework.FakeControllerSource
		var vmInformer cache.SharedIndexInformer
		var vmsInformer cache.SharedIndexInformer
		var stop chan struct{}
		var controller *VMSnapshotController
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue
		//var vmFeeder *testutils.VirtualMachineFeeder

		syncCaches := func(stop chan struct{}) {
			go vmInformer.Run(stop)
			go vmsInformer.Run(stop)
			Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, vmsInformer.HasSynced)).To(BeTrue())
			log.Log.Info("Cache synced")
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
			vmsInterface = kubecli.NewMockVirtualMachineSnapshotInterface(ctrl)

			vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
			vmsInformer, vmsSource = testutils.NewFakeInformerFor(&v1.VirtualMachineSnapshot{})
			recorder = record.NewFakeRecorder(100)

			controller = NewVMSnapshotController(vmInformer, vmsInformer, recorder, virtClient)
			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockQueue = testutils.NewMockWorkQueue(controller.Queue)
			controller.Queue = mockQueue
			//vmFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmSource)

			// Set up mock client
			virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()
			virtClient.EXPECT().VirtualMachineSnapshot(metav1.NamespaceDefault).Return(vmsInterface).AnyTimes()
			syncCaches(stop)
		})

		addSnapshot := func(s *v1.VirtualMachineSnapshot) {
			mockQueue.ExpectAdds(1)
			vmsSource.Add(s)
			mockQueue.Wait()
		}

		It("should copy the VirtualMachine spec", func() {
			vms, vm := DefaultVMSnapshot()

			updatedVMS, _ := updateVirtualMachineSnapshotOwner(vms, vm)
			updatedVMS, _ = copyVirtualMachineToStatus(updatedVMS, vm)

			addSnapshot(vms)
			vmSource.Add(vm)

			vmInterface.EXPECT().Get(vm, gomock.Any()).Return(vm, nil)
			vmsInterface.EXPECT().Update(updatedVMS)

			controller.Execute()
		})

		It("should not do nothing when VirtualMachine is running", func() {
			vms, vm := DefaultVMSnapshot()
            t := true
            vm.Spec.Running = &t
			vm.Status.Ready = true

			addSnapshot(vms)
			vmSource.Add(vm)

			vmInterface.EXPECT().Get(vm, gomock.Any()).Return(vm, nil)
			vmsInterface.EXPECT().Update(gomock.Any()).Do(func(obj *v1.VirtualMachineSnapshot) {
				Expect(obj.Status.VirtualMachine).To(BeNil())
				Expect(obj.Status.Conditions[0].Reason).To(Equal(VirtualMachineRunningSnapshotReason))
			})

			controller.Execute()
		})

		It("should copy the VirtualMachine spec once VirtualMachine is running again", func() {
			vms, vm := DefaultVMSnapshot()
            t := true
            vm.Spec.Running = &t
			vm.Status.Ready = true

			addSnapshot(vms)
			vmSource.Add(vm)

			vmInterface.EXPECT().Get(vm, gomock.Any()).Return(vm, nil)
			vmsInterface.EXPECT().Update(gomock.Any()).Do(func(obj *v1.VirtualMachineSnapshot) {
				Expect(obj.Status.VirtualMachine).To(BeNil())
				Expect(obj.Status.Conditions[0].Reason).To(Equal(VirtualMachineRunningSnapshotReason))
			})

			controller.Execute()

            updatedVM := vm.DeepCopy()
            f := false
			updatedVM.Spec.Running = &f
			updatedVM.Status.Ready = false
			vmSource.Modify(updatedVM)

			updatedVMS, _ := updateVirtualMachineSnapshotOwner(vms, updatedVM)
			vmsSource.Modify(updatedVMS)

			updatedVMS, _ = copyVirtualMachineToStatus(updatedVMS, updatedVM)

			vmInterface.EXPECT().Get(vm, gomock.Any()).Return(updatedVM, nil)
			vmsInterface.EXPECT().Update(updatedVMS).Do(func(obj *v1.VirtualMachineSnapshot) {
				Expect(len(obj.Status.Conditions)).To(Equal(0))
				Expect(*obj.Status.VirtualMachine.Spec.Running).To(BeFalse())
			})

			controller.Execute()
		})

		It("should do nothing when the VirtualMachine become running", func() {
			vms, vm := DefaultVMSnapshot()

			addSnapshot(vms)
			vmSource.Add(vm)

			updatedVMS, _ := updateVirtualMachineSnapshotOwner(vms, vm)
			updatedVMS, _ = copyVirtualMachineToStatus(updatedVMS, vm)

			vmInterface.EXPECT().Get(vm, gomock.Any()).Return(vm, nil)
			vmsInterface.EXPECT().Update(updatedVMS).Do(func(obj *v1.VirtualMachineSnapshot) {
				Expect(len(obj.Status.Conditions)).To(Equal(0))
				Expect(*obj.Status.VirtualMachine.Spec.Running).To(BeFalse())
			})

			controller.Execute()

			updatedVM := vm.DeepCopy()
            t := true
            updatedVM.Spec.Running = &t
			updatedVM.Status.Ready = true
			vmSource.Modify(updatedVM)
			vmsSource.Modify(updatedVMS)

			vmInterface.EXPECT().Get(vm, gomock.Any()).Return(updatedVM, nil)
			vmsInterface.EXPECT().Update(updatedVMS).Do(func(obj *v1.VirtualMachineSnapshot) {
				Expect(len(obj.Status.Conditions)).To(Equal(0))
				Expect(*obj.Status.VirtualMachine.Spec.Running).To(BeFalse())
			})

			controller.Execute()
		})

	})
})

func DefaultVMSnapshot() (*v1.VirtualMachineSnapshot, *v1.VirtualMachine) {
	vm, _ := DefaultVirtualMachineWithNames(false, "test_vm", "test_vmi")
	vms := &v1.VirtualMachineSnapshot{
		ObjectMeta: metav1.ObjectMeta{Name: "test_vms", Namespace: vm.ObjectMeta.Namespace, ResourceVersion: "1"},
		Spec: v1.VirtualMachineSnapshotSpec{
			VirtualMachine: vm.Name,
		},
	}
	return vms, vm
}
