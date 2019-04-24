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

var _ = Describe("VirtualMachineRestore", func() {

	Context("Valid VirtualMachineSnapshot given", func() {

		var ctrl *gomock.Controller

		var vmInterface *kubecli.MockVirtualMachineInterface
		var vmSource *framework.FakeControllerSource
		var vmInformer cache.SharedIndexInformer

		var vmsInterface *kubecli.MockVirtualMachineSnapshotInterface
		var vmsSource *framework.FakeControllerSource
		var vmsInformer cache.SharedIndexInformer

		var vmrInterface *kubecli.MockVirtualMachineRestoreInterface
		var vmrSource *framework.FakeControllerSource
		var vmrInformer cache.SharedIndexInformer

		var stop chan struct{}
		var controller *VMRestoreController
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue

		syncCaches := func(stop chan struct{}) {
			go vmInformer.Run(stop)
			go vmsInformer.Run(stop)
			go vmrInformer.Run(stop)
			Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, vmsInformer.HasSynced, vmrInformer.HasSynced)).To(BeTrue())
			log.Log.Info("Cache synced")
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)

			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
			vmsInterface = kubecli.NewMockVirtualMachineSnapshotInterface(ctrl)
			vmrInterface = kubecli.NewMockVirtualMachineRestoreInterface(ctrl)

			vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
			vmsInformer, vmsSource = testutils.NewFakeInformerFor(&v1.VirtualMachineSnapshot{})
			vmrInformer, vmrSource = testutils.NewFakeInformerFor(&v1.VirtualMachineRestore{})


			recorder = record.NewFakeRecorder(100)

			controller = NewVMRestoreController(vmInformer, vmsInformer, vmrInformer, recorder, virtClient)
			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockQueue = testutils.NewMockWorkQueue(controller.Queue)
			controller.Queue = mockQueue

			// Set up mock client
			virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()
			virtClient.EXPECT().VirtualMachineSnapshot(metav1.NamespaceDefault).Return(vmsInterface).AnyTimes()
			virtClient.EXPECT().VirtualMachineRestore(metav1.NamespaceDefault).Return(vmrInterface).AnyTimes()
			syncCaches(stop)
		})

		addRestore := func(s *v1.VirtualMachineRestore) {
			mockQueue.ExpectAdds(1)
			vmrSource.Add(s)
			mockQueue.Wait()
		}

	 	It("should restore the VirtualMachine spec", func() {
			vms, vm := DefaultVMSnapshot()
			vmr := DefaultVMRestore(vms)

			vms.Status.VirtualMachine = vm.DeepCopy()

			vmSource.Add(vm)
			vmsSource.Add(vms)
			addRestore(vmr)

			vmInterface.EXPECT().Get(vm, gomock.Any()).Return(vm, nil)
			vmsInterface.EXPECT().Get(vms, gomock.Any()).Return(vms, nil)
			vmInterface.EXPECT().Update(vm).Return(vm, nil)
			vmrInterface.EXPECT().Update(gomock.Any()).Do(func(obj *v1.VirtualMachineRestore) {
				Expect(obj.Status.RestoredOn).ToNot(BeNil())
			})

			controller.Execute()
		})

		FIt("should not restore the VirtualMachine spec once VirtualMachine is running", func() {
			vms, vm := DefaultVMSnapshot()
			vmr := DefaultVMRestore(vms)

			vms.Status.VirtualMachine = vm.DeepCopy()

			vm.Spec.Running = true

			vmSource.Add(vm)
			vmsSource.Add(vms)
			addRestore(vmr)

			vmInterface.EXPECT().Get(vm, gomock.Any()).Return(vm, nil)
			vmsInterface.EXPECT().Get(vms, gomock.Any()).Return(vms, nil)

			controller.Execute()
		})

		It("should restore once VM is not running", func() {

			controller.Execute()
		})

		It("should not restore VM when there is no snapshot", func() {

		})

		It("shoudl not restore VM when snapshot does not have VM copied", func() {

		})

	})

	Context("Valid multiple VirtualMachineSnapshots given", func() {
		It("should restore VM only from one snapshot once", func() {

		})
	})
})

func DefaultVMRestore(vms *v1.VirtualMachineSnapshot) *v1.VirtualMachineRestore {
	vmr := &v1.VirtualMachineRestore{
		ObjectMeta: metav1.ObjectMeta{Name: vms.Spec.VirtualMachine, Namespace: vms.ObjectMeta.Namespace, ResourceVersion: "1"},
		Spec: v1.VirtualMachineRestoreSpec{
			VirtualMachineSnapshot: vms.Name,
		},
	}
	return vmr
}
