package convertmachinetype_test

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	"k8s.io/utils/pointer"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"
	. "kubevirt.io/kubevirt/pkg/virtctl/update/machine-type/convert-machine-type"
)

var _ = Describe("JobController", func() {
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var kubeClient *fake.Clientset
	var vmInformer cache.SharedIndexInformer
	var vmiInformer cache.SharedIndexInformer
	var mockQueue *testutils.MockWorkQueue
	var controller *JobController
	var stop chan struct{}
	var err error

	shouldExpectVMICreation := func(vmi *virtv1.VirtualMachineInstance) {
		kubeClient.Fake.PrependReactor("create", "virtualmachineinstances", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(create.GetObject().(*virtv1.VirtualMachineInstance).Name).To(Equal(vmi.Name))

			return true, create.GetObject(), nil
		})
	}

	shouldExpectVMCreation := func(vm *virtv1.VirtualMachine) {
		kubeClient.Fake.PrependReactor("create", "virtualmachine", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			Expect(create.GetObject().(*virtv1.VirtualMachineInstance).Name).To(Equal(vm.Name))

			return true, create.GetObject(), nil
		})
	}

	shouldExpectUpdateVMStatus := func(vm *virtv1.VirtualMachine) {
		patchData := `[{ "op": "replace", "path": "/status/machineTypeRestartRequired", "value": false }]`

		vmInterface.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, []byte(patchData), &metav1.PatchOptions{}).Times(1)

		kubeClient.Fake.PrependReactor("patch", "virtualmachines", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			patch, ok := action.(testing.PatchAction)
			Expect(ok).To(BeTrue())
			Expect(patch.GetPatch()).To(Equal([]byte(patchData)))
			Expect(patch.GetPatchType()).To(Equal(types.JSONPatchType))
			Expect(vm.Status.MachineTypeRestartRequired).To(BeFalse())
			return true, vm, nil
		})
	}

	addVm := func(vm *virtv1.VirtualMachine) {
		err = vmInformer.GetStore().Add(vm)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		key, err := cache.MetaNamespaceKeyFunc(vm)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		mockQueue.Add(key)
	}

	addVmi := func(vmi *virtv1.VirtualMachineInstance) {
		err = vmiInformer.GetStore().Add(vmi)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	stopVM := func(vm *virtv1.VirtualMachine) {
		vm.Spec.Running = pointer.Bool(false)
		err = vmInformer.GetStore().Update(vm)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		key, err := cache.MetaNamespaceKeyFunc(vm)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		mockQueue.Add(key)
	}

	Describe("When VMI is deleted", func() {
		var vm *virtv1.VirtualMachine
		var vmi *virtv1.VirtualMachineInstance

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			kubeClient = fake.NewSimpleClientset()
			stop = make(chan struct{})

			vmInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
			vmiInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})

			controller, err = NewJobController(vmInformer, vmiInformer, virtClient)
			Expect(err).ToNot(HaveOccurred())

			mockQueue = testutils.NewMockWorkQueue(controller.Queue)
			controller.Queue = mockQueue

			go controller.VmInformer.Run(stop)
			go controller.VmiInformer.Run(stop)
			Expect(cache.WaitForCacheSync(stop, controller.VmInformer.HasSynced)).To(BeTrue())

			virtClient.EXPECT().VirtualMachine(gomock.Any()).Return(vmInterface).AnyTimes()
			virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface).AnyTimes()

			MachineTypeGlob = machineTypeGlob
			Testing = true
		})

		Context("if VM machine type has been updated", func() {
			It("should set MachineTypeRestartRequired to false in VM Status", func() {
				vm = newVMRestartRequired("q35")
				vmi = newVMIWithMachineType(machineTypeNoUpdate, vm.Name)
				addVmi(vmi)
				addVm(vm)

				shouldExpectVMCreation(vm)
				shouldExpectVMICreation(vmi)
				shouldExpectUpdateVMStatus(vm)

				stopVM(vm)
				controller.Execute()
				Testing = false
			})
		})
	})
})

func newVMRestartRequired(machineType string) *virtv1.VirtualMachine {
	testVM := &virtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("test-vm-%s", machineType),
			Namespace: metav1.NamespaceDefault,
		},
		Spec: virtv1.VirtualMachineSpec{
			Running: pointer.Bool(true),
			Template: &virtv1.VirtualMachineInstanceTemplateSpec{
				Spec: virtv1.VirtualMachineInstanceSpec{
					Domain: virtv1.DomainSpec{},
				},
			},
		},
		Status: virtv1.VirtualMachineStatus{
			MachineTypeRestartRequired: true,
		},
	}
	if machineType != "q35" {
		testVM.Spec.Template.Spec.Domain.Machine = &virtv1.Machine{Type: machineType}
	}

	return testVM
}

func newVMIWithMachineType(machineType string, name string) *virtv1.VirtualMachineInstance {
	statusMachineType := machineType
	if machineType == "q35" {
		statusMachineType = "pc-q35-rhel9.2.0"
	}

	testvmi := &virtv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: virtv1.VirtualMachineInstanceSpec{
			Domain: virtv1.DomainSpec{
				Machine: &virtv1.Machine{
					Type: machineType,
				},
			},
		},
		Status: virtv1.VirtualMachineInstanceStatus{
			Machine: &virtv1.Machine{
				Type: statusMachineType,
			},
			Phase: virtv1.Running,
		},
	}

	return testvmi
}
