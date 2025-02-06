package clone

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	clone "kubevirt.io/api/clone/v1beta1"
	virtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	"kubevirt.io/kubevirt/pkg/testutils"

	gfh "github.com/AdaLogics/go-fuzz-headers"

	kvcontroller "kubevirt.io/kubevirt/pkg/controller"
)

// FuzzVMCloneController adds up to 5 VMs and VM Clones
// to the VM store and queue respectively and then
// invokes the VM Clone Controller.
func FuzzVMCloneController(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, numberOfVMs, numberOfVMClones uint8) {
		// Create 2 slices with random vms and vm clones.
		// The fuzzer will add these to the store and
		// queue after creating the controller
		if int(numberOfVMs) == 0 {
			return
		}
		if int(numberOfVMClones) == 0 {
			return
		}
		fdp := gfh.NewConsumer(data)
		vms := make([]*virtv1.VirtualMachine, 0)
		vmClones := make([]*clone.VirtualMachineClone, 0)

		maxVMs := int(numberOfVMs) % 5
		maxVMClones := int(numberOfVMClones) % 5

		for _ = range maxVMs {
			vm := &virtv1.VirtualMachine{}
			err := fdp.GenerateStruct(vm)
			if err != nil {
				return
			}
			vms = append(vms, vm)
		}
		if len(vms) == 0 {
			return
		}

		for _ = range maxVMClones {
			vmClone := &clone.VirtualMachineClone{}
			err := fdp.GenerateStruct(vmClone)
			if err != nil {
				return
			}
			vmClones = append(vmClones, vmClone)
		}
		if len(vmClones) == 0 {
			return
		}
		// Done creating the VMs and VM Clones

		// Set up the controller
		var (
			recorder   *record.FakeRecorder
			mockQueue  *testutils.MockWorkQueue[string]
			controller *VMCloneController
		)

		ctrl := gomock.NewController(t)
		vmInformer, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
		snapshotInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshot{})
		restoreInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineRestore{})
		cloneInformer, _ := testutils.NewFakeInformerFor(&clone.VirtualMachineClone{})
		snapshotContentInformer, _ := testutils.NewFakeInformerFor(&snapshotv1.VirtualMachineSnapshotContent{})
		pvcInformer, _ := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})

		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		controller, _ = NewVmCloneController(
			virtClient,
			cloneInformer,
			snapshotInformer,
			restoreInformer,
			vmInformer,
			snapshotContentInformer,
			pvcInformer,
			recorder)
		mockQueue = testutils.NewMockWorkQueue(controller.vmCloneQueue)
		controller.vmCloneQueue = mockQueue

		client := kubevirtfake.NewSimpleClientset()
		// Done setting up the controller

		// Add vms to vm store
		for _, randomVM := range vms {
			err := controller.vmStore.Add(randomVM)
			if err != nil {
				return
			}
		}

		// Add vm clones to the queue
		for _, randomVMClone := range vmClones {
			vmClone, err := client.CloneV1beta1().VirtualMachineClones(metav1.NamespaceDefault).Create(context.Background(), randomVMClone, metav1.CreateOptions{})
			if err != nil {
				return
			}
			controller.vmCloneIndexer.Add(vmClone)
			key, err := kvcontroller.KeyFunc(vmClone)
			if err != nil {
				return
			}
			mockQueue.Add(key)
		}
		controller.Execute()
	})
}
